// framebuffer包提供了Linux Framebuffer设备的底层操作接口
// 用于直接在帧缓冲区设备上进行图形绘制和显示操作
package framebuffer

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// FrameBuffer 结构体封装了帧缓冲区设备的所有操作
// 包含设备句柄、屏幕信息、内存映射数据等
type FrameBuffer struct {
	device     *os.File        // 帧缓冲区设备文件句柄，通常为/dev/fb0
	screenInfo FixedScreenInfo // 固定屏幕信息，包含硬件相关的不可变参数
	varInfo    VarScreenInfo   // 可变屏幕信息，包含分辨率、色深等可配置参数
	fbData     []byte          // 内存映射的帧缓冲区数据，直接操作此数组即可修改屏幕内容
	width      int             // 屏幕宽度（像素）
	height     int             // 屏幕高度（像素）
	bpp        int             // 每像素位数（bits per pixel）
	mu         sync.RWMutex    // 读写锁，保护并发访问
	closed     bool            // 关闭状态标志
}

// FixedScreenInfo 固定屏幕信息结构体
// 对应Linux内核中的fb_fix_screeninfo结构，包含硬件固定参数
type FixedScreenInfo struct {
	Id         [16]int8 // 帧缓冲区标识符字符串
	Smem       uintptr  // 屏幕内存起始地址
	SmemLen    uint32   // 屏幕内存长度（字节）
	Type       uint32   // 帧缓冲区类型
	TypeAux    uint32   // 辅助类型信息
	Visual     uint32   // 视觉模式（如伪彩色、真彩色等）
	XPanstep   uint16   // 水平滚动步长
	YPanstep   uint16   // 垂直滚动步长
	YWrapstep  uint16   // 垂直环绕步长
	LineLength uint32   // 每行字节数（包含填充）
	Mmio       uintptr  // 内存映射I/O起始地址
	MmioLen    uint32   // 内存映射I/O长度
	Accel      uint32   // 硬件加速器类型
	Reserved   [3]uint16 // 保留字段
}

// VarScreenInfo 可变屏幕信息结构体
// 对应Linux内核中的fb_var_screeninfo结构，包含可配置的显示参数
type VarScreenInfo struct {
	XRes           uint32 // 水平分辨率（像素）
	YRes           uint32 // 垂直分辨率（像素）
	XResVirtual    uint32 // 虚拟水平分辨率
	YResVirtual    uint32 // 虚拟垂直分辨率
	XOffset        uint32 // 水平偏移量
	YOffset        uint32 // 垂直偏移量
	BitsPerPixel   uint32 // 每像素位数
	Grayscale      uint32 // 灰度模式标志（0=彩色，1=灰度）
	RedOffset      uint32 // 红色分量在像素中的位偏移
	RedLength      uint32 // 红色分量的位长度
	RedMsbRight    uint32 // 红色分量最高位在右侧标志
	GreenOffset    uint32 // 绿色分量在像素中的位偏移
	GreenLength    uint32 // 绿色分量的位长度
	GreenMsbRight  uint32 // 绿色分量最高位在右侧标志
	BlueOffset     uint32 // 蓝色分量在像素中的位偏移
	BlueLength     uint32 // 蓝色分量的位长度
	BlueMsbRight   uint32 // 蓝色分量最高位在右侧标志
	TranspOffset   uint32 // 透明度分量在像素中的位偏移
	TranspLength   uint32 // 透明度分量的位长度
	TranspMsbRight uint32 // 透明度分量最高位在右侧标志
	Nonstd         uint32 // 非标准像素格式标志
	Activate       uint32 // 激活标志
	Height         uint32 // 屏幕物理高度（毫米）
	Width          uint32 // 屏幕物理宽度（毫米）
	AccelFlags     uint32 // 硬件加速标志
	PixClock       uint32 // 像素时钟（皮秒）
	LeftMargin     uint32 // 左边距
	RightMargin    uint32 // 右边距
	UpperMargin    uint32 // 上边距
	LowerMargin    uint32 // 下边距
	HsyncLen       uint32 // 水平同步长度
	VsyncLen       uint32 // 垂直同步长度
	Sync           uint32 // 同步标志
	Vmode          uint32 // 视频模式
	Rotate         uint32 // 旋转角度
	Reserved       [5]uint32 // 保留字段
}

// Linux帧缓冲区相关的ioctl命令常量
const (
	FBIOGET_FSCREENINFO = 0x4602 // 获取固定屏幕信息的ioctl命令
	FBIOGET_VSCREENINFO = 0x4600 // 获取可变屏幕信息的ioctl命令
)

// NewFrameBuffer 创建并初始化一个新的帧缓冲区对象
// 参数device: 帧缓冲区设备路径，如"/dev/fb0"
// 返回初始化完成的FrameBuffer对象或错误信息
func NewFrameBuffer(device string) (*FrameBuffer, error) {
	fb := &FrameBuffer{} // 创建FrameBuffer实例
	
	var err error
	// 打开帧缓冲区设备文件，需要读写权限
	fb.device, err = os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("无法打开帧缓冲区设备: %v", err)
	}

	// 获取屏幕信息（分辨率、色深等参数）
	err = fb.getScreenInfo()
	if err != nil {
		fb.device.Close()
		return nil, err
	}

	// 将帧缓冲区内存映射到程序地址空间
	err = fb.mapMemory()
	if err != nil {
		fb.device.Close()
		return nil, err
	}

	return fb, nil
}

// getScreenInfo 获取帧缓冲区的屏幕信息
// 通过ioctl系统调用获取固定和可变屏幕信息
func (fb *FrameBuffer) getScreenInfo() error {
	// 获取固定屏幕信息（硬件相关的不可变参数）
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fb.device.Fd()),
		FBIOGET_FSCREENINFO,
		uintptr(unsafe.Pointer(&fb.screenInfo)))
	if errno != 0 {
		return fmt.Errorf("无法获取固定屏幕信息: %v", errno)
	}

	// 获取可变屏幕信息（分辨率、色深等可配置参数）
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fb.device.Fd()),
		FBIOGET_VSCREENINFO,
		uintptr(unsafe.Pointer(&fb.varInfo)))
	if errno != 0 {
		return fmt.Errorf("无法获取可变屏幕信息: %v", errno)
	}

	// 从屏幕信息中提取基本参数
	fb.width = int(fb.varInfo.XRes)      // 屏幕宽度
	fb.height = int(fb.varInfo.YRes)     // 屏幕高度
	fb.bpp = int(fb.varInfo.BitsPerPixel) // 每像素位数

	return nil
}

// mapMemory 将帧缓冲区内存映射到程序地址空间
// 使用mmap系统调用将设备内存映射为可直接访问的字节数组
func (fb *FrameBuffer) mapMemory() error {
	screenSize := int(fb.screenInfo.SmemLen) // 获取屏幕内存大小
	
	// 验证屏幕大小的合理性
	if screenSize <= 0 || screenSize > 1024*1024*1024 { // 限制最大1GB
		return fmt.Errorf("屏幕内存大小不合理: %d bytes", screenSize)
	}
	
	// 使用mmap将帧缓冲区内存映射到程序地址空间
	fbData, err := syscall.Mmap(
		int(fb.device.Fd()),                    // 文件描述符
		0,                                      // 偏移量
		screenSize,                             // 映射大小
		syscall.PROT_READ|syscall.PROT_WRITE,   // 读写权限
		syscall.MAP_SHARED,                     // 共享映射
	)
	if err != nil {
		return fmt.Errorf("无法映射帧缓冲区内存: %v", err)
	}

	// 验证映射结果
	if len(fbData) != screenSize {
		syscall.Munmap(fbData) // 清理映射
		return fmt.Errorf("映射大小不匹配: 期望 %d, 实际 %d", screenSize, len(fbData))
	}

	fb.fbData = fbData
	return nil
}

// GetDimensions 获取屏幕尺寸
// 返回屏幕的宽度和高度（像素）
func (fb *FrameBuffer) GetDimensions() (int, int) {
	return fb.width, fb.height
}

// Clear 清空屏幕
// 将整个帧缓冲区填充为0（通常为黑色）
func (fb *FrameBuffer) Clear() {
	fb.mu.Lock()
	defer fb.mu.Unlock()
	
	if fb.closed || fb.fbData == nil {
		return
	}
	
	// 使用更高效的清零方法
	for i := range fb.fbData {
		fb.fbData[i] = 0
	}
}

// SetPixel 在指定位置设置像素颜色
// 参数x,y: 像素坐标  参数c: 颜色值
// 根据不同的色深格式写入相应的像素数据
func (fb *FrameBuffer) SetPixel(x, y int, c color.Color) {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	
	// 检查状态
	if fb.closed || fb.fbData == nil {
		return
	}
	
	// 边界检查，超出屏幕范围则直接返回
	if x < 0 || x >= fb.width || y < 0 || y >= fb.height {
		return
	}

	// 提取RGB颜色分量并转换为8位
	r, g, b, _ := c.RGBA()
	r >>= 8  // 将16位颜色值转换为8位
	g >>= 8
	b >>= 8

	// 计算像素在帧缓冲区中的字节偏移量
	offset := y*int(fb.screenInfo.LineLength) + x*(fb.bpp/8)
	
	// 边界检查：确保不会越界访问
	bytesPerPixel := fb.bpp / 8
	if offset < 0 || offset+bytesPerPixel > len(fb.fbData) {
		return
	}
	
	// 根据不同的色深格式写入像素数据
	switch fb.bpp {
	case 16: // 16位色深（RGB565格式）
		pixel := uint16((r&0xF8)<<8 | (g&0xFC)<<3 | (b&0xF8)>>3)
		fb.fbData[offset] = byte(pixel & 0xFF)     // 低字节
		fb.fbData[offset+1] = byte(pixel >> 8)     // 高字节
	case 24: // 24位色深（RGB888格式）
		fb.fbData[offset] = byte(b)     // 蓝色分量
		fb.fbData[offset+1] = byte(g)   // 绿色分量
		fb.fbData[offset+2] = byte(r)   // 红色分量
	case 32: // 32位色深（ARGB8888格式）
		fb.fbData[offset] = byte(b)     // 蓝色分量
		fb.fbData[offset+1] = byte(g)   // 绿色分量
		fb.fbData[offset+2] = byte(r)   // 红色分量
		fb.fbData[offset+3] = 0         // Alpha通道（透明度）
	}
}

// DrawImage 在指定位置绘制图像
// 参数img: 要绘制的图像  参数x,y: 绘制位置的左上角坐标
func (fb *FrameBuffer) DrawImage(img image.Image, x, y int) {
	fb.mu.RLock()
	defer fb.mu.RUnlock()
	
	if fb.closed || fb.fbData == nil {
		return
	}
	
	bounds := img.Bounds() // 获取图像边界
	
	// 裁剪绘制区域，避免越界
	startX := max(0, x)
	startY := max(0, y)
	endX := min(fb.width, x+bounds.Dx())
	endY := min(fb.height, y+bounds.Dy())
	
	// 逐像素绘制图像
	for py := startY; py < endY; py++ {
		for px := startX; px < endX; px++ {
			// 计算源图像坐标
			srcX := bounds.Min.X + (px - x)
			srcY := bounds.Min.Y + (py - y)
			
			// 获取源图像的像素颜色
			c := img.At(srcX, srcY)
			// 直接设置像素（避免重复锁定）
			fb.setPixelUnsafe(px, py, c)
		}
	}
}

// Close 关闭帧缓冲区并释放资源
// 取消内存映射并关闭设备文件
// setPixelUnsafe 不安全的像素设置方法，调用前需要确保已加锁
func (fb *FrameBuffer) setPixelUnsafe(x, y int, c color.Color) {
	// 边界检查，超出屏幕范围则直接返回
	if x < 0 || x >= fb.width || y < 0 || y >= fb.height {
		return
	}

	// 提取RGB颜色分量并转换为8位
	r, g, b, _ := c.RGBA()
	r >>= 8  // 将16位颜色值转换为8位
	g >>= 8
	b >>= 8

	// 计算像素在帧缓冲区中的字节偏移量
	offset := y*int(fb.screenInfo.LineLength) + x*(fb.bpp/8)
	
	// 边界检查：确保不会越界访问
	bytesPerPixel := fb.bpp / 8
	if offset < 0 || offset+bytesPerPixel > len(fb.fbData) {
		return
	}
	
	// 根据不同的色深格式写入像素数据
	switch fb.bpp {
	case 16: // 16位色深（RGB565格式）
		pixel := uint16((r&0xF8)<<8 | (g&0xFC)<<3 | (b&0xF8)>>3)
		fb.fbData[offset] = byte(pixel & 0xFF)     // 低字节
		fb.fbData[offset+1] = byte(pixel >> 8)     // 高字节
	case 24: // 24位色深（RGB888格式）
		fb.fbData[offset] = byte(b)     // 蓝色分量
		fb.fbData[offset+1] = byte(g)   // 绿色分量
		fb.fbData[offset+2] = byte(r)   // 红色分量
	case 32: // 32位色深（ARGB8888格式）
		fb.fbData[offset] = byte(b)     // 蓝色分量
		fb.fbData[offset+1] = byte(g)   // 绿色分量
		fb.fbData[offset+2] = byte(r)   // 红色分量
		fb.fbData[offset+3] = 0         // Alpha通道（透明度）
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max 返回两个整数中的较大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (fb *FrameBuffer) Close() error {
	fb.mu.Lock()
	defer fb.mu.Unlock()
	
	if fb.closed {
		return nil // 已经关闭
	}
	
	var err error
	
	// 取消内存映射
	if fb.fbData != nil {
		if munmapErr := syscall.Munmap(fb.fbData); munmapErr != nil {
			err = fmt.Errorf("取消内存映射失败: %v", munmapErr)
		}
		fb.fbData = nil
	}
	
	// 关闭设备文件
	if fb.device != nil {
		if closeErr := fb.device.Close(); closeErr != nil {
			if err != nil {
				err = fmt.Errorf("%v; 关闭设备文件失败: %v", err, closeErr)
			} else {
				err = fmt.Errorf("关闭设备文件失败: %v", closeErr)
			}
		}
		fb.device = nil
	}
	
	fb.closed = true
	return err
}

// GetBestFramebufferDevice 获取最佳的帧缓冲区设备
// 按优先级检查可用的帧缓冲区设备，返回第一个存在的设备路径
func GetBestFramebufferDevice() string {
	devices := []string{"/dev/fb0", "/dev/fb1", "/dev/fb2"} // 常见的帧缓冲区设备
	
	// 检查设备文件是否存在
	for _, device := range devices {
		if _, err := os.Stat(device); err == nil {
			return device
		}
	}
	
	// 如果都不存在，返回默认设备
	return "/dev/fb0"
}

// GetConsoleResolution 获取控制台分辨率
// 从系统文件中读取帧缓冲区的虚拟分辨率信息
func GetConsoleResolution() (int, int, error) {
	// 读取虚拟分辨率文件
	data, err := os.ReadFile("/sys/class/graphics/fb0/virtual_size")
	if err != nil {
		// 如果读取失败，返回默认分辨率
		return 1920, 1080, nil
	}
	
	// 解析分辨率字符串（格式：width,height）
	parts := strings.Split(strings.TrimSpace(string(data)), ",")
	if len(parts) != 2 {
		return 1920, 1080, nil
	}
	
	// 转换字符串为整数
	width, err1 := strconv.Atoi(parts[0])
	height, err2 := strconv.Atoi(parts[1])
	
	if err1 != nil || err2 != nil {
		return 1920, 1080, nil
	}
	
	return width, height, nil
}
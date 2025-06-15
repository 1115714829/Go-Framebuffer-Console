// input包提供了键盘输入处理功能
// 实现了原始模式的键盘输入捕获，支持实时响应用户操作
package input

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// KeyboardInput 键盘输入处理器结构体
// 封装了终端设备和相关的配置信息
type KeyboardInput struct {
	device     *os.File        // 终端设备文件句柄（通常为/dev/stdin）
	ttyDevice  *os.File        // TTY设备文件句柄（用于写入控制序列）
	oldTermios syscall.Termios // 原始终端属性，用于恢复设置
	mu         sync.Mutex      // 保护并发访问
	closed     bool            // 关闭状态标志
	restored   bool            // 终端状态恢复标志
}

// InputEvent 输入事件结构体
// 对应Linux内核的input_event结构，用于处理输入事件
type InputEvent struct {
	Time  syscall.Timeval // 事件发生时间
	Type  uint16          // 事件类型（如键盘、鼠标等）
	Code  uint16          // 事件代码（具体的键值）
	Value int32           // 事件值（按下、抬起等）
}

// Linux输入子系统相关常量
// 用于处理不同类型的输入事件
const (
	EV_KEY    = 0x01 // 键盘事件类型
	KEY_ENTER = 28   // 回车键的扫描码
	KEY_Q     = 16   // Q键的扫描码
	KEY_1     = 2    // 数字1键的扫描码
	KEY_2     = 3    // 数字2键的扫描码
	KEY_3     = 4    // 数字3键的扫描码
	KEY_4     = 5    // 数字4键的扫描码
	KEY_5     = 6    // 数字5键的扫描码
	KEY_ESC   = 1    // ESC键的扫描码
)

// 终端控制相关的ioctl命令常量
// 用于获取和设置终端属性
const (
	TCGETS = 0x5401 // 获取终端属性的ioctl命令
	TCSETS = 0x5402 // 设置终端属性的ioctl命令
)

// NewKeyboardInput 创建新的键盘输入处理器
// 初始化终端设备并设置为原始模式，实现无缓冲的字符输入
// 返回初始化完成的键盘输入器或错误信息
func NewKeyboardInput() (*KeyboardInput, error) {
	ki := &KeyboardInput{} // 创建键盘输入器实例

	var err error
	// 打开标准输入设备（终端）
	ki.device, err = os.OpenFile("/dev/stdin", os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("无法打开标准输入设备: %v", err)
	}

	// 打开TTY设备用于写入控制序列
	ki.ttyDevice, err = os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		// 如果无法打开/dev/tty，尝试使用stdout
		ki.ttyDevice = os.Stdout
	}

	// 设置终端为原始模式（无缓冲、无回显）
	err = ki.setRawMode()
	if err != nil {
		ki.device.Close()
		if ki.ttyDevice != os.Stdout {
			ki.ttyDevice.Close()
		}
		return nil, err
	}

	return ki, nil
}

// setRawMode 设置终端为原始模式
// 禁用行编辑、回显和特殊字符处理，实现字符级的实时输入
func (ki *KeyboardInput) setRawMode() error {
	fd := int(ki.device.Fd()) // 获取文件描述符

	// 获取当前终端属性（用于后续恢复）
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd),
		TCGETS,
		uintptr(unsafe.Pointer(&ki.oldTermios)))
	if errno != 0 {
		return fmt.Errorf("无法获取终端属性: %v", errno)
	}

	// 复制当前属性并进行修改
	newTermios := ki.oldTermios
	// 禁用本地模式标志：行编辑、回显等
	newTermios.Lflag &^= syscall.ICANON | syscall.ECHO | syscall.ECHOE | syscall.ECHOK | syscall.ECHONL | syscall.ECHOPRT | syscall.ECHOKE | syscall.ICRNL
	// 禁用输入模式标志：流控制等
	newTermios.Iflag &^= syscall.IXON | syscall.IXOFF | syscall.IXANY
	// 设置特殊字符：最少读取1个字符，无超时
	newTermios.Cc[syscall.VMIN] = 1  // 最少读取字符数
	newTermios.Cc[syscall.VTIME] = 0 // 读取超时时间（0表示阻塞）

	// 应用新的终端属性
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd),
		TCSETS,
		uintptr(unsafe.Pointer(&newTermios)))
	if errno != 0 {
		return fmt.Errorf("无法设置终端属性: %v", errno)
	}

	// 隐藏光标
	if err := ki.hideCursor(); err != nil {
		return fmt.Errorf("隐藏光标失败: %v", err)
	}

	return nil
}

func (ki *KeyboardInput) ReadKey() (byte, error) {
	ki.mu.Lock()
	defer ki.mu.Unlock()

	if ki.closed || ki.device == nil {
		return 0, fmt.Errorf("键盘设备已关闭")
	}

	buf := make([]byte, 1)
	n, err := ki.device.Read(buf)
	if err != nil {
		return 0, fmt.Errorf("读取键盘输入失败: %v", err)
	}
	if n == 0 {
		return 0, fmt.Errorf("no data read")
	}
	return buf[0], nil
}

func (ki *KeyboardInput) ReadKeyNonBlocking() (byte, bool, error) {
	ki.mu.Lock()
	defer ki.mu.Unlock()

	if ki.closed || ki.device == nil {
		return 0, false, fmt.Errorf("键盘设备已关闭")
	}

	buf := make([]byte, 1)
	fd := int(ki.device.Fd())

	// 检查文件描述符的有效性
	if fd < 0 {
		return 0, false, fmt.Errorf("无效的文件描述符")
	}

	var readfds syscall.FdSet
	readfds.Bits[fd/64] |= 1 << (uint(fd) % 64)

	timeout := syscall.Timeval{Sec: 0, Usec: 0}

	n, err := syscall.Select(fd+1, &readfds, nil, nil, &timeout)
	if err != nil {
		return 0, false, fmt.Errorf("select调用失败: %v", err)
	}

	if n == 0 {
		return 0, false, nil
	}

	n2, err := ki.device.Read(buf)
	if err != nil {
		return 0, false, fmt.Errorf("读取数据失败: %v", err)
	}
	if n2 == 0 {
		return 0, false, nil
	}

	return buf[0], true, nil
}

func (ki *KeyboardInput) ReadKeyNonBlockingWithTimeout(timeout time.Duration) (byte, bool, error) {
	ki.mu.Lock()
	defer ki.mu.Unlock()

	if ki.closed || ki.device == nil {
		return 0, false, fmt.Errorf("键盘设备已关闭")
	}

	buf := make([]byte, 1)
	fd := int(ki.device.Fd())

	if fd < 0 {
		return 0, false, fmt.Errorf("无效的文件描述符")
	}

	var readfds syscall.FdSet
	readfds.Bits[fd/64] |= 1 << (uint(fd) % 64)

	tv := syscall.NsecToTimeval(timeout.Nanoseconds())

	n, err := syscall.Select(fd+1, &readfds, nil, nil, &tv)
	if err != nil {
		// EINTR 表示系统调用被信号中断，这在我们的场景中是正常现象，不应视为错误
		if errno, ok := err.(syscall.Errno); ok && errno == syscall.EINTR {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("select调用失败: %v", err)
	}

	if n == 0 {
		return 0, false, nil // 超时
	}

	n2, err := ki.device.Read(buf)
	if err != nil {
		return 0, false, fmt.Errorf("读取数据失败: %v", err)
	}
	if n2 == 0 {
		return 0, false, nil
	}

	return buf[0], true, nil
}

func (ki *KeyboardInput) WaitForKey(keys ...byte) (byte, error) {
	return ki.WaitForKeyWithTimeout(30*time.Second, keys...)
}

func (ki *KeyboardInput) WaitForKeyWithTimeout(timeout time.Duration, keys ...byte) (byte, error) {
	start := time.Now()
	for {
		// 检查超时
		if time.Since(start) > timeout {
			return 0, fmt.Errorf("等待键盘输入超时")
		}

		key, available, err := ki.ReadKeyNonBlocking()
		if err != nil {
			return 0, err
		}

		if available {
			if len(keys) == 0 {
				return key, nil
			}

			for _, validKey := range keys {
				if key == validKey {
					return key, nil
				}
			}
		}

		// 短暂睡眠避免占用CPU
		time.Sleep(10 * time.Millisecond)
	}
}

func (ki *KeyboardInput) WaitForEnter() error {
	_, err := ki.WaitForKeyWithTimeout(30*time.Second, '\n', '\r')
	return err
}

func (ki *KeyboardInput) WaitForEnterWithContext(ctx context.Context) error {
	done := make(chan error, 1)
	go func() {
		_, err := ki.WaitForKey('\n', '\r')
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (ki *KeyboardInput) WaitForMenuChoice() (int, error) {
	return ki.WaitForMenuChoiceWithTimeout(30 * time.Second)
}

func (ki *KeyboardInput) WaitForMenuChoiceWithTimeout(timeout time.Duration) (int, error) {
	start := time.Now()
	for {
		// 检查超时
		if time.Since(start) > timeout {
			return 0, fmt.Errorf("等待菜单选择超时")
		}

		key, available, err := ki.ReadKeyNonBlocking()
		if err != nil {
			return 0, err
		}

		if available {
			switch key {
			case '1':
				return 1, nil
			case '2':
				return 2, nil
			case '3':
				return 3, nil
			case '4':
				return 4, nil
			case '5':
				return 5, nil
			case 'q', 'Q':
				return -1, nil
			case '\n', '\r':
				return 0, nil
			}
		}

		// 短暂睡眠避免占用CPU
		time.Sleep(10 * time.Millisecond)
	}
}

func (ki *KeyboardInput) Close() error {
	ki.mu.Lock()
	defer ki.mu.Unlock()

	if ki.closed {
		return nil // 已经关闭
	}

	var err error

	// 先恢复终端状态
	if !ki.restored {
		if restoreErr := ki.restoreTerminalUnsafe(); restoreErr != nil {
			err = fmt.Errorf("恢复终端状态失败: %v", restoreErr)
		}
	}

	// 关闭设备
	if ki.device != nil {
		if closeErr := ki.device.Close(); closeErr != nil {
			if err != nil {
				err = fmt.Errorf("%v; 关闭输入设备失败: %v", err, closeErr)
			} else {
				err = fmt.Errorf("关闭输入设备失败: %v", closeErr)
			}
		}
		ki.device = nil
	}

	// 关闭TTY设备
	if ki.ttyDevice != nil && ki.ttyDevice != os.Stdout {
		if closeErr := ki.ttyDevice.Close(); closeErr != nil {
			if err != nil {
				err = fmt.Errorf("%v; 关闭TTY设备失败: %v", err, closeErr)
			} else {
				err = fmt.Errorf("关闭TTY设备失败: %v", closeErr)
			}
		}
		ki.ttyDevice = nil
	}

	ki.closed = true
	return err
}

func (ki *KeyboardInput) RestoreTerminal() error {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	return ki.restoreTerminalUnsafe()
}

func (ki *KeyboardInput) restoreTerminalUnsafe() error {
	if ki.device == nil || ki.restored {
		return nil
	}

	fd := int(ki.device.Fd())
	if fd < 0 {
		return fmt.Errorf("无效的文件描述符")
	}

	// 先恢复光标显示
	ki.showCursor()

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(fd),
		TCSETS,
		uintptr(unsafe.Pointer(&ki.oldTermios)))
	if errno != 0 {
		return fmt.Errorf("failed to restore terminal: %v", errno)
	}

	ki.restored = true
	return nil
}

// hideCursor 隐藏终端光标
func (ki *KeyboardInput) hideCursor() error {
	if ki.ttyDevice == nil {
		return nil
	}
	_, err := ki.ttyDevice.Write([]byte("\033[?25l"))
	return err
}

// showCursor 显示终端光标
func (ki *KeyboardInput) showCursor() error {
	if ki.ttyDevice == nil {
		return nil
	}
	_, err := ki.ttyDevice.Write([]byte("\033[?25h"))
	return err
}

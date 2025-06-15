package menu

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"go-framebuffer-console/pkg/font"
	"go-framebuffer-console/pkg/framebuffer"
	"go-framebuffer-console/pkg/system"
)

type MenuRenderer struct {
	fb       *framebuffer.FrameBuffer
	renderer *font.Renderer
	width    int
	height   int
	// 智能刷新相关
	lastContent    string // 上次显示的内容
	needsClear     bool   // 是否需要清屏
	staticRendered bool   // 静态内容是否已渲染
}

func NewMenuRenderer(fb *framebuffer.FrameBuffer, fontRenderer *font.Renderer) *MenuRenderer {
	width, height := fb.GetDimensions()
	return &MenuRenderer{
		fb:             fb,
		renderer:       fontRenderer,
		width:          width,
		height:         height,
		needsClear:     true, // 初始需要清屏
		staticRendered: false,
	}
}

func (mr *MenuRenderer) RenderMainMenu(sysInfo *system.SystemInfo) error {
	// 使用14号字体
	mr.renderer.SetSize(14)
	
	// 生成当前内容
	currentContent := mr.generateMainMenuContent(sysInfo)
	
	// 检查是否需要刷新
	if currentContent == mr.lastContent && mr.staticRendered {
		return nil // 内容没有变化，无需刷新
	}
	
	// 如果是首次渲染或内容完全变化，需要清屏
	if mr.needsClear || !mr.staticRendered {
		mr.fb.Clear()
		mr.needsClear = false
	}
	
	// 分区域渲染
	if err := mr.renderStaticContent(); err != nil {
		return err
	}
	
	if err := mr.renderDynamicContent(sysInfo); err != nil {
		return err
	}
	
	mr.lastContent = currentContent
	mr.staticRendered = true
	return nil
}

// renderStaticContent 渲染静态内容（标题和操作指南）
func (mr *MenuRenderer) renderStaticContent() error {
	if mr.staticRendered {
		return nil // 静态内容已渲染，跳过
	}
	
	staticContent := `=== 系统状态监控 ===

操作指南:
- 按回车键(Enter): 进入配置菜单
- 按 Ctrl+C: 退出程序
- 系统状态每5秒自动更新`
	
	lines := strings.Split(staticContent, "\n")
	img, err := mr.renderer.RenderMultilineText(lines, color.RGBA{255, 255, 255, 255}, 3)
	if err != nil {
		return fmt.Errorf("failed to render static content: %v", err)
	}
	
	// 在底部显示操作指南
	x := 20
	y := mr.height - img.Bounds().Dy() - 40
	mr.fb.DrawImage(img, x, y)
	
	return nil
}

// renderDynamicContent 渲染动态内容（系统状态）
func (mr *MenuRenderer) renderDynamicContent(sysInfo *system.SystemInfo) error {
	dynamicContent := fmt.Sprintf(
		"运行时间: %s\n"+
			"处理器: %s (%d 核心)\n"+
			"内存使用: %s\n"+
			"磁盘大小: %s (共 %d 个磁盘)\n"+
			"系统时间: %s\n"+
			"IP地址: %s",
		sysInfo.Uptime,
		sysInfo.CPUModel,
		sysInfo.CPUCores,
		sysInfo.MemoryUsage,
		sysInfo.DiskSize,
		sysInfo.DiskCount,
		sysInfo.CurrentTime,
		sysInfo.IPAddress,
	)
	
	lines := strings.Split(dynamicContent, "\n")
	img, err := mr.renderer.RenderMultilineText(lines, color.RGBA{255, 255, 255, 255}, 3)
	if err != nil {
		return fmt.Errorf("failed to render dynamic content: %v", err)
	}
	
	// 清除动态内容区域（避免残留）
	mr.clearDynamicArea(img.Bounds().Dx()+40, img.Bounds().Dy()+20)
	
	// 显示在标题下方
	x := 20
	y := 60
	mr.fb.DrawImage(img, x, y)
	
	return nil
}

// clearDynamicArea 清除动态内容区域
func (mr *MenuRenderer) clearDynamicArea(width, height int) {
	x := 20
	y := 60
	
	// 只清除动态内容区域，而不是整个屏幕
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			mr.fb.SetPixel(x+dx, y+dy, color.RGBA{0, 0, 0, 255})
		}
	}
}

func (mr *MenuRenderer) RenderConfigMenu() error {
	mr.fb.Clear()
	
	// 标记需要重新渲染主菜单
	mr.needsClear = true
	mr.staticRendered = false

	// 使用14号字体
	mr.renderer.SetSize(14)
	
	content := mr.generateConfigMenuContent()
	lines := strings.Split(content, "\n")
	
	img, err := mr.renderer.RenderMultilineText(lines, color.RGBA{255, 255, 255, 255}, 3)
	if err != nil {
		return fmt.Errorf("failed to render config menu: %v", err)
	}

	// 左上角左对齐显示，留出边距
	x := 20
	y := 20

	mr.fb.DrawImage(img, x, y)
	return nil
}

// InvalidateCache 使缓存失效，强制重新渲染
func (mr *MenuRenderer) InvalidateCache() {
	mr.needsClear = true
	mr.staticRendered = false
	mr.lastContent = ""
}

func (mr *MenuRenderer) RenderNetworkInfo(interfaces []system.NetworkInterface) error {
	mr.fb.Clear()

	// 使用14号字体
	mr.renderer.SetSize(14)
	
	content := mr.generateNetworkInfoContent(interfaces)
	lines := strings.Split(content, "\n")
	
	img, err := mr.renderer.RenderMultilineText(lines, color.RGBA{255, 255, 255, 255}, 3)
	if err != nil {
		return fmt.Errorf("failed to render network info: %v", err)
	}

	// 左上角左对齐显示，留出边距
	x := 20
	y := 20

	mr.fb.DrawImage(img, x, y)
	return nil
}

func (mr *MenuRenderer) RenderMessage(message string) error {
	mr.fb.Clear()

	// 使用14号字体
	mr.renderer.SetSize(14)
	
	lines := strings.Split(message, "\n")
	
	img, err := mr.renderer.RenderMultilineText(lines, color.RGBA{255, 255, 255, 255}, 3)
	if err != nil {
		return fmt.Errorf("failed to render message: %v", err)
	}

	// 左上角左对齐显示，留出边距
	x := 20
	y := 20

	mr.fb.DrawImage(img, x, y)
	return nil
}

func (mr *MenuRenderer) generateMainMenuContent(sysInfo *system.SystemInfo) string {
	return fmt.Sprintf(
		"=== 系统状态监控 ===\n\n"+
			"运行时间: %s\n"+
			"处理器: %s (%d 核心)\n"+
			"内存使用: %s\n"+
			"磁盘大小: %s (共 %d 个磁盘)\n"+
			"系统时间: %s\n"+
			"IP地址: %s\n\n"+
			"操作指南:\n"+
			"- 按回车键(Enter): 进入配置菜单\n"+
			"- 按 Ctrl+C: 退出程序\n"+
			"- 系统状态每5秒自动更新",
		sysInfo.Uptime,
		sysInfo.CPUModel,
		sysInfo.CPUCores,
		sysInfo.MemoryUsage,
		sysInfo.DiskSize,
		sysInfo.DiskCount,
		sysInfo.CurrentTime,
		sysInfo.IPAddress,
	)
}

func (mr *MenuRenderer) generateConfigMenuContent() string {
	return "============================\n" +
		"配置菜单\n" +
		"============================\n" +
		"1. 查看网卡信息\n" +
		"2. 重启系统服务\n" +
		"3. 检测设备网络\n" +
		"4. 重启设备\n" +
		"5. 关机\n" +
		"============================\n" +
		"请输入选项(1-5)，按q返回首页"
}

func (mr *MenuRenderer) generateNetworkInfoContent(interfaces []system.NetworkInterface) string {
	content := "============================\n"
	content += "网卡信息\n"
	content += "============================\n"
	
	for _, iface := range interfaces {
		content += fmt.Sprintf("网卡: %s\n", iface.Name)
		content += fmt.Sprintf("状态: %s\n", iface.Status)
		if iface.IPv4 != "" {
			content += fmt.Sprintf("IPv4: %s\n", iface.IPv4)
		}
		if iface.IPv6 != "" {
			content += fmt.Sprintf("IPv6: %s\n", iface.IPv6)
		}
		content += "----------------------------\n"
	}
	
	content += "按任意键返回配置菜单"
	return content
}

func (mr *MenuRenderer) generateBuddha() string {
	return `
                    _ooOoo_
                   o8888888o
                   88" . "88
                   (| -_- |)
                   O\  =  /O
                ____/` + "`" + `---'/____
              .'  \\|     |//  ` + "`" + `.
             /  \\|||  :  |||//  \
            /  _||||| -:- |||||-  \
            |   | \\\  -  /// |   |
            | \_|  ''---''  |   |
            \  .-\__  ` + "`" + `-` + "`" + `  ___/-. /
          ___` + "`" + `. .'  /--.--\  ` + "`" + `. . ___
       ."" '<  ` + "`" + `.___\_<|>_/___.'  >'"".
      | | :  ` + "`" + `- ` + "`" + `.;` + "`" + `\ _ /` + "`" + `/;.;` + "`" + `/ - ` + "`" + ` : | |
      \  \ ` + "`" + `-.   \_ __ \ /__ _/   .;` + "`" + ` /  /
  ======` + "`" + `-.____` + "`" + `-.___\_____/___.-` + "`" + `____.-'======
                    ` + "`" + `=---='
  ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
           佛祖保佑       永不宕机`
}

func (mr *MenuRenderer) ShowProgressBar(progress float64, message string) error {
	mr.fb.Clear()

	mr.renderer.SetSize(18)
	
	barWidth := 400
	barHeight := 30
	
	barX := (mr.width - barWidth) / 2
	barY := mr.height / 2
	
	img := image.NewRGBA(image.Rect(0, 0, mr.width, mr.height))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.Point{}, draw.Src)
	
	// 优化：使用更高效的矩形绘制方法
	mr.drawRect(img, barX, barY, barWidth, barHeight, color.RGBA{255, 255, 255, 255}, true)
	
	// 绘制进度条填充部分
	fillWidth := int(float64(barWidth-4) * progress)
	if fillWidth > 0 {
		mr.drawRect(img, barX+2, barY+2, fillWidth, barHeight-4, color.RGBA{0, 255, 0, 255}, false)
	}
	
	if message != "" {
		textImg, err := mr.renderer.RenderText(message, color.RGBA{255, 255, 255, 255})
		if err == nil {
			textBounds := textImg.Bounds()
			textX := (mr.width - textBounds.Dx()) / 2
			textY := barY - 50
			draw.Draw(img, image.Rect(textX, textY, textX+textBounds.Dx(), textY+textBounds.Dy()), 
				textImg, textBounds.Min, draw.Over)
		}
	}
	
	mr.fb.DrawImage(img, 0, 0)
	return nil
}

// drawRect 高效绘制矩形的辅助方法
func (mr *MenuRenderer) drawRect(img *image.RGBA, x, y, width, height int, col color.RGBA, outline bool) {
	if outline {
		// 绘制边框
		for i := 0; i < width; i++ {
			img.Set(x+i, y, col)                // 上边
			img.Set(x+i, y+height-1, col)      // 下边
		}
		for i := 0; i < height; i++ {
			img.Set(x, y+i, col)               // 左边
			img.Set(x+width-1, y+i, col)       // 右边
		}
	} else {
		// 填充矩形
		for j := 0; j < height; j++ {
			for i := 0; i < width; i++ {
				img.Set(x+i, y+j, col)
			}
		}
	}
}
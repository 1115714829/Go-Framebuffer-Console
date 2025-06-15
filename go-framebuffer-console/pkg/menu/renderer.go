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
	"rsc.io/qr"
)

type MenuRenderer struct {
	fb       *framebuffer.FrameBuffer
	renderer *font.Renderer
	width    int
	height   int
	// 智能刷新相关
	lastContent       string // 上次显示的内容
	needsClear        bool   // 是否需要清屏
	staticRendered    bool   // 静态内容是否已渲染
	lastDynamicHeight int    // 上次动态区域的高度，用于清除残留
}

func NewMenuRenderer(fb *framebuffer.FrameBuffer, fontRenderer *font.Renderer) *MenuRenderer {
	width, height := fb.GetDimensions()
	return &MenuRenderer{
		fb:                fb,
		renderer:          fontRenderer,
		width:             width,
		height:            height,
		needsClear:        true, // 初始需要清屏
		staticRendered:    false,
		lastDynamicHeight: 0,
	}
}

func (mr *MenuRenderer) RenderMainMenu(sysInfo *system.SystemInfo) error {
	// 使用14号字体
	mr.renderer.SetSize(14)

	// 生成当前内容
	currentContent := mr.generateNewMainMenuContent(sysInfo)

	// 检查是否需要刷新
	if currentContent == mr.lastContent && mr.staticRendered {
		return nil // 内容没有变化，无需刷新
	}

	// 清屏并重新渲染
	mr.fb.Clear()
	mr.needsClear = false

	// 按新格式渲染整个主菜单
	if err := mr.renderNewMainMenu(sysInfo); err != nil {
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

	mr.lastDynamicHeight = img.Bounds().Dy()
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
	clearHeight := mr.lastDynamicHeight
	if img.Bounds().Dy() > clearHeight {
		clearHeight = img.Bounds().Dy()
	}
	// 清除一个足够宽的区域，同时使用上次和本次渲染中更高的高度，确保完全覆盖
	mr.clearDynamicArea(mr.width-40, clearHeight+20)

	// 显示在标题下方
	x := 20
	y := 60
	mr.fb.DrawImage(img, x, y)

	mr.lastDynamicHeight = img.Bounds().Dy()
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
	if len(interfaces) == 0 {
		return "未找到任何物理网络接口。\n\n按任意键返回"
	}

	var builder strings.Builder
	builder.WriteString("物理网卡信息:\n")
	builder.WriteString("========================================\n")

	for _, iface := range interfaces {
		builder.WriteString(fmt.Sprintf("接口名称: %s\n", iface.Name))
		builder.WriteString(fmt.Sprintf("  状态: %s\n", iface.Status))
		builder.WriteString(fmt.Sprintf("  MAC地址: %s\n", iface.MAC))

		builder.WriteString("  IPv4地址:\n")
		if iface.IPv4Address != "" {
			builder.WriteString(fmt.Sprintf("    - %s\n", iface.IPv4Address))
		} else {
			builder.WriteString("    - (未配置)\n")
		}

		builder.WriteString("  IPv6地址:\n")
		if len(iface.IPv6Addresses) > 0 {
			for _, ip := range iface.IPv6Addresses {
				builder.WriteString(fmt.Sprintf("    - %s\n", ip))
			}
		} else {
			builder.WriteString("    - (未配置)\n")
		}
		builder.WriteString("----------------------------------------\n")
	}
	builder.WriteString("\n按任意键返回")
	return builder.String()
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
      | | :  ` + "`" + `- ` + "`" + `.;` + "`" + `/;.;` + "`" + `/ - ` + "`" + ` : | |
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
			img.Set(x+i, y, col)          // 上边
			img.Set(x+i, y+height-1, col) // 下边
		}
		for i := 0; i < height; i++ {
			img.Set(x, y+i, col)         // 左边
			img.Set(x+width-1, y+i, col) // 右边
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

// generateNewMainMenuContent 生成新的主菜单内容（用于内容比较）
func (mr *MenuRenderer) generateNewMainMenuContent(sysInfo *system.SystemInfo) string {
	return fmt.Sprintf(
		"%s|%s|%d|%s|%s|%d|%s|%s|%s",
		sysInfo.Uptime,
		sysInfo.CPUModel,
		sysInfo.CPUCores,
		sysInfo.MemoryUsage,
		sysInfo.DiskSize,
		sysInfo.DiskCount,
		sysInfo.CurrentTime,
		sysInfo.IPAddress,
		sysInfo.QianKunCloudID,
	)
}

// renderNewMainMenu 按新格式渲染主菜单
func (mr *MenuRenderer) renderNewMainMenu(sysInfo *system.SystemInfo) error {
	// 计算汉字宽度作为上边距
	_, charHeight := mr.renderer.GetTextBounds("字")
	y := charHeight + 10 // 上边距为1个汉字的高度加10像素

	// 1. 系统信息标题
	titleContent := "系统信息"
	if err := mr.renderTextAt(titleContent, 20, y); err != nil {
		return err
	}
	y += charHeight + 5

	// 2. 第一条分隔线
	separatorLine := "================================"
	if err := mr.renderTextAt(separatorLine, 20, y); err != nil {
		return err
	}
	y += charHeight + 5

	// 3. 系统信息内容
	systemContent := []string{
		fmt.Sprintf("操作系统运行时间：%s", sysInfo.Uptime),
		fmt.Sprintf("处理器型号：%s *%d 核", sysInfo.CPUModel, sysInfo.CPUCores),
		fmt.Sprintf("内存使用状态：%s", sysInfo.MemoryUsage),
		fmt.Sprintf("系统安装磁盘大小：%s（共%d个磁盘）", sysInfo.DiskSize, sysInfo.DiskCount),
		fmt.Sprintf("当前系统时间：%s", sysInfo.CurrentTime),
		fmt.Sprintf("设备IP地址：%s", sysInfo.IPAddress),
		"",
		fmt.Sprintf("设备ID：%s", sysInfo.QianKunCloudID),
	}

	for _, line := range systemContent {
		if err := mr.renderTextAt(line, 20, y); err != nil {
			return err
		}
		y += charHeight + 3
	}

	// 4. 第二条分隔线
	if err := mr.renderTextAt(separatorLine, 20, y); err != nil {
		return err
	}
	y += charHeight + 10

	// 5. 生成并显示二维码
	if sysInfo.QianKunCloudID != "" && sysInfo.QianKunCloudID != "未获取到" {
		qrY, err := mr.renderQRCode(sysInfo.QianKunCloudID, 20, y)
		if err != nil {
			return err
		}
		y = qrY + 20
	} else {
		// 如果无法获取设备ID，显示提示信息
		if err := mr.renderTextAt("二维码生成失败：无法获取乾坤云设备ID", 20, y); err != nil {
			return err
		}
		y += charHeight + 20
	}

	// 6. 第三条分隔线
	separatorLine2 := "==============================="
	if err := mr.renderTextAt(separatorLine2, 20, y); err != nil {
		return err
	}
	y += charHeight + 10

	// 7. 客服信息
	customerServiceContent := []string{
		"如有问题请咨询技术客服：微信：your-service-wechat",
		"",
		"按回车键进入配置菜单",
	}

	for _, line := range customerServiceContent {
		if err := mr.renderTextAt(line, 20, y); err != nil {
			return err
		}
		y += charHeight + 3
	}

	return nil
}

// renderTextAt 在指定位置渲染文本
func (mr *MenuRenderer) renderTextAt(text string, x, y int) error {
	if text == "" {
		return nil // 空行不渲染
	}

	textImg, err := mr.renderer.RenderText(text, color.RGBA{255, 255, 255, 255})
	if err != nil {
		return fmt.Errorf("failed to render text '%s': %v", text, err)
	}

	mr.fb.DrawImage(textImg, x, y)
	return nil
}

// renderQRCode 生成并渲染二维码
func (mr *MenuRenderer) renderQRCode(content string, x, y int) (int, error) {
	// 计算二维码的显示区域
	currentY := y
	
	// 显示二维码说明
	headerText := "此处为二维码展示，二维码的值为设备ID"
	if err := mr.renderTextAt(headerText, x, currentY); err != nil {
		return currentY, err
	}
	
	_, charHeight := mr.renderer.GetTextBounds("字")
	currentY += charHeight + 10
	
	// 使用rsc.io/qr生成二维码
	code, err := qr.Encode(content, qr.M)
	if err != nil {
		// 如果生成失败，显示错误信息
		if err := mr.renderTextAt(fmt.Sprintf("二维码生成失败: %v", err), x, currentY); err != nil {
			return currentY, err
		}
		return currentY + charHeight, nil
	}
	
	// 计算二维码尺寸
	qrSize := code.Size
	pixelSize := 4 // 每个二维码像素放大4倍
	border := 2 * pixelSize // 左右边距各2个像素单位
	
	// 创建二维码图像（白色背景）
	totalWidth := qrSize*pixelSize + border*2
	totalHeight := qrSize*pixelSize + border*2
	
	qrImg := image.NewRGBA(image.Rect(0, 0, totalWidth, totalHeight))
	
	// 填充白色背景
	draw.Draw(qrImg, qrImg.Bounds(), &image.Uniform{color.RGBA{255, 255, 255, 255}}, image.Point{}, draw.Src)
	
	// 绘制二维码像素
	for qy := 0; qy < qrSize; qy++ {
		for qx := 0; qx < qrSize; qx++ {
			if code.Black(qx, qy) {
				// 绘制黑色像素块
				for py := 0; py < pixelSize; py++ {
					for px := 0; px < pixelSize; px++ {
						imgX := border + qx*pixelSize + px
						imgY := border + qy*pixelSize + py
						qrImg.Set(imgX, imgY, color.RGBA{0, 0, 0, 255})
					}
				}
			}
		}
	}
	
	// 将二维码图像绘制到帧缓冲区
	mr.fb.DrawImage(qrImg, x, currentY)
	
	// 返回二维码结束位置
	return currentY + totalHeight, nil
}

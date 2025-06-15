// main包是Go Framebuffer Console的主程序入口
// 负责初始化各个模块并协调整个应用程序的运行流程
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"go-framebuffer-console/internal/config"
	"go-framebuffer-console/pkg/font"
	"go-framebuffer-console/pkg/framebuffer"
	"go-framebuffer-console/pkg/input"
	"go-framebuffer-console/pkg/menu"
	"go-framebuffer-console/pkg/system"
)

func initLog() {
	logFile, err := os.OpenFile("console.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("无法打开日志文件: %v", err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("==========================================")
	log.Println("日志系统初始化完成")
}

// Application 主应用程序结构体
// 包含了程序运行所需的所有核心组件
type Application struct {
	config         *config.Config           // 配置管理器
	fb             *framebuffer.FrameBuffer // 帧缓冲区操作对象
	fontRenderer   *font.Renderer           // 字体渲染器
	keyboard       *input.KeyboardInput     // 键盘输入处理器
	menuRenderer   *menu.MenuRenderer       // 菜单渲染器
	ctx            context.Context          // 上下文管理器
	cancel         context.CancelFunc       // 取消函数
	mu             sync.RWMutex             // 读写锁
	running        bool                     // 运行状态
	keyEventChan   chan byte                // 键盘事件通道
	disableCtrlC   bool                     // 是否禁用Ctrl+C退出功能
}

// main 主函数 - 程序入口点
// 负责初始化应用程序并启动主运行循环
func main() {
	// 解析命令行参数
	var disableCtrlC = flag.Bool("d", false, "禁用Ctrl+C退出功能，使程序持续运行")
	var showHelp = flag.Bool("h", false, "显示帮助信息")
	flag.Usage = printUsage
	flag.Parse()

	// 显示帮助信息
	if *showHelp {
		printUsage()
		return
	}

	initLog()

	// 记录启动参数
	log.Printf("程序启动，参数: 禁用Ctrl+C = %v", *disableCtrlC)

	// 创建并初始化应用程序
	app, err := NewApplication(*disableCtrlC)
	if err != nil {
		log.Fatalf("应用程序初始化失败: %v", err)
	}
	log.Printf("应用程序初始化成功，禁用Ctrl+C = %v", app.disableCtrlC)
	// 确保程序退出时清理资源
	defer func() {
		if r := recover(); r != nil {
			log.Printf("程序异常退出: %v", r)
		}
		app.Cleanup()
	}()

	// 设置信号处理器，优雅处理中断信号
	app.setupSignalHandler()

	// 启动主程序循环
	if err := app.Run(); err != nil {
		log.Printf("应用程序运行错误: %v", err)
	}
}

// printUsage 打印使用帮助信息
func printUsage() {
	fmt.Printf("Go Framebuffer Console - 系统状态监控应用\n\n")
	fmt.Printf("用法:\n")
	fmt.Printf("  %s [选项]\n\n", os.Args[0])
	fmt.Printf("选项:\n")
	fmt.Printf("  -d    禁用Ctrl+C退出功能，使程序持续运行（默认启用Ctrl+C退出）\n")
	fmt.Printf("  -h    显示此帮助信息\n\n")
	fmt.Printf("示例:\n")
	fmt.Printf("  %s           # 正常运行，支持Ctrl+C退出\n", os.Args[0])
	fmt.Printf("  %s -d        # 运行并禁用Ctrl+C退出功能\n", os.Args[0])
	fmt.Printf("  %s -h        # 显示帮助信息\n\n", os.Args[0])
	fmt.Printf("说明:\n")
	fmt.Printf("  - 默认情况下，可以使用Ctrl+C或在配置菜单中退出程序\n")
	fmt.Printf("  - 使用-d参数后，只能通过配置菜单退出程序\n")
	fmt.Printf("  - 程序每5秒自动刷新系统状态信息\n")
	fmt.Printf("  - 按回车键进入配置菜单进行系统管理\n")
}

func NewApplication(disableCtrlC bool) (*Application, error) {
	ctx, cancel := context.WithCancel(context.Background())
	app := &Application{
		config:       config.NewConfig(),
		ctx:          ctx,
		cancel:       cancel,
		running:      false,
		keyEventChan: make(chan byte, 1),
		disableCtrlC: disableCtrlC,
	}

	// 1. 首先初始化Framebuffer来获取屏幕尺寸
	if err := app.initFramebuffer(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize framebuffer: %v", err)
	}

	// 2. 根据屏幕高度动态计算字体大小
	width, height := app.fb.GetDimensions()
	log.Printf("检测到屏幕分辨率: %d x %d", width, height)

	// 根据用户要求，使用固定的14号字体
	app.config.FontSize = 14.0
	log.Printf("使用固定字体大小: %.2f", app.config.FontSize)

	// 3. 使用动态计算出的字体大小初始化字体渲染器
	if err := app.initFontRenderer(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize font renderer: %v", err)
	}

	// 4. 初始化键盘
	if err := app.initKeyboard(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize keyboard: %v", err)
	}

	// 5. 初始化菜单渲染器
	app.menuRenderer = menu.NewMenuRenderer(app.fb, app.fontRenderer)

	return app, nil
}

func (app *Application) initFramebuffer() error {
	device := framebuffer.GetBestFramebufferDevice()
	fb, err := framebuffer.NewFrameBuffer(device)
	if err != nil {
		return err
	}
	app.fb = fb
	return nil
}

func (app *Application) initFontRenderer() error {
	renderer, err := font.NewRenderer(app.config.FontPath, app.config.FontSize, app.config.DPI)
	if err != nil {
		return err
	}
	app.fontRenderer = renderer
	return nil
}

func (app *Application) initKeyboard() error {
	keyboard, err := input.NewKeyboardInput()
	if err != nil {
		return err
	}
	app.keyboard = keyboard
	return nil
}

func (app *Application) setupSignalHandler() {
	c := make(chan os.Signal, 1)
	// 监听所有可能导致程序退出的信号
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGTSTP, syscall.SIGQUIT)
	go func() {
		for {
			select {
			case sig := <-c:
				// 如果禁用了退出功能，则拦截所有退出信号
				if app.disableCtrlC {
					log.Printf("接收到信号: %v，但退出功能已禁用，继续运行", sig)
					continue // 不退出，继续监听
				}
				log.Printf("接收到信号: %v，开始优雅退出", sig)
				app.mu.Lock()
				app.running = false
				app.mu.Unlock()
				app.cancel()
				// 给程序时间进行清理
				time.Sleep(1 * time.Second)
				app.Cleanup()
				os.Exit(0)
			case <-app.ctx.Done():
				return
			}
		}
	}()
}

func (app *Application) startKeyboardListener() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("键盘监听goroutine异常: %v", r)
		}
	}()

	for {
		// 检查上下文是否已取消
		select {
		case <-app.ctx.Done():
			return
		default:
		}

		// 使用带超时的非阻塞读取，避免永久阻塞
		key, available, err := app.keyboard.ReadKeyNonBlockingWithTimeout(100 * time.Millisecond)
		if err != nil {
			if app.isContextError(err) || !app.isRunning() {
				return
			}
			// 只有在不是预期的中断错误时才记录日志
			if !strings.Contains(err.Error(), "interrupted system call") && !strings.Contains(err.Error(), "select调用失败") {
				log.Printf("读取键盘输入时发生错误: %v", err)
			}
			continue
		}

		if available {
			// 将按键事件发送到通道
			select {
			case app.keyEventChan <- key:
			case <-app.ctx.Done():
				return
			case <-time.After(50 * time.Millisecond):
				// 超时，防止阻塞
			}
		}
	}
}

func (app *Application) Run() error {
	app.mu.Lock()
	app.running = true
	app.mu.Unlock()

	// 启动键盘监听
	go app.startKeyboardListener()

	// 创建5秒定时器用于自动刷新
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// 立即显示第一次系统状态
	if err := app.showMainMenu(); err != nil {
		return fmt.Errorf("初始显示主菜单失败: %v", err)
	}

	log.Printf("系统状态监控已启动，每5秒自动刷新")

	for {
		select {
		case <-app.ctx.Done():
			log.Printf("接收到退出信号，程序即将退出")
			return nil
		case <-ticker.C:
			// 5秒定时器触发，刷新系统状态
			if app.isRunning() {
				// 强制使缓存失效，确保重新渲染
				app.menuRenderer.InvalidateCache()
				if err := app.showMainMenu(); err != nil {
					log.Printf("自动刷新系统状态失败: %v", err)
				}
			}
		case key := <-app.keyEventChan:
			// 如果程序当前不在运行状态（例如在配置菜单中），则忽略按键
			if !app.isRunning() {
				continue
			}
			// 处理键盘输入
			switch key {
			case '\n', '\r':
				// 按下回车键，进入配置菜单
				log.Printf("检测到回车键，进入配置菜单")
				if err := app.enterConfigMenu(ticker); err != nil {
					log.Printf("配置菜单操作失败: %v", err)
				}
				// 从配置菜单返回后，立即刷新主菜单
				app.menuRenderer.InvalidateCache()
				if err := app.showMainMenu(); err != nil {
					log.Printf("返回主菜单时刷新失败: %v", err)
				}
			case 3: // Ctrl+C
				if !app.disableCtrlC {
					log.Printf("在主页面检测到Ctrl+C，程序即将退出")
					app.cancel()
				} else {
					log.Printf("在主页面检测到Ctrl+C，但退出功能已禁用")
				}
			case 26: // Ctrl+Z
				if app.disableCtrlC {
					log.Printf("在主页面检测到Ctrl+Z，但退出功能已禁用")
				} else {
					log.Printf("在主页面检测到Ctrl+Z，程序即将退出")
					app.cancel()
				}
			case 28: // Ctrl+\
				if app.disableCtrlC {
					log.Printf("在主页面检测到Ctrl+\\，但退出功能已禁用")
				} else {
					log.Printf("在主页面检测到Ctrl+\\，程序即将退出")
					app.cancel()
				}
			case 4: // Ctrl+D (EOF)
				if app.disableCtrlC {
					log.Printf("在主页面检测到Ctrl+D，但退出功能已禁用")
				} else {
					log.Printf("在主页面检测到Ctrl+D，程序即将退出")
					app.cancel()
				}
			}
		}
	}
}

func (app *Application) showMainMenu() error {
	sysInfo, err := system.GetSystemInfo()
	if err != nil {
		return fmt.Errorf("failed to get system info: %v", err)
	}

	return app.menuRenderer.RenderMainMenu(sysInfo)
}

func (app *Application) showConfigMenu() error {
	return app.menuRenderer.RenderConfigMenu()
}

func (app *Application) handleMenuChoice(choice int) error {
	switch choice {
	case 1:
		return app.showNetworkInfo()
	case 2:
		return app.showSystemServiceMenu()
	case 3:
		return app.testNetworkConnectivity()
	case 4:
		return app.confirmAndReboot()
	case 5:
		return app.confirmAndShutdown()
	default:
		return app.showMessage("无效选项，请重新选择")
	}
}

func (app *Application) showNetworkInfo() error {
	interfaces, err := system.GetNetworkInterfaces()
	if err != nil {
		return app.showMessage(fmt.Sprintf("获取网卡信息失败: %v", err))
	}

	if err := app.menuRenderer.RenderNetworkInfo(interfaces); err != nil {
		return err
	}

	// 循环等待按键，处理控制键
	for {
		key, err := app.keyboard.ReadKey()
		if err != nil {
			return err
		}
		
		// 处理控制键
		if app.handleControlKey(key, "网卡信息页面") {
			return nil // 控制键触发退出
		}
		
		// 其他任意按键都返回
		return nil
	}
}

func (app *Application) showSystemServiceMenu() error {
	message := "系统服务管理\n\n" +
		"此功能暂时未实现\n" +
		"将来可以添加以下功能：\n" +
		"- 重启网络服务\n" +
		"- 重启SSH服务\n" +
		"- 重启防火墙服务\n" +
		"- 查看服务状态\n\n" +
		"按任意键返回"

	if err := app.menuRenderer.RenderMessage(message); err != nil {
		return err
	}

	// 循环等待按键，处理控制键
	for {
		key, err := app.keyboard.ReadKey()
		if err != nil {
			return err
		}
		
		// 处理控制键
		if app.handleControlKey(key, "系统服务菜单页面") {
			return nil // 控制键触发退出
		}
		
		// 其他任意按键都返回
		return nil
	}
}

func (app *Application) testNetworkConnectivity() error {
	// 显示开始测试的消息
	if err := app.menuRenderer.RenderMessage("正在初始化网络连通性测试...\n\n请稍候..."); err != nil {
		return err
	}

	// 创建进度回调函数
	progressCallback := func(target string, current, total int, message string) {
		progressText := fmt.Sprintf("网络连通性测试进度: %d/%d\n\n当前测试: %s\n%s", current, total, target, message)
		app.menuRenderer.RenderMessage(progressText)
	}

	// 执行高级网络测试
	results, err := system.TestAdvancedNetworkConnectivity(progressCallback)
	if err != nil {
		message := fmt.Sprintf("网络测试执行失败: %v\n\n按任意键返回", err)
		if err := app.menuRenderer.RenderMessage(message); err != nil {
			return err
		}
		_, err = app.keyboard.ReadKey()
		return err
	}

	// 格式化并显示测试结果
	resultMessage := app.formatNetworkTestResults(results)
	if err := app.menuRenderer.RenderMessage(resultMessage); err != nil {
		return err
	}

	// 循环等待按键，处理控制键
	for {
		key, err := app.keyboard.ReadKey()
		if err != nil {
			return err
		}
		
		// 处理控制键
		if app.handleControlKey(key, "网络测试结果页面") {
			return nil // 控制键触发退出
		}
		
		// 其他任意按键都返回
		return nil
	}
}

// formatNetworkTestResults 格式化网络测试结果
func (app *Application) formatNetworkTestResults(results []system.NetworkTestResult) string {
	var builder strings.Builder
	builder.WriteString("=== 网络连通性测试结果 ===\n\n")

	successCount := 0
	for _, result := range results {
		// 状态显示
		status := "异常"
		if result.Success && result.PacketLoss == 0 {
			status = "正常"
			successCount++
		} else if result.Success && result.PacketLoss > 0 {
			status = "部分正常"
		}

		builder.WriteString(fmt.Sprintf("• %s (%s):\n", result.Target.Name, result.Target.Host))
		builder.WriteString(fmt.Sprintf("  状态: %s\n", status))
		
		if result.Success || result.PacketsRecv > 0 {
			builder.WriteString(fmt.Sprintf("  数据包: 发送%d 接收%d 丢失%.1f%%\n", 
				result.PacketsSent, result.PacketsRecv, result.PacketLoss))
			if result.AvgLatency != "N/A" && result.AvgLatency != "" {
				builder.WriteString(fmt.Sprintf("  平均延迟: %s\n", result.AvgLatency))
			}
		}
		
		if result.ErrorMsg != "" {
			builder.WriteString(fmt.Sprintf("  详情: %s\n", result.ErrorMsg))
		}
		builder.WriteString("\n")
	}

	// 总结
	builder.WriteString("----------------------------------------\n")
	if successCount == len(results) {
		builder.WriteString("✓ 网络连接状态: 良好\n")
		builder.WriteString("所有测试目标均可正常访问")
	} else if successCount > 0 {
		builder.WriteString("⚠ 网络连接状态: 部分异常\n")
		builder.WriteString(fmt.Sprintf("可访问 %d/%d 个测试目标", successCount, len(results)))
	} else {
		builder.WriteString("✗ 网络连接状态: 异常\n")
		builder.WriteString("所有测试目标均无法访问")
	}

	builder.WriteString("\n\n按任意键返回")
	return builder.String()
}

func (app *Application) confirmAndReboot() error {
	message := "确认要重启设备吗？\n\n" +
		"按 'y' 确认重启\n" +
		"按任意其他键取消"

	if err := app.menuRenderer.RenderMessage(message); err != nil {
		return err
	}

	// 循环等待按键，处理控制键
	for {
		key, err := app.keyboard.ReadKey()
		if err != nil {
			return err
		}
		
		// 处理控制键
		if app.handleControlKey(key, "重启确认页面") {
			return nil // 控制键触发退出
		}
		
		if key == 'y' || key == 'Y' {
			if err := app.menuRenderer.RenderMessage("正在重启设备..."); err != nil {
				return err
			}

			time.Sleep(2 * time.Second)
			return system.RebootSystem()
		}

		// 其他任意按键都取消
		return nil
	}
}

func (app *Application) confirmAndShutdown() error {
	message := "确认要关机吗？\n\n" +
		"按 'y' 确认关机\n" +
		"按任意其他键取消"

	if err := app.menuRenderer.RenderMessage(message); err != nil {
		return err
	}

	// 循环等待按键，处理控制键
	for {
		key, err := app.keyboard.ReadKey()
		if err != nil {
			return err
		}
		
		// 处理控制键
		if app.handleControlKey(key, "关机确认页面") {
			return nil // 控制键触发退出
		}
		
		if key == 'y' || key == 'Y' {
			if err := app.menuRenderer.RenderMessage("正在关机..."); err != nil {
				return err
			}

			time.Sleep(2 * time.Second)
			return system.ShutdownSystem()
		}

		// 其他任意按键都取消
		return nil
	}
}

func (app *Application) showMessage(message string) error {
	fullMessage := message + "\n\n按任意键继续"
	if err := app.menuRenderer.RenderMessage(fullMessage); err != nil {
		return err
	}

	// 循环等待按键，处理控制键
	for {
		key, err := app.keyboard.ReadKey()
		if err != nil {
			return err
		}
		
		// 处理控制键
		if app.handleControlKey(key, "消息页面") {
			return nil // 控制键触发退出
		}
		
		// 其他任意按键都返回
		return nil
	}
}

func (app *Application) enterConfigMenu(ticker *time.Ticker) error {
	// 标记程序状态为非运行（暂停主界面的任何活动）
	app.mu.Lock()
	app.running = false
	app.mu.Unlock()

	// 函数退出时恢复状态并重启定时器
	defer func() {
		app.mu.Lock()
		app.running = true
		app.mu.Unlock()
		ticker.Reset(5 * time.Second)
		log.Printf("已退出配置菜单，恢复主界面自动刷新")
	}()

	// 暂停主屏幕的自动刷新
	ticker.Stop()
	log.Printf("已进入配置菜单，暂停主界面自动刷新")

	for {
		// 显示配置菜单
		if err := app.showConfigMenu(); err != nil {
			return fmt.Errorf("显示配置菜单失败: %v", err)
		}

		// 等待用户选择 (1-5, q)
		// 注意：这里的WaitForKey是阻塞的，它会阻止Run循环的进行
		// 但由于我们在独立的goroutine中监听键盘，这里需要换一种方式
		// 我们改为从keyEventChan读取
		select {
		case key := <-app.keyEventChan:
			// 处理控制键
			if app.handleControlKey(key, "配置菜单") {
				return nil // 控制键触发退出
			}
			
			var choice int
			switch key {
			case '1', '2', '3', '4', '5':
				choice = int(key - '0')
			case 'q', 'Q', 27: // q, Q, ESC
				return nil // 退出配置菜单
			default:
				continue // 忽略其他键
			}

			// 处理菜单选择
			if err := app.handleMenuChoice(choice); err != nil {
				log.Printf("处理菜单选择失败: %v", err)
				// 显示错误信息后继续
				app.showMessage(fmt.Sprintf("操作失败: %v", err))
			}
		case <-app.ctx.Done():
			return nil
		}
	}
}

func (app *Application) isContextError(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
}

// handleControlKey 处理控制键，如果禁用了退出功能则拦截，否则退出程序
// 返回true表示应该退出当前函数，false表示继续处理
func (app *Application) handleControlKey(key byte, location string) bool {
	switch key {
	case 3: // Ctrl+C
		if app.disableCtrlC {
			log.Printf("在%s检测到Ctrl+C，但退出功能已禁用", location)
			return false // 继续运行
		} else {
			log.Printf("在%s检测到Ctrl+C，程序即将退出", location)
			app.cancel()
			return true // 退出当前函数
		}
	case 26: // Ctrl+Z
		if app.disableCtrlC {
			log.Printf("在%s检测到Ctrl+Z，但退出功能已禁用", location)
			return false // 继续运行
		} else {
			log.Printf("在%s检测到Ctrl+Z，程序即将退出", location)
			app.cancel()
			return true // 退出当前函数
		}
	case 28: // Ctrl+\
		if app.disableCtrlC {
			log.Printf("在%s检测到Ctrl+\\，但退出功能已禁用", location)
			return false // 继续运行
		} else {
			log.Printf("在%s检测到Ctrl+\\，程序即将退出", location)
			app.cancel()
			return true // 退出当前函数
		}
	case 4: // Ctrl+D (EOF)
		if app.disableCtrlC {
			log.Printf("在%s检测到Ctrl+D，但退出功能已禁用", location)
			return false // 继续运行
		} else {
			log.Printf("在%s检测到Ctrl+D，程序即将退出", location)
			app.cancel()
			return true // 退出当前函数
		}
	}
	return false // 不是控制键，继续处理
}

func (app *Application) isRunning() bool {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.running
}

func (app *Application) Cleanup() {
	app.mu.Lock()
	defer app.mu.Unlock()

	if app.cancel != nil {
		app.cancel()
	}

	if app.keyboard != nil {
		if err := app.keyboard.RestoreTerminal(); err != nil {
			log.Printf("恢复终端状态失败: %v", err)
		}
		if err := app.keyboard.Close(); err != nil {
			log.Printf("关闭键盘设备失败: %v", err)
		}
		app.keyboard = nil
	}

	if app.fb != nil {
		if err := app.fb.Close(); err != nil {
			log.Printf("关闭帧缓冲区失败: %v", err)
		}
		app.fb = nil
	}

	app.running = false
}

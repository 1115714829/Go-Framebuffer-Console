// main包是Go Framebuffer Console的主程序入口
// 负责初始化各个模块并协调整个应用程序的运行流程
package main

import (
	"context"
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
	log.Println("日志系统初始化完成")
}

// Application 主应用程序结构体
// 包含了程序运行所需的所有核心组件
type Application struct {
	config       *config.Config           // 配置管理器
	fb           *framebuffer.FrameBuffer // 帧缓冲区操作对象
	fontRenderer *font.Renderer           // 字体渲染器
	keyboard     *input.KeyboardInput     // 键盘输入处理器
	menuRenderer *menu.MenuRenderer       // 菜单渲染器
	ctx          context.Context          // 上下文管理器
	cancel       context.CancelFunc       // 取消函数
	mu           sync.RWMutex             // 读写锁
	running      bool                     // 运行状态
	keyEventChan chan byte                // 键盘事件通道
}

// main 主函数 - 程序入口点
// 负责初始化应用程序并启动主运行循环
func main() {
	initLog()

	// 创建并初始化应用程序
	app, err := NewApplication()
	if err != nil {
		log.Fatalf("应用程序初始化失败: %v", err)
	}
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

func NewApplication() (*Application, error) {
	ctx, cancel := context.WithCancel(context.Background())
	app := &Application{
		config:       config.NewConfig(),
		ctx:          ctx,
		cancel:       cancel,
		running:      false,
		keyEventChan: make(chan byte, 1),
	}

	// 1. 首先初始化Framebuffer来获取屏幕尺寸
	if err := app.initFramebuffer(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize framebuffer: %v", err)
	}

	// 2. 根据屏幕高度动态计算字体大小
	width, height := app.fb.GetDimensions()
	log.Printf("检测到屏幕分辨率: %d x %d", width, height)
	// 以768p高度下字体大小为14作为基准
	baseHeight := 768.0
	baseFontSize := 14.0
	dynamicFontSize := baseFontSize * (float64(height) / baseHeight)

	// 限制字体大小在合理范围内 (例如, 10到36)
	if dynamicFontSize < 10 {
		dynamicFontSize = 10
	} else if dynamicFontSize > 36 {
		dynamicFontSize = 36
	}
	app.config.FontSize = dynamicFontSize
	log.Printf("动态设置字体大小为: %.2f", dynamicFontSize)

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
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		select {
		case sig := <-c:
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
				log.Printf("检测到Ctrl+C，程序即将退出")
				app.cancel()
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

	_, err = app.keyboard.ReadKey()
	return err
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

	_, err := app.keyboard.ReadKey()
	return err
}

func (app *Application) testNetworkConnectivity() error {
	if err := app.menuRenderer.RenderMessage("正在测试网络连接..."); err != nil {
		return err
	}

	connected, err := system.TestNetworkConnectivity()

	var message string
	if err != nil {
		message = fmt.Sprintf("网络测试失败: %v\n\n按任意键返回", err)
	} else if connected {
		message = "网络连接正常！\n\n按任意键返回"
	} else {
		message = "网络连接异常！\n\n按任意键返回"
	}

	if err := app.menuRenderer.RenderMessage(message); err != nil {
		return err
	}

	_, err = app.keyboard.ReadKey()
	return err
}

func (app *Application) confirmAndReboot() error {
	message := "确认要重启设备吗？\n\n" +
		"按 'y' 确认重启\n" +
		"按任意其他键取消"

	if err := app.menuRenderer.RenderMessage(message); err != nil {
		return err
	}

	key, err := app.keyboard.ReadKey()
	if err != nil {
		return err
	}

	if key == 'y' || key == 'Y' {
		if err := app.menuRenderer.RenderMessage("正在重启设备..."); err != nil {
			return err
		}

		time.Sleep(2 * time.Second)
		return system.RebootSystem()
	}

	return nil
}

func (app *Application) confirmAndShutdown() error {
	message := "确认要关机吗？\n\n" +
		"按 'y' 确认关机\n" +
		"按任意其他键取消"

	if err := app.menuRenderer.RenderMessage(message); err != nil {
		return err
	}

	key, err := app.keyboard.ReadKey()
	if err != nil {
		return err
	}

	if key == 'y' || key == 'Y' {
		if err := app.menuRenderer.RenderMessage("正在关机..."); err != nil {
			return err
		}

		time.Sleep(2 * time.Second)
		return system.ShutdownSystem()
	}

	return nil
}

func (app *Application) showMessage(message string) error {
	fullMessage := message + "\n\n按任意键继续"
	if err := app.menuRenderer.RenderMessage(fullMessage); err != nil {
		return err
	}

	_, err := app.keyboard.ReadKey()
	return err
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
		case <-time.After(30 * time.Second):
			log.Printf("配置菜单超时，自动返回主界面")
			return nil
		}
	}
}

func (app *Application) isContextError(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
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

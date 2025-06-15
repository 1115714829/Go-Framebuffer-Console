// main包是Go Framebuffer Console的主程序入口
// 负责初始化各个模块并协调整个应用程序的运行流程
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
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

// Application 主应用程序结构体
// 包含了程序运行所需的所有核心组件
type Application struct {
	config       *config.Config             // 配置管理器
	fb           *framebuffer.FrameBuffer   // 帧缓冲区操作对象
	fontRenderer *font.Renderer             // 字体渲染器
	keyboard     *input.KeyboardInput       // 键盘输入处理器
	menuRenderer *menu.MenuRenderer         // 菜单渲染器
	ctx          context.Context            // 上下文管理器
	cancel       context.CancelFunc         // 取消函数
	mu           sync.RWMutex               // 读写锁
	running      bool                       // 运行状态
}

// main 主函数 - 程序入口点
// 负责初始化应用程序并启动主运行循环
func main() {
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
		config:  config.NewConfig(),
		ctx:     ctx,
		cancel:  cancel,
		running: false,
	}

	if err := app.initFramebuffer(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize framebuffer: %v", err)
	}

	if err := app.initFontRenderer(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize font renderer: %v", err)
	}

	if err := app.initKeyboard(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize keyboard: %v", err)
	}

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

func (app *Application) Run() error {
	app.mu.Lock()
	app.running = true
	app.mu.Unlock()

	// 创建5秒定时器用于自动刷新
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// 立即显示第一次系统状态
	if err := app.showMainMenu(); err != nil {
		return fmt.Errorf("初始显示主菜单失败: %v", err)
	}

	// 启动键盘输入处理协程
	keyboardChan := make(chan byte, 10)
	keyboardErrChan := make(chan error, 1)
	
	go app.keyboardInputHandler(keyboardChan, keyboardErrChan)

	for {
		select {
		case <-app.ctx.Done():
			return nil
		case <-ticker.C:
			// 5秒定时器触发，刷新系统状态
			app.mu.RLock()
			if app.running {
				app.mu.RUnlock()
				if err := app.showMainMenu(); err != nil {
					// 只在严重错误时记录，避免干扰屏幕
					return fmt.Errorf("刷新系统状态失败: %v", err)
				}
			} else {
				app.mu.RUnlock()
			}
		case key := <-keyboardChan:
			// 处理键盘输入
			if err := app.handleKeypress(key); err != nil {
				if app.isContextError(err) {
					return nil
				}
				return fmt.Errorf("处理键盘输入失败: %v", err)
			}
		case err := <-keyboardErrChan:
			// 键盘输入错误
			if app.isContextError(err) {
				return nil
			}
			return fmt.Errorf("键盘输入错误: %v", err)
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

func (app *Application) checkKeyboardInput() error {
	key, available, err := app.keyboard.ReadKeyNonBlocking()
	if err != nil {
		return fmt.Errorf("读取键盘输入失败: %v", err)
	}
	
	if available {
		switch key {
		case '\n', '\r':
			// 按下回车键，进入配置菜单
			log.Printf("检测到回车键，进入配置菜单")
			return app.enterConfigMenu()
		case 'q', 'Q', 27: // 'q' 或 'Q' 或 ESC 键
			log.Printf("检测到退出键，程序即将退出")
			app.cancel()
			return nil
		default:
			// 忽略其他按键
		}
	}
	
	return nil
}

func (app *Application) enterConfigMenu() error {
	for {
		// 显示配置菜单
		if err := app.showConfigMenu(); err != nil {
			return fmt.Errorf("显示配置菜单失败: %v", err)
		}

		// 等待用户选择
		choice, err := app.keyboard.WaitForMenuChoiceWithTimeout(30 * time.Second)
		if err != nil {
			log.Printf("配置菜单等待用户输入超时: %v", err)
			break // 超时返回主界面
		}

		if choice == -1 {
			// 用户选择退出配置菜单
			break
		}

		// 处理菜单选择
		if err := app.handleMenuChoice(choice); err != nil {
			log.Printf("处理菜单选择失败: %v", err)
			// 显示错误信息后继续
			app.showMessage(fmt.Sprintf("操作失败: %v", err))
		}
	}
	
	// 返回主界面前刷新一次状态
	return app.showMainMenu()
}

func (app *Application) isContextError(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
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
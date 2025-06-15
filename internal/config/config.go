// config包提供了应用程序的配置管理功能
// 定义了字体路径、设备路径等关键配置项的默认值
package config

import (
	"os"
)

// 默认配置常量
// 这些值在程序初始化时使用，可以根据实际部署环境进行调整
const (
	DefaultFontPath = "./fonts/SourceHanSansSC-Regular.ttf" // 默认字体文件路径（TTF格式）
	BackupFontPath  = "./fonts/SourceHanSansSC-Regular.otf" // 备用字体文件路径（OTF格式）
	DefaultFontSize = 20.0                                  // 默认字体大小（点）
	DefaultDPI      = 72.0                                  // 默认DPI分辨率
	DefaultDevice   = "/dev/fb0"                            // 默认帧缓冲区设备路径
)

// Config 应用程序配置结构体
// 包含了程序运行所需的各种配置参数
type Config struct {
	FontPath string  // 字体文件路径
	FontSize float64 // 字体大小
	DPI      float64 // 屏幕分辨率（每英寸点数）
	Device   string  // 帧缓冲区设备路径
}

// NewConfig 创建新的配置对象
// 使用默认配置值初始化配置结构体
// 返回包含默认配置的Config对象
func NewConfig() *Config {
	return &Config{
		FontPath: GetBestFontPath(), // 设置最佳字体路径
		FontSize: DefaultFontSize,   // 设置默认字体大小
		DPI:      DefaultDPI,        // 设置默认DPI
		Device:   DefaultDevice,     // 设置默认设备路径
	}
}

// GetBestFontPath 获取最佳的字体文件路径
// 优先选择TTF格式，如果不存在则尝试OTF格式
func GetBestFontPath() string {
	// 检查TTF文件是否存在
	if _, err := os.Stat(DefaultFontPath); err == nil {
		return DefaultFontPath
	}
	
	// 检查OTF文件是否存在
	if _, err := os.Stat(BackupFontPath); err == nil {
		return BackupFontPath
	}
	
	// 都不存在时返回默认TTF路径（会在后续处理中给出错误提示）
	return DefaultFontPath
}
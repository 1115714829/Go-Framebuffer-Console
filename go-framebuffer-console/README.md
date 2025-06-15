# Go Framebuffer Console System

一个基于Go语言开发的Linux Framebuffer控制台系统管理程序，支持完美的简体中文字体显示和直接在/dev/console上运行。

## 项目特性

- ✅ **完美简体中文支持**: 使用思源黑体字体，完美显示简体中文字符
- ✅ **直接控制台显示**: 程序运行后直接霸屏在/dev/console上显示
- ✅ **系统信息展示**: 显示运行时间、CPU型号、内存使用、磁盘信息等
- ✅ **ASCII艺术**: 内置Linux佛祖ASCII艺术图案
- ✅ **交互式菜单**: 支持键盘快捷键操作的配置界面
- ✅ **网络管理**: 查看网卡信息和网络连接测试
- ✅ **系统控制**: 支持重启和关机操作
- ✅ **模块化设计**: 分级分模块的代码架构
- ✅ **CentOS 7.9兼容**: 针对生产环境优化

## 项目结构

```
go-framebuffer-console/
├── cmd/main/                    # 主程序入口
│   └── main.go
├── pkg/                         # 核心功能包
│   ├── framebuffer/            # Framebuffer显示模块
│   │   └── framebuffer.go
│   ├── font/                   # 字体渲染模块
│   │   └── renderer.go
│   ├── input/                  # 键盘输入监听模块
│   │   └── keyboard.go
│   ├── system/                 # 系统信息获取模块
│   │   └── info.go
│   └── menu/                   # 菜单渲染和内容准备模块
│       └── renderer.go
├── internal/config/            # 内部配置模块
│   └── config.go
├── go.mod                      # Go模块文件
├── Makefile                    # 构建脚本
└── README.md                   # 项目文档
```

## 系统要求

### 软件要求
- Go 1.21 或更高版本
- Linux操作系统 (推荐CentOS 7.9)
- 支持Framebuffer的显示系统

### 硬件要求
- 至少512MB内存
- 支持Framebuffer的显卡

### 字体文件
- 需要思源黑体字体文件：`/root/go-framebuffer-master/fonts/SourceHanSansSC-Regular.otf`

## 快速开始

### 1. 克隆项目
```bash
cd /mnt/d/console
```

### 2. 安装依赖
```bash
cd go-framebuffer-console
make deps
```

### 3. 编译程序
```bash
make build
```

### 4. 运行程序
```bash
make run
# 或者直接运行
sudo ./framebuffer-console
```

## 功能说明

### 主界面
程序启动后会显示系统信息界面，包含：
- 操作系统运行时间
- 处理器型号和核心数
- 内存使用状态
- 系统磁盘信息
- 当前系统时间
- 设备IP地址
- Linux佛祖ASCII艺术

按回车键进入配置界面。

### 配置界面
提供以下功能选项：
1. **查看网卡信息** - 显示所有网络接口的详细信息
2. **重启系统服务** - 系统服务管理（暂时未实现）
3. **检测设备网络** - 测试网络连接状态
4. **重启设备** - 安全重启系统
5. **关机** - 安全关闭系统

按 `q` 键返回首页。

### 操作说明
- 使用数字键 `1-5` 选择菜单项
- 按 `q` 键返回上级菜单
- 按 `Enter` 键确认选择
- 系统操作需要输入 `y` 确认

## 模块详解

### 1. Framebuffer模块 (`pkg/framebuffer/`)
负责Linux Framebuffer设备的操作：
- 设备初始化和内存映射
- 像素绘制和图像渲染
- 屏幕清除和显示控制

### 2. 字体渲染模块 (`pkg/font/`)
处理中文字体的渲染：
- 支持TrueType/OpenType字体
- 多行文本渲染
- 字体大小和样式控制

### 3. 键盘输入模块 (`pkg/input/`)
处理用户输入：
- 原始键盘输入捕获
- 非阻塞输入检测
- 终端模式管理

### 4. 系统信息模块 (`pkg/system/`)
获取系统状态信息：
- CPU和内存信息
- 磁盘使用情况
- 网络接口状态
- 系统控制功能

### 5. 菜单渲染模块 (`pkg/menu/`)
负责界面显示：
- 主菜单和配置菜单渲染
- 进度条显示
- 消息提示框

## 编译选项

### 标准编译
```bash
make build
```

### 静态编译（推荐用于生产环境）
```bash
make build-static
```

### 开发模式（包含格式化和检查）
```bash
make dev
```

## 安装到系统

### 安装到系统路径
```bash
make install
```

程序将被安装到 `/usr/local/bin/framebuffer-console`

### 创建服务文件（可选）
```bash
sudo tee /etc/systemd/system/framebuffer-console.service > /dev/null <<EOF
[Unit]
Description=Framebuffer Console System
After=multi-user.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/framebuffer-console
Restart=always
RestartSec=10
StandardOutput=null
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable framebuffer-console
```

## 故障排除

### 常见问题

1. **字体文件不存在**
   ```bash
   # 确保字体文件存在
   ls -la /root/go-framebuffer-master/fonts/SourceHanSansSC-Regular.otf
   # 如果不存在，请下载思源黑体字体文件
   ```

2. **Framebuffer设备不可用**
   ```bash
   # 检查framebuffer设备
   ls -la /dev/fb*
   # 检查内核模块
   lsmod | grep fb
   ```

3. **权限不足**
   ```bash
   # 程序需要root权限运行
   sudo ./framebuffer-console
   ```

4. **编译错误**
   ```bash
   # 确保安装了必要的开发工具
   sudo yum groupinstall "Development Tools"
   sudo yum install gcc
   ```

### 调试模式
```bash
# 查看详细错误信息
go run cmd/main/main.go
```

## 技术细节

### 兼容性
- 支持16/24/32位色深的Framebuffer
- 兼容CentOS 7.9和其他主流Linux发行版
- 自动检测最佳Framebuffer设备

### 性能优化
- 内存映射方式访问Framebuffer
- 优化的字体渲染算法
- 非阻塞键盘输入处理

### 安全特性
- 安全的系统调用封装
- 优雅的错误处理
- 信号处理和资源清理

## 开发指南

### 添加新功能
1. 在相应的pkg目录下添加新模块
2. 实现相应的接口
3. 在main.go中集成新功能
4. 更新配置和文档

### 代码规范
- 遵循Go语言标准编码规范
- 使用gofmt格式化代码
- 添加适当的错误处理
- 编写清晰的注释

## 许可证

本项目采用MIT许可证，详情请参见LICENSE文件。

## 贡献

欢迎提交Issue和Pull Request来改进这个项目。

## 作者

该项目由Claude Code助手开发完成，专为Linux系统管理而设计。
# Go Framebuffer Console - 系统状态监控应用

## 项目简介

Go Framebuffer Console 是一个基于 Linux Framebuffer 的系统状态监控应用程序，专为 CentOS 7.9 环境优化设计。该应用直接操作帧缓冲设备，无需 X11 环境即可在终端显示图形化的系统监控界面。

### 核心特性

- **直接帧缓冲渲染**：无需图形环境，直接操作 `/dev/fb0` 设备
- **中文字体支持**：完美支持 TTF 格式中文字体渲染
- **实时系统监控**：每5秒自动刷新系统状态信息
- **网络连通性测试**：内置高级网络诊断功能，支持多目标ping测试
- **二维码显示**：自动生成乾坤云设备ID的二维码
- **智能缓存渲染**：避免闪烁，提供流畅的用户体验
- **可控退出机制**：支持禁用Ctrl+C等控制键退出功能
- **完整的系统管理**：支持重启、关机等系统操作

## 系统要求

### 硬件要求
- **处理器**：x86_64 架构
- **内存**：最小 256MB RAM
- **显示**：支持 Framebuffer 的显示设备

### 软件要求
- **操作系统**：CentOS 7.9 或兼容的 Linux 发行版
- **内核**：支持 Framebuffer 设备 (`/dev/fb0`)
- **字体**：TTF 格式的中文字体文件
- **权限**：读取 `/dev/fb0`、`/proc/*`、`/sys/*` 的权限

### 依赖库
- `github.com/golang/freetype` - 字体渲染引擎
- `golang.org/x/image` - 图像处理库
- `rsc.io/qr` - 二维码生成库

## 功能详解

### 🖥️ 主界面显示

主界面采用清晰的信息布局，显示以下系统信息：

```
系统信息
================================ 
操作系统运行时间：X天 X小时 X分钟
处理器型号：Intel(R) Xeon(R) CPU E5-2696 v4 @2.20GHz *20 核
内存使用状态：444M/19995MB
系统安装磁盘大小：20G（共2个磁盘）
当前系统时间：2025-06-15 12:00:00
设备IP地址：192.168.1.100

设备ID：your-device-id

================================

[二维码区域]

===============================

如有问题请咨询技术客服：微信：your-service-wechat

按回车键进入配置菜单
```

### 📊 系统信息监控

#### 处理器信息
- **型号识别**：自动识别CPU型号和架构
- **核心统计**：显示物理核心数量
- **格式化显示**：`处理器型号 *核心数 核`

#### 内存监控
- **实时统计**：已用内存/总内存（MB单位）
- **精确计算**：基于 `/proc/meminfo` 的 MemAvailable 计算
- **格式示例**：`444M/19995MB`

#### 磁盘统计
- **物理磁盘识别**：只统计真实物理磁盘（SATA/SAS/NVMe）
- **智能过滤**：排除虚拟设备（loop、ram、dm-等）
- **容量汇总**：显示所有物理磁盘总容量和数量

#### 网络信息
- **设备IP获取**：通过默认路由确定主要网卡IP地址
- **接口检测**：自动识别活跃的网络接口
- **地址验证**：排除回环和链路本地地址

### 🌐 高级网络连通性测试

内置专业级网络诊断功能，支持多目标并发测试：

#### 测试目标
1. **字节跳动官网** (`bytedance.com`)
2. **百度首页** (`baidu.com`)
3. **哔哩哔哩** (`bilibili.com`)
4. **腾讯官网** (`tencent.com`)
5. **阿里DNS服务器** (`223.5.5.5`)

#### 测试特性
- **并发测试**：同时对5个目标进行连通性检测
- **详细统计**：每个目标发送4个ping包
- **实时进度**：显示测试进度 `X/5`
- **结果分析**：
  - 数据包统计（发送/接收/丢失率）
  - 平均延迟时间
  - 连接状态（正常/部分正常/异常）

#### 结果展示
```
=== 网络连通性测试结果 ===

• 字节跳动 (bytedance.com):
  状态: 正常
  数据包: 发送4 接收4 丢失0.0%
  平均延迟: 15.2 ms

• 百度 (baidu.com):
  状态: 部分正常
  数据包: 发送4 接收3 丢失25.0%
  详情: 25.0% 数据包丢失

----------------------------------------
✓ 网络连接状态: 良好
可访问 4/5 个测试目标
```

### 📱 二维码功能

#### 自动生成
- **数据源**：读取 `/usr/local/etc/device/id` 文件（可配置）
- **渲染引擎**：使用 `rsc.io/qr` 生成标准二维码
- **显示格式**：白色背景，黑色方块，左右边距

#### 技术规格
- **纠错级别**：M级别（15%纠错能力）
- **像素放大**：4倍放大确保清晰度
- **边距设计**：左右各2个像素单位边距
- **扫描兼容**：兼容所有主流二维码扫描器

### 📝 日志系统

#### 自动轮转特性
- **按日切割**：每天0点自动创建新日志文件
- **文件格式**：`console-YYYY-MM-DD.log`
- **自动清理**：保留最近3天的日志，自动删除旧文件
- **实时记录**：所有操作和错误都记录到日志

#### 日志内容包含
- 程序启动和退出事件
- 用户操作记录  
- 控制键拦截详情
- 网络测试结果
- 系统错误信息
- 性能监控数据

### ⚙️ 配置菜单

按回车键进入配置菜单，提供以下功能：

```
============================
配置菜单
============================
1. 查看网卡信息
2. 重启系统服务  
3. 检测设备网络
4. 重启设备
5. 关机
============================
请输入选项(1-5)，按q返回首页
```

#### 1. 查看网卡信息
- **物理接口识别**：只显示真实的物理网卡
- **状态检测**：Up/Down/Running状态
- **地址信息**：IPv4和IPv6地址列表
- **硬件信息**：MAC地址显示

#### 2. 重启系统服务
- **服务管理**：基于 systemctl 的服务控制
- **权限检查**：要求root权限
- **安全验证**：防止命令注入攻击

#### 3. 检测设备网络
执行高级网络连通性测试（详见网络测试功能）

#### 4. 重启设备
- **确认机制**：需要按 'y' 确认
- **权限检查**：要求root权限
- **优雅重启**：使用 `reboot` 命令

#### 5. 关机
- **确认机制**：需要按 'y' 确认  
- **权限检查**：要求root权限
- **安全关机**：使用 `shutdown -h now` 命令

### 🔒 退出控制机制

#### 命令行参数
- **`-d`**：禁用所有退出功能
- **`-h`**：显示帮助信息

#### 默认模式（无参数）
支持以下退出方式：
- **Ctrl+C** (SIGINT) - 中断退出
- **Ctrl+Z** (SIGTSTP) - 挂起退出
- **Ctrl+\** (SIGQUIT) - 退出信号
- **Ctrl+D** (EOF) - 文件结束
- **配置菜单退出**

#### 禁用模式（`-d` 参数）
- **全局拦截**：在任何界面都无法通过控制键退出
- **信号屏蔽**：拦截所有退出相关的系统信号
- **日志记录**：每次拦截尝试都记录到日志
- **唯一退出**：只能通过配置菜单正常退出

## 编译说明

### 环境准备

#### 1. 安装 Go 环境
```bash
# CentOS 7.9
sudo yum install -y epel-release
sudo yum install -y golang

# 或下载最新版本
wget https://golang.org/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### 2. 验证环境
```bash
go version
# 输出：go version go1.21.0 linux/amd64
```

### 获取源码

```bash
# 克隆项目（如果使用Git）
git clone <repository-url>
cd go-framebuffer-console

# 或解压源码包
tar -xzf go-framebuffer-console.tar.gz
cd go-framebuffer-console
```

### 依赖管理

```bash
# 下载依赖
go mod tidy

# 验证依赖
go mod verify
```

### 静态编译

静态编译生成独立可执行文件，不依赖系统动态库：

```bash
# 基本静态编译
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -a -ldflags '-linkmode external -extldflags "-static"' \
    -o framebuffer-console-static ./cmd/main

# 优化版本（减小文件大小）
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -a -ldflags '-linkmode external -extldflags "-static" -s -w' \
    -tags 'netgo osusergo static_build' \
    -o framebuffer-console-static ./cmd/main

# 验证静态链接
ldd framebuffer-console-static
# 输出：not a dynamic executable
```

#### 静态编译优势
- **独立部署**：无需安装Go运行时和系统库
- **版本兼容**：避免不同系统的库版本冲突
- **便于分发**：单文件分发，简化部署流程

### 动态编译

动态编译生成较小的可执行文件，但依赖系统库：

```bash
# 标准动态编译
go build -o framebuffer-console ./cmd/main

# 优化版本
go build -ldflags "-s -w" -o framebuffer-console ./cmd/main

# 验证动态链接
ldd framebuffer-console
# 输出：依赖的动态库列表
```

#### 动态编译优势
- **文件较小**：可执行文件体积更小
- **编译速度快**：编译时间较短
- **库共享**：可以利用系统库的更新

### 交叉编译

#### 为不同架构编译

```bash
# ARM64 静态编译
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
CC=aarch64-linux-gnu-gcc go build \
    -a -ldflags '-linkmode external -extldflags "-static"' \
    -o framebuffer-console-arm64 ./cmd/main

# ARM 静态编译  
CGO_ENABLED=1 GOOS=linux GOARCH=arm \
CC=arm-linux-gnueabi-gcc go build \
    -a -ldflags '-linkmode external -extldflags "-static"' \
    -o framebuffer-console-arm ./cmd/main
```

#### 交叉编译环境准备

```bash
# CentOS 7.9 安装交叉编译工具链
sudo yum install -y gcc-aarch64-linux-gnu gcc-arm-linux-gnu

# Ubuntu/Debian
sudo apt-get install -y gcc-aarch64-linux-gnu gcc-arm-linux-gnueabi
```

### 编译脚本

创建 `build.sh` 自动化编译脚本：

```bash
#!/bin/bash

# 编译配置
APP_NAME="framebuffer-console"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION=$(go version | awk '{print $3}')

# 编译标志
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GoVersion=${GO_VERSION}"

echo "开始编译 ${APP_NAME} v${VERSION}"

# 静态编译
echo "静态编译..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -a -ldflags "-linkmode external -extldflags \"-static\" -s -w ${LDFLAGS}" \
    -tags 'netgo osusergo static_build' \
    -o "${APP_NAME}-static" ./cmd/main

# 动态编译
echo "动态编译..."
go build -ldflags "-s -w ${LDFLAGS}" -o "${APP_NAME}" ./cmd/main

# 验证编译结果
echo "编译完成！"
echo "静态版本："
ls -lh "${APP_NAME}-static"
echo "动态版本："
ls -lh "${APP_NAME}"

# 创建发布包
echo "创建发布包..."
tar -czf "${APP_NAME}-${VERSION}-linux-amd64.tar.gz" \
    "${APP_NAME}" "${APP_NAME}-static" README.md

echo "发布包：${APP_NAME}-${VERSION}-linux-amd64.tar.gz"
```

## 部署指南

### 准备工作

#### 1. 字体文件
```bash
# 在项目目录下创建字体目录
mkdir -p fonts

# 复制TTF字体文件到项目字体目录（推荐使用中文字体）
cp your-chinese-font.ttf fonts/SourceHanSansSC-Regular.ttf

# 验证字体文件
ls -la fonts/
```

**注意**：程序会按以下优先级查找字体：
1. `./fonts/SourceHanSansSC-Regular.ttf` (TTF格式，推荐)
2. `./fonts/SourceHanSansSC-Regular.otf` (OTF格式，备用)

字体文件必须放在项目目录下的 `fonts/` 文件夹中，不是系统字体目录。

#### 2. 设备ID配置
```bash
# 创建配置目录
sudo mkdir -p /usr/local/etc/device

# 设置设备ID
echo "your-device-id" | sudo tee /usr/local/etc/device/id
```

#### 3. 权限设置
```bash
# 添加用户到video组（访问framebuffer）
sudo usermod -a -G video $USER

# 或设置文件权限
sudo chmod 666 /dev/fb0
```

### 安装部署

#### 方式一：直接运行
```bash
# 复制可执行文件
sudo cp framebuffer-console /usr/local/bin/
sudo chmod +x /usr/local/bin/framebuffer-console

# 运行
framebuffer-console          # 普通模式
framebuffer-console -d       # 禁用退出模式
framebuffer-console -h       # 显示帮助
```

#### 方式二：系统服务
创建 systemd 服务文件：

```bash
sudo tee /etc/systemd/system/framebuffer-console.service << EOF
[Unit]
Description=Go Framebuffer Console System Monitor
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/framebuffer-console -d
Restart=always
RestartSec=5
WorkingDirectory=/usr/local/bin

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable framebuffer-console
sudo systemctl start framebuffer-console

# 查看状态
sudo systemctl status framebuffer-console
```

### 配置文件

应用程序支持配置文件 `/etc/framebuffer-console.conf`：

```ini
# 字体配置
font_path=./fonts/SourceHanSansSC-Regular.ttf
font_size=14
dpi=96

# 显示配置
framebuffer_device=/dev/fb0
refresh_interval=5

# 网络测试配置
network_test_timeout=20
ping_count=4

# 设备ID配置
device_id_file=/usr/local/etc/device/id
```

## 使用指南

### 基本操作

#### 启动程序
```bash
# 标准模式（支持Ctrl+C退出）
./framebuffer-console

# 生产模式（禁用退出功能）
./framebuffer-console -d

# 查看帮助
./framebuffer-console -h
```

#### 界面导航
- **主界面**：显示系统状态，每5秒自动刷新
- **回车键**：进入配置菜单
- **配置菜单**：按1-5选择功能，按q返回
- **任意键**：在信息页面按任意键返回

#### 退出方式
- **标准模式**：Ctrl+C、Ctrl+Z、Ctrl+\、Ctrl+D
- **禁用模式**：只能通过配置菜单退出

### 高级功能

#### 网络诊断
1. 进入配置菜单（回车键）
2. 选择"3. 检测设备网络"
3. 观察测试进度和结果
4. 按任意键返回菜单

#### 系统管理
1. 重启设备：配置菜单 → 4 → 按y确认
2. 关机：配置菜单 → 5 → 按y确认
3. 查看网卡：配置菜单 → 1

#### 日志查看
```bash
# 查看今天的实时日志
tail -f console-$(date +%Y-%m-%d).log

# 查看今天的完整日志
cat console-$(date +%Y-%m-%d).log

# 查看指定日期的日志
cat console-2025-06-15.log

# 查看最近3天的所有日志
ls console-*.log | sort
```

## 故障排除

### 常见问题

#### 1. 无法启动 - 权限错误
```
错误：无法打开/dev/fb0: permission denied
解决：sudo usermod -a -G video $USER
     或 sudo chmod 666 /dev/fb0
```

#### 2. 字体显示异常
```
错误：bad ttf version 或 无法读取字体文件
解决：1. 确保字体文件放在 ./fonts/ 目录下
     2. 文件名必须为 SourceHanSansSC-Regular.ttf
     3. 确保使用TTF格式，不要使用OTF格式
     4. 检查字体文件是否损坏
```

#### 3. 网络测试失败
```
错误：所有目标都无法访问
检查：网络连接、DNS配置、防火墙设置
调试：ping命令手动测试各个目标
```

#### 4. 二维码无法生成
```
错误：读取设备ID失败
检查：/usr/local/etc/device/id文件是否存在
      文件内容是否正确
      文件权限是否可读
```

#### 5. Framebuffer不可用
```
错误：/dev/fb0: no such device
检查：内核是否支持framebuffer
      显卡驱动是否正确安装
      是否在图形环境中运行
```

### 调试方法

#### 启用详细日志
```bash
# 查看系统信息
cat /proc/cpuinfo
cat /proc/meminfo
cat /proc/partitions

# 检查framebuffer
ls -la /dev/fb*
cat /proc/fb

# 测试网络
ping -c 4 baidu.com
ip route show default
```

#### 编译调试版本
```bash
# 调试编译
go build -gcflags "-N -l" -o framebuffer-console-debug ./cmd/main

# 使用delve调试
dlv exec ./framebuffer-console-debug
```

### 性能优化

#### 1. 内存优化
- 限制缓存大小
- 定期清理资源
- 优化图像处理

#### 2. CPU优化
- 减少不必要的重绘
- 优化字体渲染
- 使用缓存机制

#### 3. 磁盘优化
- 日志轮转
- 临时文件清理
- 配置文件优化

## 自定义配置

### 修改显示信息

#### 1. 客服微信号
在 `pkg/menu/renderer.go` 中修改：
```go
// 找到第461行左右的客服信息
customerServiceContent := []string{
    "如有问题请咨询技术客服：微信：your-service-wechat", // 修改这里
    "",
    "按回车键进入配置菜单",
}
```

#### 2. 设备ID文件路径
在 `pkg/system/info.go` 中修改：
```go
// 找到getQianKunCloudID函数，修改文件路径
func getQianKunCloudID() (string, error) {
    data, err := os.ReadFile("/usr/local/etc/device/id") // 修改这里
    if err != nil {
        return "", fmt.Errorf("读取设备ID失败: %v", err)
    }
    // ...
}
```

#### 3. 二维码说明文字
在 `pkg/menu/renderer.go` 中修改：
```go
// 找到renderQRCode函数中的说明文字
headerText := "此处为二维码展示，二维码的值为设备ID" // 修改这里
```

#### 4. 字体文件路径
在 `internal/config/config.go` 中修改：
```go
const (
    DefaultFontPath = "./fonts/YourCustomFont.ttf" // 修改字体文件名
    BackupFontPath  = "./fonts/YourCustomFont.otf" // 修改备用字体
    // ...
)
```

### 品牌定制

可以修改以下内容来适配你的品牌：
- 系统标题："系统信息" → "你的品牌名称"
- 客服信息：微信号和联系方式
- 设备ID字段名称
- 配置文件路径
- 日志文件名称

## 开发指南

### 项目结构

```
go-framebuffer-console/
├── cmd/main/                 # 主程序入口
│   └── main.go
├── internal/config/          # 内部配置管理
│   └── config.go
├── pkg/                      # 公共包
│   ├── font/                 # 字体渲染
│   │   └── renderer.go
│   ├── framebuffer/          # 帧缓冲操作
│   │   └── framebuffer.go
│   ├── input/                # 输入处理
│   │   └── keyboard.go
│   ├── menu/                 # 菜单渲染
│   │   └── renderer.go
│   └── system/               # 系统信息
│       └── info.go
├── fonts/                    # 字体文件目录（必需）
│   ├── SourceHanSansSC-Regular.ttf  # 主字体文件
│   └── SourceHanSansSC-Regular.otf  # 备用字体文件
├── go.mod                    # Go模块文件
├── go.sum                    # 依赖校验
├── README.md                 # 项目文档
└── console-YYYY-MM-DD.log   # 按日期命名的日志文件
```

### 扩展开发

#### 添加新的系统监控项
```go
// 在 pkg/system/info.go 中添加
type SystemInfo struct {
    // 现有字段...
    YourNewField string // 新监控项
}

func getYourNewInfo() (string, error) {
    // 实现获取逻辑
    return "your-value", nil
}
```

#### 添加新的菜单功能
```go
// 在 cmd/main/main.go 中添加
func (app *Application) yourNewFunction() error {
    // 实现新功能
    message := "您的新功能\n\n按任意键返回"
    if err := app.menuRenderer.RenderMessage(message); err != nil {
        return err
    }
    
    // 处理用户输入
    for {
        key, err := app.keyboard.ReadKey()
        if err != nil {
            return err
        }
        
        if app.handleControlKey(key, "新功能页面") {
            return nil
        }
        
        return nil // 其他键返回
    }
}
```

#### 自定义渲染样式
```go
// 在 pkg/menu/renderer.go 中修改
func (mr *MenuRenderer) customRender() error {
    // 自定义渲染逻辑
    mr.renderer.SetSize(16) // 调整字体大小
    
    // 自定义颜色
    customColor := color.RGBA{255, 128, 0, 255} // 橙色
    
    // 渲染内容
    textImg, err := mr.renderer.RenderText("自定义内容", customColor)
    if err != nil {
        return err
    }
    
    mr.fb.DrawImage(textImg, x, y)
    return nil
}
```

### 贡献指南

#### 代码规范
- 使用 `gofmt` 格式化代码
- 遵循 Go 命名约定
- 添加适当的注释
- 错误处理完整

#### 提交流程
1. Fork 项目
2. 创建功能分支
3. 开发和测试
4. 提交 Pull Request

#### 测试要求
- 单元测试覆盖
- 集成测试验证
- 性能测试检查
- 兼容性测试

## 许可证

本项目采用 [MIT License](LICENSE) 开源许可证。

## 技术支持

- **问题反馈**：提交 GitHub Issues
- **功能建议**：创建 Feature Request
- **技术讨论**：参与 Discussions
- **紧急支持**：联系维护团队

## 更新日志

### v1.0.0 (2025-06-01)
- ✅ 初始版本发布
- ✅ 基础系统监控功能
- ✅ Framebuffer 渲染支持
- ✅ 中文字体显示

### v1.1.0 (2025-06-10)
- ✅ 新增高级网络测试
- ✅ 二维码显示功能
- ✅ 智能缓存渲染
- ✅ 性能优化

### v1.2.0 (2025-06-15)
- ✅ 退出控制机制
- ✅ 全局信号拦截
- ✅ 自动日志轮转
- ✅ 系统服务支持
- ✅ 品牌信息脱敏

---

*最后更新：2025-06-15*
*版本：v1.2.0*
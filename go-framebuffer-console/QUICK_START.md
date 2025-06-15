# Go Framebuffer Console 快速开始指南

## ✅ 已完成的工作

1. **完整的中文注释**：所有代码文件都已添加详细的中文注释
2. **字体文件就绪**：已复制字体文件到项目 `fonts/` 目录
3. **配置优化**：更新了字体路径配置，使用相对路径
4. **构建脚本完善**：提供了自动化编译和部署脚本

## 🚀 快速编译步骤

### 1. 进入项目目录
```bash
cd /mnt/d/console/go-framebuffer-console
```

### 2. 一键编译和打包（推荐）
```bash
./build_and_deploy.sh
```

### 3. 手动编译（可选）
```bash
# 下载依赖
go mod tidy

# 静态编译
CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o framebuffer-console ./cmd/main
```

## 📦 部署到CentOS设备

### 方法一：使用自动生成的部署包
```bash
# 编译完成后会生成 framebuffer-console-deploy.tar.gz
# 传输到CentOS设备：
scp framebuffer-console-deploy.tar.gz root@你的CentOS_IP:/tmp/

# 在CentOS设备上：
cd /tmp
tar -xzf framebuffer-console-deploy.tar.gz
cd deploy-package
sudo ./install.sh
```

### 方法二：手动复制文件
需要复制到CentOS设备的文件：
1. `framebuffer-console` - 主程序
2. `fonts/SourceHanSansSC-Regular.otf` - 字体文件

## 🎯 在CentOS上运行

### 确保字体文件在正确位置
程序会在以下位置查找字体文件：
- `./fonts/SourceHanSansSC-Regular.otf`（当前目录下的fonts文件夹）
- `/usr/local/bin/fonts/SourceHanSansSC-Regular.otf`（系统安装位置）

### 运行程序
```bash
# 方法一：在程序目录下运行（推荐）
cd /path/to/framebuffer-console
sudo ./framebuffer-console

# 方法二：如果已安装到系统路径
cd /usr/local/bin
sudo framebuffer-console
```

## ⚠️ 重要提醒

1. **字体文件位置**：
   - 现在使用相对路径 `./fonts/SourceHanSansSC-Regular.otf`
   - 确保在程序运行目录下有 `fonts/` 文件夹
   - 字体文件大小约15.7MB，是必需的

2. **运行权限**：
   - 必须使用 `sudo` 运行程序
   - 程序需要访问 `/dev/fb0` 等设备

3. **系统要求**：
   - 支持framebuffer的Linux系统
   - CentOS 7.9或其他主流发行版
   - 确保 `/dev/fb*` 设备存在

## 🔧 故障排除

### 字体文件找不到
```bash
# 检查字体文件是否存在
ls -la ./fonts/SourceHanSansSC-Regular.otf
# 或
ls -la /usr/local/bin/fonts/SourceHanSansSC-Regular.otf
```

### framebuffer设备不可用
```bash
# 检查设备
ls -la /dev/fb*

# 加载模块（如果需要）
modprobe fbcon
modprobe vesafb
```

### 权限问题
```bash
# 确保以root权限运行
sudo ./framebuffer-console

# 检查设备权限
ls -la /dev/fb0
```

## 📋 文件清单

编译完成后，您应该有以下文件：
- `framebuffer-console` - 主程序（约10-20MB，静态编译）
- `fonts/SourceHanSansSC-Regular.otf` - 字体文件（15.7MB）
- `framebuffer-console-deploy.tar.gz` - 完整部署包

现在所有代码都有详细的中文注释，字体文件也已正确配置。按照上述步骤即可编译并部署到CentOS设备上运行！
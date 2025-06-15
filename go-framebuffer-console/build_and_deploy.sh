#!/bin/bash

# Go Framebuffer Console 自动编译和部署脚本
# 用于快速编译程序并准备部署包

set -e  # 遇到错误时立即退出

echo "==============================================="
echo "Go Framebuffer Console 编译和部署脚本"
echo "==============================================="

# 1. 检查Go环境
echo "检查Go语言环境..."
if ! command -v go &> /dev/null; then
    echo "错误：未找到Go语言环境，请先安装Go 1.21或更高版本"
    exit 1
fi

go_version=$(go version | awk '{print $3}' | sed 's/go//')
echo "Go版本: $go_version"

# 2. 下载依赖
echo "下载依赖包..."
go mod tidy
go mod download

# 3. 编译程序（静态编译）
echo "编译程序（静态编译版本）..."
CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o framebuffer-console ./cmd/main

# 4. 验证编译结果
if [ ! -f "framebuffer-console" ]; then
    echo "错误：编译失败，未生成可执行文件"
    exit 1
fi

echo "编译成功！"
ls -la framebuffer-console
file framebuffer-console

# 5. 创建部署包
echo "创建部署包..."
rm -rf deploy-package
mkdir -p deploy-package
mkdir -p deploy-package/fonts

# 复制主程序
cp framebuffer-console deploy-package/

# 检查并复制字体文件
FONT_SOURCE="./fonts/SourceHanSansSC-Regular.otf"
if [ -f "$FONT_SOURCE" ]; then
    cp "$FONT_SOURCE" deploy-package/fonts/
    echo "字体文件已复制"
else
    echo "警告：未找到字体文件 $FONT_SOURCE"
    echo "请确保字体文件存在，否则程序无法正常运行"
fi

# 创建安装脚本
cat > deploy-package/install.sh << 'EOF'
#!/bin/bash
echo "开始安装Go Framebuffer Console..."

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then
    echo "请使用root权限运行此脚本"
    exit 1
fi

# 创建字体目录
mkdir -p /usr/local/bin/fonts

# 复制字体文件到程序目录
if [ -f "fonts/SourceHanSansSC-Regular.otf" ]; then
    cp fonts/SourceHanSansSC-Regular.otf /usr/local/bin/fonts/
    echo "字体文件已安装到 /usr/local/bin/fonts/"
else
    echo "警告：字体文件不存在，程序可能无法正常显示中文"
fi

# 复制主程序到系统路径
cp framebuffer-console /usr/local/bin/
chmod +x /usr/local/bin/framebuffer-console

echo "安装完成！"
echo ""
echo "使用方法："
echo "1. 直接运行：sudo framebuffer-console"
echo "2. 停止程序：按Ctrl+C"
echo ""
echo "注意事项："
echo "- 程序必须以root权限运行"
echo "- 确保系统支持framebuffer设备（/dev/fb*）"
echo "- 字体文件路径：/usr/local/bin/fonts/SourceHanSansSC-Regular.otf"
echo "- 程序会在当前工作目录下查找 ./fonts/ 目录中的字体文件"
EOF

chmod +x deploy-package/install.sh

# 创建README文件
cat > deploy-package/README.txt << 'EOF'
Go Framebuffer Console 部署包
=============================

文件说明：
- framebuffer-console: 主程序（静态编译版本）
- fonts/SourceHanSansSC-Regular.otf: 思源黑体字体文件
- install.sh: 自动安装脚本

安装步骤：
1. 将此目录复制到目标CentOS设备
2. 进入目录：cd deploy-package
3. 运行安装脚本：sudo ./install.sh
4. 运行程序：sudo framebuffer-console

系统要求：
- CentOS 7.9 或其他Linux发行版
- 支持framebuffer的系统
- root权限

技术支持：
如遇问题，请检查：
- 字体文件是否正确安装
- framebuffer设备是否可用（ls /dev/fb*）
- 程序是否有足够权限
EOF

# 6. 打包
echo "打包部署文件..."
tar -czf framebuffer-console-deploy.tar.gz deploy-package/

echo ""
echo "==============================================="
echo "编译和打包完成！"
echo "==============================================="
echo "生成的文件："
echo "- framebuffer-console: 可执行程序"
echo "- deploy-package/: 部署目录"
echo "- framebuffer-console-deploy.tar.gz: 部署包"
echo ""
echo "部署到CentOS设备的步骤："
echo "1. 传输文件到目标设备："
echo "   scp framebuffer-console-deploy.tar.gz root@目标IP:/tmp/"
echo ""
echo "2. 在目标设备上执行："
echo "   cd /tmp"
echo "   tar -xzf framebuffer-console-deploy.tar.gz"
echo "   cd deploy-package"
echo "   sudo ./install.sh"
echo ""
echo "3. 运行程序："
echo "   sudo framebuffer-console"
echo "==============================================="
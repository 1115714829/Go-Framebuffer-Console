# Go Framebuffer Console 编译和部署指南

## 编译环境要求

### 在WSL环境中编译（推荐）

1. **安装Go语言**
```bash
# 下载Go 1.21或更高版本
wget https://golang.org/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# 设置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

2. **安装编译依赖**
```bash
# 安装C编译器和开发工具
sudo apt update
sudo apt install gcc libc6-dev pkg-config
```

## 编译步骤

### 1. 下载依赖
```bash
cd /mnt/d/console/go-framebuffer-console
go mod tidy
go mod download
```

### 2. 编译程序
```bash
# 标准编译（推荐用于开发测试）
make build

# 静态编译（推荐用于生产部署）
make build-static
```

编译完成后会生成 `framebuffer-console` 可执行文件。

### 3. 验证编译结果
```bash
# 检查可执行文件
ls -la framebuffer-console
file framebuffer-console

# 查看依赖（静态编译版本应该没有动态依赖）
ldd framebuffer-console
```

## 部署到CentOS 7.9设备

### 需要复制的文件

#### 1. 主程序文件
```bash
framebuffer-console          # 主程序（必需）
```

#### 2. 字体文件
```bash
# 需要将字体文件复制到目标设备的指定路径
# 源文件（在您的环境中）：
/root/go-framebuffer-master/fonts/SourceHanSansSC-Regular.otf

# 目标路径（在CentOS设备中）：
/root/go-framebuffer-master/fonts/SourceHanSansSC-Regular.otf
```

### 完整部署步骤

#### 1. 准备部署包
在WSL环境中创建部署包：
```bash
# 创建部署目录
mkdir -p deploy-package
mkdir -p deploy-package/fonts

# 复制主程序
cp framebuffer-console deploy-package/

# 复制字体文件
cp /root/go-framebuffer-master/fonts/SourceHanSansSC-Regular.otf deploy-package/fonts/

# 创建部署脚本
cat > deploy-package/install.sh << 'EOF'
#!/bin/bash
echo "开始安装Go Framebuffer Console..."

# 创建字体目录
mkdir -p /root/go-framebuffer-master/fonts

# 复制字体文件
cp fonts/SourceHanSansSC-Regular.otf /root/go-framebuffer-master/fonts/

# 复制主程序到系统路径
cp framebuffer-console /usr/local/bin/
chmod +x /usr/local/bin/framebuffer-console

echo "安装完成！"
echo "使用命令运行程序：sudo framebuffer-console"
EOF

chmod +x deploy-package/install.sh

# 打包
tar -czf framebuffer-console-deploy.tar.gz deploy-package/
```

#### 2. 传输到CentOS设备
```bash
# 使用scp传输（替换IP地址和用户名）
scp framebuffer-console-deploy.tar.gz root@192.168.1.100:/tmp/

# 或者使用其他方式传输，如U盘、网络共享等
```

#### 3. 在CentOS设备上安装
```bash
# 登录到CentOS设备
ssh root@192.168.1.100

# 解压部署包
cd /tmp
tar -xzf framebuffer-console-deploy.tar.gz
cd deploy-package

# 运行安装脚本
./install.sh
```

#### 4. 验证安装
```bash
# 检查主程序
which framebuffer-console
ls -la /usr/local/bin/framebuffer-console

# 检查字体文件
ls -la /root/go-framebuffer-master/fonts/SourceHanSansSC-Regular.otf

# 检查framebuffer设备
ls -la /dev/fb*
```

### 运行程序

#### 1. 直接运行
```bash
# 必须使用root权限运行
sudo framebuffer-console
```

#### 2. 创建systemd服务（可选）
```bash
# 创建服务文件
cat > /etc/systemd/system/framebuffer-console.service << 'EOF'
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

# 启用和启动服务
systemctl daemon-reload
systemctl enable framebuffer-console
systemctl start framebuffer-console

# 查看服务状态
systemctl status framebuffer-console
```

## 故障排除

### 常见问题

#### 1. 字体文件路径错误
```bash
# 确保字体文件在正确位置
ls -la /root/go-framebuffer-master/fonts/SourceHanSansSC-Regular.otf
```

#### 2. framebuffer设备不可用
```bash
# 检查framebuffer设备
ls -la /dev/fb*

# 如果没有fb设备，可能需要配置显卡驱动或内核模块
modprobe fbcon
modprobe vesafb
```

#### 3. 权限问题
```bash
# 程序必须以root权限运行
sudo framebuffer-console

# 检查设备权限
ls -la /dev/fb0
```

#### 4. 程序无法启动
```bash
# 检查依赖（静态编译版本通常没有这个问题）
ldd /usr/local/bin/framebuffer-console

# 查看错误日志
journalctl -u framebuffer-console -f
```

## 重要提醒

1. **字体文件是必需的**：程序运行需要思源黑体字体文件，路径必须正确
2. **需要root权限**：程序需要访问framebuffer设备，必须以root身份运行
3. **检查设备兼容性**：确保目标设备支持framebuffer
4. **备份重要数据**：部署前请备份重要数据
5. **测试环境**：建议先在测试环境中验证程序运行正常

## 技术支持

如果遇到问题，请检查：
- 字体文件路径是否正确
- framebuffer设备是否可用
- 程序是否有足够权限
- 系统日志中的错误信息

编译完成的程序是静态链接的，理论上可以在任何Linux x86_64系统上运行，包括CentOS 7.9。
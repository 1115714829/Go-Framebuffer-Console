# 字体使用指南

## 🚨 重要提示

当前OTF字体不被完全支持，建议使用TTF格式的字体文件。

## 📁 字体文件路径

程序会按以下优先级查找字体文件：

1. `./fonts/SourceHanSansSC-Regular.ttf` (推荐)
2. `./fonts/SourceHanSansSC-Regular.otf` (备用，可能有问题)

## 🔧 解决方案

### 方案1：下载TTF版本字体
```bash
# 创建字体目录
mkdir -p fonts

# 下载思源黑体TTF版本
wget https://github.com/adobe-fonts/source-han-sans/releases/download/2.004R/SourceHanSansSC.zip
unzip SourceHanSansSC.zip
cp SourceHanSansSC/Regular/SourceHanSansSC-Regular.ttf fonts/
```

### 方案2：转换现有OTF字体为TTF

#### 使用FontForge转换：
```bash
# 安装FontForge
sudo yum install fontforge

# 转换OTF到TTF
fontforge -lang=ff -c 'Open("fonts/SourceHanSansSC-Regular.otf"); Generate("fonts/SourceHanSansSC-Regular.ttf")'
```

#### 使用在线工具：
- 访问 https://convertio.co/otf-ttf/
- 上传你的OTF文件
- 下载转换后的TTF文件

### 方案3：使用其他TTF字体

如果没有思源黑体，可以使用系统自带的TTF字体：

```bash
# 查找系统TTF字体
find /usr/share/fonts -name "*.ttf" | head -10

# 复制系统字体到项目目录
cp /usr/share/fonts/truetype/dejavu/DejaVuSans.ttf fonts/SourceHanSansSC-Regular.ttf
```

## 🛠 支持的字体格式

✅ **支持的格式：**
- TTF (TrueType Font) - **推荐**
- TTC (TrueType Collection)

❌ **不支持的格式：**
- OTF (OpenType Font) - 部分支持，可能出现 "bad ttf version" 错误
- WOFF (Web Open Font Format)
- WOFF2 (Web Open Font Format 2.0)

## 🔍 问题排查

如果仍然遇到字体问题：

1. **检查文件是否存在：**
   ```bash
   ls -la fonts/
   ```

2. **检查文件格式：**
   ```bash
   file fonts/SourceHanSansSC-Regular.ttf
   ```

3. **检查文件权限：**
   ```bash
   chmod 644 fonts/SourceHanSansSC-Regular.ttf
   ```

4. **使用绝对路径：**
   修改配置使用绝对路径，如 `/root/fonts/SourceHanSansSC-Regular.ttf`

## 📝 推荐字体下载

- **思源黑体 TTF版本**: https://github.com/adobe-fonts/source-han-sans/releases
- **文泉驿微米黑**: https://sourceforge.net/projects/wqy/files/wqy-microhei/
- **Noto Sans CJK**: https://github.com/googlefonts/noto-cjk/releases
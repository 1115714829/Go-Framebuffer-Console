// font包提供了字体渲染功能，支持TrueType/OpenType字体
// 专门针对中文字体进行了优化，支持复杂的汉字渲染
package font

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

// Renderer 字体渲染器结构体
// 封装了字体文件、渲染上下文和相关参数
type Renderer struct {
	font    *truetype.Font    // TrueType字体对象
	context *freetype.Context // FreeType渲染上下文
	dpi     float64           // 每英寸点数（分辨率）
	size    float64           // 字体大小（点）
}

// NewRenderer 创建新的字体渲染器
// 参数fontPath: 字体文件路径（支持.ttf/.otf格式）
// 参数size: 字体大小（点）
// 参数dpi: 分辨率（每英寸点数）
// 返回初始化完成的渲染器或错误信息
func NewRenderer(fontPath string, size float64, dpi float64) (*Renderer, error) {
	// 验证参数
	if fontPath == "" {
		return nil, fmt.Errorf("字体文件路径不能为空")
	}
	if size <= 0 || size > 200 {
		return nil, fmt.Errorf("字体大小无效: %f", size)
	}
	if dpi <= 0 || dpi > 600 {
		return nil, fmt.Errorf("DPI值无效: %f", dpi)
	}

	// 读取字体文件内容 (使用新的API)
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取字体文件 %s: %v", fontPath, err)
	}

	// 验证文件大小
	if len(fontBytes) == 0 {
		return nil, fmt.Errorf("字体文件为空: %s", fontPath)
	}
	if len(fontBytes) > 50*1024*1024 { // 限制50MB
		return nil, fmt.Errorf("字体文件过大: %d bytes", len(fontBytes))
	}

	// 检查字体文件格式
	if err := validateFontFormat(fontBytes); err != nil {
		return nil, fmt.Errorf("不支持的字体格式 %s: %v", fontPath, err)
	}

	// 解析字体文件
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		// 如果是OTF文件，尝试转换或给出更友好的错误信息
		if isOTFFont(fontBytes) {
			return nil, fmt.Errorf("OTF字体格式支持有限，建议使用TTF格式的字体文件。当前文件: %s", fontPath)
		}
		return nil, fmt.Errorf("无法解析字体文件 %s: %v", fontPath, err)
	}

	// 创建FreeType渲染上下文
	c := freetype.NewContext()
	c.SetFont(f)        // 设置字体
	c.SetFontSize(size) // 设置字体大小
	c.SetDPI(dpi)       // 设置分辨率

	return &Renderer{
		font:    f,
		context: c,
		dpi:     dpi,
		size:    size,
	}, nil
}

// SetSize 设置字体大小
// 参数size: 新的字体大小（点）
// 动态调整渲染器的字体大小，用于不同场景的文字显示
func (r *Renderer) SetSize(size float64) {
	r.size = size               // 更新内部字体大小记录
	r.context.SetFontSize(size) // 更新FreeType上下文的字体大小
}

// GetTextBounds 使用现代的 `golang.org/x/image/font` 库来精确计算文本的边界尺寸
// 参数text: 要测量的文本字符串
// 返回文本的宽度和高度（像素）
// 这个方法能正确处理kerning等高级字体特性，确保尺寸的精确性
func (r *Renderer) GetTextBounds(text string) (int, int) {
	face := truetype.NewFace(r.font, &truetype.Options{
		Size:    r.size,
		DPI:     r.dpi,
		Hinting: font.HintingFull, // 使用完整的字体微调，以获得最精确的尺寸
	})

	bounds, advance := font.BoundString(face, text)

	// advance 是画笔前进的距离，这是最准确的行宽度
	width := int(advance >> 6) // 从 26.6 fixed-point 格式转换为 int pixels

	// bounds 描述的是实际像素占用的矩形区域，我们可以用它来获取高度
	height := int((bounds.Max.Y - bounds.Min.Y) >> 6)

	// 为宽度和高度增加一点额外的边距，确保文本不被截断
	return width + 2, height + 2
}

// RenderText 渲染单行文本为图像
// 参数text: 要渲染的文本字符串
// 参数textColor: 文本颜色
// 返回包含渲染文本的图像或错误信息
// 支持中文字符的完美渲染，包括复杂汉字
func (r *Renderer) RenderText(text string, textColor color.Color) (image.Image, error) {
	// 计算文本尺寸
	width, height := r.GetTextBounds(text)
	// 如果计算失败，使用默认尺寸
	if width == 0 || height == 0 {
		width = 100
		height = int(r.size)
	}

	// 创建RGBA图像，额外添加10像素高度以防止裁剪
	img := image.NewRGBA(image.Rect(0, 0, width, height+10))
	// 用透明色填充背景
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	// 设置FreeType渲染参数
	r.context.SetClip(img.Bounds())             // 设置裁剪区域
	r.context.SetDst(img)                       // 设置目标图像
	r.context.SetSrc(&image.Uniform{textColor}) // 设置文本颜色

	// 计算文本基线位置
	pt := freetype.Pt(0, int(r.context.PointToFixed(r.size)>>6))
	// 绘制文本字符串
	_, err := r.context.DrawString(text, pt)
	if err != nil {
		return nil, fmt.Errorf("无法绘制文本: %v", err)
	}

	return img, nil
}

// RenderMultilineText 渲染多行文本为图像
// 参数lines: 文本行数组，每个元素为一行文本
// 参数textColor: 文本颜色
// 参数lineSpacing: 行间距（像素）
// 返回包含渲染文本的图像或错误信息
// 支持多行中文文本的排版和渲染
func (r *Renderer) RenderMultilineText(lines []string, textColor color.Color, lineSpacing int) (image.Image, error) {
	// 如果没有文本行，返回最小图像
	if len(lines) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
	}

	face := truetype.NewFace(r.font, &truetype.Options{Size: r.size, DPI: r.dpi})
	metrics := face.Metrics()
	// 使用字体文件中定义的标准行高，这是最可靠的方式
	fontLineHeight := int(metrics.Height >> 6)

	maxWidth := 0
	for _, line := range lines {
		w, _ := r.GetTextBounds(line) // 只需要宽度用于计算画布最大宽度
		if w > maxWidth {
			maxWidth = w
		}
	}

	// 根据标准行高计算总高度
	totalHeight := (fontLineHeight + lineSpacing) * len(lines)

	// 设置默认尺寸（防止计算失败）
	if maxWidth == 0 {
		maxWidth = 100
	}
	if totalHeight == 0 {
		totalHeight = 20 * len(lines)
	}

	// 创建图像并填充透明背景
	img := image.NewRGBA(image.Rect(0, 0, maxWidth, totalHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	// 设置FreeType渲染参数
	r.context.SetClip(img.Bounds())
	r.context.SetDst(img)
	r.context.SetSrc(&image.Uniform{textColor})

	// 逐行绘制文本
	ascent := int(metrics.Ascent >> 6)
	y := ascent // 第一行的基线位置
	for _, line := range lines {
		pt := freetype.Pt(0, y) // 当前行的绘制位置
		_, err := r.context.DrawString(line, pt)
		if err != nil {
			return nil, fmt.Errorf("无法绘制文本行: %v", err)
		}
		// 根据标准行高移动到下一行
		y += fontLineHeight + lineSpacing
	}

	return img, nil
}

// MeasureString 测量文本字符串的尺寸
// 参数text: 要测量的文本字符串
// 返回文本的宽度和高度（像素）
// 这是GetTextBounds的简化接口，用于快速获取文本尺寸
func (r *Renderer) MeasureString(text string) (width, height int) {
	return r.GetTextBounds(text)
}

// DrawTextAt 在指定位置绘制文本到目标图像
// 参数dst: 目标图像（实现draw.Image接口）
// 参数x,y: 绘制位置的左上角坐标
// 参数text: 要绘制的文本字符串
// 参数textColor: 文本颜色
// 返回绘制过程中的错误信息
// 先渲染文本为图像，再复制到目标位置，支持透明度混合
func (r *Renderer) DrawTextAt(dst draw.Image, x, y int, text string, textColor color.Color) error {
	// 渲染文本为图像
	textImg, err := r.RenderText(text, textColor)
	if err != nil {
		return err
	}

	// 获取文本图像边界
	bounds := textImg.Bounds()
	// 使用Over混合模式将文本图像绘制到目标图像的指定位置
	// Over模式支持透明度，可以在现有内容上叠加文本
	draw.Draw(dst, image.Rect(x, y, x+bounds.Dx(), y+bounds.Dy()), textImg, bounds.Min, draw.Over)
	return nil
}

// validateFontFormat 检查字体文件格式是否支持
func validateFontFormat(fontData []byte) error {
	if len(fontData) < 4 {
		return fmt.Errorf("字体文件太小，无法确定格式")
	}

	// 检查TTF签名
	if bytes.HasPrefix(fontData, []byte{0x00, 0x01, 0x00, 0x00}) {
		return nil // TTF格式
	}

	// 检查TTC签名 (TrueType Collection)
	if bytes.HasPrefix(fontData, []byte("ttcf")) {
		return nil // TTC格式
	}

	// 检查OTF签名
	if bytes.HasPrefix(fontData, []byte("OTTO")) {
		return fmt.Errorf("检测到OTF格式，freetype库对此格式支持有限")
	}

	// 检查WOFF签名
	if bytes.HasPrefix(fontData, []byte("wOFF")) {
		return fmt.Errorf("不支持WOFF格式，请使用TTF格式")
	}

	// 检查WOFF2签名
	if bytes.HasPrefix(fontData, []byte("wOF2")) {
		return fmt.Errorf("不支持WOFF2格式，请使用TTF格式")
	}

	return fmt.Errorf("未知的字体格式，仅支持TTF格式")
}

// isOTFFont 检查是否为OTF字体
func isOTFFont(fontData []byte) bool {
	return len(fontData) >= 4 && bytes.HasPrefix(fontData, []byte("OTTO"))
}

// GetSupportedFontInfo 返回支持的字体格式信息
func GetSupportedFontInfo() string {
	return `支持的字体格式:
- TTF (TrueType Font) - 推荐
- TTC (TrueType Collection)

不支持的格式:
- OTF (OpenType Font) - 部分支持，可能出现错误
- WOFF (Web Open Font Format)
- WOFF2 (Web Open Font Format 2.0)

建议：
1. 将OTF字体转换为TTF格式
2. 使用免费转换工具如FontForge
3. 或下载TTF版本的相同字体`
}

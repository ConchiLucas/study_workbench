package glyph

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	CanvasSize  = 1024
	MarginRatio = 0.12
	// Thick enough to stay visible when the list previews at ~88–104px.
	gridStroke = 12
	defaultSize = 720
)

var (
	bgColor = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	fgColor = color.RGBA{R: 0x1A, G: 0x1A, B: 0x1A, A: 0xFF}
	// Soft 田字格 — darker than paper gray so thumbs still show the cross.
	gridColor = color.RGBA{R: 0xB8, G: 0xB8, B: 0xB8, A: 0xFF}
	// 数学本淡蓝方格（国内算术练习簿常见；GitHub worksheet 多为纯白，方格更贴学龄前语境）。
	mathGridColor   = color.RGBA{R: 0xB5, G: 0xD0, B: 0xE4, A: 0xFF}
	mathBorderColor = color.RGBA{R: 0x7A, G: 0xA8, B: 0xC8, A: 0xFF}
)

type Renderer struct {
	ft   *opentype.Font
	face font.Face // default ~720 face for single-rune CJK
}

var (
	defaultOnce sync.Once
	defaultR    *Renderer
	defaultErr  error
)

func Default() (*Renderer, error) {
	defaultOnce.Do(func() {
		path := os.Getenv("GLYPH_FONT_PATH")
		if path == "" {
			path = "assets/fonts/NotoSerifSC-Regular.otf"
		}
		defaultR, defaultErr = NewFromFile(path)
	})
	return defaultR, defaultErr
}

func NewFromFile(path string) (*Renderer, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read font: %w", err)
	}
	return NewFromBytes(raw)
}

func NewFromBytes(raw []byte) (*Renderer, error) {
	ft, err := opentype.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}
	face, err := faceAt(ft, defaultSize)
	if err != nil {
		return nil, err
	}
	return &Renderer{ft: ft, face: face}, nil
}

func faceAt(ft *opentype.Font, size float64) (font.Face, error) {
	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("font face: %w", err)
	}
	return face, nil
}

func innerBoxWidth() float64 {
	margin := float64(CanvasSize) * MarginRatio
	return float64(CanvasSize) - 2*margin
}

// RenderPNG draws 1–16 runes (hanzi / pinyin) on the locked 田字格 template.
func (r *Renderer) RenderPNG(text string) ([]byte, error) {
	return r.renderTextPNG(text, true)
}

// RenderEnglishPNG draws an English word on a plain white card (no 田字格).
// Matches common GitHub / worksheet flashcards: spelling only on white.
func (r *Renderer) RenderEnglishPNG(text string) ([]byte, error) {
	return r.renderTextPNG(text, false)
}

// RenderMathNumberPNG draws Arabic numerals 1–20 on white 田字格 (same template as literacy glyphs).
// Distinct from RenderMathPNG which uses blue 数学本 graph paper for equations.
func (r *Renderer) RenderMathNumberPNG(text string) ([]byte, error) {
	return r.renderTextPNG(text, true)
}

func (r *Renderer) renderTextPNG(text string, withTianZiGe bool) ([]byte, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty character")
	}
	n := utf8.RuneCountInString(text)
	if n < 1 || n > 16 {
		return nil, fmt.Errorf("expected 1–16 runes, got %q (%d)", text, n)
	}

	face := r.face
	var sizedFace font.Face
	if n > 1 {
		maxW := fixed.Int26_6(innerBoxWidth() * 0.80 * 64)
		size := float64(defaultSize)
		minSize := 28.0
		if n <= 4 {
			minSize = 48
		}
		var err error
		for size >= minSize {
			sizedFace, err = faceAt(r.ft, size)
			if err != nil {
				return nil, err
			}
			d := &font.Drawer{Face: sizedFace}
			if d.MeasureString(text) <= maxW {
				face = sizedFace
				break
			}
			closeFace(sizedFace)
			sizedFace = nil
			size -= 8
		}
		if face == r.face {
			// Fallback: use smallest size tried.
			sizedFace, err = faceAt(r.ft, minSize)
			if err != nil {
				return nil, err
			}
			face = sizedFace
		}
		defer closeFace(sizedFace)
	}

	img := image.NewRGBA(image.Rect(0, 0, CanvasSize, CanvasSize))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)
	if withTianZiGe {
		drawTianZiGe(img)
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(fgColor),
		Face: face,
	}
	advance := d.MeasureString(text)
	metrics := face.Metrics()
	x := (fixed.I(CanvasSize) - advance) / 2
	ascent := metrics.Ascent
	descent := metrics.Descent
	y := (fixed.I(CanvasSize) + ascent - descent) / 2

	d.Dot = fixed.Point26_6{X: x, Y: y}
	d.DrawString(text)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RenderMathPNG draws arithmetic text (e.g. "1+2", "3 ○ 5", "圆形") on soft square graph paper.
// Unlike literacy 田字格, math uses a light blue quad grid like Chinese 数学本.
func (r *Renderer) RenderMathPNG(text string) ([]byte, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty math text")
	}
	n := utf8.RuneCountInString(text)
	if n < 1 || n > 12 {
		return nil, fmt.Errorf("expected 1–12 runes, got %q (%d)", text, n)
	}

	face := r.face
	var sizedFace font.Face
	maxW := fixed.Int26_6(innerBoxWidth() * 0.86 * 64)
	size := float64(defaultSize)
	if n >= 3 {
		size = 520
	}
	if n >= 6 {
		size = 360
	}
	var err error
	for size >= 40 {
		sizedFace, err = faceAt(r.ft, size)
		if err != nil {
			return nil, err
		}
		d := &font.Drawer{Face: sizedFace}
		if d.MeasureString(text) <= maxW {
			face = sizedFace
			break
		}
		closeFace(sizedFace)
		sizedFace = nil
		size -= 12
	}
	if sizedFace == nil {
		sizedFace, err = faceAt(r.ft, 40)
		if err != nil {
			return nil, err
		}
		face = sizedFace
	}
	defer closeFace(sizedFace)

	img := image.NewRGBA(image.Rect(0, 0, CanvasSize, CanvasSize))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)
	drawMathGraphPaper(img)

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(fgColor),
		Face: face,
	}
	advance := d.MeasureString(text)
	metrics := face.Metrics()
	x := (fixed.I(CanvasSize) - advance) / 2
	ascent := metrics.Ascent
	descent := metrics.Descent
	y := (fixed.I(CanvasSize) + ascent - descent) / 2

	d.Dot = fixed.Point26_6{X: x, Y: y}
	d.DrawString(text)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// drawMathGraphPaper paints a light blue square grid + outer border (数学本风格).
func drawMathGraphPaper(img *image.RGBA) {
	size := CanvasSize
	margin := int(float64(size) * MarginRatio)
	left, top := margin, margin
	right, bottom := CanvasSize-margin-1, CanvasSize-margin-1

	const cells = 8
	cell := (right - left) / cells
	stroke := 4
	borderStroke := 10

	for i := 1; i < cells; i++ {
		x := left + i*cell
		y := top + i*cell
		vline(img, top, bottom, x, stroke, mathGridColor)
		hline(img, left, right, y, stroke, mathGridColor)
	}
	hline(img, left, right, top, borderStroke, mathBorderColor)
	hline(img, left, right, bottom, borderStroke, mathBorderColor)
	vline(img, top, bottom, left, borderStroke, mathBorderColor)
	vline(img, top, bottom, right, borderStroke, mathBorderColor)
}

func closeFace(face font.Face) {
	if face == nil {
		return
	}
	if c, ok := face.(interface{ Close() error }); ok {
		_ = c.Close()
	}
}

// drawTianZiGe paints a light practice square: outer border + cross (+).
func drawTianZiGe(img *image.RGBA) {
	size := CanvasSize
	margin := int(float64(size) * MarginRatio)
	left, top := margin, margin
	right, bottom := CanvasSize-margin-1, CanvasSize-margin-1
	midX := (left + right) / 2
	midY := (top + bottom) / 2

	hline(img, left, right, top, gridStroke, gridColor)
	hline(img, left, right, bottom, gridStroke, gridColor)
	vline(img, top, bottom, left, gridStroke, gridColor)
	vline(img, top, bottom, right, gridStroke, gridColor)
	hline(img, left, right, midY, gridStroke, gridColor)
	vline(img, top, bottom, midX, gridStroke, gridColor)
}

func hline(img *image.RGBA, x0, x1, y, thickness int, c color.RGBA) {
	half := thickness / 2
	for t := -half; t <= thickness-half-1; t++ {
		yy := y + t
		if yy < 0 || yy >= CanvasSize {
			continue
		}
		for x := x0; x <= x1; x++ {
			img.SetRGBA(x, yy, c)
		}
	}
}

func vline(img *image.RGBA, y0, y1, x, thickness int, c color.RGBA) {
	half := thickness / 2
	for t := -half; t <= thickness-half-1; t++ {
		xx := x + t
		if xx < 0 || xx >= CanvasSize {
			continue
		}
		for y := y0; y <= y1; y++ {
			img.SetRGBA(xx, y, c)
		}
	}
}

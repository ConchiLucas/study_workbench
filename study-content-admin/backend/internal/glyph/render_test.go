package glyph_test

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/conchi/study-content-admin/internal/glyph"
	"github.com/stretchr/testify/require"
)

func TestRenderPNG(t *testing.T) {
	r, err := glyph.NewFromFile("../../assets/fonts/NotoSerifSC-Regular.otf")
	require.NoError(t, err)
	pngBytes, err := r.RenderPNG("人")
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(pngBytes))
	require.NoError(t, err)
	require.Equal(t, glyph.CanvasSize, img.Bounds().Dx())
	require.Equal(t, glyph.CanvasSize, img.Bounds().Dy())

	// Outer corner stays white (grid is inset by MarginRatio).
	r0, g0, b0, a0 := img.At(0, 0).RGBA()
	require.Equal(t, uint32(0xffff), r0)
	require.Equal(t, uint32(0xffff), g0)
	require.Equal(t, uint32(0xffff), b0)
	require.Equal(t, uint32(0xffff), a0)

	// Top-left of 田字格 (margin inset) should be grid gray, not white.
	size := glyph.CanvasSize
	margin := int(float64(size) * glyph.MarginRatio)
	gr, gg, gb, _ := img.At(margin, margin).RGBA()
	require.Less(t, gr, uint32(0xffff))
	require.Equal(t, gr, gg)
	require.Equal(t, gg, gb)
}

func TestRenderPNGMultiLetter(t *testing.T) {
	r, err := glyph.NewFromFile("../../assets/fonts/NotoSerifSC-Regular.otf")
	require.NoError(t, err)
	for _, letter := range []string{"zh", "ang"} {
		t.Run(letter, func(t *testing.T) {
			pngBytes, err := r.RenderPNG(letter)
			require.NoError(t, err)
			img, err := png.Decode(bytes.NewReader(pngBytes))
			require.NoError(t, err)
			require.Equal(t, glyph.CanvasSize, img.Bounds().Dx())
			require.Equal(t, glyph.CanvasSize, img.Bounds().Dy())
		})
	}
}

func TestRenderEnglishPNGPlainWhite(t *testing.T) {
	r, err := glyph.NewFromFile("../../assets/fonts/NotoSerifSC-Regular.otf")
	require.NoError(t, err)
	pngBytes, err := r.RenderEnglishPNG("cat")
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(pngBytes))
	require.NoError(t, err)
	require.Equal(t, glyph.CanvasSize, img.Bounds().Dx())

	// Margin corner stays pure white — no 田字格 lines.
	size := glyph.CanvasSize
	margin := int(float64(size) * glyph.MarginRatio)
	rr, gg, bb, aa := img.At(margin, margin).RGBA()
	require.Equal(t, uint32(0xffff), rr)
	require.Equal(t, uint32(0xffff), gg)
	require.Equal(t, uint32(0xffff), bb)
	require.Equal(t, uint32(0xffff), aa)
}

func TestRenderMathNumberPNGWhiteGrid(t *testing.T) {
	r, err := glyph.NewFromFile("../../assets/fonts/NotoSerifSC-Regular.otf")
	require.NoError(t, err)
	for _, text := range []string{"1", "7", "20"} {
		t.Run(text, func(t *testing.T) {
			pngBytes, err := r.RenderMathNumberPNG(text)
			require.NoError(t, err)
			img, err := png.Decode(bytes.NewReader(pngBytes))
			require.NoError(t, err)
			require.Equal(t, glyph.CanvasSize, img.Bounds().Dx())

			// Outer corner white; 田字格 corner is gray (not blue math paper).
			r0, g0, b0, _ := img.At(0, 0).RGBA()
			require.Equal(t, uint32(0xffff), r0)
			require.Equal(t, uint32(0xffff), g0)
			require.Equal(t, uint32(0xffff), b0)

			size := glyph.CanvasSize
			margin := int(float64(size) * glyph.MarginRatio)
			gr, gg, gb, _ := img.At(margin, margin).RGBA()
			require.Less(t, gr, uint32(0xffff))
			require.Equal(t, gr, gg)
			require.Equal(t, gg, gb)
		})
	}
}

func TestRenderMathPNG(t *testing.T) {
	r, err := glyph.NewFromFile("../../assets/fonts/NotoSerifSC-Regular.otf")
	require.NoError(t, err)
	for _, text := range []string{"1+2", "3 ○ 5", "10+10"} {
		t.Run(text, func(t *testing.T) {
			pngBytes, err := r.RenderMathPNG(text)
			require.NoError(t, err)
			img, err := png.Decode(bytes.NewReader(pngBytes))
			require.NoError(t, err)
			require.Equal(t, glyph.CanvasSize, img.Bounds().Dx())

			size := glyph.CanvasSize
			margin := int(float64(size) * glyph.MarginRatio)
			cr, cg, cb, _ := img.At(margin, margin).RGBA()
			require.Less(t, cr, uint32(0xffff))
			require.Greater(t, cb, cr)
			_ = cg
		})
	}
}

func TestRenderMathShapePNG(t *testing.T) {
	r, err := glyph.NewFromFile("../../assets/fonts/NotoSerifSC-Regular.otf")
	require.NoError(t, err)
	for _, title := range []string{"圆形", "正方形", "长方形", "三角形", "椭圆形", "梯形", "菱形", "五角星"} {
		t.Run(title, func(t *testing.T) {
			pngBytes, err := r.RenderMathShapePNG(title)
			require.NoError(t, err)
			img, err := png.Decode(bytes.NewReader(pngBytes))
			require.NoError(t, err)
			require.Equal(t, glyph.CanvasSize, img.Bounds().Dx())
			mid := glyph.CanvasSize / 2
			cr, cg, cb, _ := img.At(mid, mid).RGBA()
			require.Less(t, cr, uint32(0x8000))
			require.Equal(t, cr, cg)
			require.Equal(t, cg, cb)
		})
	}
}

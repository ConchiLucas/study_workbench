package glyph

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"
)

// Shape titles used by the math catalog (认识图形).
var shapeTitles = map[string]string{
	"圆形": "circle", "正方形": "square", "长方形": "rect", "三角形": "triangle",
	"椭圆形": "oval", "梯形": "trapezoid", "菱形": "rhombus", "五角星": "star",
}

func IsMathShapeTitle(title string) bool {
	_, ok := shapeTitles[strings.TrimSpace(title)]
	return ok
}

// RenderMathShapePNG draws a filled geometric shape on soft blue graph paper.
func (r *Renderer) RenderMathShapePNG(title string) ([]byte, error) {
	title = strings.TrimSpace(title)
	key, ok := shapeTitles[title]
	if !ok {
		return nil, fmt.Errorf("未知图形: %q", title)
	}

	img := image.NewRGBA(image.Rect(0, 0, CanvasSize, CanvasSize))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)
	drawMathGraphPaper(img)
	drawMathShape(img, key, fgColor)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawMathShape(img *image.RGBA, key string, c color.RGBA) {
	// Geometry in a 100×100 design box, mapped into the inner graph-paper area.
	size := CanvasSize
	margin := int(float64(size) * MarginRatio)
	pad := int(float64(size-2*margin) * 0.12)
	left := margin + pad
	top := margin + pad
	box := size - 2*margin - 2*pad
	scale := func(x, y float64) image.Point {
		return image.Point{
			X: left + int(x/100*float64(box)+0.5),
			Y: top + int(y/100*float64(box)+0.5),
		}
	}

	switch key {
	case "circle":
		fillEllipse(img, scale(50, 50), box*34/100, box*34/100, c)
	case "oval":
		fillEllipse(img, scale(50, 50), box*40/100, box*26/100, c)
	case "square":
		fillPolygon(img, []image.Point{
			scale(18, 18), scale(82, 18), scale(82, 82), scale(18, 82),
		}, c)
	case "rect":
		fillPolygon(img, []image.Point{
			scale(10, 30), scale(90, 30), scale(90, 70), scale(10, 70),
		}, c)
	case "triangle":
		fillPolygon(img, []image.Point{
			scale(50, 16), scale(86, 82), scale(14, 82),
		}, c)
	case "trapezoid":
		fillPolygon(img, []image.Point{
			scale(30, 22), scale(70, 22), scale(88, 78), scale(12, 78),
		}, c)
	case "rhombus":
		fillPolygon(img, []image.Point{
			scale(50, 12), scale(86, 50), scale(50, 88), scale(14, 50),
		}, c)
	case "star":
		fillPolygon(img, []image.Point{
			scale(50, 10), scale(61, 38), scale(92, 38), scale(66, 57),
			scale(76, 87), scale(50, 68), scale(24, 87), scale(34, 57),
			scale(8, 38), scale(39, 38),
		}, c)
	}
}

func fillEllipse(img *image.RGBA, center image.Point, rx, ry int, c color.RGBA) {
	if rx <= 0 || ry <= 0 {
		return
	}
	for y := -ry; y <= ry; y++ {
		yy := float64(y) / float64(ry)
		w := int(math.Sqrt(math.Max(0, 1-yy*yy))*float64(rx) + 0.5)
		for x := -w; x <= w; x++ {
			px, py := center.X+x, center.Y+y
			if px >= 0 && px < CanvasSize && py >= 0 && py < CanvasSize {
				img.SetRGBA(px, py, c)
			}
		}
	}
}

func fillPolygon(img *image.RGBA, pts []image.Point, c color.RGBA) {
	if len(pts) < 3 {
		return
	}
	minY, maxY := pts[0].Y, pts[0].Y
	for _, p := range pts[1:] {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	if minY < 0 {
		minY = 0
	}
	if maxY >= CanvasSize {
		maxY = CanvasSize - 1
	}
	n := len(pts)
	for y := minY; y <= maxY; y++ {
		var xs []int
		for i := 0; i < n; i++ {
			a, b := pts[i], pts[(i+1)%n]
			if a.Y == b.Y {
				continue
			}
			if (y < a.Y && y < b.Y) || (y >= a.Y && y >= b.Y) {
				continue
			}
			t := float64(y-a.Y) / float64(b.Y-a.Y)
			x := int(float64(a.X) + t*float64(b.X-a.X) + 0.5)
			xs = append(xs, x)
		}
		// sort xs
		for i := 1; i < len(xs); i++ {
			for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
				xs[j], xs[j-1] = xs[j-1], xs[j]
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			x0, x1 := xs[i], xs[i+1]
			if x0 < 0 {
				x0 = 0
			}
			if x1 >= CanvasSize {
				x1 = CanvasSize - 1
			}
			for x := x0; x <= x1; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

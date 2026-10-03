// Package gfx generates all pixel-art images procedurally. It depends only on
// the standard image packages so it can be tested (and dumped to PNG) headless.
package gfx

import (
	"image"
	"image/color"
	"math"
)

// Canvas is a thin helper for drawing pixel art into an RGBA image.
type Canvas struct{ Img *image.RGBA }

// NewCanvas allocates a transparent canvas.
func NewCanvas(w, h int) *Canvas { return &Canvas{image.NewRGBA(image.Rect(0, 0, w, h))} }

// C builds a color.
func C(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

// Px sets one pixel (bounds-checked).
func (c *Canvas) Px(x, y int, col color.RGBA) {
	if image.Pt(x, y).In(c.Img.Rect) {
		c.Img.SetRGBA(x, y, col)
	}
}

// Rect fills [x0,x1) x [y0,y1).
func (c *Canvas) Rect(x0, y0, x1, y1 int, col color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			c.Px(x, y, col)
		}
	}
}

// Frame draws a 1px rectangle outline.
func (c *Canvas) Frame(x0, y0, x1, y1 int, col color.RGBA) {
	for x := x0; x < x1; x++ {
		c.Px(x, y0, col)
		c.Px(x, y1-1, col)
	}
	for y := y0; y < y1; y++ {
		c.Px(x0, y, col)
		c.Px(x1-1, y, col)
	}
}

// Line draws a Bresenham line.
func (c *Canvas) Line(x0, y0, x1, y1 int, col color.RGBA) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		c.Px(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// Disc fills a filled circle.
func (c *Canvas) Disc(cx, cy, r int, col color.RGBA) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				c.Px(cx+x, cy+y, col)
			}
		}
	}
}

// Ellipse fills an ellipse centred at cx,cy with radii rx,ry.
func (c *Canvas) Ellipse(cx, cy, rx, ry int, col color.RGBA) {
	for y := -ry; y <= ry; y++ {
		for x := -rx; x <= rx; x++ {
			if float64(x*x)/float64(rx*rx)+float64(y*y)/float64(ry*ry) <= 1 {
				c.Px(cx+x, cy+y, col)
			}
		}
	}
}

// Outline darkens transparent pixels that touch opaque ones, giving sprites a crisp edge.
func (c *Canvas) Outline(col color.RGBA) {
	b := c.Img.Rect
	src := image.NewRGBA(b)
	copy(src.Pix, c.Img.Pix)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if src.RGBAAt(x, y).A != 0 {
				continue
			}
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if image.Pt(nx, ny).In(b) && src.RGBAAt(nx, ny).A != 0 {
					c.Img.SetRGBA(x, y, col)
					break
				}
			}
		}
	}
}

// Shade multiplies RGB by k.
func Shade(col color.RGBA, k float64) color.RGBA {
	f := func(v uint8) uint8 { return uint8(math.Max(0, math.Min(255, float64(v)*k))) }
	return color.RGBA{f(col.R), f(col.G), f(col.B), col.A}
}

// Mix blends two colors.
func Mix(a, b color.RGBA, t float64) color.RGBA {
	f := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 255}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// hash2 is a cheap deterministic 2D hash in [0,1).
func hash2(x, y int, seed uint32) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + seed*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h&0xFFFF) / 65536.0
}

// TeamColors are the four player colours.
var TeamColors = []color.RGBA{
	{225, 72, 62, 255},
	{70, 132, 238, 255},
	{92, 198, 92, 255},
	{238, 206, 62, 255},
}

// TeamName returns a readable name for a colour index.
func TeamName(i int) string {
	return [...]string{"Красные", "Синие", "Зелёные", "Жёлтые"}[i%4]
}

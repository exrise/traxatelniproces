package gfx

import (
	"image"
	"image/color"
	"math"
)

// TerrainColor computes the display color for one terrain pixel.
// mat: 1 dirt, 2 rock. airAbove is the distance in px up to the nearest air pixel (0 = directly under air).
func TerrainColor(mat uint8, x, y, airAbove int, edgeSide bool) color.RGBA {
	n := hash2(x, y, 3)
	n2 := hash2(x/3, y/3, 9)
	var col color.RGBA
	switch mat {
	case 2:
		base := 118 + int(n*22) + int(n2*14)
		col = C(uint8(base), uint8(base+2), uint8(base+10))
		if (x/6+y/5)%7 == 0 {
			col = Shade(col, 0.88)
		}
	default:
		switch {
		case airAbove < 3:
			g := 150 + int(n*40)
			col = C(uint8(70+n*20), uint8(g), uint8(58+n*14))
		case airAbove < 6:
			col = C(uint8(92+n*16), uint8(72+n*12), uint8(46+n*10))
		default:
			t := 1 - math.Min(1, float64(airAbove)/260)
			r := 120 + int(n*18) + int(n2*12)
			col = C(uint8(float64(r)*(0.65+0.35*t)), uint8(float64(r-34)*(0.65+0.35*t)), uint8(float64(r-66)*(0.65+0.35*t)))
			if n > 0.97 {
				col = C(168, 150, 120) // pebbles
			}
		}
	}
	if edgeSide {
		col = Shade(col, 0.72)
	}
	return col
}

// PaintTerrain fills rect r of img from the mask (w x h).
func PaintTerrain(img *image.RGBA, mask []uint8, w, h int, r image.Rectangle) {
	r = r.Intersect(image.Rect(0, 0, w, h))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			m := mask[y*w+x]
			if m == 0 {
				img.SetRGBA(x, y, color.RGBA{})
				continue
			}
			air := 99
			for k := 1; k <= 8; k++ {
				if y-k < 0 || mask[(y-k)*w+x] == 0 {
					air = k - 1
					break
				}
			}
			edge := false
			if x > 0 && mask[y*w+x-1] == 0 || x < w-1 && mask[y*w+x+1] == 0 {
				edge = true
			}
			if y == h-1 {
				edge = false
			}
			img.SetRGBA(x, y, TerrainColor(m, x, y, air, edge))
		}
	}
}

// Sky renders a vertical gradient with a sun.
func Sky(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	top, bottom := C(70, 130, 210), C(190, 224, 244)
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		col := Mix(top, bottom, t)
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, col)
		}
	}
	c := &Canvas{img}
	sx, sy := w*3/4, h/6
	for r := 70; r > 0; r -= 2 {
		a := 1 - float64(r)/70
		c.Disc(sx, sy, r, Mix(Mix(top, bottom, 0.2), C(255, 250, 214), a*a))
	}
	c.Disc(sx, sy, 22, C(255, 248, 200))
	return img
}

// MountainLayer renders a parallax silhouette strip of the given size.
func MountainLayer(w, h int, seed uint32, col color.RGBA, amp, base float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c := &Canvas{img}
	two := 2 * math.Pi / float64(w) // periodic so the layer tiles seamlessly
	ph := float64(seed)
	for x := 0; x < w; x++ {
		fx := float64(x)
		y := base - amp*(0.5+0.28*math.Sin(fx*two*3+ph)+0.18*math.Sin(fx*two*7+ph*2)+0.08*math.Sin(fx*two*19+ph*3))
		for yy := int(y); yy < h; yy++ {
			c.Px(x, yy, col)
		}
	}
	return img
}

// Cloud draws a puffy cloud sprite.
func Cloud(seed uint32) *image.RGBA {
	c := NewCanvas(120, 44)
	white := C(255, 255, 255)
	shade := C(225, 235, 248)
	for i := 0; i < 7; i++ {
		x := 16 + int(hash2(i, 1, seed)*88)
		r := 8 + int(hash2(i, 2, seed)*10)
		c.Disc(x, 28-int(hash2(i, 3, seed)*8), r, white)
	}
	c.Ellipse(60, 32, 50, 7, shade)
	return c.Img
}

package gfx

import (
	"image"
	"image/color"
)

var (
	concreteC = C(150, 152, 158)
	steelC    = C(78, 88, 100)
	sandC     = C(204, 176, 112)
	woodC     = C(128, 88, 52)
	oliveC    = C(92, 108, 66)
	metalC    = C(110, 116, 124)
	outlineC  = C(26, 24, 30)
)

// Barrel describes the rotating gun part of a weapon structure.
type Barrel struct {
	Len, Thick int
	Col        color.RGBA
	// PivotY is the pivot height above the structure's top edge (negative = below).
	PivotY int
}

// BarrelFor returns the barrel drawing parameters, ok=false if the structure has none.
func BarrelFor(id string) (Barrel, bool) {
	switch id {
	case "dshk":
		return Barrel{18, 3, C(36, 36, 40), 10}, true
	case "d30":
		return Barrel{34, 4, C(60, 66, 54), 12}, true
	case "grad":
		return Barrel{34, 8, C(70, 80, 60), 6}, true
	case "kornet":
		return Barrel{22, 5, C(60, 64, 50), 8}, true
	case "iskander":
		return Barrel{40, 8, C(210, 212, 216), 8}, true
	case "oreshnik":
		return Barrel{52, 9, C(200, 204, 210), 8}, true
	case "zu23":
		return Barrel{20, 4, C(36, 36, 40), 10}, true
	}
	return Barrel{}, false
}

func sandbagRow(c *Canvas, x0, y0, x1, y1 int) {
	for y := y0; y < y1; y += 5 {
		off := ((y - y0) / 5 % 2) * 4
		for x := x0 - off; x < x1; x += 9 {
			xa, xb := max(x0, x), min(x1, x+9)
			if xb <= xa {
				continue
			}
			c.Rect(xa, y, xb, min(y1, y+5), sandC)
			c.Rect(xa, min(y1, y+5)-1, xb, min(y1, y+5), Shade(sandC, 0.72))
			c.Px(xa, y, Shade(sandC, 0.8))
			c.Px(xb-1, y, Shade(sandC, 0.8))
		}
	}
}

func wheel(c *Canvas, cx, cy, r int) {
	c.Disc(cx, cy, r, C(30, 30, 34))
	c.Disc(cx, cy, r-2, C(70, 72, 78))
	c.Disc(cx, cy, 1, C(150, 150, 156))
}

func truck(c *Canvas, w, h int, body color.RGBA) {
	c.Rect(2, h-12, w-2, h-5, body)
	c.Rect(w-14, h-18, w-2, h-5, Shade(body, 1.1)) // cab
	c.Rect(w-11, h-16, w-5, h-11, C(150, 200, 230))
	wheel(c, 9, h-5, 4)
	wheel(c, w/2, h-5, 4)
	wheel(c, w-10, h-5, 4)
}

// StructSprite renders a structure of w x h cells (w*16 x h*16 px).
func StructSprite(id string, w, h int) *image.RGBA {
	W, H := w*16, h*16
	c := NewCanvas(W, H)
	switch id {
	case "sandbag":
		sandbagRow(c, 0, 0, W, H)
	case "concrete":
		c.Rect(0, 0, W, H, concreteC)
		for x := 0; x < W; x += 16 {
			c.Rect(x, 0, x+1, H, Shade(concreteC, 0.7))
		}
		c.Rect(0, 0, W, 2, Shade(concreteC, 1.2))
		c.Rect(0, H-2, W, H, Shade(concreteC, 0.75))
		for i := 0; i < 8; i++ {
			c.Px(int(hash2(i, 1, 7)*float64(W)), int(hash2(i, 2, 7)*float64(H)), Shade(concreteC, 0.82))
		}
	case "window":
		c.Rect(0, 0, W, H, concreteC)
		c.Rect(0, 0, W, 2, Shade(concreteC, 1.2))
		c.Rect(0, H-2, W, H, Shade(concreteC, 0.75))
		c.Rect(4, 4, W-4, H-4, C(24, 28, 40))
		c.Rect(6, 6, W-6, H-6, C(120, 190, 235))
		c.Line(8, H-7, W/2-2, 6, C(215, 238, 252))
		c.Line(W/2+6, H-7, W-9, 6, C(215, 238, 252))
		c.Rect(W/2-1, 4, W/2+1, H-4, C(24, 28, 40))
	case "armor":
		c.Rect(0, 0, W, H, steelC)
		c.Rect(1, 1, W-1, H-1, Shade(steelC, 1.15))
		for y := 3; y < H; y += 8 {
			c.Px(3, y, C(190, 196, 206))
			c.Px(W-4, y, C(190, 196, 206))
		}
		c.Line(3, 3, W-4, H-4, Shade(steelC, 0.8))
	case "ezh":
		for _, d := range [][4]int{{2, H - 2, W - 3, 2}, {2, 2, W - 3, H - 2}} {
			c.Line(d[0], d[1], d[2], d[3], steelC)
			c.Line(d[0], d[1]-1, d[2], d[3]-1, Shade(steelC, 1.3))
		}
		c.Line(W/2, 1, W/2, H-1, steelC)
	case "bunker":
		c.Rect(0, 8, W, H, concreteC)
		c.Ellipse(W/2, 12, W/2-1, 11, concreteC)
		c.Rect(0, H-3, W, H, Shade(concreteC, 0.7))
		c.Rect(10, 20, W-10, 27, C(20, 22, 26)) // embrasure
		c.Rect(10, 20, W-10, 21, C(70, 72, 78))
		c.Rect(W/2-8, 4, W/2+8, 7, C(96, 150, 80)) // turf on top
		c.Rect(W-18, H-18, W-8, H, C(24, 24, 28))  // door
	case "net":
		c.Rect(0, 0, 2, H, woodC)
		c.Rect(W-2, 0, W, H, woodC)
		for x := 2; x < W-2; x += 4 {
			c.Line(x, 0, x+4, H-1, C(120, 160, 120))
			c.Line(x+4, 0, x, H-1, C(120, 160, 120))
		}
	case "hq":
		c.Rect(2, 16, W-2, H, C(120, 96, 78))
		c.Rect(2, 16, W-2, 20, C(160, 70, 60)) // roof band
		c.Rect(4, 8, W-4, 18, C(142, 112, 90))
		c.Rect(W/2-5, H-18, W/2+5, H, C(50, 36, 30)) // door
		c.Rect(10, 24, 20, 32, C(140, 210, 240))
		c.Rect(W-20, 24, W-10, 32, C(140, 210, 240))
		c.Rect(W/2-1, 0, W/2+1, 10, C(200, 200, 205)) // flagpole
		c.Rect(W/2+1, 0, W/2+12, 6, C(255, 255, 255)) // flag (recoloured at draw)
	case "dshk":
		sandbagRow(c, 0, H-14, W, H)
		c.Rect(W/2-8, H-18, W/2+8, H-12, C(50, 54, 50))
		c.Line(W/2-4, H-12, W/2-8, H-2, C(30, 30, 34))
		c.Line(W/2+4, H-12, W/2+8, H-2, C(30, 30, 34))
	case "d30":
		wheel(c, 12, H-8, 7)
		wheel(c, W-12, H-8, 7)
		c.Rect(8, H-18, W-8, H-10, oliveC)
		c.Rect(W/2-6, H-24, W/2+6, H-16, Shade(oliveC, 0.85))
	case "grad":
		truck(c, W, H, oliveC)
		c.Rect(4, H-18, W-18, H-12, C(60, 70, 52))
	case "kornet":
		c.Line(W/2, H-14, 6, H-1, C(40, 44, 40))
		c.Line(W/2, H-14, W-6, H-1, C(40, 44, 40))
		c.Rect(W/2-9, H-24, W/2+9, H-12, oliveC)
		c.Rect(W/2-6, H-22, W/2+6, H-18, C(30, 34, 30))
	case "iskander":
		truck(c, W, H, C(86, 98, 76))
		c.Rect(6, H-16, W-20, H-12, C(60, 66, 56))
	case "oreshnik":
		c.Rect(2, H-12, W-2, H-5, C(70, 80, 66))
		for x := 8; x < W-8; x += 12 {
			wheel(c, x, H-5, 5)
		}
		c.Rect(8, H-16, W-14, H-10, C(54, 62, 50))
		c.Rect(W-14, H-22, W-2, H-8, C(90, 100, 84))
	case "zu23":
		wheel(c, 8, H-5, 4)
		wheel(c, W-8, H-5, 4)
		c.Rect(6, H-12, W-6, H-6, oliveC)
		c.Rect(W/2-6, H-18, W/2+6, H-10, Shade(oliveC, 0.85))
	case "pantsir":
		truck(c, W, H, C(96, 110, 84))
		c.Rect(6, H-24, 20, H-14, C(70, 78, 64))
		c.Disc(14, H-28, 6, C(160, 170, 176)) // radar
		c.Rect(26, H-26, 38, H-14, C(60, 66, 56))
		for i := 0; i < 3; i++ {
			c.Rect(28+i*4, H-30, 30+i*4, H-24, C(220, 220, 224))
		}
	case "s400":
		truck(c, W, H, C(88, 100, 78))
		c.Rect(8, H-30, W-24, H-14, C(64, 72, 58))
		for i := 0; i < 4; i++ {
			c.Rect(10+i*7, H-36, 14+i*7, H-30, C(225, 225, 230))
		}
		c.Rect(W-22, H-34, W-14, H-20, C(60, 66, 56))
		c.Disc(W-18, H-38, 5, C(170, 180, 186))
	case "reb":
		c.Rect(W/2-6, H-10, W/2+6, H-2, steelC)
		c.Rect(W/2-1, 4, W/2+1, H-10, C(180, 186, 196))
		c.Disc(W/2, 4, 3, C(255, 90, 70))
		c.Line(W/2, 10, W/2-9, 6, C(180, 186, 196))
		c.Line(W/2, 10, W/2+9, 6, C(180, 186, 196))
	case "oil":
		c.Line(4, H-1, W/2, 6, C(60, 60, 66))
		c.Line(W-4, H-1, W/2, 6, C(60, 60, 66))
		c.Line(10, H-14, W-10, H-14, C(60, 60, 66))
		c.Rect(W/2-1, 0, W/2+1, H, C(40, 40, 46))
		c.Rect(2, H-8, W-2, H, C(36, 30, 26))
	case "farm":
		c.Rect(2, 12, W-2, H, C(150, 60, 50))
		c.Rect(2, 8, W-2, 14, C(110, 44, 40))
		c.Rect(W/2-6, H-14, W/2+6, H, C(250, 240, 230))
		c.Rect(8, 18, 16, 24, C(255, 214, 140))
		c.Disc(W-14, 22, 3, pigSkin)
	default:
		c.Rect(0, 0, W, H, C(200, 80, 200))
	}
	c.Outline(outlineC)
	return c.Img
}

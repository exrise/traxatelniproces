package gfx

import (
	"image"
	"image/color"
)

var (
	pigSkin    = C(244, 170, 176)
	pigSkinDk  = C(214, 126, 138)
	pigSnout   = C(255, 196, 198)
	pigNostril = C(150, 70, 84)
	pigDark    = C(40, 30, 36)
	camoA      = C(86, 98, 62)
	camoB      = C(62, 74, 46)
	boot       = C(48, 38, 34)
)

// PigW/PigH are the sprite canvas dimensions; feet are at (PigW/2, PigH-1).
const (
	PigW = 24
	PigH = 24
)

// PigFrames is the number of animation frames per pig sprite.
const PigFrames = 4

// hat styles per unit type
func hatFor(unit string) string {
	switch unit {
	case "sniper":
		return "boonie"
	case "rpg":
		return "helmet"
	case "mortar":
		return "helmet"
	case "dronner":
		return "goggles"
	case "spotter":
		return "beret"
	case "engineer":
		return "hardhat"
	case "shotgun":
		return "bandana"
	default:
		return "cap"
	}
}

// PigSprite draws a pig facing right. frame 0 = idle, 1..3 = walk cycle.
func PigSprite(team int, unit string, frame int) *image.RGBA {
	c := NewCanvas(PigW, PigH)
	tc := TeamColors[team%len(TeamColors)]
	cx := PigW / 2
	bob := 0
	l1, l2 := 0, 0
	switch frame {
	case 1:
		l1, l2, bob = -2, 2, 1
	case 3:
		l1, l2, bob = 2, -2, 1
	case 2:
		bob = 0
	}
	// tail
	c.Px(cx-6, 14+bob, pigSkinDk)
	c.Px(cx-7, 13+bob, pigSkinDk)
	c.Px(cx-7, 12+bob, pigSkin)
	// legs
	c.Rect(cx-4+l1, 18, cx-1+l1, PigH-2, camoB)
	c.Rect(cx-4+l1, PigH-2, cx-0+l1, PigH, boot)
	c.Rect(cx+1+l2, 18, cx+4+l2, PigH-2, camoB)
	c.Rect(cx+1+l2, PigH-2, cx+5+l2, PigH, boot)
	// torso (vest in camo with a team stripe)
	c.Rect(cx-5, 10+bob, cx+5, 19, camoA)
	c.Rect(cx-5, 14+bob, cx+5, 16+bob, camoB)
	c.Rect(cx-5, 10+bob, cx+5, 11+bob, tc) // shoulder strap
	c.Rect(cx-1, 10+bob, cx+1, 19, Shade(camoA, 0.8))
	// arm (skin) toward the front
	c.Rect(cx+3, 12+bob, cx+7, 15+bob, pigSkin)
	// head
	c.Ellipse(cx+1, 6+bob, 6, 5, pigSkin)
	// ear
	c.Rect(cx-3, 0+bob, cx+0, 3+bob, pigSkin)
	c.Px(cx-2, 1+bob, pigSkinDk)
	c.Px(cx-2, 2+bob, pigSkinDk)
	// snout
	c.Rect(cx+4, 5+bob, cx+9, 9+bob, pigSnout)
	c.Px(cx+6, 6+bob, pigNostril)
	c.Px(cx+8, 6+bob, pigNostril)
	// eye
	c.Rect(cx+2, 4+bob, cx+4, 6+bob, C(255, 255, 255))
	c.Px(cx+3, 5+bob, pigDark)
	// headgear
	switch hatFor(unit) {
	case "cap":
		c.Rect(cx-4, 1+bob, cx+4, 3+bob, tc)
		c.Rect(cx+3, 2+bob, cx+8, 3+bob, Shade(tc, 0.75))
	case "helmet":
		c.Ellipse(cx, 2+bob, 5, 3, Shade(camoA, 1.05))
		c.Rect(cx-5, 2+bob, cx+5, 4+bob, camoB)
		c.Rect(cx-5, 3+bob, cx+5, 4+bob, tc)
	case "boonie":
		c.Rect(cx-6, 3+bob, cx+7, 4+bob, camoA)
		c.Rect(cx-3, 0+bob, cx+3, 3+bob, camoA)
		c.Px(cx, 1+bob, tc)
	case "goggles":
		c.Rect(cx-4, 1+bob, cx+4, 3+bob, pigDark)
		c.Rect(cx+1, 4+bob, cx+5, 6+bob, C(90, 220, 255))
		c.Px(cx+2, 5+bob, C(255, 255, 255))
	case "beret":
		c.Ellipse(cx, 1+bob, 5, 2, tc)
		c.Px(cx+3, 0+bob, Shade(tc, 0.7))
	case "hardhat":
		c.Ellipse(cx, 2+bob, 5, 3, C(255, 196, 40))
		c.Rect(cx-6, 3+bob, cx+7, 4+bob, C(236, 170, 20))
	case "bandana":
		c.Rect(cx-5, 2+bob, cx+5, 4+bob, tc)
		c.Rect(cx-7, 3+bob, cx-4, 6+bob, tc)
	}
	c.Outline(pigDark)
	return c.Img
}

// TombSprite is what a fallen pig leaves behind.
func TombSprite() *image.RGBA {
	c := NewCanvas(14, 16)
	c.Rect(3, 4, 11, 16, C(150, 150, 158))
	c.Ellipse(7, 4, 4, 3, C(150, 150, 158))
	c.Rect(6, 6, 8, 12, C(90, 90, 98))
	c.Rect(4, 8, 10, 10, C(90, 90, 98))
	c.Outline(pigDark)
	return c.Img
}

// Tint multiplies an image by a color (used for hurt flashes).
func Tint(src *image.RGBA, col color.RGBA, amount float64) *image.RGBA {
	out := image.NewRGBA(src.Rect)
	for i := 0; i < len(src.Pix); i += 4 {
		a := src.Pix[i+3]
		if a == 0 {
			continue
		}
		out.Pix[i] = uint8(float64(src.Pix[i])*(1-amount) + float64(col.R)*amount)
		out.Pix[i+1] = uint8(float64(src.Pix[i+1])*(1-amount) + float64(col.G)*amount)
		out.Pix[i+2] = uint8(float64(src.Pix[i+2])*(1-amount) + float64(col.B)*amount)
		out.Pix[i+3] = a
	}
	return out
}

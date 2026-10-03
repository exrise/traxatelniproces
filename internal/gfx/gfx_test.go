package gfx

import (
	"image"
	"image/png"
	"os"
	"testing"

	"svinovoyna/internal/balance"
)

// TestDumpSprites writes a contact sheet when SVINO_DUMP is set, for eyeballing.
func TestDumpSprites(t *testing.T) {
	dir := os.Getenv("SVINO_DUMP")
	if dir == "" {
		t.Skip("set SVINO_DUMP=dir to dump sprites")
	}
	cfg := balance.Default()
	sheet := image.NewRGBA(image.Rect(0, 0, 1200, 420))
	x, y := 4, 4
	blit := func(img *image.RGBA, scale int) {
		b := img.Bounds()
		if x+b.Dx()*scale > 1190 {
			x = 4
			y += 100
		}
		for yy := 0; yy < b.Dy()*scale; yy++ {
			for xx := 0; xx < b.Dx()*scale; xx++ {
				sheet.Set(x+xx, y+yy, img.At(xx/scale, yy/scale))
			}
		}
		x += b.Dx()*scale + 6
	}
	for i := range sheet.Pix {
		sheet.Pix[i] = 90
	}
	for _, u := range cfg.Units {
		blit(PigSprite(0, u.ID, 0), 3)
		blit(PigSprite(1, u.ID, 1), 3)
	}
	x, y = 4, 110
	for _, s := range cfg.Structs {
		blit(StructSprite(s.ID, s.W, s.H), 1)
	}
	f, _ := os.Create(dir + "/sprites.png")
	defer f.Close()
	png.Encode(f, sheet)
}

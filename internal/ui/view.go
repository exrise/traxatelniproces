package ui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"svinovoyna/internal/gfx"
	"svinovoyna/internal/sim"
)

// Camera maps world pixels to the screen.
type Camera struct {
	X, Y, Zoom float64
	WorldW     float64
	ShakeX     float64
	ShakeY     float64
}

// ToScreen converts world to screen coordinates.
func (c *Camera) ToScreen(p sim.Vec) (float64, float64) {
	return (p.X-c.X)*c.Zoom + c.ShakeX, (p.Y-c.Y)*c.Zoom + c.ShakeY
}

// ToWorld converts screen to world coordinates.
func (c *Camera) ToWorld(x, y float64) sim.Vec {
	return sim.Vec{X: (x-c.ShakeX)/c.Zoom + c.X, Y: (y-c.ShakeY)/c.Zoom + c.Y}
}

// Clamp keeps the camera inside the playfield.
func (c *Camera) Clamp() {
	c.Zoom = math.Max(0.55, math.Min(2.2, c.Zoom))
	vw, vh := ScreenW/c.Zoom, ScreenH/c.Zoom
	c.X = math.Max(-60, math.Min(c.WorldW-vw+60, c.X))
	c.Y = math.Max(-420, math.Min(sim.MapH-vh+120, c.Y))
}

// CenterOn moves smoothly so p is in the middle.
func (c *Camera) CenterOn(p sim.Vec, k float64) {
	tx := p.X - ScreenW/c.Zoom/2
	ty := p.Y - ScreenH/c.Zoom/2
	c.X += (tx - c.X) * k
	c.Y += (ty - c.Y) * k
}

// ZoomAt changes zoom keeping the world point under (sx,sy) fixed.
func (c *Camera) ZoomAt(f, sx, sy float64) {
	before := c.ToWorld(sx, sy)
	c.Zoom *= f
	c.Zoom = math.Max(0.55, math.Min(2.2, c.Zoom))
	after := c.ToWorld(sx, sy)
	c.X += before.X - after.X
	c.Y += before.Y - after.Y
}

type cloud struct {
	x, y, speed float64
	img         int
}

// View renders a world and owns terrain textures and effects.
type View struct {
	a                  *App
	W                  *sim.World
	Cam                Camera
	Fx                 Effects
	Only               int // when >=0 only that player's units/structures are shown (build fog)
	UnitWeapon         map[int]string
	SelUnit, SelStruct int

	terr     *ebiten.Image
	terrRGBA *image.RGBA
	sky      *ebiten.Image
	layers   []*ebiten.Image
	clouds   []cloud
	clock    float64
	Time     float64
}

// NewView prepares textures for a world.
func NewView(a *App, w *sim.World) *View {
	v := &View{a: a, W: w, Only: -1, UnitWeapon: map[int]string{}, SelUnit: -1, SelStruct: -1}
	v.Cam = Camera{Zoom: 1.2, WorldW: float64(w.Terr.W)}
	t := w.Terr
	v.terrRGBA = image.NewRGBA(image.Rect(0, 0, t.W, t.H))
	gfx.PaintTerrain(v.terrRGBA, t.Mask, t.W, t.H, v.terrRGBA.Rect)
	v.terr = ebiten.NewImageFromImage(v.terrRGBA)
	t.Dirty = nil
	v.sky = a.Img("sky", func() *image.RGBA { return gfx.Sky(ScreenW, ScreenH) })
	v.layers = []*ebiten.Image{
		a.Img("mtn_far", func() *image.RGBA { return gfx.MountainLayer(1600, 420, 1, gfx.C(150, 176, 210), 170, 300) }),
		a.Img("mtn_mid", func() *image.RGBA { return gfx.MountainLayer(1600, 420, 4, gfx.C(112, 146, 150), 130, 330) }),
		a.Img("mtn_near", func() *image.RGBA { return gfx.MountainLayer(1600, 420, 9, gfx.C(76, 116, 92), 90, 360) }),
	}
	for i := 0; i < 9; i++ {
		v.clouds = append(v.clouds, cloud{x: float64(i) * float64(t.W) / 9, y: 20 + float64(i*47%260), speed: 6 + float64(i%4)*3, img: i % 3})
	}
	return v
}

// syncTerrain uploads changed terrain rectangles to the GPU image.
func (v *View) syncTerrain() {
	t := v.W.Terr
	for _, r := range t.Dirty {
		rect := image.Rect(r[0], r[1], r[2], r[3]).Intersect(v.terrRGBA.Rect)
		if rect.Empty() {
			continue
		}
		gfx.PaintTerrain(v.terrRGBA, t.Mask, t.W, t.H, rect)
		pix := make([]byte, rect.Dx()*rect.Dy()*4)
		for y := 0; y < rect.Dy(); y++ {
			src := v.terrRGBA.PixOffset(rect.Min.X, rect.Min.Y+y)
			copy(pix[y*rect.Dx()*4:(y+1)*rect.Dx()*4], v.terrRGBA.Pix[src:src+rect.Dx()*4])
		}
		v.terr.SubImage(rect).(*ebiten.Image).WritePixels(pix)
	}
	t.Dirty = t.Dirty[:0]
}

// Tick consumes world events and advances effects; call once per sim tick.
func (v *View) Tick() {
	v.clock += sim.Dt
	v.syncTerrain()
	for _, e := range v.W.DrainEvents() {
		if v.Only >= 0 && (e.Type == sim.EvTracer || e.Type == sim.EvShot) {
			continue
		}
		v.Fx.Handle(e, v.W, v.a.Play)
	}
	for _, p := range v.W.Projs {
		if p.Alive {
			v.Fx.Trail(p)
		}
	}
	v.Fx.Update(sim.Dt)
	v.Cam.ShakeX = (math.Sin(v.clock * 90)) * v.Fx.Shake
	v.Cam.ShakeY = (math.Cos(v.clock * 83)) * v.Fx.Shake
}

func (v *View) visible(owner int) bool { return v.Only < 0 || v.Only == owner }

// Draw renders the whole scene.
func (v *View) Draw(dst *ebiten.Image) {
	a := v.a
	cam := &v.Cam
	dst.DrawImage(v.sky, nil)
	// parallax mountains
	for i, l := range v.layers {
		f := 0.08 + 0.1*float64(i)
		ox := -math.Mod(cam.X*f*cam.Zoom, 1600)
		if ox > 0 {
			ox -= 1600
		}
		oy := 120 + float64(i)*70 - cam.Y*0.12*cam.Zoom
		for x := ox; x < ScreenW; x += 1600 {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(x, oy)
			dst.DrawImage(l, op)
		}
	}
	// clouds
	for _, c := range v.clouds {
		cx := math.Mod(c.x+v.clock*c.speed, cam.WorldW+400) - 200
		sx, sy := (cx-cam.X*0.6)*cam.Zoom, (c.y-cam.Y*0.3)*cam.Zoom*0.8
		img := a.Img("cloud"+string(rune('0'+c.img)), func() *image.RGBA { return gfx.Cloud(uint32(c.img + 3)) })
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(cam.Zoom, cam.Zoom)
		op.GeoM.Translate(sx, sy)
		op.ColorScale.ScaleAlpha(0.85)
		dst.DrawImage(img, op)
	}
	// terrain
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(cam.Zoom, cam.Zoom)
	sx, sy := cam.ToScreen(sim.Vec{})
	op.GeoM.Translate(sx, sy)
	dst.DrawImage(v.terr, op)

	// capture points
	for i, p := range v.W.Points {
		v.drawPoint(dst, p, i)
	}
	// structures
	for _, s := range v.W.Structs {
		if s.Alive && v.visible(s.Owner) {
			v.drawStruct(dst, s)
		}
	}
	// mines
	for _, p := range v.W.Projs {
		if p.Alive && p.Kind == sim.PMine && (v.Only < 0) {
			x, y := cam.ToScreen(p.Pos)
			a.Rect(dst, x-5*cam.Zoom, y-3*cam.Zoom, 10*cam.Zoom, 3*cam.Zoom, color.RGBA{60, 64, 50, 255})
			if int(v.clock*3)%2 == 0 {
				a.Rect(dst, x-1*cam.Zoom, y-5*cam.Zoom, 2*cam.Zoom, 2*cam.Zoom, color.RGBA{255, 60, 40, 255})
			}
		}
	}
	// units
	for _, u := range v.W.Units {
		if u.Alive && v.visible(u.Owner) {
			v.drawUnit(dst, u)
		}
	}
	// projectiles
	if v.Only < 0 {
		for _, p := range v.W.Projs {
			if p.Alive && p.Kind != sim.PMine {
				v.drawProj(dst, p)
			}
		}
	}
	// water
	v.drawWater(dst)
	v.Fx.Draw(a, dst, cam)
}

func (v *View) drawWater(dst *ebiten.Image) {
	a := v.a
	cam := &v.Cam
	_, wy := cam.ToScreen(sim.Vec{X: 0, Y: sim.WaterY})
	if wy < ScreenH {
		a.Rect(dst, 0, wy, ScreenW, ScreenH-wy+4, color.RGBA{36, 96, 170, 150})
		for k := 0; k < 3; k++ {
			prevX, prevY := 0.0, 0.0
			for x := 0.0; x <= ScreenW+16; x += 16 {
				wx := x/cam.Zoom + cam.X
				y := wy + float64(k)*7*cam.Zoom + math.Sin(wx*0.03+v.clock*2+float64(k))*2.5*cam.Zoom
				if x > 0 {
					a.Line(dst, prevX, prevY, x, y, 2, color.RGBA{170, 214, 250, uint8(160 - k*40)})
				}
				prevX, prevY = x, y
			}
		}
	}
}

func (v *View) drawPoint(dst *ebiten.Image, p *sim.CapturePoint, idx int) {
	a := v.a
	cam := &v.Cam
	x, y := cam.ToScreen(p.Pos)
	z := cam.Zoom
	col := color.RGBA{200, 200, 205, 255}
	if p.Owner >= 0 {
		col = gfx.TeamColors[v.W.Players[p.Owner].Color%4]
	}
	a.Rect(dst, x-1.5*z, y-40*z, 3*z, 40*z, color.RGBA{60, 60, 66, 255})
	wave := math.Sin(v.clock*5) * 2 * z
	a.Rect(dst, x+1.5*z, y-40*z, 20*z, 11*z+wave*0.3, col)
	a.Border(dst, x+1.5*z, y-40*z, 20*z, 11*z+wave*0.3, 1, color.RGBA{20, 20, 24, 255})
	a.Circle(dst, x, y-2*z, 4*z, color.RGBA{50, 50, 56, 255})
	// capture radius hint
	a.Border(dst, x-90*z, y-90*z, 180*z, 180*z, 1, color.RGBA{255, 255, 255, 36})
}

func (v *View) drawStruct(dst *ebiten.Image, s *sim.Struct) {
	a := v.a
	cam := &v.Cam
	d := v.W.Cfg.S(s.Def)
	img := a.Img("s_"+s.Def, func() *image.RGBA { return gfx.StructSprite(s.Def, d.W, d.H) })
	x0, y0 := float64(s.CX*sim.Cell), float64(s.CY*sim.Cell)
	sx, sy := cam.ToScreen(sim.Vec{X: x0, Y: y0})
	z := cam.Zoom
	pl := v.W.Players[s.Owner]
	tc := gfx.TeamColors[pl.Color%4]

	// rotating barrel behind the body
	if b, ok := gfx.BarrelFor(s.Def); ok {
		px, py := sx+float64(d.W*sim.Cell)/2*z, sy+float64(b.PivotY)*z
		dir := sim.Dir(s.Aim)
		a.Line(dst, px, py, px+dir.X*float64(b.Len)*z, py+dir.Y*float64(b.Len)*z, float64(b.Thick)*z, b.Col)
		a.Line(dst, px, py, px+dir.X*float64(b.Len)*z, py+dir.Y*float64(b.Len)*z, math.Max(1, float64(b.Thick)*z-2), gfx.Shade(b.Col, 1.35))
		a.Circle(dst, px, py, float64(b.Thick)*0.8*z, gfx.Shade(b.Col, 0.8))
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(z, z)
	op.GeoM.Translate(sx, sy)
	if s.Hurt > 0 {
		op.ColorScale.Scale(1.7, 1.7, 1.7, 1)
	}
	dst.DrawImage(img, op)

	w, h := float64(d.W*sim.Cell)*z, float64(d.H*sim.Cell)*z
	// team marking
	a.Rect(dst, sx, sy-2*z, w, 2*z, color.RGBA{tc.R, tc.G, tc.B, 220})
	if s.Def == "hq" {
		a.Rect(dst, sx+w/2+1*z, sy, 11*z, 6*z, tc)
	}
	// damage look
	frac := s.HP / s.MaxHP
	if frac < 0.75 {
		n := int((1 - frac) * 7)
		for i := 0; i < n; i++ {
			rx := sx + w*(0.1+0.8*float64((i*37)%10)/10)
			ry := sy + h*(0.1+0.8*float64((i*53)%10)/10)
			a.Line(dst, rx, ry, rx+5*z, ry+8*z, 1.2, color.RGBA{20, 18, 20, 220})
		}
	}
	if frac < 0.4 && int(v.clock*8+float64(s.ID))%3 == 0 {
		v.Fx.add(particle{P: sim.Vec{X: x0 + float64(d.W*sim.Cell)*rnd(0.2, 0.8), Y: y0 + float64(d.H*sim.Cell)*0.3}, V: sim.Vec{Y: -25}, Life: 0.9, Size: 4,
			Col: color.RGBA{70, 70, 70, 200}, Kind: pSmoke})
	}
	// hp bar for damaged / important things
	if frac < 0.999 || s.Def == "hq" {
		bw := w
		a.Rect(dst, sx, sy-8*z, bw, 4, color.RGBA{0, 0, 0, 170})
		hc := color.RGBA{110, 220, 110, 255}
		if frac < 0.5 {
			hc = color.RGBA{240, 200, 70, 255}
		}
		if frac < 0.25 {
			hc = color.RGBA{240, 80, 70, 255}
		}
		a.Rect(dst, sx, sy-8*z, bw*frac, 4, hc)
	}
	// ammo pips for weapon structures
	if d.Kind == "weapon" && v.W.Phase == sim.PhaseBattle && v.Only < 0 {
		wd := v.W.Cfg.W(d.Weapon)
		for i := 0; i < wd.Ammo && i < 8; i++ {
			c := color.RGBA{255, 220, 90, 255}
			if i >= s.Ammo {
				c = color.RGBA{70, 70, 74, 255}
			}
			a.Rect(dst, sx+float64(i)*5*z, sy+h+2*z, 4*z, 3*z, c)
		}
	}
	if v.SelStruct == s.ID {
		a.Border(dst, sx-2, sy-2, w+4, h+4, 2, color.RGBA{255, 240, 120, uint8(150 + 100*math.Sin(v.clock*8))})
	}
}

func (v *View) pigImage(owner int, def string, frame int, hurt bool) *ebiten.Image {
	col := v.W.Players[owner].Color % 4
	key := "pig_" + string(rune('0'+col)) + "_" + def + "_" + string(rune('0'+frame))
	if hurt {
		key += "h"
	}
	return v.a.Img(key, func() *image.RGBA {
		im := gfx.PigSprite(col, def, frame)
		if hurt {
			return gfx.Tint(im, color.RGBA{255, 255, 255, 255}, 0.7)
		}
		return im
	})
}

type wvis struct {
	l, th float64
	col   color.RGBA
	tip   color.RGBA
}

var weaponVis = map[string]wvis{
	"ak":      {16, 2.5, color.RGBA{40, 40, 44, 255}, color.RGBA{120, 80, 40, 255}},
	"saiga":   {14, 3, color.RGBA{50, 44, 40, 255}, color.RGBA{150, 100, 50, 255}},
	"svd":     {24, 2.5, color.RGBA{50, 48, 44, 255}, color.RGBA{150, 120, 60, 255}},
	"rpg":     {22, 4, color.RGBA{70, 80, 60, 255}, color.RGBA{190, 70, 50, 255}},
	"mortar":  {16, 5, color.RGBA{54, 60, 56, 255}, color.RGBA{54, 60, 56, 255}},
	"makarov": {7, 2.5, color.RGBA{40, 40, 44, 255}, color.RGBA{40, 40, 44, 255}},
	"grenade": {5, 4, color.RGBA{86, 110, 66, 255}, color.RGBA{86, 110, 66, 255}},
	"fpv":     {9, 5, color.RGBA{50, 50, 56, 255}, color.RGBA{255, 90, 70, 255}},
	"tm62":    {6, 4, color.RGBA{70, 74, 60, 255}, color.RGBA{70, 74, 60, 255}},
	"repair":  {12, 3, color.RGBA{160, 160, 170, 255}, color.RGBA{240, 190, 60, 255}},
}

func (v *View) drawUnit(dst *ebiten.Image, u *sim.Unit) {
	a := v.a
	cam := &v.Cam
	z := cam.Zoom
	frame := 0
	if u.Walk != 0 && u.Ground {
		frame = 1 + int(u.Anim)%3
	}
	img := v.pigImage(u.Owner, u.Def, frame, u.Hurt > 0)
	sx, sy := cam.ToScreen(u.Pos)
	// weapon
	wid := v.UnitWeapon[u.ID]
	if wid == "" {
		wid = v.W.Cfg.U(u.Def).Weapons[0]
	}
	if wv, ok := weaponVis[wid]; ok {
		shoulder := sim.Vec{X: u.Pos.X + float64(u.Face)*2, Y: u.Pos.Y - 11}
		ang := u.Aim
		dir := sim.Dir(ang)
		px, py := cam.ToScreen(shoulder)
		a.Line(dst, px, py, px+dir.X*wv.l*z, py+dir.Y*wv.l*z, wv.th*z, wv.col)
		a.Line(dst, px+dir.X*wv.l*0.55*z, py+dir.Y*wv.l*0.55*z, px+dir.X*wv.l*z, py+dir.Y*wv.l*z, wv.th*z*0.7, wv.tip)
	}
	op := &ebiten.DrawImageOptions{}
	if u.Face < 0 {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(gfx.PigW/2, 0)
	} else {
		op.GeoM.Translate(-gfx.PigW/2, 0)
	}
	if u.Face < 0 {
		op.GeoM.Translate(-gfx.PigW/2, 0)
	}
	op.GeoM.Translate(0, -gfx.PigH)
	op.GeoM.Scale(z, z)
	op.GeoM.Translate(sx, sy+1*z)
	dst.DrawImage(img, op)

	// hp bar + name
	frac := u.HP / u.MaxHP
	bw := 26 * z
	bx, by := sx-bw/2, sy-gfx.PigH*z-9*z
	a.Rect(dst, bx-1, by-1, bw+2, 5, color.RGBA{0, 0, 0, 190})
	hc := gfx.TeamColors[v.W.Players[u.Owner].Color%4]
	a.Rect(dst, bx, by, bw*frac, 3, hc)
	if v.SelUnit == u.ID {
		bob := math.Sin(v.clock*7) * 3 * z
		ax, ay := sx, by-14*z+bob
		a.Circle(dst, ax, ay, 5*z+1, color.RGBA{255, 240, 100, 255})
		a.Line(dst, ax-5*z, ay, ax, ay+8*z, 3, color.RGBA{255, 240, 100, 255})
		a.Line(dst, ax+5*z, ay, ax, ay+8*z, 3, color.RGBA{255, 240, 100, 255})
	}
}

func (v *View) drawProj(dst *ebiten.Image, p *sim.Proj) {
	a := v.a
	cam := &v.Cam
	z := cam.Zoom
	x, y := cam.ToScreen(p.Pos)
	ang := math.Atan2(p.Vel.Y, p.Vel.X)
	dir := sim.Dir(ang)
	switch p.Kind {
	case sim.PShell:
		wd := v.W.Cfg.W(p.Weapon)
		col := color.RGBA{40, 40, 44, 255}
		r := 3.0
		if wd != nil && wd.Fuse > 0 {
			col, r = color.RGBA{80, 110, 60, 255}, 3.5
		}
		a.Circle(dst, x, y, r*z, col)
		a.Circle(dst, x-0.8*z, y-0.8*z, r*z*0.4, color.RGBA{200, 200, 210, 255})
	case sim.PRocket:
		a.Line(dst, x-dir.X*10*z, y-dir.Y*10*z, x+dir.X*4*z, y+dir.Y*4*z, 3*z, color.RGBA{200, 205, 210, 255})
		a.Circle(dst, x-dir.X*11*z, y-dir.Y*11*z, 3*z, color.RGBA{255, 170, 60, 230})
	case sim.PBallis:
		if p.Phase == 0 {
			a.Line(dst, x, y+22*z, x, y-22*z, 6*z, color.RGBA{215, 218, 224, 255})
			a.Circle(dst, x, y+26*z, 6*z, color.RGBA{255, 190, 80, 240})
		} else {
			a.Line(dst, x-dir.X*24*z, y-dir.Y*24*z, x+dir.X*6*z, y+dir.Y*6*z, 5*z, color.RGBA{215, 218, 224, 255})
			a.Circle(dst, x-dir.X*25*z, y-dir.Y*25*z, 5*z, color.RGBA{255, 190, 80, 240})
		}
	case sim.PWarhead:
		a.Circle(dst, x, y, 6*z, color.RGBA{255, 150, 50, 160})
		a.Line(dst, x-dir.X*18*z, y-dir.Y*18*z, x, y, 4*z, color.RGBA{255, 220, 140, 255})
		a.Circle(dst, x, y, 3*z, color.RGBA{255, 255, 220, 255})
	case sim.PBomb:
		a.Circle(dst, x, y, 5*z, color.RGBA{60, 64, 60, 255})
		a.Line(dst, x-dir.X*6*z, y-dir.Y*6*z, x-dir.X*12*z, y-dir.Y*12*z, 5*z, color.RGBA{90, 94, 90, 255})
	case sim.PPlane:
		d := 1.0
		if p.Vel.X < 0 {
			d = -1
		}
		body := color.RGBA{150, 156, 166, 255}
		a.Line(dst, x-30*z*d, y, x+30*z*d, y, 8*z, body)
		a.Line(dst, x-4*z*d, y, x-18*z*d, y+14*z, 5*z, gfx.Shade(body, 0.8))
		a.Line(dst, x-4*z*d, y, x-18*z*d, y-14*z, 5*z, gfx.Shade(body, 0.8))
		a.Line(dst, x-26*z*d, y, x-32*z*d, y-10*z, 4*z, body)
		a.Circle(dst, x+22*z*d, y-1*z, 3*z, color.RGBA{120, 200, 240, 255})
		a.Circle(dst, x-32*z*d, y, 3*z, color.RGBA{255, 160, 60, 220})
	case sim.PDrone:
		if p.Phase == 1 { // guided missile (ПТУР)
			dd := sim.Dir(p.Heading)
			pr := sim.Vec{X: -dd.Y, Y: dd.X}
			a.Line(dst, x-dd.X*12*z, y-dd.Y*12*z, x+dd.X*10*z, y+dd.Y*10*z, 4*z, color.RGBA{214, 218, 224, 255})
			a.Line(dst, x+dd.X*6*z, y+dd.Y*6*z, x+dd.X*11*z, y+dd.Y*11*z, 4*z, color.RGBA{220, 70, 50, 255})
			a.Line(dst, x-dd.X*10*z+pr.X*5*z, y-dd.Y*10*z+pr.Y*5*z, x-dd.X*13*z, y-dd.Y*13*z, 2*z, color.RGBA{150, 156, 166, 255})
			a.Line(dst, x-dd.X*10*z-pr.X*5*z, y-dd.Y*10*z-pr.Y*5*z, x-dd.X*13*z, y-dd.Y*13*z, 2*z, color.RGBA{150, 156, 166, 255})
			a.Circle(dst, x-dd.X*15*z, y-dd.Y*15*z, (3+2*math.Sin(v.clock*50))*z, color.RGBA{255, 190, 70, 235})
			return
		}
		h := p.Heading
		dd := sim.Dir(h)
		a.Line(dst, x-dd.X*6*z, y-dd.Y*6*z, x+dd.X*6*z, y+dd.Y*6*z, 4*z, color.RGBA{40, 42, 48, 255})
		pr := dd.Mul(1)
		a.Line(dst, x-pr.Y*8*z, y+pr.X*8*z, x+pr.Y*8*z, y-pr.X*8*z, 2*z, color.RGBA{200, 200, 210, uint8(120 + 100*math.Sin(v.clock*60))})
		a.Circle(dst, x+dd.X*6*z, y+dd.Y*6*z, 2*z, color.RGBA{255, 80, 60, 255})
	case sim.PGeran:
		h := math.Atan2(p.Vel.Y, p.Vel.X)
		dd := sim.Dir(h)
		pr := sim.Vec{X: -dd.Y, Y: dd.X}
		a.Line(dst, x-dd.X*14*z, y-dd.Y*14*z, x+dd.X*14*z, y+dd.Y*14*z, 5*z, color.RGBA{214, 214, 220, 255})
		a.Line(dst, x-dd.X*10*z+pr.X*12*z, y-dd.Y*10*z+pr.Y*12*z, x+dd.X*4*z, y+dd.Y*4*z, 3*z, color.RGBA{190, 190, 198, 255})
		a.Line(dst, x-dd.X*10*z-pr.X*12*z, y-dd.Y*10*z-pr.Y*12*z, x+dd.X*4*z, y+dd.Y*4*z, 3*z, color.RGBA{190, 190, 198, 255})
	}
}

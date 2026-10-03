package ui

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"svinovoyna/internal/sim"
)

type pKind int

const (
	pSpark pKind = iota
	pSmoke
	pFlash
	pRing
	pText
	pDebris
	pLine
	pDrop
)

type particle struct {
	P, V sim.Vec
	Life float64
	Max  float64
	Size float64
	Col  color.RGBA
	Kind pKind
	Grav float64
	Text string
	End  sim.Vec
	Big  bool
}

// Effects holds all transient visual effects.
type Effects struct {
	ps    []particle
	Shake float64
}

func (f *Effects) add(p particle) {
	if p.Max == 0 {
		p.Max = p.Life
	}
	f.ps = append(f.ps, p)
	if len(f.ps) > 3500 {
		f.ps = f.ps[len(f.ps)-3000:]
	}
}

func rnd(a, b float64) float64 { return a + rand.Float64()*(b-a) }

// Handle spawns effects for a simulation event.
func (f *Effects) Handle(e sim.Event, w *sim.World, snd func(string, float64)) {
	switch e.Type {
	case sim.EvExplosion:
		r := e.R
		f.add(particle{P: e.Pos, Life: 0.18, Size: r * 1.1, Col: color.RGBA{255, 240, 180, 255}, Kind: pFlash})
		f.add(particle{P: e.Pos, Life: 0.45, Size: r * 1.5, Col: color.RGBA{255, 160, 60, 255}, Kind: pRing})
		n := int(r*0.8) + 6
		for i := 0; i < n; i++ {
			a := rnd(0, 2*math.Pi)
			s := rnd(60, 80+r*5)
			f.add(particle{P: e.Pos, V: sim.Vec{X: math.Cos(a) * s, Y: math.Sin(a)*s - 40}, Life: rnd(0.3, 0.9), Size: rnd(2, 4),
				Col: color.RGBA{255, uint8(rnd(150, 230)), uint8(rnd(40, 110)), 255}, Kind: pSpark, Grav: 300})
		}
		for i := 0; i < int(r/6)+3; i++ {
			a := rnd(0, 2*math.Pi)
			s := rnd(10, 40+r)
			g := uint8(rnd(50, 95))
			f.add(particle{P: e.Pos, V: sim.Vec{X: math.Cos(a) * s, Y: math.Sin(a)*s - 30}, Life: rnd(0.8, 1.8), Size: rnd(r*0.25, r*0.5),
				Col: color.RGBA{g, g, g, 200}, Kind: pSmoke})
		}
		for i := 0; i < int(r/8)+2; i++ {
			a := rnd(math.Pi, 2*math.Pi)
			s := rnd(100, 160+r*2)
			f.add(particle{P: e.Pos, V: sim.Vec{X: math.Cos(a) * s * 0.7, Y: math.Sin(a) * s}, Life: rnd(0.7, 1.4), Size: rnd(2, 4),
				Col: color.RGBA{112, 86, 56, 255}, Kind: pDebris, Grav: 700})
		}
		f.Shake = math.Max(f.Shake, math.Min(14, r*0.14))
		if snd != nil {
			if r > 55 {
				snd("boom_big", 1)
			} else {
				snd("boom", 1)
			}
		}
	case sim.EvTracer:
		col := color.RGBA{255, 240, 150, 255}
		f.add(particle{P: e.Pos, End: e.To, Life: 0.11, Col: col, Kind: pLine, Size: 1.5})
		if e.F > 0 {
			for i := 0; i < 3; i++ {
				f.add(particle{P: e.To, V: sim.Vec{X: rnd(-80, 80), Y: rnd(-120, -10)}, Life: rnd(0.15, 0.4), Size: 2,
					Col: color.RGBA{255, 220, 120, 255}, Kind: pSpark, Grav: 400})
			}
		}
	case sim.EvShot:
		f.add(particle{P: e.Pos, Life: 0.08, Size: 9, Col: color.RGBA{255, 230, 150, 255}, Kind: pFlash})
		for i := 0; i < 3; i++ {
			f.add(particle{P: e.Pos, V: sim.Vec{X: rnd(-20, 20), Y: rnd(-30, -5)}, Life: rnd(0.4, 0.9), Size: rnd(3, 6),
				Col: color.RGBA{190, 190, 190, 160}, Kind: pSmoke})
		}
		if snd != nil {
			snd("shot_"+e.Text, 1)
		}
	case sim.EvDamage:
		col := color.RGBA{255, 230, 120, 255}
		if e.F >= 40 {
			col = color.RGBA{255, 120, 80, 255}
		}
		f.add(particle{P: e.Pos, V: sim.Vec{X: rnd(-12, 12), Y: -42}, Life: 1.1, Text: fmt.Sprintf("%.0f", e.F), Col: col, Kind: pText, Size: 15})
	case sim.EvMoney:
		if e.B > 0 {
			f.add(particle{P: sim.Vec{X: e.Pos.X, Y: e.Pos.Y - 10}, V: sim.Vec{X: rnd(-8, 8), Y: -30}, Life: 1.4, Text: fmt.Sprintf("+$%d", e.B),
				Col: color.RGBA{120, 235, 130, 255}, Kind: pText, Size: 14})
		}
	case sim.EvUnitDied:
		for i := 0; i < 14; i++ {
			a := rnd(0, 2*math.Pi)
			f.add(particle{P: e.Pos, V: sim.Vec{X: math.Cos(a) * rnd(30, 120), Y: math.Sin(a)*rnd(30, 120) - 80}, Life: rnd(0.6, 1.2), Size: rnd(2, 4),
				Col: color.RGBA{250, 160, 170, 255}, Kind: pDebris, Grav: 500})
		}
		f.add(particle{P: e.Pos, V: sim.Vec{Y: -26}, Life: 1.4, Text: "R.I.P.", Col: color.RGBA{230, 230, 240, 255}, Kind: pText, Size: 15})
		if snd != nil {
			snd("squeal", 1)
		}
	case sim.EvStructDestroyed:
		for i := 0; i < 26; i++ {
			a := rnd(math.Pi, 2*math.Pi)
			f.add(particle{P: sim.Vec{X: e.Pos.X + rnd(-e.R, e.R), Y: e.Pos.Y + rnd(-e.R, e.R)*0.5}, V: sim.Vec{X: math.Cos(a) * rnd(40, 200), Y: math.Sin(a) * rnd(60, 240)},
				Life: rnd(0.8, 1.8), Size: rnd(3, 6), Col: color.RGBA{uint8(rnd(90, 150)), uint8(rnd(90, 140)), uint8(rnd(90, 130)), 255}, Kind: pDebris, Grav: 700})
		}
		f.Shake = math.Max(f.Shake, 7)
	case sim.EvSplash:
		for i := 0; i < 16; i++ {
			f.add(particle{P: e.Pos, V: sim.Vec{X: rnd(-90, 90), Y: rnd(-260, -60)}, Life: rnd(0.5, 1.0), Size: 3,
				Col: color.RGBA{170, 215, 250, 255}, Kind: pDrop, Grav: 700})
		}
		if snd != nil {
			snd("splash", 1)
		}
	case sim.EvIntercept:
		f.add(particle{P: e.Pos, Life: 0.25, Size: 20, Col: color.RGBA{255, 255, 255, 255}, Kind: pFlash})
		f.add(particle{P: e.To, End: e.Pos, Life: 0.18, Col: color.RGBA{255, 200, 100, 255}, Kind: pLine, Size: 2})
		if e.B == 0 {
			f.add(particle{P: e.Pos, Life: 0.5, Size: 36, Col: color.RGBA{255, 190, 90, 255}, Kind: pRing})
			f.add(particle{P: sim.Vec{X: e.Pos.X, Y: e.Pos.Y - 14}, V: sim.Vec{Y: -20}, Life: 1.2, Text: "СБИТ", Col: color.RGBA{255, 220, 120, 255}, Kind: pText, Size: 16})
			if snd != nil {
				snd("boom", 0.6)
			}
		} else {
			f.add(particle{P: sim.Vec{X: e.Pos.X, Y: e.Pos.Y - 14}, V: sim.Vec{Y: -20}, Life: 0.9, Text: "мимо", Col: color.RGBA{190, 200, 215, 255}, Kind: pText, Size: 13})
		}
	case sim.EvHeal:
		f.add(particle{P: e.Pos, V: sim.Vec{Y: -26}, Life: 1.0, Text: fmt.Sprintf("+%.0f", e.F), Col: color.RGBA{120, 235, 130, 255}, Kind: pText, Size: 14})
	case sim.EvBuild:
		for i := 0; i < 8; i++ {
			f.add(particle{P: e.Pos, V: sim.Vec{X: rnd(-60, 60), Y: rnd(-80, -10)}, Life: rnd(0.3, 0.6), Size: 2, Col: color.RGBA{230, 230, 200, 255}, Kind: pSpark, Grav: 300})
		}
		if snd != nil {
			snd("build", 1)
		}
	case sim.EvPlane:
		if snd != nil {
			snd("plane", 1)
		}
	case sim.EvJump:
		if snd != nil {
			snd("jump", 0.6)
		}
	}
}

// Trail leaves smoke behind a moving projectile.
func (f *Effects) Trail(p *sim.Proj) {
	switch p.Kind {
	case sim.PRocket, sim.PBallis, sim.PWarhead, sim.PGeran:
		col := color.RGBA{200, 200, 200, 150}
		sz := rnd(2, 4)
		if p.Kind == sim.PWarhead || (p.Kind == sim.PBallis && p.Phase == 1) {
			col = color.RGBA{255, 170, 70, 200}
			sz = rnd(3, 6)
		}
		f.add(particle{P: p.Pos, V: sim.Vec{X: rnd(-10, 10), Y: rnd(-10, 10)}, Life: rnd(0.4, 0.9), Size: sz, Col: col, Kind: pSmoke})
	case sim.PShell:
		if rand.Intn(2) == 0 {
			f.add(particle{P: p.Pos, Life: 0.3, Size: 1.5, Col: color.RGBA{220, 220, 220, 120}, Kind: pSmoke})
		}
	case sim.PDrone:
		if rand.Intn(3) == 0 {
			f.add(particle{P: p.Pos, Life: 0.3, Size: 1.5, Col: color.RGBA{200, 200, 200, 90}, Kind: pSmoke})
		}
	case sim.PPlane:
		f.add(particle{P: sim.Vec{X: p.Pos.X - math.Copysign(30, p.Vel.X), Y: p.Pos.Y}, Life: 1.2, Size: 4, Col: color.RGBA{255, 255, 255, 120}, Kind: pSmoke})
	}
}

// Update advances all particles.
func (f *Effects) Update(dt float64) {
	live := f.ps[:0]
	for i := range f.ps {
		p := f.ps[i]
		p.Life -= dt
		if p.Life <= 0 {
			continue
		}
		p.V.Y += p.Grav * dt
		p.P = p.P.Add(p.V.Mul(dt))
		if p.Kind == pSmoke {
			p.V = p.V.Mul(0.97)
			p.V.Y -= 14 * dt
		}
		live = append(live, p)
	}
	f.ps = live
	f.Shake *= 0.9
}

// Draw renders particles through the world->screen transform.
func (f *Effects) Draw(a *App, dst *ebiten.Image, cam *Camera) {
	for _, p := range f.ps {
		t := p.Life / p.Max
		x, y := cam.ToScreen(p.P)
		z := cam.Zoom
		switch p.Kind {
		case pSpark, pDebris:
			c := p.Col
			c.A = uint8(255 * math.Min(1, t*2))
			a.Rect(dst, x-p.Size*z/2, y-p.Size*z/2, p.Size*z, p.Size*z, c)
		case pDrop:
			c := p.Col
			c.A = uint8(255 * math.Min(1, t*2))
			a.Rect(dst, x-1.5*z, y-1.5*z, 3*z, 3*z, c)
		case pSmoke:
			c := p.Col
			c.A = uint8(float64(p.Col.A) * t * 0.8)
			r := p.Size * z * (1.6 - 0.6*t)
			a.Circle(dst, x, y, r, c)
		case pFlash:
			c := p.Col
			c.A = uint8(255 * t)
			a.Circle(dst, x, y, p.Size*z*(0.6+0.4*t), c)
		case pRing:
			c := p.Col
			c.A = uint8(220 * t)
			a.Circle(dst, x, y, p.Size*z*(1-t*0.9)*0.8, color.RGBA{c.R, c.G, c.B, uint8(60 * t)})
		case pLine:
			c := p.Col
			c.A = uint8(255 * t)
			x2, y2 := cam.ToScreen(p.End)
			a.Line(dst, x, y, x2, y2, p.Size*z, c)
		case pText:
			c := p.Col
			c.A = uint8(255 * math.Min(1, t*2.2))
			a.TextCenter(dst, p.Text, x, y-8, p.Size, c, true)
		}
	}
}

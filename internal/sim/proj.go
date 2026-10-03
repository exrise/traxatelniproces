package sim

import (
	"math"

	"svinovoyna/internal/balance"
)

func (w *World) spec(p *Proj, wd *balance.Weapon) ExplosionSpec {
	return ExplosionSpec{Radius: wd.Radius, Damage: wd.Damage * p.Dmg, Crater: wd.Crater, BlockMul: wd.BlockMul, Pierce: wd.Pierce, Owner: p.Owner}
}

func (w *World) detonate(p *Proj, at Vec) {
	if !p.Alive {
		return
	}
	p.Alive = false
	wd := w.Cfg.W(p.Weapon)
	if wd == nil {
		return
	}
	w.curWeapon = wd.ID
	w.Explode(at, w.spec(p, wd))
	w.curWeapon = ""
}

func (w *World) cancelSpawns(parent int) {
	out := w.Spawns[:0]
	for _, s := range w.Spawns {
		if s.Parent != parent {
			out = append(out, s)
		}
	}
	w.Spawns = out
}

// projBlocked tells whether a projectile at its position has hit something.
func (w *World) projBlocked(p *Proj) bool {
	x, y := p.Pos.X, p.Pos.Y
	if w.Terr.Solid(int(math.Floor(x)), int(math.Floor(y))) {
		return true
	}
	if s := w.StructAtPx(x, y); s != nil && !(ownGhost(p) && s.Owner == p.Owner) {
		return true
	}
	for _, u := range w.Units {
		if !u.Alive || (ownGhost(p) && u.Owner == p.Owner) {
			continue
		}
		if math.Abs(x-u.Pos.X) <= UnitW/2+2 && y >= u.Pos.Y-UnitH-2 && y <= u.Pos.Y+1 {
			return true
		}
	}
	return false
}

// advance moves a projectile along its velocity in small sub-steps.
// It returns true when it hit something (position is at the hit).
func (w *World) advance(p *Proj, bounce bool) bool {
	n := int(math.Ceil(p.Vel.Len() * Dt / 3))
	if n < 1 {
		n = 1
	}
	step := p.Vel.Mul(Dt / float64(n))
	for i := 0; i < n; i++ {
		prev := p.Pos
		p.Pos = p.Pos.Add(step)
		if p.Pos.X < -400 || p.Pos.X > float64(w.Terr.W)+400 || p.Pos.Y > MapH+300 {
			p.Alive = false
			return false
		}
		if p.Pos.Y > WaterY {
			w.emit(Event{Type: EvSplash, Pos: Vec{p.Pos.X, WaterY}})
			p.Alive = false
			return false
		}
		if p.Pos.Y >= 0 && w.projBlocked(p) {
			if bounce {
				n := w.Terr.Normal(p.Pos.X, p.Pos.Y)
				if !w.Terr.Solid(int(p.Pos.X), int(p.Pos.Y)) {
					// struct: choose an axis normal from the previous position
					if w.StructAtPx(prev.X, p.Pos.Y) == nil {
						n = Vec{math.Copysign(1, -step.X), 0}
					} else {
						n = Vec{0, math.Copysign(1, -step.Y)}
					}
				}
				dot := p.Vel.X*n.X + p.Vel.Y*n.Y
				p.Vel = p.Vel.Sub(n.Mul(2 * dot)).Mul(0.45)
				p.Pos = prev
				if p.Vel.Len() < 30 {
					p.Vel = Vec{}
				}
				return false
			}
			return true
		}
	}
	return false
}

func angDiff(a, b float64) float64 {
	d := math.Mod(b-a, 2*math.Pi)
	if d > math.Pi {
		d -= 2 * math.Pi
	} else if d < -math.Pi {
		d += 2 * math.Pi
	}
	return d
}

func (w *World) stepProjs() {
	// delayed spawns
	if len(w.Spawns) > 0 {
		keep := w.Spawns[:0]
		for _, s := range w.Spawns {
			if w.Time >= s.At {
				w.newProj(s.Proj)
			} else {
				keep = append(keep, s)
			}
		}
		w.Spawns = keep
	}
	for i := 0; i < len(w.Projs); i++ {
		p := w.Projs[i]
		if p.Alive {
			w.stepProj(p)
		}
	}
	if w.Phase == PhaseBattle {
		w.stepAA()
	}
	live := w.Projs[:0]
	for _, p := range w.Projs {
		if p.Alive {
			live = append(live, p)
		}
	}
	w.Projs = live
}

func (w *World) stepProj(p *Proj) {
	wd := w.Cfg.W(p.Weapon)
	if wd == nil {
		p.Alive = false
		return
	}
	p.Age += Dt
	g := w.Cfg.GravityPx
	switch p.Kind {
	case PShell, PRocket:
		p.Vel.Y += g * wd.Gravity * Dt
		p.Vel.X += w.Wind * wd.WindK * Dt
		if p.Kind == PRocket {
			p.Heading = math.Atan2(p.Vel.Y, p.Vel.X)
		}
		if w.advance(p, wd.Fuse > 0) {
			w.detonate(p, p.Pos)
			return
		}
		if wd.Fuse > 0 && p.Age >= wd.Fuse && p.Alive {
			w.detonate(p, p.Pos)
		}
	case PBallis:
		if p.Phase == 0 {
			p.Pos = p.Pos.Add(p.Vel.Mul(Dt))
			if p.Pos.Y < -700 {
				w.beginDescent(p, wd)
			}
			return
		}
		if w.advance(p, false) {
			w.detonate(p, p.Pos)
		}
	case PWarhead:
		if w.advance(p, false) {
			w.detonate(p, p.Pos)
		}
	case PBomb:
		bg := wd.Gravity
		if bg <= 0 {
			bg = 1
		}
		p.Vel.Y += g * bg * Dt
		p.Vel.X += w.Wind * 0.3 * Dt
		if w.advance(p, false) {
			w.detonate(p, p.Pos)
		}
	case PPlane:
		p.Pos = p.Pos.Add(p.Vel.Mul(Dt))
		if p.Pos.X < -250 || p.Pos.X > float64(w.Terr.W)+250 {
			p.Alive = false
		}
	case PDrone:
		p.Flight -= Dt
		if p.Flight > 0 {
			diff := angDiff(p.Heading, p.Steer)
			p.Heading += clamp(diff, -3.2*Dt, 3.2*Dt)
			p.Vel = Dir(p.Heading).Mul(wd.Speed)
		} else {
			p.Vel.Y += g * Dt
			p.Heading = math.Atan2(p.Vel.Y, p.Vel.X)
		}
		if w.advance(p, false) {
			w.detonate(p, p.Pos)
		}
	case PGeran:
		w.stepGeran(p, wd)
	case PMine:
		if !w.unitOnGroundPx(p.Pos) {
			p.Pos.Y += 120 * Dt
			if p.Pos.Y > WaterY {
				p.Alive = false
			}
			return
		}
		if p.Armed > 0 {
			p.Armed -= Dt
			return
		}
		if p.Fuse < 0 {
			w.detonate(p, p.Pos)
			return
		}
		for _, u := range w.Units {
			if u.Alive && w.Hostile(p.Owner, u.Owner) && math.Abs(u.Pos.X-p.Pos.X) < 22 && math.Abs(u.Pos.Y-p.Pos.Y) < 22 {
				w.detonate(p, p.Pos)
				return
			}
		}
	}
}

func (w *World) unitOnGroundPx(pos Vec) bool {
	return w.SolidPx(pos.X, pos.Y+1)
}

func (w *World) beginDescent(p *Proj, wd *balance.Weapon) {
	tx := p.Target.X
	if wd.Kind == balance.KindMIRV {
		n := max(1, wd.Count)
		for i := 0; i < n; i++ {
			off := 0.0
			if n > 1 {
				off = (float64(i)/float64(n-1) - 0.5) * wd.Spread
			}
			x := tx + off
			gy := float64(w.Terr.SurfaceY(int(clamp(x, 0, float64(w.Terr.W-1))), 0))
			start := Vec{x - 170 - float64(i)*8, -700 - float64(i)*55}
			v := Vec{x, gy}.Sub(start)
			v = v.Mul(wd.Speed / v.Len())
			w.newProj(Proj{Kind: PWarhead, Owner: p.Owner, Weapon: p.Weapon, Pos: start, Vel: v, Phase: 1, Class: wd.Class, Dmg: p.Dmg})
		}
		p.Alive = false
		return
	}
	start := Vec{tx - 170, -700}
	v := p.Target.Sub(start)
	p.Vel = v.Mul(wd.Speed / v.Len())
	p.Pos = start
	p.Phase = 1
	p.Class = wd.Class
}

func (w *World) stepGeran(p *Proj, wd *balance.Weapon) {
	p.Flight -= Dt
	cruise := 240.0
	switch p.Phase {
	case 0: // climb
		if p.Pos.Y <= cruise {
			p.Phase = 1
			dir := 1.0
			if p.Target.X < p.Pos.X {
				dir = -1
			}
			p.Vel = Vec{dir * wd.Speed, 0}
		}
	case 1: // cruise
		if math.Abs(p.Pos.X-p.Target.X) < 24 {
			p.Phase = 2
			p.Vel = Vec{0, wd.Speed * 3.5}
		}
	}
	if p.Flight <= 0 && p.Phase < 2 {
		p.Phase = 2
		p.Vel = Vec{p.Vel.X * 0.5, wd.Speed * 3}
	}
	p.Heading = math.Atan2(p.Vel.Y, p.Vel.X)
	if w.advance(p, false) {
		w.detonate(p, p.Pos)
	}
}

// ProjectilesBusy reports whether anything is still flying or about to spawn.
func (w *World) ProjectilesBusy() bool {
	if len(w.Spawns) > 0 {
		return true
	}
	for _, p := range w.Projs {
		if p.Alive && p.Kind != PMine {
			return true
		}
	}
	return false
}

// UnitsSettled: nobody is airborne.
func (w *World) UnitsSettled() bool {
	for _, u := range w.Units {
		if u.Alive && (!u.Ground || u.Vel.Len() > 4) {
			return false
		}
	}
	return true
}

// StructsSettled: nothing is collapsing.
func (w *World) StructsSettled() bool {
	for _, s := range w.Structs {
		if s.Alive && s.Moving {
			return false
		}
	}
	return true
}

// ownGhost: young projectiles (and climbing loitering munitions / fresh drones)
// pass through their owner's own base so they can leave it.
func ownGhost(p *Proj) bool {
	switch p.Kind {
	case PGeran:
		return p.Phase == 0
	case PDrone:
		return p.Age < 0.6
	}
	return p.Age < 0.12
}

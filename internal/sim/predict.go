package sim

import (
	"math"

	"svinovoyna/internal/balance"
)

// PredictShell simulates a ballistic projectile without side effects and
// returns where it ends up (impact point, true if it hit something solid).
type unitBox struct {
	x, y  float64
	owner int
}

func (w *World) unitBoxes() []unitBox {
	var bs []unitBox
	for _, u := range w.Units {
		if u.Alive {
			bs = append(bs, unitBox{u.Pos.X, u.Pos.Y, u.Owner})
		}
	}
	return bs
}

func (w *World) blockedFast(p *Proj, bs []unitBox) bool {
	x, y := p.Pos.X, p.Pos.Y
	if w.Terr.Solid(int(math.Floor(x)), int(math.Floor(y))) {
		return true
	}
	if s := w.StructAtPx(x, y); s != nil && !(p.Age < 0.1 && s.Owner == p.Owner) {
		return true
	}
	for _, u := range bs {
		if p.Age < 0.12 && u.owner == p.Owner {
			continue
		}
		if math.Abs(x-u.x) <= UnitW/2+2 && y >= u.y-UnitH-2 && y <= u.y+1 {
			return true
		}
	}
	return false
}

func (w *World) PredictShell(from, vel Vec, wd *balance.Weapon, owner int) (Vec, bool) {
	p := &Proj{Pos: from, Vel: vel, Owner: owner}
	bs := w.unitBoxes()
	g := w.Cfg.GravityPx
	for t := 0.0; t < 12; t += Dt {
		p.Age += Dt
		p.Vel.Y += g * wd.Gravity * Dt
		p.Vel.X += w.Wind * wd.WindK * Dt
		n := int(math.Ceil(p.Vel.Len() * Dt / 3))
		if n < 1 {
			n = 1
		}
		step := p.Vel.Mul(Dt / float64(n))
		for i := 0; i < n; i++ {
			p.Pos = p.Pos.Add(step)
			if p.Pos.X < -300 || p.Pos.X > float64(w.Terr.W)+300 || p.Pos.Y > WaterY {
				return p.Pos, false
			}
			if p.Pos.Y >= 0 && w.blockedFast(p, bs) {
				return p.Pos, true
			}
		}
		if wd.Fuse > 0 && p.Age >= wd.Fuse {
			return p.Pos, true
		}
	}
	return p.Pos, false
}

// TraceRay finds what a bullet fired from `from` at `ang` would hit first.
func (w *World) TraceRay(from Vec, ang float64, maxD float64, ignoreUnit, ignoreStruct int) (Vec, *Unit, *Struct) {
	dir := Dir(ang)
	for d := 0.0; d < maxD; d += 2 {
		p := from.Add(dir.Mul(d))
		if p.X < -50 || p.X > float64(w.Terr.W)+50 || p.Y > WaterY {
			return p, nil, nil
		}
		for _, u := range w.Units {
			if !u.Alive || u.ID == ignoreUnit {
				continue
			}
			if math.Abs(p.X-u.Pos.X) <= UnitW/2+1 && p.Y >= u.Pos.Y-UnitH-1 && p.Y <= u.Pos.Y+1 {
				return p, u, nil
			}
		}
		if s := w.StructAtPx(p.X, p.Y); s != nil && s.ID != ignoreStruct {
			return p, nil, s
		}
		if w.Terr.Solid(int(p.X), int(p.Y)) {
			return p, nil, nil
		}
	}
	return from.Add(dir.Mul(maxD)), nil, nil
}

// EstimateExplosion returns the expected "value" (in money-like units) of an
// explosion at pos for the attacker: positive for enemy damage, negative for
// self/ally damage. It has no side effects.
func (w *World) EstimateExplosion(pos Vec, radius, damage float64, owner int, blockMul, pierce float64) float64 {
	if blockMul == 0 {
		blockMul = 1
	}
	score := 0.0
	for _, u := range w.Units {
		if !u.Alive {
			continue
		}
		c := unitCenter(u)
		d := c.Dist(pos)
		reach := radius + 8
		if d > reach {
			continue
		}
		f := 1 - 0.75*(d/reach)
		dmg := damage * f * w.blastShield(pos, c, -1)
		cost := float64(w.Cfg.U(u.Def).Cost)
		eff := math.Min(dmg, u.HP)
		val := eff/u.MaxHP*cost + eff*0.3
		if dmg >= u.HP {
			val += cost*0.5 + float64(w.Cfg.KillBonus)
		}
		if u.Owner == owner || !w.Hostile(owner, u.Owner) {
			score -= val * 1.6
		} else {
			score += val
		}
	}
	for _, s := range w.Structs {
		if !s.Alive {
			continue
		}
		x0, y0, x1, y1 := w.StructRect(s)
		d := math.Hypot(pos.X-clamp(pos.X, x0, x1), pos.Y-clamp(pos.Y, y0, y1))
		if d > radius {
			continue
		}
		f := 1 - 0.75*(d/radius)
		def := w.Cfg.S(s.Def)
		dmg := damage * f * blockMul * (1 - def.Blast*(1-pierce)) * (0.5 + 0.5*w.blastShield(pos, w.StructCenter(s), s.ID))
		cost := float64(def.Cost)
		if def.Kind == balance.SHQ {
			cost = 1600 * (1 + 2*(1-s.HP/s.MaxHP))
		}
		eff := math.Min(dmg, s.HP)
		val := eff / s.MaxHP * cost * 0.8
		if dmg >= s.HP {
			val += cost * 0.15
		}
		if s.Owner == owner || !w.Hostile(owner, s.Owner) {
			score -= val * 1.6
		} else {
			score += val
		}
	}
	return score
}

package sim

import (
	"math"

	"svinovoyna/internal/balance"
)

func contains(a []int, v int) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}

// interceptable reports whether a projectile can currently be shot at.
func interceptable(p *Proj) bool {
	if p.Class == balance.ClassNone || !p.Alive {
		return false
	}
	switch p.Kind {
	case PBallis:
		return p.Phase == 1
	case PWarhead, PRocket, PDrone, PGeran, PPlane, PShell:
		return true
	}
	return false
}

// stepAA lets air-defence structures engage hostile airborne projectiles.
func (w *World) stepAA() {
	for _, s := range w.Structs {
		if !s.Alive {
			continue
		}
		d := w.Cfg.S(s.Def)
		if d.Kind != balance.SAA && d.Kind != balance.SJammer {
			continue
		}
		c := w.StructCenter(s)
		for _, p := range w.Projs {
			if !interceptable(p) || !w.Hostile(s.Owner, p.Owner) || contains(p.Tried, s.ID) {
				continue
			}
			if aaDist(p.Pos, c) > d.AARange {
				continue
			}
			prob, ok := d.AAHit[p.Class]
			if !ok || prob <= 0 {
				continue
			}
			if d.Kind == balance.SAA {
				if s.AAAmmo <= 0 {
					break
				}
				s.AAAmmo--
			}
			p.Tried = append(p.Tried, s.ID)
			if w.RNG.F() < prob {
				p.Alive = false
				w.stat(p.Weapon).Intercepted++
				w.emit(Event{Type: EvIntercept, Pos: p.Pos, To: c, A: s.Owner, Text: string(p.Class)})
				if p.Kind == PPlane {
					w.cancelSpawns(p.ID)
				}
			} else {
				w.emit(Event{Type: EvIntercept, Pos: p.Pos, To: c, A: s.Owner, B: 1, Text: string(p.Class)})
			}
		}
	}
}

// aaDist is the engagement distance: vertical distance counts for less than
// horizontal, so ground batteries can reach aircraft flying high above them.
func aaDist(a, b Vec) float64 {
	dx, dy := a.X-b.X, (a.Y-b.Y)*0.4
	return math.Sqrt(dx*dx + dy*dy)
}

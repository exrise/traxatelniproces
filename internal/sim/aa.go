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
	if p.Class == balance.ClassNone || !p.Alive || p.Doom > 0 {
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

// burstLen is how many ticks an air-defence battery keeps firing at a target.
func burstLen(d *balance.StructDef) int {
	switch d.ID {
	case "zu23":
		return 22 // a long twin-barrel burst of 23 mm
	case "pantsir":
		return 12
	}
	return 8
}

// stepAA lets air-defence structures engage hostile airborne projectiles. Every
// engagement is visible: the barrel turns, tracers / a missile fly out, and a
// successful hit destroys the target a moment later.
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
		w.aaBurstVisuals(s, d)
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
			if d.Kind == balance.SAA {
				s.Burst = burstLen(d)
				s.BurstID = p.ID
				s.BurstPos = p.Pos
				snd := "zu23gun"
				if d.ID != "zu23" {
					snd = "aamissile"
				}
				w.emit(Event{Type: EvShot, Pos: w.StructMuzzle(s), Text: snd, F: s.Aim})
			}
			if w.RNG.F() < prob {
				// the target dies a moment into the burst, sooner for fast things
				p.Doom = math.Min(0.55, math.Max(0.08, 120/math.Max(60, p.Vel.Len())))
				if p.Kind == PPlane {
					w.cancelSpawns(p.ID) // a hit bomber never drops its bombs
				}
				p.DoomBy = s.ID
			} else {
				w.emit(Event{Type: EvIntercept, Pos: p.Pos, To: c, A: s.Owner, B: 1, Text: string(p.Class)})
			}
		}
	}
}

// aaBurstVisuals aims the battery and spits tracers while a burst is on.
func (w *World) aaBurstVisuals(s *Struct, d *balance.StructDef) {
	if s.Burst <= 0 {
		return
	}
	s.Burst--
	tgt := s.BurstPos
	for _, p := range w.Projs {
		if p.ID == s.BurstID && p.Alive {
			tgt = p.Pos
			s.BurstPos = p.Pos
		}
	}
	muzzle := w.StructMuzzle(s)
	ang := ClampStructAim(math.Atan2(tgt.Y-muzzle.Y, tgt.X-muzzle.X))
	s.Aim = ang
	if s.Burst%2 != 0 {
		return
	}
	dist := tgt.Dist(muzzle)
	jit := dist * 0.025
	kind := 0.0
	if d.ID != "zu23" {
		kind = 4 // a missile streak instead of cannon tracers
	}
	n := 2
	if kind != 0 {
		n = 1
	}
	for i := 0; i < n; i++ {
		to := Vec{X: tgt.X + w.RNG.Range(-jit, jit), Y: tgt.Y + w.RNG.Range(-jit, jit)}
		w.emit(Event{Type: EvTracer, Pos: muzzle.Add(Dir(ang).Mul(float64(i) * 4)), To: to, F: kind})
	}
}

// aaDist is the engagement distance: vertical distance counts for less than
// horizontal, so ground batteries can reach aircraft flying high above them.
func aaDist(a, b Vec) float64 {
	dx, dy := a.X-b.X, (a.Y-b.Y)*0.4
	return math.Sqrt(dx*dx + dy*dy)
}

// resolveDoom finishes a projectile that was hit by air defence.
func (w *World) resolveDoom(p *Proj) {
	p.Alive = false
	w.stat(p.Weapon).Intercepted++
	var from Vec
	if p.DoomBy >= 0 && p.DoomBy < len(w.Structs) {
		from = w.StructCenter(w.Structs[p.DoomBy])
	}
	owner := -1
	if p.DoomBy >= 0 && p.DoomBy < len(w.Structs) {
		owner = w.Structs[p.DoomBy].Owner
	}
	w.emit(Event{Type: EvIntercept, Pos: p.Pos, To: from, A: owner, Text: string(p.Class)})
	if p.Kind == PPlane {
		w.cancelSpawns(p.ID)
	}
}

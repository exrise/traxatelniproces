package sim

import (
	"math"
)

func (w *World) unitCanAct(u *Unit) bool {
	return w.Phase == PhaseBattle && w.Cur == u.Owner && w.SelUnit == u.ID &&
		(w.Stage == StageActive || (w.Stage == StageRetreat && w.FiredUnit == u.ID) || w.Stage == StagePlan)
}

// stepUnits advances physics for every unit.
func (w *World) stepUnits() {
	for _, u := range w.Units {
		if !u.Alive {
			continue
		}
		w.stepUnit(u)
	}
}

func (w *World) stepUnit(u *Unit) {
	if u.Hurt > 0 {
		u.Hurt -= Dt
	}
	if !w.unitBodyFree(u.Pos.X, u.Pos.Y) {
		// pushed into something (falling structure, build-up): climb out
		for k := 1; k <= 40; k++ {
			if w.unitBodyFree(u.Pos.X, u.Pos.Y-float64(k)) {
				u.Pos.Y -= float64(k)
				break
			}
		}
	}
	ground := w.unitOnGround(u.Pos.X, u.Pos.Y)
	moving := false
	if u.Walk != 0 && ground && w.unitCanAct(u) && u.Vel.Len() < 1 {
		u.Face = u.Walk
		moving = w.walkUnit(u, float64(u.Walk)*w.Cfg.UnitSpeed*Dt)
		ground = w.unitOnGround(u.Pos.X, u.Pos.Y)
	}
	if moving {
		u.Anim += Dt * 9
	}
	if ground && u.Vel.Y >= 0 {
		if u.FallFrom != 0 {
			w.landUnit(u)
		}
		if math.Abs(u.Vel.X) > 1 {
			u.Vel.X *= 0.8
		} else {
			u.Vel.X = 0
		}
		u.Vel.Y = 0
		u.Ground = true
	} else {
		if u.Ground || u.FallFrom == 0 {
			u.FallFrom = u.Pos.Y
		}
		u.Ground = false
		u.Vel.Y += w.Cfg.GravityPx * Dt
	}
	if u.Vel.X != 0 || u.Vel.Y != 0 {
		w.moveUnitBy(u, u.Vel.Mul(Dt))
	}
	// map bounds and water
	if u.Pos.X < 6 {
		u.Pos.X, u.Vel.X = 6, 0
	}
	if mx := float64(w.Terr.W) - 6; u.Pos.X > mx {
		u.Pos.X, u.Vel.X = mx, 0
	}
	if u.Pos.Y > WaterY {
		w.emit(Event{Type: EvSplash, Pos: Vec{u.Pos.X, WaterY}})
		w.killUnit(u, -1, "утонул")
	}
}

func (w *World) landUnit(u *Unit) {
	dist := u.Pos.Y - u.FallFrom
	u.FallFrom = 0
	if dist > 170 {
		dmg := (dist - 170) * w.Cfg.FallDamage
		w.hurtUnit(u, dmg, -1, 0)
	}
}

// walkUnit moves horizontally up to |dx| pixels, climbing small steps.
func (w *World) walkUnit(u *Unit, dx float64) bool {
	dir := 1.0
	if dx < 0 {
		dir = -1
	}
	moved := false
	for rem := math.Abs(dx); rem > 0; rem -= 1 {
		step := math.Min(1, rem) * dir
		nx := u.Pos.X + step
		if w.unitBodyFree(nx, u.Pos.Y) {
			u.Pos.X = nx
			moved = true
		} else {
			climbed := false
			for k := 1; k <= 7; k++ {
				if w.unitBodyFree(nx, u.Pos.Y-float64(k)) {
					u.Pos.X, u.Pos.Y = nx, u.Pos.Y-float64(k)
					climbed, moved = true, true
					break
				}
			}
			if !climbed {
				break
			}
		}
		// follow the ground downhill
		if !w.unitOnGround(u.Pos.X, u.Pos.Y) {
			for k := 1; k <= 5; k++ {
				if w.unitOnGround(u.Pos.X, u.Pos.Y+float64(k)) && w.unitBodyFree(u.Pos.X, u.Pos.Y+float64(k)) {
					u.Pos.Y += float64(k)
					break
				}
			}
		}
	}
	return moved
}

// moveUnitBy moves along a velocity vector pixel by pixel with collisions.
func (w *World) moveUnitBy(u *Unit, d Vec) {
	n := int(math.Ceil(math.Max(math.Abs(d.X), math.Abs(d.Y))))
	if n < 1 {
		n = 1
	}
	sx, sy := d.X/float64(n), d.Y/float64(n)
	for i := 0; i < n; i++ {
		if sx != 0 {
			nx := u.Pos.X + sx
			if w.unitBodyFree(nx, u.Pos.Y) {
				u.Pos.X = nx
			} else {
				ok := false
				for k := 1; k <= 3; k++ {
					if w.unitBodyFree(nx, u.Pos.Y-float64(k)) {
						u.Pos.X, u.Pos.Y = nx, u.Pos.Y-float64(k)
						ok = true
						break
					}
				}
				if !ok {
					u.Vel.X *= -0.25
					sx = 0
				}
			}
		}
		if sy != 0 {
			ny := u.Pos.Y + sy
			if w.unitBodyFree(u.Pos.X, ny) {
				u.Pos.Y = ny
			} else {
				if sy > 0 {
					// landed
					if u.Vel.Y > 380 {
						u.Vel.Y *= -0.2
					} else {
						u.Vel.Y = 0
					}
				} else {
					u.Vel.Y = 0
				}
				sy = 0
			}
		}
	}
}

// Knock applies an impulse to a unit.
func (w *World) knock(u *Unit, imp Vec) {
	u.Vel = u.Vel.Add(imp)
	if imp.Y < 0 || imp.Len() > 40 {
		u.Ground = false
		u.Pos.Y -= 2
		if u.FallFrom == 0 {
			u.FallFrom = u.Pos.Y
		}
	}
}

// Jump launches the selected unit.
func (w *World) jumpUnit(u *Unit) {
	if !u.Ground {
		return
	}
	u.Vel = Vec{float64(u.Face) * 120, -300}
	u.Ground = false
	u.Pos.Y -= 2
	u.FallFrom = u.Pos.Y
	w.emit(Event{Type: EvJump, Pos: u.Pos})
}

// settleStructs makes unsupported structures fall one cell at a time.
func (w *World) settleStructs() {
	for _, s := range w.Structs {
		if !s.Alive {
			continue
		}
		d := w.Cfg.S(s.Def)
		if s.Hurt > 0 {
			s.Hurt -= Dt
		}
		if w.supportedAt(s.CX, s.CY, d.W, d.H, s.ID) {
			if s.Fall > 0 {
				s.Fall = 0
			}
			continue
		}
		s.Fall += Dt
		if s.Fall < 0.10 {
			continue
		}
		s.Fall = 0.0001
		// can it move down?
		free := true
		for x := 0; x < d.W; x++ {
			if !w.cellFree(s.CX+x, s.CY+d.H, s.ID) {
				free = false
			}
		}
		if !free {
			continue
		}
		w.setOcc(s, 0)
		s.CY++
		w.setOcc(s, int32(s.ID+1))
		if w.supportedAt(s.CX, s.CY, d.W, d.H, s.ID) {
			w.damageStructRaw(s, 6, -1)
			w.emit(Event{Type: EvExplosion, Pos: Vec{w.StructCenter(s).X, float64((s.CY + d.H) * Cell)}, R: 14, F: 0})
		}
	}
}

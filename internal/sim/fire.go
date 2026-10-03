package sim

import (
	"math"

	"svinovoyna/internal/balance"
)

// FireSpec fully describes one shot, whoever the actor is.
type FireSpec struct {
	Player  int
	Unit    int // actor unit id or -1
	Struct  int // actor structure id or -1 (both -1: call from HQ)
	Weapon  string
	Angle   float64
	Power   float64
	TargetX float64
}

func (w *World) dmgMul() float64 {
	if w.Cfg.TurnMode == balance.TurnUnitAndStruct {
		return w.Cfg.TwoHandDmgMul
	}
	return 1
}

func (w *World) hasSpotter(pid int) bool {
	for _, u := range w.Units {
		if u.Alive && u.Owner == pid && u.Def == "spotter" {
			return true
		}
	}
	return false
}

func (w *World) newProj(p Proj) *Proj {
	p.ID = w.nextProj
	w.nextProj++
	p.Alive = true
	if p.Dmg == 0 {
		p.Dmg = 1
	}
	np := p
	w.Projs = append(w.Projs, &np)
	return &np
}

// ClampStructAim keeps structure weapons pointing upwards.
func ClampStructAim(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	if a > -0.1 && a <= math.Pi/2 {
		a = -0.1
	} else if a > math.Pi/2 || a < -math.Pi+0.1 {
		a = -math.Pi + 0.1
	}
	return a
}

// IsItem reports whether the weapon is a purchasable consumable.
func (w *World) IsItem(id string) bool {
	wd := w.Cfg.W(id)
	return wd != nil && wd.Cost > 0 && (wd.Kind == balance.KindAirstrike || wd.Kind == balance.KindGeran)
}

// NeedsTargetX: weapons aimed by a ground target rather than a direction.
func NeedsTargetX(k balance.WeaponKind) bool {
	return k == balance.KindBallis || k == balance.KindMIRV || k == balance.KindAirstrike || k == balance.KindGeran
}

// launch spawns the effects of a shot. It does no turn bookkeeping.
func (w *World) launch(sp FireSpec) error {
	wd := w.Cfg.W(sp.Weapon)
	if wd == nil {
		return ErrUnknown
	}
	var muzzle Vec
	var shooter *Unit
	ignoreStruct := -1
	angle := sp.Angle
	switch {
	case sp.Unit >= 0:
		shooter = w.Units[sp.Unit]
		if math.Cos(angle) >= 0 {
			shooter.Face = 1
		} else {
			shooter.Face = -1
		}
		shooter.Aim = angle
		muzzle = Vec{shooter.Pos.X, shooter.Pos.Y - UnitH*0.6}.Add(Dir(angle).Mul(9))
	case sp.Struct >= 0:
		s := w.Structs[sp.Struct]
		angle = ClampStructAim(angle)
		s.Aim = angle
		muzzle = w.StructMuzzle(s).Add(Dir(angle).Mul(6))
		ignoreStruct = s.ID
	default:
		hq := w.Structs[w.Players[sp.Player].HQ]
		muzzle = w.StructMuzzle(hq)
		ignoreStruct = hq.ID
	}
	w.curWeapon = wd.ID
	defer func() { w.curWeapon = "" }()
	w.stat(wd.ID).Shots++
	pw := clamp(sp.Power, 0.1, 1)
	k := w.dmgMul()
	owner := sp.Player
	w.emit(Event{Type: EvShot, Pos: muzzle, Text: wd.ID, F: angle})

	switch wd.Kind {
	case balance.KindBurst, balance.KindPellets, balance.KindShot:
		su := -1
		if shooter != nil {
			su = shooter.ID
		}
		if wd.Kind == balance.KindBurst {
			// a real burst: bullets leave one after another along the aim line
			// with a small spread, the first one dead on target
			n := max(1, wd.Count)
			gap := math.Min(0.08, math.Max(0.025, 0.5/float64(n)))
			for i := 0; i < n; i++ {
				a := angle
				if i > 0 {
					a += w.RNG.Range(-wd.Spread, wd.Spread) * 0.5
				}
				if i == 0 {
					w.rayShot(muzzle, a, wd, owner, k, su, ignoreStruct)
				} else {
					w.Shots = append(w.Shots, QShot{At: w.Time + float64(i)*gap, From: muzzle, Angle: a, Weapon: wd.ID, Owner: owner, Unit: su, IgnoreStruct: ignoreStruct, K: k})
				}
			}
		} else {
			for i := 0; i < max(1, wd.Count); i++ {
				a := angle + w.RNG.Range(-wd.Spread, wd.Spread)
				w.rayShot(muzzle, a, wd, owner, k, su, ignoreStruct)
			}
		}
	case balance.KindShell:
		w.newProj(Proj{Kind: PShell, Owner: owner, Weapon: wd.ID, Pos: muzzle, Vel: Dir(angle).Mul(wd.Speed * pw), Fuse: wd.Fuse, Class: wd.Class, Dmg: k})
	case balance.KindMissile:
		w.newProj(Proj{Kind: PRocket, Owner: owner, Weapon: wd.ID, Pos: muzzle, Vel: Dir(angle).Mul(wd.Speed), Class: wd.Class, Dmg: k})
	case balance.KindSalvo:
		for i := 0; i < wd.Count; i++ {
			a := angle + w.RNG.Range(-wd.Spread, wd.Spread)
			v := wd.Speed * pw * w.RNG.Range(0.93, 1.07)
			w.Spawns = append(w.Spawns, Spawn{At: w.Time + float64(i)*0.07, Parent: -1, Proj: Proj{Kind: PRocket, Owner: owner, Weapon: wd.ID,
				Pos: muzzle, Vel: Dir(a).Mul(v), Class: wd.Class, Dmg: k}})
		}
	case balance.KindBallis, balance.KindMIRV:
		tx := clamp(sp.TargetX, 20, float64(w.Terr.W-20))
		gy := float64(w.Terr.SurfaceY(int(tx), 0))
		w.newProj(Proj{Kind: PBallis, Owner: owner, Weapon: wd.ID, Pos: muzzle, Vel: Vec{0, -wd.Speed}, Target: Vec{tx, gy}, Class: balance.ClassNone, Dmg: k})
	case balance.KindDrone, balance.KindGuided:
		ph := 0
		if wd.Kind == balance.KindGuided {
			ph = 1 // guided missile: same steering as a drone, drawn as a missile
		}
		w.newProj(Proj{Kind: PDrone, Owner: owner, Weapon: wd.ID, Pos: muzzle, Heading: angle, Steer: angle, Phase: ph,
			Vel: Dir(angle).Mul(wd.Speed), Flight: wd.Flight, Class: wd.Class, Dmg: k})
	case balance.KindGeran:
		tx := clamp(sp.TargetX, 20, float64(w.Terr.W-20))
		gy := float64(w.Terr.SurfaceY(int(tx), 0))
		dir := 1.0
		if tx < muzzle.X {
			dir = -1
		}
		w.newProj(Proj{Kind: PGeran, Owner: owner, Weapon: wd.ID, Pos: muzzle, Vel: Vec{dir * wd.Speed * 0.6, -wd.Speed * 0.8},
			Target: Vec{tx, gy}, Flight: wd.Flight, Class: wd.Class, Dmg: k})
	case balance.KindAirstrike:
		w.callAirstrike(sp, wd, k)
	case balance.KindMine:
		if shooter == nil {
			return ErrAction
		}
		pos := Vec{shooter.Pos.X + float64(shooter.Face)*13, shooter.Pos.Y - 3}
		w.newProj(Proj{Kind: PMine, Owner: owner, Weapon: wd.ID, Pos: pos, Armed: 1.2, Dmg: 1})
	case balance.KindRepair:
		if shooter == nil {
			return ErrAction
		}
		for _, s := range w.Structs {
			if s.Alive && s.Owner == owner && w.StructCenter(s).Dist(shooter.Pos) < wd.Radius+20 && s.HP < s.MaxHP {
				s.HP = math.Min(s.MaxHP, s.HP+wd.Damage)
				w.emit(Event{Type: EvHeal, Pos: w.StructCenter(s), F: wd.Damage})
			}
		}
	}
	return nil
}

func (w *World) callAirstrike(sp FireSpec, wd *balance.Weapon, k float64) {
	owner := sp.Player
	tx := clamp(sp.TargetX, 40, float64(w.Terr.W-40))
	spreadK := 1.0
	if !w.hasSpotter(owner) {
		spreadK = 3
	}
	// enter from the side the owner is on
	hq := w.Structs[w.Players[owner].HQ]
	dir := 1.0
	if w.StructCenter(hq).X > float64(w.Terr.W)/2 {
		dir = -1
	}
	// if the target is behind the entry edge, fly the other way
	startX := -150.0
	if dir < 0 {
		startX = float64(w.Terr.W) + 150
	}
	const planeY = 320.0
	gk := wd.Gravity
	if gk <= 0 {
		gk = 1
	}
	plane := w.newProj(Proj{Kind: PPlane, Owner: owner, Weapon: wd.ID, Pos: Vec{startX, planeY}, Vel: Vec{dir * wd.Speed, 0}, Class: wd.Class})
	w.emit(Event{Type: EvPlane, Pos: plane.Pos, A: owner})
	n := max(1, wd.Count)
	for i := 0; i < n; i++ {
		off := (float64(i) - float64(n-1)/2) * wd.Spread
		x := tx + off + w.RNG.Norm()*wd.Spread*0.08*spreadK
		if n == 1 {
			x = tx + w.RNG.Norm()*(wd.Spread*(spreadK-1)*0.5+4)
		}
		gy := float64(w.Terr.SurfaceY(int(clamp(x, 0, float64(w.Terr.W-1))), 0))
		t := math.Sqrt(2 * (gy - planeY) / (w.Cfg.GravityPx * gk))
		dropX := x - dir*wd.Speed*t
		at := w.Time + (dropX-startX)/(dir*wd.Speed)
		w.Spawns = append(w.Spawns, Spawn{At: at, Parent: plane.ID, Proj: Proj{Kind: PBomb, Owner: owner, Weapon: wd.ID,
			Pos: Vec{dropX, planeY}, Vel: Vec{dir * wd.Speed, 0}, Dmg: k}})
	}
}

// rayShot fires one instant bullet.
func (w *World) rayShot(from Vec, ang float64, wd *balance.Weapon, owner int, k float64, shooterUnit, ignoreStruct int) {
	dir := Dir(ang)
	maxD := wd.Range
	if maxD <= 0 {
		maxD = 600
	}
	end := from
	for d := 0.0; d < maxD; d += 2 {
		p := from.Add(dir.Mul(d))
		end = p
		if p.X < -50 || p.X > float64(w.Terr.W)+50 || p.Y > WaterY {
			break
		}
		for _, u := range w.Units {
			if !u.Alive || u.ID == shooterUnit {
				continue
			}
			if math.Abs(p.X-u.Pos.X) <= UnitW/2+1 && p.Y >= u.Pos.Y-UnitH-1 && p.Y <= u.Pos.Y+1 {
				dmg := wd.Damage * k
				if wd.Falloff > 0 {
					dmg *= 1 - wd.Falloff*d/maxD
				}
				w.damageUnit(u, dmg, owner)
				if u.Alive {
					w.knock(u, dir.Mul(dmg*1.2))
				}
				w.emit(Event{Type: EvTracer, Pos: from, To: p, F: 1})
				return
			}
		}
		if s := w.StructAtPx(p.X, p.Y); s != nil && s.ID != ignoreStruct && !w.passes(owner, s) {
			def := w.Cfg.S(s.Def)
			bm := wd.BlockMul
			if bm == 0 {
				bm = 1
			}
			dmg := wd.Damage * k * bm * (1 - def.Armor*(1-wd.Pierce))
			if wd.Falloff > 0 {
				dmg *= 1 - wd.Falloff*d/maxD
			}
			w.damageStruct(s, dmg, owner)
			w.emit(Event{Type: EvTracer, Pos: from, To: p, F: 2})
			return
		}
		if w.Terr.Solid(int(p.X), int(p.Y)) {
			if wd.Kind == balance.KindShot {
				w.Terr.Carve(int(p.X), int(p.Y), 3, true)
			}
			w.emit(Event{Type: EvTracer, Pos: from, To: p, F: 3})
			return
		}
	}
	w.emit(Event{Type: EvTracer, Pos: from, To: end, F: 0})
}

package sim

import (
	"math"

	"svinovoyna/internal/balance"
)

// unitCenter is the hit-box centre of a unit.
func unitCenter(u *Unit) Vec { return Vec{u.Pos.X, u.Pos.Y - UnitH/2} }

// damageAllowed implements the friendly-fire rule.
func (w *World) damageAllowed(attacker, victimOwner int) bool {
	if attacker < 0 || attacker == victimOwner {
		return true
	}
	if w.Hostile(attacker, victimOwner) {
		return true
	}
	return w.Cfg.FriendlyFire
}

// credit pays the attacker for damage dealt to an enemy.
func (w *World) credit(attacker, victim int, amount float64, at Vec) {
	if attacker < 0 || !w.Hostile(attacker, victim) || amount <= 0 {
		return
	}
	switch victim {
	case w.Leader:
		amount *= w.Cfg.LeaderMult
	case w.Underdog:
		amount *= w.Cfg.UnderdogMult
	}
	p := w.Players[attacker]
	p.Frac += amount
	whole := int(p.Frac)
	if whole > 0 {
		p.Frac -= float64(whole)
		p.Money += whole
		p.Earned += whole
		w.emit(Event{Type: EvMoney, Pos: at, A: attacker, B: whole})
	}
}

// hurtUnit lowers HP without any money logic; returns damage actually applied.
func (w *World) hurtUnit(u *Unit, dmg float64, attacker int, weaponKind int) float64 {
	if !u.Alive || dmg <= 0 {
		return 0
	}
	applied := math.Min(dmg, u.HP)
	u.HP -= dmg
	u.Hurt = 0.25
	w.emit(Event{Type: EvDamage, Pos: Vec{u.Pos.X, u.Pos.Y - UnitH}, F: dmg, A: u.Owner})
	if attacker >= 0 {
		w.Players[attacker].DamageDone += applied
	}
	if u.HP <= 0 {
		w.killUnit(u, attacker, "")
	}
	return applied
}

// DamageUnit damages a unit and pays the attacker.
func (w *World) damageUnit(u *Unit, dmg float64, attacker int) {
	if !u.Alive || !w.damageAllowed(attacker, u.Owner) {
		return
	}
	applied := w.hurtUnit(u, dmg, attacker, 0)
	w.credit(attacker, u.Owner, applied*w.Cfg.DmgMoneyUnit, unitCenter(u))
}

func (w *World) killUnit(u *Unit, attacker int, why string) {
	if !u.Alive {
		return
	}
	u.Alive = false
	u.HP = 0
	cost := w.Cfg.U(u.Def).Cost
	w.Players[u.Owner].LossValue += cost
	w.emit(Event{Type: EvUnitDied, Pos: unitCenter(u), A: u.Owner, B: u.ID, Text: why})
	if attacker >= 0 && w.Hostile(attacker, u.Owner) {
		w.Players[attacker].Kills++
		w.Players[attacker].Money += w.Cfg.KillBonus
		w.Players[attacker].Earned += w.Cfg.KillBonus
		w.emit(Event{Type: EvMoney, Pos: unitCenter(u), A: attacker, B: w.Cfg.KillBonus})
	}
	if w.SelUnit == u.ID && w.Cur == u.Owner {
		w.SelUnit = -1
	}
}

// damageStructRaw applies damage to a structure with no armor or money logic.
func (w *World) damageStructRaw(s *Struct, dmg float64, attacker int) float64 {
	if !s.Alive || dmg <= 0 {
		return 0
	}
	applied := math.Min(dmg, s.HP)
	s.HP -= dmg
	s.Hurt = 0.2
	if attacker >= 0 {
		w.Players[attacker].DamageDone += applied
	}
	if s.HP <= 0 {
		w.destroyStruct(s, attacker)
	}
	return applied
}

// damageStruct damages a structure and pays the attacker.
func (w *World) damageStruct(s *Struct, dmg float64, attacker int) {
	if !s.Alive || !w.damageAllowed(attacker, s.Owner) {
		return
	}
	hpBefore := s.HP
	applied := w.damageStructRaw(s, dmg, attacker)
	w.credit(attacker, s.Owner, applied*w.Cfg.DmgMoneyStruct, w.StructCenter(s))
	if hpBefore > 0 && !s.Alive && attacker >= 0 && w.Hostile(attacker, s.Owner) {
		bonus := int(float64(w.Cfg.S(s.Def).Cost) * w.Cfg.StructKillPct)
		if s.Def == "hq" {
			bonus = 150
		}
		w.Players[attacker].Money += bonus
		w.Players[attacker].Earned += bonus
		if bonus > 0 {
			w.emit(Event{Type: EvMoney, Pos: w.StructCenter(s), A: attacker, B: bonus})
		}
	}
}

func (w *World) destroyStruct(s *Struct, attacker int) {
	if !s.Alive {
		return
	}
	d := w.Cfg.S(s.Def)
	w.removeStruct(s)
	w.Players[s.Owner].LossValue += d.Cost
	c := w.StructCenter(s)
	x0, y0, x1, y1 := w.StructRect(s)
	w.emit(Event{Type: EvStructDestroyed, Pos: c, R: math.Max(x1-x0, y1-y0) / 2, A: s.Owner, B: s.ID})
	if d.Kind == balance.SHQ {
		w.msg("Штаб игрока %s уничтожен!", w.Players[s.Owner].Name)
	}
	_ = attacker
}

// blastShield returns the fraction (0..1) of explosion energy that reaches `to` from `from`.
func (w *World) blastShield(from, to Vec, ignoreStruct int) float64 {
	d := to.Sub(from)
	l := d.Len()
	if l < 6 {
		return 1
	}
	n := int(l / 4)
	mult := 1.0
	terr := 0
	seen := map[int]bool{}
	for i := 2; i < n; i++ {
		p := from.Add(d.Mul(float64(i) / float64(n)))
		if w.Terr.Solid(int(p.X), int(p.Y)) {
			terr++
		}
		if s := w.StructAtPx(p.X, p.Y); s != nil && s.ID != ignoreStruct && !seen[s.ID] {
			seen[s.ID] = true
			mult *= 1 - w.Cfg.S(s.Def).Blast*0.8
		}
	}
	mult *= 1 - math.Min(0.8, float64(terr)*0.07)
	return mult
}

// ExplosionSpec carries weapon-dependent explosion parameters.
type ExplosionSpec struct {
	Radius   float64
	Damage   float64
	Crater   float64
	BlockMul float64
	Pierce   float64
	Owner    int
}

// Explode detonates at pos: damages units and structures, carves the terrain.
func (w *World) Explode(pos Vec, sp ExplosionSpec) {
	if sp.BlockMul == 0 {
		sp.BlockMul = 1
	}
	if sp.Crater == 0 {
		sp.Crater = 1
	}
	w.emit(Event{Type: EvExplosion, Pos: pos, R: sp.Radius, A: sp.Owner})
	// units
	for _, u := range w.Units {
		if !u.Alive {
			continue
		}
		c := unitCenter(u)
		d := c.Dist(pos)
		reach := sp.Radius + 8
		if d > reach {
			continue
		}
		f := 1 - 0.75*(d/reach)
		sh := w.blastShield(pos, c, -1)
		dmg := sp.Damage * f * sh
		if dmg >= 1 {
			dirv := c.Sub(pos)
			if dirv.Len() < 1 {
				dirv = Vec{0, -1}
			}
			dirv = dirv.Mul(1 / dirv.Len())
			imp := math.Min(380, (sp.Damage*f*sh)*3.2+40)
			w.knock(u, dirv.Mul(imp).Add(Vec{0, -imp * 0.35}))
			w.damageUnit(u, dmg, sp.Owner)
		}
	}
	// structures
	for _, s := range w.Structs {
		if !s.Alive {
			continue
		}
		x0, y0, x1, y1 := w.StructRect(s)
		nx := clamp(pos.X, x0, x1)
		ny := clamp(pos.Y, y0, y1)
		d := math.Hypot(pos.X-nx, pos.Y-ny)
		if d > sp.Radius {
			continue
		}
		f := 1 - 0.75*(d/sp.Radius)
		def := w.Cfg.S(s.Def)
		sh := w.blastShield(pos, w.StructCenter(s), s.ID)
		red := def.Blast * (1 - sp.Pierce)
		dmg := sp.Damage * f * sp.BlockMul * (1 - red) * (0.5 + 0.5*sh)
		w.damageStruct(s, dmg, sp.Owner)
	}
	// mines nearby detonate
	for _, p := range w.Projs {
		if p.Alive && p.Kind == PMine && p.Pos.Dist(pos) < sp.Radius*0.6 {
			p.Armed = 0
			p.Fuse = -1
		}
	}
	w.Terr.Carve(int(pos.X), int(pos.Y), int(sp.Radius*sp.Crater), true)
}

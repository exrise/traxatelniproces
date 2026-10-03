package sim

import (
	"math"
	"sort"

	"svinovoyna/internal/balance"
)

// Plan is one player's action for simultaneous mode.
type Plan struct {
	Player  int
	Done    bool
	Unit    int
	Struct  int
	Weapon  string
	Angle   float64
	Power   float64
	TargetX float64
	Walk    float64 // pixels to walk before firing (signed)
}

// Step advances the simulation by one fixed tick.
func (w *World) Step() {
	w.Tick++
	w.Time += Dt
	switch w.Phase {
	case PhaseBuild:
		w.stepBuild()
	case PhaseBattle:
		w.stepBattle()
	}
	w.settleStructs()
	w.stepUnits()
	w.stepProjs()
}

func (w *World) startBattle() {
	w.Phase = PhaseBattle
	w.BattleNo++
	w.Round = 1
	w.BuildTimerOn = false
	w.Projs, w.Spawns = nil, nil
	for _, s := range w.Structs {
		if !s.Alive {
			continue
		}
		d := w.Cfg.S(s.Def)
		s.AAAmmo = d.AAAmmo
		if d.Kind == balance.SWeapon {
			s.Ammo = w.Cfg.W(d.Weapon).Ammo
		}
	}
	for _, u := range w.Units {
		u.Used = map[string]int{}
		u.Walk = 0
	}
	for _, p := range w.Players {
		p.Earned = 0
		p.LossValue = 0
		p.DamageDone = 0
		p.Kills = 0
		p.Ready = false
	}
	w.emit(Event{Type: EvPhase, A: int(PhaseBattle), B: w.BattleNo})
	w.buildOrder()
	w.OrderPos = -1
	if w.Cfg.TurnMode == balance.TurnSimultaneous {
		w.beginPlan()
		return
	}
	w.nextTurn()
}

func (w *World) buildOrder() {
	var alive []int
	for _, p := range w.Players {
		if w.PlayerAlive(p.ID) {
			alive = append(alive, p.ID)
		}
	}
	sort.Ints(alive)
	w.Order = nil
	if len(alive) == 0 {
		return
	}
	shift := (w.BattleNo - 1 + w.Round - 1) % len(alive)
	for i := range alive {
		w.Order = append(w.Order, alive[(i+shift)%len(alive)])
	}
}

func (w *World) refreshElim() {
	for _, p := range w.Players {
		if !p.Elim && !w.PlayerAlive(p.ID) {
			p.Elim = true
			w.msg("%s выбывает из игры", p.Name)
			// everything the player owned collapses with the HQ
			for _, u := range w.Units {
				if u.Alive && u.Owner == p.ID {
					w.killUnit(u, -1, "")
				}
			}
			for _, s := range w.Structs {
				if s.Alive && s.Owner == p.ID {
					w.destroyStruct(s, -1)
				}
			}
		}
	}
}

func (w *World) checkWinner() bool {
	w.refreshElim()
	n, last := w.AliveTeams()
	if n <= 1 {
		w.Phase = PhaseOver
		w.Cur = -1
		w.Winner = last
		if n == 0 {
			w.Winner = -2
			w.WinnerName = "Ничья"
		} else {
			names := ""
			for _, p := range w.Players {
				if p.Team == last {
					if names != "" {
						names += " + "
					}
					names += p.Name
				}
			}
			w.WinnerName = names
		}
		w.emit(Event{Type: EvWinner, A: w.Winner, Text: w.WinnerName})
		return true
	}
	return false
}

func (w *World) weather() {
	w.Wind = w.RNG.Range(-w.Cfg.WindMax, w.Cfg.WindMax)
}

func (w *World) payIncome(pid int) {
	p := w.Players[pid]
	total := 0
	for _, s := range w.Structs {
		if s.Alive && s.Owner == pid {
			total += w.Cfg.S(s.Def).Income
		}
	}
	for _, c := range w.Points {
		if c.Owner >= 0 && w.Players[c.Owner].Team == p.Team && c.Owner == pid {
			total += w.Cfg.CaptureIncome
		}
	}
	if total > 0 {
		p.Money += total
		p.Earned += total
		w.emit(Event{Type: EvMoney, Pos: w.hqPos(pid), A: pid, B: total})
	}
}

func (w *World) hqPos(pid int) Vec {
	p := w.Players[pid]
	if p.HQ >= 0 {
		return w.StructCenter(w.Structs[p.HQ])
	}
	return Vec{float64(p.ZoneX0+p.ZoneX1) / 2, PlateauY - 40}
}

func (w *World) nextTurn() {
	w.refreshElim()
	if w.checkWinner() {
		return
	}
	for {
		w.OrderPos++
		if w.OrderPos >= len(w.Order) {
			w.Round++
			if w.Round > w.Cfg.RoundsPerBattle {
				w.endBattle()
				return
			}
			w.buildOrder()
			w.OrderPos = 0
			if len(w.Order) == 0 {
				w.endBattle()
				return
			}
		}
		if w.PlayerAlive(w.Order[w.OrderPos]) {
			break
		}
	}
	pid := w.Order[w.OrderPos]
	w.Cur = pid
	w.weather()
	w.updateLeaders()
	w.Stage = StageActive
	w.TurnTimer = w.Cfg.TurnTime
	w.UnitActed, w.StructActed = false, false
	w.FiredUnit = -1
	w.SelStruct = -1
	w.SelUnit = w.pickUnit(pid)
	w.payIncome(pid)
	w.emit(Event{Type: EvTurn, A: pid, B: w.Round})
}

// pickUnit chooses the next unit of the player in rotation.
func (w *World) pickUnit(pid int) int {
	us := w.UnitsOf(pid)
	if len(us) == 0 {
		return -1
	}
	last := w.LastUnit[pid]
	for _, u := range us {
		if u.ID > last {
			return u.ID
		}
	}
	return us[0].ID
}

func (w *World) endBattle() {
	w.Cur = -1
	w.refreshElim()
	if w.checkWinner() {
		return
	}
	for _, p := range w.Players {
		for _, c := range w.Points {
			_ = c
		}
		_ = p
	}
	w.emit(Event{Type: EvPhase, A: int(PhaseBuild), B: w.BuildNo + 1})
	w.startBuild()
}

func (w *World) captureCheck(pid int) {
	for i, c := range w.Points {
		mine, theirs := false, false
		for _, u := range w.Units {
			if !u.Alive {
				continue
			}
			if math.Abs(u.Pos.X-c.Pos.X) < 90 && math.Abs(u.Pos.Y-c.Pos.Y) < 90 {
				if u.Owner == pid {
					mine = true
				} else if w.Hostile(pid, u.Owner) {
					theirs = true
				}
			}
		}
		if mine && !theirs && c.Owner != pid {
			c.Owner = pid
			w.emit(Event{Type: EvCapture, Pos: c.Pos, A: pid, B: i})
		}
	}
}

func (w *World) endTurn() {
	if w.Stage == StageSettle {
		return
	}
	w.Stage = StageSettle
	w.SettleT = 0
	for _, u := range w.Units {
		u.Walk = 0
	}
}

func (w *World) stepBattle() {
	if w.Cfg.TurnMode == balance.TurnSimultaneous {
		w.stepSimultaneous()
		return
	}
	if w.Cur < 0 {
		return
	}
	switch w.Stage {
	case StageActive:
		w.TurnTimer -= Dt
		if w.TurnTimer <= 0 {
			w.endTurn()
		}
		if len(w.UnitsOf(w.Cur)) == 0 && !w.hasWeaponStruct(w.Cur) && !w.hasItems(w.Cur) {
			w.endTurn()
		}
	case StageRetreat:
		w.RetreatTime -= Dt
		if w.RetreatTime <= 0 {
			w.endTurn()
		}
	case StageSettle:
		w.SettleT += Dt
		quiet := !w.ProjectilesBusy() && w.UnitsSettled() && w.StructsSettled()
		if (quiet && w.SettleT > 0.8) || w.SettleT > 40 {
			w.finishTurn()
		}
	}
}

func (w *World) finishTurn() {
	pid := w.Cur
	if pid >= 0 {
		if w.SelUnit >= 0 {
			w.LastUnit[pid] = w.SelUnit
		}
		w.captureCheck(pid)
	}
	w.nextTurn()
}

func (w *World) hasWeaponStruct(pid int) bool {
	for _, s := range w.Structs {
		if s.Alive && s.Owner == pid && w.Cfg.S(s.Def).Kind == balance.SWeapon && s.Ammo > 0 {
			return true
		}
	}
	return false
}

func (w *World) hasItems(pid int) bool {
	for _, n := range w.Players[pid].Items {
		if n > 0 {
			return true
		}
	}
	return false
}

// ---- simultaneous mode ------------------------------------------------------

func (w *World) beginPlan() {
	w.Stage = StagePlan
	w.Cur = -1
	w.TurnTimer = w.Cfg.PlanTime
	w.Plans = map[int]*Plan{}
	w.weather()
	w.updateLeaders()
	for _, p := range w.Players {
		if w.PlayerAlive(p.ID) {
			w.payIncome(p.ID)
		}
	}
	w.emit(Event{Type: EvTurn, A: -1, B: w.Round})
}

func (w *World) stepSimultaneous() {
	switch w.Stage {
	case StagePlan:
		w.TurnTimer -= Dt
		all := true
		for _, p := range w.Players {
			if w.PlayerAlive(p.ID) {
				if pl := w.Plans[p.ID]; pl == nil || !pl.Done {
					all = false
				}
			}
		}
		if all || w.TurnTimer <= 0 {
			w.resolvePlans()
		}
	case StageResolve:
		w.SettleT += Dt
		quiet := !w.ProjectilesBusy() && w.UnitsSettled() && w.StructsSettled()
		if (quiet && w.SettleT > 1.0) || w.SettleT > 40 {
			for _, p := range w.Players {
				if w.PlayerAlive(p.ID) {
					w.captureCheck(p.ID)
				}
			}
			if w.checkWinner() {
				return
			}
			w.Round++
			if w.Round > w.Cfg.RoundsPerBattle {
				w.endBattle()
				return
			}
			w.beginPlan()
		}
	}
}

func (w *World) resolvePlans() {
	w.Stage = StageResolve
	w.SettleT = 0
	ids := make([]int, 0, len(w.Plans))
	for id := range w.Plans {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		pl := w.Plans[id]
		if !pl.Done || !w.PlayerAlive(id) {
			continue
		}
		sp := FireSpec{Player: id, Unit: -1, Struct: -1, Weapon: pl.Weapon, Angle: pl.Angle, Power: pl.Power, TargetX: pl.TargetX}
		if pl.Unit >= 0 && pl.Unit < len(w.Units) {
			u := w.Units[pl.Unit]
			if !u.Alive || u.Owner != id {
				continue
			}
			if pl.Walk != 0 {
				w.walkUnit(u, math.Max(-120, math.Min(120, pl.Walk)))
				u.Anim += 1
			}
			sp.Unit = u.ID
			if !w.unitHasWeapon(u, pl.Weapon) || !w.useUnitAmmo(u, pl.Weapon) {
				continue
			}
		} else if pl.Struct >= 0 && pl.Struct < len(w.Structs) {
			s := w.Structs[pl.Struct]
			if !s.Alive || s.Owner != id || w.Cfg.S(s.Def).Kind != balance.SWeapon || s.Ammo <= 0 {
				continue
			}
			sp.Struct = s.ID
			sp.Weapon = w.Cfg.S(s.Def).Weapon
			s.Ammo--
		} else if w.IsItem(pl.Weapon) && w.Players[id].Items[pl.Weapon] > 0 && w.HasHQ(id) {
			w.Players[id].Items[pl.Weapon]--
		} else {
			continue
		}
		_ = w.launch(sp)
	}
}

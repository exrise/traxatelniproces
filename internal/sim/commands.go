package sim

import (
	"math"

	"svinovoyna/internal/balance"
)

// CmdType enumerates all player commands.
type CmdType int

const (
	// build phase
	CmdPlaceStruct CmdType = iota
	CmdPlaceUnit
	CmdMoveUnit
	CmdSell
	CmdBuyItem
	CmdSellItem
	CmdRepair
	CmdReady
	// battle
	CmdSelectUnit
	CmdSelectStruct
	CmdWalk
	CmdJump
	CmdAim
	CmdFire
	CmdEndTurn
	CmdSteer
	CmdDetonate
	CmdPlan
)

// Command is one player intent. The same type is used by UI, bots and the network.
type Command struct {
	Player int
	Type   CmdType
	ID     int
	Def    string
	CX, CY int
	X, Y   float64
	Angle  float64
	Power  float64
	Dir    int
	Flag   bool
	Weapon string
}

func (w *World) unitHasWeapon(u *Unit, id string) bool {
	for _, x := range w.Cfg.U(u.Def).Weapons {
		if x == id {
			return true
		}
	}
	return false
}

// useUnitAmmo consumes a use of a limited weapon; false when exhausted.
func (w *World) useUnitAmmo(u *Unit, id string) bool {
	wd := w.Cfg.W(id)
	if wd == nil {
		return false
	}
	if wd.Ammo > 0 {
		if u.Used[id] >= wd.Ammo {
			return false
		}
		u.Used[id]++
	}
	return true
}

// UnitAmmoLeft returns remaining uses (-1 = unlimited).
func (w *World) UnitAmmoLeft(u *Unit, id string) int {
	wd := w.Cfg.W(id)
	if wd == nil || wd.Ammo <= 0 {
		return -1
	}
	return wd.Ammo - u.Used[id]
}

func (w *World) myTurn(pid int) bool {
	return w.Phase == PhaseBattle && w.Cur == pid && (w.Stage == StageActive || w.Stage == StageRetreat)
}

// Apply validates and executes a command.
func (w *World) Apply(c Command) error {
	if c.Player < 0 || c.Player >= len(w.Players) {
		return ErrUnknown
	}
	pid := c.Player
	switch c.Type {
	case CmdPlaceStruct:
		return w.PlaceStruct(pid, c.Def, c.CX, c.CY)
	case CmdPlaceUnit:
		return w.PlaceUnit(pid, c.Def, c.X, c.Y)
	case CmdMoveUnit:
		return w.MoveUnitBuild(pid, c.ID, c.X, c.Y)
	case CmdSell:
		return w.Sell(pid, c.Flag, c.ID)
	case CmdBuyItem:
		return w.BuyItem(pid, c.Def)
	case CmdSellItem:
		return w.SellItem(pid, c.Def)
	case CmdRepair:
		return w.Repair(pid, c.ID)
	case CmdReady:
		return w.SetReady(pid, c.Flag)
	}

	if w.Phase != PhaseBattle {
		return ErrPhase
	}
	sim := w.Cfg.TurnMode == balance.TurnSimultaneous

	switch c.Type {
	case CmdSteer:
		for _, p := range w.Projs {
			if p.Alive && p.Kind == PDrone && p.Owner == pid {
				p.Steer = c.Angle
			}
		}
		return nil
	case CmdDetonate:
		for _, p := range w.Projs {
			if p.Alive && p.Kind == PDrone && p.Owner == pid {
				w.detonate(p, p.Pos)
			}
		}
		return nil
	case CmdPlan:
		if !sim || w.Stage != StagePlan || !w.PlayerAlive(pid) {
			return ErrTurn
		}
		pl := &Plan{Player: pid, Done: c.Flag, Unit: -1, Struct: -1, Weapon: c.Weapon, Angle: c.Angle, Power: c.Power, TargetX: c.X, Walk: c.Y}
		if c.Def == "struct" {
			pl.Struct = c.ID
		} else if c.Def == "unit" {
			pl.Unit = c.ID
		}
		w.Plans[pid] = pl
		return nil
	}

	if sim {
		return ErrTurn
	}
	if !w.myTurn(pid) {
		return ErrTurn
	}
	switch c.Type {
	case CmdSelectUnit:
		if w.Stage != StageActive || w.UnitActed && w.Cfg.TurnMode == balance.TurnClassic || w.StructActed && w.Cfg.TurnMode == balance.TurnClassic {
			return ErrAction
		}
		if c.ID < 0 || c.ID >= len(w.Units) || !w.Units[c.ID].Alive || w.Units[c.ID].Owner != pid {
			return ErrUnknown
		}
		w.clearWalk()
		w.SelUnit, w.SelStruct = c.ID, -1
		return nil
	case CmdSelectStruct:
		if w.Stage != StageActive {
			return ErrAction
		}
		if c.ID < 0 || c.ID >= len(w.Structs) || !w.Structs[c.ID].Alive || w.Structs[c.ID].Owner != pid ||
			w.Cfg.S(w.Structs[c.ID].Def).Kind != balance.SWeapon {
			return ErrUnknown
		}
		if w.Cfg.TurnMode == balance.TurnClassic && (w.UnitActed || w.StructActed) {
			return ErrAction
		}
		w.clearWalk()
		w.SelStruct = c.ID
		return nil
	case CmdWalk:
		if w.SelUnit < 0 {
			return nil
		}
		u := w.Units[w.SelUnit]
		if u.Alive && u.Owner == pid {
			u.Walk = c.Dir
		}
		return nil
	case CmdJump:
		if w.SelUnit < 0 {
			return nil
		}
		u := w.Units[w.SelUnit]
		if u.Alive && u.Owner == pid && w.unitCanAct(u) {
			w.jumpUnit(u)
		}
		return nil
	case CmdAim:
		if w.SelStruct >= 0 {
			w.Structs[w.SelStruct].Aim = ClampStructAim(c.Angle)
		} else if w.SelUnit >= 0 {
			u := w.Units[w.SelUnit]
			u.Aim = c.Angle
			if w.unitCanAct(u) {
				if math.Cos(c.Angle) >= 0 {
					u.Face = 1
				} else {
					u.Face = -1
				}
			}
		}
		return nil
	case CmdEndTurn:
		w.endTurn()
		return nil
	case CmdFire:
		return w.fireAction(pid, c)
	}
	return ErrUnknown
}

func (w *World) clearWalk() {
	for _, u := range w.Units {
		u.Walk = 0
	}
}

func (w *World) fireAction(pid int, c Command) error {
	if w.Stage != StageActive {
		return ErrAction
	}
	mode := w.Cfg.TurnMode
	spec := FireSpec{Player: pid, Unit: -1, Struct: -1, Weapon: c.Weapon, Angle: c.Angle, Power: c.Power, TargetX: c.X}
	var structAction bool
	switch {
	case w.IsItem(c.Weapon):
		if !w.HasHQ(pid) || w.Players[pid].Items[c.Weapon] <= 0 {
			return ErrAction
		}
		structAction = true
	case w.SelStruct >= 0:
		s := w.Structs[w.SelStruct]
		d := w.Cfg.S(s.Def)
		if !s.Alive || s.Owner != pid || d.Kind != balance.SWeapon || s.Ammo <= 0 {
			return ErrAction
		}
		spec.Struct = s.ID
		spec.Weapon = d.Weapon
		structAction = true
	case w.SelUnit >= 0:
		u := w.Units[w.SelUnit]
		if !u.Alive || u.Owner != pid || !w.unitHasWeapon(u, c.Weapon) {
			return ErrAction
		}
		if w.UnitAmmoLeft(u, c.Weapon) == 0 {
			return ErrAction
		}
		spec.Unit = u.ID
	default:
		return ErrAction
	}
	// action budget
	if structAction {
		if w.StructActed || (mode == balance.TurnClassic && w.UnitActed) {
			return ErrAction
		}
	} else if w.UnitActed || (mode == balance.TurnClassic && w.StructActed) {
		return ErrAction
	}
	wd := w.Cfg.W(spec.Weapon)
	if wd == nil {
		return ErrUnknown
	}
	// consume
	switch {
	case w.IsItem(c.Weapon):
		w.Players[pid].Items[c.Weapon]--
	case spec.Struct >= 0:
		w.Structs[spec.Struct].Ammo--
	default:
		w.useUnitAmmo(w.Units[spec.Unit], spec.Weapon)
	}
	if err := w.launch(spec); err != nil {
		return err
	}
	if structAction {
		w.StructActed = true
	} else {
		w.UnitActed = true
		w.FiredUnit = spec.Unit
		w.Units[spec.Unit].Spent = true
	}
	w.clearWalkExcept(w.FiredUnit)
	done := w.StructActed || w.UnitActed
	if mode == balance.TurnUnitAndStruct {
		done = w.StructActed && w.UnitActed
		if !done {
			w.TurnTimer = math.Min(w.TurnTimer, 20)
		}
	}
	if done {
		if w.FiredUnit >= 0 {
			w.Stage = StageRetreat
			w.RetreatTime = w.Cfg.RetreatTime
		} else {
			w.endTurn()
		}
	}
	return nil
}

func (w *World) clearWalkExcept(keep int) {
	for _, u := range w.Units {
		if u.ID != keep {
			u.Walk = 0
		}
	}
}

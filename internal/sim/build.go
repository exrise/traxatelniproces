package sim

import (
	"errors"
	"fmt"
	"math"

	"svinovoyna/internal/balance"
)

var (
	ErrPhase   = errors.New("не та фаза игры")
	ErrMoney   = errors.New("не хватает денег")
	ErrZone    = errors.New("за пределами твоей зоны")
	ErrBlocked = errors.New("место занято")
	ErrSupport = errors.New("нет опоры")
	ErrLimit   = errors.New("достигнут лимит")
	ErrTier    = errors.New("пока недоступно")
	ErrNoHQ    = errors.New("нет штаба")
	ErrUnknown = errors.New("неизвестный объект")
	ErrTurn    = errors.New("не твой ход")
	ErrAction  = errors.New("действие недоступно")
	ErrActed   = errors.New("в этот ход выстрел уже сделан — можно только отступать")
)

func (w *World) startBuild() {
	w.Phase = PhaseBuild
	w.BuildNo++
	w.Projs = nil
	w.Spawns = nil
	w.Shots = nil
	w.Cur = -1
	w.SelUnit, w.SelStruct = -1, -1
	for _, p := range w.Players {
		p.LastEarned = p.Earned
		if w.BuildNo > 1 && !p.Elim {
			p.Money += w.Cfg.BaseIncome
			cu := int(float64(p.LossValue) * w.Cfg.CatchUpPct)
			if cu > w.Cfg.CatchUpCap {
				cu = w.Cfg.CatchUpCap
			}
			p.Money += cu
		}
		p.LossValue = 0
		p.Ready = !w.PlayerAlive(p.ID) || !w.HasHQ(p.ID)
		for _, u := range w.UnitsOf(p.ID) {
			u.HP = math.Min(u.MaxHP, u.HP+w.Cfg.BuildHeal)
			u.Used = map[string]int{}
			u.Walk = 0
		}
	}
	w.BuildTimerOn = true
	if w.BuildNo == 1 {
		w.BuildTimer = w.Cfg.BuildTimeFirst
	} else {
		w.BuildTimer = w.Cfg.BuildTime
	}
	w.emit(Event{Type: EvPhase, A: int(PhaseBuild), B: w.BuildNo})
}

func (w *World) countStructs(owner int, def string) int {
	n := 0
	for _, s := range w.Structs {
		if s.Alive && s.Owner == owner && s.Def == def {
			n++
		}
	}
	return n
}

func (w *World) countUnits(owner int, def string) int {
	n := 0
	for _, u := range w.Units {
		if u.Alive && u.Owner == owner && (def == "" || u.Def == def) {
			n++
		}
	}
	return n
}

func (w *World) buildCheck(pid int) (*Player, error) {
	if w.Phase != PhaseBuild {
		return nil, ErrPhase
	}
	p := w.Players[pid]
	if p.Ready || !w.HasHQ(pid) {
		return nil, ErrPhase
	}
	return p, nil
}

// PlaceStruct puts a structure on the grid.
func (w *World) PlaceStruct(pid int, def string, cx, cy int) error {
	if err := w.CanPlaceStruct(pid, def, cx, cy); err != nil {
		return err
	}
	return w.placeStruct(pid, def, cx, cy)
}

// CanPlaceStruct validates a placement without changing anything.
func (w *World) CanPlaceStruct(pid int, def string, cx, cy int) error {
	p, err := w.buildCheck(pid)
	if err != nil {
		return err
	}
	d := w.Cfg.S(def)
	if d == nil || d.Kind == balance.SHQ {
		return ErrUnknown
	}
	if d.Tier > w.BuildNo {
		return ErrTier
	}
	if d.Max > 0 && w.countStructs(pid, def) >= d.Max {
		return ErrLimit
	}
	if p.Money < d.Cost {
		return ErrMoney
	}
	if cx*Cell < p.ZoneX0 || (cx+d.W)*Cell > p.ZoneX1 {
		return ErrZone
	}
	for y := 0; y < d.H; y++ {
		for x := 0; x < d.W; x++ {
			if !w.cellFree(cx+x, cy+y, -1) {
				return ErrBlocked
			}
		}
	}
	if !w.supportedAt(cx, cy, d.W, d.H, -1) {
		return ErrSupport
	}
	return nil
}

func (w *World) placeStruct(pid int, def string, cx, cy int) error {
	p := w.Players[pid]
	d := w.Cfg.S(def)
	p.Money -= d.Cost
	p.Spent += d.Cost
	s := w.addStruct(pid, def, cx, cy)
	// weapon structures face the nearest enemy side by default
	if d.Kind == balance.SWeapon || d.Kind == balance.SAA {
		if float64(cx*Cell) > float64(p.ZoneX0+p.ZoneX1)/2 {
			s.Aim = -math.Pi + math.Pi/3
		}
	}
	w.emit(Event{Type: EvBuild, Pos: w.StructCenter(s), A: pid})
	return nil
}

// dropUnitPos finds the resting position for a unit dropped from (x,y).
func (w *World) dropUnitPos(x, y float64) (Vec, bool) {
	if !w.unitBodyFree(x, y) {
		// maybe the click was inside the ground: look upward for free air
		found := false
		for dy := 0.0; dy < 80; dy += 2 {
			if w.unitBodyFree(x, y-dy) {
				y -= dy
				found = true
				break
			}
		}
		if !found {
			return Vec{}, false
		}
	}
	for i := 0; i < 900 && !w.unitOnGround(x, y); i++ {
		y++
		if y > WaterY {
			return Vec{}, false
		}
	}
	return Vec{x, y}, true
}

// PlaceUnit buys a new unit.
func (w *World) PlaceUnit(pid int, def string, x, y float64) error {
	p, err := w.buildCheck(pid)
	if err != nil {
		return err
	}
	d := w.Cfg.U(def)
	if d == nil {
		return ErrUnknown
	}
	if d.Max > 0 && w.countUnits(pid, def) >= d.Max {
		return ErrLimit
	}
	if w.countUnits(pid, "") >= w.Cfg.MaxUnits {
		return ErrLimit
	}
	if p.Money < d.Cost {
		return ErrMoney
	}
	if x-UnitW/2 < float64(p.ZoneX0) || x+UnitW/2 > float64(p.ZoneX1) {
		return ErrZone
	}
	pos, ok := w.dropUnitPos(x, y)
	if !ok {
		return ErrBlocked
	}
	p.Money -= d.Cost
	p.Spent += d.Cost
	u := w.addUnit(pid, def, pos)
	w.emit(Event{Type: EvBuild, Pos: u.Pos, A: pid})
	return nil
}

// MoveUnitBuild repositions an existing unit during build (free).
func (w *World) MoveUnitBuild(pid, uid int, x, y float64) error {
	p, err := w.buildCheck(pid)
	if err != nil {
		return err
	}
	if uid < 0 || uid >= len(w.Units) || w.Units[uid].Owner != pid || !w.Units[uid].Alive {
		return ErrUnknown
	}
	if x-UnitW/2 < float64(p.ZoneX0) || x+UnitW/2 > float64(p.ZoneX1) {
		return ErrZone
	}
	pos, ok := w.dropUnitPos(x, y)
	if !ok {
		return ErrBlocked
	}
	w.Units[uid].Pos = pos
	w.Units[uid].Vel = Vec{}
	return nil
}

// BuyItem purchases a consumable (airstrike, drone).
func (w *World) BuyItem(pid int, id string) error {
	p, err := w.buildCheck(pid)
	if err != nil {
		return err
	}
	wd := w.Cfg.W(id)
	if wd == nil || wd.Cost <= 0 {
		return ErrUnknown
	}
	if p.Money < wd.Cost {
		return ErrMoney
	}
	p.Money -= wd.Cost
	p.Spent += wd.Cost
	p.Items[id]++
	return nil
}

// SellItem refunds a consumable.
func (w *World) SellItem(pid int, id string) error {
	p, err := w.buildCheck(pid)
	if err != nil {
		return err
	}
	wd := w.Cfg.W(id)
	if wd == nil || p.Items[id] <= 0 {
		return ErrUnknown
	}
	p.Items[id]--
	p.Money += int(float64(wd.Cost) * w.Cfg.SellRefund)
	return nil
}

// Sell removes a structure or unit for a partial refund.
func (w *World) Sell(pid int, isUnit bool, id int) error {
	p, err := w.buildCheck(pid)
	if err != nil {
		return err
	}
	if isUnit {
		if id < 0 || id >= len(w.Units) || w.Units[id].Owner != pid || !w.Units[id].Alive {
			return ErrUnknown
		}
		u := w.Units[id]
		u.Alive = false
		p.Money += int(float64(w.Cfg.U(u.Def).Cost) * w.Cfg.SellRefund)
		return nil
	}
	if id < 0 || id >= len(w.Structs) || w.Structs[id].Owner != pid || !w.Structs[id].Alive {
		return ErrUnknown
	}
	s := w.Structs[id]
	if s.Def == "hq" {
		return ErrAction
	}
	d := w.Cfg.S(s.Def)
	p.Money += int(float64(d.Cost) * w.Cfg.SellRefund * s.HP / s.MaxHP)
	w.removeStruct(s)
	return nil
}

func (w *World) removeStruct(s *Struct) {
	s.Alive = false
	w.setOcc(s, 0)
}

// RepairCost returns the money needed to fully repair a structure.
func (w *World) RepairCost(s *Struct) int {
	d := w.Cfg.S(s.Def)
	base := float64(d.Cost)
	if d.Kind == balance.SHQ {
		base = w.Cfg.MaxHQHP * 1.5
	}
	return int(math.Ceil(base * w.Cfg.RepairCostPct * (1 - s.HP/s.MaxHP)))
}

// Repair restores a structure to full HP for money.
func (w *World) Repair(pid, sid int) error {
	p, err := w.buildCheck(pid)
	if err != nil {
		return err
	}
	if sid < 0 || sid >= len(w.Structs) || w.Structs[sid].Owner != pid || !w.Structs[sid].Alive {
		return ErrUnknown
	}
	s := w.Structs[sid]
	c := w.RepairCost(s)
	if c <= 0 {
		return nil
	}
	if p.Money < c {
		return ErrMoney
	}
	p.Money -= c
	p.Spent += c
	s.HP = s.MaxHP
	return nil
}

// SetReady marks a player as done building; when everyone is ready the battle starts.
func (w *World) SetReady(pid int, ready bool) error {
	if w.Phase != PhaseBuild {
		return ErrPhase
	}
	w.Players[pid].Ready = ready
	return nil
}

func (w *World) stepBuild() {
	w.settleStructs()
	all := true
	for _, p := range w.Players {
		if !p.Ready {
			all = false
		}
	}
	if w.BuildTimerOn {
		w.BuildTimer -= Dt
		if w.BuildTimer <= 0 {
			all = true
		}
	}
	if all {
		w.startBattle()
	}
}

// StatusLine is used by logs/tests.
func (w *World) StatusLine() string {
	return fmt.Sprintf("phase=%d build=%d battle=%d round=%d t=%.1f", w.Phase, w.BuildNo, w.BattleNo, w.Round, w.Time)
}

// DropUnitPos exposes the unit placement helper to the UI (ghost preview).
func (w *World) DropUnitPos(x, y float64) (Vec, bool) { return w.dropUnitPos(x, y) }

// CanPlaceUnit validates buying a unit at (x,y) without changing anything.
func (w *World) CanPlaceUnit(pid int, def string, x, y float64) (Vec, error) {
	p, err := w.buildCheck(pid)
	if err != nil {
		return Vec{}, err
	}
	d := w.Cfg.U(def)
	if d == nil {
		return Vec{}, ErrUnknown
	}
	if d.Max > 0 && w.countUnits(pid, def) >= d.Max {
		return Vec{}, ErrLimit
	}
	if w.countUnits(pid, "") >= w.Cfg.MaxUnits {
		return Vec{}, ErrLimit
	}
	if p.Money < d.Cost {
		return Vec{}, ErrMoney
	}
	if x-UnitW/2 < float64(p.ZoneX0) || x+UnitW/2 > float64(p.ZoneX1) {
		return Vec{}, ErrZone
	}
	pos, ok := w.dropUnitPos(x, y)
	if !ok {
		return Vec{}, ErrBlocked
	}
	return pos, nil
}

// CountUnits counts alive units of a player (def "" = all).
func (w *World) CountUnits(owner int, def string) int { return w.countUnits(owner, def) }

package netplay

import (
	"fmt"
	"hash/fnv"

	"svinovoyna/internal/sim"
)

// SendState remembers what a given client already knows.
type SendState struct {
	hPlayers, hUnits, hStructs, hPoints uint64
	craters                             int
	sentOnce                            bool
}

func hashSlice[T any](items []*T) uint64 {
	h := fnv.New64a()
	for _, it := range items {
		fmt.Fprintf(h, "%+v|", *it)
	}
	return h.Sum64()
}

// Build creates the snapshot of w for the player in `seat`, sending only what changed.
// events are the simulation events produced since the last snapshot.
func Build(w *sim.World, seat int, st *SendState, events []sim.Event) *Snapshot {
	s := &Snapshot{
		Tick: w.Tick, Time: w.Time, Phase: w.Phase, BuildNo: w.BuildNo, BattleNo: w.BattleNo, Round: w.Round,
		Wind: w.Wind, Cur: w.Cur, Stage: w.Stage, TurnTimer: w.TurnTimer, RetreatTime: w.RetreatTime,
		BuildTimer: w.BuildTimer, BuildTimerOn: w.BuildTimerOn, SelUnit: w.SelUnit, SelStruct: w.SelStruct,
		UnitActed: w.UnitActed, StructActed: w.StructActed, FiredUnit: w.FiredUnit, SettleT: w.SettleT,
		Order: w.Order, OrderPos: w.OrderPos, Leader: w.Leader, Underdog: w.Underdog, Winner: w.Winner, WinnerName: w.WinnerName,
	}
	fog := w.Phase == sim.PhaseBuild

	players := make([]*sim.Player, len(w.Players))
	for i, p := range w.Players {
		cp := *p
		if fog && i != seat {
			cp.Items = nil
		}
		players[i] = &cp
	}
	units := make([]*sim.Unit, len(w.Units))
	for i, u := range w.Units {
		if fog && u.Owner != seat {
			units[i] = &sim.Unit{ID: i, Owner: u.Owner}
			continue
		}
		units[i] = u
	}
	structs := make([]*sim.Struct, len(w.Structs))
	for i, x := range w.Structs {
		if fog && x.Owner != seat && x.Def != "hq" {
			structs[i] = &sim.Struct{ID: i, Owner: x.Owner, Def: x.Def}
			continue
		}
		structs[i] = x
	}
	if h := hashSlice(players); h != st.hPlayers || !st.sentOnce {
		st.hPlayers = h
		s.Players = &PlayerList{OK: true, L: players}
	}
	if h := hashSlice(units); h != st.hUnits || !st.sentOnce {
		st.hUnits = h
		s.Units = &UnitList{OK: true, L: units}
	}
	if h := hashSlice(structs); h != st.hStructs || !st.sentOnce {
		st.hStructs = h
		s.Structs = &StructList{OK: true, L: structs}
	}
	if h := hashSlice(w.Points); h != st.hPoints || !st.sentOnce {
		st.hPoints = h
		s.Points = &PointList{OK: true, L: w.Points}
	}
	if !fog {
		s.Projs = w.Projs
	}
	s.CraterBase = st.craters
	if st.craters < len(w.Terr.Craters) {
		s.Craters = w.Terr.Craters[st.craters:]
		st.craters = len(w.Terr.Craters)
	}
	for _, e := range events {
		if fog && !(e.A == seat && (e.Type == sim.EvBuild || e.Type == sim.EvMoney)) && e.Type != sim.EvPhase && e.Type != sim.EvMessage {
			continue
		}
		s.Events = append(s.Events, e)
	}
	st.sentOnce = true
	return s
}

// Apply writes a snapshot into a client-side replica world.
func Apply(w *sim.World, s *Snapshot) {
	w.Tick, w.Time, w.Phase = s.Tick, s.Time, s.Phase
	w.BuildNo, w.BattleNo, w.Round = s.BuildNo, s.BattleNo, s.Round
	w.Wind, w.Cur, w.Stage = s.Wind, s.Cur, s.Stage
	w.TurnTimer, w.RetreatTime, w.BuildTimer, w.BuildTimerOn = s.TurnTimer, s.RetreatTime, s.BuildTimer, s.BuildTimerOn
	w.SelUnit, w.SelStruct, w.UnitActed, w.StructActed, w.FiredUnit = s.SelUnit, s.SelStruct, s.UnitActed, s.StructActed, s.FiredUnit
	w.SettleT = s.SettleT
	w.Order, w.OrderPos, w.Leader, w.Underdog = s.Order, s.OrderPos, s.Leader, s.Underdog
	w.Winner, w.WinnerName = s.Winner, s.WinnerName
	if s.Players != nil {
		for i, p := range s.Players.L {
			if i < len(w.Players) {
				if p.Items == nil {
					p.Items = map[string]int{}
				}
				*w.Players[i] = *p
			}
		}
	}
	if s.Units != nil {
		w.Units = s.Units.L
		for _, u := range w.Units {
			if u.Used == nil {
				u.Used = map[string]int{}
			}
		}
	}
	if s.Structs != nil {
		w.Structs = s.Structs.L
		w.RebuildOcc()
	}
	if s.Points != nil {
		w.Points = s.Points.L
	}
	w.Projs = s.Projs
	if s.CraterBase == len(w.Terr.Craters) {
		for _, c := range s.Craters {
			w.Terr.Carve(c.X, c.Y, c.R, true)
		}
	}
	w.Events = append(w.Events, s.Events...)
}

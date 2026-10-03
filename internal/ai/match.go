package ai

import (
	"svinovoyna/internal/balance"
	"svinovoyna/internal/sim"
)

// MatchResult summarises one headless bot match.
type MatchResult struct {
	Winner     int // player index, -1 draw / timeout
	Seats      []Style
	Seconds    float64
	Builds     int
	Finished   bool
	Money      []int
	DamageDone []float64
	Stats      map[string]sim.WStat
	HQLeft     []float64
}

// RunMatch plays a complete game between bots using the given strategies.
// maxSeconds limits the simulated time; unfinished games count as draws.
func RunMatch(cfg *balance.Config, seed uint64, styles []Style, skill float64, maxSeconds float64) MatchResult {
	return RunMatchLog(cfg, seed, styles, skill, maxSeconds, nil)
}

// RunMatchLog is RunMatch with an optional per-battle log callback.
func RunMatchLog(cfg *balance.Config, seed uint64, styles []Style, skill float64, maxSeconds float64, logf func(string, ...any)) MatchResult {
	var setups []sim.PlayerSetup
	for i := range styles {
		setups = append(setups, sim.PlayerSetup{Name: styles[i].String(), Color: i, Bot: true})
	}
	w := sim.NewWorld(cfg.Clone(), seed, setups)
	bots := make([]*Bot, len(styles))
	for i, st := range styles {
		bots[i] = New(i, st, skill, seed)
	}
	maxTicks := int(maxSeconds * sim.TickRate)
	last := w.Phase
	for w.Tick < maxTicks && w.Phase != sim.PhaseOver {
		if w.Phase == sim.PhaseBuild {
			for _, b := range bots {
				b.Build(w)
			}
		} else if w.Phase == sim.PhaseBattle {
			for _, b := range bots {
				for _, c := range b.Think(w) {
					_ = w.Apply(c)
				}
			}
		}
		w.Step()
		w.Events = w.Events[:0]
		if logf != nil && w.Phase != last {
			last = w.Phase
			if w.Phase == sim.PhaseBuild || w.Phase == sim.PhaseOver {
				logf("конец боя %d (t=%.0f с)", w.BattleNo, w.Time)
				for _, p := range w.Players {
					hq := w.Structs[p.HQ]
					logf("  %-16s $%-5d штаб %4.0f/%.0f юнитов %d построек %d урон %.0f убийств %d", styles[p.ID], p.Money, hq.HP, hq.MaxHP,
						len(w.UnitsOf(p.ID)), len(w.StructsOf(p.ID)), p.DamageDone, p.Kills)
				}
			}
		}
	}
	res := MatchResult{Winner: -1, Seats: styles, Seconds: w.Time, Builds: w.BuildNo, Finished: w.Phase == sim.PhaseOver}
	if w.Phase == sim.PhaseOver && w.Winner >= 0 {
		for _, p := range w.Players {
			if p.Team == w.Winner {
				res.Winner = p.ID
			}
		}
	}
	for _, p := range w.Players {
		res.Money = append(res.Money, p.Money)
		res.DamageDone = append(res.DamageDone, p.DamageDone)
	}
	collectStats(w, &res)
	return res
}

func collectStats(w *sim.World, res *MatchResult) {
	res.Stats = map[string]sim.WStat{}
	for k, v := range w.Stats {
		res.Stats[k] = *v
	}
	for _, p := range w.Players {
		f := 0.0
		if p.HQ >= 0 && w.Structs[p.HQ].Alive {
			f = w.Structs[p.HQ].HP / w.Structs[p.HQ].MaxHP
		}
		res.HQLeft = append(res.HQLeft, f)
	}
}

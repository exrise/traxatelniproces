package sim

import (
	"math"
	"testing"

	"svinovoyna/internal/balance"
)

// stuckReport describes why a turn does not finish.
func (w *World) stuckReport() string {
	s := ""
	if w.ProjectilesBusy() {
		s += " proj"
	}
	for _, u := range w.Units {
		if u.Alive && (!u.Ground || u.Vel.Len() > 4) {
			s += " unit(" + string(rune('0'+u.ID%10)) + ")"
		}
	}
	if !w.StructsSettled() {
		s += " structs"
	}
	return s
}

// TestRandomPlayNeverHangs plays random actions with many units and checks that
// no turn lingers in the settle stage and nobody ends up embedded in terrain.
func TestRandomPlayNeverHangs(t *testing.T) {
	for seed := uint64(1); seed <= 6; seed++ {
		w := newTest(3, seed)
		w.Cfg.RoundsPerBattle = 4
		rng := NewRNG(seed * 77)
		for _, p := range w.Players {
			p.Money = 6000
			mid := float64(p.ZoneX0+p.ZoneX1) / 2
			for i, def := range []string{"assault", "mortar", "rpg", "sniper", "engineer"} {
				if err := w.PlaceUnit(p.ID, def, mid+float64(i-2)*60, 100); err != nil {
					t.Fatalf("place %s: %v", def, err)
				}
			}
			w.SetReady(p.ID, true)
		}
		w.Step()
		maxSettle := 0.0
		for tick := 0; tick < 60*60*8 && w.Phase == PhaseBattle; tick++ {
			if w.Cur >= 0 && w.Stage == StageActive && w.TurnTimer < 40 {
				// act: random walk, jump, or fire
				us := w.UnitsOf(w.Cur)
				if len(us) > 0 {
					u := us[rng.Intn(len(us))]
					w.Apply(Command{Player: w.Cur, Type: CmdSelectUnit, ID: u.ID})
					switch rng.Intn(3) {
					case 0:
						w.Apply(Command{Player: w.Cur, Type: CmdJump})
					default:
						ws := w.Cfg.U(u.Def).Weapons
						wid := ws[rng.Intn(len(ws))]
						w.Apply(Command{Player: w.Cur, Type: CmdFire, Weapon: wid, Angle: rng.Range(-math.Pi, 0), Power: rng.Range(0.3, 1)})
					}
				}
				w.Apply(Command{Player: w.Cur, Type: CmdEndTurn})
			}
			w.Step()
			if w.Stage == StageSettle {
				maxSettle = math.Max(maxSettle, w.SettleT)
				if w.SettleT > 12 {
					t.Fatalf("seed %d: turn stuck in settle for %.1fs:%s", seed, w.SettleT, w.stuckReport())
				}
			}
			w.Events = w.Events[:0]
			for _, u := range w.Units {
				if u.Alive && !w.unitBodyFree(u.Pos.X, u.Pos.Y) {
					t.Fatalf("seed %d: unit %d embedded in terrain at %v", seed, u.ID, u.Pos)
				}
			}
		}
		t.Logf("seed %d: max settle %.1fs, phase=%d battle=%d", seed, maxSettle, w.Phase, w.BattleNo)
	}
}

var _ = balance.Default

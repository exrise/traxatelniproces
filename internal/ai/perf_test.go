package ai

import (
	"testing"
	"time"

	"svinovoyna/internal/balance"
	"svinovoyna/internal/sim"
)

// TestThinkIsFast makes sure a bot decision never freezes the UI thread for long.
func TestThinkIsFast(t *testing.T) {
	cfg := balance.Default()
	styles := []Style{Rocket, Turret, Balanced, Infantry}
	var setups []sim.PlayerSetup
	for i := range styles {
		setups = append(setups, sim.PlayerSetup{Name: styles[i].String(), Color: i, Bot: true})
	}
	w := sim.NewWorld(cfg, 9, setups)
	bots := make([]*Bot, len(styles))
	for i, s := range styles {
		bots[i] = New(i, s, 0.8, 9)
	}
	var worst time.Duration
	var total time.Duration
	calls := 0
	for w.Tick < 60*60*25 && w.Phase != sim.PhaseOver {
		if w.Phase == sim.PhaseBuild {
			for _, b := range bots {
				b.Build(w)
			}
		} else {
			for _, b := range bots {
				t0 := time.Now()
				cmds := b.Think(w)
				d := time.Since(t0)
				if d > 3*time.Millisecond {
					calls++
					total += d
				}
				if d > worst {
					worst = d
				}
				for _, c := range cmds {
					_ = w.Apply(c)
				}
			}
		}
		w.Step()
		w.Events = w.Events[:0]
	}
	t.Logf("worst Think: %v, heavy calls: %d, avg heavy: %v", worst, calls, total/time.Duration(max(1, calls)))
	if worst > 40*time.Millisecond {
		t.Fatalf("bot thinking blocks the frame for %v", worst)
	}
}

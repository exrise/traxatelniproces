package ai

import (
	"testing"

	"svinovoyna/internal/balance"
)

func TestBotMatchFinishes(t *testing.T) {
	cfg := balance.Default()
	for _, n := range []int{2, 3, 4} {
		styles := make([]Style, n)
		for i := range styles {
			styles[i] = Style(i % int(NumStyles))
		}
		r := RunMatch(cfg, uint64(100+n), styles, 0.8, 3600)
		t.Logf("n=%d finished=%v winner=%d builds=%d secs=%.0f dmg=%v", n, r.Finished, r.Winner, r.Builds, r.Seconds, r.DamageDone)
		if r.Builds < 1 {
			t.Fatal("no build phase")
		}
	}
}

func TestTurnModesProgress(t *testing.T) {
	for _, mode := range []balance.TurnMode{balance.TurnClassic, balance.TurnUnitAndStruct, balance.TurnSimultaneous} {
		cfg := balance.Default()
		cfg.TurnMode = mode
		cfg.RoundsPerBattle = 3
		r := RunMatch(cfg, 5, []Style{Balanced, Rocket, Air}, 0.8, 1800)
		t.Logf("%s: finished=%v winner=%d builds=%d secs=%.0f", mode, r.Finished, r.Winner, r.Builds, r.Seconds)
		if r.Builds < 2 {
			t.Fatalf("%s: game never left the first battle", mode)
		}
		shots := 0
		for _, s := range r.Stats {
			shots += s.Shots
		}
		if shots < 5 {
			t.Fatalf("%s: almost nothing was fired (%d shots)", mode, shots)
		}
	}
}

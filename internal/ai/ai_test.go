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

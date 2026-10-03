package sfx

import (
	"math"
	"math/rand"
	"testing"
)

func checkSamples(t *testing.T, name string, s []float64) {
	t.Helper()
	if len(s) == 0 {
		t.Fatalf("%s: empty", name)
	}
	peak, energy := 0.0, 0.0
	for _, v := range s {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("%s: NaN/Inf sample", name)
		}
		peak = math.Max(peak, math.Abs(v))
		energy += v * v
	}
	if peak < 0.05 {
		t.Fatalf("%s: practically silent (peak %.3f)", name, peak)
	}
	if peak > 6 {
		t.Fatalf("%s: wildly too loud (peak %.1f)", name, peak)
	}
	_ = energy
}

func TestSoundsAreSane(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	checkSamples(t, "boom", explosion(r, 0.9, 1, 70))
	checkSamples(t, "pop", pop(r, 0.09, 1800, 1))
	checkSamples(t, "whoosh", whoosh(r, 0.7))
	checkSamples(t, "thump", thump(r, 0.4, 120))
	checkSamples(t, "engine", engine(1.8))
	checkSamples(t, "hammer", hammer())
	checkSamples(t, "squeal", squeal())
	checkSamples(t, "splash", splash(r))
	checkSamples(t, "blip", blip(400, 700, 0.1))
	m := track()
	checkSamples(t, "music", m)
	if secs := float64(len(m)) / rate; secs < 10 || secs > 30 {
		t.Fatalf("music length %.1fs", secs)
	}
}

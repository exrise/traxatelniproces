// Package sfx synthesizes all sound effects procedurally and plays them with
// Ebitengine's audio package. No audio files are needed.
package sfx

import (
	"math"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"bytes"

	"github.com/ebitengine/oto/v3"
)

const rate = 44100

var (
	ctx     *oto.Context
	sounds  = map[string][]byte{}
	mu      sync.Mutex
	active  []*oto.Player
	lastHit = map[string]time.Time{}
	enabled bool
)

// Init prepares the audio context and synthesizes the sounds. It never panics.
func Init() {
	defer func() { _ = recover() }()
	if os.Getenv("SVINO_NOSOUND") != "" {
		return
	}
	var ready chan struct{}
	var err error
	ctx, ready, err = oto.NewContext(&oto.NewContextOptions{SampleRate: rate, ChannelCount: 2, Format: oto.FormatSignedInt16LE})
	if err != nil {
		ctx = nil
		return
	}
	<-ready
	rng := rand.New(rand.NewSource(7))
	sounds["boom"] = pcm(explosion(rng, 0.9, 1.0, 70))
	sounds["boom_big"] = pcm(explosion(rng, 1.7, 1.3, 48))
	sounds["shot_rifle"] = pcm(pop(rng, 0.09, 1800, 1.0))
	sounds["shot_pistol"] = pcm(pop(rng, 0.06, 2600, 0.7))
	sounds["shot_shotgun"] = pcm(pop(rng, 0.16, 900, 1.2))
	sounds["shot_sniper"] = pcm(pop(rng, 0.25, 700, 1.1))
	sounds["shot_cannon"] = pcm(explosion(rng, 0.55, 0.8, 90))
	sounds["shot_whoosh"] = pcm(whoosh(rng, 0.7))
	sounds["shot_mortar"] = pcm(thump(rng, 0.4, 120))
	sounds["plane"] = pcm(engine(1.8))
	sounds["build"] = pcm(hammer())
	sounds["squeal"] = pcm(squeal())
	sounds["splash"] = pcm(splash(rng))
	sounds["jump"] = pcm(blip(420, 700, 0.12))
	enabled = true
	go StartMusic()
}

func pcm(mono []float64) []byte {
	out := make([]byte, 0, len(mono)*4)
	for _, v := range mono {
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		s := int16(v * 30000)
		lo, hi := byte(s), byte(s>>8)
		out = append(out, lo, hi, lo, hi)
	}
	return out
}

func env(i, n int, attack, decayPow float64) float64 {
	t := float64(i) / float64(n)
	a := math.Min(1, t/attack)
	return a * math.Pow(1-t, decayPow)
}

func explosion(r *rand.Rand, dur, gain, thumpHz float64) []float64 {
	n := int(dur * rate)
	out := make([]float64, n)
	lp := 0.0
	for i := range out {
		noise := r.Float64()*2 - 1
		cut := 0.05 + 0.5*math.Pow(1-float64(i)/float64(n), 3)
		lp += (noise - lp) * cut
		t := float64(i) / rate
		th := math.Sin(2*math.Pi*thumpHz*t*(1-0.5*t/dur)) * math.Exp(-t*4)
		out[i] = (lp*1.6*env(i, n, 0.002, 2.2) + th*0.9) * gain
	}
	return out
}

func pop(r *rand.Rand, dur, cutoff, gain float64) []float64 {
	n := int(dur * rate)
	out := make([]float64, n)
	lp := 0.0
	k := math.Min(1, 2*math.Pi*cutoff/rate)
	for i := range out {
		lp += ((r.Float64()*2 - 1) - lp) * k
		out[i] = lp * 1.5 * math.Pow(1-float64(i)/float64(n), 3) * gain
	}
	return out
}

func whoosh(r *rand.Rand, dur float64) []float64 {
	n := int(dur * rate)
	out := make([]float64, n)
	lp := 0.0
	for i := range out {
		t := float64(i) / float64(n)
		k := 0.02 + 0.3*math.Sin(math.Pi*t)
		lp += ((r.Float64()*2 - 1) - lp) * k
		out[i] = lp * 1.4 * math.Sin(math.Pi*t) * 0.9
	}
	return out
}

func thump(r *rand.Rand, dur, hz float64) []float64 {
	n := int(dur * rate)
	out := make([]float64, n)
	for i := range out {
		t := float64(i) / rate
		out[i] = (math.Sin(2*math.Pi*hz*t*math.Exp(-t*3)) + 0.25*(r.Float64()*2-1)*math.Exp(-t*30)) * math.Exp(-t*7)
	}
	return out
}

func engine(dur float64) []float64 {
	n := int(dur * rate)
	out := make([]float64, n)
	for i := range out {
		t := float64(i) / rate
		saw := 2*math.Mod(t*90, 1) - 1
		saw2 := 2*math.Mod(t*135.5, 1) - 1
		out[i] = (saw*0.3 + saw2*0.2) * math.Sin(math.Pi*float64(i)/float64(n)) * 0.7
	}
	return out
}

func hammer() []float64 {
	n := int(0.35 * rate)
	out := make([]float64, n)
	for i := range out {
		t := float64(i) / rate
		hit := func(t0 float64) float64 {
			if t < t0 {
				return 0
			}
			return math.Sin(2*math.Pi*820*(t-t0)) * math.Exp(-(t-t0)*60)
		}
		out[i] = (hit(0) + 0.8*hit(0.14)) * 0.7
	}
	return out
}

func squeal() []float64 {
	n := int(0.55 * rate)
	out := make([]float64, n)
	ph := 0.0
	for i := range out {
		t := float64(i) / float64(n)
		f := 700 + 900*math.Sin(math.Pi*t) + 60*math.Sin(t*60)
		ph += 2 * math.Pi * f / rate
		out[i] = (math.Sin(ph) + 0.3*math.Sin(2*ph)) * math.Sin(math.Pi*t) * 0.6
	}
	return out
}

func splash(r *rand.Rand) []float64 {
	n := int(0.6 * rate)
	out := make([]float64, n)
	lp := 0.0
	for i := range out {
		t := float64(i) / float64(n)
		lp += ((r.Float64()*2 - 1) - lp) * 0.25
		out[i] = lp * math.Sin(math.Pi*math.Min(1, t*3)) * math.Pow(1-t, 1.5) * 0.9
	}
	return out
}

func blip(f0, f1, dur float64) []float64 {
	n := int(dur * rate)
	out := make([]float64, n)
	ph := 0.0
	for i := range out {
		t := float64(i) / float64(n)
		ph += 2 * math.Pi * (f0 + (f1-f0)*t) / rate
		out[i] = math.Sin(ph) * (1 - t) * 0.4
	}
	return out
}

// kindFor maps a weapon id (from "shot_<id>") to a synthesized sound.
func kindFor(name string) string {
	id := strings.TrimPrefix(name, "shot_")
	switch id {
	case "ak", "dshk":
		return "shot_rifle"
	case "makarov":
		return "shot_pistol"
	case "saiga":
		return "shot_shotgun"
	case "svd":
		return "shot_sniper"
	case "rpg", "kornet", "grad", "fpv", "iskander", "oreshnik", "geran":
		return "shot_whoosh"
	case "mortar", "grenade":
		return "shot_mortar"
	case "d30":
		return "shot_cannon"
	case "fab", "kab":
		return "plane"
	}
	return ""
}

// Play plays a named sound at the given volume (0..1). Safe to call from any goroutine.
func Play(name string, vol float64) {
	if !enabled || vol <= 0 {
		return
	}
	defer func() { _ = recover() }()
	if strings.HasPrefix(name, "shot_") {
		name = kindFor(name)
		if name == "" {
			return
		}
	}
	data, ok := sounds[name]
	if !ok {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	if t, ok := lastHit[name]; ok && now.Sub(t) < 35*time.Millisecond {
		return
	}
	lastHit[name] = now
	live := active[:0]
	for _, p := range active {
		if p.IsPlaying() {
			live = append(live, p)
		}
	}
	active = live
	if len(active) > 28 {
		return
	}
	p := ctx.NewPlayer(bytes.NewReader(data))
	p.SetVolume(math.Min(1, vol))
	p.Play()
	active = append(active, p)
}

package sfx

import (
	"bytes"
	"io"
	"math"
	"math/rand"
	"sync"
)

// loopReader endlessly repeats a PCM buffer.
type loopReader struct {
	data []byte
	pos  int
}

func (l *loopReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		c := copy(p[n:], l.data[l.pos:])
		n += c
		l.pos = (l.pos + c) % len(l.data)
	}
	return n, nil
}

var (
	musicMu     sync.Mutex
	musicPlayer interface {
		SetVolume(float64)
		Play()
		Pause()
		IsPlaying() bool
	}
	musicVol = 0.22
	musicOn  = true
)

// StartMusic begins the looping background track (no-op without audio).
func StartMusic() {
	if !enabled {
		return
	}
	defer func() { _ = recover() }()
	musicMu.Lock()
	defer musicMu.Unlock()
	if musicPlayer != nil {
		return
	}
	data := pcm(track())
	p := ctx.NewPlayer(io.Reader(&loopReader{data: data}))
	p.SetVolume(musicVol)
	if musicOn {
		p.Play()
	}
	musicPlayer = p
	_ = bytes.MinRead
}

// SetMusic turns the background track on or off.
func SetMusic(on bool) {
	musicMu.Lock()
	defer musicMu.Unlock()
	musicOn = on
	if musicPlayer == nil {
		return
	}
	if on {
		musicPlayer.Play()
	} else {
		musicPlayer.Pause()
	}
}

// MusicOn reports the toggle state.
func MusicOn() bool {
	musicMu.Lock()
	defer musicMu.Unlock()
	return musicOn
}

// SetMusicVolume sets the music loudness (0..1).
func SetMusicVolume(v float64) {
	musicMu.Lock()
	defer musicMu.Unlock()
	musicVol = v
	if musicPlayer != nil {
		musicPlayer.SetVolume(v)
	}
}

func midi(n float64) float64 { return 440 * math.Pow(2, (n-69)/12) }

// track synthesizes an 8-bar marching chiptune loop.
func track() []float64 {
	const bpm = 128.0
	beat := 60.0 / bpm
	step := beat / 2 // eighth note
	roots := []float64{45, 45, 41, 41, 48, 48, 43, 43}
	bars := len(roots)
	total := int(float64(bars*8) * step * rate)
	out := make([]float64, total)
	r := rand.New(rand.NewSource(3))
	add := func(at float64, dur float64, f func(t float64) float64, gain float64) {
		i0 := int(at * rate)
		n := int(dur * rate)
		for i := 0; i < n && i0+i < total; i++ {
			t := float64(i) / rate
			env := math.Min(1, t/0.004) * math.Exp(-t*(3.5/dur))
			out[i0+i] += f(t) * env * gain
		}
	}
	square := func(freq, duty float64) func(float64) float64 {
		return func(t float64) float64 {
			if math.Mod(t*freq, 1) < duty {
				return 1
			}
			return -1
		}
	}
	tri := func(freq float64) func(float64) float64 {
		return func(t float64) float64 { return 2*math.Abs(2*math.Mod(t*freq, 1)-1) - 1 }
	}
	chord := []float64{0, 3, 7, 12, 15, 12, 7, 3}
	for bar := 0; bar < bars; bar++ {
		root := roots[bar]
		for s := 0; s < 8; s++ {
			at := (float64(bar*8) + float64(s)) * step
			// bass: root on beats, octave up on off-beats
			n := root
			if s%2 == 1 {
				n += 12
			}
			add(at, step*0.95, tri(midi(n)), 0.34)
			// lead arpeggio in sixteenths
			for h := 0; h < 2; h++ {
				note := root + 24 + chord[(s*2+h+bar)%len(chord)]
				add(at+float64(h)*step/2, step/2*0.9, square(midi(note), 0.25), 0.07)
			}
			// drums
			if s%4 == 0 {
				add(at, 0.18, func(t float64) float64 { return math.Sin(2 * math.Pi * (110 * math.Exp(-t*18)) * t) }, 0.55)
			}
			if s%4 == 2 {
				add(at, 0.12, func(t float64) float64 { return r.Float64()*2 - 1 }, 0.16)
			}
			add(at, 0.03, func(t float64) float64 { return r.Float64()*2 - 1 }, 0.035)
		}
		// melody every two bars: a simple march theme on top
		if bar%2 == 0 {
			th := []float64{0, 0, 7, 7, 8, 7, 5, 3}
			for k, d := range th {
				add((float64(bar*8)+float64(k))*step, step*0.8, square(midi(root+36+d), 0.5), 0.05)
			}
		}
	}
	// normalise
	peak := 0.0
	for _, v := range out {
		peak = math.Max(peak, math.Abs(v))
	}
	for i := range out {
		out[i] = out[i] / peak * 0.8
	}
	return out
}

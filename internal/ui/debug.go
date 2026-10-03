package ui

import (
	"image/png"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// debugShot saves a screenshot when SVINO_DEBUG is set and the request file exists.
// Used for automated visual checks under Xvfb; harmless otherwise.
func (a *App) debugShot(screen *ebiten.Image) {
	if os.Getenv("SVINO_DEBUG") == "" {
		return
	}
	b, err := os.ReadFile("/tmp/svino_shot_request")
	if err != nil {
		return
	}
	_ = os.Remove("/tmp/svino_shot_request")
	path := strings.TrimSpace(string(b))
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	img := screen.SubImage(screen.Bounds())
	_ = w
	_ = h
	_ = png.Encode(f, img)
}

// debugStart auto-starts a bots-only match when SVINO_DEBUG=bots:N.
func debugStart(a *App) Scene {
	v := os.Getenv("SVINO_DEBUG")
	if !strings.HasPrefix(v, "bots:") && !strings.HasPrefix(v, "human:") {
		return nil
	}
	n := int(v[len(v)-1] - '0')
	if n < 2 || n > 4 {
		n = 2
	}
	slots := make([]Slot, n)
	for i := range slots {
		slots[i] = Slot{Name: []string{"Хрюша", "Борька", "Кабан", "Пятачок"}[i], Human: strings.HasPrefix(v, "human:") && i == 0, Skill: 0.8, Color: i, Team: i}
	}
	cfg := a.Presets[0].Clone()
	return NewMatch(a, NewLocalSession(cfg, 7, slots), nil)
}

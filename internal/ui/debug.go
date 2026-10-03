package ui

import (
	"image/png"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"svinovoyna/internal/balance"
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
	f, err := os.Create(strings.TrimSpace(string(b)))
	if err != nil {
		return
	}
	defer f.Close()
	_ = png.Encode(f, screen.SubImage(screen.Bounds()))
}

// debugStart auto-starts a match when SVINO_DEBUG=<bots|human|hot>:<players>[:<classic|two|sim>].
func debugStart(a *App) Scene {
	parts := strings.Split(os.Getenv("SVINO_DEBUG"), ":")
	if len(parts) < 2 || (parts[0] != "bots" && parts[0] != "human" && parts[0] != "hot") {
		return nil
	}
	n := int(parts[1][0] - '0')
	if n < 2 || n > 4 {
		n = 2
	}
	slots := make([]Slot, n)
	for i := range slots {
		human := parts[0] == "hot" || (parts[0] == "human" && i == 0)
		slots[i] = Slot{Name: []string{"Хрюша", "Борька", "Кабан", "Пятачок"}[i], Human: human, Skill: 0.8, Color: i, Team: i}
	}
	cfg := a.Presets[0].Clone()
	if len(parts) > 2 {
		switch parts[2] {
		case "two":
			cfg.TurnMode = balance.TurnUnitAndStruct
		case "sim":
			cfg.TurnMode = balance.TurnSimultaneous
		}
	}
	return NewMatch(a, NewLocalSession(cfg, 7, slots), nil)
}

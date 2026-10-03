package ui

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"svinovoyna/internal/balance"
)

// rulesPanel is the right-hand side of the lobby screens: presets, turn mode and
// the main numeric rules. It edits a working copy of a config.
type rulesPanel struct {
	cfg     *balance.Config
	preset  int
	seed    uint64
	presets []*Button
	modes   [3]*Button
	steps   []*stepper
	teams   *Button
	ff      *Button
	editBtn *Button
	seedBtn *Button
	changed bool
}

func newRulesPanel(a *App) *rulesPanel {
	r := &rulesPanel{seed: uint64(rand.Int63()) % 100000}
	r.cfg = a.Presets[0].Clone()
	for i, p := range a.Presets {
		b := NewButton(690+(i%3)*180, 150+(i/3)*50, 172, 42, p.Name)
		b.Size = 17
		r.presets = append(r.presets, b)
	}
	for i := range r.modes {
		r.modes[i] = NewButton(690+i*180, 290, 172, 56, balance.TurnMode(i).String())
		r.modes[i].Size = 16
	}
	x := 690
	r.steps = []*stepper{
		newStepper("Раундов в бою", x, 372, func() string { return fmt.Sprint(r.cfg.RoundsPerBattle) }, func() { r.cfg.RoundsPerBattle = max(1, r.cfg.RoundsPerBattle-1) }, func() { r.cfg.RoundsPerBattle = min(30, r.cfg.RoundsPerBattle+1) }),
		newStepper("Время хода, с", x, 414, func() string { return fmt.Sprint(int(r.cfg.TurnTime)) }, func() { r.cfg.TurnTime = max(10, r.cfg.TurnTime-5) }, func() { r.cfg.TurnTime = min(180, r.cfg.TurnTime+5) }),
		newStepper("Стартовые деньги", x, 456, func() string { return fmt.Sprint(r.cfg.StartMoney) }, func() { r.cfg.StartMoney = max(300, r.cfg.StartMoney-100) }, func() { r.cfg.StartMoney += 100 }),
		newStepper("Время стройки, с", x, 498, func() string { return fmt.Sprint(int(r.cfg.BuildTimeFirst)) }, func() { r.cfg.BuildTimeFirst = max(20, r.cfg.BuildTimeFirst-15) }, func() { r.cfg.BuildTimeFirst += 15 }),
	}
	r.teams = NewButton(690, 548, 262, 42, "Команды 2×2")
	r.teams.Size = 18
	r.ff = NewButton(968, 548, 262, 42, "Огонь по своим")
	r.ff.Size = 18
	r.editBtn = NewButton(690, 598, 262, 46, "Тонкая настройка…")
	r.seedBtn = NewButton(968, 598, 262, 46, "")
	r.seedBtn.Size = 18
	return r
}

// Update processes input; back is the scene to return to from the balance editor.
func (r *rulesPanel) Update(a *App, back Scene, players int) {
	for i, b := range r.presets {
		if b.Update(a) {
			r.preset = i
			r.cfg = a.Presets[i].Clone()
			r.changed = true
		}
		b.On = i == r.preset
	}
	for i, b := range r.modes {
		if b.Update(a) {
			r.cfg.TurnMode = balance.TurnMode(i)
			r.changed = true
		}
		b.On = int(r.cfg.TurnMode) == i
	}
	for _, s := range r.steps {
		s.Update(a)
	}
	if r.teams.Update(a) {
		r.cfg.TeamsEnabled = !r.cfg.TeamsEnabled
	}
	r.teams.On = r.cfg.TeamsEnabled
	r.teams.Enabled = players == 4
	if players != 4 {
		r.cfg.TeamsEnabled = false
	}
	if r.ff.Update(a) {
		r.cfg.FriendlyFire = !r.cfg.FriendlyFire
	}
	r.ff.On = r.cfg.FriendlyFire
	if r.editBtn.Update(a) {
		a.SetScene(NewBalanceEditor(a, r.cfg, back, func(c *balance.Config) { r.cfg = c; r.changed = true }))
	}
	r.seedBtn.Label = fmt.Sprintf("Карта №%d (сменить)", r.seed)
	if r.seedBtn.Update(a) {
		r.seed = uint64(rand.Int63()) % 100000
	}
}

func (r *rulesPanel) Draw(a *App, dst *ebiten.Image) {
	a.TextB(dst, "Баланс и правила", 690, 108, 28, colText)
	for _, b := range r.presets {
		b.Draw(a, dst)
	}
	for _, b := range r.modes {
		b.Draw(a, dst)
	}
	for _, s := range r.steps {
		s.Draw(a, dst)
	}
	r.teams.Draw(a, dst)
	r.ff.Draw(a, dst)
	r.editBtn.Draw(a, dst)
	r.seedBtn.Draw(a, dst)
}

// readOnlySummary draws a compact description of a config (for clients).
func drawCfgSummary(a *App, dst *ebiten.Image, cfg *balance.Config, x, y float64) {
	a.TextB(dst, "Правила хоста", x, y, 28, colText)
	lines := []string{
		"Пресет: " + cfg.Name,
		"Режим хода: " + cfg.TurnMode.String(),
		fmt.Sprintf("Раундов в бою: %d,  время хода: %.0f с", cfg.RoundsPerBattle, cfg.TurnTime),
		fmt.Sprintf("Стартовые деньги: $%d,  стройка: %.0f с", cfg.StartMoney, cfg.BuildTimeFirst),
		fmt.Sprintf("Огонь по своим: %v,  команды: %v", yesNo(cfg.FriendlyFire), yesNo(cfg.TeamsEnabled)),
	}
	for i, l := range lines {
		a.Text(dst, l, x, y+48+float64(i)*34, 22, color.RGBA{225, 232, 245, 255})
	}
}

func yesNo(b bool) string {
	if b {
		return "да"
	}
	return "нет"
}

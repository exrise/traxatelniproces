package ui

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"svinovoyna/internal/gfx"
)

var botLevels = []struct {
	name  string
	skill float64
}{{"Бот: лёгкий", 0.45}, {"Бот: средний", 0.72}, {"Бот: сильный", 0.92}}

// stepper is a labelled +/- control.
type stepper struct {
	label    string
	minus    *Button
	plus     *Button
	get      func() string
	dec, inc func()
}

func newStepper(label string, x, y int, get func() string, dec, inc func()) *stepper {
	return &stepper{label: label, minus: NewButton(x+300, y, 36, 34, "-"), plus: NewButton(x+420, y, 36, 34, "+"), get: get, dec: dec, inc: inc}
}

func (s *stepper) Update(a *App) {
	if s.minus.Update(a) {
		s.dec()
	}
	if s.plus.Update(a) {
		s.inc()
	}
}

func (s *stepper) Draw(a *App, dst *ebiten.Image) {
	a.Text(dst, s.label, float64(s.minus.R.Min.X)-300, float64(s.minus.R.Min.Y)+4, 21, colText)
	a.TextCenter(dst, s.get(), float64(s.minus.R.Max.X+s.plus.R.Min.X)/2, float64(s.minus.R.Min.Y)+4, 21, colGold, true)
	s.minus.Draw(a, dst)
	s.plus.Draw(a, dst)
}

// Lobby configures a local game.
type Lobby struct {
	n        int
	slots    [4]Slot
	names    [4]*TextInput
	typeBtn  [4]*Button
	colorBtn [4]*Button
	teamBtn  [4]*Button
	plusN    *Button
	minusN   *Button
	rules    *rulesPanel
	startBtn *Button
	backBtn  *Button
}

// NewLobby builds the local game setup screen.
func NewLobby(a *App) *Lobby {
	l := &Lobby{n: 2, rules: newRulesPanel(a)}
	for i := 0; i < 4; i++ {
		l.slots[i] = Slot{Name: fmt.Sprintf("Игрок %d", i+1), Human: i == 0, Skill: botLevels[1].skill, Color: i, Team: i % 2}
		l.names[i] = &TextInput{R: rectXYWH(110, 150+i*84, 220, 40), Text: l.slots[i].Name, Max: 14}
		l.typeBtn[i] = NewButton(340, 150+i*84, 190, 40, "")
		l.typeBtn[i].Size = 18
		l.colorBtn[i] = NewButton(60, 150+i*84, 40, 40, "")
		l.teamBtn[i] = NewButton(540, 150+i*84, 70, 40, "")
		l.teamBtn[i].Size = 18
	}
	l.names[0].Text = a.Set.Name
	l.slots[0].Name = a.Set.Name
	l.minusN = NewButton(60, 490, 40, 40, "-")
	l.plusN = NewButton(250, 490, 40, 40, "+")
	l.startBtn = NewButton(ScreenW-340, ScreenH-62, 300, 52, "В БОЙ!")
	l.startBtn.Col = color.RGBA{40, 130, 60, 255}
	l.startBtn.Size = 26
	l.backBtn = NewButton(40, ScreenH-62, 200, 52, "Назад")
	return l
}

func (l *Lobby) Update(a *App) error {
	for i := 0; i < l.n; i++ {
		l.names[i].Update(a)
		if l.slots[i].Human {
			l.slots[i].Name = l.names[i].Text
		}
		if l.typeBtn[i].Update(a) {
			s := &l.slots[i]
			switch {
			case s.Human:
				s.Human, s.Skill = false, botLevels[0].skill
			case s.Skill < 0.5:
				s.Skill = botLevels[1].skill
			case s.Skill < 0.8:
				s.Skill = botLevels[2].skill
			default:
				s.Human = true
			}
			if !s.Human {
				l.names[i].Text = fmt.Sprintf("Бот %d", i+1)
			} else {
				l.names[i].Text = fmt.Sprintf("Игрок %d", i+1)
			}
			s.Name = l.names[i].Text
		}
		if l.colorBtn[i].Update(a) {
			l.slots[i].Color = (l.slots[i].Color + 1) % 4
		}
		if l.rules.cfg.TeamsEnabled && l.teamBtn[i].Update(a) {
			l.slots[i].Team = 1 - l.slots[i].Team
		}
	}
	if l.minusN.Update(a) && l.n > 2 {
		l.n--
	}
	if l.plusN.Update(a) && l.n < 4 {
		l.n++
	}
	l.rules.Update(a, l, l.n)
	if l.backBtn.Update(a) || a.EscapeKey {
		a.SetScene(NewMenu(a))
	}
	humans := 0
	for i := 0; i < l.n; i++ {
		if l.slots[i].Human {
			humans++
		}
	}
	l.startBtn.Enabled = humans >= 1
	if l.startBtn.Update(a) {
		a.Set.Name = l.slots[0].Name
		a.SaveSettings()
		slots := make([]Slot, l.n)
		copy(slots, l.slots[:l.n])
		for i := range slots {
			if !l.rules.cfg.TeamsEnabled {
				slots[i].Team = i
			}
		}
		l.rules.cfg.Index()
		a.SetScene(NewMatch(a, NewLocalSession(l.rules.cfg.Clone(), l.rules.seed, slots), nil))
	}
	return nil
}

func (l *Lobby) Draw(a *App, dst *ebiten.Image) {
	dst.Fill(colorBG)
	a.TextB(dst, "Настройка игры", 40, 24, 44, colGold)
	a.TextB(dst, "Игроки", 60, 108, 28, colText)
	for i := 0; i < l.n; i++ {
		y := float64(144 + i*84)
		a.Rect(dst, 50, y-6, 570, 56, color.RGBA{32, 44, 72, 255})
		tc := gfx.TeamColors[l.slots[i].Color%4]
		l.colorBtn[i].Col = tc
		l.colorBtn[i].Draw(a, dst)
		l.names[i].Draw(a, dst)
		if l.slots[i].Human {
			l.typeBtn[i].Label = "Человек"
		} else {
			for _, bl := range botLevels {
				if l.slots[i].Skill == bl.skill {
					l.typeBtn[i].Label = bl.name
				}
			}
		}
		l.typeBtn[i].Draw(a, dst)
		if l.rules.cfg.TeamsEnabled {
			l.teamBtn[i].Label = fmt.Sprintf("Ком. %c", 'A'+l.slots[i].Team)
			l.teamBtn[i].Draw(a, dst)
		}
	}
	a.Text(dst, fmt.Sprintf("Игроков: %d", l.n), 112, 497, 24, colText)
	l.minusN.Draw(a, dst)
	l.plusN.Draw(a, dst)
	a.Text(dst, "Нажми на цветной квадрат — сменить цвет.", 60, 550, 17, colDim)
	a.Text(dst, "Несколько людей играют по очереди за одним компьютером (hotseat).", 60, 574, 17, colDim)
	l.rules.Draw(a, dst)
	l.startBtn.Draw(a, dst)
	l.backBtn.Draw(a, dst)
	if !l.startBtn.Enabled {
		a.Text(dst, "Нужен хотя бы один человек", ScreenW-330, ScreenH-90, 18, colBad)
	}
}

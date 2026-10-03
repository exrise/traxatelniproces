package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"svinovoyna/internal/gfx"
	"svinovoyna/internal/sim"
)

type summaryRow struct {
	name                 string
	color                int
	earned, money, units int
	hq                   float64
	dead                 bool
}

// Match is the in-game scene: it switches between building, battle and game over.
type Match struct {
	a      *App
	sess   Session
	w      *sim.World
	view   *View
	build  *BuildUI
	battle *BattleUI
	onExit func()

	lastPhase   sim.Phase
	lastBuildNo int
	lastBattle  int
	lastCur     int

	banner  string
	bannerT float64

	summary  []summaryRow
	summaryT float64

	gateSeat int // hotseat: seat that has confirmed the hand-over
	gate     bool

	paused bool
	pauseB []*Button
	fast   bool
	help   bool
	toast  string
	toastT float64
	over   *Button
}

// NewMatch starts a match scene around a session.
func NewMatch(a *App, sess Session, onExit func()) *Match {
	w := sess.World()
	m := &Match{a: a, sess: sess, w: w, onExit: onExit, lastPhase: -1, gateSeat: -1, lastCur: -2}
	m.view = NewView(a, w)
	m.build = NewBuildUI(m)
	m.battle = NewBattleUI(m)
	m.pauseB = []*Button{NewButton(ScreenW/2-150, 270, 300, 54, "Продолжить"), NewButton(ScreenW/2-150, 340, 300, 54, "В главное меню")}
	m.over = NewButton(ScreenW/2-150, 470, 300, 56, "В главное меню")
	return m
}

// Me returns the local player the HUD should be shown for.
func (m *Match) Me() int {
	w := m.w
	switch w.Phase {
	case sim.PhaseBuild:
		if s := m.sess.BuildSeat(); s >= 0 {
			return s
		}
	case sim.PhaseBattle:
		if w.Cur >= 0 && m.sess.Local(w.Cur) {
			return w.Cur
		}
	}
	for _, p := range w.Players {
		if m.sess.Local(p.ID) && !p.Elim {
			return p.ID
		}
	}
	for _, p := range w.Players {
		if m.sess.Local(p.ID) {
			return p.ID
		}
	}
	return 0
}

func (m *Match) exit() {
	m.sess.Close()
	if m.onExit != nil {
		m.onExit()
	} else {
		m.a.SetScene(NewMenu(m.a))
	}
}

func (m *Match) say(s string) { m.banner, m.bannerT = s, 3.2 }

func (m *Match) Update(a *App) error {
	if m.paused {
		if m.pauseB[0].Update(a) || a.EscapeKey {
			m.paused = false
		}
		if m.pauseB[1].Update(a) {
			m.exit()
		}
		if _, ok := m.sess.(*LocalSession); !ok {
			m.paused = false
		}
		return nil
	}
	steps := 1
	if m.fast {
		steps = 3
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		m.help = !m.help
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) && ebiten.IsKeyPressed(ebiten.KeyShift) {
		m.fast = !m.fast
	}
	for i := 0; i < steps; i++ {
		m.sess.Update()
		m.view.Tick()
		m.detectChanges()
	}
	if m.bannerT > 0 {
		m.bannerT -= sim.Dt * float64(steps)
	}
	if m.summaryT > 0 {
		m.summaryT -= sim.Dt
		if a.Click && a.MY < 420 {
			m.summaryT = 0
		}
	}
	switch m.w.Phase {
	case sim.PhaseBuild:
		m.view.Only = -2
		if !m.gate {
			m.view.Only = m.Me()
			m.build.Update(a)
		} else {
			m.updateGate(a)
		}
	case sim.PhaseBattle:
		m.view.Only = -1
		m.battle.Update(a)
	case sim.PhaseOver:
		m.view.Only = -1
		if m.over.Update(a) {
			m.exit()
		}
		m.battle.cameraIdle(a)
	}
	if e := m.sess.PopError(); e != "" {
		m.toast, m.toastT = e, 2.2
	}
	if m.toastT > 0 {
		m.toastT -= sim.Dt
	}
	if a.EscapeKey && m.w.Phase != sim.PhaseOver {
		if _, ok := m.sess.(*LocalSession); ok {
			m.paused = true
		}
	}
	return nil
}

func (m *Match) detectChanges() {
	w := m.w
	if w.Phase != m.lastPhase || (w.Phase == sim.PhaseBuild && w.BuildNo != m.lastBuildNo) {
		prev := m.lastPhase
		m.lastPhase = w.Phase
		switch w.Phase {
		case sim.PhaseBuild:
			m.lastBuildNo = w.BuildNo
			m.say(fmt.Sprintf("СТРОЙКА %d", w.BuildNo))
			m.build.Enter()
			m.gateSeat = -1
			m.gate = false
			if prev == sim.PhaseBattle {
				m.makeSummary()
			}
		case sim.PhaseBattle:
			m.say("БОЙ!")
			m.battle.Enter()
		case sim.PhaseOver:
			m.say("")
		}
	}
	if w.Phase == sim.PhaseBuild {
		seat := m.sess.BuildSeat()
		if seat >= 0 && seat != m.gateSeat {
			m.gateSeat = seat
			if m.hotseatMany() {
				m.gate = true
			}
			m.build.CenterOn(seat)
		}
	}
}

func (m *Match) hotseatMany() bool {
	n := 0
	for _, p := range m.w.Players {
		if m.sess.Local(p.ID) {
			n++
		}
	}
	return n > 1
}

func (m *Match) updateGate(a *App) {
	if a.Click || a.EnterKey {
		m.gate = false
	}
}

func (m *Match) makeSummary() {
	m.summary = nil
	for _, p := range m.w.Players {
		hq := 0.0
		if p.HQ >= 0 {
			s := m.w.Structs[p.HQ]
			if s.Alive {
				hq = s.HP / s.MaxHP
			}
		}
		m.summary = append(m.summary, summaryRow{p.Name, p.Color, p.LastEarned, p.Money, len(m.w.UnitsOf(p.ID)), hq, p.Elim})
	}
	m.summaryT = 12
}

func (m *Match) Draw(a *App, dst *ebiten.Image) {
	m.view.Draw(dst)
	switch m.w.Phase {
	case sim.PhaseBuild:
		if m.gate {
			m.drawGate(a, dst)
		} else {
			m.build.Draw(a, dst)
		}
	case sim.PhaseBattle:
		m.battle.Draw(a, dst)
	case sim.PhaseOver:
		m.battle.DrawHUDMinimal(a, dst)
		m.drawOver(a, dst)
	}
	if m.summaryT > 0 && m.w.Phase == sim.PhaseBuild && !m.gate {
		m.drawSummary(a, dst)
	}
	if m.bannerT > 0 && m.banner != "" {
		al := math.Min(1, m.bannerT)
		c := color.RGBA{255, 230, 130, uint8(255 * al)}
		a.TextCenter(dst, m.banner, ScreenW/2, 150, 84, c, true)
	}
	if m.paused {
		a.Rect(dst, 0, 0, ScreenW, ScreenH, color.RGBA{0, 0, 0, 160})
		a.TextCenter(dst, "Пауза", ScreenW/2, 190, 48, colText, true)
		m.pauseB[0].Draw(a, dst)
		m.pauseB[1].Draw(a, dst)
	}
	if m.toastT > 0 && m.toast != "" {
		a.TextCenter(dst, m.toast, ScreenW/2, ScreenH-215, 24, colBad, true)
	}
	if m.help {
		m.drawHelp(a, dst)
	}
	if m.fast {
		a.Text(dst, ">> ускорение (Shift+F)", ScreenW-250, ScreenH-24, 16, colGold)
	}
}

func (m *Match) drawGate(a *App, dst *ebiten.Image) {
	a.Rect(dst, 0, 0, ScreenW, ScreenH, color.RGBA{18, 24, 38, 255})
	p := m.w.Players[m.gateSeat]
	a.TextCenter(dst, "Передай управление", ScreenW/2, 220, 40, colText, true)
	a.TextCenter(dst, p.Name, ScreenW/2, 280, 64, gfx.TeamColors[p.Color%4], true)
	a.TextCenter(dst, "Остальные — отвернитесь! Щёлкни, когда готов строить.", ScreenW/2, 380, 24, colDim, false)
}

func (m *Match) drawSummary(a *App, dst *ebiten.Image) {
	x, y, w := float64(ScreenW/2-330), 70.0, 660.0
	h := 70.0 + float64(len(m.summary))*40
	a.Panel(dst, rectXYWH(int(x), int(y), int(w), int(h)))
	a.TextCenter(dst, "Итоги боя", x+w/2, y+8, 30, colGold, true)
	for i, r := range m.summary {
		yy := y + 54 + float64(i)*40
		a.Circle(dst, x+30, yy+14, 9, gfx.TeamColors[r.color%4])
		col := color.Color(colText)
		if r.dead {
			col = colDim
		}
		a.Text(dst, r.name, x+50, yy+2, 21, col)
		a.Text(dst, fmt.Sprintf("заработал $%d", r.earned), x+210, yy+2, 20, colGood)
		a.Text(dst, fmt.Sprintf("юнитов: %d", r.units), x+400, yy+2, 20, colText)
		if r.dead {
			a.Text(dst, "выбыл", x+540, yy+2, 20, colBad)
		} else {
			a.Rect(dst, x+520, yy+8, 100, 12, color.RGBA{50, 20, 20, 255})
			a.Rect(dst, x+520, yy+8, 100*r.hq, 12, color.RGBA{110, 220, 110, 255})
		}
	}
}

func (m *Match) drawOver(a *App, dst *ebiten.Image) {
	a.Rect(dst, 0, 0, ScreenW, ScreenH, color.RGBA{0, 0, 0, 150})
	a.TextCenter(dst, "ПОБЕДА!", ScreenW/2, 150, 90, colGold, true)
	name := m.w.WinnerName
	col := color.Color(colText)
	if m.w.Winner >= 0 {
		for _, p := range m.w.Players {
			if p.Team == m.w.Winner {
				col = gfx.TeamColors[p.Color%4]
			}
		}
	}
	a.TextCenter(dst, name, ScreenW/2, 270, 56, col, true)
	a.TextCenter(dst, fmt.Sprintf("Сыграно боёв: %d  •  время: %d мин", m.w.BattleNo, int(m.w.Time/60)), ScreenW/2, 370, 24, colDim, false)
	m.over.Draw(a, dst)
}

func (m *Match) drawHelp(a *App, dst *ebiten.Image) {
	a.Rect(dst, 0, 0, ScreenW, ScreenH, color.RGBA{0, 0, 0, 190})
	x, y := 160.0, 70.0
	a.TextB(dst, "Справка (F1 — закрыть)", x, y, 38, colGold)
	lines := []string{
		"ЦЕЛЬ: разрушить штаб врага. Потерял штаб — вылетел.",
		"СТРОЙКА: покупай юнитов, орудия, ПВО, защиту. ЛКМ — поставить, ПКМ — продать, R — починить.",
		"Постройки без опоры падают. Крыша над штабом ловит бомбы и снаряды сверху.",
		"БОЙ: A/D — идти, W — прыжок, мышь — прицел, ЛКМ или Пробел — огонь (держи — сила выстрела).",
		"1…9 — выбор оружия, Tab — следующий юнит, Q — следующее орудие, E — закончить ход.",
		"Искандер, Орешник, авиаудары, «Герань»: кликни по карте (цель) и нажми «Огонь».",
		"FPV-дрон летит за курсором, клик — подрыв. ПВО стреляет по самолётам, дронам и ракетам само.",
		"ДЕНЬГИ: за урон и убийства, с нефтевышек и ферм, с точек захвата (флажки на карте).",
		"Урон по лидеру (★) оплачивается лучше. Проигравшим возвращается часть потерь.",
		"Камера: ПКМ + движение, колесо — масштаб, клик по миникарте. T — траектория, Esc — пауза.",
	}
	for i, l := range lines {
		a.Text(dst, l, x, y+70+float64(i)*40, 22, colText)
	}
}

package ui

import (
	"bytes"
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"svinovoyna/internal/balance"
	"svinovoyna/internal/gfx"
	"svinovoyna/internal/netplay"
)

// NetHost is the lobby of a hosted network game.
type NetHost struct {
	a        *App
	host     *netplay.Host
	err      string
	name     *TextInput
	typeBtn  [4]*Button
	colorBtn [4]*Button
	teamBtn  [4]*Button
	plusN    *Button
	minusN   *Button
	rules    *rulesPanel
	startBtn *Button
	backBtn  *Button
	ips      []string
	port     string
	lastSig  string
	frame    int
	msg      string
}

// NewNetHost opens a listening socket and shows the host lobby.
func NewNetHost(a *App) Scene {
	h := &NetHost{a: a, rules: newRulesPanel(a), port: a.Set.Port}
	if h.port == "" {
		h.port = netplay.DefaultPort
	}
	lob := netplay.LobbyState{Slots: []netplay.LobbySlot{
		{Kind: netplay.SlotHost, Name: a.Set.Name, Color: 0, Team: 0},
		{Kind: netplay.SlotOpen, Color: 1, Team: 1, Skill: botLevels[1].skill},
	}}
	host, err := netplay.Listen(h.port, lob)
	if err != nil {
		return &errScene{msg: "Не удалось открыть порт " + h.port + ": " + err.Error() + "\nВозможно, он занят другой копией игры."}
	}
	h.host = host
	h.ips = netplay.LocalIPs()
	h.name = &TextInput{R: rectXYWH(110, 150, 220, 40), Text: a.Set.Name, Max: 14}
	for i := 0; i < 4; i++ {
		h.typeBtn[i] = NewButton(340, 150+i*84, 190, 40, "")
		h.typeBtn[i].Size = 18
		h.colorBtn[i] = NewButton(60, 150+i*84, 40, 40, "")
		h.teamBtn[i] = NewButton(540, 150+i*84, 70, 40, "")
		h.teamBtn[i].Size = 18
	}
	h.minusN = NewButton(60, 490, 40, 40, "-")
	h.plusN = NewButton(250, 490, 40, 40, "+")
	h.startBtn = NewButton(ScreenW-340, ScreenH-62, 300, 52, "СТАРТ!")
	h.startBtn.Col = color.RGBA{40, 130, 60, 255}
	h.startBtn.Size = 26
	h.backBtn = NewButton(40, ScreenH-62, 200, 52, "Отмена")
	h.sync(true)
	return h
}

// sync pushes the rules into the lobby when they changed.
func (h *NetHost) sync(force bool) {
	cfg := h.rules.cfg
	cfg.Index()
	js := cfg.Marshal()
	sig := fmt.Sprintf("%s|%d|%d", js, h.rules.seed, h.host.Lobby().Slots[0].Color)
	if !force && sig == h.lastSig {
		return
	}
	h.lastSig = sig
	h.host.Edit(func(l *netplay.LobbyState) {
		l.CfgJSON = js
		l.Seed = h.rules.seed
		l.Preset = cfg.Name
	})
}

func (h *NetHost) Update(a *App) error {
	h.frame++
	lob := h.host.Lobby()
	n := len(lob.Slots)
	h.name.Update(a)
	if h.name.Text != lob.Slots[0].Name && h.name.Text != "" {
		nm := h.name.Text
		h.host.Edit(func(l *netplay.LobbyState) { l.Slots[0].Name = nm })
		a.Set.Name = nm
	}
	for i := 0; i < n; i++ {
		i := i
		s := lob.Slots[i]
		if h.colorBtn[i].Update(a) {
			h.host.Edit(func(l *netplay.LobbyState) { l.Slots[i].Color = (l.Slots[i].Color + 1) % 4 })
		}
		if h.rules.cfg.TeamsEnabled && h.teamBtn[i].Update(a) {
			h.host.Edit(func(l *netplay.LobbyState) { l.Slots[i].Team = 1 - l.Slots[i].Team })
		}
		if i == 0 {
			h.typeBtn[0].Enabled = false
			continue
		}
		if h.typeBtn[i].Update(a) {
			switch s.Kind {
			case netplay.SlotRemote:
				h.host.Kick(i)
			case netplay.SlotOpen:
				h.host.Edit(func(l *netplay.LobbyState) {
					if l.Slots[i].Kind == netplay.SlotOpen {
						l.Slots[i].Kind, l.Slots[i].Skill, l.Slots[i].Name = netplay.SlotBot, botLevels[0].skill, fmt.Sprintf("Бот %d", i+1)
					}
				})
			case netplay.SlotBot:
				h.host.Edit(func(l *netplay.LobbyState) {
					sl := &l.Slots[i]
					switch {
					case sl.Skill < 0.5:
						sl.Skill = botLevels[1].skill
					case sl.Skill < 0.8:
						sl.Skill = botLevels[2].skill
					default:
						sl.Kind, sl.Name = netplay.SlotOpen, ""
					}
				})
			}
		}
	}
	if h.minusN.Update(a) && n > 2 {
		if lob.Slots[n-1].Kind == netplay.SlotRemote {
			h.host.Kick(n - 1)
		}
		h.host.Edit(func(l *netplay.LobbyState) {
			if len(l.Slots) > 2 && l.Slots[len(l.Slots)-1].Kind != netplay.SlotRemote {
				l.Slots = l.Slots[:len(l.Slots)-1]
			}
		})
	}
	if h.plusN.Update(a) && n < 4 {
		h.host.Edit(func(l *netplay.LobbyState) {
			if len(l.Slots) < 4 {
				k := len(l.Slots)
				l.Slots = append(l.Slots, netplay.LobbySlot{Kind: netplay.SlotOpen, Color: k % 4, Team: k % 2, Skill: botLevels[1].skill})
			}
		})
	}
	h.rules.Update(a, h, n)
	if h.frame%6 == 0 || h.rules.changed {
		h.sync(false)
		h.rules.changed = false
	}
	open := 0
	for _, s := range lob.Slots {
		if s.Kind == netplay.SlotOpen {
			open++
		}
	}
	h.startBtn.Enabled = open == 0
	if h.startBtn.Update(a) {
		h.sync(true)
		seats, err := h.host.Start()
		if err != nil {
			h.err = err.Error()
			return nil
		}
		lob = h.host.Lobby()
		cfg := h.rules.cfg.Clone()
		slots := make([]Slot, len(lob.Slots))
		for i, s := range lob.Slots {
			slots[i] = Slot{Name: s.Name, Human: s.Kind == netplay.SlotHost || s.Kind == netplay.SlotRemote, Skill: s.Skill, Color: s.Color, Team: s.Team}
			if !cfg.TeamsEnabled {
				slots[i].Team = i
			}
		}
		a.SaveSettings()
		a.SetScene(NewMatch(a, NewHostSession(h.host, cfg, h.rules.seed, slots, seats), nil))
		return nil
	}
	if h.backBtn.Update(a) || a.EscapeKey {
		h.host.Close()
		a.SetScene(NewMenu(a))
	}
	return nil
}

func (h *NetHost) Draw(a *App, dst *ebiten.Image) {
	dst.Fill(colorBG)
	lob := h.host.Lobby()
	a.TextB(dst, "Сетевая игра — ты хост", 40, 24, 40, colGold)
	for i, s := range lob.Slots {
		y := float64(144 + i*84)
		a.Rect(dst, 50, y-6, 570, 56, color.RGBA{32, 44, 72, 255})
		h.colorBtn[i].Col = gfx.TeamColors[s.Color%4]
		h.colorBtn[i].Draw(a, dst)
		switch s.Kind {
		case netplay.SlotHost:
			h.name.Draw(a, dst)
			h.typeBtn[i].Label = "Ты (хост)"
		case netplay.SlotOpen:
			a.Text(dst, "— ждём игрока —", 116, y+6, 22, colDim)
			h.typeBtn[i].Label = "Открыто"
		case netplay.SlotRemote:
			a.Text(dst, s.Name, 116, y+6, 22, colGood)
			h.typeBtn[i].Label = "Выгнать"
		case netplay.SlotBot:
			a.Text(dst, s.Name, 116, y+6, 22, colText)
			for _, bl := range botLevels {
				if s.Skill == bl.skill {
					h.typeBtn[i].Label = bl.name
				}
			}
		}
		h.typeBtn[i].Draw(a, dst)
		if h.rules.cfg.TeamsEnabled {
			h.teamBtn[i].Label = fmt.Sprintf("Ком. %c", 'A'+s.Team)
			h.teamBtn[i].Draw(a, dst)
		}
	}
	a.Text(dst, fmt.Sprintf("Игроков: %d", len(lob.Slots)), 112, 497, 24, colText)
	h.minusN.Draw(a, dst)
	h.plusN.Draw(a, dst)

	// connection info
	a.Rect(dst, 50, 548, 570, 78, color.RGBA{20, 28, 46, 255})
	a.Border(dst, 50.5, 548.5, 569, 77, 1, color.RGBA{110, 130, 170, 255})
	var parts []string
	radmin := ""
	for _, ip := range h.ips {
		if strings.HasPrefix(ip, "26.") {
			radmin = ip
		} else {
			parts = append(parts, ip)
		}
	}
	if radmin != "" {
		a.TextB(dst, "Radmin VPN: "+radmin+":"+h.port, 62, 556, 24, colGood)
		a.Text(dst, "Сообщи друзьям этот адрес. Другие: "+strings.Join(parts, ", "), 62, 590, 15, colDim)
	} else {
		a.Text(dst, "Адрес для друзей (IP:порт):", 62, 554, 18, colText)
		a.TextB(dst, strings.Join(parts, "  ")+" : "+h.port, 62, 578, 22, colGold)
		a.Text(dst, "Radmin VPN не найден — запусти его и открой это окно заново.", 62, 606, 14, colDim)
	}
	h.rules.Draw(a, dst)
	h.startBtn.Draw(a, dst)
	h.backBtn.Draw(a, dst)
	if !h.startBtn.Enabled {
		a.Text(dst, "Заполни или закрой пустые места (кнопка справа от места)", ScreenW-700, ScreenH-90, 17, colBad)
	}
	if h.err != "" {
		a.TextCenter(dst, h.err, ScreenW/2, ScreenH-130, 20, colBad, true)
	}
}

// ---- joining ---------------------------------------------------------------

type dialRes struct {
	cl  *netplay.Client
	err error
}

// NetJoin connects to a host and waits in its lobby.
type NetJoin struct {
	a        *App
	ip       *TextInput
	port     *TextInput
	name     *TextInput
	connect  *Button
	back     *Button
	stage    int // 0 form, 1 connecting, 2 lobby
	res      chan dialRes
	cl       *netplay.Client
	lob      netplay.LobbyState
	err      string
	cfg      *balance.Config
	cfgBytes []byte
}

// NewNetJoin shows the connection form.
func NewNetJoin(a *App) Scene {
	j := &NetJoin{a: a}
	j.ip = &TextInput{R: rectXYWH(ScreenW/2-210, 250, 420, 44), Text: a.Set.LastIP, Max: 40, Hint: "IP хоста, например 26.12.34.56", Allow: IPChars, Focus: true}
	j.port = &TextInput{R: rectXYWH(ScreenW/2-210, 330, 200, 44), Text: a.Set.Port, Max: 5, Hint: "порт", Allow: func(r rune) bool { return r >= '0' && r <= '9' }}
	j.name = &TextInput{R: rectXYWH(ScreenW/2-210, 410, 420, 44), Text: a.Set.Name, Max: 14, Hint: "Твоё имя"}
	j.connect = NewButton(ScreenW/2-210, 490, 420, 60, "Подключиться")
	j.connect.Col = color.RGBA{40, 130, 60, 255}
	j.back = NewButton(40, ScreenH-62, 200, 52, "Назад")
	return j
}

func (j *NetJoin) Update(a *App) error {
	switch j.stage {
	case 0:
		j.ip.Update(a)
		j.port.Update(a)
		j.name.Update(a)
		j.connect.Enabled = strings.TrimSpace(j.ip.Text) != ""
		if j.connect.Update(a) || (a.EnterKey && j.connect.Enabled) {
			addr := strings.TrimSpace(j.ip.Text)
			if !strings.Contains(addr, ":") {
				p := j.port.Text
				if p == "" {
					p = netplay.DefaultPort
				}
				addr += ":" + p
			}
			a.Set.LastIP = strings.TrimSpace(j.ip.Text)
			a.Set.Port = j.port.Text
			if j.name.Text != "" {
				a.Set.Name = j.name.Text
			}
			a.SaveSettings()
			j.res = make(chan dialRes, 1)
			name := a.Set.Name
			go func() {
				cl, err := netplay.Dial(addr, name)
				j.res <- dialRes{cl, err}
			}()
			j.stage, j.err = 1, ""
		}
		if j.back.Update(a) || a.EscapeKey {
			a.SetScene(NewMenu(a))
		}
	case 1:
		select {
		case r := <-j.res:
			if r.err != nil {
				j.err = "Не удалось подключиться: " + r.err.Error()
				j.stage = 0
			} else {
				j.cl, j.lob, j.stage = r.cl, r.cl.Lobby, 2
			}
		default:
		}
		if a.EscapeKey {
			j.stage = 0
		}
	case 2:
		msgs, ok := j.cl.Poll()
		for _, m := range msgs {
			switch m.Kind {
			case "lobby":
				if m.Lobby != nil {
					j.lob = *m.Lobby
				}
			case "start":
				if m.Lobby != nil {
					sess, err := NewClientSession(j.cl, *m.Lobby, m.Seat)
					if err != nil {
						j.err = "Ошибка старта: " + err.Error()
						j.cl.Close()
						j.stage = 0
						return nil
					}
					a.SetScene(NewMatch(a, sess, nil))
					return nil
				}
			case "reject":
				j.err = "Хост: " + m.Reason
				j.cl.Close()
				j.stage = 0
				return nil
			}
		}
		if !ok {
			j.err = "Связь с хостом потеряна"
			j.stage = 0
			return nil
		}
		if !bytes.Equal(j.cfgBytes, j.lob.CfgJSON) {
			j.cfgBytes = j.lob.CfgJSON
			j.cfg, _ = balance.Unmarshal(j.lob.CfgJSON)
		}
		if j.back.Update(a) || a.EscapeKey {
			j.cl.Close()
			j.stage = 0
		}
	}
	return nil
}

func (j *NetJoin) Draw(a *App, dst *ebiten.Image) {
	dst.Fill(colorBG)
	switch j.stage {
	case 0, 1:
		a.TextCenter(dst, "Подключение к игре", ScreenW/2, 110, 52, colGold, true)
		a.TextCenter(dst, "Запусти Radmin VPN, зайди в ту же сеть и введи IP хоста (обычно 26.x.x.x)", ScreenW/2, 190, 20, colDim, false)
		a.Text(dst, "IP:", ScreenW/2-260, 262, 22, colDim)
		a.Text(dst, "Порт:", ScreenW/2-285, 342, 22, colDim)
		a.Text(dst, "Имя:", ScreenW/2-270, 422, 22, colDim)
		j.ip.Draw(a, dst)
		j.port.Draw(a, dst)
		j.name.Draw(a, dst)
		if j.stage == 1 {
			a.TextCenter(dst, "Соединяюсь…", ScreenW/2, 580, 30, colGold, true)
		} else {
			j.connect.Draw(a, dst)
		}
		j.back.Draw(a, dst)
		if j.err != "" {
			a.TextCenter(dst, j.err, ScreenW/2, 580, 22, colBad, true)
		}
	case 2:
		a.TextB(dst, "Лобби хоста", 40, 24, 40, colGold)
		for i, s := range j.lob.Slots {
			y := float64(144 + i*84)
			a.Rect(dst, 50, y-6, 570, 56, color.RGBA{32, 44, 72, 255})
			a.Circle(dst, 80, y+20, 16, gfx.TeamColors[s.Color%4])
			label, col := "— ждём игрока —", color.Color(colDim)
			switch s.Kind {
			case netplay.SlotHost:
				label, col = s.Name+"  (хост)", colText
			case netplay.SlotRemote:
				label, col = s.Name, colGood
				if i == j.cl.Seat {
					label += "  ← это ты"
				}
			case netplay.SlotBot:
				label, col = s.Name+"  (бот)", colText
			}
			a.Text(dst, label, 112, y+6, 24, col)
		}
		if j.cfg != nil {
			drawCfgSummary(a, dst, j.cfg, 690, 120)
		}
		a.TextCenter(dst, "Ждём, пока хост нажмёт «СТАРТ»…", 960, 420, 26, colGold, true)
		j.back.Label = "Выйти"
		j.back.Draw(a, dst)
	}
}

type errScene struct{ msg string }

func (e *errScene) Update(a *App) error {
	if a.Click || a.EscapeKey || a.EnterKey {
		a.SetScene(NewMenu(a))
	}
	return nil
}

func (e *errScene) Draw(a *App, dst *ebiten.Image) {
	dst.Fill(colorBG)
	for i, l := range strings.Split(e.msg, "\n") {
		a.TextCenter(dst, l, ScreenW/2, 260+float64(i)*40, 26, colBad, true)
	}
	a.TextCenter(dst, "Щёлкни, чтобы вернуться", ScreenW/2, 440, 20, colDim, false)
}

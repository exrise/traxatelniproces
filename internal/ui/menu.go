package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Menu is the main menu scene.
type Menu struct {
	buttons []*Button
	acts    []func(a *App)
	name    *TextInput
	info    string
}

// NewMenu creates the main menu.
func NewMenu(a *App) *Menu {
	m := &Menu{}
	m.name = &TextInput{R: rectXYWH(ScreenW/2-150, 262, 300, 38), Text: a.Set.Name, Max: 16, Hint: "Имя игрока"}
	add := func(label, sub string, f func(a *App)) {
		b := NewButton(ScreenW/2-190, 330+len(m.buttons)*74, 380, 62, label)
		b.Sub = sub
		m.buttons = append(m.buttons, b)
		m.acts = append(m.acts, f)
	}
	add("Играть", "против ботов или на одном компьютере", func(a *App) { a.SetScene(NewLobby(a)) })
	add("Создать сетевую игру", "хост; друзья подключаются по IP (Radmin VPN)", func(a *App) { a.SetScene(NewNetHost(a)) })
	add("Подключиться к игре", "ввести IP хоста", func(a *App) { a.SetScene(NewNetJoin(a)) })
	add("Выход", "", func(a *App) { a.Quit = true })
	return m
}

func (m *Menu) Update(a *App) error {
	m.name.Update(a)
	if m.name.Text != "" {
		a.Set.Name = m.name.Text
	}
	for i, b := range m.buttons {
		if b.Update(a) {
			a.SaveSettings()
			m.acts[i](a)
			return nil
		}
	}
	return nil
}

func (m *Menu) Draw(a *App, dst *ebiten.Image) {
	dst.Fill(color.RGBA{24, 34, 54, 255})
	// decorative banner
	for i := 0; i < 8; i++ {
		a.Rect(dst, 0, float64(i)*90, ScreenW, 45, color.RGBA{28, 40, 64, 255})
	}
	a.TextCenter(dst, "СВИНОВОЙНА", ScreenW/2, 70, 92, color.RGBA{255, 214, 120, 255}, true)
	a.TextCenter(dst, "строй базу  •  командуй свиньями  •  разнеси всех", ScreenW/2, 176, 26, color.RGBA{190, 210, 240, 255}, false)
	a.Text(dst, "Имя:", ScreenW/2-205, 270, 22, colDim)
	m.name.Draw(a, dst)
	for _, b := range m.buttons {
		b.Draw(a, dst)
	}
	a.TextCenter(dst, "F11 — полный экран   •   M — музыка вкл/выкл   •   F1 — справка в игре", ScreenW/2, ScreenH-34, 16, colDim, false)
}

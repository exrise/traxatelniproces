package ui

import "github.com/hajimehoshi/ebiten/v2"

type stubScene struct{ msg string }

func (s *stubScene) Update(a *App) error {
	if a.Click || a.EscapeKey {
		a.SetScene(NewMenu(a))
	}
	return nil
}
func (s *stubScene) Draw(a *App, dst *ebiten.Image) {
	dst.Fill(colorBG)
	a.TextCenter(dst, s.msg, ScreenW/2, 300, 30, colText, true)
}

// NewNetHost is replaced by the real host lobby.
func NewNetHost(a *App) Scene { return &stubScene{"Сеть — скоро"} }

// NewNetJoin is replaced by the real join screen.
func NewNetJoin(a *App) Scene { return &stubScene{"Сеть — скоро"} }

// SvinoVoyna — a Forts x Worms style artillery game with pigs.
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"svinovoyna/internal/sfx"
	"svinovoyna/internal/ui"
)

func main() {
	sfx.Init()
	ui.PlaySound = sfx.Play
	ui.SetMusic = sfx.SetMusic
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Свиновойна")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)
	app := ui.NewApp()
	if app.Set.MusicOff {
		sfx.SetMusic(false)
	}
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}

// SvinoVoyna — a Forts x Worms style artillery game with pigs.
package main

import (
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"svinovoyna/internal/gfx"
	"svinovoyna/internal/sfx"
	"svinovoyna/internal/ui"
)

func main() {
	sfx.Init()
	ui.PlaySound = sfx.Play
	ui.SetMusic = sfx.SetMusic
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Свиновойна")
	ebiten.SetWindowIcon([]image.Image{gfx.Icon(32), gfx.Icon(64)})
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)
	ebiten.SetRunnableOnUnfocused(true) // the host must keep simulating when alt-tabbed
	app := ui.NewApp()
	if app.Set.MusicOff {
		sfx.SetMusic(false)
	}
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}

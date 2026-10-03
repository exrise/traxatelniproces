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
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Свиновойна")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(60)
	if err := ebiten.RunGame(ui.NewApp()); err != nil {
		log.Fatal(err)
	}
}

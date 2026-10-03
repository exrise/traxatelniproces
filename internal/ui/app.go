// Package ui contains the Ebitengine front-end: scenes, widgets and rendering.
package ui

import (
	"bytes"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"svinovoyna/internal/balance"
)

// Logical screen size.
const (
	ScreenW = 1280
	ScreenH = 720
)

// Scene is one screen of the game.
type Scene interface {
	Update(a *App) error
	Draw(a *App, dst *ebiten.Image)
}

// Settings persisted between runs.
type Settings struct {
	Name       string
	LastIP     string
	Port       string
	Volume     float64
	Fullscreen bool
	MusicOff   bool
}

// App is the root ebiten.Game.
type App struct {
	scene   Scene
	next    Scene
	regular *text.GoTextFaceSource
	bold    *text.GoTextFaceSource
	Presets []*balance.Config
	Dir     string // directory next to the executable
	Set     Settings
	tick    int

	// input snapshot (logical coordinates)
	MX, MY     int
	Click      bool // left button just pressed
	RightClick bool
	LeftDown   bool
	RightDown  bool
	LeftUp     bool
	Wheel      float64
	Quit       bool
	typed      []rune
	Typed      string
	Backspace  bool
	EnterKey   bool
	EscapeKey  bool

	imgCache map[string]*ebiten.Image
	white    *ebiten.Image
}

// NewApp builds the application with the menu as the first scene.
func NewApp() *App {
	exe, _ := os.Executable()
	a := &App{Dir: filepath.Dir(exe), imgCache: map[string]*ebiten.Image{}}
	if wd, err := os.Getwd(); err == nil {
		// when started with `go run` the exe is in a temp dir; prefer the working dir
		if _, err := os.Stat(filepath.Join(wd, "assets")); err == nil {
			a.Dir = wd
		}
	}
	a.regular, _ = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	a.bold, _ = text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	a.Presets = balance.Presets()
	a.Presets = append(a.Presets, balance.LoadCustom(filepath.Join(a.Dir, "presets"))...)
	a.Set = Settings{Name: "Хряк", Port: "27015", Volume: 0.7}
	a.loadSettings()
	a.white = ebiten.NewImage(1, 1)
	a.white.Fill(color.White)
	a.scene = NewMenu(a)
	if s := debugStart(a); s != nil {
		a.scene = s
	}
	return a
}

// SetScene switches scenes at the end of the current update.
func (a *App) SetScene(s Scene) { a.next = s }

// Layout implements ebiten.Game.
func (a *App) Layout(int, int) (int, int) { return ScreenW, ScreenH }

// Update implements ebiten.Game.
func (a *App) Update() error {
	a.tick++
	a.MX, a.MY = ebiten.CursorPosition()
	a.Click = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	a.RightClick = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	a.LeftDown = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	a.RightDown = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	a.LeftUp = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	_, wy := ebiten.Wheel()
	a.Wheel = wy
	a.typed = ebiten.AppendInputChars(a.typed[:0])
	a.Typed = string(a.typed)
	a.Backspace = inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || (inpututil.KeyPressDuration(ebiten.KeyBackspace) > 25 && a.tick%3 == 0)
	a.EnterKey = inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter)
	a.EscapeKey = inpututil.IsKeyJustPressed(ebiten.KeyEscape)
	if inpututil.IsKeyJustPressed(ebiten.KeyM) && SetMusic != nil {
		a.Set.MusicOff = !a.Set.MusicOff
		SetMusic(!a.Set.MusicOff)
		a.SaveSettings()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if a.scene != nil {
		if err := a.scene.Update(a); err != nil {
			return err
		}
	}
	if a.next != nil {
		a.scene, a.next = a.next, nil
	}
	if a.Quit {
		return ebiten.Termination
	}
	return nil
}

// Draw implements ebiten.Game.
func (a *App) Draw(dst *ebiten.Image) {
	if a.scene != nil {
		a.scene.Draw(a, dst)
		a.debugShot(dst)
	}
}

// Ticks returns frames since start (for animations).
func (a *App) Ticks() int { return a.tick }

// ---- drawing helpers -------------------------------------------------------

// Text draws a string with its top-left at (x,y).
func (a *App) Text(dst *ebiten.Image, s string, x, y, size float64, col color.Color) {
	a.text(dst, s, x, y, size, col, false)
}

// TextB draws bold text.
func (a *App) TextB(dst *ebiten.Image, s string, x, y, size float64, col color.Color) {
	a.text(dst, s, x, y, size, col, true)
}

func (a *App) text(dst *ebiten.Image, s string, x, y, size float64, col color.Color, bold bool) {
	src := a.regular
	if bold {
		src = a.bold
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(nc(col))
	text.Draw(dst, s, &text.GoTextFace{Source: src, Size: size}, op)
}

// TextShadow draws text with a dark drop shadow.
func (a *App) TextShadow(dst *ebiten.Image, s string, x, y, size float64, col color.Color, bold bool) {
	a.text(dst, s, x+1, y+1, size, color.RGBA{0, 0, 0, shadowAlpha(col)}, bold)
	a.text(dst, s, x, y, size, col, bold)
}

// TextCenter draws text horizontally centred on cx.
func (a *App) TextCenter(dst *ebiten.Image, s string, cx, y, size float64, col color.Color, bold bool) {
	w := a.Measure(s, size, bold)
	a.TextShadow(dst, s, cx-w/2, y, size, col, bold)
}

// Measure returns the width of s.
func (a *App) Measure(s string, size float64, bold bool) float64 {
	src := a.regular
	if bold {
		src = a.bold
	}
	w, _ := text.Measure(s, &text.GoTextFace{Source: src, Size: size}, 0)
	return w
}

// Rect fills a rectangle.
func (a *App) Rect(dst *ebiten.Image, x, y, w, h float64, col color.Color) {
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), nc(col), false)
}

// Border draws a rectangle outline.
func (a *App) Border(dst *ebiten.Image, x, y, w, h, t float64, col color.Color) {
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), float32(t), nc(col), false)
}

// Line draws a line.
func (a *App) Line(dst *ebiten.Image, x0, y0, x1, y1, t float64, col color.Color) {
	vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), float32(t), nc(col), false)
}

// Circle fills a disc.
func (a *App) Circle(dst *ebiten.Image, x, y, r float64, col color.Color) {
	vector.DrawFilledCircle(dst, float32(x), float32(y), float32(r), nc(col), true)
}

// Panel draws a framed translucent panel.
func (a *App) Panel(dst *ebiten.Image, r image.Rectangle) {
	a.Rect(dst, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), color.RGBA{16, 20, 30, 215})
	a.Border(dst, float64(r.Min.X)+0.5, float64(r.Min.Y)+0.5, float64(r.Dx())-1, float64(r.Dy())-1, 1, color.RGBA{110, 130, 170, 255})
}

// Img converts and caches an image under key.
func (a *App) Img(key string, mk func() *image.RGBA) *ebiten.Image {
	if im, ok := a.imgCache[key]; ok {
		return im
	}
	// user override: assets/<key>.png
	var im *ebiten.Image
	if p := filepath.Join(a.Dir, "assets", key+".png"); fileExists(p) {
		if f, err := os.Open(p); err == nil {
			if src, _, err := image.Decode(f); err == nil {
				im = ebiten.NewImageFromImage(src)
			}
			f.Close()
		}
	}
	if im == nil {
		im = ebiten.NewImageFromImage(mk())
	}
	a.imgCache[key] = im
	return im
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// In tests whether the mouse is in r.
func (a *App) In(r image.Rectangle) bool { return image.Pt(a.MX, a.MY).In(r) }

// Pulse returns a 0..1 oscillation.
func (a *App) Pulse(speed float64) float64 { return 0.5 + 0.5*math.Sin(float64(a.tick)*speed) }

// Play triggers a sound effect (no-op until the audio engine is attached).
func (a *App) Play(name string, vol float64) {
	if PlaySound != nil {
		PlaySound(name, vol*a.Set.Volume)
	}
}

// PlaySound is set by the audio package at startup.
var PlaySound func(name string, vol float64)

// nc treats color.RGBA literals as straight (non-premultiplied) colors, which is
// how all UI colors in this package are written.
func nc(c color.Color) color.Color {
	if r, ok := c.(color.RGBA); ok {
		return color.NRGBA{R: r.R, G: r.G, B: r.B, A: r.A}
	}
	return c
}

func shadowAlpha(c color.Color) uint8 {
	if r, ok := c.(color.RGBA); ok {
		return uint8(int(r.A) * 200 / 255)
	}
	return 200
}

// SetMusic is set by the audio package; it switches the background music.
var SetMusic func(on bool)

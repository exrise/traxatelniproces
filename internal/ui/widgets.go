package ui

import (
	"image"
	"image/color"
	"strings"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
)

// Button is a clickable rectangle.
type Button struct {
	R       image.Rectangle
	Label   string
	Sub     string
	Enabled bool
	On      bool // toggled / selected look
	Col     color.RGBA
	Hover   bool
	Size    float64
}

// NewButton creates an enabled button.
func NewButton(x, y, w, h int, label string) *Button {
	return &Button{R: image.Rect(x, y, x+w, y+h), Label: label, Enabled: true, Col: color.RGBA{52, 74, 120, 255}, Size: 20}
}

// Update returns true when clicked this frame.
func (b *Button) Update(a *App) bool {
	b.Hover = b.Enabled && a.In(b.R)
	return b.Hover && a.Click
}

// Draw renders the button.
func (b *Button) Draw(a *App, dst *ebiten.Image) {
	col := b.Col
	if !b.Enabled {
		col = color.RGBA{40, 44, 54, 255}
	} else if b.On {
		col = lighten(col, 60)
	} else if b.Hover {
		col = lighten(col, 34)
	}
	x, y, w, h := float64(b.R.Min.X), float64(b.R.Min.Y), float64(b.R.Dx()), float64(b.R.Dy())
	a.Rect(dst, x, y, w, h, col)
	a.Rect(dst, x, y, w, 2, color.RGBA{255, 255, 255, 40})
	bc := color.RGBA{150, 170, 210, 255}
	if b.On {
		bc = color.RGBA{255, 230, 120, 255}
	}
	a.Border(dst, x+0.5, y+0.5, w-1, h-1, 1, bc)
	tc := color.RGBA{240, 244, 250, 255}
	if !b.Enabled {
		tc = color.RGBA{120, 126, 140, 255}
	}
	if b.Sub != "" {
		a.TextCenter(dst, b.Label, x+w/2, y+h/2-b.Size*0.95, b.Size, tc, true)
		a.TextCenter(dst, b.Sub, x+w/2, y+h/2+1, b.Size*0.72, color.RGBA{200, 210, 230, 255}, false)
	} else {
		a.TextCenter(dst, b.Label, x+w/2, y+h/2-b.Size*0.62, b.Size, tc, true)
	}
}

// TextInput is a single-line text field.
type TextInput struct {
	R     image.Rectangle
	Text  string
	Max   int
	Focus bool
	Hint  string
	Allow func(r rune) bool
}

// Update handles focus and typing.
func (t *TextInput) Update(a *App) {
	if a.Click {
		t.Focus = a.In(t.R)
	}
	if !t.Focus {
		return
	}
	for _, r := range a.Typed {
		if r < 32 || (t.Allow != nil && !t.Allow(r)) {
			continue
		}
		if utf8.RuneCountInString(t.Text) < t.Max {
			t.Text += string(r)
		}
	}
	if a.Backspace && t.Text != "" {
		_, sz := utf8.DecodeLastRuneInString(t.Text)
		t.Text = t.Text[:len(t.Text)-sz]
	}
}

// Draw renders the field.
func (t *TextInput) Draw(a *App, dst *ebiten.Image) {
	x, y, w, h := float64(t.R.Min.X), float64(t.R.Min.Y), float64(t.R.Dx()), float64(t.R.Dy())
	a.Rect(dst, x, y, w, h, color.RGBA{14, 18, 28, 255})
	bc := color.RGBA{100, 116, 150, 255}
	if t.Focus {
		bc = color.RGBA{255, 230, 120, 255}
	}
	a.Border(dst, x+0.5, y+0.5, w-1, h-1, 1, bc)
	s := t.Text
	col := color.Color(color.RGBA{235, 240, 250, 255})
	if s == "" && !t.Focus {
		s, col = t.Hint, color.RGBA{110, 118, 140, 255}
	}
	if t.Focus && (a.Ticks()/30)%2 == 0 {
		s += "|"
	}
	a.Text(dst, s, x+8, y+h/2-10, 20, col)
}

// IPChars allows only characters valid in an IPv4 address / host with port.
func IPChars(r rune) bool {
	return strings.ContainsRune("0123456789.:abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-", r)
}

func rgba(r, g, b, a uint8) color.RGBA { return color.RGBA{r, g, b, a} }

var (
	colText = color.RGBA{235, 240, 250, 255}
	colDim  = color.RGBA{160, 170, 190, 255}
	colGold = color.RGBA{255, 214, 90, 255}
	colGood = color.RGBA{120, 230, 130, 255}
	colBad  = color.RGBA{255, 110, 100, 255}
)

func rectXYWH(x, y, w, h int) image.Rectangle { return image.Rect(x, y, x+w, y+h) }

func lighten(c color.RGBA, d int) color.RGBA {
	f := func(v uint8) uint8 { return uint8(min(255, int(v)+d)) }
	return color.RGBA{f(c.R), f(c.G), f(c.B), 255}
}

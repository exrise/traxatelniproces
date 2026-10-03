package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"svinovoyna/internal/balance"
	"svinovoyna/internal/gfx"
	"svinovoyna/internal/sim"
)

type palItem struct {
	kind string // "unit", "struct", "item"
	id   string
}

var buildTabs = []string{"Юниты", "Оружие", "ПВО", "Защита", "Экономика", "Удары"}

// BuildUI is the build-phase interface.
type BuildUI struct {
	m        *Match
	tab      int
	tabBtn   []*Button
	sel      *palItem
	moving   int // unit being moved or -1
	ready    *Button
	repair   *Button
	scroll   int
	err      string
	errT     int
	hoverTip *palItem
	items    []palItem
}

// NewBuildUI creates the build interface.
func NewBuildUI(m *Match) *BuildUI {
	b := &BuildUI{m: m, moving: -1}
	for i, t := range buildTabs {
		btn := NewButton(12+(i%3)*104, 66+(i/3)*38, 100, 34, t)
		btn.Size = 16
		b.tabBtn = append(b.tabBtn, btn)
	}
	b.ready = NewButton(ScreenW-250, 8, 236, 48, "ГОТОВ")
	b.ready.Col = color.RGBA{40, 130, 60, 255}
	b.repair = NewButton(ScreenW-470, 8, 210, 48, "Починить всё")
	b.repair.Size = 18
	return b
}

// Enter is called when a build phase starts.
func (b *BuildUI) Enter() {
	b.sel = nil
	b.moving = -1
	b.CenterOn(b.m.Me())
}

// CenterOn points the camera at a player's build zone.
func (b *BuildUI) CenterOn(pid int) {
	p := b.m.w.Players[pid]
	cam := &b.m.view.Cam
	cam.Zoom = 1.7
	cam.X = float64(p.ZoneX0+p.ZoneX1)/2 - (ScreenW/2+160)/cam.Zoom
	cam.Y = sim.PlateauY - 330
	cam.Clamp()
}

func (b *BuildUI) itemsForTab() []palItem {
	cfg := b.m.w.Cfg
	var out []palItem
	switch b.tab {
	case 0:
		for _, u := range cfg.Units {
			out = append(out, palItem{"unit", u.ID})
		}
	case 1, 2, 3, 4:
		for _, s := range cfg.Structs {
			ok := false
			switch b.tab {
			case 1:
				ok = s.Kind == balance.SWeapon
			case 2:
				ok = s.Kind == balance.SAA || s.Kind == balance.SJammer
			case 3:
				ok = s.Kind == balance.SBlock || s.Kind == balance.SBunker || s.Kind == balance.SNet || s.Kind == balance.SWindow
			case 4:
				ok = s.Kind == balance.SEco
			}
			if ok && s.Tier < 50 {
				out = append(out, palItem{"struct", s.ID})
			}
		}
	case 5:
		for _, w := range cfg.Weapons {
			if w.Cost > 0 && (w.Kind == balance.KindAirstrike || w.Kind == balance.KindGeran) {
				out = append(out, palItem{"item", w.ID})
			}
		}
	}
	return out
}

func (b *BuildUI) itemName(it palItem) string {
	cfg := b.m.w.Cfg
	switch it.kind {
	case "unit":
		return cfg.U(it.id).Name
	case "struct":
		return cfg.S(it.id).Name
	}
	return cfg.W(it.id).Name
}

func (b *BuildUI) itemCost(it palItem) int {
	cfg := b.m.w.Cfg
	switch it.kind {
	case "unit":
		return cfg.U(it.id).Cost
	case "struct":
		return cfg.S(it.id).Cost
	}
	return cfg.W(it.id).Cost
}

// itemState returns how many exist, the limit (0 = none) and an unavailable reason.
func (b *BuildUI) itemState(pid int, it palItem) (have, limit int, reason string) {
	w := b.m.w
	switch it.kind {
	case "unit":
		d := w.Cfg.U(it.id)
		have, limit = w.CountUnits(pid, it.id), d.Max
		if w.CountUnits(pid, "") >= w.Cfg.MaxUnits {
			reason = "лимит юнитов"
		} else if d.Max > 0 && have >= d.Max {
			reason = "лимит"
		}
	case "struct":
		d := w.Cfg.S(it.id)
		have, limit = w.CountStructs(pid, it.id), d.Max
		if d.Tier > w.BuildNo {
			reason = fmt.Sprintf("со стройки %d", d.Tier)
		} else if d.Max > 0 && have >= d.Max {
			reason = "лимит"
		}
	case "item":
		have = w.Players[pid].Items[it.id]
	}
	if reason == "" && b.itemCost(it) > w.Players[pid].Money {
		reason = "нет денег"
	}
	return
}

func (b *BuildUI) panelRect() image.Rectangle { return image.Rect(0, 58, 330, ScreenH) }

func (b *BuildUI) rowRect(i int) image.Rectangle {
	y := 150 + i*56 - b.scroll
	return image.Rect(8, y, 322, y+52)
}

func (b *BuildUI) flash(s string) { b.err, b.errT = s, 150 }

// Update handles input for the build phase.
func (b *BuildUI) Update(a *App) {
	m := b.m
	w := m.w
	pid := m.Me()
	p := w.Players[pid]
	cam := &m.view.Cam
	b.items = b.itemsForTab()
	b.hoverTip = nil
	if b.errT > 0 {
		b.errT--
	}

	for i, tb := range b.tabBtn {
		if tb.Update(a) {
			b.tab, b.scroll, b.sel = i, 0, nil
		}
		tb.On = i == b.tab
	}
	for k := 0; k < 6; k++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(k)) {
			b.tab, b.scroll, b.sel = k, 0, nil
		}
	}
	// palette
	overPalette := a.In(b.panelRect())
	if overPalette && a.Wheel != 0 {
		maxScroll := max(0, len(b.items)*56-(ScreenH-160))
		b.scroll = max(0, min(maxScroll, b.scroll-int(a.Wheel*40)))
	}
	for i, it := range b.items {
		r := b.rowRect(i)
		if a.In(r) {
			itc := it
			b.hoverTip = &itc
			if a.Click {
				_, _, reason := b.itemState(pid, it)
				if it.kind == "item" {
					if reason == "" || reason == "нет денег" {
						if reason == "" {
							m.sess.Send(sim.Command{Player: pid, Type: sim.CmdBuyItem, Def: it.id})
						} else {
							b.flash("Не хватает денег")
						}
					}
				} else if reason != "" && reason != "нет денег" {
					b.flash("Недоступно: " + reason)
				} else {
					itc := it
					b.sel = &itc
					b.moving = -1
				}
			}
			if a.RightClick && it.kind == "item" {
				m.sess.Send(sim.Command{Player: pid, Type: sim.CmdSellItem, Def: it.id})
			}
		}
	}

	// ready / repair
	if b.ready.Update(a) {
		m.sess.Send(sim.Command{Player: pid, Type: sim.CmdReady, Flag: !p.Ready})
	}
	b.ready.Label = "ГОТОВ"
	if p.Ready {
		b.ready.Label = "Ждём остальных…"
	}
	b.ready.On = p.Ready
	if b.repair.Update(a) {
		for _, s := range w.StructsOf(pid) {
			if w.RepairCost(s) > 0 {
				m.sess.Send(sim.Command{Player: pid, Type: sim.CmdRepair, ID: s.ID})
			}
		}
	}

	// camera
	b.updateCamera(a, p)

	if overPalette || a.In(image.Rect(0, 0, ScreenW, 58)) || p.Ready {
		if a.EscapeKey {
			b.sel = nil
		}
		return
	}
	wp := cam.ToWorld(float64(a.MX), float64(a.MY))
	if a.EscapeKey {
		b.sel, b.moving = nil, -1
	}
	if a.RightClick {
		// sell what is under the cursor
		if u := b.unitAt(pid, wp); u != nil {
			m.sess.Send(sim.Command{Player: pid, Type: sim.CmdSell, Flag: true, ID: u.ID})
		} else if s := w.StructAtPx(wp.X, wp.Y); s != nil && s.Owner == pid && s.Def != "hq" {
			m.sess.Send(sim.Command{Player: pid, Type: sim.CmdSell, ID: s.ID})
		} else {
			b.sel, b.moving = nil, -1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		if s := w.StructAtPx(wp.X, wp.Y); s != nil && s.Owner == pid {
			m.sess.Send(sim.Command{Player: pid, Type: sim.CmdRepair, ID: s.ID})
		}
	}
	if !a.Click {
		return
	}
	switch {
	case b.moving >= 0:
		m.sess.Send(sim.Command{Player: pid, Type: sim.CmdMoveUnit, ID: b.moving, X: wp.X, Y: wp.Y})
		b.moving = -1
	case b.sel == nil:
		if u := b.unitAt(pid, wp); u != nil {
			b.moving = u.ID
		}
	case b.sel.kind == "unit":
		if _, err := w.CanPlaceUnit(pid, b.sel.id, wp.X, wp.Y); err != nil {
			b.flash(err.Error())
		} else {
			m.sess.Send(sim.Command{Player: pid, Type: sim.CmdPlaceUnit, Def: b.sel.id, X: wp.X, Y: wp.Y})
		}
	case b.sel.kind == "struct":
		cx, cy := b.ghostCell(wp, b.sel.id)
		if err := w.CanPlaceStruct(pid, b.sel.id, cx, cy); err != nil {
			b.flash(err.Error())
		} else {
			m.sess.Send(sim.Command{Player: pid, Type: sim.CmdPlaceStruct, Def: b.sel.id, CX: cx, CY: cy})
		}
	}
}

func (b *BuildUI) ghostCell(wp sim.Vec, def string) (int, int) {
	w := b.m.w
	d := w.Cfg.S(def)
	cx := int(math.Floor(wp.X/sim.Cell)) - d.W/2
	row := int(math.Floor(wp.Y/sim.Cell)) - d.H/2
	cy := w.DropCell(cx, d.W, d.H, row)
	return cx, cy
}

func (b *BuildUI) unitAt(pid int, wp sim.Vec) *sim.Unit {
	for _, u := range b.m.w.UnitsOf(pid) {
		if math.Abs(wp.X-u.Pos.X) < 12 && wp.Y > u.Pos.Y-26 && wp.Y < u.Pos.Y+4 {
			return u
		}
	}
	return nil
}

func (b *BuildUI) updateCamera(a *App, p *sim.Player) {
	cam := &b.m.view.Cam
	if a.RightDown && a.MX > 330 {
		// right-drag pans, but a plain right-click sells; require movement
	}
	if a.Wheel != 0 && a.MX > 330 {
		cam.ZoomAt(math.Pow(1.12, a.Wheel), float64(a.MX), float64(a.MY))
	}
	spd := 9 / cam.Zoom
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		cam.X -= spd
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		cam.X += spd
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		cam.Y -= spd
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		cam.Y += spd
	}
	vw := ScreenW / cam.Zoom
	cam.X = math.Max(float64(p.ZoneX0)-420, math.Min(float64(p.ZoneX1)+420-vw+330/cam.Zoom, cam.X))
	cam.Y = math.Max(150, math.Min(sim.MapH-ScreenH/cam.Zoom-40, cam.Y))
}

// Draw renders the build HUD and placement ghost.
func (b *BuildUI) Draw(a *App, dst *ebiten.Image) {
	m := b.m
	w := m.w
	pid := m.Me()
	p := w.Players[pid]
	cam := &m.view.Cam
	tc := gfx.TeamColors[p.Color%4]

	// own zone outline
	zx0, _ := cam.ToScreen(sim.Vec{X: float64(p.ZoneX0), Y: 0})
	zx1, _ := cam.ToScreen(sim.Vec{X: float64(p.ZoneX1), Y: 0})
	a.Rect(dst, zx0, 58, 2, ScreenH, color.RGBA{tc.R, tc.G, tc.B, 150})
	a.Rect(dst, zx1-2, 58, 2, ScreenH, color.RGBA{tc.R, tc.G, tc.B, 150})
	a.Rect(dst, zx0, 58, zx1-zx0, ScreenH, color.RGBA{tc.R, tc.G, tc.B, 18})
	a.TextCenter(dst, "ТВОЯ ЗОНА СТРОЙКИ", (zx0+zx1)/2, 70, 18, color.RGBA{tc.R, tc.G, tc.B, 220}, true)

	if !p.Ready && !a.In(b.panelRect()) && a.MY > 58 {
		b.drawGhost(a, dst, pid)
	}

	// top bar
	a.Rect(dst, 0, 0, ScreenW, 58, color.RGBA{14, 18, 28, 240})
	a.Circle(dst, 28, 29, 11, tc)
	a.TextB(dst, p.Name, 48, 14, 26, colText)
	a.TextB(dst, fmt.Sprintf("$%d", p.Money), 250, 12, 32, colGold)
	t := m.sess.BuildSeatTime()
	tcol := color.Color(colText)
	if t < 15 {
		tcol = colBad
	}
	a.TextB(dst, fmt.Sprintf("Стройка %d   %d:%02d", w.BuildNo, int(t)/60, int(t)%60), 400, 14, 24, tcol)
	if b.errT > 0 {
		a.TextCenter(dst, b.err, ScreenW/2+160, 66, 22, colBad, true)
	}
	b.drawReadyList(a, dst)
	b.repair.Enabled = !p.Ready
	b.repair.Draw(a, dst)
	b.ready.Draw(a, dst)

	// palette
	a.Panel(dst, b.panelRect())
	for _, tb := range b.tabBtn {
		tb.Draw(a, dst)
	}
	clipTop, clipBot := 146, ScreenH
	for i, it := range b.items {
		r := b.rowRect(i)
		if r.Max.Y < clipTop || r.Min.Y > clipBot-4 {
			continue
		}
		have, limit, reason := b.itemState(pid, it)
		bg := color.RGBA{34, 44, 70, 255}
		if b.sel != nil && *b.sel == it {
			bg = color.RGBA{70, 90, 40, 255}
		} else if a.In(r) {
			bg = color.RGBA{50, 64, 100, 255}
		}
		a.Rect(dst, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), bg)
		b.drawIcon(a, dst, it, pid, r.Min.X+4, r.Min.Y+4, 44)
		col := color.Color(colText)
		if reason != "" {
			col = colDim
		}
		a.TextB(dst, b.itemName(it), float64(r.Min.X+56), float64(r.Min.Y+4), 19, col)
		cc := color.Color(colGold)
		if reason == "нет денег" {
			cc = colBad
		}
		a.Text(dst, fmt.Sprintf("$%d", b.itemCost(it)), float64(r.Min.X+56), float64(r.Min.Y+28), 19, cc)
		info := ""
		if it.kind == "item" {
			info = fmt.Sprintf("в запасе: %d", have)
		} else if limit > 0 {
			info = fmt.Sprintf("%d/%d", have, limit)
		} else if have > 0 {
			info = fmt.Sprintf("есть: %d", have)
		}
		if reason != "" && reason != "нет денег" {
			info = reason
		}
		a.Text(dst, info, float64(r.Max.X)-a.Measure(info, 16, false)-8, float64(r.Min.Y+30), 16, colDim)
	}
	// tooltip
	tip := b.hoverTip
	if tip == nil {
		tip = b.sel
	}
	if tip != nil {
		lines := b.describe(*tip)
		h := 28 + len(lines)*22
		r := image.Rect(340, ScreenH-h-10, 340+420, ScreenH-10)
		a.Panel(dst, r)
		a.TextB(dst, b.itemName(*tip), float64(r.Min.X+10), float64(r.Min.Y+4), 21, colGold)
		for i, ln := range lines {
			a.Text(dst, ln, float64(r.Min.X+10), float64(r.Min.Y+30+i*22), 17, colText)
		}
	}
	hint := "ЛКМ — поставить   ПКМ — продать   R — починить   колесо — масштаб   WASD — камера"
	if b.moving >= 0 {
		hint = "Кликни, куда переставить юнита (Esc — отмена)"
	} else if b.sel == nil {
		hint = "Выбери предмет слева. Клик по своему юниту — переставить. ПКМ по объекту — продать."
	}
	a.Text(dst, hint, 346, 66+24, 16, color.RGBA{230, 235, 245, 200})
}

func (b *BuildUI) drawIcon(a *App, dst *ebiten.Image, it palItem, pid, x, y, size int) {
	w := b.m.w
	var img *ebiten.Image
	col := w.Players[pid].Color % 4
	switch it.kind {
	case "unit":
		img = a.Img(fmt.Sprintf("pig_%d_%s_0", col, it.id), func() *image.RGBA { return gfx.PigSprite(col, it.id, 0) })
	case "struct":
		d := w.Cfg.S(it.id)
		img = a.Img("s_"+it.id, func() *image.RGBA { return gfx.StructSprite(it.id, d.W, d.H) })
	default:
		wd := w.Cfg.W(it.id)
		a.Rect(dst, float64(x), float64(y), float64(size), float64(size), color.RGBA{30, 36, 52, 255})
		sym := map[string]string{"fab": "ФАБ", "kab": "КАБ", "geran": "ГЕР"}[it.id]
		a.TextCenter(dst, sym, float64(x+size/2), float64(y+size/2-11), 19, colGold, true)
		_ = wd
		return
	}
	bw, bh := img.Bounds().Dx(), img.Bounds().Dy()
	sc := math.Min(float64(size)/float64(bw), float64(size)/float64(bh))
	if sc > 2 {
		sc = 2
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(sc, sc)
	op.GeoM.Translate(float64(x)+(float64(size)-float64(bw)*sc)/2, float64(y)+(float64(size)-float64(bh)*sc)/2)
	dst.DrawImage(img, op)
}

func (b *BuildUI) describe(it palItem) []string {
	cfg := b.m.w.Cfg
	var out []string
	add := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	switch it.kind {
	case "unit":
		u := cfg.U(it.id)
		add("%s", u.Desc)
		add("Здоровье: %.0f", u.HP)
		var ws []string
		for _, id := range u.Weapons {
			ws = append(ws, cfg.W(id).Name)
		}
		add("Оружие: %s", strings.Join(ws, ", "))
		if u.Max > 0 {
			add("Лимит на игрока: %d", u.Max)
		}
	case "item":
		wd := cfg.W(it.id)
		add("Покупается заранее, вызывается в бою из штаба")
		add("Урон %.0f, радиус %.0f, зарядов %d", wd.Damage, wd.Radius, max(1, wd.Count))
		add("Точность х3 лучше с живым Наводчиком")
		add("ПКМ — вернуть половину цены")
	case "struct":
		d := cfg.S(it.id)
		add("%s", d.Desc)
		add("Прочность: %.0f   Размер: %d×%d", d.HP, d.W, d.H)
		switch d.Kind {
		case balance.SWeapon:
			wd := cfg.W(d.Weapon)
			add("Урон %.0f, радиус %.0f, боезапас %d", wd.Damage, wd.Radius, wd.Ammo)
			if wd.Count > 1 {
				add("Снарядов в залпе: %d", wd.Count)
			}
		case balance.SAA, balance.SJammer:
			add("Радиус %.0f (по высоте больше), боезапас %d", d.AARange, d.AAAmmo)
			var hs []string
			for _, cl := range []balance.AAClass{balance.ClassDrone, balance.ClassAir, balance.ClassRocket, balance.ClassBallis, balance.ClassOreshnik} {
				if v, ok := d.AAHit[cl]; ok {
					hs = append(hs, fmt.Sprintf("%s %.0f%%", aaClassName(cl), v*100))
				}
			}
			add("Перехват: %s", strings.Join(hs, ", "))
			if d.Weapon != "" {
				wd := cfg.W(d.Weapon)
				add("Стреляет и вручную: урон %.0f ×%d, боезапас %d", wd.Damage, max(1, wd.Count), wd.Ammo)
			}
		case balance.SEco:
			add("Доход: +$%d каждый твой ход", d.Income)
		}
		if d.Max > 0 {
			add("Лимит на игрока: %d", d.Max)
		}
	}
	sort.Strings(nil)
	return out
}

func (b *BuildUI) drawGhost(a *App, dst *ebiten.Image, pid int) {
	if b.sel == nil && b.moving < 0 {
		return
	}
	m := b.m
	w := m.w
	cam := &m.view.Cam
	wp := cam.ToWorld(float64(a.MX), float64(a.MY))
	z := cam.Zoom
	if b.moving >= 0 || (b.sel != nil && b.sel.kind == "unit") {
		def := ""
		var pos sim.Vec
		var err error
		if b.moving >= 0 {
			def = w.Units[b.moving].Def
			p, ok := w.DropUnitPos(wp.X, wp.Y)
			pos = p
			if !ok {
				err = sim.ErrBlocked
			}
		} else {
			def = b.sel.id
			pos, err = w.CanPlaceUnit(pid, def, wp.X, wp.Y)
			if err != nil {
				pos, _ = w.DropUnitPos(wp.X, wp.Y)
			}
		}
		col := w.Players[pid].Color % 4
		img := a.Img(fmt.Sprintf("pig_%d_%s_0", col, def), func() *image.RGBA { return gfx.PigSprite(col, def, 0) })
		sx, sy := cam.ToScreen(pos)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-gfx.PigW/2, -gfx.PigH)
		op.GeoM.Scale(z, z)
		op.GeoM.Translate(sx, sy+z)
		if err != nil {
			op.ColorScale.Scale(1, 0.4, 0.4, 0.7)
		} else {
			op.ColorScale.Scale(0.7, 1, 0.7, 0.8)
		}
		dst.DrawImage(img, op)
		if err != nil {
			a.TextCenter(dst, err.Error(), sx, sy-60*z, 18, colBad, true)
		}
		return
	}
	if b.sel != nil && b.sel.kind == "struct" {
		d := w.Cfg.S(b.sel.id)
		cx, cy := b.ghostCell(wp, b.sel.id)
		err := w.CanPlaceStruct(pid, b.sel.id, cx, cy)
		img := a.Img("s_"+b.sel.id, func() *image.RGBA { return gfx.StructSprite(b.sel.id, d.W, d.H) })
		sx, sy := cam.ToScreen(sim.Vec{X: float64(cx * sim.Cell), Y: float64(cy * sim.Cell)})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(z, z)
		op.GeoM.Translate(sx, sy)
		bc := color.RGBA{120, 255, 120, 255}
		if err != nil {
			op.ColorScale.Scale(1, 0.4, 0.4, 0.7)
			bc = color.RGBA{255, 100, 100, 255}
		} else {
			op.ColorScale.Scale(0.8, 1, 0.8, 0.8)
		}
		dst.DrawImage(img, op)
		a.Border(dst, sx, sy, float64(d.W*sim.Cell)*z, float64(d.H*sim.Cell)*z, 2, bc)
		if d.AARange > 0 {
			ccx, ccy := sx+float64(d.W*sim.Cell)*z/2, sy+float64(d.H*sim.Cell)*z/2
			vecEllipseOutline(a, dst, ccx, ccy, d.AARange*z, d.AARange*z*2.5, color.RGBA{120, 220, 255, 120})
		}
		if err != nil {
			a.TextCenter(dst, err.Error(), sx+float64(d.W*sim.Cell)*z/2, sy-26*z, 18, colBad, true)
		}
	}
}

func vecCircleOutline(a *App, dst *ebiten.Image, x, y, r float64, col color.RGBA) {
	const n = 64
	for i := 0; i < n; i += 2 {
		a0, a1 := float64(i)/n*2*math.Pi, float64(i+1)/n*2*math.Pi
		a.Line(dst, x+math.Cos(a0)*r, y+math.Sin(a0)*r, x+math.Cos(a1)*r, y+math.Sin(a1)*r, 2, col)
	}
}

func vecEllipseOutline(a *App, dst *ebiten.Image, x, y, rx, ry float64, col color.RGBA) {
	const n = 96
	for i := 0; i < n; i += 2 {
		a0, a1 := float64(i)/n*2*math.Pi, float64(i+1)/n*2*math.Pi
		a.Line(dst, x+math.Cos(a0)*rx, y+math.Sin(a0)*ry, x+math.Cos(a1)*rx, y+math.Sin(a1)*ry, 2, col)
	}
}

// drawReadyList shows who is still building (top bar, middle).
func (b *BuildUI) drawReadyList(a *App, dst *ebiten.Image) {
	w := b.m.w
	x := 640.0
	for _, p := range w.Players {
		col := gfx.TeamColors[p.Color%4]
		a.Circle(dst, x+8, 29, 7, col)
		st, sc := "строит", color.Color(colDim)
		if p.Ready {
			st, sc = "готов", colGood
		}
		a.Text(dst, p.Name, x+20, 8, 15, colText)
		a.Text(dst, st, x+20, 28, 15, sc)
		x += 20 + math.Max(a.Measure(p.Name, 15, false), 50) + 12
	}
}

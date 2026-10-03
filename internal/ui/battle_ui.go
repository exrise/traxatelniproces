package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"svinovoyna/internal/balance"
	"svinovoyna/internal/gfx"
	"svinovoyna/internal/sim"
)

type wopt struct {
	id, name, kind string
	ammo           int // -1 unlimited
}

// BattleUI is the battle-phase interface.
type BattleUI struct {
	m          *Match
	weapon     string
	charging   bool
	charge     float64
	targetX    float64
	hasTarget  bool
	walkDir    int
	planWalk   float64
	camManual  float64
	dragging   bool
	dragX      int
	dragY      int
	dragCamX   float64
	dragCamY   float64
	moved      bool
	showTraj   bool
	endBtn     *Button
	fireBtn    *Button
	opts       []wopt
	turnKey    int
	focusT     float64
	holdT      float64
	lastAim    float64
	mini       *ebiten.Image
	miniAt     int
	hint       string
	hintT      int
	planned    bool
	planUnit   int
	planStruct int
}

// NewBattleUI creates the battle interface.
func NewBattleUI(m *Match) *BattleUI {
	b := &BattleUI{m: m, showTraj: true, turnKey: -1, planUnit: -1, planStruct: -1}
	b.endBtn = NewButton(ScreenW-290, ScreenH-168, 270, 44, "Закончить ход (E)")
	b.endBtn.Size = 18
	b.fireBtn = NewButton(ScreenW/2-110, ScreenH-168, 220, 46, "ОГОНЬ! (Пробел)")
	b.fireBtn.Col = color.RGBA{170, 50, 40, 255}
	b.fireBtn.Size = 20
	return b
}

// Enter prepares the UI when the battle starts.
func (b *BattleUI) Enter() {
	cam := &b.m.view.Cam
	cam.Zoom = 1.05
	b.turnKey = -1
	b.camManual = 0
}

func (b *BattleUI) cameraIdle(a *App) { b.handleCameraInput(a) }

func (b *BattleUI) flash(s string) { b.hint, b.hintT = s, 180 }

func (b *BattleUI) origin(u *sim.Unit, s *sim.Struct) sim.Vec {
	if s != nil {
		return b.m.w.StructMuzzle(s)
	}
	return sim.Vec{X: u.Pos.X, Y: u.Pos.Y - 11}
}

func (b *BattleUI) actors() (*sim.Unit, *sim.Struct) {
	w := b.m.w
	su, ss := w.SelUnit, w.SelStruct
	if w.Cfg.TurnMode == balance.TurnSimultaneous {
		su, ss = b.planUnit, b.planStruct
	}
	if ss >= 0 && ss < len(w.Structs) && w.Structs[ss].Alive {
		return nil, w.Structs[ss]
	}
	if su >= 0 && su < len(w.Units) && w.Units[su].Alive {
		return w.Units[su], nil
	}
	return nil, nil
}

func (b *BattleUI) buildOpts(pid int) {
	w := b.m.w
	b.opts = b.opts[:0]
	u, s := b.actors()
	if s != nil {
		wd := w.Cfg.W(w.Cfg.S(s.Def).Weapon)
		b.opts = append(b.opts, wopt{wd.ID, wd.Name, "struct", s.Ammo})
	} else if u != nil {
		for _, id := range w.Cfg.U(u.Def).Weapons {
			wd := w.Cfg.W(id)
			b.opts = append(b.opts, wopt{id, wd.Name, "unit", w.UnitAmmoLeft(u, id)})
		}
	}
	if w.HasHQ(pid) {
		for _, id := range []string{"fab", "kab", "geran"} {
			if n := w.Players[pid].Items[id]; n > 0 {
				b.opts = append(b.opts, wopt{id, w.Cfg.W(id).Name, "item", n})
			}
		}
	}
	found := false
	for _, o := range b.opts {
		if o.id == b.weapon && o.ammo != 0 {
			found = true
		}
	}
	if !found {
		b.weapon = ""
		for _, o := range b.opts {
			if o.ammo != 0 {
				b.weapon = o.id
				break
			}
		}
	}
	if u != nil && b.weapon != "" && w.Cfg.W(b.weapon) != nil {
		for _, o := range b.opts {
			if o.id == b.weapon && o.kind == "unit" {
				b.m.view.UnitWeapon[u.ID] = b.weapon
			}
		}
	}
}

func needsPower(k balance.WeaponKind) bool { return k == balance.KindShell || k == balance.KindSalvo }

func (b *BattleUI) handleCameraInput(a *App) {
	cam := &b.m.view.Cam
	if a.Wheel != 0 {
		cam.ZoomAt(math.Pow(1.12, a.Wheel), float64(a.MX), float64(a.MY))
		b.camManual = 4
	}
	if a.RightClick || (a.Click && b.overMinimap(a)) {
		b.dragging = a.RightClick
		b.dragX, b.dragY, b.dragCamX, b.dragCamY = a.MX, a.MY, cam.X, cam.Y
		b.moved = false
	}
	if b.dragging {
		if !a.RightDown {
			b.dragging = false
		} else {
			dx, dy := float64(a.MX-b.dragX), float64(a.MY-b.dragY)
			if math.Abs(dx)+math.Abs(dy) > 3 {
				b.moved = true
			}
			cam.X = b.dragCamX - dx/cam.Zoom
			cam.Y = b.dragCamY - dy/cam.Zoom
			b.camManual = 4
		}
	}
	if a.LeftDown && b.overMinimap(a) {
		r := b.miniRect()
		fx := float64(a.MX-r.Min.X) / float64(r.Dx())
		cam.CenterOn(sim.Vec{X: fx * cam.WorldW, Y: 600}, 1)
		b.camManual = 4
	}
	cam.Clamp()
}

func (b *BattleUI) miniRect() image.Rectangle {
	return image.Rect(ScreenW-290, ScreenH-110, ScreenW-14, ScreenH-14)
}

func (b *BattleUI) overMinimap(a *App) bool { return a.In(b.miniRect()) }

// Update handles input and camera in battle.
func (b *BattleUI) Update(a *App) {
	m := b.m
	w := m.w
	cam := &m.view.Cam
	if b.hintT > 0 {
		b.hintT--
	}
	if b.camManual > 0 {
		b.camManual -= sim.Dt
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		b.showTraj = !b.showTraj
	}
	key := w.BattleNo*100000 + w.Round*100 + w.OrderPos
	if w.Cfg.TurnMode == balance.TurnSimultaneous {
		key = w.BattleNo*100000 + w.Round*100 + int(w.Stage)
	}
	if key != b.turnKey {
		b.turnKey = key
		b.charging, b.charge, b.hasTarget, b.planWalk, b.planned = false, 0, false, 0, false
		b.focusT = 2.2
		b.camManual = 0
		b.weapon = ""
		if w.Cfg.TurnMode == balance.TurnSimultaneous {
			if w.Stage == sim.StagePlan {
				m.say("ПЛАНИРУЙ ВЫСТРЕЛ!")
			}
		} else if w.Cur >= 0 && m.sess.Local(w.Cur) && m.hotseatMany() {
			m.say(w.Players[w.Cur].Name + ": ТВОЙ ХОД!")
		} else if w.Cur >= 0 && m.sess.Local(w.Cur) {
			m.say("ТВОЙ ХОД!")
		}
	}
	m.view.SelUnit, m.view.SelStruct = w.SelUnit, w.SelStruct
	b.handleCameraInput(a)

	sim3 := w.Cfg.TurnMode == balance.TurnSimultaneous
	me := w.Cur
	if sim3 {
		me = -1
		for _, p := range w.Players {
			if m.sess.Local(p.ID) && w.PlayerAlive(p.ID) {
				me = p.ID
				break
			}
		}
	}
	mine := me >= 0 && m.sess.Local(me)
	b.autoCamera(a, me)
	if !mine {
		m.view.SelUnit, m.view.SelStruct = w.SelUnit, w.SelStruct
		return
	}
	if sim3 {
		b.updatePlan(a, me)
		return
	}
	if w.Stage != sim.StageActive && w.Stage != sim.StageRetreat {
		b.stopWalk(me)
		// drone control continues in settle stage
		b.droneControl(a, me)
		return
	}
	send := func(c sim.Command) { c.Player = me; m.sess.Send(c) }
	b.buildOpts(me)
	u, s := b.actors()

	if b.squadClick(a, me, send) {
		return
	}
	// selection
	if a.Click && !b.overUI(a) && !b.charging {
		wp := cam.ToWorld(float64(a.MX), float64(a.MY))
		if cu := b.unitAt(me, wp); cu != nil && w.Stage == sim.StageActive {
			send(sim.Command{Type: sim.CmdSelectUnit, ID: cu.ID})
			b.weapon = ""
			return
		}
		if st := w.StructAtPx(wp.X, wp.Y); st != nil && st.Owner == me && w.Cfg.S(st.Def).Kind == balance.SWeapon && w.Stage == sim.StageActive {
			send(sim.Command{Type: sim.CmdSelectStruct, ID: st.ID})
			b.weapon = ""
			return
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		b.cycleUnit(me, dirFor(ebiten.IsKeyPressed(ebiten.KeyShift)), send)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		b.cycleStruct(me, dirFor(ebiten.IsKeyPressed(ebiten.KeyShift)), send)
	}
	for i := range b.opts {
		if inpututil.IsKeyJustPressed(ebiten.Key1+ebiten.Key(i)) && b.opts[i].ammo != 0 {
			b.weapon = b.opts[i].id
			b.hasTarget = false
		}
	}
	// weapon bar clicks
	if a.Click {
		for i := range b.opts {
			if a.In(b.optRect(i)) && b.opts[i].ammo != 0 {
				b.weapon = b.opts[i].id
				b.hasTarget = false
				return
			}
		}
	}
	// walk & jump
	if u != nil && w.Stage != sim.StageSettle {
		dir := 0
		if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
			dir--
		}
		if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
			dir++
		}
		if dir != b.walkDir {
			b.walkDir = dir
			send(sim.Command{Type: sim.CmdWalk, Dir: dir})
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			send(sim.Command{Type: sim.CmdJump})
		}
	} else {
		b.stopWalk(me)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) || b.endBtn.Update(a) {
		send(sim.Command{Type: sim.CmdEndTurn})
		return
	}
	b.droneControl(a, me)
	if b.droneAlive(me) {
		return
	}

	// aiming & firing
	if (u == nil && s == nil) || b.weapon == "" || w.Stage != sim.StageActive {
		return
	}
	wd := w.Cfg.W(b.weapon)
	if wd == nil {
		return
	}
	orig := b.origin(u, s)
	wp := cam.ToWorld(float64(a.MX), float64(a.MY))
	ang := math.Atan2(wp.Y-orig.Y, wp.X-orig.X)
	if math.Abs(ang-b.lastAim) > 0.015 && !(b.overUI(a)) {
		b.lastAim = ang
		send(sim.Command{Type: sim.CmdAim, Angle: ang})
	}
	fire := func(power float64) {
		send(sim.Command{Type: sim.CmdFire, Weapon: b.weapon, Angle: ang, Power: power, X: b.targetX})
		b.charging, b.charge = false, 0
	}
	space := ebiten.IsKeyPressed(ebiten.KeySpace)
	switch {
	case NeedsTarget(wd.Kind):
		if a.Click && !b.overUI(a) {
			b.targetX, b.hasTarget = wp.X, true
		}
		b.fireBtn.Enabled = b.hasTarget
		if (inpututil.IsKeyJustPressed(ebiten.KeySpace) || b.fireBtn.Update(a)) && b.hasTarget {
			fire(1)
			b.hasTarget = false
		}
	case needsPower(wd.Kind):
		startHold := (a.Click && !b.overUI(a) && !b.moved) || inpututil.IsKeyJustPressed(ebiten.KeySpace)
		if startHold && !b.charging {
			b.charging, b.charge = true, 0
		}
		if b.charging {
			b.charge += sim.Dt / 1.15
			if b.charge >= 1 {
				fire(1)
			} else if !a.LeftDown && !space {
				fire(math.Max(0.12, b.charge))
			}
		}
	default:
		if (a.Click && !b.overUI(a) && !b.moved) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			fire(1)
		}
	}
}

// NeedsTarget mirrors sim.NeedsTargetX for the UI.
func NeedsTarget(k balance.WeaponKind) bool { return sim.NeedsTargetX(k) }

func (b *BattleUI) stopWalk(me int) {
	if b.walkDir != 0 {
		b.walkDir = 0
		b.m.sess.Send(sim.Command{Player: me, Type: sim.CmdWalk, Dir: 0})
	}
}

func (b *BattleUI) overUI(a *App) bool {
	if a.MY > ScreenH-120 || a.MY < 50 {
		return true
	}
	if b.overSquad(a, b.m.w.Cur) || (b.m.w.Cfg.TurnMode == balance.TurnSimultaneous && b.overSquad(a, b.m.Me())) {
		return true
	}
	if a.In(image.Rect(ScreenW-290, ScreenH-170, ScreenW, ScreenH)) {
		return true
	}
	for i := range b.opts {
		if a.In(b.optRect(i)) {
			return true
		}
	}
	return false
}

func (b *BattleUI) optRect(i int) image.Rectangle {
	n := len(b.opts)
	wd := 118
	x0 := ScreenW/2 - n*(wd+6)/2
	return image.Rect(x0+i*(wd+6), ScreenH-112, x0+i*(wd+6)+wd, ScreenH-10)
}

func (b *BattleUI) unitAt(pid int, wp sim.Vec) *sim.Unit {
	for _, u := range b.m.w.UnitsOf(pid) {
		if math.Abs(wp.X-u.Pos.X) < 14 && wp.Y > u.Pos.Y-30 && wp.Y < u.Pos.Y+6 {
			return u
		}
	}
	return nil
}

func (b *BattleUI) droneAlive(me int) bool {
	for _, p := range b.m.w.Projs {
		if p.Alive && p.Kind == sim.PDrone && p.Owner == me {
			return true
		}
	}
	return false
}

func (b *BattleUI) droneControl(a *App, me int) {
	for _, p := range b.m.w.Projs {
		if p.Alive && p.Kind == sim.PDrone && p.Owner == me {
			wp := b.m.view.Cam.ToWorld(float64(a.MX), float64(a.MY))
			ang := math.Atan2(wp.Y-p.Pos.Y, wp.X-p.Pos.X)
			b.m.sess.Send(sim.Command{Player: me, Type: sim.CmdSteer, Angle: ang})
			if a.Click || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
				b.m.sess.Send(sim.Command{Player: me, Type: sim.CmdDetonate})
			}
			b.camManual = 0
		}
	}
}

// updatePlan handles the simultaneous-turn planning stage.
func (b *BattleUI) updatePlan(a *App, me int) {
	m := b.m
	w := m.w
	cam := &m.view.Cam
	if w.Stage != sim.StagePlan {
		return
	}

	if b.planUnit < 0 && b.planStruct < 0 || (b.planUnit >= 0 && (!w.Units[b.planUnit].Alive || w.Units[b.planUnit].Owner != me)) {
		b.planStruct = -1
		b.planUnit = -1
		if us := w.UnitsOf(me); len(us) > 0 {
			b.planUnit = us[0].ID
		}
	}
	b.buildOpts(me)
	m.view.SelUnit, m.view.SelStruct = b.planUnit, b.planStruct
	if a.Click {
		for _, r := range b.squadRows(me) {
			if a.In(r.r) {
				if r.isUnit {
					b.planUnit, b.planStruct = r.id, -1
				} else {
					b.planStruct = r.id
				}
				b.weapon = ""
				return
			}
		}
	}
	if a.Click && !b.overUI(a) && !b.charging {
		wp := cam.ToWorld(float64(a.MX), float64(a.MY))
		if cu := b.unitAt(me, wp); cu != nil {
			b.planUnit, b.planStruct = cu.ID, -1
			b.weapon = ""
			return
		}
		if st := w.StructAtPx(wp.X, wp.Y); st != nil && st.Owner == me && w.Cfg.S(st.Def).Kind == balance.SWeapon {
			b.planStruct = st.ID
			b.weapon = ""
			return
		}
	}
	for i := range b.opts {
		if (inpututil.IsKeyJustPressed(ebiten.Key1+ebiten.Key(i)) || (a.Click && a.In(b.optRect(i)))) && b.opts[i].ammo != 0 {
			b.weapon = b.opts[i].id
			b.hasTarget = false
		}
	}
	u, s := b.actors()
	if u == nil && s == nil || b.weapon == "" {
		return
	}
	wd := w.Cfg.W(b.weapon)
	if u != nil {
		if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
			b.planWalk = math.Max(-120, b.planWalk-1.2)
		}
		if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
			b.planWalk = math.Min(120, b.planWalk+1.2)
		}
	}
	orig := b.origin(u, s)
	wp := cam.ToWorld(float64(a.MX), float64(a.MY))
	ang := math.Atan2(wp.Y-orig.Y, wp.X-orig.X)
	commit := func(power float64) {
		c := sim.Command{Player: me, Type: sim.CmdPlan, Weapon: b.weapon, Angle: ang, Power: power, X: b.targetX, Y: b.planWalk, Flag: true, ID: -1}
		if s != nil {
			c.Def, c.ID = "struct", s.ID
		} else if u != nil {
			c.Def, c.ID = "unit", u.ID
		}
		m.sess.Send(c)
		b.planned = true
		b.charging, b.charge = false, 0
		b.flash("План принят — можно переделать до конца времени")
	}
	space := ebiten.IsKeyPressed(ebiten.KeySpace)
	switch {
	case NeedsTarget(wd.Kind):
		if a.Click && !b.overUI(a) {
			b.targetX, b.hasTarget = wp.X, true
			commit(1)
		}
	case needsPower(wd.Kind):
		if ((a.Click && !b.overUI(a) && !b.moved) || inpututil.IsKeyJustPressed(ebiten.KeySpace)) && !b.charging {
			b.charging, b.charge = true, 0
		}
		if b.charging {
			b.charge = math.Min(1, b.charge+sim.Dt/1.15)
			if !a.LeftDown && !space || b.charge >= 1 {
				commit(math.Max(0.12, b.charge))
			}
		}
	default:
		if (a.Click && !b.overUI(a) && !b.moved) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			commit(1)
		}
	}

}

func (b *BattleUI) autoCamera(a *App, me int) {
	m := b.m
	w := m.w
	cam := &m.view.Cam
	var follow *sim.Proj
	for _, p := range w.Projs {
		if !p.Alive || p.Kind == sim.PMine || p.Kind == sim.PPlane || (p.Kind == sim.PBallis && p.Phase == 0) {
			continue
		}
		follow = p
	}
	if follow != nil {
		tp := follow.Pos
		// keep the ground in view when a projectile arcs high
		gx := int(math.Max(0, math.Min(float64(w.Terr.W-1), tp.X)))
		if minY := float64(w.Terr.SurfaceY(gx, 0)) - 230/cam.Zoom; tp.Y < minY {
			tp.Y = minY
		}
		cam.CenterOn(tp, 0.10)
		b.holdT = 1.0
		return
	}
	if b.holdT > 0 {
		b.holdT -= sim.Dt
		return
	}
	if b.camManual > 0 {
		return
	}
	if w.Stage == sim.StageRetreat && w.FiredUnit >= 0 && w.FiredUnit < len(w.Units) && w.Units[w.FiredUnit].Alive {
		fu := w.Units[w.FiredUnit]
		cam.CenterOn(sim.Vec{X: fu.Pos.X, Y: fu.Pos.Y - 40}, 0.06)
		return
	}
	if b.focusT > 0 {
		b.focusT -= sim.Dt
		u, s := b.actors()
		switch {
		case u != nil:
			cam.CenterOn(sim.Vec{X: u.Pos.X, Y: u.Pos.Y - 40}, 0.07)
		case s != nil:
			cam.CenterOn(w.StructCenter(s), 0.07)
		default:
			if w.Cur >= 0 {
				cam.CenterOn(w.StructCenter(w.Structs[w.Players[w.Cur].HQ]), 0.07)
			}
		}
	}
}

// ---- drawing ------------------------------------------------------------------

// Draw renders the battle HUD and aiming overlay.
func (b *BattleUI) Draw(a *App, dst *ebiten.Image) {
	m := b.m
	w := m.w
	cam := &m.view.Cam
	me := w.Cur
	sim3 := w.Cfg.TurnMode == balance.TurnSimultaneous
	if sim3 {
		me = m.Me()
	}
	mine := me >= 0 && m.sess.Local(me)
	if mine {
		b.drawAim(a, dst, me)
	}
	b.drawTop(a, dst)
	b.drawPlayers(a, dst)
	b.drawMini(a, dst)
	if mine && (w.Stage == sim.StageActive || w.Stage == sim.StageRetreat || w.Stage == sim.StagePlan) {
		b.drawWeaponBar(a, dst, me)
		b.drawSquad(a, dst, me)
		if w.Stage == sim.StageRetreat {
			msg := "Выстрел сделан — отступай! A/D — бежать, W — прыжок"
			if !w.ProjectilesBusy() {
				msg += fmt.Sprintf("  (%.0f с)", math.Ceil(math.Max(0, w.RetreatTime)))
			}
			a.TextCenter(dst, msg, float64(ScreenW)/2, 84, 24, colGold, true)
		} else if w.Stage == sim.StageActive && w.Cfg.TurnMode != balance.TurnSimultaneous {
			a.TextCenter(dst, "Выбери юнита слева или Tab · A/D — идти · ЛКМ — огонь", float64(ScreenW)/2, 84, 18, color.RGBA{225, 232, 245, 200}, false)
		}
		if w.Stage != sim.StagePlan {
			b.endBtn.Draw(a, dst)
		}
		if wd := w.Cfg.W(b.weapon); wd != nil && NeedsTarget(wd.Kind) && !sim3 {
			b.fireBtn.Draw(a, dst)
		}
	}
	if b.hintT > 0 {
		a.TextCenter(dst, b.hint, ScreenW/2, ScreenH-210, 22, colGold, true)
	}
	_ = cam
}

// DrawHUDMinimal draws the pieces shown on the game-over screen.
func (b *BattleUI) DrawHUDMinimal(a *App, dst *ebiten.Image) { b.drawPlayers(a, dst) }

func (b *BattleUI) drawTop(a *App, dst *ebiten.Image) {
	m := b.m
	w := m.w
	// wind
	a.Panel(dst, rectXYWH(10, 10, 210, 62))
	a.Text(dst, "Ветер", 20, 14, 17, colDim)
	wv := w.Wind / w.Cfg.WindMax
	cx := 120.0
	a.Rect(dst, cx-60, 52, 120, 4, color.RGBA{60, 70, 90, 255})
	if math.Abs(wv) > 0.02 {
		x1 := cx + wv*58
		a.Line(dst, cx, 40, x1, 40, 5, colGold)
		sgn := math.Copysign(1, wv)
		a.Line(dst, x1, 40, x1-sgn*9, 33, 4, colGold)
		a.Line(dst, x1, 40, x1-sgn*9, 47, 4, colGold)
	}
	a.Text(dst, fmt.Sprintf("%.0f", math.Abs(w.Wind)), 20, 40, 18, colText)

	// turn banner
	var title string
	var col color.Color = colText
	timer := w.TurnTimer
	switch {
	case w.Cfg.TurnMode == balance.TurnSimultaneous:
		if w.Stage == sim.StagePlan {
			title = "ПЛАНИРОВАНИЕ"
		} else {
			title = "ВЫПОЛНЕНИЕ"
			timer = 0
		}
	case w.Cur >= 0:
		p := w.Players[w.Cur]
		col = gfx.TeamColors[p.Color%4]
		title = "Ход: " + p.Name
		if !m.sess.Local(w.Cur) {
			title += " (ждём)"
		}
		if w.Stage == sim.StageRetreat {
			timer = w.RetreatTime
		} else if w.Stage == sim.StageSettle {
			timer = 0
		}
	}
	tw := a.Measure(title, 30, true) + 60
	x := float64(ScreenW)/2 - tw/2
	a.Panel(dst, rectXYWH(int(x), 8, int(tw), 64))
	a.TextB(dst, title, x+30, 10, 30, col)
	sub := fmt.Sprintf("Раунд %d/%d", w.Round, w.Cfg.RoundsFor(len(w.Players)))
	if timer > 0 {
		sub += fmt.Sprintf("   %02d", int(math.Ceil(timer)))
	}
	if w.Stage == sim.StageSettle {
		sub += "   …"
	}
	a.TextCenter(dst, sub, float64(ScreenW)/2, 44, 20, colDim, false)
}

func (b *BattleUI) drawPlayers(a *App, dst *ebiten.Image) {
	m := b.m
	w := m.w
	x, y := float64(ScreenW-290), 10.0
	h := 34 + float64(len(w.Players))*40
	a.Panel(dst, rectXYWH(int(x)-6, int(y), 296, int(h)))
	for i, p := range w.Players {
		yy := y + 8 + float64(i)*40
		col := gfx.TeamColors[p.Color%4]
		if w.Cur == p.ID {
			a.Rect(dst, x-2, yy-3, 288, 38, color.RGBA{col.R, col.G, col.B, 50})
		}
		a.Circle(dst, x+10, yy+14, 7, col)
		nc := color.Color(colText)
		if p.Elim {
			nc = colDim
		}
		name := p.Name
		if w.Cfg.TeamsEnabled {
			name = fmt.Sprintf("[%c] %s", 'A'+p.Team%2, p.Name)
		}
		a.Text(dst, name, x+24, yy, 18, nc)
		a.TextB(dst, fmt.Sprintf("$%d", p.Money), x+150, yy, 18, colGold)
		if p.Elim {
			a.Text(dst, "выбыл", x+222, yy, 16, colBad)
			continue
		}
		hq := 0.0
		if p.HQ >= 0 {
			hq = w.Structs[p.HQ].HP / w.Structs[p.HQ].MaxHP
		}
		a.Rect(dst, x+24, yy+22, 120, 7, color.RGBA{50, 20, 20, 255})
		a.Rect(dst, x+24, yy+22, 120*math.Max(0, hq), 7, color.RGBA{110, 220, 110, 255})
		a.Text(dst, fmt.Sprintf("юнитов %d", len(w.UnitsOf(p.ID))), x+150, yy+18, 15, colDim)
		if w.Leader == p.ID {
			a.TextB(dst, "★", x+262, yy-2, 20, colGold)
		}
	}
}

func (b *BattleUI) drawWeaponBar(a *App, dst *ebiten.Image, me int) {
	m := b.m
	w := m.w
	b.buildOpts(me)
	u, s := b.actors()
	for i, o := range b.opts {
		r := b.optRect(i)
		bg := color.RGBA{30, 38, 60, 235}
		if o.id == b.weapon {
			bg = color.RGBA{86, 100, 40, 245}
		} else if a.In(r) {
			bg = color.RGBA{48, 62, 98, 245}
		}
		if o.ammo == 0 {
			bg = color.RGBA{30, 30, 34, 235}
		}
		a.Rect(dst, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), bg)
		bc := color.RGBA{110, 130, 170, 255}
		if o.id == b.weapon {
			bc = color.RGBA{255, 230, 120, 255}
		}
		a.Border(dst, float64(r.Min.X)+0.5, float64(r.Min.Y)+0.5, float64(r.Dx())-1, float64(r.Dy())-1, 1, bc)
		a.Text(dst, fmt.Sprint(i+1), float64(r.Min.X+5), float64(r.Min.Y+2), 15, colDim)
		col := color.Color(colText)
		if o.ammo == 0 {
			col = colDim
		}
		nm := o.name
		for a.Measure(nm, 15, true) > float64(r.Dx()-8) && len([]rune(nm)) > 4 {
			rs := []rune(nm)
			nm = string(rs[:len(rs)-1])
		}
		a.TextB(dst, nm, float64(r.Min.X+5), float64(r.Min.Y+24), 15, col)
		ammo := "∞"
		if o.ammo >= 0 {
			ammo = fmt.Sprintf("×%d", o.ammo)
		}
		if o.kind == "item" {
			ammo = fmt.Sprintf("запас ×%d", o.ammo)
		}
		a.Text(dst, ammo, float64(r.Min.X+5), float64(r.Min.Y+50), 15, colGold)
		if wd := w.Cfg.W(o.id); wd != nil {
			a.Text(dst, fmt.Sprintf("урон %.0f", wd.Damage*math.Max(1, float64(wd.Count))), float64(r.Min.X+5), float64(r.Min.Y+72), 14, colDim)
		}
	}
	// actor info
	px, py := 14.0, float64(ScreenH-100)
	a.Panel(dst, rectXYWH(8, ScreenH-112, 330, 104))
	switch {
	case u != nil:
		d := w.Cfg.U(u.Def)
		a.TextB(dst, d.Name, px, py-4, 22, colText)
		a.Rect(dst, px, py+26, 200, 10, color.RGBA{50, 20, 20, 255})
		a.Rect(dst, px, py+26, 200*u.HP/u.MaxHP, 10, color.RGBA{110, 220, 110, 255})
		a.Text(dst, fmt.Sprintf("%.0f / %.0f", u.HP, u.MaxHP), px+210, py+20, 17, colText)
	case s != nil:
		d := w.Cfg.S(s.Def)
		a.TextB(dst, d.Name, px, py-4, 22, colText)
		a.Text(dst, fmt.Sprintf("боезапас: %d", s.Ammo), px, py+22, 17, colGold)
	}

	a.Text(dst, "Tab/Shift+Tab — юнит • Q — орудие", px, py+44, 14, colGold)
	a.Text(dst, "A/D — идти • W — прыжок • ЛКМ — огонь", px, py+62, 14, colDim)
}

func (b *BattleUI) drawAim(a *App, dst *ebiten.Image, me int) {
	m := b.m
	w := m.w
	cam := &m.view.Cam
	if w.Stage != sim.StageActive && w.Stage != sim.StagePlan {
		return
	}
	u, s := b.actors()
	if u == nil && s == nil || b.weapon == "" {
		return
	}
	wd := w.Cfg.W(b.weapon)
	if wd == nil {
		return
	}
	orig := b.origin(u, s)
	ox, oy := cam.ToScreen(orig)
	wp := cam.ToWorld(float64(a.MX), float64(a.MY))
	ang := math.Atan2(wp.Y-orig.Y, wp.X-orig.X)
	dir := sim.Dir(ang)
	if NeedsTarget(wd.Kind) {
		if b.hasTarget {
			tx, _ := cam.ToScreen(sim.Vec{X: b.targetX})
			a.Line(dst, tx, 0, tx, ScreenH, 2, color.RGBA{255, 80, 60, 180})
			gy := float64(w.Terr.SurfaceY(int(b.targetX), 0))
			_, ty := cam.ToScreen(sim.Vec{Y: gy})
			a.Border(dst, tx-wd.Radius*cam.Zoom, ty-wd.Radius*cam.Zoom, wd.Radius*2*cam.Zoom, wd.Radius*2*cam.Zoom, 2, color.RGBA{255, 80, 60, 220})
			a.TextCenter(dst, "ЦЕЛЬ", tx, ty-wd.Radius*cam.Zoom-26, 18, color.RGBA{255, 120, 100, 255}, true)
		} else {
			a.TextCenter(dst, "Кликни по карте — выбери цель", float64(ScreenW)/2, 90, 24, colGold, true)
		}
		return
	}
	// crosshair
	cxp, cyp := ox+dir.X*80*cam.Zoom, oy+dir.Y*80*cam.Zoom
	a.Circle(dst, cxp, cyp, 5, color.RGBA{255, 60, 50, 230})
	a.Border(dst, cxp-9, cyp-9, 18, 18, 2, color.RGBA{255, 255, 255, 230})
	a.Line(dst, ox+dir.X*14*cam.Zoom, oy+dir.Y*14*cam.Zoom, cxp-dir.X*10, cyp-dir.Y*10, 1, color.RGBA{255, 255, 255, 60})
	if b.charging || (needsPower(wd.Kind) && b.showTraj && w.Stage == sim.StageActive && false) {
		// power bar
		bw := 70.0
		a.Rect(dst, ox-bw/2-1, oy-58, bw+2, 10, color.RGBA{0, 0, 0, 200})
		pc := color.RGBA{uint8(80 + 175*b.charge), uint8(230 - 150*b.charge), 60, 255}
		a.Rect(dst, ox-bw/2, oy-57, bw*b.charge, 8, pc)
		if b.showTraj && needsPower(wd.Kind) {
			b.drawTrajectory(a, dst, orig, ang, wd, math.Max(0.12, b.charge))
		}
	}
}

func (b *BattleUI) drawTrajectory(a *App, dst *ebiten.Image, orig sim.Vec, ang float64, wd *balance.Weapon, power float64) {
	w := b.m.w
	cam := &b.m.view.Cam
	pos := orig.Add(sim.Dir(ang).Mul(10))
	vel := sim.Dir(ang).Mul(wd.Speed * power)
	g := w.Cfg.GravityPx
	for i := 0; i < 600; i++ {
		vel.Y += g * wd.Gravity * sim.Dt
		vel.X += w.Wind * wd.WindK * sim.Dt
		pos = pos.Add(vel.Mul(sim.Dt))
		if pos.Y > sim.WaterY || w.Terr.Solid(int(pos.X), int(pos.Y)) || w.StructAtPx(pos.X, pos.Y) != nil {
			sx, sy := cam.ToScreen(pos)
			a.Circle(dst, sx, sy, 5, color.RGBA{255, 90, 70, 200})
			return
		}
		if i%5 == 0 {
			sx, sy := cam.ToScreen(pos)
			al := uint8(220 - i/4)
			if i/4 > 200 {
				al = 20
			}
			a.Rect(dst, sx-1.5, sy-1.5, 3, 3, color.RGBA{255, 255, 255, al})
		}
	}
}

func (b *BattleUI) drawMini(a *App, dst *ebiten.Image) {
	m := b.m
	w := m.w
	r := b.miniRect()
	if b.mini == nil || a.Ticks()-b.miniAt > 90 {
		b.miniAt = a.Ticks()
		img := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		t := w.Terr
		for y := 0; y < img.Rect.Dy(); y++ {
			for x := 0; x < img.Rect.Dx(); x++ {
				wx := x * t.W / img.Rect.Dx()
				wy := 380 + y*(t.H-380)/img.Rect.Dy()
				if t.Solid(wx, wy) {
					c := color.RGBA{120, 100, 70, 255}
					if t.At(wx, wy) == sim.Rock {
						c = color.RGBA{110, 112, 120, 255}
					}
					img.SetRGBA(x, y, c)
				} else {
					img.SetRGBA(x, y, color.RGBA{90, 130, 190, 255})
				}
			}
		}
		if b.mini == nil {
			b.mini = ebiten.NewImageFromImage(img)
		} else {
			b.mini.WritePixels(img.Pix)
		}
	}
	dst.DrawImage(b.mini, &ebiten.DrawImageOptions{GeoM: func() (g ebiten.GeoM) { g.Translate(float64(r.Min.X), float64(r.Min.Y)); return }()})
	a.Border(dst, float64(r.Min.X)+0.5, float64(r.Min.Y)+0.5, float64(r.Dx())-1, float64(r.Dy())-1, 1, color.RGBA{200, 210, 230, 255})
	sx := float64(r.Dx()) / float64(w.Terr.W)
	sy := float64(r.Dy()) / float64(w.Terr.H-380)
	for _, s := range w.Structs {
		if s.Alive {
			c := w.StructCenter(s)
			a.Rect(dst, float64(r.Min.X)+c.X*sx-1.5, float64(r.Min.Y)+(c.Y-380)*sy-1.5, 3, 3, gfx.TeamColors[w.Players[s.Owner].Color%4])
		}
	}
	for _, u := range w.Units {
		if u.Alive {
			a.Circle(dst, float64(r.Min.X)+u.Pos.X*sx, float64(r.Min.Y)+(u.Pos.Y-380)*sy, 2, gfx.TeamColors[w.Players[u.Owner].Color%4])
		}
	}
	cam := &m.view.Cam
	a.Border(dst, float64(r.Min.X)+cam.X*sx, float64(r.Min.Y)+(cam.Y-380)*sy, ScreenW/cam.Zoom*sx, ScreenH/cam.Zoom*sy, 1, color.RGBA{255, 255, 255, 220})
}

// cycleUnit selects the next (dir=1) or previous (dir=-1) living unit.
func (b *BattleUI) cycleUnit(me int, dir int, send func(sim.Command)) {
	w := b.m.w
	us := w.UnitsOf(me)
	if len(us) == 0 {
		b.flash("Живых юнитов нет")
		return
	}
	cur := -1
	for i, u := range us {
		if u.ID == w.SelUnit && w.SelStruct < 0 {
			cur = i
		}
	}
	next := us[(cur+dir+len(us)*2)%len(us)].ID
	if cur < 0 && dir < 0 {
		next = us[len(us)-1].ID
	}
	send(sim.Command{Type: sim.CmdSelectUnit, ID: next})
	b.weapon = ""
}

// cycleStruct selects the next ready weapon structure.
func (b *BattleUI) cycleStruct(me int, dir int, send func(sim.Command)) {
	w := b.m.w
	var ids []int
	for _, s := range w.StructsOf(me) {
		if w.Cfg.S(s.Def).Kind == balance.SWeapon && s.Ammo > 0 {
			ids = append(ids, s.ID)
		}
	}
	if len(ids) == 0 {
		b.flash("Нет орудий с боезапасом")
		return
	}
	cur := -1
	for i, id := range ids {
		if id == w.SelStruct {
			cur = i
		}
	}
	next := ids[(cur+dir+len(ids)*2)%len(ids)]
	send(sim.Command{Type: sim.CmdSelectStruct, ID: next})
	b.weapon = ""
}

// ---- squad panel -------------------------------------------------------------

type squadRow struct {
	r      image.Rectangle
	isUnit bool
	id     int
}

// squadRows lays out the clickable list of my units and weapon structures.
func (b *BattleUI) squadRows(me int) []squadRow {
	w := b.m.w
	var rows []squadRow
	for _, u := range w.UnitsOf(me) {
		rows = append(rows, squadRow{isUnit: true, id: u.ID})
	}
	for _, s := range w.StructsOf(me) {
		if w.Cfg.S(s.Def).Kind == balance.SWeapon {
			rows = append(rows, squadRow{id: s.ID})
		}
	}
	h := 38
	if avail := ScreenH - 84 - 140; len(rows)*h > avail && len(rows) > 0 {
		h = avail / len(rows)
	}
	for i := range rows {
		y := 108 + i*h
		rows[i].r = image.Rect(10, y, 214, y+h-3)
	}
	return rows
}

func (b *BattleUI) squadRect(n int) image.Rectangle {
	return image.Rect(6, 80, 218, 108+n*38+4)
}

func (b *BattleUI) overSquad(a *App, me int) bool {
	if me < 0 {
		return false
	}
	rows := b.squadRows(me)
	for _, r := range rows {
		if a.In(r.r) {
			return true
		}
	}
	return a.In(image.Rect(6, 80, 218, 108))
}

func (b *BattleUI) squadClick(a *App, me int, send func(sim.Command)) bool {
	if !a.Click {
		return false
	}
	for _, r := range b.squadRows(me) {
		if a.In(r.r) {
			if r.isUnit {
				send(sim.Command{Type: sim.CmdSelectUnit, ID: r.id})
			} else {
				send(sim.Command{Type: sim.CmdSelectStruct, ID: r.id})
			}
			b.weapon = ""
			return true
		}
	}
	return false
}

func (b *BattleUI) drawSquad(a *App, dst *ebiten.Image, me int) {
	w := b.m.w
	rows := b.squadRows(me)
	if len(rows) == 0 {
		return
	}
	a.Rect(dst, 6, 80, 212, float64(rows[len(rows)-1].r.Max.Y-80+6), color.RGBA{12, 16, 26, 200})
	a.TextB(dst, "Отряд  (Tab — сменить)", 12, 84, 15, colGold)
	sel := -1
	if au, as := b.actors(); as != nil {
		sel = 1000 + as.ID
	} else if au != nil {
		sel = au.ID
	}
	teamCol := gfx.TeamColors[w.Players[me].Color%4]
	for _, row := range rows {
		r := row.r
		key := row.id
		if !row.isUnit {
			key += 1000
		}
		bg := color.RGBA{30, 38, 60, 235}
		if key == sel {
			bg = color.RGBA{84, 98, 36, 245}
		} else if a.In(r) {
			bg = color.RGBA{48, 62, 98, 245}
		}
		a.Rect(dst, float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy()), bg)
		if key == sel {
			a.Border(dst, float64(r.Min.X)+0.5, float64(r.Min.Y)+0.5, float64(r.Dx())-1, float64(r.Dy())-1, 2, color.RGBA{255, 230, 120, 255})
		}
		ih := float64(r.Dy() - 6)
		if row.isUnit {
			u := w.Units[row.id]
			img := b.m.view.pigImage(me, u.Def, 0, false)
			op := &ebiten.DrawImageOptions{}
			sc := math.Min(1.3, ih/float64(gfx.PigH))
			op.GeoM.Scale(sc, sc)
			op.GeoM.Translate(float64(r.Min.X)+4, float64(r.Min.Y)+3)
			dst.DrawImage(img, op)
			a.Text(dst, w.Cfg.U(u.Def).Name, float64(r.Min.X)+38, float64(r.Min.Y)+1, 15, colText)
			frac := u.HP / u.MaxHP
			a.Rect(dst, float64(r.Min.X)+38, float64(r.Max.Y)-9, 150, 5, color.RGBA{50, 20, 20, 255})
			a.Rect(dst, float64(r.Min.X)+38, float64(r.Max.Y)-9, 150*frac, 5, teamCol)
			if w.UnitActed && w.FiredUnit == u.ID {
				a.Text(dst, "✓", float64(r.Max.X)-18, float64(r.Min.Y)+2, 16, colGood)
			}
		} else {
			s := w.Structs[row.id]
			d := w.Cfg.S(s.Def)
			img := b.m.a.Img("s_"+s.Def, func() *image.RGBA { return gfx.StructSprite(s.Def, d.W, d.H) })
			bw, bh := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
			sc := math.Min((ih+4)/bh, 30/bw)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(sc, sc)
			op.GeoM.Translate(float64(r.Min.X)+4, float64(r.Min.Y)+3)
			dst.DrawImage(img, op)
			a.Text(dst, d.Name, float64(r.Min.X)+38, float64(r.Min.Y)+1, 15, colText)
			a.Text(dst, fmt.Sprintf("боезапас: %d", s.Ammo), float64(r.Min.X)+38, float64(r.Max.Y)-18, 13, colGold)
		}
	}
}

func dirFor(shift bool) int {
	if shift {
		return -1
	}
	return 1
}

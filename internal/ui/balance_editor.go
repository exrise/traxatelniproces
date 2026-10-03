package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"svinovoyna/internal/balance"
)

type fld struct {
	sect, label string
	get         func() float64
	set         func(float64)
	step        float64
	min, max    float64
	pct         bool
}

func buildFields(c *balance.Config) []fld {
	var f []fld
	num := func(sect, label string, p *float64, step, min, max float64) {
		f = append(f, fld{sect, label, func() float64 { return *p }, func(v float64) { *p = v }, step, min, max, false})
	}
	pct := func(sect, label string, p *float64, step, min, max float64) {
		f = append(f, fld{sect, label, func() float64 { return *p }, func(v float64) { *p = v }, step, min, max, true})
	}
	inum := func(sect, label string, p *int, step, min, max float64) {
		f = append(f, fld{sect, label, func() float64 { return float64(*p) }, func(v float64) { *p = int(math.Round(v)) }, step, min, max, false})
	}
	const g = "Общие правила"
	inum(g, "Стартовые деньги", &c.StartMoney, 100, 300, 20000)
	inum(g, "Доход за каждую стройку", &c.BaseIncome, 50, 0, 5000)
	num(g, "Время 1-й стройки, с", &c.BuildTimeFirst, 15, 20, 600)
	num(g, "Время стройки, с", &c.BuildTime, 10, 15, 600)
	inum(g, "Раундов в бою", &c.RoundsPerBattle, 1, 1, 30)
	num(g, "Время хода, с", &c.TurnTime, 5, 10, 180)
	num(g, "Время отступления, с", &c.RetreatTime, 1, 0, 20)
	num(g, "Время планирования (одновр.), с", &c.PlanTime, 5, 10, 120)
	num(g, "Прочность штаба", &c.MaxHQHP, 50, 200, 5000)
	inum(g, "Макс. юнитов у игрока", &c.MaxUnits, 1, 1, 20)
	num(g, "Хил юнитов между боями", &c.BuildHeal, 5, 0, 200)
	num(g, "Скорость ходьбы", &c.UnitSpeed, 5, 30, 200)
	num(g, "Гравитация", &c.GravityPx, 20, 200, 1500)
	num(g, "Макс. ветер", &c.WindMax, 10, 0, 400)
	pct(g, "Урон от падения", &c.FallDamage, 0.05, 0, 2)
	const e = "Экономика боя"
	pct(e, "Деньги за урон по юнитам (x)", &c.DmgMoneyUnit, 0.1, 0, 5)
	pct(e, "Деньги за урон по постройкам (x)", &c.DmgMoneyStruct, 0.05, 0, 5)
	inum(e, "Бонус за убийство", &c.KillBonus, 10, 0, 1000)
	pct(e, "Награда за разрушение (% цены)", &c.StructKillPct, 0.01, 0, 1)
	inum(e, "Доход с точки захвата", &c.CaptureIncome, 10, 0, 500)
	pct(e, "Возврат при продаже", &c.SellRefund, 0.05, 0, 1)
	pct(e, "Цена ремонта (от цены)", &c.RepairCostPct, 0.05, 0, 2)
	pct(e, "Множитель за урон лидеру", &c.LeaderMult, 0.05, 0.5, 3)
	pct(e, "Множитель за урон слабейшему", &c.UnderdogMult, 0.05, 0.1, 2)
	pct(e, "Компенсация потерь", &c.CatchUpPct, 0.05, 0, 2)
	inum(e, "Лимит компенсации", &c.CatchUpCap, 50, 0, 5000)
	pct(e, "Урон в режиме «юнит+постройка»", &c.TwoHandDmgMul, 0.05, 0.2, 1.5)
	for i := range c.Units {
		u := &c.Units[i]
		sect := "Юнит: " + u.Name
		inum(sect, "Цена", &u.Cost, 10, 0, 3000)
		num(sect, "Здоровье", &u.HP, 5, 10, 500)
		inum(sect, "Лимит на игрока (0 = без)", &u.Max, 1, 0, 10)
	}
	for i := range c.Weapons {
		w := &c.Weapons[i]
		sect := "Оружие: " + w.Name
		num(sect, "Урон", &w.Damage, 5, 0, 1000)
		if w.Radius > 0 {
			num(sect, "Радиус взрыва", &w.Radius, 2, 4, 300)
		}
		if w.Count > 1 {
			inum(sect, "Снарядов / боеголовок", &w.Count, 1, 1, 40)
		}
		if w.Ammo > 0 {
			inum(sect, "Боезапас на бой", &w.Ammo, 1, 1, 20)
		}
		if w.Cost > 0 {
			inum(sect, "Цена", &w.Cost, 10, 10, 5000)
		}
		if w.Speed > 0 && (w.Kind == balance.KindShell || w.Kind == balance.KindSalvo) {
			num(sect, "Скорость", &w.Speed, 50, 200, 3000)
		}
	}
	for i := range c.Structs {
		s := &c.Structs[i]
		sect := "Постройка: " + s.Name
		if s.Kind != balance.SHQ {
			inum(sect, "Цена", &s.Cost, 10, 0, 5000)
		}
		num(sect, "Прочность", &s.HP, 10, 10, 3000)
		if s.Max > 0 || s.Kind == balance.SWeapon || s.Kind == balance.SAA {
			inum(sect, "Лимит на игрока (0 = без)", &s.Max, 1, 0, 20)
		}
		if s.Tier > 0 {
			inum(sect, "Открывается со стройки №", &s.Tier, 1, 1, 6)
		}
		if s.Kind == balance.SEco {
			inum(sect, "Доход за ход", &s.Income, 5, 0, 500)
		}
		if s.Kind == balance.SAA || s.Kind == balance.SJammer {
			num(sect, "Дальность", &s.AARange, 20, 100, 2000)
			if s.Kind == balance.SAA {
				inum(sect, "Боезапас", &s.AAAmmo, 1, 1, 40)
			}
			for _, cl := range []balance.AAClass{balance.ClassDrone, balance.ClassAir, balance.ClassRocket, balance.ClassBallis, balance.ClassOreshnik} {
				if _, ok := s.AAHit[cl]; ok || s.Kind == balance.SAA {
					cl := cl
					f = append(f, fld{sect, "Перехват: " + aaClassName(cl), func() float64 { return s.AAHit[cl] }, func(v float64) {
						if s.AAHit == nil {
							s.AAHit = map[balance.AAClass]float64{}
						}
						s.AAHit[cl] = v
					}, 0.05, 0, 0.98, true})
				}
			}
		}
	}
	return f
}

func aaClassName(c balance.AAClass) string {
	switch c {
	case balance.ClassDrone:
		return "дроны"
	case balance.ClassAir:
		return "авиация"
	case balance.ClassRocket:
		return "ракеты/Град"
	case balance.ClassBallis:
		return "баллистика"
	case balance.ClassOreshnik:
		return "Орешник"
	}
	return string(c)
}

// BalanceEditor lets the player tweak every number of a config.
type BalanceEditor struct {
	cfg    *balance.Config
	back   Scene
	fields []fld
	scroll float64
	done   *Button
	save   *Button
	reset  *Button
	name   *TextInput
	base   *balance.Config
	msg    string
	msgT   int
	hold   int
	rows   []edRow
	onDone func(*balance.Config)
}

type edRow struct {
	header string
	f      *fld
	minus  *Button
	plus   *Button
}

// NewBalanceEditor creates the editor for cfg; onDone receives the edited config.
func NewBalanceEditor(a *App, cfg *balance.Config, back Scene, onDone func(*balance.Config)) *BalanceEditor {
	e := &BalanceEditor{cfg: cfg, back: back, onDone: onDone, base: cfg.Clone()}
	e.fields = buildFields(cfg)
	e.done = NewButton(ScreenW-260, ScreenH-62, 220, 46, "Готово")
	e.save = NewButton(ScreenW-520, ScreenH-62, 240, 46, "Сохранить пресет")
	e.reset = NewButton(ScreenW-760, ScreenH-62, 220, 46, "Сбросить")
	e.name = &TextInput{R: rectXYWH(60, ScreenH-62, 300, 46), Text: cfg.Name, Max: 24, Hint: "Название пресета"}
	last := ""
	for i := range e.fields {
		f := &e.fields[i]
		if f.sect != last {
			e.rows = append(e.rows, edRow{header: f.sect})
			last = f.sect
		}
		e.rows = append(e.rows, edRow{f: f, minus: NewButton(0, 0, 34, 28, "-"), plus: NewButton(0, 0, 34, 28, "+")})
	}
	return e
}

func (e *BalanceEditor) rowY(i int) float64 {
	y := 110.0 - e.scroll
	for k := 0; k < i; k++ {
		if e.rows[k].header != "" {
			y += 40
		} else {
			y += 34
		}
	}
	return y
}

func (e *BalanceEditor) Update(a *App) error {
	e.name.Update(a)
	if a.Wheel != 0 {
		e.scroll -= a.Wheel * 60
	}
	total := e.rowY(len(e.rows)) + e.scroll - 110
	e.scroll = math.Max(0, math.Min(math.Max(0, total-(ScreenH-190)), e.scroll))
	if a.LeftDown {
		e.hold++
	} else {
		e.hold = 0
	}
	repeat := a.Click || (e.hold > 24 && e.hold%4 == 0)
	for i := range e.rows {
		r := &e.rows[i]
		if r.f == nil {
			continue
		}
		y := e.rowY(i)
		if y < 80 || y > ScreenH-80 {
			continue
		}
		r.minus.R = rectXYWH(ScreenW-170, int(y), 34, 28)
		r.plus.R = rectXYWH(ScreenW-100, int(y), 34, 28)
		r.minus.Hover = a.In(r.minus.R)
		r.plus.Hover = a.In(r.plus.R)
		if repeat {
			step := r.f.step
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				step *= 5
			}
			v := r.f.get()
			if r.minus.Hover {
				r.f.set(math.Max(r.f.min, math.Round((v-step)*1000)/1000))
			}
			if r.plus.Hover {
				r.f.set(math.Min(r.f.max, math.Round((v+step)*1000)/1000))
			}
		}
	}
	if e.done.Update(a) {
		e.cfg.Name = e.name.Text
		e.cfg.Index()
		if e.onDone != nil {
			e.onDone(e.cfg)
		}
		a.SetScene(e.back)
	}
	if e.reset.Update(a) {
		*e.cfg = *e.base.Clone()
		e.cfg.Index()
		*e = *NewBalanceEditor(a, e.cfg, e.back, e.onDone)
	}
	if e.save.Update(a) {
		e.cfg.Name = e.name.Text
		if err := balance.SaveCustom(a.Dir+"/presets", e.cfg); err != nil {
			e.msg = "Не удалось сохранить: " + err.Error()
		} else {
			e.msg = "Сохранено в папку presets/"
			found := false
			for i, p := range a.Presets {
				if p.Name == e.cfg.Name {
					a.Presets[i] = e.cfg.Clone()
					found = true
				}
			}
			if !found {
				a.Presets = append(a.Presets, e.cfg.Clone())
			}
		}
		e.msgT = 240
	}
	if e.msgT > 0 {
		e.msgT--
	}
	if a.EscapeKey {
		a.SetScene(e.back)
	}
	return nil
}

func (e *BalanceEditor) Draw(a *App, dst *ebiten.Image) {
	dst.Fill(color.RGBA{20, 26, 40, 255})
	a.TextB(dst, "Настройка баланса", 40, 20, 36, colGold)
	a.Text(dst, "Колёсико — прокрутка, Shift — шаг ×5. Изменения действуют на текущую игру.", 40, 66, 18, colDim)
	for i := range e.rows {
		r := &e.rows[i]
		y := e.rowY(i)
		if y < 90 || y > ScreenH-84 {
			continue
		}
		if r.header != "" {
			a.Rect(dst, 40, y+4, ScreenW-80, 30, color.RGBA{44, 60, 96, 255})
			a.TextB(dst, r.header, 52, y+6, 21, colGold)
			continue
		}
		if i%2 == 0 {
			a.Rect(dst, 40, y-3, ScreenW-80, 34, color.RGBA{28, 34, 50, 255})
		}
		a.Text(dst, r.f.label, 70, y+2, 20, colText)
		v := r.f.get()
		s := fmt.Sprintf("%.0f", v)
		if r.f.pct {
			s = fmt.Sprintf("%.0f%%", v*100)
		} else if v != math.Trunc(v) {
			s = fmt.Sprintf("%.2f", v)
		}
		a.TextB(dst, s, ScreenW-260, y+1, 21, colGold)
		r.minus.Draw(a, dst)
		r.plus.Draw(a, dst)
	}
	a.Rect(dst, 0, ScreenH-80, ScreenW, 80, color.RGBA{14, 18, 28, 255})
	e.name.Draw(a, dst)
	e.reset.Draw(a, dst)
	e.save.Draw(a, dst)
	e.done.Draw(a, dst)
	if e.msgT > 0 {
		a.Text(dst, e.msg, 380, ScreenH-52, 20, colGood)
	}
}

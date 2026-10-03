// Package ai contains the computer players: shopping during the build phase
// and aiming with trajectory simulation during the battle phase.
package ai

import (
	"math"
	"sort"
	"time"

	"svinovoyna/internal/balance"
	"svinovoyna/internal/sim"
)

// Bot plays one seat.
type Bot struct {
	Pid   int
	Style Style
	Skill float64 // 0..1, higher = less aim noise
	rng   sim.RNG

	turnKey  int
	delay    int
	walk     int
	walkDir  int
	walks    int
	advances int
	planned  bool
	planning bool
	jobs     []func() *action
	jobsBest *action
}

// New creates a bot for a seat.
func New(pid int, style Style, skill float64, seed uint64) *Bot {
	return &Bot{Pid: pid, Style: style, Skill: skill, rng: sim.NewRNG(seed + uint64(pid)*977), turnKey: -1}
}

type action struct {
	unit, structID int
	weapon         string
	angle, power   float64
	tx             float64
	score          float64
	kind           string // "unit", "struct", "item"
}

type target struct {
	pos   Vec
	unit  *sim.Unit
	s     *sim.Struct
	value float64
}

// Vec is a local alias.
type Vec = sim.Vec

func (b *Bot) targets(w *sim.World, from Vec) []target {
	var ts []target
	for _, u := range w.Units {
		if u.Alive && w.Hostile(b.Pid, u.Owner) {
			ts = append(ts, target{pos: Vec{X: u.Pos.X, Y: u.Pos.Y - sim.UnitH/2}, unit: u, value: float64(w.Cfg.U(u.Def).Cost)})
		}
	}
	for _, s := range w.Structs {
		if s.Alive && w.Hostile(b.Pid, s.Owner) {
			d := w.Cfg.S(s.Def)
			v := float64(d.Cost) * 0.6
			if d.Kind == balance.SHQ {
				v = 500
			}
			ts = append(ts, target{pos: w.StructCenter(s), s: s, value: v})
		}
	}
	sort.Slice(ts, func(i, j int) bool {
		return ts[i].pos.Dist(from)/(1+ts[i].value/400) < ts[j].pos.Dist(from)/(1+ts[j].value/400)
	})
	return ts
}

func (b *Bot) allowed(w *sim.World) (unit, str bool) {
	mode := w.Cfg.TurnMode
	switch mode {
	case balance.TurnClassic:
		return !w.UnitActed && !w.StructActed, !w.UnitActed && !w.StructActed
	case balance.TurnUnitAndStruct:
		return !w.UnitActed, !w.StructActed
	}
	return true, true
}

// Think returns commands to execute this tick (battle phase).
func (b *Bot) Think(w *sim.World) []sim.Command {
	if w.Phase != sim.PhaseBattle || !w.PlayerAlive(b.Pid) {
		return nil
	}
	var out []sim.Command
	// steer own drone toward the best target
	for _, p := range w.Projs {
		if p.Alive && p.Kind == sim.PDrone && p.Owner == b.Pid {
			ts := b.targets(w, p.Pos)
			if len(ts) > 0 {
				tp := ts[0].pos
				if math.Abs(tp.X-p.Pos.X) > 260 {
					tp.Y -= 600
				}
				d := tp.Sub(p.Pos)
				out = append(out, sim.Command{Player: b.Pid, Type: sim.CmdSteer, Angle: math.Atan2(d.Y, d.X)})
			}
		}
	}
	if w.Cfg.TurnMode == balance.TurnSimultaneous {
		if w.Stage == sim.StagePlan && w.Plans[b.Pid] == nil {
			pa, done := b.stepPlan(w, true, true, planBudget)
			if !done {
				return out
			}
			if a := pa; a != nil && a.score > 12 {
				c := sim.Command{Player: b.Pid, Type: sim.CmdPlan, Weapon: a.weapon, Angle: a.angle, Power: a.power, X: a.tx, Flag: true, ID: -1}
				if a.kind == "unit" {
					c.Def, c.ID = "unit", a.unit
				} else if a.kind == "struct" {
					c.Def, c.ID = "struct", a.structID
				}
				out = append(out, c)
			} else {
				out = append(out, sim.Command{Player: b.Pid, Type: sim.CmdPlan, Flag: true, ID: -1})
			}
		}
		return out
	}
	if w.Cur != b.Pid || w.Stage != sim.StageActive {
		b.turnKey = -1
		return out
	}
	key := w.BattleNo*100000 + w.Round*100 + w.OrderPos
	if key != b.turnKey {
		b.turnKey = key
		b.delay = 40
		b.walk = 0
		b.walks = 0
		b.advances = 0
		b.planning = false
		b.jobs = nil
	}
	if b.delay > 0 {
		b.delay--
		return out
	}
	if b.walk > 0 {
		b.walk--
		if b.walk == 0 {
			out = append(out, sim.Command{Player: b.Pid, Type: sim.CmdWalk, Dir: 0})
			b.delay = 20
		} else if b.walk%20 == 1 {
			out = append(out, sim.Command{Player: b.Pid, Type: sim.CmdWalk, Dir: b.walkDir})
		}
		return out
	}
	au, as := b.allowed(w)
	a, done := b.stepPlan(w, au, as, planBudget)
	if !done {
		return out
	}
	if a == nil || a.score < 12 {
		if b.walks < 14 && w.TurnTimer > 9 && au && w.SelUnit >= 0 {
			if dir := b.approachDir(w); dir != 0 {
				b.walks++
				b.walk = 50 + b.rng.Intn(40)
				b.walkDir = dir
				return append(out, sim.Command{Player: b.Pid, Type: sim.CmdWalk, Dir: dir})
			}
		}
		return append(out, sim.Command{Player: b.Pid, Type: sim.CmdEndTurn})
	}
	if b.advances < 5 && w.TurnTimer > 18 && au {
		if u, dir := b.advanceCandidate(w); u != nil {
			b.advances++
			b.walk = 70 + b.rng.Intn(60)
			b.walkDir = dir
			if w.SelUnit != u.ID {
				out = append(out, sim.Command{Player: b.Pid, Type: sim.CmdSelectUnit, ID: u.ID})
			}
			return append(out, sim.Command{Player: b.Pid, Type: sim.CmdWalk, Dir: dir})
		}
	}
	switch a.kind {
	case "unit":
		if w.SelUnit != a.unit {
			out = append(out, sim.Command{Player: b.Pid, Type: sim.CmdSelectUnit, ID: a.unit})
		}
	case "struct":
		out = append(out, sim.Command{Player: b.Pid, Type: sim.CmdSelectStruct, ID: a.structID})
	}
	out = append(out, sim.Command{Player: b.Pid, Type: sim.CmdAim, Angle: a.angle},
		sim.Command{Player: b.Pid, Type: sim.CmdFire, Weapon: a.weapon, Angle: a.angle, Power: a.power, X: a.tx})
	b.delay = 15
	return out
}

func (b *Bot) approachDir(w *sim.World) int {
	if w.SelUnit < 0 {
		return 0
	}
	u := w.Units[w.SelUnit]
	best, bestD := Vec{}, 1e9
	for _, t := range b.targets(w, u.Pos) {
		if d := t.pos.Dist(u.Pos); d < bestD {
			best, bestD = t.pos, d
		}
	}
	for _, c := range w.Points {
		if c.Owner != b.Pid {
			if d := c.Pos.Dist(u.Pos) * 0.8; d < bestD {
				best, bestD = c.Pos, d
			}
		}
	}
	if bestD > 1e8 {
		return 0
	}
	if best.X > u.Pos.X {
		return 1
	}
	return -1
}

// plan evaluates every available action and returns the best.
func (b *Bot) plan(w *sim.World, allowUnit, allowStruct bool) *action {
	var best *action
	for _, job := range b.planJobs(w, allowUnit, allowStruct) {
		if a := job(); a != nil && (best == nil || a.score > best.score) {
			best = a
		}
	}
	return best
}

// planJobs splits planning into small independent evaluations so that the
// game can spread them over several frames (see stepPlan).
func (b *Bot) planJobs(w *sim.World, allowUnit, allowStruct bool) []func() *action {
	var jobs []func() *action
	if allowUnit {
		for _, u := range w.UnitsOf(b.Pid) {
			u := u
			for _, wid := range w.Cfg.U(u.Def).Weapons {
				wid := wid
				if w.UnitAmmoLeft(u, wid) == 0 {
					continue
				}
				wd := w.Cfg.W(wid)
				if wd == nil || wd.Kind == balance.KindMine || wd.Kind == balance.KindRepair {
					continue
				}
				jobs = append(jobs, func() *action {
					if !u.Alive {
						return nil
					}
					muzzle := Vec{X: u.Pos.X, Y: u.Pos.Y - sim.UnitH*0.6}
					a := b.evalWeapon(w, wd, muzzle, u.ID, -1, u.Pos.X)
					if a != nil {
						a.kind, a.unit, a.weapon = "unit", u.ID, wid
					}
					return a
				})
			}
		}
	}
	if allowStruct && w.HasHQ(b.Pid) {
		for _, s := range w.StructsOf(b.Pid) {
			s := s
			d := w.Cfg.S(s.Def)
			if !d.Armed() || s.Ammo <= 0 {
				continue
			}
			wd := w.Cfg.W(d.Weapon)
			jobs = append(jobs, func() *action {
				if !s.Alive {
					return nil
				}
				a := b.evalWeapon(w, wd, w.StructMuzzle(s), -1, s.ID, w.StructCenter(s).X)
				if a != nil {
					a.kind, a.structID, a.weapon = "struct", s.ID, d.Weapon
					a.angle = sim.ClampStructAim(a.angle)
				}
				return a
			})
		}
		hq := w.Structs[w.Players[b.Pid].HQ]
		for _, id := range []string{"fab", "kab", "geran"} {
			id := id
			if w.Players[b.Pid].Items[id] <= 0 {
				continue
			}
			wd := w.Cfg.W(id)
			jobs = append(jobs, func() *action {
				a := b.evalWeapon(w, wd, w.StructMuzzle(hq), -1, -1, w.StructCenter(hq).X)
				if a != nil {
					a.kind, a.weapon = "item", id
				}
				return a
			})
		}
	}
	return jobs
}

// stepPlan advances (or starts) incremental planning within a time budget.
// It returns done=true with the best action (possibly nil) when finished.
func (b *Bot) stepPlan(w *sim.World, allowUnit, allowStruct bool, budget time.Duration) (*action, bool) {
	if !b.planning {
		b.planning = true
		b.jobs = b.planJobs(w, allowUnit, allowStruct)
		b.jobsBest = nil
	}
	start := time.Now()
	for len(b.jobs) > 0 && time.Since(start) < budget {
		job := b.jobs[0]
		b.jobs = b.jobs[1:]
		if a := job(); a != nil && (b.jobsBest == nil || a.score > b.jobsBest.score) {
			b.jobsBest = a
		}
	}
	if len(b.jobs) > 0 {
		return nil, false
	}
	b.planning = false
	return b.jobsBest, true
}

// aaDiscount estimates how likely enemy air defence is to stop an air weapon aimed at x.
func (b *Bot) aaDiscount(w *sim.World, wd *balance.Weapon, tx float64) float64 {
	if wd.Class == balance.ClassNone {
		return 1
	}
	keep := 1.0
	for _, s := range w.Structs {
		if !s.Alive || !w.Hostile(b.Pid, s.Owner) {
			continue
		}
		d := w.Cfg.S(s.Def)
		if d.Kind != balance.SAA && d.Kind != balance.SJammer {
			continue
		}
		if d.Kind == balance.SAA && s.AAAmmo <= 0 {
			continue
		}
		c := w.StructCenter(s)
		if math.Abs(c.X-tx) < d.AARange*1.1 {
			keep *= 1 - d.AAHit[wd.Class]
		}
	}
	return keep
}

func (b *Bot) noise(a, power float64) (float64, float64) {
	n := (1 - b.Skill) * 0.035
	return a + b.rng.Norm()*n, clampF(power+b.rng.Norm()*n*1.5, 0.1, 1)
}

func clampF(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// evalWeapon finds the best way to use one weapon from one muzzle.
func (b *Bot) evalWeapon(w *sim.World, wd *balance.Weapon, muzzle Vec, unit, structID int, selfX float64) *action {
	ts := b.targets(w, muzzle)
	if len(ts) == 0 {
		return nil
	}
	if len(ts) > 7 {
		ts = ts[:7]
	}
	k := 1.0
	if w.Cfg.TurnMode == balance.TurnUnitAndStruct {
		k = w.Cfg.TwoHandDmgMul
	}
	switch wd.Kind {
	case balance.KindBurst, balance.KindPellets, balance.KindShot:
		var best *action
		for _, t := range ts {
			d := t.pos.Sub(muzzle)
			dist := d.Len()
			if dist > wd.Range || dist < 10 {
				continue
			}
			ang := math.Atan2(d.Y, d.X)
			if structID >= 0 {
				ang = sim.ClampStructAim(ang)
			}
			hitPos, hu, hs := w.TraceRay(b.Pid, muzzle.Add(sim.Dir(ang).Mul(8)), ang, wd.Range, unit, structID)
			_ = hitPos
			var val float64
			switch {
			case hu != nil && w.Hostile(b.Pid, hu.Owner):
				val = math.Min(hu.HP, wd.Damage*float64(max(1, wd.Count))*k) / hu.MaxHP * float64(w.Cfg.U(hu.Def).Cost)
			case hs != nil && w.Hostile(b.Pid, hs.Owner):
				def := w.Cfg.S(hs.Def)
				val = wd.Damage * float64(max(1, wd.Count)) * k * (1 - def.Armor*(1-wd.Pierce)) / hs.MaxHP * float64(def.Cost) * 0.35
			default:
				continue
			}
			spread := math.Max(wd.Spread, 0.002)
			hit := clampF((sim.UnitH*0.6)/(dist*spread+1), 0.12, 1)
			if wd.Kind == balance.KindPellets {
				val *= 1 - wd.Falloff*dist/wd.Range
			}
			score := val * hit
			if best == nil || score > best.score {
				a, p := b.noise(ang, 1)
				best = &action{angle: a, power: p, score: score}
			}
		}
		return best
	case balance.KindShell, balance.KindSalvo, balance.KindMissile:
		return b.evalBallistic(w, wd, muzzle, unit, structID, ts, k)
	case balance.KindBallis, balance.KindMIRV, balance.KindAirstrike, balance.KindGeran:
		return b.evalArea(w, wd, ts, k)
	case balance.KindDrone, balance.KindGuided:
		t := ts[0]
		d := t.pos.Sub(muzzle)
		if d.Len() > wd.Speed*wd.Flight*0.8 {
			return nil
		}
		val := t.value * 0.5 * b.aaDiscount(w, wd, t.pos.X)
		ang := -1.0
		if d.X < 0 {
			ang = -math.Pi + 1.0
		}
		return &action{angle: ang, power: 1, score: val * k}
	}
	return nil
}

func (b *Bot) evalBallistic(w *sim.World, wd *balance.Weapon, muzzle Vec, unit, structID int, ts []target, k float64) *action {
	var best *action
	mult := 1.0
	if wd.Kind == balance.KindSalvo {
		mult = float64(wd.Count) * 0.45
	}
	if wd.Class != balance.ClassNone {
		mult *= 0.8
	}
	dirs := []float64{1, -1}
	// only shoot towards the side where the best targets are
	nearest := ts[0].pos.X - muzzle.X
	if nearest < 0 {
		dirs = []float64{-1}
	} else {
		dirs = []float64{1}
	}
	if wd.Gravity <= 0.1 {
		// flat trajectory weapons: aim straight, scan small corrections
		for _, t := range ts {
			base := math.Atan2(t.pos.Y-muzzle.Y, t.pos.X-muzzle.X)
			for da := -0.12; da <= 0.12; da += 0.01 {
				ang := base + da
				if structID >= 0 {
					ang = sim.ClampStructAim(ang)
				}
				v := sim.Dir(ang).Mul(wd.Speed)
				imp, hit := w.PredictShell(muzzle.Add(sim.Dir(ang).Mul(10)), v, wd, b.Pid)
				if !hit {
					continue
				}
				sc := w.EstimateExplosion(imp, wd.Radius, wd.Damage*k, b.Pid, wd.BlockMul, wd.Pierce) * mult
				if best == nil || sc > best.score {
					a, p := b.noise(ang, 1)
					best = &action{angle: a, power: p, score: sc}
				}
			}
		}
		return best
	}
	for _, dir := range dirs {
		for ai := 0; ai < 26; ai++ {
			elev := 0.1 + float64(ai)*0.055 // radians above horizontal
			var ang float64
			if dir > 0 {
				ang = -elev
			} else {
				ang = -math.Pi + elev
			}
			for pi := 0; pi < 8; pi++ {
				pw := 0.3 + float64(pi)*0.1
				v := sim.Dir(ang).Mul(wd.Speed * pw)
				imp, hit := w.PredictShell(muzzle.Add(sim.Dir(ang).Mul(10)), v, wd, b.Pid)
				if !hit {
					continue
				}
				sc := w.EstimateExplosion(imp, wd.Radius, wd.Damage*k, b.Pid, wd.BlockMul, wd.Pierce) * mult
				if best == nil || sc > best.score {
					best = &action{angle: ang, power: pw, score: sc}
				}
			}
		}
	}
	if best == nil || best.score <= 0 {
		return best
	}
	// refine around the best combination
	ba, bp := best.angle, best.power
	for da := -0.04; da <= 0.04; da += 0.01 {
		for dp := -0.05; dp <= 0.05; dp += 0.025 {
			ang, pw := ba+da, clampF(bp+dp, 0.1, 1)
			v := sim.Dir(ang).Mul(wd.Speed * pw)
			imp, hit := w.PredictShell(muzzle.Add(sim.Dir(ang).Mul(10)), v, wd, b.Pid)
			if !hit {
				continue
			}
			sc := w.EstimateExplosion(imp, wd.Radius, wd.Damage*k, b.Pid, wd.BlockMul, wd.Pierce) * mult
			if sc > best.score {
				best = &action{angle: ang, power: pw, score: sc}
			}
		}
	}
	best.angle, best.power = b.noise(best.angle, best.power)
	return best
}

// evalArea scores weapons aimed at a ground x coordinate.
func (b *Bot) evalArea(w *sim.World, wd *balance.Weapon, ts []target, k float64) *action {
	var best *action
	n := max(1, wd.Count)
	spotterK := 1.0
	if wd.Kind == balance.KindAirstrike && !hasSpotter(w, b.Pid) {
		spotterK = 3
	}
	for _, t := range ts {
		tx := t.pos.X
		sum := 0.0
		for i := 0; i < n; i++ {
			off := 0.0
			if n > 1 {
				off = (float64(i)/float64(n-1) - 0.5) * wd.Spread
				if wd.Kind == balance.KindAirstrike {
					off = (float64(i) - float64(n-1)/2) * wd.Spread
				}
			}
			x := tx + off
			gy := float64(w.Terr.SurfaceY(int(clampF(x, 0, float64(w.Terr.W-1))), 0))
			sum += w.EstimateExplosion(Vec{X: x, Y: gy - 4}, wd.Radius, wd.Damage*k, b.Pid, wd.BlockMul, wd.Pierce)
		}
		sum *= b.aaDiscount(w, wd, tx)
		if spotterK > 1 {
			sum *= 0.6
		}
		if best == nil || sum > best.score {
			best = &action{tx: tx + b.rng.Norm()*(1-b.Skill)*20, power: 1, score: sum}
		}
	}
	return best
}

func hasSpotter(w *sim.World, pid int) bool {
	for _, u := range w.Units {
		if u.Alive && u.Owner == pid && u.Def == "spotter" {
			return true
		}
	}
	return false
}

// advanceCandidate finds a short-range infantry unit that should move closer to the enemy.
func (b *Bot) advanceCandidate(w *sim.World) (*sim.Unit, int) {
	for _, u := range w.UnitsOf(b.Pid) {
		switch u.Def {
		case "assault", "shotgun", "sniper":
		default:
			continue
		}
		reach := 0.0
		for _, id := range w.Cfg.U(u.Def).Weapons {
			if wd := w.Cfg.W(id); wd != nil && (wd.Kind == balance.KindBurst || wd.Kind == balance.KindPellets || wd.Kind == balance.KindShot) {
				reach = math.Max(reach, wd.Range)
			}
		}
		best, bestD := Vec{}, 1e9
		for _, t := range b.targets(w, u.Pos) {
			if d := t.pos.Dist(u.Pos); d < bestD {
				best, bestD = t.pos, d
			}
		}
		for _, c := range w.Points {
			if c.Owner != b.Pid {
				if d := c.Pos.Dist(u.Pos) * 0.8; d < bestD {
					best, bestD = c.Pos, d
				}
			}
		}
		if bestD > 1e8 || bestD < reach*0.6 {
			continue
		}
		if best.X > u.Pos.X {
			return u, 1
		}
		return u, -1
	}
	return nil, 0
}

// planBudget is how much CPU one game tick may spend on bot planning, so the
// UI never freezes while a bot "thinks".
var planBudget = 3 * time.Millisecond

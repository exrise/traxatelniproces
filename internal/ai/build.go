package ai

import (
	"math"
	"sort"

	"svinovoyna/internal/balance"
	"svinovoyna/internal/sim"
)

// Style is a shopping strategy.
type Style int

const (
	Balanced Style = iota
	Infantry
	Turret
	Rocket
	Turtle
	Air
	NumStyles
)

func (s Style) String() string {
	return [...]string{"Сбалансированный", "Пехота", "Турели", "Ракеты", "Черепаха", "Авиация"}[s]
}

type buy struct {
	kind  byte // u unit, s struct, i item, w wall
	def   string
	count int
}

var lists = map[Style][]buy{
	Balanced: {{'u', "assault", 1}, {'s', "zu23", 1}, {'s', "d30", 1}, {'w', "", 4}, {'u', "mortar", 1}, {'u', "sniper", 1}, {'s', "dshk", 1},
		{'u', "rpg", 1}, {'s', "farm", 1}, {'w', "", 8}, {'s', "kornet", 1}, {'s', "pantsir", 1}, {'i', "fab", 1}, {'u', "spotter", 1},
		{'s', "grad", 1}, {'s', "iskander", 1}, {'u', "shotgun", 1}, {'s', "oil", 1}, {'i', "kab", 1}, {'s', "s400", 1}, {'s', "oreshnik", 1}, {'w', "", 14}},
	Infantry: {{'u', "assault", 2}, {'u', "sniper", 1}, {'u', "rpg", 1}, {'u', "mortar", 1}, {'u', "shotgun", 1}, {'w', "", 4}, {'s', "zu23", 1},
		{'u', "engineer", 1}, {'u', "dronner", 1}, {'u', "assault", 3}, {'s', "dshk", 1}, {'u', "sniper", 2}, {'w', "", 8}, {'s', "pantsir", 1},
		{'u', "rpg", 2}, {'u', "mortar", 2}, {'s', "d30", 1}, {'w', "", 12}},
	Turret: {{'s', "dshk", 1}, {'s', "d30", 1}, {'w', "", 4}, {'s', "kornet", 1}, {'u', "mortar", 1}, {'s', "zu23", 1}, {'s', "d30", 2}, {'s', "bunker", 1},
		{'s', "grad", 1}, {'w', "", 8}, {'s', "dshk", 2}, {'s', "kornet", 2}, {'s', "pantsir", 1}, {'s', "d30", 3}, {'s', "grad", 2}, {'w', "", 14}},
	Rocket: {{'s', "grad", 1}, {'u', "spotter", 1}, {'s', "d30", 1}, {'s', "zu23", 1}, {'i', "fab", 1}, {'w', "", 3}, {'s', "iskander", 1}, {'s', "pantsir", 1},
		{'s', "grad", 2}, {'s', "oreshnik", 1}, {'i', "kab", 1}, {'u', "assault", 1}, {'w', "", 8}, {'s', "s400", 1}, {'i', "fab", 2}},
	Turtle: {{'w', "", 6}, {'s', "bunker", 1}, {'s', "zu23", 1}, {'s', "d30", 1}, {'u', "mortar", 1}, {'s', "farm", 1}, {'w', "", 10}, {'s', "pantsir", 1},
		{'s', "kornet", 1}, {'s', "oil", 1}, {'s', "reb", 1}, {'s', "s400", 1}, {'u', "engineer", 1}, {'s', "d30", 2}, {'w', "", 16}},
	Air: {{'u', "spotter", 1}, {'i', "fab", 2}, {'s', "zu23", 1}, {'u', "assault", 1}, {'i', "kab", 1}, {'w', "", 3}, {'i', "geran", 1}, {'s', "pantsir", 1},
		{'s', "d30", 1}, {'i', "fab", 3}, {'i', "kab", 2}, {'w', "", 8}, {'s', "s400", 1}, {'i', "geran", 2}, {'u', "sniper", 1}},
}

// Build performs the whole build phase for the bot and marks it ready.
func (b *Bot) Build(w *sim.World) {
	if w.Phase != sim.PhaseBuild || w.Players[b.Pid].Ready {
		return
	}
	p := w.Players[b.Pid]
	// repair damaged things first if cheap
	for _, s := range w.StructsOf(b.Pid) {
		c := w.RepairCost(s)
		if c > 0 && c < p.Money/6 {
			_ = w.Repair(b.Pid, s.ID)
		}
	}
	list := lists[b.Style]
	if list == nil {
		list = lists[Balanced]
	}
	wallsWanted := 0
	for pass := 0; pass < 3; pass++ {
		for _, it := range list {
			switch it.kind {
			case 'u':
				have := 0
				for _, u := range w.UnitsOf(b.Pid) {
					if u.Def == it.def {
						have++
					}
				}
				for have < it.count && b.buyUnit(w, it.def) {
					have++
				}
			case 's':
				for w.CountStructs(b.Pid, it.def) < it.count && b.buyStruct(w, it.def) {
				}
			case 'i':
				for w.Players[b.Pid].Items[it.def] < it.count && w.BuyItem(b.Pid, it.def) == nil {
				}
			case 'w':
				wallsWanted = it.count
				for w.CountWalls(b.Pid) < wallsWanted && b.buyWall(w) {
				}
			}
		}
		if pass == 0 && p.Money < 40 {
			break
		}
	}
	b.spendRest(w)
	// always field at least one unit
	if len(w.UnitsOf(b.Pid)) == 0 {
		for _, id := range []string{"assault", "shotgun"} {
			if b.buyUnit(w, id) {
				break
			}
		}
	}
	_ = w.SetReady(b.Pid, true)
}

func (b *Bot) buyUnit(w *sim.World, def string) bool {
	p := w.Players[b.Pid]
	d := w.Cfg.U(def)
	if d == nil || p.Money < d.Cost {
		return false
	}
	// try random positions inside the zone, favouring the middle
	for try := 0; try < 24; try++ {
		mid := float64(p.ZoneX0+p.ZoneX1) / 2
		x := mid + b.rng.Norm()*float64(sim.ZoneW)*0.22
		x = math.Max(float64(p.ZoneX0+14), math.Min(float64(p.ZoneX1-14), x))
		if w.PlaceUnit(b.Pid, def, x, float64(sim.PlateauY-60)) == nil {
			return true
		}
	}
	return false
}

func (b *Bot) buyWall(w *sim.World) bool {
	defs := []string{"sandbag", "concrete", "concrete", "armor", "sandbag"}
	def := defs[b.rng.Intn(len(defs))]
	return b.buyStruct(w, def)
}

func (b *Bot) buyStruct(w *sim.World, def string) bool {
	p := w.Players[b.Pid]
	d := w.Cfg.S(def)
	if d == nil || p.Money < d.Cost || d.Tier > w.BuildNo {
		return false
	}
	if d.Max > 0 && w.CountStructs(b.Pid, def) >= d.Max {
		return false
	}
	c0, c1 := p.ZoneX0/sim.Cell, p.ZoneX1/sim.Cell-d.W
	var cols []int
	for c := c0; c <= c1; c++ {
		cols = append(cols, c)
	}
	center := float64(c0+c1) / 2
	half := float64(c1-c0) / 2
	score := func(c int) float64 {
		// distance from the middle (0 centre .. 1 edge) plus noise
		e := math.Abs(float64(c)-center) / half
		jitter := b.rng.Range(-0.18, 0.18)
		switch d.Kind {
		case balance.SBlock, balance.SNet:
			return -e + jitter // walls first at the edges
		case balance.SAA, balance.SJammer:
			return e*0.6 + jitter // near the middle, slightly spread
		default:
			return e + jitter // weapons / economy in the middle
		}
	}
	sort.Slice(cols, func(i, j int) bool { return score(cols[i]) < score(cols[j]) })
	for _, c := range cols {
		cy := w.DropCell(c, d.W, d.H, sim.PlateauY/sim.Cell-16)
		if w.PlaceStruct(b.Pid, def, c, cy) == nil {
			return true
		}
	}
	return false
}

// spendRest burns leftover money on whatever is still useful.
func (b *Bot) spendRest(w *sim.World) {
	p := w.Players[b.Pid]
	var offence []string
	for _, s := range w.Cfg.Structs {
		switch s.Kind {
		case balance.SWeapon, balance.SAA, balance.SBunker:
			offence = append(offence, s.ID)
		}
	}
	for try := 0; try < 60 && p.Money >= 100; try++ {
		switch b.rng.Intn(5) {
		case 0, 1:
			ids := make([]string, 0, len(w.Cfg.Units))
			for _, u := range w.Cfg.Units {
				ids = append(ids, u.ID)
			}
			b.buyUnit(w, ids[b.rng.Intn(len(ids))])
		case 2:
			b.buyStruct(w, offence[b.rng.Intn(len(offence))])
		case 3:
			items := []string{"fab", "kab", "geran"}
			_ = w.BuyItem(b.Pid, items[b.rng.Intn(len(items))])
		case 4:
			b.buyWall(w)
		}
	}
	for _, s := range w.StructsOf(b.Pid) {
		if c := w.RepairCost(s); c > 0 && c < p.Money {
			_ = w.Repair(b.Pid, s.ID)
		}
	}
}

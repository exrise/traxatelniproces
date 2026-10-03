package sim

import (
	"fmt"
	"math"

	"svinovoyna/internal/balance"
)

// PlayerSetup describes one slot at match creation.
type PlayerSetup struct {
	Name  string
	Team  int
	Bot   bool
	Color int
}

// World is the entire game state.
type World struct {
	Cfg  *balance.Config
	Seed uint64
	RNG  RNG

	Tick  int
	Time  float64
	Phase Phase

	BuildNo  int // number of build phases started (1-based)
	BattleNo int
	Round    int // current battle round (1-based)

	Players []*Player
	Units   []*Unit
	Structs []*Struct
	Projs   []*Proj
	Spawns  []Spawn
	Points  []*CapturePoint
	Terr    *Terrain
	Wind    float64

	nextProj int

	cellsW, cellsH int
	occ            []int32 // struct id + 1 per cell

	Events []Event

	// build phase
	BuildTimer   float64
	BuildTimerOn bool

	// battle turn state
	Order       []int
	OrderPos    int
	Cur         int // player whose turn it is (-1 when none)
	Stage       TurnStage
	TurnTimer   float64
	RetreatTime float64
	SelUnit     int // selected unit id or -1
	SelStruct   int // selected struct id or -1
	UnitActed   bool
	StructActed bool
	FiredUnit   int // the unit that fired this turn (can still run during retreat)
	SettleT     float64
	LastUnit    map[int]int
	Plans       map[int]*Plan

	// who is currently the richest / weakest army (for catch-up multipliers)
	Leader, Underdog int

	Winner     int // winning team, -1 none yet
	WinnerName string
}

// NewWorld creates a match in build phase 1.
func NewWorld(cfg *balance.Config, seed uint64, setups []PlayerSetup) *World {
	cfg.Index()
	n := len(setups)
	w := &World{Cfg: cfg, Seed: seed, RNG: NewRNG(seed ^ 0xABCDEF), Winner: -1, Cur: -1, SelUnit: -1, SelStruct: -1, FiredUnit: -1, LastUnit: map[int]int{}, Leader: -1, Underdog: -1}
	var pts []Vec
	w.Terr, pts = GenerateTerrain(n, seed)
	for _, p := range pts {
		w.Points = append(w.Points, &CapturePoint{Pos: p, Owner: -1})
	}
	w.cellsW = (w.Terr.W + Cell - 1) / Cell
	w.cellsH = (MapH + Cell - 1) / Cell
	w.occ = make([]int32, w.cellsW*w.cellsH)

	// randomised slot -> zone assignment so map position is fair over many games
	zones := make([]int, n)
	for i := range zones {
		zones[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := w.RNG.Intn(i + 1)
		zones[i], zones[j] = zones[j], zones[i]
	}
	for i, s := range setups {
		x0, x1 := ZoneRange(n, zones[i])
		team := s.Team
		if !cfg.TeamsEnabled {
			team = i
		}
		p := &Player{ID: i, Name: s.Name, Team: team, Bot: s.Bot, Color: s.Color, Money: cfg.StartMoney,
			Zone: zones[i], ZoneX0: x0, ZoneX1: x1, HQ: -1, Items: map[string]int{}}
		w.Players = append(w.Players, p)
		w.placeHQ(p)
	}
	w.startBuild()
	return w
}

func (w *World) emit(e Event) { w.Events = append(w.Events, e) }

// DrainEvents returns and clears pending events.
func (w *World) DrainEvents() []Event {
	e := w.Events
	w.Events = nil
	return e
}

func (w *World) msg(f string, a ...any) { w.emit(Event{Type: EvMessage, Text: fmt.Sprintf(f, a...)}) }

// ---- grid ---------------------------------------------------------------

func (w *World) cellIdx(cx, cy int) int { return cy*w.cellsW + cx }

func (w *World) cellOK(cx, cy int) bool { return cx >= 0 && cy >= 0 && cx < w.cellsW && cy < w.cellsH }

// StructAtCell returns the alive structure occupying a cell or nil.
func (w *World) StructAtCell(cx, cy int) *Struct {
	if !w.cellOK(cx, cy) {
		return nil
	}
	v := w.occ[w.cellIdx(cx, cy)]
	if v == 0 {
		return nil
	}
	return w.Structs[v-1]
}

// StructAtPx returns the alive structure covering a world pixel.
func (w *World) StructAtPx(x, y float64) *Struct {
	if x < 0 || y < 0 {
		return nil
	}
	return w.StructAtCell(int(x)/Cell, int(y)/Cell)
}

func (w *World) setOcc(s *Struct, id int32) {
	d := w.Cfg.S(s.Def)
	for y := 0; y < d.H; y++ {
		for x := 0; x < d.W; x++ {
			if w.cellOK(s.CX+x, s.CY+y) {
				w.occ[w.cellIdx(s.CX+x, s.CY+y)] = id
			}
		}
	}
}

// SolidPx: terrain or structure blocks movement here.
func (w *World) SolidPx(x, y float64) bool {
	if w.Terr.Solid(int(math.Floor(x)), int(math.Floor(y))) {
		return true
	}
	return w.StructAtPx(x, y) != nil
}

// StructRect returns the pixel rectangle of a structure.
func (w *World) StructRect(s *Struct) (x0, y0, x1, y1 float64) {
	d := w.Cfg.S(s.Def)
	return float64(s.CX * Cell), float64(s.CY * Cell), float64((s.CX + d.W) * Cell), float64((s.CY + d.H) * Cell)
}

// StructCenter returns the centre of a structure in pixels.
func (w *World) StructCenter(s *Struct) Vec {
	x0, y0, x1, y1 := w.StructRect(s)
	return Vec{(x0 + x1) / 2, (y0 + y1) / 2}
}

// StructMuzzle is where a weapon structure's shots originate.
func (w *World) StructMuzzle(s *Struct) Vec {
	x0, y0, x1, _ := w.StructRect(s)
	return Vec{(x0 + x1) / 2, y0 - 2}
}

// ---- helpers ------------------------------------------------------------

func (w *World) addStruct(owner int, def string, cx, cy int) *Struct {
	d := w.Cfg.S(def)
	s := &Struct{ID: len(w.Structs), Owner: owner, Def: def, CX: cx, CY: cy, HP: d.HP, MaxHP: d.HP, Alive: true, Aim: -math.Pi / 3}
	if def == "hq" {
		s.HP, s.MaxHP = w.Cfg.MaxHQHP, w.Cfg.MaxHQHP
	}
	if d.Kind == balance.SWeapon {
		if wd := w.Cfg.W(d.Weapon); wd != nil {
			s.Ammo = wd.Ammo
		}
	}
	s.AAAmmo = d.AAAmmo
	w.Structs = append(w.Structs, s)
	w.setOcc(s, int32(s.ID+1))
	return s
}

func (w *World) addUnit(owner int, def string, pos Vec) *Unit {
	d := w.Cfg.U(def)
	face := 1
	if p := w.Players[owner]; pos.X > float64(p.ZoneX0+p.ZoneX1)/2 {
		face = -1
	}
	u := &Unit{ID: len(w.Units), Owner: owner, Def: def, Pos: pos, HP: d.HP, MaxHP: d.HP, Alive: true, Face: face, Used: map[string]int{}, Aim: 0}
	if face < 0 {
		u.Aim = math.Pi
	}
	u.Aim += -0.3 * float64(face)
	w.Units = append(w.Units, u)
	return u
}

// placeHQ auto-places the free HQ in the middle of the player's zone.
func (w *World) placeHQ(p *Player) {
	d := w.Cfg.S("hq")
	cx := (p.ZoneX0+p.ZoneX1)/2/Cell - d.W/2
	// the plateau is flat; find the lowest y where the footprint fits above ground
	cy := w.dropCell(cx, d.W, d.H, 0)
	s := w.addStruct(p.ID, "hq", cx, cy)
	p.HQ = s.ID
}

// dropCell returns the top cell row for a footprint dropped straight down at column cx.
func (w *World) dropCell(cx, cw, ch, from int) int {
	cy := from
	for ; cy+ch < w.cellsH; cy++ {
		if w.supportedAt(cx, cy, cw, ch, -1) {
			break
		}
	}
	return cy
}

// cellFree: no terrain pixels and no structure inside the cell.
func (w *World) cellFree(cx, cy int, ignore int) bool {
	if !w.cellOK(cx, cy) {
		return false
	}
	if v := w.occ[w.cellIdx(cx, cy)]; v != 0 && int(v-1) != ignore {
		return false
	}
	x0, y0 := cx*Cell, cy*Cell
	for y := y0; y < y0+Cell; y += 3 {
		for x := x0; x < x0+Cell; x += 3 {
			if w.Terr.Solid(x, y) {
				return false
			}
		}
	}
	return true
}

// supportedAt: footprint rests on terrain or another structure.
func (w *World) supportedAt(cx, cy, cw, ch, ignore int) bool {
	by := cy + ch
	for i := 0; i < cw; i++ {
		if !w.cellOK(cx+i, by) {
			return true // map bottom
		}
		if v := w.occ[w.cellIdx(cx+i, by)]; v != 0 && int(v-1) != ignore {
			return true
		}
		px, py := (cx+i)*Cell+Cell/2, by*Cell+1
		if w.Terr.Solid(px, py) || w.Terr.Solid(px-5, py) || w.Terr.Solid(px+5, py) {
			return true
		}
	}
	return false
}

// Alive team helpers ------------------------------------------------------

// PlayerAlive: has an HQ or at least one living unit.
func (w *World) PlayerAlive(id int) bool {
	p := w.Players[id]
	if p.Elim {
		return false
	}
	if p.HQ >= 0 && w.Structs[p.HQ].Alive {
		return true
	}
	for _, u := range w.Units {
		if u.Owner == id && u.Alive {
			return true
		}
	}
	return false
}

// HasHQ reports whether the player still owns a living HQ.
func (w *World) HasHQ(id int) bool {
	p := w.Players[id]
	return p.HQ >= 0 && w.Structs[p.HQ].Alive
}

// AliveTeams returns the number of teams with at least one alive player.
func (w *World) AliveTeams() (n int, last int) {
	seen := map[int]bool{}
	last = -1
	for _, p := range w.Players {
		if w.PlayerAlive(p.ID) && !seen[p.Team] {
			seen[p.Team] = true
			n++
			last = p.Team
		}
	}
	return
}

// Hostile reports whether a and b are on different teams.
func (w *World) Hostile(a, b int) bool {
	if a < 0 || b < 0 {
		return false
	}
	return w.Players[a].Team != w.Players[b].Team
}

// UnitsOf lists alive units of a player.
func (w *World) UnitsOf(id int) []*Unit {
	var out []*Unit
	for _, u := range w.Units {
		if u.Owner == id && u.Alive {
			out = append(out, u)
		}
	}
	return out
}

// StructsOf lists alive structures of a player.
func (w *World) StructsOf(id int) []*Struct {
	var out []*Struct
	for _, s := range w.Structs {
		if s.Owner == id && s.Alive {
			out = append(out, s)
		}
	}
	return out
}

// ArmyValue is the money worth of everything alive a player owns.
func (w *World) ArmyValue(id int) int {
	v := 0
	for _, u := range w.Units {
		if u.Owner == id && u.Alive {
			v += int(float64(w.Cfg.U(u.Def).Cost) * u.HP / u.MaxHP)
		}
	}
	for _, s := range w.Structs {
		if s.Owner == id && s.Alive && s.Def != "hq" {
			v += int(float64(w.Cfg.S(s.Def).Cost) * s.HP / s.MaxHP)
		}
	}
	return v
}

func (w *World) updateLeaders() {
	w.Leader, w.Underdog = -1, -1
	best, worst := -1, 1<<60
	cnt := 0
	for _, p := range w.Players {
		if !w.PlayerAlive(p.ID) {
			continue
		}
		cnt++
		v := w.ArmyValue(p.ID)
		if v > best {
			best, w.Leader = v, p.ID
		}
		if v < worst {
			worst, w.Underdog = v, p.ID
		}
	}
	if cnt < 2 || best-worst < 150 {
		w.Leader, w.Underdog = -1, -1
	}
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// Package balance holds every tunable number of the game. The simulation never
// hard-codes balance values: it reads them from a Config, so presets and the
// lobby editor can change anything.
package balance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// TurnMode selects how turns are organised during the battle phase.
type TurnMode int

const (
	// TurnClassic: one action per turn — a unit OR a weapon structure.
	TurnClassic TurnMode = iota
	// TurnUnitAndStruct: a unit action AND one structure action per turn.
	TurnUnitAndStruct
	// TurnSimultaneous: every player plans an action, then all are resolved together.
	TurnSimultaneous
)

func (t TurnMode) String() string {
	switch t {
	case TurnClassic:
		return "Классика Worms"
	case TurnUnitAndStruct:
		return "Юнит + постройка"
	case TurnSimultaneous:
		return "Одновременные ходы"
	}
	return "?"
}

// WeaponKind picks the simulation behaviour of a weapon.
type WeaponKind string

const (
	KindBurst     WeaponKind = "burst"     // several fast bullets (rifle, MG)
	KindPellets   WeaponKind = "pellets"   // shotgun cone
	KindShot      WeaponKind = "shot"      // single fast bullet (sniper)
	KindShell     WeaponKind = "shell"     // ballistic explosive (grenade, mortar, RPG, howitzer)
	KindSalvo     WeaponKind = "salvo"     // many rockets with spread (Grad)
	KindMissile   WeaponKind = "missile"   // guided / direct-fire missile (Kornet)
	KindBallis    WeaponKind = "ballis"    // ballistic missile from above (Iskander)
	KindMIRV      WeaponKind = "mirv"      // multiple warheads (Oreshnik)
	KindDrone     WeaponKind = "drone"     // player-steered kamikaze drone
	KindAirstrike WeaponKind = "airstrike" // plane drops bombs while crossing the map
	KindGeran     WeaponKind = "geran"     // slow autonomous loitering drone
	KindMine      WeaponKind = "mine"      // placed mine
	KindRepair    WeaponKind = "repair"    // engineer repair
)

// AAClass classifies a flying thing for air-defence purposes.
type AAClass string

const (
	ClassNone     AAClass = ""
	ClassDrone    AAClass = "drone"
	ClassAir      AAClass = "air"    // plane / guided bomb
	ClassRocket   AAClass = "rocket" // artillery rockets & shells with a tall arc
	ClassBallis   AAClass = "ballis" // ballistic missiles
	ClassOreshnik AAClass = "oreshnik"
)

// Weapon describes one weapon, used by units, structures and consumables.
type Weapon struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Kind     WeaponKind `json:"kind"`
	Damage   float64    `json:"damage"`    // direct hit / per bullet damage
	Radius   float64    `json:"radius"`    // explosion radius (px)
	Count    int        `json:"count"`     // bullets / pellets / rockets / bombs / warheads
	Spread   float64    `json:"spread"`    // radians, random cone
	Speed    float64    `json:"speed"`     // muzzle speed (px/s) at full power
	Range    float64    `json:"range"`     // max range for hitscan-like weapons
	Gravity  float64    `json:"gravity"`   // multiplier on world gravity
	WindK    float64    `json:"wind_k"`    // how much wind pushes it
	Fuse     float64    `json:"fuse"`      // seconds until detonation (grenades), 0 = on impact
	Falloff  float64    `json:"falloff"`   // per-pixel damage falloff for pellets (0..1 over Range)
	Pierce   float64    `json:"pierce"`    // 0..1 fraction of structure armor ignored
	BlockMul float64    `json:"block_mul"` // damage multiplier vs structure blocks
	Crater   float64    `json:"crater"`    // terrain crater radius multiplier (1 = Radius)
	Class    AAClass    `json:"class"`     // air-defence classification of the projectile
	Flight   float64    `json:"flight"`    // drone flight time (s)
	Ammo     int        `json:"ammo"`      // shots per battle (structures); 0 = unlimited in-turn
	Cost     int        `json:"cost"`      // price for consumables (airstrikes, mines)
}

// UnitDef is a pig type.
type UnitDef struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Cost    int      `json:"cost"`
	HP      float64  `json:"hp"`
	Weapons []string `json:"weapons"`
	Max     int      `json:"max"` // max per player, 0 = unlimited
	Desc    string   `json:"desc"`
}

// StructKind is the role of a structure.
type StructKind string

const (
	SBlock  StructKind = "block"
	SWeapon StructKind = "weapon"
	SAA     StructKind = "aa"
	SEco    StructKind = "eco"
	SHQ     StructKind = "hq"
	SBunker StructKind = "bunker"
	SNet    StructKind = "net" // anti-drone net
	SJammer StructKind = "jammer"
)

// StructDef is a placeable structure. Size is in 16px grid cells.
type StructDef struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Kind   StructKind `json:"kind"`
	Cost   int        `json:"cost"`
	W      int        `json:"w"`
	H      int        `json:"h"`
	HP     float64    `json:"hp"`
	Armor  float64    `json:"armor"`  // 0..1 damage reduction vs. bullets/shrapnel
	Blast  float64    `json:"blast"`  // 0..1 damage reduction vs. explosions
	Weapon string     `json:"weapon"` // for SWeapon
	Max    int        `json:"max"`    // max per player, 0 = unlimited
	Income int        `json:"income"` // per own turn, for SEco
	Tier   int        `json:"tier"`   // earliest build phase number (1-based) it unlocks in
	Desc   string     `json:"desc"`
	// Air defence parameters (SAA / SJammer).
	AARange float64             `json:"aa_range"`
	AAHit   map[AAClass]float64 `json:"aa_hit"` // probability to intercept per attempt
	AAAmmo  int                 `json:"aa_ammo"`
}

// Config is the complete set of tunables for a match.
type Config struct {
	Name string `json:"name"`

	// Phases
	TurnMode          TurnMode `json:"turn_mode"`
	StartMoney        int      `json:"start_money"`
	BuildTimeFirst    float64  `json:"build_time_first"`
	BuildTime         float64  `json:"build_time"`
	RoundsPerBattle   int      `json:"rounds_per_battle"`
	AutoRounds        bool     `json:"auto_rounds"` // fewer rounds with more players (-1 per extra player)
	TurnTime          float64  `json:"turn_time"`
	RetreatTime       float64  `json:"retreat_time"`
	PlanTime          float64  `json:"plan_time"` // simultaneous mode planning time
	MaxHQHP           float64  `json:"hq_hp"`
	SuddenDeathBattle int      `json:"sudden_death_battle"` // from this battle on HQs burn every round (0 = off)
	SuddenDeathDmg    float64  `json:"sudden_death_dmg"`

	// Economy
	BaseIncome     int     `json:"base_income"`      // each build phase
	DmgMoneyUnit   float64 `json:"dmg_money_unit"`   // $ per damage point dealt to units
	DmgMoneyStruct float64 `json:"dmg_money_struct"` // $ per damage point dealt to structures
	KillBonus      int     `json:"kill_bonus"`
	StructKillPct  float64 `json:"struct_kill_pct"` // fraction of structure cost paid on destruction
	CaptureIncome  int     `json:"capture_income"`  // per own turn per held point
	SellRefund     float64 `json:"sell_refund"`
	RepairCostPct  float64 `json:"repair_cost_pct"` // cost per fraction of HP restored
	LeaderMult     float64 `json:"leader_mult"`     // damage-money multiplier against the richest army
	UnderdogMult   float64 `json:"underdog_mult"`   // ... against the weakest army
	CatchUpPct     float64 `json:"catch_up_pct"`    // refund of losses at next build phase
	CatchUpCap     int     `json:"catch_up_cap"`
	TwoHandDmgMul  float64 `json:"two_hand_dmg_mul"` // damage multiplier in TurnUnitAndStruct
	FriendlyFire   bool    `json:"friendly_fire"`
	TeamsEnabled   bool    `json:"teams"`
	MaxUnits       int     `json:"max_units"`
	GravityPx      float64 `json:"gravity"`
	WindMax        float64 `json:"wind_max"`
	FallDamage     float64 `json:"fall_damage"`
	BuildHeal      float64 `json:"build_heal"` // HP restored to units at each build phase
	UnitSpeed      float64 `json:"unit_speed"`

	Weapons []Weapon    `json:"weapons"`
	Units   []UnitDef   `json:"units"`
	Structs []StructDef `json:"structs"`

	wIdx map[string]int
	uIdx map[string]int
	sIdx map[string]int
}

// Index builds the lookup tables. Must be called after loading/modifying slices.
func (c *Config) Index() {
	c.wIdx = map[string]int{}
	c.uIdx = map[string]int{}
	c.sIdx = map[string]int{}
	for i, w := range c.Weapons {
		c.wIdx[w.ID] = i
	}
	for i, u := range c.Units {
		c.uIdx[u.ID] = i
	}
	for i, s := range c.Structs {
		c.sIdx[s.ID] = i
	}
}

// W returns the weapon with the given id (nil if unknown).
func (c *Config) W(id string) *Weapon {
	if c.wIdx == nil {
		c.Index()
	}
	if i, ok := c.wIdx[id]; ok {
		return &c.Weapons[i]
	}
	return nil
}

// U returns a unit definition (nil if unknown).
func (c *Config) U(id string) *UnitDef {
	if c.uIdx == nil {
		c.Index()
	}
	if i, ok := c.uIdx[id]; ok {
		return &c.Units[i]
	}
	return nil
}

// S returns a structure definition (nil if unknown).
func (c *Config) S(id string) *StructDef {
	if c.sIdx == nil {
		c.Index()
	}
	if i, ok := c.sIdx[id]; ok {
		return &c.Structs[i]
	}
	return nil
}

// Clone makes a deep copy (via JSON) so presets can be modified safely.
func (c *Config) Clone() *Config {
	b, _ := json.Marshal(c)
	var n Config
	if err := json.Unmarshal(b, &n); err != nil {
		panic(err)
	}
	n.Index()
	return &n
}

// Validate checks internal consistency and returns all problems found.
func (c *Config) Validate() []string {
	c.Index()
	var errs []string
	add := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }
	if c.StartMoney <= 0 {
		add("start_money must be > 0")
	}
	if c.RoundsPerBattle < 1 {
		add("rounds_per_battle must be >= 1")
	}
	if c.TurnTime < 5 {
		add("turn_time too small")
	}
	seen := map[string]bool{}
	for _, w := range c.Weapons {
		if seen["w"+w.ID] {
			add("duplicate weapon %s", w.ID)
		}
		seen["w"+w.ID] = true
		if w.Damage < 0 || w.Radius < 0 {
			add("weapon %s has negative damage/radius", w.ID)
		}
	}
	for _, u := range c.Units {
		if seen["u"+u.ID] {
			add("duplicate unit %s", u.ID)
		}
		seen["u"+u.ID] = true
		if u.HP <= 0 || u.Cost < 0 {
			add("unit %s has bad hp/cost", u.ID)
		}
		if len(u.Weapons) == 0 {
			add("unit %s has no weapons", u.ID)
		}
		for _, w := range u.Weapons {
			if c.W(w) == nil {
				add("unit %s uses unknown weapon %s", u.ID, w)
			}
		}
	}
	hq := false
	for _, s := range c.Structs {
		if seen["s"+s.ID] {
			add("duplicate struct %s", s.ID)
		}
		seen["s"+s.ID] = true
		if s.W < 1 || s.H < 1 || s.HP <= 0 {
			add("struct %s has bad size/hp", s.ID)
		}
		if s.Kind == SWeapon && c.W(s.Weapon) == nil {
			add("struct %s uses unknown weapon %s", s.ID, s.Weapon)
		}
		if s.Kind == SHQ {
			hq = true
		}
	}
	if !hq {
		add("no HQ structure defined")
	}
	sort.Strings(errs)
	return errs
}

// Marshal pretty-prints the config.
func (c *Config) Marshal() []byte {
	b, _ := json.MarshalIndent(c, "", "  ")
	return b
}

// Unmarshal parses and indexes a config.
func Unmarshal(b []byte) (*Config, error) {
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	c.Index()
	if errs := c.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("invalid config: %v", errs)
	}
	return &c, nil
}

// SaveCustom writes a custom preset into dir (created if needed).
func SaveCustom(dir string, c *Config) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := c.Name
	if name == "" {
		name = "custom"
	}
	return os.WriteFile(filepath.Join(dir, name+".json"), c.Marshal(), 0o644)
}

// LoadCustom loads all *.json presets from dir; broken files are skipped.
func LoadCustom(dir string) []*Config {
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	var out []*Config
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if c, err := Unmarshal(b); err == nil {
			out = append(out, c)
		}
	}
	return out
}

// RoundsFor returns the number of battle rounds for n players.
func (c *Config) RoundsFor(n int) int {
	r := c.RoundsPerBattle
	if c.AutoRounds && n > 2 {
		r -= n - 2
	}
	if r < 2 {
		r = 2
	}
	return r
}

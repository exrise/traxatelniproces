// Package sim is the whole game simulation. It has no dependency on the
// graphics library so it can run headless (tests, bots, the dedicated host).
package sim

import (
	"math"

	"svinovoyna/internal/balance"
)

// Fixed simulation step.
const (
	TickRate = 60
	Dt       = 1.0 / TickRate

	Cell = 16 // structure grid size in pixels

	MapH      = 1008
	WaterY    = MapH - 60
	ZoneW     = 480 // build zone width in px
	ZoneStep  = 1100
	MapMargin = 400

	UnitW = 14
	UnitH = 18
)

// Vec is a 2D vector in world pixels (Y grows downward).
type Vec struct{ X, Y float64 }

func (a Vec) Add(b Vec) Vec      { return Vec{a.X + b.X, a.Y + b.Y} }
func (a Vec) Sub(b Vec) Vec      { return Vec{a.X - b.X, a.Y - b.Y} }
func (a Vec) Mul(k float64) Vec  { return Vec{a.X * k, a.Y * k} }
func (a Vec) Len() float64       { return math.Hypot(a.X, a.Y) }
func (a Vec) Dist(b Vec) float64 { return math.Hypot(a.X-b.X, a.Y-b.Y) }
func Dir(angle float64) Vec      { return Vec{math.Cos(angle), math.Sin(angle)} }

// Phase of the match.
type Phase int

const (
	PhaseLobby Phase = iota
	PhaseBuild
	PhaseBattle
	PhaseOver
)

// TurnStage is the state of the current battle turn.
type TurnStage int

const (
	StageActive  TurnStage = iota // player may act
	StageRetreat                  // action done; short window to run for cover
	StageSettle                   // waiting for projectiles / falling things to finish
	StagePlan                     // simultaneous mode: everyone plans
	StageResolve                  // simultaneous mode: plans are being executed
)

// Player is a participant.
type Player struct {
	ID     int
	Name   string
	Team   int
	Color  int // palette index
	Bot    bool
	Money  int
	Zone   int // zone index (position on the map)
	ZoneX0 int // build zone in px
	ZoneX1 int
	Ready  bool
	Elim   bool
	HQ     int            // struct id, -1 if none
	Items  map[string]int // consumables: fab, kab, geran
	// per-battle bookkeeping
	LossValue  int
	Earned     int
	DamageDone float64
	Kills      int
	Spent      int
	// summary of the last battle for the UI
	LastEarned int
	Frac       float64 // fractional money carried between hits
}

// Unit is a pig soldier.
type Unit struct {
	ID        int
	Owner     int
	Def       string
	Pos       Vec // feet centre
	Vel       Vec
	HP        float64
	MaxHP     float64
	Alive     bool
	Face      int // -1 left, 1 right
	Ground    bool
	Aim       float64
	Walk      int // current walk direction while it is this unit's turn
	WeaponIdx int
	Used      map[string]int
	FallFrom  float64
	Anim      float64
	Hurt      float64 // flash timer
	Spent     bool    // acted this turn
}

// Struct is a placed building. Cells (CX,CY) is the top-left grid cell.
type Struct struct {
	ID     int
	Owner  int
	Def    string
	CX, CY int
	HP     float64
	MaxHP  float64
	Alive  bool
	Ammo   int
	AAAmmo int
	Aim    float64
	Fall   float64 // fall timer
	Hurt   float64
	Acted  bool
}

// ProjKind enumerates projectile behaviours.
type ProjKind int

const (
	PShell ProjKind = iota
	PRocket
	PBallis  // ballistic missile (ascent, then descent)
	PWarhead // descending MIRV warhead
	PPlane   // aircraft crossing the map
	PBomb    // falling bomb
	PDrone   // steerable FPV drone
	PGeran   // autonomous loitering munition
	PMine
)

// Proj is any moving or armed object spawned by a weapon.
type Proj struct {
	ID      int
	Kind    ProjKind
	Owner   int
	Weapon  string
	Pos     Vec
	Vel     Vec
	Age     float64
	Fuse    float64
	Alive   bool
	Class   balance.AAClass
	Phase   int
	Target  Vec
	Heading float64
	Steer   float64
	Flight  float64
	Left    int     // bombs / warheads still to drop
	Next    float64 // next drop x / timer
	SpreadK float64
	Dmg     float64 // damage multiplier already applied (turn mode)
	Tried   []int   // ids of AA structures that already rolled against this
	Armed   float64
	Index   int
}

// Spawn is a delayed projectile (salvo, bomb run).
type Spawn struct {
	At     float64
	Parent int // projectile id this spawn depends on (-1 none)
	Proj   Proj
}

// CapturePoint is a neutral tower that pays income to its holder.
type CapturePoint struct {
	Pos      Vec
	Owner    int // -1 neutral
	Progress float64
}

// EventType enumerates things the UI wants to know about.
type EventType int

const (
	EvExplosion EventType = iota
	EvCrater
	EvTracer
	EvShot
	EvUnitDied
	EvStructDestroyed
	EvDamage
	EvSplash
	EvTurn
	EvPhase
	EvMoney
	EvIntercept
	EvMessage
	EvCapture
	EvPlane
	EvBuild
	EvHeal
	EvWinner
	EvJump
)

// Event is a transient notification for the UI/sound/network layer.
type Event struct {
	Type EventType
	Pos  Vec
	To   Vec
	R    float64
	A, B int
	F    float64
	Text string
}

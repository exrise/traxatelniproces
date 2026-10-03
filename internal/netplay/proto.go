// Package netplay implements the TCP multiplayer: an authoritative host that
// runs the simulation and thin clients that send commands and render
// snapshots. It is meant for LAN / Radmin VPN play.
package netplay

import (
	"svinovoyna/internal/sim"
)

// Version must match between host and client.
const Version = "svino-1"

// DefaultPort is the TCP port used by default.
const DefaultPort = "27015"

// SlotKind describes a seat in the network lobby.
type SlotKind int

const (
	SlotHost   SlotKind = iota // the host's own seat
	SlotOpen                   // waiting for a remote player
	SlotRemote                 // a connected remote player
	SlotBot                    // computer player
)

// LobbySlot is one seat.
type LobbySlot struct {
	Kind  SlotKind
	Name  string
	Skill float64
	Color int
	Team  int
}

// LobbyState is what the host broadcasts while players gather.
type LobbyState struct {
	Slots   []LobbySlot
	CfgJSON []byte
	Seed    uint64
	Preset  string
}

// Msg is the single wire message type (gob-encoded, one stream per connection).
type Msg struct {
	Kind    string // hello welcome reject lobby start cmd snap bye
	Name    string
	Version string
	Seat    int
	Reason  string
	Lobby   *LobbyState
	Cmd     *sim.Command
	Snap    *Snapshot
}

// PlayerList etc. wrap slices so that "nil" can mean "unchanged" on the wire.
type PlayerList struct {
	OK bool
	L  []*sim.Player
}

type UnitList struct {
	OK bool
	L  []*sim.Unit
}

type StructList struct {
	OK bool
	L  []*sim.Struct
}

type PointList struct {
	OK bool
	L  []*sim.CapturePoint
}

// Snapshot is the replicated dynamic state of a world.
type Snapshot struct {
	Tick         int
	Time         float64
	Phase        sim.Phase
	BuildNo      int
	BattleNo     int
	Round        int
	Wind         float64
	Cur          int
	Stage        sim.TurnStage
	TurnTimer    float64
	RetreatTime  float64
	BuildTimer   float64
	BuildTimerOn bool
	SelUnit      int
	SelStruct    int
	UnitActed    bool
	StructActed  bool
	FiredUnit    int
	SettleT      float64
	Order        []int
	OrderPos     int
	Leader       int
	Underdog     int
	Winner       int
	WinnerName   string

	Players *PlayerList
	Units   *UnitList
	Structs *StructList
	Points  *PointList
	Projs   []*sim.Proj

	CraterBase int
	Craters    []sim.Crater
	Events     []sim.Event
}

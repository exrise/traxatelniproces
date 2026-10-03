package netplay

import (
	"hash/fnv"
	"math"
	"net"
	"testing"
	"time"

	"svinovoyna/internal/balance"
	"svinovoyna/internal/sim"
)

func maskHash(w *sim.World) uint64 {
	h := fnv.New64a()
	h.Write(w.Terr.Mask)
	return h.Sum64()
}

func TestHostClientRoundTrip(t *testing.T) {
	cfg := balance.Default()
	lob := LobbyState{Slots: []LobbySlot{{Kind: SlotHost, Name: "Host"}, {Kind: SlotOpen}}, CfgJSON: cfg.Marshal(), Seed: 42}
	h, err := Listen("0", lob)
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	port := h.Addr().(*net.TCPAddr).Port
	cl, err := Dial("127.0.0.1:"+itoa(port), "Guest")
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	if cl.Seat != 1 || cl.Lobby.Slots[1].Name != "Guest" {
		t.Fatalf("bad welcome: seat=%d lobby=%+v", cl.Seat, cl.Lobby.Slots)
	}
	if _, err := h.Start(); err != nil {
		t.Fatal(err)
	}
	// client should get "start"
	deadline := time.Now().Add(3 * time.Second)
	started := false
	for time.Now().Before(deadline) && !started {
		ms, _ := cl.Poll()
		for _, m := range ms {
			if m.Kind == "start" {
				started = true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !started {
		t.Fatal("client never received start")
	}

	setups := []sim.PlayerSetup{{Name: "Host"}, {Name: "Guest", Color: 1}}
	hw := sim.NewWorld(cfg.Clone(), 42, setups)
	cw := sim.NewWorld(cfg.Clone(), 42, setups)
	if maskHash(hw) != maskHash(cw) {
		t.Fatal("terrain generation is not deterministic")
	}
	for _, p := range hw.Players {
		x := float64(p.ZoneX0+p.ZoneX1) / 2
		if err := hw.PlaceUnit(p.ID, "mortar", x+100, 100); err != nil {
			t.Fatal(err)
		}
		hw.SetReady(p.ID, true)
	}
	st := h.State(1)
	if st == nil {
		t.Fatal("no send state")
	}
	var evs []sim.Event
	pump := func() {
		snap := Build(hw, 1, st, evs)
		evs = nil
		h.SendSnap(1, snap)
		time.Sleep(30 * time.Millisecond)
		ms, ok := cl.Poll()
		if !ok {
			t.Fatal("client disconnected")
		}
		for _, m := range ms {
			if m.Kind == "snap" {
				Apply(cw, m.Snap)
			}
		}
	}
	for i := 0; i < 5; i++ {
		hw.Step()
		evs = append(evs, hw.DrainEvents()...)
		pump()
	}
	if hw.Phase != sim.PhaseBattle {
		t.Fatalf("host not in battle: %s", hw.StatusLine())
	}
	// fire something that digs craters
	cur := hw.Cur
	u := hw.Units[hw.SelUnit]
	if err := hw.Apply(sim.Command{Player: cur, Type: sim.CmdFire, Weapon: "mortar", Angle: -math.Pi / 3, Power: 0.8}); err != nil {
		t.Fatal(err)
	}
	_ = u
	for i := 0; i < 60*8; i++ {
		hw.Step()
		evs = append(evs, hw.DrainEvents()...)
		if i%3 == 0 {
			pump()
		}
	}
	pump()
	if len(hw.Terr.Craters) == 0 {
		t.Fatal("expected craters on host")
	}
	if maskHash(hw) != maskHash(cw) {
		t.Fatalf("terrain diverged: host %d craters, client %d", len(hw.Terr.Craters), len(cw.Terr.Craters))
	}
	if cw.Phase != hw.Phase || cw.Cur != hw.Cur || cw.Players[0].Money != hw.Players[0].Money {
		t.Fatalf("state diverged: host %s cur=%d, client %s cur=%d", hw.StatusLine(), hw.Cur, cw.StatusLine(), cw.Cur)
	}
	for i, u := range hw.Units {
		cu := cw.Units[i]
		if cu.Alive != u.Alive || math.Abs(cu.Pos.X-u.Pos.X) > 0.001 || math.Abs(cu.HP-u.HP) > 0.001 {
			t.Fatalf("unit %d diverged: %+v vs %+v", i, u, cu)
		}
	}
}

func itoa(i int) string {
	return string([]byte(func() string {
		if i == 0 {
			return "0"
		}
		var b []byte
		for i > 0 {
			b = append([]byte{byte('0' + i%10)}, b...)
			i /= 10
		}
		return string(b)
	}()))
}

package ui

import (
	"svinovoyna/internal/ai"
	"svinovoyna/internal/balance"
	"svinovoyna/internal/netplay"
	"svinovoyna/internal/sim"
)

// HostSession runs the authoritative simulation and streams it to remote players.
type HostSession struct {
	*LocalSession
	host   *netplay.Host
	me     int
	remote map[int]bool
	evBuf  []sim.Event
	tick   int
	skill  map[int]float64
	seed   uint64
}

// NewHostSession starts a hosted match. slots[0] is the host's own seat; remote lists
// the seats that are controlled by connected clients.
func NewHostSession(host *netplay.Host, cfg *balance.Config, seed uint64, slots []Slot, remote []int) *HostSession {
	ls := NewLocalSession(cfg, seed, slots)
	// every non-bot seat is a human, but only seat 0 is local
	ls.humans = nil
	ls.bots = map[int]*ai.Bot{}
	hs := &HostSession{LocalSession: ls, host: host, me: 0, remote: map[int]bool{}, skill: map[int]float64{}, seed: seed}
	for _, r := range remote {
		hs.remote[r] = true
	}
	for i, sl := range slots {
		if sl.Human {
			ls.humans = append(ls.humans, i)
		} else {
			ls.bots[i] = ai.New(i, ai.Style(int(seed+uint64(i)*7)%int(ai.NumStyles)), sl.Skill, seed)
		}
		hs.skill[i] = sl.Skill
	}
	ls.hotseat = false
	return hs
}

func (s *HostSession) Local(pid int) bool { return pid == s.me }

func (s *HostSession) IsHuman(pid int) bool { return s.LocalSession.IsHuman(pid) }

func (s *HostSession) Send(c sim.Command) {
	c.Player = s.me
	if err := s.W.Apply(c); err != nil {
		s.LocalSession.lastErr = err.Error()
	}
}

func (s *HostSession) BuildSeat() int {
	if s.W.Phase == sim.PhaseBuild && !s.W.Players[s.me].Ready {
		return s.me
	}
	return -1
}

func (s *HostSession) BuildSeatTime() float64 { return s.W.BuildTimer }

func (s *HostSession) Close() { s.host.Close() }

func (s *HostSession) Status() string { return "хост" }

func (s *HostSession) dropToBot(seat int) {
	delete(s.remote, seat)
	hum := s.humans[:0]
	for _, h := range s.humans {
		if h != seat {
			hum = append(hum, h)
		}
	}
	s.humans = hum
	s.bots[seat] = ai.New(seat, ai.Style(int(s.seed+uint64(seat)*7)%int(ai.NumStyles)), 0.7, s.seed)
	s.W.Players[seat].Bot = true
	s.W.Players[seat].Name += " (бот)"
	if s.W.Phase == sim.PhaseBuild {
		s.bots[seat].Build(s.W)
	}
}

func (s *HostSession) Update() {
	w := s.W
	for _, ic := range s.host.Poll() {
		if s.remote[ic.Seat] && ic.Msg.Cmd != nil {
			c := *ic.Msg.Cmd
			c.Player = ic.Seat
			_ = w.Apply(c)
		}
	}
	for _, seat := range s.host.Dropped() {
		if s.remote[seat] {
			s.dropToBot(seat)
		}
	}
	s.LocalSession.Update()
	s.evBuf = append(s.evBuf, w.Events...)
	s.tick++
	if s.tick%3 == 0 {
		for seat := range s.remote {
			if st := s.host.State(seat); st != nil {
				s.host.SendSnap(seat, netplay.Build(w, seat, st, s.evBuf))
			}
		}
		s.evBuf = nil
	}
	if len(s.evBuf) > 4000 {
		s.evBuf = nil
	}
}

// ClientSession mirrors a world that is simulated on the host.
type ClientSession struct {
	cl   *netplay.Client
	w    *sim.World
	me   int
	lost bool
}

// NewClientSession builds the replica world from the lobby info of the start message.
func NewClientSession(cl *netplay.Client, lob netplay.LobbyState, seat int) (*ClientSession, error) {
	cfg, err := balance.Unmarshal(lob.CfgJSON)
	if err != nil {
		return nil, err
	}
	var setups []sim.PlayerSetup
	for _, sl := range lob.Slots {
		setups = append(setups, sim.PlayerSetup{Name: sl.Name, Team: sl.Team, Bot: sl.Kind == netplay.SlotBot, Color: sl.Color})
	}
	return &ClientSession{cl: cl, w: sim.NewWorld(cfg, lob.Seed, setups), me: seat}, nil
}

func (s *ClientSession) World() *sim.World      { return s.w }
func (s *ClientSession) IsHuman(pid int) bool   { return !s.w.Players[pid].Bot }
func (s *ClientSession) Local(pid int) bool     { return pid == s.me }
func (s *ClientSession) Close()                 { s.cl.Close() }
func (s *ClientSession) Status() string         { return "клиент" }
func (s *ClientSession) BuildSeatTime() float64 { return s.w.BuildTimer }

func (s *ClientSession) BuildSeat() int {
	if s.w.Phase == sim.PhaseBuild && !s.w.Players[s.me].Ready {
		return s.me
	}
	return -1
}

func (s *ClientSession) Send(c sim.Command) {
	c.Player = s.me
	s.cl.SendCmd(c)
}

func (s *ClientSession) Update() {
	w := s.w
	msgs, ok := s.cl.Poll()
	got := false
	for _, m := range msgs {
		if m.Kind == "snap" && m.Snap != nil {
			netplay.Apply(w, m.Snap)
			got = true
		}
	}
	if !ok && !s.lost {
		s.lost = true
		if w.Phase != sim.PhaseOver {
			w.Phase = sim.PhaseOver
			w.Winner = -2
			w.WinnerName = "Связь с хостом потеряна"
		}
	}
	if !got && w.Phase != sim.PhaseOver {
		// smooth the gap between snapshots
		w.Time += sim.Dt
		w.TurnTimer -= sim.Dt
		w.BuildTimer -= sim.Dt
		w.RetreatTime -= sim.Dt
		for _, p := range w.Projs {
			if p.Alive {
				p.Pos = p.Pos.Add(p.Vel.Mul(sim.Dt))
				if p.Kind == sim.PShell || p.Kind == sim.PBomb {
					p.Vel.Y += w.Cfg.GravityPx * sim.Dt
				}
			}
		}
	}
}

// PopError implements Session (the host rejects silently).
func (s *ClientSession) PopError() string { return "" }

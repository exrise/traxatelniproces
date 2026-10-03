package ui

import (
	"svinovoyna/internal/ai"
	"svinovoyna/internal/balance"
	"svinovoyna/internal/sim"
)

// Slot describes one seat of a match.
type Slot struct {
	Name  string
	Human bool
	Skill float64 // bot skill
	Color int
	Team  int
}

// Session is the game-state provider the match scene talks to: a local game,
// a network host or a network client.
type Session interface {
	World() *sim.World
	Send(c sim.Command)
	Update()
	IsHuman(pid int) bool
	// Local reports whether the player is controlled from this machine.
	Local(pid int) bool
	// BuildSeat returns the local player who is currently allowed to build (-1 none).
	BuildSeat() int
	BuildSeatTime() float64
	Close()
	Status() string
	// PopError returns (and clears) the last rejected command message.
	PopError() string
}

// LocalSession runs the simulation in-process with bots and hotseat humans.
type LocalSession struct {
	W       *sim.World
	bots    map[int]*ai.Bot
	humans  []int
	seat    int
	seatT   float64
	seatNo  int // build number the seat state belongs to
	hotseat bool
	lastErr string
}

// NewLocalSession creates a match from lobby slots.
func NewLocalSession(cfg *balance.Config, seed uint64, slots []Slot) *LocalSession {
	var setups []sim.PlayerSetup
	for i, s := range slots {
		setups = append(setups, sim.PlayerSetup{Name: s.Name, Team: s.Team, Bot: !s.Human, Color: s.Color})
		_ = i
	}
	s := &LocalSession{W: sim.NewWorld(cfg, seed, setups), bots: map[int]*ai.Bot{}, seat: -1}
	for i, sl := range slots {
		if sl.Human {
			s.humans = append(s.humans, i)
		} else {
			s.bots[i] = ai.New(i, ai.Style(int(seed+uint64(i)*7)%int(ai.NumStyles)), sl.Skill, seed)
		}
	}
	s.hotseat = len(s.humans) > 1
	return s
}

func (s *LocalSession) World() *sim.World { return s.W }
func (s *LocalSession) Close()            {}
func (s *LocalSession) Status() string    { return "локальная игра" }

func (s *LocalSession) IsHuman(pid int) bool {
	for _, h := range s.humans {
		if h == pid {
			return true
		}
	}
	return false
}

func (s *LocalSession) Local(pid int) bool { return s.IsHuman(pid) }

func (s *LocalSession) BuildSeat() int { return s.seat }

func (s *LocalSession) BuildSeatTime() float64 {
	if s.hotseat {
		return s.seatT
	}
	return s.W.BuildTimer
}

// Send applies a command from a human.
func (s *LocalSession) Send(c sim.Command) {
	if !s.IsHuman(c.Player) {
		return
	}
	if err := s.W.Apply(c); err != nil {
		s.lastErr = err.Error()
	}
}

func (s *LocalSession) nextSeat() {
	s.seat = -1
	for _, h := range s.humans {
		if !s.W.Players[h].Ready {
			s.seat = h
			s.seatT = s.W.Cfg.BuildTime
			if s.W.BuildNo == 1 {
				s.seatT = s.W.Cfg.BuildTimeFirst
			}
			return
		}
	}
}

// Update advances one tick.
func (s *LocalSession) Update() {
	w := s.W
	switch w.Phase {
	case sim.PhaseBuild:
		if s.seatNo != w.BuildNo {
			s.seatNo = w.BuildNo
			for _, b := range s.bots {
				b.Build(w)
			}
			if s.hotseat {
				w.BuildTimerOn = false
			}
			s.nextSeat()
		}
		if s.hotseat {
			if s.seat >= 0 {
				if w.Players[s.seat].Ready {
					s.nextSeat()
				} else {
					s.seatT -= sim.Dt
					if s.seatT <= 0 {
						w.Players[s.seat].Ready = true
						s.nextSeat()
					}
				}
			}
		} else if len(s.humans) == 1 {
			s.seat = s.humans[0]
			if w.Players[s.seat].Ready {
				s.seat = -1
			}
		}
	case sim.PhaseBattle:
		s.seat = -1
		for _, b := range s.bots {
			for _, c := range b.Think(w) {
				_ = w.Apply(c)
			}
		}
	}
	w.Step()
}

// PopError implements Session.
func (s *LocalSession) PopError() string {
	e := s.lastErr
	s.lastErr = ""
	return e
}

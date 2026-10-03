package netplay

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type peer struct {
	conn net.Conn
	enc  *gob.Encoder
	dec  *gob.Decoder
	out  chan Msg
	seat int
	name string
	dead bool
	st   SendState
}

// InCmd is a command received from a remote seat.
type InCmd struct {
	Seat int
	Msg  Msg
}

// Host accepts clients, owns the lobby and relays messages during the game.
type Host struct {
	ln      net.Listener
	mu      sync.Mutex
	peers   map[int]*peer
	lobby   LobbyState
	started bool
	in      chan InCmd
	dropped []int
	closed  atomic.Bool
}

// Listen starts accepting connections on port.
func Listen(port string, lobby LobbyState) (*Host, error) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}
	h := &Host{ln: ln, peers: map[int]*peer{}, lobby: lobby, in: make(chan InCmd, 1024)}
	go h.acceptLoop()
	return h, nil
}

func (h *Host) acceptLoop() {
	for {
		c, err := h.ln.Accept()
		if err != nil {
			return
		}
		go h.handshake(c)
	}
}

func send(p *peer, m Msg) bool {
	select {
	case p.out <- m:
		return true
	default:
		return false
	}
}

func (h *Host) handshake(c net.Conn) {
	_ = c.SetReadDeadline(time.Now().Add(6 * time.Second))
	p := &peer{conn: c, enc: gob.NewEncoder(c), dec: gob.NewDecoder(c), out: make(chan Msg, 512), seat: -1}
	var hello Msg
	if err := p.dec.Decode(&hello); err != nil || hello.Kind != "hello" {
		c.Close()
		return
	}
	reject := func(reason string) {
		_ = c.SetWriteDeadline(time.Now().Add(3 * time.Second))
		_ = p.enc.Encode(Msg{Kind: "reject", Reason: reason})
		c.Close()
	}
	if hello.Version != Version {
		reject("версия игры не совпадает (у хоста " + Version + ")")
		return
	}
	h.mu.Lock()
	if h.started {
		h.mu.Unlock()
		reject("игра уже началась")
		return
	}
	seat := -1
	for i, s := range h.lobby.Slots {
		if s.Kind == SlotOpen {
			seat = i
			break
		}
	}
	if seat < 0 {
		h.mu.Unlock()
		reject("нет свободных мест")
		return
	}
	name := hello.Name
	if name == "" {
		name = fmt.Sprintf("Игрок %d", seat+1)
	}
	h.lobby.Slots[seat].Kind = SlotRemote
	h.lobby.Slots[seat].Name = uniqueName(h.lobby.Slots, name)
	name = h.lobby.Slots[seat].Name
	p.seat, p.name = seat, name
	h.peers[seat] = p
	lob := h.lobby.clone()
	h.mu.Unlock()

	_ = c.SetReadDeadline(time.Time{})
	go h.writer(p)
	go h.reader(p)
	send(p, Msg{Kind: "welcome", Seat: seat, Lobby: &lob})
	h.broadcastLobby()
}

func (l LobbyState) clone() LobbyState {
	n := l
	n.Slots = append([]LobbySlot(nil), l.Slots...)
	n.CfgJSON = append([]byte(nil), l.CfgJSON...)
	return n
}

func (h *Host) writer(p *peer) {
	for m := range p.out {
		_ = p.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := p.enc.Encode(m); err != nil {
			h.drop(p)
			return
		}
	}
}

func (h *Host) reader(p *peer) {
	for {
		var m Msg
		if err := p.dec.Decode(&m); err != nil {
			h.drop(p)
			return
		}
		if m.Kind == "cmd" && m.Cmd != nil {
			select {
			case h.in <- InCmd{Seat: p.seat, Msg: m}:
			default:
			}
		}
	}
}

func (h *Host) drop(p *peer) {
	h.mu.Lock()
	if p.dead {
		h.mu.Unlock()
		return
	}
	p.dead = true
	_ = p.conn.Close()
	delete(h.peers, p.seat)
	if !h.started && p.seat >= 0 && p.seat < len(h.lobby.Slots) {
		h.lobby.Slots[p.seat].Kind = SlotOpen
		h.lobby.Slots[p.seat].Name = ""
	}
	if h.started {
		h.dropped = append(h.dropped, p.seat)
	}
	h.mu.Unlock()
	if !h.closed.Load() {
		h.broadcastLobby()
	}
}

func (h *Host) broadcastLobby() {
	h.mu.Lock()
	lob := h.lobby.clone()
	var ps []*peer
	for _, p := range h.peers {
		ps = append(ps, p)
	}
	h.mu.Unlock()
	for _, p := range ps {
		send(p, Msg{Kind: "lobby", Lobby: &lob})
	}
}

// Lobby returns a copy of the current lobby.
func (h *Host) Lobby() LobbyState {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lobby.clone()
}

// Edit mutates the lobby under the lock and broadcasts the result.
func (h *Host) Edit(f func(l *LobbyState)) {
	h.mu.Lock()
	f(&h.lobby)
	h.mu.Unlock()
	h.broadcastLobby()
}

// Kick disconnects the player in a seat and reopens it.
func (h *Host) Kick(seat int) {
	h.mu.Lock()
	p := h.peers[seat]
	h.mu.Unlock()
	if p != nil {
		send(p, Msg{Kind: "reject", Reason: "хост отключил тебя"})
		time.AfterFunc(300*time.Millisecond, func() { h.drop(p) })
	}
}

// Start freezes the lobby and tells every client to begin.
func (h *Host) Start() ([]int, error) {
	h.mu.Lock()
	for _, s := range h.lobby.Slots {
		if s.Kind == SlotOpen {
			h.mu.Unlock()
			return nil, errors.New("есть пустые места: закрой их или поставь ботов")
		}
	}
	h.started = true
	lob := h.lobby.clone()
	var seats []int
	var ps []*peer
	for s, p := range h.peers {
		seats = append(seats, s)
		ps = append(ps, p)
	}
	h.mu.Unlock()
	for _, p := range ps {
		send(p, Msg{Kind: "start", Seat: p.seat, Lobby: &lob})
	}
	return seats, nil
}

// Poll returns the commands received since the last call.
func (h *Host) Poll() []InCmd {
	var out []InCmd
	for {
		select {
		case c := <-h.in:
			out = append(out, c)
		default:
			return out
		}
	}
}

// Dropped returns seats whose clients disconnected during the game.
func (h *Host) Dropped() []int {
	h.mu.Lock()
	d := h.dropped
	h.dropped = nil
	h.mu.Unlock()
	return d
}

// State returns the per-client send state for delta snapshots.
func (h *Host) State(seat int) *SendState {
	h.mu.Lock()
	defer h.mu.Unlock()
	if p := h.peers[seat]; p != nil {
		return &p.st
	}
	return nil
}

// SendSnap queues a snapshot for a seat; the connection is dropped if it cannot keep up.
func (h *Host) SendSnap(seat int, s *Snapshot) {
	h.mu.Lock()
	p := h.peers[seat]
	h.mu.Unlock()
	if p == nil {
		return
	}
	if !send(p, Msg{Kind: "snap", Snap: s}) {
		go h.drop(p)
	}
}

// Connected tells whether a seat still has a live client.
func (h *Host) Connected(seat int) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.peers[seat] != nil
}

// Close shuts everything down.
func (h *Host) Close() {
	h.closed.Store(true)
	_ = h.ln.Close()
	h.mu.Lock()
	for _, p := range h.peers {
		_ = p.conn.Close()
	}
	h.mu.Unlock()
}

// LocalIPs lists the host's IPv4 addresses (Radmin VPN addresses start with 26.).
func LocalIPs() []string {
	var out []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	for _, a := range addrs {
		if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil && !ipn.IP.IsLoopback() {
			out = append(out, ipn.IP.String())
		}
	}
	return out
}

// Addr returns the listening address (useful with port "0").
func (h *Host) Addr() net.Addr { return h.ln.Addr() }

// uniqueName appends a suffix when another seat already uses the name.
func uniqueName(slots []LobbySlot, name string) string {
	taken := func(n string) bool {
		for _, s := range slots {
			if s.Kind != SlotOpen && s.Name == n {
				return true
			}
		}
		return false
	}
	if !taken(name) {
		return name
	}
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s %d", name, i)
		if !taken(cand) {
			return cand
		}
	}
}

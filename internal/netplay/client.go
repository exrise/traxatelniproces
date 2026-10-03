package netplay

import (
	"encoding/gob"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"svinovoyna/internal/sim"
)

// Client is the connection of a remote player to a host.
type Client struct {
	conn net.Conn
	enc  *gob.Encoder
	dec  *gob.Decoder
	out  chan Msg
	in   chan Msg
	mu   sync.Mutex
	err  error

	Seat  int
	Lobby LobbyState
}

// Dial connects to host:port and performs the handshake.
func Dial(addr, name string) (*Client, error) {
	if !strings.Contains(addr, ":") {
		addr += ":" + DefaultPort
	}
	c, err := net.DialTimeout("tcp", addr, 6*time.Second)
	if err != nil {
		return nil, err
	}
	cl := &Client{conn: c, enc: gob.NewEncoder(c), dec: gob.NewDecoder(c), out: make(chan Msg, 512), in: make(chan Msg, 1024)}
	_ = c.SetDeadline(time.Now().Add(8 * time.Second))
	if err := cl.enc.Encode(Msg{Kind: "hello", Name: name, Version: Version}); err != nil {
		c.Close()
		return nil, err
	}
	var m Msg
	if err := cl.dec.Decode(&m); err != nil {
		c.Close()
		return nil, err
	}
	if m.Kind == "reject" {
		c.Close()
		return nil, errors.New(m.Reason)
	}
	if m.Kind != "welcome" || m.Lobby == nil {
		c.Close()
		return nil, errors.New("неожиданный ответ хоста")
	}
	_ = c.SetDeadline(time.Time{})
	cl.Seat = m.Seat
	cl.Lobby = *m.Lobby
	go cl.reader()
	go cl.writer()
	return cl, nil
}

func (c *Client) reader() {
	for {
		var m Msg
		if err := c.dec.Decode(&m); err != nil {
			c.mu.Lock()
			c.err = err
			c.mu.Unlock()
			close(c.in)
			return
		}
		c.in <- m
	}
}

func (c *Client) writer() {
	for m := range c.out {
		_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.enc.Encode(m); err != nil {
			c.mu.Lock()
			c.err = err
			c.mu.Unlock()
			return
		}
	}
}

// Poll returns all pending messages; the bool is false once the connection is gone.
func (c *Client) Poll() ([]Msg, bool) {
	var out []Msg
	for {
		select {
		case m, ok := <-c.in:
			if !ok {
				return out, false
			}
			out = append(out, m)
		default:
			return out, true
		}
	}
}

// SendCmd sends a game command.
func (c *Client) SendCmd(cmd sim.Command) {
	select {
	case c.out <- Msg{Kind: "cmd", Cmd: &cmd}:
	default:
	}
}

// Close drops the connection.
func (c *Client) Close() {
	_ = c.conn.Close()
}

package engineio

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Session represents an Engine.IO session
type Session struct {
	id            string
	conn          *websocket.Conn
	server        *Server
	outgoing      chan *Packet
	pingTimer     *time.Timer
	pingTimeout   *time.Timer
	closeOnce     sync.Once
	closed        chan struct{}
	mu            sync.RWMutex
	onMessage     func([]byte)
	onClose       func(string)
	lastActivity  time.Time
}

// NewSession creates a new Engine.IO session
func NewSession(id string, conn *websocket.Conn, server *Server) *Session {
	s := &Session{
		id:           id,
		conn:         conn,
		server:       server,
		outgoing:     make(chan *Packet, 256),
		closed:       make(chan struct{}),
		lastActivity: time.Now(),
	}

	return s
}

// ID returns the session ID
func (s *Session) ID() string {
	return s.id
}

// Start starts the session loops
func (s *Session) Start() {
	go s.writeLoop()
	go s.readLoop()
	s.schedulePing()
}

// Send sends a packet to the client
func (s *Session) Send(packet *Packet) error {
	select {
	case s.outgoing <- packet:
		return nil
	case <-s.closed:
		return ErrSessionClosed
	default:
		// Channel full, connection might be slow
		return ErrSlowClient
	}
}

// Close closes the session
func (s *Session) Close(reason string) {
	s.closeOnce.Do(func() {
		close(s.closed)

		if s.pingTimer != nil {
			s.pingTimer.Stop()
		}
		if s.pingTimeout != nil {
			s.pingTimeout.Stop()
		}

		// Send close packet
		packet := &Packet{Type: PacketTypeClose}
		s.conn.WriteMessage(websocket.TextMessage, packet.Encode())

		s.conn.Close()

		if s.onClose != nil {
			s.onClose(reason)
		}
	})
}

// OnMessage sets the message handler
func (s *Session) OnMessage(fn func([]byte)) {
	s.mu.Lock()
	s.onMessage = fn
	s.mu.Unlock()
}

// OnClose sets the close handler
func (s *Session) OnClose(fn func(string)) {
	s.mu.Lock()
	s.onClose = fn
	s.mu.Unlock()
}

func (s *Session) readLoop() {
	defer s.Close("read error")

	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			return
		}

		s.updateActivity()

		packet, err := DecodePacket(data)
		if err != nil {
			continue
		}

		s.handlePacket(packet)
	}
}

func (s *Session) writeLoop() {
	for {
		select {
		case packet := <-s.outgoing:
			if err := s.conn.WriteMessage(websocket.TextMessage, packet.Encode()); err != nil {
				s.Close("write error")
				return
			}
		case <-s.closed:
			return
		}
	}
}

func (s *Session) handlePacket(packet *Packet) {
	switch packet.Type {
	case PacketTypePing:
		s.handlePing()
	case PacketTypePong:
		s.handlePong()
	case PacketTypeMessage:
		s.handleMessage(packet.Data)
	case PacketTypeClose:
		s.Close("client closed")
	}
}

func (s *Session) handlePing() {
	s.Send(&Packet{Type: PacketTypePong})
}

func (s *Session) handlePong() {
	if s.pingTimeout != nil {
		s.pingTimeout.Stop()
	}
	s.schedulePing()
}

func (s *Session) handleMessage(data []byte) {
	s.mu.RLock()
	handler := s.onMessage
	s.mu.RUnlock()

	if handler != nil {
		handler(data)
	}
}

func (s *Session) schedulePing() {
	s.pingTimer = time.AfterFunc(time.Duration(s.server.config.PingInterval)*time.Millisecond, func() {
		s.Send(&Packet{Type: PacketTypePing})
		s.schedulePingTimeout()
	})
}

func (s *Session) schedulePingTimeout() {
	s.pingTimeout = time.AfterFunc(time.Duration(s.server.config.PingTimeout)*time.Millisecond, func() {
		s.Close("ping timeout")
	})
}

func (s *Session) updateActivity() {
	s.mu.Lock()
	s.lastActivity = time.Now()
	s.mu.Unlock()
}

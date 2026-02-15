package engineio

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	ErrSessionClosed = errors.New("session closed")
	ErrSlowClient    = errors.New("slow client")
)

// Config holds Engine.IO server configuration
type Config struct {
	PingInterval int // milliseconds
	PingTimeout  int // milliseconds
	MaxPayload   int // bytes
}

// DefaultConfig returns default Engine.IO configuration
func DefaultConfig() *Config {
	return &Config{
		PingInterval: 25000, // 25 seconds
		PingTimeout:  20000, // 20 seconds
		MaxPayload:   1e6,   // 1MB
	}
}

// Server represents an Engine.IO server
type Server struct {
	config    *Config
	upgrader  websocket.Upgrader
	sessions  sync.Map
	onConnect func(*Session)
}

// NewServer creates a new Engine.IO server
func NewServer(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	return &Server{
		config: config,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // TODO: Make configurable
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// ServeHTTP handles HTTP requests and upgrades to WebSocket
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only handle WebSocket upgrade
	if r.URL.Query().Get("transport") != "websocket" {
		http.Error(w, "Only WebSocket transport is supported", http.StatusBadRequest)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	sid := generateSID()
	session := NewSession(sid, conn, s)

	s.sessions.Store(sid, session)

	// Send handshake
	handshake, err := EncodeHandshake(sid, s.config.PingInterval, s.config.PingTimeout, s.config.MaxPayload)
	if err != nil {
		conn.Close()
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, handshake); err != nil {
		conn.Close()
		return
	}

	session.OnClose(func(reason string) {
		s.sessions.Delete(sid)
	})

	session.Start()

	if s.onConnect != nil {
		s.onConnect(session)
	}
}

// OnConnect sets the connection handler
func (s *Server) OnConnect(fn func(*Session)) {
	s.onConnect = fn
}

// GetSession retrieves a session by ID
func (s *Server) GetSession(sid string) (*Session, bool) {
	val, ok := s.sessions.Load(sid)
	if !ok {
		return nil, false
	}
	return val.(*Session), true
}

// Close closes all sessions
func (s *Server) Close() {
	s.sessions.Range(func(key, value interface{}) bool {
		session := value.(*Session)
		session.Close("server shutdown")
		return true
	})
}

func generateSID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

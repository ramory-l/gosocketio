package gosocketio

import (
	"net/http"
	"strings"
	"sync"

	"github.com/ramory-l/gosocketio/engineio"
)

// Server represents a Socket.IO server
type Server struct {
	eio        *engineio.Server
	namespaces map[string]*Namespace
	nsMu       sync.RWMutex
}

// Config represents Socket.IO server configuration
type Config struct {
	PingInterval int
	PingTimeout  int
	MaxPayload   int
}

// NewServer creates a new Socket.IO server
func NewServer(config *Config) *Server {
	var eioConfig *engineio.Config
	if config != nil {
		eioConfig = &engineio.Config{
			PingInterval: config.PingInterval,
			PingTimeout:  config.PingTimeout,
			MaxPayload:   config.MaxPayload,
		}
	}

	server := &Server{
		eio:        engineio.NewServer(eioConfig),
		namespaces: make(map[string]*Namespace),
	}

	// Create default namespace
	server.Of("/")

	// Handle Engine.IO connections
	server.eio.OnConnect(server.handleConnection)

	return server
}

// Of returns a namespace, creating it if it doesn't exist
func (s *Server) Of(name string) *Namespace {
	if name == "" {
		name = "/"
	}

	s.nsMu.RLock()
	ns, exists := s.namespaces[name]
	s.nsMu.RUnlock()

	if exists {
		return ns
	}

	s.nsMu.Lock()
	defer s.nsMu.Unlock()

	// Double-check after acquiring write lock
	if ns, exists := s.namespaces[name]; exists {
		return ns
	}

	ns = NewNamespace(name, s)
	s.namespaces[name] = ns

	return ns
}

// OnConnect sets the connection handler for the default namespace
func (s *Server) OnConnect(handler func(*Socket)) {
	s.Of("/").OnConnect(handler)
}

// Emit broadcasts to all clients in the default namespace
func (s *Server) Emit(event string, data ...interface{}) error {
	return s.Of("/").Emit(event, data...)
}

// To returns a BroadcastOperator for the default namespace
func (s *Server) To(rooms ...string) *BroadcastOperator {
	return s.Of("/").To(rooms...)
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse namespace from path
	path := r.URL.Path
	if !strings.HasPrefix(path, "/socket.io/") {
		http.NotFound(w, r)
		return
	}

	// Delegate to Engine.IO
	s.eio.ServeHTTP(w, r)
}

// Close closes the server and all connections
func (s *Server) Close() error {
	s.eio.Close()

	s.nsMu.RLock()
	defer s.nsMu.RUnlock()

	for _, ns := range s.namespaces {
		ns.adapter.Close()
	}

	return nil
}

func (s *Server) handleConnection(session *engineio.Session) {
	// Default to root namespace
	ns := s.Of("/")
	ns.addSocket(session)
}

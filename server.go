package gosocketio

import (
	"net/http"
	"strings"
	"sync"

	"github.com/ramory-l/gosocketio/engineio"
)

// Server represents a Socket.IO server instance.
//
// It manages multiple namespaces and handles client connections through
// the Engine.IO transport layer. The server implements http.Handler and
// can be mounted on any HTTP path.
//
// Example:
//
//	server := gosocketio.NewServer(nil)
//	http.Handle("/socket.io/", server)
//	http.ListenAndServe(":3000", nil)
type Server struct {
	eio        *engineio.Server
	namespaces map[string]*Namespace
	nsMu       sync.RWMutex
}

// Config represents Socket.IO server configuration options.
//
// All values are in milliseconds for time-based settings and bytes for size-based settings.
// If nil is passed to NewServer, default values will be used.
type Config struct {
	PingInterval int // Interval between ping packets in milliseconds (default: 25000)
	PingTimeout  int // Timeout for ping response in milliseconds (default: 20000)
	MaxPayload   int // Maximum payload size in bytes (default: 1000000)
}

// NewServer creates a new Socket.IO server with the given configuration.
//
// If config is nil, default configuration values will be used:
//   - PingInterval: 25000ms (25 seconds)
//   - PingTimeout: 20000ms (20 seconds)
//   - MaxPayload: 1000000 bytes (1MB)
//
// The server automatically creates a default namespace ("/") and is ready
// to accept connections immediately after creation.
//
// Example:
//
//	// Use default configuration
//	server := gosocketio.NewServer(nil)
//
//	// Custom configuration
//	config := &gosocketio.Config{
//	    PingInterval: 30000,
//	    PingTimeout: 25000,
//	    MaxPayload: 2000000,
//	}
//	server := gosocketio.NewServer(config)
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

// Of returns a namespace with the given name, creating it if it doesn't exist.
//
// Namespaces provide logical separation of concerns and allow you to multiplex
// a single connection. Each namespace has its own event handlers and rooms.
//
// The name should start with a "/" (e.g., "/admin", "/chat"). If an empty string
// is provided, it defaults to the root namespace "/".
//
// Example:
//
//	// Get default namespace
//	ns := server.Of("/")
//
//	// Get or create custom namespace
//	adminNs := server.Of("/admin")
//	adminNs.OnConnect(func(socket *gosocketio.Socket) {
//	    // Handle admin connections
//	})
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

// OnConnect sets the connection handler for the default namespace ("/").
//
// The handler is called whenever a client successfully connects to the server.
// This is a convenience method equivalent to calling server.Of("/").OnConnect(handler).
//
// Example:
//
//	server.OnConnect(func(socket *gosocketio.Socket) {
//	    log.Printf("Client connected: %s", socket.ID())
//
//	    socket.On("message", func(data ...interface{}) {
//	        // Handle message event
//	    })
//	})
func (s *Server) OnConnect(handler func(*Socket)) {
	s.Of("/").OnConnect(handler)
}

// Emit broadcasts an event to all connected clients in the default namespace.
//
// This is a convenience method equivalent to calling server.Of("/").Emit(event, data...).
//
// Example:
//
//	server.Emit("news", "Hello everyone!")
func (s *Server) Emit(event string, data ...interface{}) error {
	return s.Of("/").Emit(event, data...)
}

// To returns a BroadcastOperator for broadcasting to specific rooms in the default namespace.
//
// This is a convenience method equivalent to calling server.Of("/").To(rooms...).
//
// Example:
//
//	// Broadcast to specific rooms
//	server.To("room1", "room2").Emit("news", "Hello rooms!")
//
//	// Exclude specific sockets
//	server.To("room1").Except(socketID).Emit("news", "Hello others!")
func (s *Server) To(rooms ...string) *BroadcastOperator {
	return s.Of("/").To(rooms...)
}

// ServeHTTP implements http.Handler, allowing the server to be used with the standard library's HTTP server.
//
// The server expects to be mounted at "/socket.io/" path. Requests to other paths will return 404.
//
// Example:
//
//	server := gosocketio.NewServer(nil)
//	http.Handle("/socket.io/", server)
//	log.Fatal(http.ListenAndServe(":3000", nil))
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

// Close gracefully closes the server and all active connections.
//
// This method closes all namespaces and their associated resources.
// It is recommended to call this method during application shutdown.
//
// Example:
//
//	defer server.Close()
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

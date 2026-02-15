package gosocketio

import (
	"sync"

	"github.com/ramory-l/gosocketio/engineio"
)

// Namespace represents a Socket.IO namespace.
//
// Namespaces provide logical separation within a single Socket.IO server.
// Each namespace has its own event handlers, rooms, and connected sockets.
// The default namespace is "/" and is created automatically.
//
// Namespaces are useful for:
//   - Separating application logic (e.g., /chat, /admin, /notifications)
//   - Multi-tenancy (e.g., /tenant1, /tenant2)
//   - Different authorization levels
type Namespace struct {
	name      string
	server    *Server
	adapter   Adapter
	sockets   map[string]*Socket
	mu        sync.RWMutex
	onConnect func(*Socket)
}

// NewNamespace creates a new namespace
func NewNamespace(name string, server *Server) *Namespace {
	ns := &Namespace{
		name:    name,
		server:  server,
		sockets: make(map[string]*Socket),
	}

	ns.adapter = NewMemoryAdapter(ns)

	return ns
}

// Name returns the name of the namespace (e.g., "/", "/admin", "/chat").
func (ns *Namespace) Name() string {
	return ns.name
}

// OnConnect sets the connection handler for this namespace.
//
// The handler is called whenever a client successfully connects to this namespace.
//
// Example:
//
//	adminNs := server.Of("/admin")
//	adminNs.OnConnect(func(socket *gosocketio.Socket) {
//	    log.Printf("Admin connected: %s", socket.ID())
//	})
func (ns *Namespace) OnConnect(handler func(*Socket)) {
	ns.onConnect = handler
}

// To returns a BroadcastOperator for broadcasting to specific rooms in this namespace.
//
// Example:
//
//	ns.To("room1", "room2").Emit("news", "Hello rooms!")
func (ns *Namespace) To(rooms ...string) *BroadcastOperator {
	return &BroadcastOperator{
		namespace: ns,
		rooms:     rooms,
	}
}

// Emit broadcasts an event to all connected sockets in this namespace.
//
// Example:
//
//	ns.Emit("announcement", "Hello everyone in this namespace!")
func (ns *Namespace) Emit(event string, data ...interface{}) error {
	return ns.To().Emit(event, data...)
}

// Sockets returns all currently connected sockets in this namespace.
//
// Example:
//
//	for _, socket := range ns.Sockets() {
//	    log.Printf("Socket: %s", socket.ID())
//	}
func (ns *Namespace) Sockets() []*Socket {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	sockets := make([]*Socket, 0, len(ns.sockets))
	for _, socket := range ns.sockets {
		sockets = append(sockets, socket)
	}
	return sockets
}

// GetSocket retrieves a socket by its ID.
//
// Returns the socket and true if found, or nil and false if not.
//
// Example:
//
//	if socket, ok := ns.GetSocket("abc123"); ok {
//	    socket.Emit("private", "Hello!")
//	}
func (ns *Namespace) GetSocket(id string) (*Socket, bool) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	socket, ok := ns.sockets[id]
	return socket, ok
}

// SetAdapter sets a custom adapter for this namespace.
//
// Adapters manage rooms and broadcasting. The default is MemoryAdapter,
// which stores rooms in memory. For multi-server deployments, use a
// distributed adapter like RedisAdapter.
//
// Example:
//
//	ns.SetAdapter(NewRedisAdapter(redisClient))
func (ns *Namespace) SetAdapter(adapter Adapter) {
	ns.adapter = adapter
}

func (ns *Namespace) addSocket(session *engineio.Session) {
	socket := NewSocket(session.ID(), session, ns)

	ns.mu.Lock()
	ns.sockets[socket.ID()] = socket
	ns.mu.Unlock()

	// Auto-join own room
	socket.Join(socket.ID())

	// Send connect packet
	connectPacket := &Packet{
		Type:      PacketTypeConnect,
		Namespace: ns.name,
		Data:      map[string]interface{}{"sid": socket.ID()},
	}
	socket.sendPacket(connectPacket)

	if ns.onConnect != nil {
		ns.onConnect(socket)
	}
}

func (ns *Namespace) removeSocket(id string) {
	ns.mu.Lock()
	delete(ns.sockets, id)
	ns.mu.Unlock()

	ns.adapter.RemoveAll(id)
}

// BroadcastOperator provides a fluent interface for broadcasting events to specific rooms.
//
// It allows chaining methods to build complex broadcast targets:
//   - To() - add rooms to broadcast to
//   - Except() - exclude specific sockets from the broadcast
//   - Emit() - send the event
//
// Example:
//
//	server.To("room1", "room2").Except(socket.ID()).Emit("news", "Hello!")
type BroadcastOperator struct {
	namespace *Namespace
	rooms     []string
	except    []string
}

// To adds additional rooms to broadcast to.
//
// This method can be chained multiple times to add more rooms.
//
// Example:
//
//	server.To("room1").To("room2").Emit("news", "Hello!")
func (b *BroadcastOperator) To(rooms ...string) *BroadcastOperator {
	b.rooms = append(b.rooms, rooms...)
	return b
}

// Except excludes specific socket IDs from receiving the broadcast.
//
// This is useful for broadcasting to a room while excluding the sender.
//
// Example:
//
//	// Broadcast to all in room1 except the sender
//	server.To("room1").Except(socket.ID()).Emit("message", msg)
func (b *BroadcastOperator) Except(socketIDs ...string) *BroadcastOperator {
	b.except = append(b.except, socketIDs...)
	return b
}

// Emit broadcasts an event to all targeted sockets.
//
// If no rooms were specified with To(), broadcasts to all sockets in the namespace.
// Sockets specified in Except() will not receive the event.
//
// Example:
//
//	server.To("room1").Except(socket.ID()).Emit("message", "Hello room!")
func (b *BroadcastOperator) Emit(event string, data ...interface{}) error {
	args := make([]interface{}, 0, len(data)+1)
	args = append(args, event)
	args = append(args, data...)

	packet := &Packet{
		Type:      PacketTypeEvent,
		Namespace: b.namespace.name,
		Data:      args,
	}

	return b.namespace.adapter.Broadcast(packet, b.rooms, b.except)
}

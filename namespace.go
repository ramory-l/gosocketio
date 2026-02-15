package gosocketio

import (
	"sync"

	"github.com/ramory-l/gosocketio/engineio"
)

// Namespace represents a Socket.IO namespace
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

// Name returns the namespace name
func (ns *Namespace) Name() string {
	return ns.name
}

// OnConnect sets the connection handler for this namespace
func (ns *Namespace) OnConnect(handler func(*Socket)) {
	ns.onConnect = handler
}

// To returns a BroadcastOperator for emitting to specific rooms
func (ns *Namespace) To(rooms ...string) *BroadcastOperator {
	return &BroadcastOperator{
		namespace: ns,
		rooms:     rooms,
	}
}

// Emit broadcasts an event to all sockets in the namespace
func (ns *Namespace) Emit(event string, data ...interface{}) error {
	return ns.To().Emit(event, data...)
}

// Sockets returns all connected sockets
func (ns *Namespace) Sockets() []*Socket {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	sockets := make([]*Socket, 0, len(ns.sockets))
	for _, socket := range ns.sockets {
		sockets = append(sockets, socket)
	}
	return sockets
}

// GetSocket retrieves a socket by ID
func (ns *Namespace) GetSocket(id string) (*Socket, bool) {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	socket, ok := ns.sockets[id]
	return socket, ok
}

// SetAdapter sets a custom adapter
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

// BroadcastOperator provides methods for broadcasting to specific rooms
type BroadcastOperator struct {
	namespace *Namespace
	rooms     []string
	except    []string
}

// To adds rooms to broadcast to
func (b *BroadcastOperator) To(rooms ...string) *BroadcastOperator {
	b.rooms = append(b.rooms, rooms...)
	return b
}

// Except excludes specific socket IDs from the broadcast
func (b *BroadcastOperator) Except(socketIDs ...string) *BroadcastOperator {
	b.except = append(b.except, socketIDs...)
	return b
}

// Emit broadcasts an event
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

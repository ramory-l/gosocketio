package gosocketio

import (
	"sync"

	"github.com/ramory-l/gosocketio/engineio"
)

// MemoryAdapter is an in-memory implementation of the Adapter interface
type MemoryAdapter struct {
	rooms       map[string]map[string]bool // room -> socketIDs
	socketRooms map[string]map[string]bool // socketID -> rooms
	mu          sync.RWMutex
	namespace   *Namespace
}

// NewMemoryAdapter creates a new in-memory adapter
func NewMemoryAdapter(namespace *Namespace) *MemoryAdapter {
	return &MemoryAdapter{
		rooms:       make(map[string]map[string]bool),
		socketRooms: make(map[string]map[string]bool),
		namespace:   namespace,
	}
}

// Add adds a socket to a room
func (a *MemoryAdapter) Add(socketID, room string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.rooms[room] == nil {
		a.rooms[room] = make(map[string]bool)
	}
	a.rooms[room][socketID] = true

	if a.socketRooms[socketID] == nil {
		a.socketRooms[socketID] = make(map[string]bool)
	}
	a.socketRooms[socketID][room] = true
}

// Remove removes a socket from a room
func (a *MemoryAdapter) Remove(socketID, room string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.rooms[room] != nil {
		delete(a.rooms[room], socketID)
		if len(a.rooms[room]) == 0 {
			delete(a.rooms, room)
		}
	}

	if a.socketRooms[socketID] != nil {
		delete(a.socketRooms[socketID], room)
		if len(a.socketRooms[socketID]) == 0 {
			delete(a.socketRooms, socketID)
		}
	}
}

// RemoveAll removes a socket from all rooms
func (a *MemoryAdapter) RemoveAll(socketID string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	rooms := a.socketRooms[socketID]
	for room := range rooms {
		if a.rooms[room] != nil {
			delete(a.rooms[room], socketID)
			if len(a.rooms[room]) == 0 {
				delete(a.rooms, room)
			}
		}
	}

	delete(a.socketRooms, socketID)
}

// Sockets returns all socket IDs in a room
func (a *MemoryAdapter) Sockets(room string) []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sockets := a.rooms[room]
	result := make([]string, 0, len(sockets))
	for socketID := range sockets {
		result = append(result, socketID)
	}
	return result
}

// SocketRooms returns all rooms a socket is in
func (a *MemoryAdapter) SocketRooms(socketID string) []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	rooms := a.socketRooms[socketID]
	result := make([]string, 0, len(rooms))
	for room := range rooms {
		result = append(result, room)
	}
	return result
}

// Broadcast sends a packet to all sockets in specified rooms except excluded ones
func (a *MemoryAdapter) Broadcast(packet *Packet, rooms []string, except []string) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Build exclusion map
	excludeMap := make(map[string]bool)
	for _, sid := range except {
		excludeMap[sid] = true
	}

	// Collect target socket IDs
	targets := make(map[string]bool)

	if len(rooms) == 0 {
		// Broadcast to all sockets in namespace
		a.namespace.mu.RLock()
		for sid := range a.namespace.sockets {
			if !excludeMap[sid] {
				targets[sid] = true
			}
		}
		a.namespace.mu.RUnlock()
	} else {
		// Broadcast to specific rooms
		for _, room := range rooms {
			for socketID := range a.rooms[room] {
				if !excludeMap[socketID] {
					targets[socketID] = true
				}
			}
		}
	}

	// Send to target sockets
	encoded, err := packet.Encode()
	if err != nil {
		return err
	}

	a.namespace.mu.RLock()
	defer a.namespace.mu.RUnlock()

	for socketID := range targets {
		if socket, ok := a.namespace.sockets[socketID]; ok {
			// Non-blocking send
			go socket.session.Send(&engineio.Packet{
				Type: engineio.PacketTypeMessage,
				Data: []byte(encoded),
			})
		}
	}

	return nil
}

// Close cleans up the adapter
func (a *MemoryAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.rooms = make(map[string]map[string]bool)
	a.socketRooms = make(map[string]map[string]bool)

	return nil
}

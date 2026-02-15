package gosocketio

// Adapter is the interface for managing rooms and broadcasting
type Adapter interface {
	// Add adds a socket to a room
	Add(socketID, room string)

	// Remove removes a socket from a room
	Remove(socketID, room string)

	// RemoveAll removes a socket from all rooms
	RemoveAll(socketID string)

	// Sockets returns all socket IDs in a room
	Sockets(room string) []string

	// SocketRooms returns all rooms a socket is in
	SocketRooms(socketID string) []string

	// Broadcast sends a packet to all sockets in specified rooms except excluded ones
	Broadcast(packet *Packet, rooms []string, except []string) error

	// Close cleans up the adapter
	Close() error
}

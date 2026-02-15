package gosocketio

// Adapter is the interface for managing rooms and broadcasting in Socket.IO.
//
// Adapters are responsible for:
//   - Managing which sockets are in which rooms
//   - Broadcasting packets to sockets in specific rooms
//   - Cleaning up resources on shutdown
//
// The default implementation is MemoryAdapter, which stores everything in memory.
// For multi-server deployments, custom adapters can be implemented (e.g., RedisAdapter)
// to synchronize rooms and broadcasts across multiple server instances.
//
// Example custom adapter:
//
//	type RedisAdapter struct {
//	    // ...
//	}
//
//	func (a *RedisAdapter) Add(socketID, room string) {
//	    // Store in Redis
//	}
//
//	// Implement other methods...
//
//	ns.SetAdapter(NewRedisAdapter(redisClient))
type Adapter interface {
	// Add adds a socket to a room.
	//
	// A socket can be in multiple rooms simultaneously.
	Add(socketID, room string)

	// Remove removes a socket from a specific room.
	//
	// If the socket is not in the room, this is a no-op.
	Remove(socketID, room string)

	// RemoveAll removes a socket from all rooms it has joined.
	//
	// This is typically called when a socket disconnects.
	RemoveAll(socketID string)

	// Sockets returns all socket IDs currently in the specified room.
	//
	// Returns an empty slice if the room doesn't exist or has no members.
	Sockets(room string) []string

	// SocketRooms returns all rooms that the specified socket has joined.
	//
	// Returns an empty slice if the socket hasn't joined any rooms.
	SocketRooms(socketID string) []string

	// Broadcast sends a packet to all sockets in the specified rooms,
	// except those listed in the except parameter.
	//
	// If rooms is empty or nil, broadcasts to all sockets in the namespace.
	// The except parameter allows excluding specific sockets from the broadcast.
	Broadcast(packet *Packet, rooms []string, except []string) error

	// Close cleans up any resources used by the adapter.
	//
	// This is called when the namespace is being shut down.
	Close() error
}

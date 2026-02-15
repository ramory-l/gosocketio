package gosocketio

import (
	"sync"
	"sync/atomic"

	"github.com/ramory-l/gosocketio/engineio"
)

// Socket represents a client connection
type Socket struct {
	id         string
	session    *engineio.Session
	namespace  *Namespace
	rooms      map[string]bool
	roomsMu    sync.RWMutex
	handlers   map[string][]EventHandler
	handlersMu sync.RWMutex
	ackID      atomic.Int64
	ackHandlers sync.Map
	data       sync.Map
	onDisconnect []func(string)
	disconnectMu sync.RWMutex
}

// EventHandler handles Socket.IO events
type EventHandler func(...interface{})

// AckHandler handles acknowledgment responses
type AckHandler func(...interface{})

// NewSocket creates a new socket
func NewSocket(id string, session *engineio.Session, namespace *Namespace) *Socket {
	socket := &Socket{
		id:        id,
		session:   session,
		namespace: namespace,
		rooms:     make(map[string]bool),
		handlers:  make(map[string][]EventHandler),
	}

	session.OnMessage(socket.handleMessage)
	session.OnClose(socket.handleClose)

	return socket
}

// ID returns the socket ID
func (s *Socket) ID() string {
	return s.id
}

// Emit sends an event to the client
func (s *Socket) Emit(event string, data ...interface{}) error {
	args := make([]interface{}, 0, len(data)+1)
	args = append(args, event)
	args = append(args, data...)

	packet := &Packet{
		Type:      PacketTypeEvent,
		Namespace: s.namespace.name,
		Data:      args,
	}

	return s.sendPacket(packet)
}

// EmitWithAck sends an event and expects an acknowledgment
func (s *Socket) EmitWithAck(event string, ack AckHandler, data ...interface{}) error {
	args := make([]interface{}, 0, len(data)+1)
	args = append(args, event)
	args = append(args, data...)

	id := int(s.ackID.Add(1))

	packet := &Packet{
		Type:      PacketTypeEvent,
		Namespace: s.namespace.name,
		Data:      args,
		ID:        &id,
	}

	s.ackHandlers.Store(id, ack)

	return s.sendPacket(packet)
}

// On registers an event handler
func (s *Socket) On(event string, handler EventHandler) {
	s.handlersMu.Lock()
	s.handlers[event] = append(s.handlers[event], handler)
	s.handlersMu.Unlock()
}

// Off removes event handlers
func (s *Socket) Off(event string) {
	s.handlersMu.Lock()
	delete(s.handlers, event)
	s.handlersMu.Unlock()
}

// Join adds the socket to a room
func (s *Socket) Join(room string) {
	s.roomsMu.Lock()
	s.rooms[room] = true
	s.roomsMu.Unlock()

	s.namespace.adapter.Add(s.id, room)
}

// Leave removes the socket from a room
func (s *Socket) Leave(room string) {
	s.roomsMu.Lock()
	delete(s.rooms, room)
	s.roomsMu.Unlock()

	s.namespace.adapter.Remove(s.id, room)
}

// Rooms returns all rooms the socket is in
func (s *Socket) Rooms() []string {
	s.roomsMu.RLock()
	defer s.roomsMu.RUnlock()

	rooms := make([]string, 0, len(s.rooms))
	for room := range s.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

// Set stores arbitrary data on the socket
func (s *Socket) Set(key string, value interface{}) {
	s.data.Store(key, value)
}

// Get retrieves data from the socket
func (s *Socket) Get(key string) (interface{}, bool) {
	return s.data.Load(key)
}

// OnDisconnect registers a disconnect handler
func (s *Socket) OnDisconnect(handler func(string)) {
	s.disconnectMu.Lock()
	s.onDisconnect = append(s.onDisconnect, handler)
	s.disconnectMu.Unlock()
}

// Disconnect disconnects the socket
func (s *Socket) Disconnect() {
	s.session.Close("server disconnect")
}

func (s *Socket) sendPacket(packet *Packet) error {
	encoded, err := packet.Encode()
	if err != nil {
		return err
	}

	return s.session.Send(&engineio.Packet{
		Type: engineio.PacketTypeMessage,
		Data: []byte(encoded),
	})
}

func (s *Socket) handleMessage(data []byte) {
	packet, err := DecodePacket(string(data))
	if err != nil {
		return
	}

	switch packet.Type {
	case PacketTypeEvent:
		s.handleEvent(packet)
	case PacketTypeAck:
		s.handleAck(packet)
	case PacketTypeDisconnect:
		s.Disconnect()
	}
}

func (s *Socket) handleEvent(packet *Packet) {
	dataArray, ok := packet.Data.([]interface{})
	if !ok || len(dataArray) == 0 {
		return
	}

	event, ok := dataArray[0].(string)
	if !ok {
		return
	}

	args := dataArray[1:]

	// Handle acknowledgment
	if packet.ID != nil {
		ackFunc := func(ackData ...interface{}) {
			ackPacket := &Packet{
				Type:      PacketTypeAck,
				Namespace: s.namespace.name,
				Data:      ackData,
				ID:        packet.ID,
			}
			s.sendPacket(ackPacket)
		}
		args = append(args, ackFunc)
	}

	s.handlersMu.RLock()
	handlers := s.handlers[event]
	s.handlersMu.RUnlock()

	for _, handler := range handlers {
		go handler(args...)
	}
}

func (s *Socket) handleAck(packet *Packet) {
	if packet.ID == nil {
		return
	}

	val, ok := s.ackHandlers.LoadAndDelete(*packet.ID)
	if !ok {
		return
	}

	handler := val.(AckHandler)

	var args []interface{}
	if dataArray, ok := packet.Data.([]interface{}); ok {
		args = dataArray
	}

	go handler(args...)
}

func (s *Socket) handleClose(reason string) {
	// Leave all rooms
	s.roomsMu.RLock()
	rooms := make([]string, 0, len(s.rooms))
	for room := range s.rooms {
		rooms = append(rooms, room)
	}
	s.roomsMu.RUnlock()

	for _, room := range rooms {
		s.Leave(room)
	}

	// Call disconnect handlers
	s.disconnectMu.RLock()
	handlers := s.onDisconnect
	s.disconnectMu.RUnlock()

	for _, handler := range handlers {
		go handler(reason)
	}

	// Remove from namespace
	s.namespace.removeSocket(s.id)
}

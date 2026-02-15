package engineio

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// PacketType represents Engine.IO packet types
type PacketType byte

const (
	PacketTypeOpen PacketType = iota
	PacketTypeClose
	PacketTypePing
	PacketTypePong
	PacketTypeMessage
	PacketTypeUpgrade
	PacketTypeNoop
)

// Packet represents an Engine.IO packet
type Packet struct {
	Type PacketType
	Data []byte
}

// Encode encodes the packet to bytes
func (p *Packet) Encode() []byte {
	result := make([]byte, 0, len(p.Data)+1)
	result = append(result, byte('0'+p.Type))
	result = append(result, p.Data...)
	return result
}

// DecodePacket decodes bytes into a packet
func DecodePacket(data []byte) (*Packet, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty packet")
	}

	typeChar := data[0]
	if typeChar < '0' || typeChar > '6' {
		return nil, fmt.Errorf("invalid packet type: %c", typeChar)
	}

	packet := &Packet{
		Type: PacketType(typeChar - '0'),
	}

	if len(data) > 1 {
		packet.Data = data[1:]
	}

	return packet, nil
}

// HandshakeData represents the Engine.IO handshake response
type HandshakeData struct {
	SID          string   `json:"sid"`
	Upgrades     []string `json:"upgrades"`
	PingInterval int      `json:"pingInterval"`
	PingTimeout  int      `json:"pingTimeout"`
	MaxPayload   int      `json:"maxPayload"`
}

// EncodeHandshake creates an open packet with handshake data
func EncodeHandshake(sid string, pingInterval, pingTimeout, maxPayload int) ([]byte, error) {
	data := HandshakeData{
		SID:          sid,
		Upgrades:     []string{}, // No upgrades for WebSocket-only
		PingInterval: pingInterval,
		PingTimeout:  pingTimeout,
		MaxPayload:   maxPayload,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	packet := &Packet{
		Type: PacketTypeOpen,
		Data: jsonData,
	}

	return packet.Encode(), nil
}

// String returns the packet type as a string
func (pt PacketType) String() string {
	switch pt {
	case PacketTypeOpen:
		return "open"
	case PacketTypeClose:
		return "close"
	case PacketTypePing:
		return "ping"
	case PacketTypePong:
		return "pong"
	case PacketTypeMessage:
		return "message"
	case PacketTypeUpgrade:
		return "upgrade"
	case PacketTypeNoop:
		return "noop"
	default:
		return "unknown(" + strconv.Itoa(int(pt)) + ")"
	}
}

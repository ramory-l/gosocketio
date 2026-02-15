package gosocketio

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// PacketType represents Socket.IO packet types
type PacketType int

const (
	PacketTypeConnect PacketType = iota
	PacketTypeDisconnect
	PacketTypeEvent
	PacketTypeAck
	PacketTypeConnectError
	PacketTypeBinaryEvent
	PacketTypeBinaryAck
)

// Packet represents a Socket.IO packet
type Packet struct {
	Type      PacketType
	Namespace string
	Data      interface{}
	ID        *int
}

// Encode encodes a Socket.IO packet to string
func (p *Packet) Encode() (string, error) {
	var builder strings.Builder

	// Packet type
	builder.WriteString(strconv.Itoa(int(p.Type)))

	// Namespace (if not default)
	if p.Namespace != "" && p.Namespace != "/" {
		builder.WriteString(p.Namespace)
		builder.WriteByte(',')
	}

	// Ack ID
	if p.ID != nil {
		builder.WriteString(strconv.Itoa(*p.ID))
	}

	// Data
	if p.Data != nil {
		jsonData, err := json.Marshal(p.Data)
		if err != nil {
			return "", fmt.Errorf("failed to marshal packet data: %w", err)
		}
		builder.Write(jsonData)
	}

	return builder.String(), nil
}

// DecodePacket decodes a Socket.IO packet from string
func DecodePacket(data string) (*Packet, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty packet")
	}

	packet := &Packet{
		Namespace: "/",
	}

	pos := 0

	// Parse packet type
	if data[pos] < '0' || data[pos] > '6' {
		return nil, fmt.Errorf("invalid packet type: %c", data[pos])
	}
	packet.Type = PacketType(data[pos] - '0')
	pos++

	if pos >= len(data) {
		return packet, nil
	}

	// Parse namespace
	if data[pos] == '/' {
		end := strings.IndexByte(data[pos:], ',')
		if end == -1 {
			// Namespace without data
			packet.Namespace = data[pos:]
			return packet, nil
		}
		packet.Namespace = data[pos : pos+end]
		pos += end + 1
	}

	if pos >= len(data) {
		return packet, nil
	}

	// Parse ack ID
	if data[pos] >= '0' && data[pos] <= '9' {
		end := pos
		for end < len(data) && data[end] >= '0' && data[end] <= '9' {
			end++
		}
		id, _ := strconv.Atoi(data[pos:end])
		packet.ID = &id
		pos = end
	}

	if pos >= len(data) {
		return packet, nil
	}

	// Parse data
	if pos < len(data) {
		if err := json.Unmarshal([]byte(data[pos:]), &packet.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal packet data: %w", err)
		}
	}

	return packet, nil
}

// String returns the packet type as a string
func (pt PacketType) String() string {
	switch pt {
	case PacketTypeConnect:
		return "connect"
	case PacketTypeDisconnect:
		return "disconnect"
	case PacketTypeEvent:
		return "event"
	case PacketTypeAck:
		return "ack"
	case PacketTypeConnectError:
		return "connect_error"
	case PacketTypeBinaryEvent:
		return "binary_event"
	case PacketTypeBinaryAck:
		return "binary_ack"
	default:
		return "unknown"
	}
}

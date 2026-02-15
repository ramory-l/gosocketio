// Package gosocketio provides a Socket.IO v4 server implementation in Go.
//
// This library implements the Socket.IO v4 protocol with WebSocket transport only,
// optimized for high-concurrency applications such as chat aggregation platforms.
// It is designed to handle tens of thousands of concurrent connections efficiently.
//
// # Features
//
//   - Socket.IO v4 protocol support
//   - WebSocket transport only (Engine.IO v4)
//   - Namespaces for logical separation
//   - Rooms for grouping connections
//   - Event acknowledgments
//   - Binary data support
//   - Efficient broadcasting
//   - Compatible with official Socket.IO clients
//
// # Quick Start
//
//	server := gosocketio.NewServer(nil)
//
//	server.OnConnect(func(socket *gosocketio.Socket) {
//	    log.Printf("Client connected: %s", socket.ID())
//
//	    socket.On("message", func(data ...interface{}) {
//	        log.Printf("Received: %v", data)
//	        socket.Emit("response", "Message received!")
//	    })
//
//	    socket.OnDisconnect(func(reason string) {
//	        log.Printf("Client disconnected: %s", reason)
//	    })
//	})
//
//	http.Handle("/socket.io/", server)
//	http.ListenAndServe(":3000", nil)
//
// # Namespaces
//
// Namespaces provide logical separation of concerns. Each namespace has its own
// event handlers and rooms.
//
//	// Default namespace "/"
//	server.OnConnect(func(socket *gosocketio.Socket) {
//	    // Handle connection
//	})
//
//	// Custom namespace
//	adminNs := server.Of("/admin")
//	adminNs.OnConnect(func(socket *gosocketio.Socket) {
//	    // Handle admin connection
//	})
//
// # Rooms
//
// Rooms allow you to group sockets for targeted broadcasting.
//
//	socket.Join("room1")
//	server.To("room1").Emit("news", "Hello room!")
//	socket.Leave("room1")
//
// # Event Acknowledgments
//
// Request acknowledgments from clients:
//
//	socket.EmitWithAck("question", func(response ...interface{}) {
//	    log.Printf("Client answered: %v", response)
//	}, "What's your name?")
//
// Handle acknowledgment requests from clients:
//
//	socket.On("ping", func(data ...interface{}) {
//	    // Last argument is the ack function if client requested acknowledgment
//	    if len(data) > 0 {
//	        if ackFn, ok := data[len(data)-1].(func(...interface{})); ok {
//	            ackFn("pong")
//	        }
//	    }
//	})
//
// # Broadcasting
//
// Broadcast to all clients or specific rooms:
//
//	// To all clients in default namespace
//	server.Emit("broadcast", "Hello everyone!")
//
//	// To specific rooms
//	server.To("room1", "room2").Emit("news", "Hello rooms!")
//
//	// Exclude specific sockets
//	server.To("room1").Except(socket.ID()).Emit("news", "Hello others!")
//
// # Configuration
//
// Customize server behavior with Config:
//
//	config := &gosocketio.Config{
//	    PingInterval: 25000, // 25 seconds
//	    PingTimeout:  20000, // 20 seconds
//	    MaxPayload:   1000000, // 1MB
//	}
//	server := gosocketio.NewServer(config)
//
// # Thread Safety
//
// All operations are goroutine-safe. Event handlers are called in separate
// goroutines, allowing concurrent processing of events.
package gosocketio

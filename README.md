# gosocketio

A comprehensive Socket.IO v4 server implementation in Go, following Go idioms and optimized for high-concurrency applications.

## Features

- ✅ Socket.IO v4 protocol support
- ✅ WebSocket transport only (Engine.IO v4)
- ✅ Namespaces
- ✅ Rooms
- ✅ Event acknowledgments
- ✅ Binary data support
- ✅ Efficient broadcasting for high-concurrency scenarios
- ✅ Compatible with official Socket.IO clients

## Installation

```bash
go get github.com/ramory-l/gosocketio
```

## Quick Start

```go
package main

import (
    "log"
    "net/http"

    sio "github.com/ramory-l/gosocketio"
)

func main() {
    server := sio.NewServer(nil)

    server.OnConnect(func(socket *sio.Socket) {
        log.Printf("Client connected: %s", socket.ID())

        socket.On("message", func(data ...interface{}) {
            log.Printf("Received: %v", data)
            socket.Emit("response", "Message received!")
        })

        socket.OnDisconnect(func(reason string) {
            log.Printf("Client disconnected: %s", reason)
        })
    })

    http.Handle("/socket.io/", server)
    log.Fatal(http.ListenAndServe(":3000", nil))
}
```

## Architecture

Built for chat aggregation platforms handling tens of thousands of concurrent connections with:
- Efficient room-based broadcasting
- Minimal memory allocations
- Goroutine-safe operations
- Clean separation between Engine.IO and Socket.IO layers

## License

MIT

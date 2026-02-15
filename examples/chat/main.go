package main

import (
	"log"
	"net/http"

	sio "github.com/ramory-l/gosocketio"
)

func main() {
	// Create Socket.IO server
	server := sio.NewServer(nil)

	// Handle connections
	server.OnConnect(func(socket *sio.Socket) {
		log.Printf("Client connected: %s", socket.ID())

		// Handle join room event
		socket.On("join", func(data ...interface{}) {
			if len(data) > 0 {
				if room, ok := data[0].(string); ok {
					socket.Join(room)
					log.Printf("Socket %s joined room: %s", socket.ID(), room)

					// Notify room
					server.To(room).Except(socket.ID()).Emit("user_joined", map[string]interface{}{
						"socketId": socket.ID(),
						"room":     room,
					})

					// Send current room members count
					members := server.Of("/").To(room).Emit("room_info", map[string]interface{}{
						"room": room,
					})
					_ = members
				}
			}
		})

		// Handle leave room event
		socket.On("leave", func(data ...interface{}) {
			if len(data) > 0 {
				if room, ok := data[0].(string); ok {
					socket.Leave(room)
					log.Printf("Socket %s left room: %s", socket.ID(), room)

					// Notify room
					server.To(room).Emit("user_left", map[string]interface{}{
						"socketId": socket.ID(),
						"room":     room,
					})
				}
			}
		})

		// Handle chat message
		socket.On("message", func(data ...interface{}) {
			if len(data) > 0 {
				message := data[0]
				log.Printf("Message from %s: %v", socket.ID(), message)

				// Get rooms the socket is in
				rooms := socket.Rooms()

				// Broadcast to all rooms this socket is in
				for _, room := range rooms {
					// Don't broadcast to own room (socket ID room)
					if room != socket.ID() {
						server.To(room).Emit("message", map[string]interface{}{
							"from":    socket.ID(),
							"message": message,
							"room":    room,
						})
					}
				}
			}
		})

		// Handle message with acknowledgment
		socket.On("message_ack", func(data ...interface{}) {
			if len(data) < 2 {
				return
			}

			message := data[0]
			ackFunc, ok := data[1].(func(...interface{}))
			if !ok {
				return
			}

			log.Printf("Message (ack) from %s: %v", socket.ID(), message)

			// Send acknowledgment
			ackFunc("Message received!")

			// Broadcast
			rooms := socket.Rooms()
			for _, room := range rooms {
				if room != socket.ID() {
					server.To(room).Emit("message", map[string]interface{}{
						"from":    socket.ID(),
						"message": message,
						"room":    room,
					})
				}
			}
		})

		// Handle ping event
		socket.On("ping", func(data ...interface{}) {
			socket.Emit("pong", "pong from server")
		})

		// Store custom data
		socket.Set("connected_at", "now")

		// Handle disconnect
		socket.OnDisconnect(func(reason string) {
			log.Printf("Client disconnected: %s, reason: %s", socket.ID(), reason)

			// Get rooms before cleanup
			rooms := socket.Rooms()

			// Notify all rooms
			for _, room := range rooms {
				if room != socket.ID() {
					server.To(room).Emit("user_disconnected", map[string]interface{}{
						"socketId": socket.ID(),
						"room":     room,
					})
				}
			}
		})

		// Send welcome message
		socket.Emit("welcome", map[string]interface{}{
			"message": "Welcome to the chat server!",
			"id":      socket.ID(),
		})
	})

	// Serve Socket.IO
	http.Handle("/socket.io/", server)

	// Serve static files (for testing)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(indexHTML))
		} else {
			http.NotFound(w, r)
		}
	})

	log.Println("Chat server listening on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

const indexHTML = `<!DOCTYPE html>
<html>
<head>
    <title>Socket.IO Chat Example</title>
    <script src="https://cdn.socket.io/4.5.4/socket.io.min.js"></script>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        #messages { border: 1px solid #ccc; height: 300px; overflow-y: scroll; padding: 10px; margin: 20px 0; }
        .message { margin: 5px 0; padding: 5px; }
        .system { color: #666; font-style: italic; }
        input, button { padding: 10px; margin: 5px; }
        input[type="text"] { width: 300px; }
    </style>
</head>
<body>
    <h1>Socket.IO Chat Example</h1>
    <div>
        <input type="text" id="room" placeholder="Room name">
        <button onclick="joinRoom()">Join Room</button>
        <button onclick="leaveRoom()">Leave Room</button>
    </div>
    <div id="messages"></div>
    <div>
        <input type="text" id="message" placeholder="Type a message">
        <button onclick="sendMessage()">Send</button>
        <button onclick="sendMessageAck()">Send (with ack)</button>
    </div>

    <script>
        const socket = io('http://localhost:3000', {
            transports: ['websocket']
        });

        socket.on('connect', () => {
            addMessage('Connected to server', 'system');
        });

        socket.on('welcome', (data) => {
            addMessage('Welcome: ' + data.message + ' (ID: ' + data.id + ')', 'system');
        });

        socket.on('message', (data) => {
            addMessage('[' + data.room + '] ' + data.from + ': ' + data.message, 'message');
        });

        socket.on('user_joined', (data) => {
            addMessage(data.socketId + ' joined ' + data.room, 'system');
        });

        socket.on('user_left', (data) => {
            addMessage(data.socketId + ' left ' + data.room, 'system');
        });

        socket.on('user_disconnected', (data) => {
            addMessage(data.socketId + ' disconnected from ' + data.room, 'system');
        });

        socket.on('disconnect', () => {
            addMessage('Disconnected from server', 'system');
        });

        function joinRoom() {
            const room = document.getElementById('room').value;
            if (room) {
                socket.emit('join', room);
                addMessage('Joining room: ' + room, 'system');
            }
        }

        function leaveRoom() {
            const room = document.getElementById('room').value;
            if (room) {
                socket.emit('leave', room);
                addMessage('Leaving room: ' + room, 'system');
            }
        }

        function sendMessage() {
            const message = document.getElementById('message').value;
            if (message) {
                socket.emit('message', message);
                document.getElementById('message').value = '';
            }
        }

        function sendMessageAck() {
            const message = document.getElementById('message').value;
            if (message) {
                socket.emit('message_ack', message, (ack) => {
                    addMessage('Server acknowledged: ' + ack, 'system');
                });
                document.getElementById('message').value = '';
            }
        }

        function addMessage(text, className) {
            const messages = document.getElementById('messages');
            const div = document.createElement('div');
            div.className = 'message ' + className;
            div.textContent = text;
            messages.appendChild(div);
            messages.scrollTop = messages.scrollHeight;
        }
    </script>
</body>
</html>`

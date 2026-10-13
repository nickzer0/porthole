package websocket

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// EventMessage is a server-pushed event broadcast to all connected clients.
type EventMessage struct {
	Type    string `json:"type"`              // "notepad" | "files-changed"
	Content string `json:"content,omitempty"` // notepad body, when Type == "notepad"
}

// Hub maintains the set of persistent event subscribers and fans out
// broadcasts to every connected client. It lets one client's action
// (a notepad save, a completed upload) update everyone else's page live.
type Hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

// NewHub creates an empty Hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]struct{})}
}

// Add registers a subscriber connection.
func (h *Hub) Add(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()
}

// Remove unregisters a subscriber connection.
func (h *Hub) Remove(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
}

// broadcast writes a message to every connected client. The mutex is held for
// the whole loop so that concurrent broadcasts can never write to the same
// connection at once (gorilla/websocket forbids concurrent writes). Clients
// that error out are dropped.
func (h *Hub) broadcast(msg EventMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("Hub: dropping unreachable client: %v", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

// BroadcastNotepad pushes updated notepad content to all clients.
func (h *Hub) BroadcastNotepad(content string) {
	h.broadcast(EventMessage{Type: "notepad", Content: content})
}

// BroadcastFilesChanged tells all clients to refresh their file list.
func (h *Hub) BroadcastFilesChanged() {
	h.broadcast(EventMessage{Type: "files-changed"})
}

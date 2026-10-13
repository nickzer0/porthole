package websocket

import (
	"Porthole/pkg/files"
	"log"

	"github.com/gorilla/websocket"
)

// NotepadMessage represents a notepad WebSocket message
type NotepadMessage struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

// Notepad handles WebSocket-based notepad operations
type Notepad struct {
	fileManager *files.Manager
	conn        *websocket.Conn
	hub         *Hub
}

// NewNotepad creates a new WebSocket notepad handler
func NewNotepad(conn *websocket.Conn, fileManager *files.Manager, hub *Hub) *Notepad {
	return &Notepad{
		conn:        conn,
		fileManager: fileManager,
		hub:         hub,
	}
}

// HandleNotepad processes notepad WebSocket messages
func (n *Notepad) HandleNotepad() error {
	defer n.conn.Close()

	for {
		var msg NotepadMessage
		err := n.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNoStatusReceived,
				websocket.CloseNormalClosure) {
				log.Printf("WebSocket notepad error: %v", err)
			}
			break
		}

		switch msg.Type {
		case "get":
			content := n.fileManager.GetNotepad()
			n.sendMessage(NotepadMessage{
				Type:    "content",
				Content: content,
			})

		case "set":
			n.fileManager.SetNotepad(msg.Content)
			n.sendMessage(NotepadMessage{
				Type: "saved",
			})
			// Push the new content to every other open page.
			if n.hub != nil {
				n.hub.BroadcastNotepad(msg.Content)
			}

		case "clear":
			n.fileManager.ClearNotepad()
			n.sendMessage(NotepadMessage{
				Type: "cleared",
			})
			// Propagate the cleared state to every open page.
			if n.hub != nil {
				n.hub.BroadcastNotepad("")
			}

		default:
			n.sendMessage(NotepadMessage{
				Type:    "error",
				Content: "Unknown message type: " + msg.Type,
			})
		}
	}

	return nil
}

// sendMessage sends a message to the client
func (n *Notepad) sendMessage(msg NotepadMessage) {
	err := n.conn.WriteJSON(msg)
	if err != nil {
		log.Printf("Failed to send notepad message: %v", err)
	}
}

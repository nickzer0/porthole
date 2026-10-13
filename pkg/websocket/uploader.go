package websocket

import (
	"Porthole/pkg/files"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

// Message types
const (
	MessageTypeStart    = "start"
	MessageTypeChunk    = "chunk"
	MessageTypeComplete = "complete"
	MessageTypeError    = "error"
	MessageTypeProgress = "progress"
)

// UploadMessage represents a message sent over WebSocket
type UploadMessage struct {
	Type        string `json:"type"`
	FileName    string `json:"fileName,omitempty"`
	FileSize    int64  `json:"fileSize,omitempty"`
	ChunkNumber int    `json:"chunkNumber,omitempty"`
	TotalChunks int    `json:"totalChunks,omitempty"`
	Data        string `json:"data,omitempty"` // Base64 encoded data
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
}

// Uploader handles WebSocket-based file uploads
type Uploader struct {
	fileManager *files.Manager
	conn        *websocket.Conn
	hub         *Hub
}

// NewUploader creates a new WebSocket uploader
func NewUploader(conn *websocket.Conn, fileManager *files.Manager, hub *Hub) *Uploader {
	return &Uploader{
		conn:        conn,
		fileManager: fileManager,
		hub:         hub,
	}
}

// HandleUpload processes a WebSocket upload session
func (u *Uploader) HandleUpload() error {
	defer u.conn.Close()

	var currentFile *files.UploadSession

	for {
		var msg UploadMessage
		err := u.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNoStatusReceived,
				websocket.CloseNormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		switch msg.Type {
		case MessageTypeStart:
			// Start new upload session
			if err := u.fileManager.EnsureDownloadsDir(); err != nil {
				u.sendError(fmt.Sprintf("Failed to create downloads directory: %v", err))
				continue
			}

			currentFile, err = u.fileManager.StartUploadSession(msg.FileName)
			if err != nil {
				u.sendError(fmt.Sprintf("Failed to start upload: %v", err))
				continue
			}

			u.sendMessage(UploadMessage{
				Type:    "ready",
				Message: "Ready to receive file",
			})

		case MessageTypeChunk:
			if currentFile == nil {
				u.sendError("No active upload session")
				continue
			}

			// Decode base64 data
			data, err := base64.StdEncoding.DecodeString(msg.Data)
			if err != nil {
				u.sendError(fmt.Sprintf("Failed to decode chunk data: %v", err))
				continue
			}

			// Write chunk
			err = currentFile.WriteChunk(data)
			if err != nil {
				u.sendError(fmt.Sprintf("Failed to write chunk: %v", err))
				continue
			}

			// Send progress
			u.sendMessage(UploadMessage{
				Type:        MessageTypeProgress,
				ChunkNumber: msg.ChunkNumber,
				TotalChunks: msg.TotalChunks,
				Message:     fmt.Sprintf("Received chunk %d/%d", msg.ChunkNumber+1, msg.TotalChunks),
			})

			fmt.Printf("WebSocket: Received chunk %d/%d for %s\n", msg.ChunkNumber+1, msg.TotalChunks, currentFile.FileName)

		case MessageTypeComplete:
			if currentFile == nil {
				u.sendError("No active upload session")
				continue
			}

			// Close the file
			err := currentFile.Close()
			if err != nil {
				u.sendError(fmt.Sprintf("Failed to finalize upload: %v", err))
				continue
			}

			u.sendMessage(UploadMessage{
				Type:    MessageTypeComplete,
				Message: fmt.Sprintf("Upload complete: %s", currentFile.FileName),
			})

			// Tell every open page a new file is available.
			if u.hub != nil {
				u.hub.BroadcastFilesChanged()
			}

			currentFile = nil

		default:
			u.sendError(fmt.Sprintf("Unknown message type: %s", msg.Type))
		}
	}

	return nil
}

// sendMessage sends a message to the client
func (u *Uploader) sendMessage(msg UploadMessage) {
	err := u.conn.WriteJSON(msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

// sendError sends an error message to the client
func (u *Uploader) sendError(errMsg string) {
	u.sendMessage(UploadMessage{
		Type:  MessageTypeError,
		Error: errMsg,
	})
}

package websocket

import (
	"Porthole/pkg/files"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gorilla/websocket"
)

// DownloadMessage represents a message sent over WebSocket for downloads
type DownloadMessage struct {
	Type        string `json:"type"`
	FileName    string `json:"fileName,omitempty"`
	FileSize    int64  `json:"fileSize,omitempty"`
	ChunkNumber int    `json:"chunkNumber,omitempty"`
	TotalChunks int    `json:"totalChunks,omitempty"`
	Data        string `json:"data,omitempty"` // Base64 encoded data
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
}

// Downloader handles WebSocket-based file downloads
type Downloader struct {
	fileManager *files.Manager
	conn        *websocket.Conn
	hub         *Hub
}

// NewDownloader creates a new WebSocket downloader
func NewDownloader(conn *websocket.Conn, fileManager *files.Manager, hub *Hub) *Downloader {
	return &Downloader{
		conn:        conn,
		fileManager: fileManager,
		hub:         hub,
	}
}

// HandleDownload processes a WebSocket download session
func (d *Downloader) HandleDownload() error {
	defer d.conn.Close()

	for {
		var msg DownloadMessage
		err := d.conn.ReadJSON(&msg)
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
		case "list":
			// Client requests file listing
			fileList, err := d.fileManager.ListFiles()
			if err != nil {
				d.sendError(fmt.Sprintf("Failed to list files: %v", err))
				continue
			}

			// Send file list
			d.sendMessage(DownloadMessage{
				Type:    "list",
				Message: fmt.Sprintf("%d files found", len(fileList)),
			})

			// Send files as JSON by marshaling to the Message field
			// We'll send the actual file list data in a custom way
			err = d.conn.WriteJSON(map[string]interface{}{
				"type":  "files",
				"files": fileList,
			})
			if err != nil {
				log.Printf("Failed to send file list: %v", err)
			}

		case MessageTypeStart:
			// Client requests to download a file
			if msg.FileName == "" {
				d.sendError("File name is required")
				continue
			}

			d.StreamFile(msg.FileName)

		case "delete":
			// Client requests deletion of a listed file
			if msg.FileName == "" {
				d.sendError("File name is required")
				continue
			}

			if err := d.fileManager.DeleteFile(msg.FileName); err != nil {
				d.sendError(fmt.Sprintf("Failed to delete file: %v", err))
				continue
			}

			d.sendMessage(DownloadMessage{
				Type:     "deleted",
				FileName: msg.FileName,
				Message:  fmt.Sprintf("Deleted: %s", msg.FileName),
			})

			// Tell every open page to refresh its file list
			if d.hub != nil {
				d.hub.BroadcastFilesChanged()
			}

		default:
			d.sendError(fmt.Sprintf("Unknown message type: %s", msg.Type))
		}
	}

	return nil
}

// StreamFile sends a file over the connection using the chunked download
// protocol: metadata → base64 chunks → complete. Errors are reported to the
// client as error messages and returned for logging.
func (d *Downloader) StreamFile(fileName string) error {
	filePath, err := d.fileManager.GetFilePath(fileName)
	if err != nil {
		d.sendError(fmt.Sprintf("File not found: %v", err))
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		d.sendError(fmt.Sprintf("Failed to open file: %v", err))
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		d.sendError(fmt.Sprintf("Failed to get file info: %v", err))
		return err
	}

	fileSize := fileInfo.Size()
	chunkSize := int64(512 * 1024) // 512KB chunks
	totalChunks := int((fileSize + chunkSize - 1) / chunkSize)

	// Send metadata
	d.sendMessage(DownloadMessage{
		Type:        "metadata",
		FileName:    fileName,
		FileSize:    fileSize,
		TotalChunks: totalChunks,
	})

	// Send file in chunks
	buffer := make([]byte, chunkSize)
	chunkNumber := 0

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			d.sendError(fmt.Sprintf("Failed to read file: %v", err))
			return err
		}

		if n == 0 {
			break
		}

		// Encode chunk to base64
		encodedData := base64.StdEncoding.EncodeToString(buffer[:n])

		// Send chunk
		d.sendMessage(DownloadMessage{
			Type:        MessageTypeChunk,
			ChunkNumber: chunkNumber,
			TotalChunks: totalChunks,
			Data:        encodedData,
		})

		fmt.Printf("WebSocket: Sent chunk %d/%d for %s\n", chunkNumber+1, totalChunks, fileName)

		chunkNumber++

		if err == io.EOF {
			break
		}
	}

	// Send completion message
	d.sendMessage(DownloadMessage{
		Type:    MessageTypeComplete,
		Message: fmt.Sprintf("Download complete: %s", fileName),
	})

	return nil
}

// sendMessage sends a message to the client
func (d *Downloader) sendMessage(msg DownloadMessage) {
	err := d.conn.WriteJSON(msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

// sendError sends an error message to the client
func (d *Downloader) sendError(errMsg string) {
	d.sendMessage(DownloadMessage{
		Type:  MessageTypeError,
		Error: errMsg,
	})
}

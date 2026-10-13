package handlers

import (
	"Porthole/pkg/auth"
	"Porthole/pkg/files"
	"Porthole/pkg/share"
	"Porthole/pkg/ui"
	ws "Porthole/pkg/websocket"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 1024, // 1MB
	WriteBufferSize: 1024 * 1024, // 1MB
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins - adjust for production
	},
}

// Handlers holds all HTTP handlers and their dependencies
type Handlers struct {
	sessionManager *auth.SessionManager
	fileManager    *files.Manager
	hub            *ws.Hub
	shareManager   *share.Manager
}

// New creates a new Handlers instance
func New(sessionManager *auth.SessionManager, fileManager *files.Manager, hub *ws.Hub, shareManager *share.Manager) *Handlers {
	return &Handlers{
		sessionManager: sessionManager,
		fileManager:    fileManager,
		hub:            hub,
		shareManager:   shareManager,
	}
}

// Login handles the login page and authentication
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		w.Write([]byte(ui.LoginPage()))
		return
	}

	if r.Method == "POST" {
		r.ParseForm()
		inputPassword := r.FormValue("password")

		if h.sessionManager.ValidatePassword(inputPassword) {
			sessionID, err := h.sessionManager.CreateSession()
			if err != nil {
				http.Error(w, "Error creating session", http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, auth.CreateCookie(sessionID))
			http.Redirect(w, r, "/app", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Landing handles the initial landing page that bypasses ngrok warning
func (h *Handlers) Landing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(ui.LandingPage()))
}

// Main handles the main upload/download page
func (h *Handlers) Main(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(ui.MainPageWithWebSocket()))
}

// Upload handles chunked file uploads
func (h *Handlers) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the multipart form with 10MB max memory per chunk
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("chunk")
	if err != nil {
		http.Error(w, "Error retrieving file chunk", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileName := r.FormValue("fileName")
	chunkNumber := r.FormValue("chunkNumber")
	totalChunks := r.FormValue("totalChunks")

	if fileName == "" {
		http.Error(w, "Missing fileName", http.StatusBadRequest)
		return
	}

	// Ensure downloads directory exists
	if err := h.fileManager.EnsureDownloadsDir(); err != nil {
		http.Error(w, "Error creating directory", http.StatusInternalServerError)
		return
	}

	// Create or open file
	isFirstChunk := chunkNumber == "0"
	destFile, err := h.fileManager.CreateOrOpenFile(fileName, isFirstChunk)
	if err != nil {
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, file)
	if err != nil {
		http.Error(w, "Error saving chunk", http.StatusInternalServerError)
		return
	}

	chunkNum, _ := strconv.Atoi(chunkNumber)
	totalNum, _ := strconv.Atoi(totalChunks)

	fmt.Printf("Received chunk %d/%d for %s\n", chunkNum+1, totalNum, fileName)

	// On the final chunk the file is fully written — notify every open page.
	if h.hub != nil && chunkNum+1 >= totalNum {
		h.hub.BroadcastFilesChanged()
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Chunk %d/%d received", chunkNum+1, totalNum)))
}

// Files returns a JSON list of uploaded files
func (h *Handlers) Files(w http.ResponseWriter, r *http.Request) {
	fileList, err := h.fileManager.ListFiles()
	if err != nil {
		http.Error(w, "Error reading files", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileList)
}

// Download serves a file for download
func (h *Handlers) Download(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/download/"):]
	if filename == "" {
		http.Error(w, "Filename required", http.StatusBadRequest)
		return
	}

	filePath, err := h.fileManager.GetFilePath(filename)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	http.ServeFile(w, r, filePath)
}

// Delete removes an uploaded file (HTTP fallback for the WebSocket delete)
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := r.URL.Path[len("/delete/"):]
	if filename == "" {
		http.Error(w, "Filename required", http.StatusBadRequest)
		return
	}

	if err := h.fileManager.DeleteFile(filename); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete file: %v", err), http.StatusBadRequest)
		return
	}

	// Tell every open page to refresh its file list
	if h.hub != nil {
		h.hub.BroadcastFilesChanged()
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Deleted: %s", filename)))
}

// WebSocketUpload handles WebSocket-based file uploads
func (h *Handlers) WebSocketUpload(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// Create uploader and handle the session
	uploader := ws.NewUploader(conn, h.fileManager, h.hub)
	if err := uploader.HandleUpload(); err != nil {
		log.Printf("WebSocket upload error: %v", err)
	}
}

// WebSocketDownload handles WebSocket-based file downloads
func (h *Handlers) WebSocketDownload(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// Create downloader and handle the session
	downloader := ws.NewDownloader(conn, h.fileManager, h.hub)
	if err := downloader.HandleDownload(); err != nil {
		log.Printf("WebSocket download error: %v", err)
	}
}

// WebSocketFiles handles WebSocket-based file operations (list, download)
func (h *Handlers) WebSocketFiles(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// Create downloader and handle the session (handles both list and download)
	downloader := ws.NewDownloader(conn, h.fileManager, h.hub)
	if err := downloader.HandleDownload(); err != nil {
		log.Printf("WebSocket files error: %v", err)
	}
}

// WebSocketNotepad handles WebSocket-based notepad operations
func (h *Handlers) WebSocketNotepad(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// Create notepad handler and handle the session
	notepad := ws.NewNotepad(conn, h.fileManager, h.hub)
	if err := notepad.HandleNotepad(); err != nil {
		log.Printf("WebSocket notepad error: %v", err)
	}
}

// WebSocketEvents holds a persistent connection per client and pushes
// server-side events (notepad updates, new files) so every open page stays
// in sync without manual refreshing.
func (h *Handlers) WebSocketEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	h.hub.Add(conn)
	defer func() {
		h.hub.Remove(conn)
		conn.Close()
	}()

	// Block on reads until the client disconnects. The connection is otherwise
	// write-only (the server pushes events); draining reads keeps the
	// connection healthy and detects when the client goes away.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// Share creates an anonymous download link for a file (authenticated)
func (h *Handlers) Share(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FileName      string `json:"fileName"`
		ExpiryMinutes int    `json:"expiryMinutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.FileName == "" {
		http.Error(w, "fileName is required", http.StatusBadRequest)
		return
	}
	if req.ExpiryMinutes <= 0 {
		http.Error(w, "expiryMinutes must be positive", http.StatusBadRequest)
		return
	}

	// Verify the file exists before handing out a link
	if _, err := h.fileManager.GetFilePath(req.FileName); err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	token, expiresAt, err := h.shareManager.Create(req.FileName, time.Duration(req.ExpiryMinutes)*time.Minute)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create link: %v", err), http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url":       fmt.Sprintf("%s://%s/dl/%s", scheme, r.Host, token),
		"token":     token,
		"expiresAt": expiresAt.Format(time.RFC3339),
	})
}

// SharedDownload serves an anonymous download link (no auth, token-gated).
// Browsers get an HTML page that downloads over WebSocket; everything else
// (curl, wget, scripts) gets the file streamed directly. ?direct=1 forces
// the direct stream regardless of client.
func (h *Handlers) SharedDownload(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/dl/")
	if token == "" || strings.Contains(token, "/") {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	fileName, err := h.shareManager.Resolve(token)
	if err != nil {
		http.Error(w, "Link not found or expired", http.StatusNotFound)
		return
	}

	filePath, err := h.fileManager.GetFilePath(fileName)
	if err != nil {
		http.Error(w, "File no longer available", http.StatusNotFound)
		return
	}

	isBrowser := strings.Contains(r.Header.Get("Accept"), "text/html")
	if isBrowser && r.URL.Query().Get("direct") != "1" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var fileSize int64
		if info, err := os.Stat(filePath); err == nil {
			fileSize = info.Size()
		}
		w.Write([]byte(ui.ShareDownloadPage(token, fileName, fileSize)))
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	http.ServeFile(w, r, filePath)
}

// WebSocketSharedDownload streams a shared file over WebSocket (no auth,
// token-gated). The filename comes from the token, never from the client,
// so the anonymous endpoint can only ever serve the one shared file.
func (h *Handlers) WebSocketSharedDownload(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/ws-dl/")
	if token == "" || strings.Contains(token, "/") {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	fileName, err := h.shareManager.Resolve(token)
	if err != nil {
		http.Error(w, "Link not found or expired", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer conn.Close()

	// Stream immediately on connect — the client cannot request anything else
	downloader := ws.NewDownloader(conn, h.fileManager, h.hub)
	if err := downloader.StreamFile(fileName); err != nil {
		log.Printf("WebSocket shared download error: %v", err)
	}
}

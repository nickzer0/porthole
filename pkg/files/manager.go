package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// UploadSession represents an active file upload
type UploadSession struct {
	FileName string
	File     *os.File
	mutex    sync.Mutex
}

// FileInfo represents metadata about an uploaded file
type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// Manager handles file operations
type Manager struct {
	downloadsDir   string
	notepadContent string
	notepadMu      sync.RWMutex
}

// NewManager creates a new file manager that stores files in downloadsDir
func NewManager(downloadsDir string) *Manager {
	return &Manager{
		downloadsDir: downloadsDir,
	}
}

// GetNotepad returns the current notepad content
func (m *Manager) GetNotepad() string {
	m.notepadMu.RLock()
	defer m.notepadMu.RUnlock()
	return m.notepadContent
}

// SetNotepad sets the notepad content
func (m *Manager) SetNotepad(content string) {
	m.notepadMu.Lock()
	defer m.notepadMu.Unlock()
	m.notepadContent = content
}

// ClearNotepad clears the notepad content
func (m *Manager) ClearNotepad() {
	m.notepadMu.Lock()
	defer m.notepadMu.Unlock()
	m.notepadContent = ""
}

// EnsureDownloadsDir creates the downloads directory if it doesn't exist
func (m *Manager) EnsureDownloadsDir() error {
	return os.MkdirAll(m.downloadsDir, 0755)
}

// ListFiles returns all files in the downloads directory
func (m *Manager) ListFiles() ([]FileInfo, error) {
	files, err := os.ReadDir(m.downloadsDir)
	if err != nil {
		// If directory doesn't exist, return empty array instead of error
		if os.IsNotExist(err) {
			return []FileInfo{}, nil
		}
		return nil, fmt.Errorf("error reading files: %w", err)
	}

	var fileList []FileInfo
	for _, file := range files {
		if !file.IsDir() {
			info, err := file.Info()
			if err != nil {
				continue
			}
			fileList = append(fileList, FileInfo{
				Name:    file.Name(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
		}
	}

	// Ensure we always return an array, not nil
	if fileList == nil {
		fileList = []FileInfo{}
	}

	return fileList, nil
}

// GetFilePath returns the full path to a file, preventing directory traversal
func (m *Manager) GetFilePath(filename string) (string, error) {
	// Prevent directory traversal
	filename = filepath.Base(filename)
	filePath := filepath.Join(m.downloadsDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found")
	}

	return filePath, nil
}

// DeleteFile removes a file from the downloads directory. Only regular files
// directly inside the directory — i.e. exactly those returned by ListFiles —
// can be deleted: names containing path separators or ".." are rejected
// outright rather than normalized, and directories and symlinks are refused.
func (m *Manager) DeleteFile(filename string) error {
	if filename == "" || filename == "." || filename == ".." ||
		strings.ContainsAny(filename, `/\`) {
		return fmt.Errorf("invalid filename")
	}

	filePath := filepath.Join(m.downloadsDir, filename)

	info, err := os.Lstat(filePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found")
	}
	if err != nil {
		return fmt.Errorf("error accessing file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}

	return os.Remove(filePath)
}

// CreateOrOpenFile creates a new file or opens existing for appending
func (m *Manager) CreateOrOpenFile(filename string, isFirstChunk bool) (*os.File, error) {
	savePath := filepath.Join(m.downloadsDir, filename)

	var file *os.File
	var err error

	if isFirstChunk {
		file, err = os.Create(savePath)
	} else {
		file, err = os.OpenFile(savePath, os.O_APPEND|os.O_WRONLY, 0644)
	}

	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	return file, nil
}

// StartUploadSession starts a new upload session for WebSocket uploads
func (m *Manager) StartUploadSession(filename string) (*UploadSession, error) {
	// Prevent directory traversal
	filename = filepath.Base(filename)
	savePath := filepath.Join(m.downloadsDir, filename)

	file, err := os.Create(savePath)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}

	return &UploadSession{
		FileName: filename,
		File:     file,
	}, nil
}

// WriteChunk writes a chunk of data to the upload session
func (s *UploadSession) WriteChunk(data []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, err := s.File.Write(data)
	if err != nil {
		return fmt.Errorf("error writing chunk: %w", err)
	}

	return nil
}

// Close closes the upload session
func (s *UploadSession) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.File != nil {
		return s.File.Close()
	}
	return nil
}

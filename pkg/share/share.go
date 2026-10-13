package share

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Link is an anonymous download link for a single file
type Link struct {
	FileName  string
	ExpiresAt time.Time
}

// Manager holds active share links in memory, keyed by token
type Manager struct {
	mu    sync.Mutex
	links map[string]Link
}

// NewManager creates a new share link manager
func NewManager() *Manager {
	return &Manager{
		links: make(map[string]Link),
	}
}

// Create generates a new share link for fileName that expires after ttl
func (m *Manager) Create(fileName string, ttl time.Duration) (string, time.Time, error) {
	if ttl <= 0 {
		return "", time.Time{}, fmt.Errorf("expiry must be positive")
	}

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", time.Time{}, fmt.Errorf("error generating token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	expiresAt := time.Now().Add(ttl)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.purgeExpiredLocked()
	m.links[token] = Link{
		FileName:  fileName,
		ExpiresAt: expiresAt,
	}

	return token, expiresAt, nil
}

// Resolve returns the filename for a token, or an error if the token is
// unknown or expired. Expired entries are removed as they are encountered.
func (m *Manager) Resolve(token string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	link, ok := m.links[token]
	if !ok {
		return "", fmt.Errorf("link not found")
	}
	if time.Now().After(link.ExpiresAt) {
		delete(m.links, token)
		return "", fmt.Errorf("link expired")
	}

	return link.FileName, nil
}

// purgeExpiredLocked removes expired links. Caller must hold m.mu.
func (m *Manager) purgeExpiredLocked() {
	now := time.Now()
	for token, link := range m.links {
		if now.After(link.ExpiresAt) {
			delete(m.links, token)
		}
	}
}

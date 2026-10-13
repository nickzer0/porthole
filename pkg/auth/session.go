package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

const SessionDuration = 24 * time.Hour

// SessionManager manages user sessions
type SessionManager struct {
	sessions map[string]time.Time
	mutex    sync.RWMutex
	password string
}

// NewSessionManager creates a new session manager
func NewSessionManager(password string) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]time.Time),
		password: password,
	}
}

// GenerateSessionID creates a cryptographically secure session ID
func (sm *SessionManager) GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// IsAuthenticated checks if a request has a valid session
func (sm *SessionManager) IsAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}

	sm.mutex.RLock()
	expiry, exists := sm.sessions[cookie.Value]
	sm.mutex.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		sm.mutex.Lock()
		delete(sm.sessions, cookie.Value)
		sm.mutex.Unlock()
		return false
	}

	return true
}

// ValidatePassword checks if the provided password is correct
func (sm *SessionManager) ValidatePassword(password string) bool {
	return password == sm.password
}

// CreateSession creates a new session and returns the session ID
func (sm *SessionManager) CreateSession() (string, error) {
	sessionID, err := sm.GenerateSessionID()
	if err != nil {
		return "", err
	}

	sm.mutex.Lock()
	sm.sessions[sessionID] = time.Now().Add(SessionDuration)
	sm.mutex.Unlock()

	return sessionID, nil
}

// CreateCookie creates an HTTP cookie for the session
func CreateCookie(sessionID string) *http.Cookie {
	return &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(SessionDuration.Seconds()),
		SameSite: http.SameSiteStrictMode,
	}
}

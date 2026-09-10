package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
)

type SessionData struct {
	UserID    int64
	Username  string
	Nivel     int
	NombreRol string
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*SessionData),
	}
}

func (s *SessionStore) Create(data *SessionData) string {
	b := make([]byte, 32)
	rand.Read(b)
	id := hex.EncodeToString(b)

	s.mu.Lock()
	s.sessions[id] = data
	s.mu.Unlock()
	return id
}

func (s *SessionStore) Get(sessionID string) (*SessionData, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.sessions[sessionID]
	return d, ok
}

func (s *SessionStore) Delete(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

const SessionCookieName = "session_id"

func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func GetSessionID(r *http.Request) string {
	c, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

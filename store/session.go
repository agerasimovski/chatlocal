package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

const SessionCookieName = "session"
const sessionDuration = 24 * 7 * time.Hour

type sessionEntry struct {
	UserID    string    `json:"userId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type SessionStore struct {
	dir string
	mu  sync.RWMutex
}

func NewSessionStore(dataDir string) (*SessionStore, error) {
	dir := filepath.Join(dataDir, "sessions")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &SessionStore{dir: dir}, nil
}

func (st *SessionStore) path(id string) string {
	return filepath.Join(st.dir, id+".json")
}

func (st *SessionStore) Create(userID string) (sessionID string, err error) {
	id := uuid.New().String()
	ent := sessionEntry{
		UserID:    userID,
		ExpiresAt: time.Now().Add(sessionDuration),
	}
	data, err := json.Marshal(ent)
	if err != nil {
		return "", err
	}
	p := st.path(id)
	if err := os.WriteFile(p, data, 0600); err != nil {
		return "", err
	}
	return id, nil
}

func (st *SessionStore) Get(sessionID string) (userID string, ok bool) {
	if sessionID == "" {
		return "", false
	}
	data, err := os.ReadFile(st.path(sessionID))
	if err != nil {
		return "", false
	}
	var ent sessionEntry
	if err := json.Unmarshal(data, &ent); err != nil {
		return "", false
	}
	if time.Now().After(ent.ExpiresAt) {
		_ = os.Remove(st.path(sessionID))
		return "", false
	}
	return ent.UserID, true
}

func (st *SessionStore) Delete(sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return os.Remove(st.path(sessionID))
}

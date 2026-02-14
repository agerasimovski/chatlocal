package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Hash     string `json:"hash"`
}

type UserStore struct {
	path string
	mu   sync.RWMutex
	byID map[string]*User
	byName map[string]*User
}

func NewUserStore(dataDir string) (*UserStore, error) {
	path := filepath.Join(dataDir, "users.json")
	s := &UserStore{
		path:   path,
		byID:   make(map[string]*User),
		byName: make(map[string]*User),
	}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *UserStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var list []User
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range list {
		u := &list[i]
		s.byID[u.ID] = u
		s.byName[u.Username] = u
	}
	return nil
}

func (s *UserStore) save() error {
	s.mu.RLock()
	list := make([]User, 0, len(s.byID))
	for _, u := range s.byID {
		list = append(list, *u)
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

func (s *UserStore) ByID(id string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byID[id]
}

func (s *UserStore) ByUsername(username string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byName[username]
}

func (s *UserStore) Register(username, password string) (*User, error) {
	if username == "" || len(password) < 1 {
		return nil, ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.byName[username] != nil {
		s.mu.Unlock()
		return nil, ErrUserExists
	}
	u := &User{
		ID:       uuid.New().String(),
		Username: username,
		Hash:     string(hash),
	}
	s.byID[u.ID] = u
	s.byName[u.Username] = u
	s.mu.Unlock()
	if err := s.save(); err != nil {
		s.mu.Lock()
		delete(s.byID, u.ID)
		delete(s.byName, u.Username)
		s.mu.Unlock()
		return nil, err
	}
	return u, nil
}

func (s *UserStore) Login(username, password string) (*User, error) {
	u := s.ByUsername(username)
	if u == nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

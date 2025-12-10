package service

import (
	"sync"
)

type MemStorage struct {
	sessions map[string]string // sessionID → userID
	mu       sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		sessions: make(map[string]string),
	}
}

func (ms *MemStorage) GetSession(sessionID string) (string, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	val, ok := ms.sessions[sessionID]
	return val, ok
}

func (ms *MemStorage) AddSession(sessionID string, login string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.sessions[sessionID] = login
}

func (ms *MemStorage) DeleteSession(sessionID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.sessions, sessionID)
}

package service

import (
	"sync"
)

type MemSessionStorage struct {
	sessions map[string]string
	mu       sync.RWMutex
}

func NewMemStorage() *MemSessionStorage {
	return &MemSessionStorage{
		sessions: make(map[string]string),
	}
}

func (ms *MemSessionStorage) GetSession(sessionID string) (string, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	val, ok := ms.sessions[sessionID]
	return val, ok
}

func (ms *MemSessionStorage) AddSession(sessionID string, login string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.sessions[sessionID] = login
}

func (ms *MemSessionStorage) DeleteSession(sessionID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.sessions, sessionID)
}

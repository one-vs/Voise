package session

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type Manager struct {
	sessions map[uuid.UUID]*Session
	mu       sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[uuid.UUID]*Session),
	}
}

func (m *Manager) Create(ctx context.Context, callID uuid.UUID) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	s := &Session{
		ID:     uuid.New(),
		CallID: callID,
	}
	m.sessions[s.ID] = s
	return s
}

func (m *Manager) Get(id uuid.UUID) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

func (m *Manager) Remove(id uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

package session

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"voise/internal/voiceagent/tools"
)

type Manager struct {
	sessions   map[uuid.UUID]*Session
	db         *sqlx.DB
	toolRouter *tools.ToolRouter
	mu         sync.RWMutex
}

func NewManager(db *sqlx.DB, router *tools.ToolRouter) *Manager {
	return &Manager{
		sessions:   make(map[uuid.UUID]*Session),
		db:         db,
		toolRouter: router,
	}
}

func (m *Manager) Create(ctx context.Context, callID uuid.UUID) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	s := &Session{
		ID:         uuid.New(),
		CallID:     callID,
		DB:         m.db,
		ToolRouter: m.toolRouter,
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

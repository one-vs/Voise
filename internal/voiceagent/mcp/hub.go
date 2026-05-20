package mcp

import (
	"context"
	"sync"
)

type Hub struct {
	servers map[string]*Client
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		servers: make(map[string]*Client),
	}
}

func (h *Hub) RegisterServer(name string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.servers[name] = client
}

func (h *Hub) Initialize(ctx context.Context) error {
	// 1. Load config
	// 2. Start servers
	// 3. List tools
	return nil
}

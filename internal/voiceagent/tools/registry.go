package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type ToolHandler func(ctx context.Context, args json.RawMessage) (interface{}, error)

type ToolRegistry struct {
	tools map[string]ToolHandler
	mu    sync.RWMutex
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolHandler),
	}
}

func (r *ToolRegistry) Register(name string, handler ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = handler
}

func (r *ToolRegistry) Get(name string) ToolHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

type ToolRouter struct {
	registry *ToolRegistry
}

func NewToolRouter(r *ToolRegistry) *ToolRouter {
	return &ToolRouter{registry: r}
}

func (tr *ToolRouter) Invoke(ctx context.Context, name string, args json.RawMessage) (interface{}, error) {
	handler := tr.registry.Get(name)
	if handler == nil {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return handler(ctx, args)
}

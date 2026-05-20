package mcp

import (
	"context"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
	"voise/internal/voiceagent/voicelog"
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

type HubConfig struct {
	Servers []ServerConfig `yaml:"servers"`
}

type ServerConfig struct {
	Name      string            `yaml:"name"`
	Transport string            `yaml:"transport"`
	Command   string            `yaml:"command,omitempty"`
	Args      []string          `yaml:"args,omitempty"`
	URL       string            `yaml:"url,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	Enabled   bool              `yaml:"enabled"`
}

func (h *Hub) LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg HubConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	for _, s := range cfg.Servers {
		if !s.Enabled {
			continue
		}

		if s.Transport == "stdio" {
			transport, err := NewStdioTransport(s.Command, s.Args...)
			if err != nil {
				voicelog.Logger.Error().Err(err).Str("server", s.Name).Msg("Failed to start MCP server")
				continue
			}
			h.RegisterServer(s.Name, &Client{transport: transport})
		}
	}

	return nil
}

func (h *Hub) RegisterServer(name string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.servers[name] = client
}

func (h *Hub) ListTools(ctx context.Context) (map[string]interface{}, error) {
	// Logic to call list_tools on all servers
	return nil, nil
}

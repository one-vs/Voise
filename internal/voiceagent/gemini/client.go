package gemini

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// Client is a client for the Gemini Live API.
type Client struct {
	apiKey string
	model  string
}

// NewClient creates a new Gemini Live client.
func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
	}
}

// Conn represents a connection to the Gemini Live API.
type Conn struct {
	ws *websocket.Conn
}

// Connect connects to the Gemini Live API.
func (c *Client) Connect(ctx context.Context) (*Conn, error) {
	url := fmt.Sprintf("wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContent?key=%s", c.apiKey)

	dialer := websocket.DefaultDialer
	ws, _, err := dialer.DialContext(ctx, url, http.Header{})
	if err != nil {
		return nil, fmt.Errorf("failed to dial gemini: %w", err)
	}

	return &Conn{ws: ws}, nil
}

// SendRaw sends a raw JSON message to Gemini.
func (c *Conn) SendRaw(v interface{}) error {
	return c.ws.WriteJSON(v)
}

// ReceiveRaw receives a raw JSON message from Gemini.
func (c *Conn) ReceiveRaw(v interface{}) error {
	return c.ws.ReadJSON(v)
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.ws.Close()
}

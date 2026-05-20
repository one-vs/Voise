package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

type HTTPTransport struct {
	URL string
}

func (t *HTTPTransport) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	// Logic to send a POST request with JSON-RPC payload
	return nil, fmt.Errorf("HTTP transport call not fully implemented in skeleton")
}

func (t *HTTPTransport) Close() error {
	return nil
}

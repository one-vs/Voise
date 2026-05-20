package gemini

import "encoding/json"

// ServerContent represents the 'serverContent' message from Gemini.
type ServerContent struct {
	ModelTurn   *ModelTurn `json:"modelTurn,omitempty"`
	Interrupted bool       `json:"interrupted,omitempty"`
	TurnComplete bool       `json:"turnComplete,omitempty"`
}

// ModelTurn represents the model's turn in the conversation.
type ModelTurn struct {
	Parts []Part `json:"parts,omitempty"`
}

// Part represents a part of a message (text or audio).
type Part struct {
	Text   string `json:"text,omitempty"`
	InlineData *Blob `json:"inlineData,omitempty"`
}

// Blob represents audio data.
type Blob struct {
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"` // Base64 encoded
}

// ToolCall represents a tool call from Gemini.
type ToolCall struct {
	FunctionCalls []FunctionCall `json:"functionCalls,omitempty"`
}

// FunctionCall represents a single function call.
type FunctionCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
	ID   string          `json:"id"`
}

// GeminiResponse is a top-level response from the Gemini WS.
type GeminiResponse struct {
	ServerContent         *ServerContent         `json:"serverContent,omitempty"`
	ToolCall              *ToolCall              `json:"toolCall,omitempty"`
	SetupComplete         *SetupComplete         `json:"setupComplete,omitempty"`
	RealtimeInputResponse *RealtimeInputResponse `json:"realtimeInputResponse,omitempty"`
}

// SetupComplete represents the 'setupComplete' message.
type SetupComplete struct{}

// Transcription represents the transcription from Gemini.
type Transcription struct {
	Text      string `json:"text"`
	Speaker   string `json:"speaker"`
	IsFinal   bool   `json:"isFinal"`
	Timestamp string `json:"timestamp"`
}

type RealtimeInputResponse struct {
	Transcription *Transcription `json:"transcription,omitempty"`
}

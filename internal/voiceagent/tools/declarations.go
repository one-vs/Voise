package tools

import (
	"encoding/json"
	"voise/internal/voiceagent/gemini"
)

// MCPTool represents a tool definition from an MCP server.
type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ConvertMCPToGemini converts an MCP tool definition to Gemini FunctionDeclaration.
func ConvertMCPToGemini(mcpTool MCPTool) gemini.FunctionDeclaration {
	return gemini.FunctionDeclaration{
		Name:        mcpTool.Name,
		Description: mcpTool.Description,
		Parameters:  mcpTool.InputSchema,
	}
}

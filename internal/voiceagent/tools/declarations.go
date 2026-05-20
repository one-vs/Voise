package tools

import "voise/internal/voiceagent/gemini"

// ConvertMCPToGemini converts an MCP tool definition to Gemini FunctionDeclaration.
func ConvertMCPToGemini(mcpTool interface{}) gemini.FunctionDeclaration {
	// Implementation would map MCP JSON Schema to Gemini parameters
	return gemini.FunctionDeclaration{}
}

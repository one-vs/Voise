package config

import (
	"context"
	"os"
)

// GetGeminiKey retrieves the Gemini API key from environment or config.
func GetGeminiKey(ctx context.Context, cfg *Config) string {
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return key
	}
	return cfg.VoiceAgent.Gemini.APIKey
}

// GetTwilioCreds retrieves Twilio credentials.
func GetTwilioCreds(ctx context.Context, cfg *Config) (string, string) {
	sid := os.Getenv("TWILIO_ACCOUNT_SID")
	if sid == "" {
		sid = cfg.VoiceAgent.Twilio.AccountSid
	}
	token := os.Getenv("TWILIO_AUTH_TOKEN")
	if token == "" {
		token = cfg.VoiceAgent.Twilio.AuthToken
	}
	return sid, token
}

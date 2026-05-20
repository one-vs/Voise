package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	VoiceAgent VoiceAgentConfig `mapstructure:"voice_agent"`
}

type VoiceAgentConfig struct {
	Gemini         GeminiConfig    `mapstructure:"gemini"`
	Twilio         TwilioConfig    `mapstructure:"twilio"`
	MCP            MCPConfig       `mapstructure:"mcp"`
	Session        SessionConfig   `mapstructure:"session"`
	Recording      RecordingConfig `mapstructure:"recording"`
	DatabaseURL    string          `mapstructure:"database_url"`
}

type GeminiConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type TwilioConfig struct {
	AccountSid string `mapstructure:"account_sid"`
	AuthToken  string `mapstructure:"auth_token"`
}

type MCPConfig struct {
	ConfigPath string `mapstructure:"config_path"`
}

type SessionConfig struct {
	MaxDuration    time.Duration `mapstructure:"max_duration"`
	SilenceTimeout time.Duration `mapstructure:"silence_timeout"`
}

type RecordingConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.VoiceAgent.Gemini.APIKey == "" {
		return fmt.Errorf("gemini api_key is required")
	}
	if c.VoiceAgent.Twilio.AuthToken == "" {
		return fmt.Errorf("twilio auth_token is required")
	}
	return nil
}

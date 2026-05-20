package gemini

type LiveConnectConfig struct {
	Model              string              `json:"model"`
	SystemInstructions *SystemInstructions `json:"system_instructions,omitempty"`
	Tools              []Tool              `json:"tools,omitempty"`
	GenerationConfig   *GenerationConfig   `json:"generation_config,omitempty"`
}

type SystemInstructions struct {
	Parts []Part `json:"parts"`
}

type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"function_declarations,omitempty"`
}

type FunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type GenerationConfig struct {
	ResponseModalities []string `json:"response_modalities,omitempty"`
	SpeechConfig       *SpeechConfig `json:"speech_config,omitempty"`
}

type SpeechConfig struct {
	VoiceConfig *VoiceConfig `json:"voice_config,omitempty"`
}

type VoiceConfig struct {
	PrebuiltVoiceConfig *PrebuiltVoiceConfig `json:"prebuilt_voice_config,omitempty"`
}

type PrebuiltVoiceConfig struct {
	VoiceName string `json:"voice_name"`
}

func NewLiveConnectConfig(model, instructions string) *LiveConnectConfig {
	return &LiveConnectConfig{
		Model: model,
		SystemInstructions: &SystemInstructions{
			Parts: []Part{{Text: instructions}},
		},
		GenerationConfig: &GenerationConfig{
			ResponseModalities: []string{"AUDIO"},
		},
	}
}

package domain

import (
	"time"
)

// Session represents a WebSocket session configuration
type Session struct {
	Type             string         `json:"type"`                   // "realtime" or "transcription"
	Object           string         `json:"object"`                 // "realtime.session"
	ID               string         `json:"id"`                     // Session ID
	Model            string         `json:"model"`                  // Model identifier
	OutputModalities []string       `json:"output_modalities"`      // ["audio", "text"]
	Instructions     string         `json:"instructions,omitempty"` // System instructions
	Tools            []Tool         `json:"tools"`                  // Available tools
	ToolChoice       string         `json:"tool_choice"`            // "auto", "none", or tool name
	MaxOutputTokens  interface{}    `json:"max_output_tokens"`      // "inf" or number
	Temperature      float64        `json:"temperature,omitempty"`  // 0.6-1.2
	Tracing          *string        `json:"tracing"`                // "none" or null
	Prompt           *string        `json:"prompt"`                 // null
	ExpiresAt        int64          `json:"expires_at"`             // Unix timestamp
	Audio            *AudioConfig   `json:"audio"`                  // Audio configuration
	Include          []string       `json:"include,omitempty"`      // e.g., ["item.input_audio_transcription.logprobs"]
	VoiceSettings    *VoiceSettings `json:"voice_settings,omitempty"`
}

// VoiceSettings represents voice customization
type VoiceSettings struct {
	Voice string  `json:"voice"`
	Speed float64 `json:"speed,omitempty"`
}

// SessionState tracks the session runtime state
type SessionState struct {
	ID              string
	Config          *Session
	Conversation    *ConversationState
	AudioBuffer     *AudioBuffer
	CurrentResponse *Response
	CreatedAt       time.Time
	LastActivity    time.Time
	ASRProvider     ASRProvider // Per-session ASR provider to avoid race conditions
	VADProvider     VADProvider // Per-session VAD provider
}

// NewSession creates a default session configuration
func NewSession(sessionID, model string) *Session {
	expiresAt := time.Now().Add(1 * time.Hour).Unix()

	return &Session{
		Type:             "realtime",
		Object:           "realtime.session",
		ID:               sessionID,
		Model:            model,
		OutputModalities: []string{"audio"},
		Instructions:     "You are a helpful, witty, and friendly AI assistant.",
		Tools:            []Tool{},
		ToolChoice:       "auto",
		MaxOutputTokens:  "inf",
		Temperature:      0.8,
		Tracing:          nil,
		Prompt:           nil,
		ExpiresAt:        expiresAt,
		Audio: &AudioConfig{
			Input: &AudioInput{
				Format: &AudioFormat{
					Type: "audio/pcm",
					Rate: 24000,
				},
				Transcription:  nil,
				NoiseReduction: nil,
				TurnDetection: &TurnDetection{
					Type:              "server_vad",
					Threshold:         0.5,
					PrefixPaddingMs:   300,
					SilenceDurationMs: 200,
					IdleTimeoutMs:     nil,
					CreateResponse:    true,
					InterruptResponse: true,
				},
			},
			Output: &AudioOutput{
				Format: &AudioFormat{
					Type: "audio/pcm",
					Rate: 24000,
				},
				Voice: "alloy",
				Speed: 1.0,
			},
		},
	}
}

// NewTranscriptionSession creates a session configured for transcription-only mode (STT)
func NewTranscriptionSession(sessionID, model, language string) *Session {
	expiresAt := time.Now().Add(1 * time.Hour).Unix()

	return &Session{
		Type:             "transcription",
		Object:           "realtime.session",
		ID:               sessionID,
		Model:            model,
		OutputModalities: []string{"text"}, // STT only outputs text
		Instructions:     "",
		Tools:            []Tool{},
		ToolChoice:       "none",
		MaxOutputTokens:  "inf",
		Temperature:      0.0,
		Tracing:          nil,
		Prompt:           nil,
		ExpiresAt:        expiresAt,
		Audio: &AudioConfig{
			Input: &AudioInput{
				Format: &AudioFormat{
					Type: "audio/pcm",
					Rate: 24000,
				},
				Transcription: &TranscriptionConfig{
					Model:    model,
					Language: language,
				},
				NoiseReduction: &NoiseReduction{
					Type: "near_field",
				},
				TurnDetection: &TurnDetection{
					Type:              "server_vad",
					Threshold:         0.5,
					PrefixPaddingMs:   300,
					SilenceDurationMs: 500,
					IdleTimeoutMs:     nil,
					CreateResponse:    false, // No response generation in STT mode
					InterruptResponse: false,
				},
			},
			Output: nil, // No audio output in transcription mode
		},
	}
}

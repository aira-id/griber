package domain

import "context"

// VADEvent represents a voice activity detection event
type VADEvent struct {
	Type      VADEventType `json:"type"`
	StartMs   int          `json:"start_ms,omitempty"`
	EndMs     int          `json:"end_ms,omitempty"`
	AudioData []byte       `json:"-"` // The audio segment (for speech segments)
}

// VADEventType represents the type of VAD event
type VADEventType string

const (
	VADEventSpeechStarted VADEventType = "speech_started"
	VADEventSpeechStopped VADEventType = "speech_stopped"
	VADEventTimeout       VADEventType = "timeout"
)

// VADProvider defines the interface for voice activity detection
type VADProvider interface {
	// ProcessAudio processes audio data and detects voice activity
	// Returns VAD events through the channel
	ProcessAudio(ctx context.Context, audio []byte) error

	// GetEvents returns a channel that receives VAD events
	GetEvents() <-chan VADEvent

	// Configure updates VAD settings
	Configure(config *VADConfig) error

	// Reset clears internal state
	Reset()

	// Close releases resources
	Close() error
}

// VADConfig holds configuration for VAD
type VADConfig struct {
	// Type of VAD: "server_vad", "semantic_vad", or nil for manual
	Type string `json:"type"`

	// Threshold for speech detection (0.0-1.0)
	Threshold float64 `json:"threshold"`

	// PrefixPaddingMs - audio to include before speech start
	PrefixPaddingMs int `json:"prefix_padding_ms"`

	// SilenceDurationMs - silence duration to consider speech ended
	SilenceDurationMs int `json:"silence_duration_ms"`

	// IdleTimeoutMs - timeout for no speech detected
	IdleTimeoutMs int `json:"idle_timeout_ms,omitempty"`

	// SampleRate of the audio (e.g., 24000)
	SampleRate int `json:"sample_rate"`

	// Channels - number of audio channels (usually 1 for mono)
	Channels int `json:"channels"`
}

// NewDefaultVADConfig creates a default VAD configuration
func NewDefaultVADConfig() *VADConfig {
	return &VADConfig{
		Type:              "server_vad",
		Threshold:         0.5,
		PrefixPaddingMs:   300,
		SilenceDurationMs: 500,
		IdleTimeoutMs:     0,
		SampleRate:        24000,
		Channels:          1,
	}
}

// VADConfigFromTurnDetection creates VADConfig from TurnDetection settings
func VADConfigFromTurnDetection(td *TurnDetection) *VADConfig {
	if td == nil {
		return nil
	}

	config := &VADConfig{
		Type:              td.Type,
		Threshold:         td.Threshold,
		PrefixPaddingMs:   td.PrefixPaddingMs,
		SilenceDurationMs: td.SilenceDurationMs,
		SampleRate:        24000,
		Channels:          1,
	}

	if td.IdleTimeoutMs != nil {
		if timeout, ok := td.IdleTimeoutMs.(float64); ok {
			config.IdleTimeoutMs = int(timeout)
		} else if timeout, ok := td.IdleTimeoutMs.(int); ok {
			config.IdleTimeoutMs = timeout
		}
	}

	return config
}

package domain

// ============================================================================
// CLIENT EVENTS
// ============================================================================

// SessionUpdateClientEvent represents a session.update client event
type SessionUpdateClientEvent struct {
	BaseEvent
	Session *Session `json:"session"`
}

// InputAudioBufferAppendEvent represents input_audio_buffer.append event
type InputAudioBufferAppendEvent struct {
	BaseEvent
	Audio string `json:"audio"` // base64-encoded audio bytes
}

// InputAudioBufferCommitEvent represents input_audio_buffer.commit event
type InputAudioBufferCommitEvent struct {
	BaseEvent
}

// InputAudioBufferClearEvent represents input_audio_buffer.clear event
type InputAudioBufferClearEvent struct {
	BaseEvent
}

// ConversationItemCreateClientEvent represents conversation.item.create event
type ConversationItemCreateClientEvent struct {
	BaseEvent
	Item           *Item   `json:"item"`
	PreviousItemID *string `json:"previous_item_id,omitempty"` // null, "root", or item ID
}

// ConversationItemRetrieveEvent represents conversation.item.retrieve event
type ConversationItemRetrieveEvent struct {
	BaseEvent
	ItemID string `json:"item_id"`
}

// ConversationItemTruncateEvent represents conversation.item.truncate event
type ConversationItemTruncateEvent struct {
	BaseEvent
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"` // Usually 0
	AudioEndMs   int    `json:"audio_end_ms"`
}

// ConversationItemDeleteEvent represents conversation.item.delete event
type ConversationItemDeleteEvent struct {
	BaseEvent
	ItemID string `json:"item_id"`
}

// ResponseCreateClientEvent represents response.create event
type ResponseCreateClientEvent struct {
	BaseEvent
	Response *ResponseCreatePayload `json:"response,omitempty"`
}

// ResponseCreatePayload represents the response payload in response.create
type ResponseCreatePayload struct {
	Modalities       []string      `json:"modalities,omitempty"`
	Instructions     string        `json:"instructions,omitempty"`
	Tools            []Tool        `json:"tools,omitempty"`
	ToolChoice       string        `json:"tool_choice,omitempty"`
	Conversation     string        `json:"conversation,omitempty"` // "default" or "none"
	OutputModalities []string      `json:"output_modalities,omitempty"`
	Metadata         interface{}   `json:"metadata,omitempty"`
	Input            []interface{} `json:"input,omitempty"` // Items or item references
	MaxOutputTokens  interface{}   `json:"max_output_tokens,omitempty"`
}

// ResponseCancelEvent represents response.cancel event
type ResponseCancelEvent struct {
	BaseEvent
	ResponseID *string `json:"response_id,omitempty"`
}

// OutputAudioBufferClearEvent represents output_audio_buffer.clear event
type OutputAudioBufferClearEvent struct {
	BaseEvent
}

// ============================================================================
// SERVER EVENTS
// ============================================================================

// ErrorServerEvent represents an error event
type ErrorServerEvent struct {
	BaseEvent
	Error *ErrorDetail `json:"error"`
}

// SessionCreatedEvent represents session.created event
type SessionCreatedEvent struct {
	BaseEvent
	Session *Session `json:"session"`
}

// SessionUpdatedEvent represents session.updated event
type SessionUpdatedEvent struct {
	BaseEvent
	Session *Session `json:"session"`
}

// ConversationItemAddedEvent represents conversation.item.added event
type ConversationItemAddedEvent struct {
	BaseEvent
	Item           *Item   `json:"item"`
	PreviousItemID *string `json:"previous_item_id"`
}

// ConversationItemDoneEvent represents conversation.item.done event
type ConversationItemDoneEvent struct {
	BaseEvent
	Item           *Item   `json:"item"`
	PreviousItemID *string `json:"previous_item_id"`
}

// ConversationItemRetrievedEvent represents conversation.item.retrieved event
type ConversationItemRetrievedEvent struct {
	BaseEvent
	Item *Item `json:"item"`
}

// ConversationItemDeletedEvent represents conversation.item.deleted event
type ConversationItemDeletedEvent struct {
	BaseEvent
	ItemID string `json:"item_id"`
}

// ConversationItemTruncatedEvent represents conversation.item.truncated event
type ConversationItemTruncatedEvent struct {
	BaseEvent
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	AudioEndMs   int    `json:"audio_end_ms"`
}

// InputAudioBufferCommittedEvent represents input_audio_buffer.committed event
type InputAudioBufferCommittedEvent struct {
	BaseEvent
	ItemID         string  `json:"item_id"`
	PreviousItemID *string `json:"previous_item_id"`
}

// InputAudioBufferClearedEvent represents input_audio_buffer.cleared event
type InputAudioBufferClearedEvent struct {
	BaseEvent
}

// InputAudioBufferSpeechStartedEvent represents input_audio_buffer.speech_started event
type InputAudioBufferSpeechStartedEvent struct {
	BaseEvent
	AudioStartMs int    `json:"audio_start_ms"`
	ItemID       string `json:"item_id"`
}

// InputAudioBufferSpeechStoppedEvent represents input_audio_buffer.speech_stopped event
type InputAudioBufferSpeechStoppedEvent struct {
	BaseEvent
	AudioEndMs int    `json:"audio_end_ms"`
	ItemID     string `json:"item_id"`
}

// InputAudioBufferTimeoutTriggeredEvent represents input_audio_buffer.timeout_triggered event
type InputAudioBufferTimeoutTriggeredEvent struct {
	BaseEvent
	AudioStartMs int    `json:"audio_start_ms"`
	AudioEndMs   int    `json:"audio_end_ms"`
	ItemID       string `json:"item_id"`
}

// ResponseCreatedEvent represents response.created event
type ResponseCreatedEvent struct {
	BaseEvent
	Response *Response `json:"response"`
}

// ResponseDoneEvent represents response.done event
type ResponseDoneEvent struct {
	BaseEvent
	Response *Response `json:"response"`
}

// ResponseOutputItemAddedEvent represents response.output_item.added event
type ResponseOutputItemAddedEvent struct {
	BaseEvent
	ResponseID  string `json:"response_id"`
	OutputIndex int    `json:"output_index"`
	Item        *Item  `json:"item"`
}

// ResponseOutputItemDoneEvent represents response.output_item.done event
type ResponseOutputItemDoneEvent struct {
	BaseEvent
	ResponseID  string `json:"response_id"`
	OutputIndex int    `json:"output_index"`
	Item        *Item  `json:"item"`
}

// ResponseContentPartAddedEvent represents response.content_part.added event
type ResponseContentPartAddedEvent struct {
	BaseEvent
	ResponseID   string       `json:"response_id"`
	ItemID       string       `json:"item_id"`
	ContentIndex int          `json:"content_index"`
	OutputIndex  int          `json:"output_index"`
	Part         *ContentPart `json:"part"`
}

// ResponseContentPartDoneEvent represents response.content_part.done event
type ResponseContentPartDoneEvent struct {
	BaseEvent
	ResponseID   string       `json:"response_id"`
	ItemID       string       `json:"item_id"`
	ContentIndex int          `json:"content_index"`
	OutputIndex  int          `json:"output_index"`
	Part         *ContentPart `json:"part"`
}

// ResponseOutputTextDeltaEvent represents response.output_text.delta event
type ResponseOutputTextDeltaEvent struct {
	BaseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	OutputIndex  int    `json:"output_index"`
	Delta        string `json:"delta"`
}

// ResponseOutputTextDoneEvent represents response.output_text.done event
type ResponseOutputTextDoneEvent struct {
	BaseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	OutputIndex  int    `json:"output_index"`
	Text         string `json:"text"`
}

// ResponseOutputAudioTranscriptDeltaEvent represents response.output_audio_transcript.delta event
type ResponseOutputAudioTranscriptDeltaEvent struct {
	BaseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	OutputIndex  int    `json:"output_index"`
	Delta        string `json:"delta"`
}

// ResponseOutputAudioTranscriptDoneEvent represents response.output_audio_transcript.done event
type ResponseOutputAudioTranscriptDoneEvent struct {
	BaseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	OutputIndex  int    `json:"output_index"`
	Transcript   string `json:"transcript"`
}

// ResponseOutputAudioDeltaEvent represents response.output_audio.delta event
type ResponseOutputAudioDeltaEvent struct {
	BaseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	OutputIndex  int    `json:"output_index"`
	Delta        string `json:"delta"` // base64-encoded audio
}

// ResponseOutputAudioDoneEvent represents response.output_audio.done event
type ResponseOutputAudioDoneEvent struct {
	BaseEvent
	ResponseID   string `json:"response_id"`
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	OutputIndex  int    `json:"output_index"`
}

// ConversationItemInputAudioTranscriptionCompletedEvent represents conversation.item.input_audio_transcription.completed event
type ConversationItemInputAudioTranscriptionCompletedEvent struct {
	BaseEvent
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	Transcript   string `json:"transcript"`
	Usage        *Usage `json:"usage"`
}

// ConversationItemInputAudioTranscriptionDeltaEvent represents conversation.item.input_audio_transcription.delta event
type ConversationItemInputAudioTranscriptionDeltaEvent struct {
	BaseEvent
	ItemID       string `json:"item_id"`
	ContentIndex int    `json:"content_index"`
	Delta        string `json:"delta"`
}

// RateLimitsUpdatedEvent represents rate_limits.updated event
type RateLimitsUpdatedEvent struct {
	BaseEvent
	RateLimits []RateLimit `json:"rate_limits"`
}

// ============================================================================
// TRANSCRIPTION SESSION EVENTS (OpenAI Realtime Transcription API compatible)
// ============================================================================

// TranscriptionSessionUpdateClientEvent represents transcription_session.update client event
// This is the flattened structure matching OpenAI's transcription API
type TranscriptionSessionUpdateClientEvent struct {
	BaseEvent
	Session *TranscriptionSessionConfig `json:"session"`
}

// TranscriptionSessionCreatedEvent represents transcription_session.created server event
type TranscriptionSessionCreatedEvent struct {
	BaseEvent
	Session *TranscriptionSessionConfig `json:"session"`
}

// TranscriptionSessionUpdatedEvent represents transcription_session.updated server event
type TranscriptionSessionUpdatedEvent struct {
	BaseEvent
	Session *TranscriptionSessionConfig `json:"session"`
}

// TranscriptionSessionConfig represents the flattened transcription session configuration
// matching OpenAI's Realtime Transcription API structure
type TranscriptionSessionConfig struct {
	Object                   string                          `json:"object,omitempty"`                      // "realtime.transcription_session"
	Type                     string                          `json:"type,omitempty"`                        // Always "transcription"
	ID                       string                          `json:"id,omitempty"`                          // Session ID
	InputAudioFormat         string                          `json:"input_audio_format,omitempty"`          // "pcm16", "g711_ulaw", "g711_alaw"
	InputAudioTranscription  *InputAudioTranscriptionConfig  `json:"input_audio_transcription,omitempty"`   // Transcription settings
	TurnDetection            *TurnDetectionConfig            `json:"turn_detection,omitempty"`              // VAD settings
	InputAudioNoiseReduction *InputAudioNoiseReductionConfig `json:"input_audio_noise_reduction,omitempty"` // Noise reduction settings
	Include                  []string                        `json:"include,omitempty"`                     // e.g., ["item.input_audio_transcription.logprobs"]
	ExpiresAt                int64                           `json:"expires_at,omitempty"`                  // Unix timestamp
}

// InputAudioTranscriptionConfig represents transcription settings in OpenAI format
type InputAudioTranscriptionConfig struct {
	Model    string `json:"model,omitempty"`    // "gpt-4o-transcribe", etc.
	Language string `json:"language,omitempty"` // ISO-639-1 code like "en"
	Prompt   string `json:"prompt,omitempty"`   // Optional prompt to guide transcription
}

// TurnDetectionConfig represents VAD settings in OpenAI format
type TurnDetectionConfig struct {
	Type              string  `json:"type,omitempty"`                // "server_vad" or "semantic_vad"
	Threshold         float64 `json:"threshold,omitempty"`           // 0.0-1.0
	PrefixPaddingMs   int     `json:"prefix_padding_ms,omitempty"`   // milliseconds
	SilenceDurationMs int     `json:"silence_duration_ms,omitempty"` // milliseconds
}

// InputAudioNoiseReductionConfig represents noise reduction settings in OpenAI format
type InputAudioNoiseReductionConfig struct {
	Type string `json:"type,omitempty"` // "near_field", "far_field"
}

// NewTranscriptionSessionConfig creates a TranscriptionSessionConfig from a Session
// This converts from the nested structure to the flattened OpenAI-compatible format
func NewTranscriptionSessionConfig(session *Session) *TranscriptionSessionConfig {
	config := &TranscriptionSessionConfig{
		Object:    "realtime.transcription_session",
		Type:      "transcription",
		ID:        session.ID,
		ExpiresAt: session.ExpiresAt,
		Include:   session.Include,
	}

	// Map audio input format
	if session.Audio != nil && session.Audio.Input != nil {
		if session.Audio.Input.Format != nil {
			// Convert format type to OpenAI naming
			switch session.Audio.Input.Format.Type {
			case "audio/pcm":
				config.InputAudioFormat = "pcm16"
			case "audio/pcmu":
				config.InputAudioFormat = "g711_ulaw"
			case "audio/pcma":
				config.InputAudioFormat = "g711_alaw"
			default:
				config.InputAudioFormat = "pcm16"
			}
		}

		// Map transcription config
		if session.Audio.Input.Transcription != nil {
			config.InputAudioTranscription = &InputAudioTranscriptionConfig{
				Model:    session.Audio.Input.Transcription.Model,
				Language: session.Audio.Input.Transcription.Language,
				Prompt:   session.Audio.Input.Transcription.Prompt,
			}
		}

		// Map turn detection (VAD)
		if session.Audio.Input.TurnDetection != nil {
			config.TurnDetection = &TurnDetectionConfig{
				Type:              session.Audio.Input.TurnDetection.Type,
				Threshold:         session.Audio.Input.TurnDetection.Threshold,
				PrefixPaddingMs:   session.Audio.Input.TurnDetection.PrefixPaddingMs,
				SilenceDurationMs: session.Audio.Input.TurnDetection.SilenceDurationMs,
			}
		}

		// Map noise reduction
		if session.Audio.Input.NoiseReduction != nil {
			config.InputAudioNoiseReduction = &InputAudioNoiseReductionConfig{
				Type: session.Audio.Input.NoiseReduction.Type,
			}
		}
	}

	return config
}

// ApplyToSession applies TranscriptionSessionConfig updates to a Session
// This converts from the flattened OpenAI format back to the nested structure
func (tsc *TranscriptionSessionConfig) ApplyToSession(session *Session) {
	// Ensure session type is transcription
	session.Type = "transcription"

	// Initialize audio config if needed
	if session.Audio == nil {
		session.Audio = &AudioConfig{}
	}
	if session.Audio.Input == nil {
		session.Audio.Input = &AudioInput{}
	}

	// Apply input audio format
	if tsc.InputAudioFormat != "" {
		if session.Audio.Input.Format == nil {
			session.Audio.Input.Format = &AudioFormat{Rate: 24000}
		}
		switch tsc.InputAudioFormat {
		case "pcm16":
			session.Audio.Input.Format.Type = "audio/pcm"
		case "g711_ulaw":
			session.Audio.Input.Format.Type = "audio/pcmu"
		case "g711_alaw":
			session.Audio.Input.Format.Type = "audio/pcma"
		}
	}

	// Apply transcription config
	if tsc.InputAudioTranscription != nil {
		if session.Audio.Input.Transcription == nil {
			session.Audio.Input.Transcription = &TranscriptionConfig{}
		}
		if tsc.InputAudioTranscription.Model != "" {
			session.Audio.Input.Transcription.Model = tsc.InputAudioTranscription.Model
		}
		if tsc.InputAudioTranscription.Language != "" {
			session.Audio.Input.Transcription.Language = tsc.InputAudioTranscription.Language
		}
		if tsc.InputAudioTranscription.Prompt != "" {
			session.Audio.Input.Transcription.Prompt = tsc.InputAudioTranscription.Prompt
		}
	}

	// Apply turn detection (VAD)
	if tsc.TurnDetection != nil {
		if session.Audio.Input.TurnDetection == nil {
			session.Audio.Input.TurnDetection = &TurnDetection{}
		}
		if tsc.TurnDetection.Type != "" {
			session.Audio.Input.TurnDetection.Type = tsc.TurnDetection.Type
		}
		if tsc.TurnDetection.Threshold > 0 {
			session.Audio.Input.TurnDetection.Threshold = tsc.TurnDetection.Threshold
		}
		if tsc.TurnDetection.PrefixPaddingMs > 0 {
			session.Audio.Input.TurnDetection.PrefixPaddingMs = tsc.TurnDetection.PrefixPaddingMs
		}
		if tsc.TurnDetection.SilenceDurationMs > 0 {
			session.Audio.Input.TurnDetection.SilenceDurationMs = tsc.TurnDetection.SilenceDurationMs
		}
	}

	// Apply noise reduction
	if tsc.InputAudioNoiseReduction != nil {
		if session.Audio.Input.NoiseReduction == nil {
			session.Audio.Input.NoiseReduction = &NoiseReduction{}
		}
		if tsc.InputAudioNoiseReduction.Type != "" {
			session.Audio.Input.NoiseReduction.Type = tsc.InputAudioNoiseReduction.Type
		}
	}

	// Apply include
	if len(tsc.Include) > 0 {
		session.Include = tsc.Include
	}
}

package domain

import "context"

// TranscriptionChunk represents a piece of transcription result
type TranscriptionChunk struct {
	Text     string    `json:"text"`
	IsFinal  bool      `json:"is_final"`
	StartMs  int       `json:"start_ms,omitempty"`
	EndMs    int       `json:"end_ms,omitempty"`
	Logprobs []Logprob `json:"logprobs,omitempty"`
}

// Logprob represents log probability information for transcription
type Logprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []byte  `json:"bytes,omitempty"`
}

// TranscriptionResult represents the complete transcription result
type TranscriptionResult struct {
	ItemID       string               `json:"item_id"`
	ContentIndex int                  `json:"content_index"`
	Transcript   string               `json:"transcript"`
	Chunks       []TranscriptionChunk `json:"chunks,omitempty"`
	Usage        *Usage               `json:"usage,omitempty"`
	Error        error                `json:"-"`
}

// ASRProvider defines the interface for speech-to-text backends
type ASRProvider interface {
	// Transcribe processes audio data and returns transcription results
	// The channel streams TranscriptionChunk for real-time updates
	// The final chunk will have IsFinal=true
	Transcribe(ctx context.Context, audio []byte, config *TranscriptionConfig) (<-chan TranscriptionChunk, error)

	// TranscribeStream processes audio data in streaming mode
	// Audio chunks are sent to the input channel, results come from output channel
	TranscribeStream(ctx context.Context, config *TranscriptionConfig) (audioIn chan<- []byte, resultOut <-chan TranscriptionChunk, err error)

	// GetSupportedModels returns list of supported ASR models
	GetSupportedModels() []string

	// GetSupportedLanguages returns list of supported language codes
	GetSupportedLanguages() []string

	// Close releases any resources held by the provider
	Close() error
}

// ASRConfig holds configuration for ASR provider initialization
type ASRConfig struct {
	Provider string                 // "sherpa-onnx"
	APIKey   string                 // API key if required
	Endpoint string                 // Custom endpoint if applicable
	Model    string                 // Default model to use
	Language string                 // Default language
	Options  map[string]interface{} // Provider-specific options
}

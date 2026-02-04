package domain

import (
	"sync"
	"time"
)

// AudioConfig represents audio input/output configuration
type AudioConfig struct {
	Input  *AudioInput  `json:"input"`
	Output *AudioOutput `json:"output"`
}

// AudioInput represents input audio configuration
type AudioInput struct {
	Format         *AudioFormat         `json:"format"`
	Transcription  *TranscriptionConfig `json:"transcription"`   // null or settings
	NoiseReduction *NoiseReduction      `json:"noise_reduction"` // null or settings
	TurnDetection  *TurnDetection       `json:"turn_detection"`
}

// TranscriptionConfig represents transcription settings for STT
type TranscriptionConfig struct {
	Model    string `json:"model"`              // "gpt-4o-transcribe", "gpt-4o-mini-transcribe"
	Language string `json:"language,omitempty"` // ISO-639-1 code like "en"
	Prompt   string `json:"prompt,omitempty"`   // Optional prompt to guide transcription
}

// NoiseReduction represents noise reduction settings
type NoiseReduction struct {
	Type string `json:"type"` // "near_field", "far_field", or null to disable
}

// AudioOutput represents output audio configuration
type AudioOutput struct {
	Format *AudioFormat `json:"format"`
	Voice  string       `json:"voice"` // "alloy", "echo", "fable", "onyx", "nova", "shimmer"
	Speed  float64      `json:"speed"` // 0.5-2.0, default 1.0
}

// AudioFormat represents audio format specification
type AudioFormat struct {
	Type string `json:"type"` // "audio/pcm"
	Rate int    `json:"rate"` // 24000, 16000, etc
}

// TurnDetection represents VAD (Voice Activity Detection) settings
type TurnDetection struct {
	Type              string      `json:"type"`                // "server_vad", "client_vad", or null
	Threshold         float64     `json:"threshold"`           // 0.0-1.0
	PrefixPaddingMs   int         `json:"prefix_padding_ms"`   // milliseconds
	SilenceDurationMs int         `json:"silence_duration_ms"` // milliseconds
	IdleTimeoutMs     interface{} `json:"idle_timeout_ms"`     // null or milliseconds
	CreateResponse    bool        `json:"create_response"`     // auto-create response after speech
	InterruptResponse bool        `json:"interrupt_response"`  // interrupt on new speech
}

// ErrBufferFull is returned when audio buffer exceeds max size
var ErrBufferFull = &BufferFullError{}

// BufferFullError indicates the audio buffer has reached its size limit
type BufferFullError struct{}

func (e *BufferFullError) Error() string {
	return "audio buffer size limit exceeded"
}

// AudioBuffer represents the input audio buffer
type AudioBuffer struct {
	Data          []byte
	Lock          chan struct{} // simple lock mechanism
	committed     bool
	mu            sync.Mutex
	startTime     time.Time
	speechStartMs int
	speechEndMs   int
	maxSize       int // maximum buffer size in bytes, 0 means unlimited
}

// Append adds audio data to the buffer
// Returns error if buffer would exceed max size
func (ab *AudioBuffer) Append(data []byte) error {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	// Check if adding data would exceed max size
	if ab.maxSize > 0 && len(ab.Data)+len(data) > ab.maxSize {
		return ErrBufferFull
	}

	ab.Data = append(ab.Data, data...)
	if ab.startTime.IsZero() {
		ab.startTime = time.Now()
	}
	return nil
}

// SetMaxSize sets the maximum buffer size
func (ab *AudioBuffer) SetMaxSize(size int) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.maxSize = size
}

// GetMaxSize returns the maximum buffer size
func (ab *AudioBuffer) GetMaxSize() int {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	return ab.maxSize
}

// Commit returns the buffer data and marks as committed
func (ab *AudioBuffer) Commit() []byte {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	data := make([]byte, len(ab.Data))
	copy(data, ab.Data)
	ab.committed = true
	return data
}

// Clear empties the buffer
func (ab *AudioBuffer) Clear() {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.Data = make([]byte, 0)
	ab.committed = false
	ab.startTime = time.Time{}
	ab.speechStartMs = 0
	ab.speechEndMs = 0
}

// GetSize returns the current buffer size
func (ab *AudioBuffer) GetSize() int {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	return len(ab.Data)
}

// IsEmpty checks if buffer is empty
func (ab *AudioBuffer) IsEmpty() bool {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	return len(ab.Data) == 0
}

// IsCommitted returns whether buffer has been committed
func (ab *AudioBuffer) IsCommitted() bool {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	return ab.committed
}

// GetData returns a copy of buffer data
func (ab *AudioBuffer) GetData() []byte {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	data := make([]byte, len(ab.Data))
	copy(data, ab.Data)
	return data
}

// SetSpeechTimings sets speech start and end times in milliseconds
func (ab *AudioBuffer) SetSpeechTimings(startMs, endMs int) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.speechStartMs = startMs
	ab.speechEndMs = endMs
}

// GetSpeechTimings returns speech start and end times
func (ab *AudioBuffer) GetSpeechTimings() (int, int) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	return ab.speechStartMs, ab.speechEndMs
}

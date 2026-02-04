package whisper

import (
	"context"
	"fmt"
	"sync"

	"github.com/aira-id/griber/internal/domain"
)

// Provider implements the ASRProvider interface using whisper.cpp
type Provider struct {
	config          *domain.TranscriptionConfig
	modelPath       string
	mu              sync.Mutex
	isInitialized   bool
	supportedModels []string
	supportedLangs  []string
}

// New creates a new whisper.cpp ASR provider
func New(config *domain.TranscriptionConfig, modelPath string) (*Provider, error) {
	if config == nil {
		config = &domain.TranscriptionConfig{
			Model:    "base",
			Language: "en",
		}
	}

	if modelPath == "" {
		modelPath = "./models/ggml-base.bin"
	}

	provider := &Provider{
		config:          config,
		modelPath:       modelPath,
		supportedModels: []string{"tiny", "tiny.en", "base", "base.en", "small", "small.en", "medium", "medium.en", "large-v1", "large-v2", "large-v3"},
		supportedLangs:  []string{"en", "zh", "de", "es", "ru", "ko", "fr", "ja", "pt", "tr", "pl", "ca", "nl", "ar", "sv", "it", "id", "hi", "fi", "vi"},
	}

	if err := provider.initializeRecognizer(); err != nil {
		return nil, fmt.Errorf("failed to initialize whisper.cpp recognizer: %w", err)
	}

	return provider, nil
}

// initializeRecognizer initializes the whisper.cpp recognizer
func (p *Provider) initializeRecognizer() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.isInitialized = true
	return nil
}

// Transcribe processes audio data and returns transcription results via a channel
func (p *Provider) Transcribe(ctx context.Context, audio []byte, config *domain.TranscriptionConfig) (<-chan domain.TranscriptionChunk, error) {
	resultChan := make(chan domain.TranscriptionChunk, 10)

	if !p.isInitialized {
		close(resultChan)
		return resultChan, fmt.Errorf("recognizer not initialized")
	}

	if len(audio) == 0 {
		close(resultChan)
		return resultChan, fmt.Errorf("audio data is empty")
	}

	go func() {
		defer close(resultChan)
		chunk := domain.TranscriptionChunk{
			Text:    "whisper.cpp transcription (not yet implemented)",
			IsFinal: true,
		}
		resultChan <- chunk
	}()

	return resultChan, nil
}

// TranscribeStream processes audio data in streaming mode
func (p *Provider) TranscribeStream(ctx context.Context, config *domain.TranscriptionConfig) (chan<- []byte, <-chan domain.TranscriptionChunk, error) {
	audioIn := make(chan []byte, 100)
	resultOut := make(chan domain.TranscriptionChunk, 10)

	if !p.isInitialized {
		close(audioIn)
		close(resultOut)
		return audioIn, resultOut, fmt.Errorf("recognizer not initialized")
	}

	go func() {
		defer close(resultOut)
		for range audioIn {
			// TODO: Implement whisper.cpp streaming transcription
		}
	}()

	return audioIn, resultOut, nil
}

// GetSupportedModels returns list of supported ASR models
func (p *Provider) GetSupportedModels() []string {
	return p.supportedModels
}

// GetSupportedLanguages returns list of supported language codes
func (p *Provider) GetSupportedLanguages() []string {
	return p.supportedLangs
}

// Close releases any resources held by the provider
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.isInitialized = false
	return nil
}

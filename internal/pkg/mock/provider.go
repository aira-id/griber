package mock

import (
	"context"
	"strings"
	"time"

	"github.com/aira-id/griber/internal/domain"
)

// Provider is a mock implementation of ASRProvider for testing
type Provider struct {
	delay       time.Duration
	chunkDelay  time.Duration
	mockResults []string
}

// New creates a new mock ASR provider
func New() *Provider {
	return &Provider{
		delay:      100 * time.Millisecond,
		chunkDelay: 50 * time.Millisecond,
		mockResults: []string{
			"Hello",
			", this is",
			" a test",
			" transcription",
			".",
		},
	}
}

// Transcribe implements ASRProvider.Transcribe
func (m *Provider) Transcribe(ctx context.Context, audio []byte, config *domain.TranscriptionConfig) (<-chan domain.TranscriptionChunk, error) {
	resultChan := make(chan domain.TranscriptionChunk, len(m.mockResults)+1)

	go func() {
		defer close(resultChan)

		// Simulate processing delay based on audio length
		select {
		case <-ctx.Done():
			return
		case <-time.After(m.delay):
		}

		// Stream mock transcription chunks
		var fullText strings.Builder
		for i, text := range m.mockResults {
			select {
			case <-ctx.Done():
				return
			default:
			}

			fullText.WriteString(text)
			isLast := i == len(m.mockResults)-1

			chunk := domain.TranscriptionChunk{
				Text:    text,
				IsFinal: isLast,
				StartMs: i * 100,
				EndMs:   (i + 1) * 100,
			}

			resultChan <- chunk

			if !isLast {
				time.Sleep(m.chunkDelay)
			}
		}
	}()

	return resultChan, nil
}

// TranscribeStream implements ASRProvider.TranscribeStream
func (m *Provider) TranscribeStream(ctx context.Context, config *domain.TranscriptionConfig) (chan<- []byte, <-chan domain.TranscriptionChunk, error) {
	audioIn := make(chan []byte, 100)
	resultOut := make(chan domain.TranscriptionChunk, 10)

	go func() {
		defer close(resultOut)

		var audioBuffer []byte
		silenceCount := 0
		const silenceThreshold = 3 // Number of empty reads to consider speech ended

		for {
			select {
			case <-ctx.Done():
				return
			case audio, ok := <-audioIn:
				if !ok {
					// Channel closed, process remaining audio
					if len(audioBuffer) > 0 {
						m.processAudioBuffer(ctx, audioBuffer, resultOut)
					}
					return
				}

				if len(audio) == 0 {
					silenceCount++
					if silenceCount >= silenceThreshold && len(audioBuffer) > 0 {
						// Process accumulated audio
						m.processAudioBuffer(ctx, audioBuffer, resultOut)
						audioBuffer = nil
						silenceCount = 0
					}
				} else {
					silenceCount = 0
					audioBuffer = append(audioBuffer, audio...)
				}
			}
		}
	}()

	return audioIn, resultOut, nil
}

func (m *Provider) processAudioBuffer(ctx context.Context, audio []byte, out chan<- domain.TranscriptionChunk) {
	// Generate mock transcription based on audio size
	// In reality, this would call the actual ASR service
	words := []string{"This", " is", " transcribed", " audio", "."}

	var fullText strings.Builder
	for i, word := range words {
		select {
		case <-ctx.Done():
			return
		default:
		}

		fullText.WriteString(word)
		isLast := i == len(words)-1

		chunk := domain.TranscriptionChunk{
			Text:    word,
			IsFinal: isLast,
			StartMs: i * 200,
			EndMs:   (i + 1) * 200,
		}

		select {
		case out <- chunk:
		case <-ctx.Done():
			return
		}

		if !isLast {
			time.Sleep(m.chunkDelay)
		}
	}
}

// GetSupportedModels implements ASRProvider.GetSupportedModels
func (m *Provider) GetSupportedModels() []string {
	return []string{"mock-transcribe"}
}

// GetSupportedLanguages implements ASRProvider.GetSupportedLanguages
func (m *Provider) GetSupportedLanguages() []string {
	return []string{"en"}
}

// Close implements ASRProvider.Close
func (m *Provider) Close() error {
	return nil
}

// SetMockResults allows setting custom mock transcription results
func (m *Provider) SetMockResults(results []string) {
	m.mockResults = results
}

// SetDelay allows setting custom delays for testing
func (m *Provider) SetDelay(initial, chunk time.Duration) {
	m.delay = initial
	m.chunkDelay = chunk
}

package vad

import (
	"context"
	"encoding/binary"
	"math"
	"sync"

	"github.com/aira-id/griber/internal/domain"
)

// SimpleVADProvider implements a basic energy-based VAD
type SimpleVADProvider struct {
	config        *domain.VADConfig
	events        chan domain.VADEvent
	mu            sync.Mutex
	isSpeaking    bool
	silentSamples int
	audioBuffer   []byte
	startMs       int
	currentMs     int
	ctx           context.Context
	cancel        context.CancelFunc
	closed        bool
	closeMu       sync.RWMutex
}

// NewSimpleVADProvider creates a new simple VAD provider
func NewSimpleVADProvider(config *domain.VADConfig) *SimpleVADProvider {
	if config == nil {
		config = domain.NewDefaultVADConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &SimpleVADProvider{
		config:      config,
		events:      make(chan domain.VADEvent, 10),
		audioBuffer: make([]byte, 0),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// ProcessAudio processes audio data and detects voice activity
func (v *SimpleVADProvider) ProcessAudio(ctx context.Context, audio []byte) error {
	// Check if closed before processing
	v.closeMu.RLock()
	if v.closed {
		v.closeMu.RUnlock()
		return nil
	}
	v.closeMu.RUnlock()

	v.mu.Lock()
	defer v.mu.Unlock()

	if len(audio) == 0 {
		return nil
	}

	// Calculate RMS energy of the audio
	energy := v.calculateEnergy(audio)

	// Convert threshold to energy level (threshold is 0-1, energy is typically 0-32768 for 16-bit audio)
	energyThreshold := v.config.Threshold * 1000 // Simplified threshold mapping

	// Calculate duration of this audio chunk in milliseconds
	// Assuming 16-bit PCM mono audio
	bytesPerSample := 2
	samplesInChunk := len(audio) / bytesPerSample
	chunkDurationMs := (samplesInChunk * 1000) / v.config.SampleRate

	wasSpeaking := v.isSpeaking

	if energy > energyThreshold {
		// Speech detected
		v.silentSamples = 0

		if !v.isSpeaking {
			// Speech just started
			v.isSpeaking = true
			v.startMs = v.currentMs

			// Include prefix padding
			prefixStart := v.startMs - v.config.PrefixPaddingMs
			if prefixStart < 0 {
				prefixStart = 0
			}

			event := domain.VADEvent{
				Type:    domain.VADEventSpeechStarted,
				StartMs: prefixStart,
			}

			v.sendEvent(event)
		}

		// Accumulate audio data during speech
		v.audioBuffer = append(v.audioBuffer, audio...)
	} else {
		// Silence detected
		if v.isSpeaking {
			v.silentSamples += chunkDurationMs

			// Still accumulate audio during silence (might be pause in speech)
			v.audioBuffer = append(v.audioBuffer, audio...)

			if v.silentSamples >= v.config.SilenceDurationMs {
				// Speech ended
				v.isSpeaking = false

				event := domain.VADEvent{
					Type:      domain.VADEventSpeechStopped,
					StartMs:   v.startMs,
					EndMs:     v.currentMs,
					AudioData: v.audioBuffer,
				}

				v.sendEvent(event)

				// Clear buffer after speech segment
				v.audioBuffer = make([]byte, 0)
			}
		}
	}

	v.currentMs += chunkDurationMs

	// Handle idle timeout
	if v.config.IdleTimeoutMs > 0 && !wasSpeaking && !v.isSpeaking {
		if v.currentMs >= v.config.IdleTimeoutMs {
			event := domain.VADEvent{
				Type:    domain.VADEventTimeout,
				StartMs: 0,
				EndMs:   v.currentMs,
			}

			v.sendEvent(event)
		}
	}

	return nil
}

// calculateEnergy calculates RMS energy of 16-bit PCM audio
func (v *SimpleVADProvider) calculateEnergy(audio []byte) float64 {
	if len(audio) < 2 {
		return 0
	}

	var sumSquares float64
	sampleCount := len(audio) / 2

	for i := 0; i < len(audio)-1; i += 2 {
		sample := int16(binary.LittleEndian.Uint16(audio[i : i+2]))
		sumSquares += float64(sample) * float64(sample)
	}

	if sampleCount == 0 {
		return 0
	}

	rms := math.Sqrt(sumSquares / float64(sampleCount))
	return rms
}

// GetEvents returns the channel for VAD events
func (v *SimpleVADProvider) GetEvents() <-chan domain.VADEvent {
	return v.events
}

// sendEvent safely sends an event to the channel if not closed
func (v *SimpleVADProvider) sendEvent(event domain.VADEvent) {
	v.closeMu.RLock()
	defer v.closeMu.RUnlock()

	if v.closed {
		return
	}

	select {
	case v.events <- event:
	default:
		// Channel full, skip event
	}
}

// Configure updates VAD settings
func (v *SimpleVADProvider) Configure(config *domain.VADConfig) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if config != nil {
		v.config = config
	}
	return nil
}

// Reset clears internal state
func (v *SimpleVADProvider) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.isSpeaking = false
	v.silentSamples = 0
	v.audioBuffer = make([]byte, 0)
	v.startMs = 0
	v.currentMs = 0
}

// Close releases resources
func (v *SimpleVADProvider) Close() error {
	v.closeMu.Lock()
	if v.closed {
		v.closeMu.Unlock()
		return nil
	}
	v.closed = true
	v.closeMu.Unlock()

	v.cancel()

	// Drain any remaining events before closing
	go func() {
		for range v.events {
			// Drain events
		}
	}()
	close(v.events)

	return nil
}

// IsSpeaking returns whether speech is currently detected
func (v *SimpleVADProvider) IsSpeaking() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.isSpeaking
}

// GetCurrentMs returns the current timestamp in milliseconds
func (v *SimpleVADProvider) GetCurrentMs() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.currentMs
}

// ForceCommit forces the current audio buffer to be committed
// This is useful for manual commit when VAD is disabled
func (v *SimpleVADProvider) ForceCommit() *domain.VADEvent {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(v.audioBuffer) == 0 {
		return nil
	}

	event := &domain.VADEvent{
		Type:      domain.VADEventSpeechStopped,
		StartMs:   v.startMs,
		EndMs:     v.currentMs,
		AudioData: v.audioBuffer,
	}

	v.audioBuffer = make([]byte, 0)
	v.isSpeaking = false
	v.startMs = v.currentMs

	return event
}

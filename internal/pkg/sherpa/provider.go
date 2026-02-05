package sherpa

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/aira-id/griber/internal/domain"
	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// RecognizerType represents the type of recognizer
type RecognizerType string

const (
	RecognizerOnline  RecognizerType = "online"  // Streaming/realtime recognition
	RecognizerOffline RecognizerType = "offline" // Batch/non-streaming recognition
)

// Config holds sherpa-onnx specific configuration
type Config struct {
	Provider   string         // cpu or gpu
	NumThreads int            // Number of threads for inference
	ModelsDir  string         // Base directory for models
	ModelName  string         // Model directory name
	ModelType  string         // Model type: "zipformer2", "whisper", etc.
	Recognizer RecognizerType // Type of recognizer: "online" or "offline"
	Encoder    string         // Encoder file name
	Decoder    string         // Decoder file name
	Joiner     string         // Joiner file name
	Tokens     string         // Tokens file name
	Languages  []string       // Supported languages
	Language   string         // Current language for transcription
}

// Provider implements the ASRProvider interface using sherpa-onnx
// Supports both OnlineRecognizer (streaming) and OfflineRecognizer (batch)
type Provider struct {
	config            *Config
	onlineRecognizer  *sherpa.OnlineRecognizer
	offlineRecognizer *sherpa.OfflineRecognizer
	mu                sync.Mutex
	recognizerType    RecognizerType
	isInitialized     bool
}

// New creates a new sherpa-onnx ASR provider
func New(config *Config) (*Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("sherpa config is required")
	}

	// Validate required fields
	if config.ModelName == "" {
		return nil, fmt.Errorf("model_name is required in sherpa config")
	}
	if config.Encoder == "" {
		return nil, fmt.Errorf("encoder is required in sherpa config")
	}
	if config.Decoder == "" {
		return nil, fmt.Errorf("decoder is required in sherpa config")
	}
	if config.Joiner == "" && config.ModelType != "whisper" {
		return nil, fmt.Errorf("joiner is required in sherpa config")
	}
	if config.Tokens == "" {
		return nil, fmt.Errorf("tokens is required in sherpa config")
	}
	if len(config.Languages) == 0 {
		return nil, fmt.Errorf("languages is required in sherpa config")
	}
	if config.Language == "" {
		return nil, fmt.Errorf("language is required in sherpa config")
	}

	// Validate language is supported
	if !config.IsLanguageSupported(config.Language) {
		return nil, fmt.Errorf("language '%s' is not supported by model '%s', supported languages: %v",
			config.Language, config.ModelName, config.Languages)
	}

	// Set defaults for optional fields
	if config.Provider == "" {
		config.Provider = "cpu"
	}
	if config.NumThreads == 0 {
		config.NumThreads = 4
	}
	if config.ModelsDir == "" {
		config.ModelsDir = "./models"
	}
	if config.Recognizer == "" {
		config.Recognizer = RecognizerOnline // Default to online
	}

	provider := &Provider{
		config:         config,
		recognizerType: config.Recognizer,
	}

	// Initialize the appropriate recognizer based on type
	switch config.Recognizer {
	case RecognizerOffline:
		if err := provider.initializeOfflineRecognizer(); err != nil {
			return nil, fmt.Errorf("failed to initialize sherpa-onnx offline recognizer: %w", err)
		}
	case RecognizerOnline:
		fallthrough
	default:
		if err := provider.initializeOnlineRecognizer(); err != nil {
			return nil, fmt.Errorf("failed to initialize sherpa-onnx online recognizer: %w", err)
		}
	}

	return provider, nil
}

// IsLanguageSupported checks if the given language is supported by this config
func (c *Config) IsLanguageSupported(lang string) bool {
	for _, l := range c.Languages {
		if l == lang {
			return true
		}
	}
	return false
}

// initializeOnlineRecognizer initializes the sherpa-onnx online (streaming) recognizer
func (p *Provider) initializeOnlineRecognizer() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Printf("Initializing sherpa-onnx online recognizer with model: %s (language: %s)",
		p.config.ModelName, p.config.Language)

	recognizerConfig := &sherpa.OnlineRecognizerConfig{}
	recognizerConfig.FeatConfig.SampleRate = 16000
	recognizerConfig.FeatConfig.FeatureDim = 80

	// Build model paths from config
	modelDir := filepath.Join(p.config.ModelsDir, p.config.ModelName)
	recognizerConfig.ModelConfig.Transducer.Encoder = filepath.Join(modelDir, p.config.Encoder)
	recognizerConfig.ModelConfig.Transducer.Decoder = filepath.Join(modelDir, p.config.Decoder)
	recognizerConfig.ModelConfig.Transducer.Joiner = filepath.Join(modelDir, p.config.Joiner)
	recognizerConfig.ModelConfig.Tokens = filepath.Join(modelDir, p.config.Tokens)

	recognizerConfig.ModelConfig.NumThreads = p.config.NumThreads
	recognizerConfig.ModelConfig.Provider = p.config.Provider
	recognizerConfig.ModelConfig.Debug = 0

	// Set decoding method based on model type if needed
	recognizerConfig.DecodingMethod = "greedy_search"
	recognizerConfig.MaxActivePaths = 4

	log.Printf("Online model paths: encoder=%s, decoder=%s, joiner=%s, tokens=%s",
		recognizerConfig.ModelConfig.Transducer.Encoder,
		recognizerConfig.ModelConfig.Transducer.Decoder,
		recognizerConfig.ModelConfig.Transducer.Joiner,
		recognizerConfig.ModelConfig.Tokens)

	p.onlineRecognizer = sherpa.NewOnlineRecognizer(recognizerConfig)
	if p.onlineRecognizer == nil {
		err := fmt.Errorf("sherpa.NewOnlineRecognizer returned nil - check model paths and library compatibility")
		log.Printf("[ERROR] %v", err)
		return err
	}

	p.isInitialized = true
	p.recognizerType = RecognizerOnline
	log.Printf("Sherpa-onnx online recognizer initialized successfully with model: %s", p.config.ModelName)

	return nil
}

// initializeOfflineRecognizer initializes the sherpa-onnx offline (batch) recognizer
func (p *Provider) initializeOfflineRecognizer() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Printf("Initializing sherpa-onnx offline recognizer with model: %s (language: %s)",
		p.config.ModelName, p.config.Language)

	recognizerConfig := &sherpa.OfflineRecognizerConfig{}
	recognizerConfig.FeatConfig.SampleRate = 16000
	recognizerConfig.FeatConfig.FeatureDim = 80

	recognizerConfig.ModelConfig.NumThreads = p.config.NumThreads
	recognizerConfig.ModelConfig.Provider = p.config.Provider
	recognizerConfig.ModelConfig.Debug = 0

	// Build model paths from config
	modelDir := filepath.Join(p.config.ModelsDir, p.config.ModelName)

	// Configure model paths based on type
	if p.config.ModelType == "whisper" {
		log.Printf("Configuring Whisper offline model path...")
		recognizerConfig.ModelConfig.Whisper.Encoder = filepath.Join(modelDir, p.config.Encoder)
		recognizerConfig.ModelConfig.Whisper.Decoder = filepath.Join(modelDir, p.config.Decoder)
		recognizerConfig.ModelConfig.Tokens = filepath.Join(modelDir, p.config.Tokens)

		log.Printf("Offline model paths (Whisper): encoder=%s, decoder=%s, tokens=%s",
			recognizerConfig.ModelConfig.Whisper.Encoder,
			recognizerConfig.ModelConfig.Whisper.Decoder,
			recognizerConfig.ModelConfig.Tokens)
	} else {
		// Default to Transducer
		recognizerConfig.ModelConfig.Transducer.Encoder = filepath.Join(modelDir, p.config.Encoder)
		recognizerConfig.ModelConfig.Transducer.Decoder = filepath.Join(modelDir, p.config.Decoder)
		recognizerConfig.ModelConfig.Transducer.Joiner = filepath.Join(modelDir, p.config.Joiner)
		recognizerConfig.ModelConfig.Tokens = filepath.Join(modelDir, p.config.Tokens)

		log.Printf("Offline model paths (Transducer): encoder=%s, decoder=%s, joiner=%s, tokens=%s",
			recognizerConfig.ModelConfig.Transducer.Encoder,
			recognizerConfig.ModelConfig.Transducer.Decoder,
			recognizerConfig.ModelConfig.Transducer.Joiner,
			recognizerConfig.ModelConfig.Tokens)
	}

	p.offlineRecognizer = sherpa.NewOfflineRecognizer(recognizerConfig)
	if p.offlineRecognizer == nil {
		err := fmt.Errorf("sherpa.NewOfflineRecognizer returned nil - check model paths and library compatibility")
		log.Printf("[ERROR] %v", err)
		return err
	}

	p.isInitialized = true
	p.recognizerType = RecognizerOffline
	log.Printf("Sherpa-onnx offline recognizer initialized successfully with model: %s", p.config.ModelName)

	return nil
}

// IsOffline returns true if this provider uses offline recognizer
func (p *Provider) IsOffline() bool {
	return p.recognizerType == RecognizerOffline
}

// IsOnline returns true if this provider uses online recognizer
func (p *Provider) IsOnline() bool {
	return p.recognizerType == RecognizerOnline
}

// Transcribe processes audio data and returns transcription results via a channel
func (p *Provider) Transcribe(ctx context.Context, audio []byte, config *domain.TranscriptionConfig) (<-chan domain.TranscriptionChunk, error) {
	if !p.isInitialized {
		resultChan := make(chan domain.TranscriptionChunk, 10)
		close(resultChan)
		return resultChan, fmt.Errorf("recognizer not initialized")
	}

	if len(audio) == 0 {
		resultChan := make(chan domain.TranscriptionChunk, 10)
		close(resultChan)
		return resultChan, fmt.Errorf("audio data is empty")
	}

	// Use appropriate recognizer based on type
	if p.recognizerType == RecognizerOffline {
		return p.transcribeOffline(ctx, audio, config)
	}
	return p.transcribeOnline(ctx, audio, config)
}

// transcribeOffline processes audio using the offline (batch) recognizer
func (p *Provider) transcribeOffline(ctx context.Context, audio []byte, config *domain.TranscriptionConfig) (<-chan domain.TranscriptionChunk, error) {
	resultChan := make(chan domain.TranscriptionChunk, 10)

	go func() {
		defer close(resultChan)

		// Create stream - we need lock here to safely access p.offlineRecognizer
		p.mu.Lock()
		if p.offlineRecognizer == nil {
			p.mu.Unlock()
			log.Printf("Error: offlineRecognizer is nil")
			return
		}
		stream := sherpa.NewOfflineStream(p.offlineRecognizer)
		p.mu.Unlock()

		if stream == nil {
			log.Printf("Error: failed to create OfflineStream")
			return
		}
		defer sherpa.DeleteOfflineStream(stream)

		// Convert bytes to float32 samples
		samples := bytesToFloat32(audio)

		// Accept waveform (does not need global lock, stream is local)
		stream.AcceptWaveform(16000, samples)

		// Check context before decoding
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Decode - The recognizer is thread-safe for decoding different streams
		p.offlineRecognizer.Decode(stream)

		// Get result
		result := stream.GetResult()

		// Send final result
		if result.Text != "" {
			durationMs := len(samples) * 1000 / 16000

			// Map tokens/timestamps
			var words []domain.Word
			// tokens are usually []string, timestamps are []float32
			if len(result.Tokens) > 0 && len(result.Timestamps) == len(result.Tokens) {
				for i, token := range result.Tokens {
					// Use the timestamp for both start and end approximation
					// since offline result typically gives just "timestamp" per token.
					// We'll estimate duration or use point timestamps.
					// Sherpa offline timestamp is usually the end time of the token.

					endTimeMs := int(result.Timestamps[i] * 1000)
					startTimeMs := 0
					if i > 0 {
						startTimeMs = int(result.Timestamps[i-1] * 1000)
					}

					words = append(words, domain.Word{
						Word:    token,
						StartMs: startTimeMs,
						EndMs:   endTimeMs,
					})
				}
			}

			finalChunk := domain.TranscriptionChunk{
				Text:    result.Text,
				IsFinal: true,
				StartMs: 0,
				EndMs:   durationMs,
				Words:   words,
			}

			select {
			case <-ctx.Done():
				return
			case resultChan <- finalChunk:
			}
			log.Printf("Offline transcription completed: %s", result.Text)
		}
	}()

	return resultChan, nil
}

// Pre-allocated silence buffers to avoid repeated allocations
var (
	leftPaddingSilence  = make([]float32, 4800) // 0.3 seconds at 16kHz
	rightPaddingSilence = make([]float32, 9600) // 0.6 seconds at 16kHz
)

// transcribeOnline processes audio using the online (streaming) recognizer
func (p *Provider) transcribeOnline(ctx context.Context, audio []byte, config *domain.TranscriptionConfig) (<-chan domain.TranscriptionChunk, error) {
	resultChan := make(chan domain.TranscriptionChunk, 10)

	go func() {
		defer close(resultChan)

		// Only lock during stream creation - recognizer access
		p.mu.Lock()
		if p.onlineRecognizer == nil {
			p.mu.Unlock()
			log.Printf("Error: onlineRecognizer is nil")
			return
		}
		stream := sherpa.NewOnlineStream(p.onlineRecognizer)
		p.mu.Unlock()

		if stream == nil {
			log.Printf("Error: failed to create OnlineStream")
			return
		}
		defer sherpa.DeleteOnlineStream(stream)

		// Convert bytes to float32 samples (outside lock)
		samples := bytesToFloat32(audio)

		// Add left padding (0.3 seconds of silence) - use pre-allocated buffer
		stream.AcceptWaveform(16000, leftPaddingSilence)

		// Process the audio
		stream.AcceptWaveform(16000, samples)

		// Add right padding (0.6 seconds of silence) - use pre-allocated buffer
		stream.AcceptWaveform(16000, rightPaddingSilence)

		// Input finished
		stream.InputFinished()

		// Decode loop - only lock during recognizer operations
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			p.mu.Lock()
			isReady := p.onlineRecognizer.IsReady(stream)
			if isReady {
				p.onlineRecognizer.Decode(stream)
			}
			p.mu.Unlock()

			if !isReady {
				break
			}
		}

		// Get final result - lock only during GetResult
		p.mu.Lock()
		result := p.onlineRecognizer.GetResult(stream)
		p.mu.Unlock()

		// Send final result
		if result != nil && result.Text != "" {
			finalChunk := domain.TranscriptionChunk{
				Text:    result.Text,
				IsFinal: true,
				StartMs: 0,
				EndMs:   len(samples) * 1000 / 16000,
			}
			select {
			case <-ctx.Done():
				return
			case resultChan <- finalChunk:
			}
			log.Printf("Online transcription completed: %s", result.Text)
		}
	}()

	return resultChan, nil
}

// TranscribeStream processes audio data in streaming mode
// Only available for online recognizer
func (p *Provider) TranscribeStream(ctx context.Context, config *domain.TranscriptionConfig) (chan<- []byte, <-chan domain.TranscriptionChunk, error) {
	audioIn := make(chan []byte, 100)
	resultOut := make(chan domain.TranscriptionChunk, 10)

	if !p.isInitialized {
		close(audioIn)
		close(resultOut)
		return audioIn, resultOut, fmt.Errorf("recognizer not initialized")
	}

	if p.recognizerType != RecognizerOnline {
		close(audioIn)
		close(resultOut)
		return audioIn, resultOut, fmt.Errorf("streaming requires online recognizer, but this provider uses offline recognizer")
	}

	go func() {
		defer close(resultOut)

		// Only lock during stream creation
		p.mu.Lock()
		if p.onlineRecognizer == nil {
			p.mu.Unlock()
			log.Printf("Error: onlineRecognizer is nil")
			return
		}
		stream := sherpa.NewOnlineStream(p.onlineRecognizer)
		p.mu.Unlock()

		if stream == nil {
			log.Printf("Error: failed to create OnlineStream")
			return
		}
		defer sherpa.DeleteOnlineStream(stream)

		var lastPartialResult string

		for {
			select {
			case <-ctx.Done():
				return

			case audio, ok := <-audioIn:
				if !ok {
					// Channel closed, finalize
					stream.InputFinished()

					// Finalize decoding with fine-grained locking
					for {
						p.mu.Lock()
						isReady := p.onlineRecognizer.IsReady(stream)
						if isReady {
							p.onlineRecognizer.Decode(stream)
						}
						p.mu.Unlock()
						if !isReady {
							break
						}
					}

					// Get final result
					p.mu.Lock()
					result := p.onlineRecognizer.GetResult(stream)
					p.mu.Unlock()

					// Send final result
					if result != nil && result.Text != "" && result.Text != lastPartialResult {
						chunk := domain.TranscriptionChunk{
							Text:    result.Text[len(lastPartialResult):],
							IsFinal: true,
						}
						select {
						case <-ctx.Done():
							return
						case resultOut <- chunk:
						}
					} else {
						// Send empty final chunk if no new text
						chunk := domain.TranscriptionChunk{
							Text:    "",
							IsFinal: true,
						}
						resultOut <- chunk
					}

					return
				}

				// Convert bytes to float32 samples (outside lock)
				samples := bytesToFloat32(audio)

				// Accept waveform (stream is local, no lock needed)
				stream.AcceptWaveform(16000, samples)

				// Decode if ready - fine-grained locking per operation
				for {
					p.mu.Lock()
					isReady := p.onlineRecognizer.IsReady(stream)
					if isReady {
						p.onlineRecognizer.Decode(stream)
					}
					p.mu.Unlock()
					if !isReady {
						break
					}
				}

				// Get current result
				p.mu.Lock()
				result := p.onlineRecognizer.GetResult(stream)
				p.mu.Unlock()

				// Send delta event if result changed
				if result != nil && result.Text != "" && result.Text != lastPartialResult {
					delta := result.Text[len(lastPartialResult):]
					if delta != "" {
						chunk := domain.TranscriptionChunk{
							Text:    delta,
							IsFinal: false,
						}
						select {
						case <-ctx.Done():
							return
						case resultOut <- chunk:
						}
						lastPartialResult = result.Text
					}
				}
			}
		}
	}()

	return audioIn, resultOut, nil
}

// GetSupportedModels returns list of supported ASR models
func (p *Provider) GetSupportedModels() []string {
	return []string{p.config.ModelName}
}

// GetSupportedLanguages returns list of supported language codes
func (p *Provider) GetSupportedLanguages() []string {
	return p.config.Languages
}

// Close releases any resources held by the provider
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.onlineRecognizer != nil {
		sherpa.DeleteOnlineRecognizer(p.onlineRecognizer)
		p.onlineRecognizer = nil
	}

	if p.offlineRecognizer != nil {
		sherpa.DeleteOfflineRecognizer(p.offlineRecognizer)
		p.offlineRecognizer = nil
	}

	p.isInitialized = false
	log.Printf("Sherpa-onnx provider closed")
	return nil
}

// bytesToFloat32 converts byte array (PCM 16-bit little-endian) to float32 array
func bytesToFloat32(data []byte) []float32 {
	numSamples := len(data) / 2
	samples := make([]float32, numSamples)

	for i := 0; i < numSamples; i++ {
		// Read 16-bit signed integer in little-endian
		b1 := int16(data[i*2])
		b2 := int16(data[i*2+1])
		sample := (b2 << 8) | (b1 & 0xFF)

		// Convert to float32 in range [-1, 1)
		samples[i] = float32(sample) / 32768.0
	}

	return samples
}

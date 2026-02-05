package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aira-id/griber/internal/config"
	"github.com/aira-id/griber/internal/domain"
	"github.com/aira-id/griber/internal/middleware"
	"github.com/aira-id/griber/internal/pkg/audio"
	"github.com/aira-id/griber/internal/usecase/asr"
)

// TranscriptionHandler handles HTTP transcription requests (OpenAI compatible)
type TranscriptionHandler struct {
	Config      *config.Config
	ASRRegistry *asr.ASRModelRegistry
	RateLimiter *middleware.RateLimiter
}

// NewTranscriptionHandler creates a new transcription handler
func NewTranscriptionHandler(cfg *config.Config, registry *asr.ASRModelRegistry) *TranscriptionHandler {
	return &TranscriptionHandler{
		Config:      cfg,
		ASRRegistry: registry,
		RateLimiter: middleware.NewRateLimiter(&cfg.Rate),
	}
}

// TranscriptionRequest represents the request parameters for transcription
// Compatible with OpenAI's /v1/audio/transcriptions API
type TranscriptionRequest struct {
	// File is the audio file to transcribe (required, handled via multipart)
	File []byte
	// Model is the ID of the model to use (required)
	Model string
	// Language is the language of the input audio (optional, ISO-639-1)
	Language string
	// Prompt is an optional text to guide the model's style (optional)
	Prompt string
	// ResponseFormat is the format of the response: json, text, srt, verbose_json, vtt (optional, default: json)
	ResponseFormat string
	// Temperature is the sampling temperature (optional, 0-1)
	Temperature float64
	// TimestampGranularities specifies the timestamp granularities (optional)
	TimestampGranularities []string
}

// TranscriptionResponse represents the transcription result
// Compatible with OpenAI's transcription response
type TranscriptionResponse struct {
	Text string `json:"text"`
}

// VerboseTranscriptionResponse represents a detailed transcription result
type VerboseTranscriptionResponse struct {
	Task     string    `json:"task"`
	Language string    `json:"language"`
	Duration float64   `json:"duration"`
	Text     string    `json:"text"`
	Words    []Word    `json:"words,omitempty"`
	Segments []Segment `json:"segments,omitempty"`
}

// Word represents a transcribed word with timing
type Word struct {
	Word  string  `json:"word"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Segment represents a transcription segment
type Segment struct {
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ServeHTTP implements http.Handler interface
func (h *TranscriptionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed", "")
		return
	}

	clientIP := middleware.GetClientIP(r)

	// Check rate limit
	if !h.RateLimiter.Allow(clientIP) {
		log.Printf("Rate limit exceeded for IP: %s", clientIP)
		h.sendError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests", "")
		return
	}

	// Validate API key
	if !h.validateAPIKey(r) {
		log.Printf("Invalid API key from IP: %s", clientIP)
		h.sendError(w, http.StatusUnauthorized, "invalid_api_key", "Invalid API key", "")
		return
	}

	// Parse the request
	req, err := h.parseRequest(r)
	if err != nil {
		log.Printf("Failed to parse request: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_request_error", err.Error(), "")
		return
	}

	// Validate required fields
	if req.Model == "" {
		h.sendError(w, http.StatusBadRequest, "invalid_request_error", "model is required", "model")
		return
	}
	if len(req.File) == 0 {
		h.sendError(w, http.StatusBadRequest, "invalid_request_error", "file is required", "file")
		return
	}

	// Set default language if not provided
	language := req.Language
	if language == "" {
		language = "en" // Default to English
	}

	// Get ASR provider from registry
	provider, err := h.ASRRegistry.GetModel(req.Model, language)
	if err != nil {
		log.Printf("Failed to get ASR model: %v", err)
		h.sendError(w, http.StatusBadRequest, "invalid_request_error", err.Error(), "model")
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), h.Config.Audio.TranscriptionTimeout)
	defer cancel()

	// Perform transcription
	transcriptionConfig := &domain.TranscriptionConfig{
		Model:                  req.Model,
		Language:               language,
		Prompt:                 req.Prompt,
		Temperature:            req.Temperature,
		TimestampGranularities: req.TimestampGranularities,
	}

	resultChan, err := provider.Transcribe(ctx, req.File, transcriptionConfig)
	if err != nil {
		log.Printf("Transcription failed: %v", err)
		h.sendError(w, http.StatusInternalServerError, "transcription_error", "Failed to transcribe audio", "")
		return
	}

	// Collect results
	var fullText strings.Builder
	var startMs, endMs int
	var allWords []Word

	for chunk := range resultChan {
		fullText.WriteString(chunk.Text)
		if chunk.StartMs > 0 && startMs == 0 {
			startMs = chunk.StartMs
		}
		if chunk.EndMs > endMs {
			endMs = chunk.EndMs
		}

		// Collect words from chunk
		if len(chunk.Words) > 0 {
			for _, w := range chunk.Words {
				allWords = append(allWords, Word{
					Word:  w.Word,
					Start: float64(w.StartMs) / 1000.0,
					End:   float64(w.EndMs) / 1000.0,
				})
			}
		}
	}

	// Check for context cancellation
	if ctx.Err() != nil {
		log.Printf("Transcription timed out")
		h.sendError(w, http.StatusGatewayTimeout, "timeout_error", "Transcription timed out", "")
		return
	}

	// Send response based on format
	responseFormat := req.ResponseFormat
	if responseFormat == "" {
		responseFormat = "json"
	}

	transcriptText := strings.TrimSpace(fullText.String())

	switch responseFormat {
	case "text":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(transcriptText))

	case "verbose_json":
		// Calculate duration from audio data (16-bit PCM at 16kHz)
		duration := float64(len(req.File)) / 2 / 16000

		// Create default segment if no words available or just one big segment
		segments := []Segment{}

		// If we have words, we can construct segments ?
		// For offline recognizer, we typically get one big chunk.
		// We'll wrap everything in one segment for now, but include the words.
		segments = append(segments, Segment{
			ID:               0,
			Seek:             0,
			Start:            float64(startMs) / 1000,
			End:              float64(endMs) / 1000,
			Text:             transcriptText,
			Tokens:           []int{}, // We don't have raw token IDs easily available yet
			Temperature:      req.Temperature,
			AvgLogprob:       0.0,
			CompressionRatio: 1.0,
			NoSpeechProb:     0.0,
		})

		response := VerboseTranscriptionResponse{
			Task:     "transcribe",
			Language: language,
			Duration: duration,
			Text:     transcriptText,
			Words:    allWords,
			Segments: segments,
		}
		h.sendJSON(w, http.StatusOK, response)

	case "srt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// Simple SRT format
		srt := fmt.Sprintf("1\n%s --> %s\n%s\n",
			formatSRTTime(startMs),
			formatSRTTime(endMs),
			transcriptText)
		w.Write([]byte(srt))

	case "vtt":
		w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// WebVTT format
		vtt := fmt.Sprintf("WEBVTT\n\n%s --> %s\n%s\n",
			formatVTTTime(startMs),
			formatVTTTime(endMs),
			transcriptText)
		w.Write([]byte(vtt))

	default: // json
		response := TranscriptionResponse{
			Text: transcriptText,
		}
		h.sendJSON(w, http.StatusOK, response)
	}

	log.Printf("Transcription completed: %d bytes audio -> %d chars text", len(req.File), len(transcriptText))
}

// parseRequest parses the multipart form request
func (h *TranscriptionHandler) parseRequest(r *http.Request) (*TranscriptionRequest, error) {
	// Parse multipart form (max 25MB for audio file, OpenAI's limit)
	maxSize := int64(25 * 1024 * 1024)
	if err := r.ParseMultipartForm(maxSize); err != nil {
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	req := &TranscriptionRequest{}

	// Get the audio file
	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	defer file.Close()

	// Read file content
	rawAudioData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Decode audio to required PCM format
	audioData, err := audio.Decode(rawAudioData)
	if err != nil {
		log.Printf("Audio decoding failed: %v", err)
		return nil, fmt.Errorf("failed to decode audio file: %w (ensure ffmpeg is installed)", err)
	}
	req.File = audioData

	// Parse other form fields
	req.Model = r.FormValue("model")
	req.Language = r.FormValue("language")
	req.Prompt = r.FormValue("prompt")
	req.ResponseFormat = r.FormValue("response_format")

	// Parse temperature
	if tempStr := r.FormValue("temperature"); tempStr != "" {
		var temp float64
		if _, err := fmt.Sscanf(tempStr, "%f", &temp); err == nil {
			req.Temperature = temp
		}
	}

	// Parse timestamp_granularities[] if provided
	if granularities := r.Form["timestamp_granularities[]"]; len(granularities) > 0 {
		req.TimestampGranularities = granularities
	}

	return req, nil
}

// validateAPIKey checks if the request has a valid API key
func (h *TranscriptionHandler) validateAPIKey(r *http.Request) bool {
	// Check Authorization header (Bearer token)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Support "Bearer <key>" format
		if strings.HasPrefix(authHeader, "Bearer ") {
			apiKey := strings.TrimPrefix(authHeader, "Bearer ")
			return h.Config.IsAPIKeyValid(apiKey)
		}
		// Also support raw key in Authorization header
		return h.Config.IsAPIKeyValid(authHeader)
	}

	// Check OpenAI-style header
	apiKey := r.Header.Get("OpenAI-Api-Key")
	if apiKey != "" {
		return h.Config.IsAPIKeyValid(apiKey)
	}

	// If no API keys configured, allow without auth
	return h.Config.IsAPIKeyValid("")
}

// sendJSON sends a JSON response
func (h *TranscriptionHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response in OpenAI format
func (h *TranscriptionHandler) sendError(w http.ResponseWriter, status int, errType, message, param string) {
	response := ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errType,
			Param:   param,
		},
	}
	h.sendJSON(w, status, response)
}

// Close cleans up handler resources
func (h *TranscriptionHandler) Close() {
	if h.RateLimiter != nil {
		h.RateLimiter.Close()
	}
}

// formatSRTTime formats milliseconds as SRT timestamp (HH:MM:SS,mmm)
func formatSRTTime(ms int) string {
	d := time.Duration(ms) * time.Millisecond
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	millis := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, millis)
}

// formatVTTTime formats milliseconds as WebVTT timestamp (HH:MM:SS.mmm)
func formatVTTTime(ms int) string {
	d := time.Duration(ms) * time.Millisecond
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	millis := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, millis)
}

package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aira-id/griber/internal/config"
	"github.com/aira-id/griber/internal/domain"
	"github.com/aira-id/griber/internal/usecase/asr"
	"github.com/aira-id/griber/internal/usecase/vad"
)

const (
	// Default models - should match config.yaml
	DefaultRealtimeModel      = "gpt-realtime-2025-08-28"
	DefaultTranscriptionModel = "gpt-4o-transcribe"
)

// Conn defines the interface for WebSocket connections
type Conn interface {
	WriteJSON(v interface{}) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
}

// SessionUsecase handles session business logic
type SessionUsecase struct {
	sessionManager       *SessionManager
	idGen                *IDGenerator
	asrRegistry          *asr.ASRModelRegistry // Registry for lazy model loading
	maxAudioBufferSize   int
	transcriptionTimeout time.Duration
}

// NewSessionUsecase creates a new session usecase (for testing, no config)
func NewSessionUsecase() *SessionUsecase {
	return &SessionUsecase{
		sessionManager:       NewSessionManager(),
		idGen:                NewIDGenerator(),
		asrRegistry:          nil,              // No registry without config
		maxAudioBufferSize:   15 * 1024 * 1024, // 15MB default
		transcriptionTimeout: 30 * time.Second,
	}
}

// NewSessionUsecaseWithConfig creates a session usecase with configuration
// Models are NOT loaded eagerly - they are loaded lazily on session.update
func NewSessionUsecaseWithConfig(cfg *config.Config) *SessionUsecase {
	// Create registry for lazy model loading (no models loaded yet)
	registry := asr.NewASRModelRegistry(&cfg.ASR)

	log.Printf("[INFO] ASR Model Registry initialized with %d available models (lazy loading enabled)",
		len(cfg.ASR.Models))
	for modelName := range cfg.ASR.Models {
		log.Printf("[INFO]   - %s", modelName)
	}

	return &SessionUsecase{
		sessionManager:       NewSessionManager(),
		idGen:                NewIDGenerator(),
		asrRegistry:          registry,
		maxAudioBufferSize:   cfg.Audio.MaxBufferSize,
		transcriptionTimeout: cfg.Audio.TranscriptionTimeout,
	}
}

// NewSessionUsecaseWithASR creates a session usecase with a custom ASR provider
func NewSessionUsecaseWithASR(asr domain.ASRProvider) *SessionUsecase {
	return &SessionUsecase{
		sessionManager:       NewSessionManager(),
		idGen:                NewIDGenerator(),
		asrRegistry:          nil,
		maxAudioBufferSize:   15 * 1024 * 1024, // 15MB default
		transcriptionTimeout: 30 * time.Second,
	}
}

// getOrCreateVAD gets or creates a VAD provider for a session
func (u *SessionUsecase) getOrCreateVAD(state *domain.SessionState) domain.VADProvider {
	if state.VADProvider != nil {
		return state.VADProvider
	}

	// Create VAD config from session config
	var vadConfig *domain.VADConfig
	if state.Config.Audio != nil && state.Config.Audio.Input != nil && state.Config.Audio.Input.TurnDetection != nil {
		vadConfig = domain.VADConfigFromTurnDetection(state.Config.Audio.Input.TurnDetection)
	} else {
		vadConfig = domain.NewDefaultVADConfig()
	}

	v := vad.NewSimpleVADProvider(vadConfig)
	state.VADProvider = v
	return v
}

// cleanupVAD cleans up the VAD provider for a session
func (u *SessionUsecase) cleanupVAD(state *domain.SessionState) {
	if state.VADProvider != nil {
		state.VADProvider.Close()
		state.VADProvider = nil
	}
}

// SessionIntent represents the type of session to create
type SessionIntent string

const (
	IntentRealtime      SessionIntent = "realtime"
	IntentTranscription SessionIntent = "transcription"
)

// HandleNewConnection handles a new WebSocket connection
// For transcription mode, pass intent="transcription"
func (u *SessionUsecase) HandleNewConnection(conn interface{}) {
	u.HandleNewConnectionWithIntent(conn, IntentRealtime)
}

// HandleNewConnectionWithIntent handles a new WebSocket connection with specified intent
// intent can be "realtime" (default) or "transcription"
func (u *SessionUsecase) HandleNewConnectionWithIntent(conn interface{}, intent SessionIntent) {
	wsConn, ok := conn.(Conn)
	if !ok {
		log.Println("Invalid connection type")
		return
	}

	// Create session and conversation
	sessionID := u.idGen.GenerateSessionID()
	conversationID := u.idGen.GenerateConversationID()

	var state *domain.SessionState
	if intent == IntentTranscription {
		// Create transcription-only session
		state = u.sessionManager.CreateTranscriptionSession(sessionID, DefaultTranscriptionModel, conversationID, "en")
	} else {
		// Create realtime session (default)
		state = u.sessionManager.CreateSession(sessionID, DefaultRealtimeModel, conversationID)
	}

	// Set audio buffer size limit
	if u.maxAudioBufferSize > 0 {
		state.AudioBuffer.SetMaxSize(u.maxAudioBufferSize)
	}

	// Send appropriate session.created event based on intent
	if intent == IntentTranscription {
		// Send transcription_session.created event with flattened format
		transcriptionSessionCreatedEvent := &domain.TranscriptionSessionCreatedEvent{
			BaseEvent: domain.BaseEvent{
				EventID: u.idGen.GenerateEventID(),
				Type:    domain.EventTranscriptionSessionCreated,
			},
			Session: domain.NewTranscriptionSessionConfig(state.Config),
		}

		if err := wsConn.WriteJSON(transcriptionSessionCreatedEvent); err != nil {
			log.Println("Error sending transcription_session.created:", err)
			return
		}
	} else {
		// Send session.created event
		sessionCreatedEvent := &domain.SessionCreatedEvent{
			BaseEvent: domain.BaseEvent{
				EventID: u.idGen.GenerateEventID(),
				Type:    domain.EventSessionCreated,
			},
			Session: state.Config,
		}

		if err := wsConn.WriteJSON(sessionCreatedEvent); err != nil {
			log.Println("Error sending session.created:", err)
			return
		}
	}

	// Message reading loop
	for {
		_, message, err := wsConn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		u.ProcessMessage(wsConn, state, message)
	}

	// Cleanup
	u.cleanupVAD(state)
	u.sessionManager.DeleteSession(sessionID)
}

// ProcessMessage processes incoming client events
func (u *SessionUsecase) ProcessMessage(conn Conn, state *domain.SessionState, message []byte) {
	var baseEvent domain.BaseEvent
	if err := json.Unmarshal(message, &baseEvent); err != nil {
		u.sendError(conn, "", "invalid_request_error", "invalid_json", "Failed to parse message", nil)
		return
	}

	log.Printf("Received event: %s", baseEvent.Type)

	switch baseEvent.Type {
	case domain.EventSessionUpdate:
		u.handleSessionUpdate(conn, state, message)

	case domain.EventInputAudioBufferAppend:
		u.handleInputAudioBufferAppend(conn, state, message)

	case domain.EventInputAudioBufferCommit:
		u.handleInputAudioBufferCommit(conn, state, message)

	case domain.EventInputAudioBufferClear:
		u.handleInputAudioBufferClear(conn, state, message)

	case domain.EventConversationItemCreate:
		u.handleConversationItemCreate(conn, state, message)

	case domain.EventConversationItemDelete:
		u.handleConversationItemDelete(conn, state, message)

	case domain.EventConversationItemTruncate:
		u.handleConversationItemTruncate(conn, state, message)

	case domain.EventResponseCreate:
		u.handleResponseCreate(conn, state, message)

	case domain.EventResponseCancel:
		u.handleResponseCancel(conn, state, message)

	case domain.EventTranscriptionSessionUpdate:
		u.handleTranscriptionSessionUpdate(conn, state, message)

	default:
		u.sendError(conn, baseEvent.EventID, "invalid_request_error", "unknown_event_type",
			fmt.Sprintf("Unknown event type: %s", baseEvent.Type), nil)
	}
}

// ============================================================================
// SESSION EVENT HANDLERS
// ============================================================================

func (u *SessionUsecase) handleSessionUpdate(conn Conn, state *domain.SessionState, message []byte) {
	var event domain.SessionUpdateClientEvent
	if err := json.Unmarshal(message, &event); err != nil {
		u.sendError(conn, "", "invalid_request_error", "invalid_event", "Failed to parse session.update", nil)
		return
	}

	if event.Session == nil {
		u.sendError(conn, event.EventID, "invalid_request_error", "missing_field", "session field is required", "session")
		return
	}

	// Check if transcription config is being updated (model/language change)
	if event.Session.Audio != nil && event.Session.Audio.Input != nil && event.Session.Audio.Input.Transcription != nil {
		transcription := event.Session.Audio.Input.Transcription
		if err := u.reconfigureASRProvider(conn, state, event.EventID, transcription.Model, transcription.Language); err != nil {
			// Error already sent to client
			return
		}
	}

	// Update session configuration
	updatedState, err := u.sessionManager.UpdateSession(state.ID, event.Session)
	if err != nil {
		u.sendError(conn, event.EventID, "server_error", "session_update_failed", err.Error(), nil)
		return
	}

	// Send session.updated event
	sessionUpdatedEvent := &domain.SessionUpdatedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventSessionUpdated,
		},
		Session: updatedState.Config,
	}

	conn.WriteJSON(sessionUpdatedEvent)
}

// handleTranscriptionSessionUpdate handles transcription_session.update client events
// This uses the flattened OpenAI Realtime Transcription API format
func (u *SessionUsecase) handleTranscriptionSessionUpdate(conn Conn, state *domain.SessionState, message []byte) {
	var event domain.TranscriptionSessionUpdateClientEvent
	if err := json.Unmarshal(message, &event); err != nil {
		u.sendError(conn, "", "invalid_request_error", "invalid_event", "Failed to parse transcription_session.update", nil)
		return
	}

	if event.Session == nil {
		u.sendError(conn, event.EventID, "invalid_request_error", "missing_field", "session field is required", "session")
		return
	}

	// Apply the flattened config to the internal session structure
	event.Session.ApplyToSession(state.Config)

	// Check if transcription config is being updated (model/language change)
	if event.Session.InputAudioTranscription != nil {
		model := event.Session.InputAudioTranscription.Model
		language := event.Session.InputAudioTranscription.Language
		if model != "" && language != "" {
			if err := u.reconfigureASRProvider(conn, state, event.EventID, model, language); err != nil {
				// Error already sent to client
				return
			}
		}
	}

	// Send transcription_session.updated event with flattened format
	transcriptionSessionUpdatedEvent := &domain.TranscriptionSessionUpdatedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventTranscriptionSessionUpdated,
		},
		Session: domain.NewTranscriptionSessionConfig(state.Config),
	}

	conn.WriteJSON(transcriptionSessionUpdatedEvent)
}

// reconfigureASRProvider loads/gets the ASR provider for the requested model and language
// Uses the registry for singleton pattern - models are loaded once and reused
func (u *SessionUsecase) reconfigureASRProvider(conn Conn, state *domain.SessionState, eventID, modelName, language string) error {
	// Check if registry is available
	if u.asrRegistry == nil {
		u.sendError(conn, eventID, "server_error", "configuration_unavailable",
			"ASR configuration not available. Server was not initialized with YAML config.", nil)
		return fmt.Errorf("ASR configuration not available")
	}

	// Validate model_name is provided
	if modelName == "" {
		u.sendError(conn, eventID, "invalid_request_error", "missing_field",
			"transcription.model is required", "audio.input.transcription.model")
		return fmt.Errorf("model is required")
	}

	// Validate language is provided
	if language == "" {
		u.sendError(conn, eventID, "invalid_request_error", "missing_field",
			"transcription.language is required", "audio.input.transcription.language")
		return fmt.Errorf("language is required")
	}

	// Get model from registry (lazy loading with singleton pattern)
	provider, err := u.asrRegistry.GetModel(modelName, language)
	if err != nil {
		// Determine error type based on error message
		errMsg := err.Error()
		if contains(errMsg, "not found") {
			u.sendError(conn, eventID, "invalid_request_error", "invalid_model",
				err.Error(), "audio.input.transcription.model")
		} else if contains(errMsg, "not supported") {
			u.sendError(conn, eventID, "invalid_request_error", "unsupported_language",
				err.Error(), "audio.input.transcription.language")
		} else {
			u.sendError(conn, eventID, "server_error", "provider_initialization_failed",
				err.Error(), nil)
		}
		return err
	}

	// Update the ASR provider for this session
	state.ASRProvider = provider

	log.Printf("[INFO] ASR provider set to model: %s, language: %s", modelName, language)
	return nil
}

// contains checks if s contains substr (simple helper to avoid importing strings)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// AUDIO BUFFER HANDLERS
// ============================================================================

func (u *SessionUsecase) handleInputAudioBufferAppend(conn Conn, state *domain.SessionState, message []byte) {
	var event domain.InputAudioBufferAppendEvent
	if err := json.Unmarshal(message, &event); err != nil {
		u.sendError(conn, "", "invalid_request_error", "invalid_event", "Failed to parse input_audio_buffer.append", nil)
		return
	}

	if event.Audio == "" {
		u.sendError(conn, event.EventID, "invalid_request_error", "missing_field", "audio field is required", "audio")
		return
	}

	// Decode base64 audio
	audioBytes, err := base64.StdEncoding.DecodeString(event.Audio)
	if err != nil {
		u.sendError(conn, event.EventID, "invalid_request_error", "invalid_audio", "Invalid base64 audio data", "audio")
		return
	}

	// Append to buffer (with size limit check)
	if err := state.AudioBuffer.Append(audioBytes); err != nil {
		if errors.Is(err, domain.ErrBufferFull) {
			u.sendError(conn, event.EventID, "invalid_request_error", "buffer_full",
				fmt.Sprintf("Audio buffer size limit exceeded (max %d bytes)", state.AudioBuffer.GetMaxSize()), "audio")
			return
		}
		u.sendError(conn, event.EventID, "server_error", "buffer_error", err.Error(), "audio")
		return
	}
	log.Printf("Appended audio to buffer, total size: %d bytes", state.AudioBuffer.GetSize())

	// Process through VAD if enabled
	if state.Config.Audio != nil && state.Config.Audio.Input != nil &&
		state.Config.Audio.Input.TurnDetection != nil &&
		state.Config.Audio.Input.TurnDetection.Type != "" {

		vad := u.getOrCreateVAD(state)
		if err := vad.ProcessAudio(context.Background(), audioBytes); err != nil {
			log.Printf("VAD processing error: %v", err)
		}

		// Check for VAD events
		u.processVADEvents(conn, state, vad)
	}

	// Note: client doesn't expect a response for append events
}

// processVADEvents handles VAD events and sends appropriate server events
func (u *SessionUsecase) processVADEvents(conn Conn, state *domain.SessionState, v domain.VADProvider) {
	for {
		select {
		case event, ok := <-v.GetEvents():
			if !ok {
				return
			}

			switch event.Type {
			case domain.VADEventSpeechStarted:
				// Generate item ID for this speech segment
				itemID := u.idGen.GenerateItemID()

				speechStartedEvent := &domain.InputAudioBufferSpeechStartedEvent{
					BaseEvent: domain.BaseEvent{
						EventID: u.idGen.GenerateEventID(),
						Type:    domain.EventInputAudioBufferSpeechStarted,
					},
					AudioStartMs: event.StartMs,
					ItemID:       itemID,
				}
				conn.WriteJSON(speechStartedEvent)
				log.Printf("Speech started at %d ms, item_id: %s", event.StartMs, itemID)

			case domain.VADEventSpeechStopped:
				itemID := u.idGen.GenerateItemID()

				speechStoppedEvent := &domain.InputAudioBufferSpeechStoppedEvent{
					BaseEvent: domain.BaseEvent{
						EventID: u.idGen.GenerateEventID(),
						Type:    domain.EventInputAudioBufferSpeechStopped,
					},
					AudioEndMs: event.EndMs,
					ItemID:     itemID,
				}
				conn.WriteJSON(speechStoppedEvent)
				log.Printf("Speech stopped at %d ms, item_id: %s", event.EndMs, itemID)

				// Auto-commit if VAD detected speech end
				if len(event.AudioData) > 0 {
					u.commitAndTranscribe(conn, state, itemID, event.AudioData)
				}

			case domain.VADEventTimeout:
				// Send timeout event
				timeoutEvent := &domain.InputAudioBufferTimeoutTriggeredEvent{
					BaseEvent: domain.BaseEvent{
						EventID: u.idGen.GenerateEventID(),
						Type:    "input_audio_buffer.timeout_triggered",
					},
					AudioStartMs: event.StartMs,
					AudioEndMs:   event.EndMs,
					ItemID:       u.idGen.GenerateItemID(),
				}
				conn.WriteJSON(timeoutEvent)
			}

		default:
			// No more events
			return
		}
	}
}

func (u *SessionUsecase) handleInputAudioBufferCommit(conn Conn, state *domain.SessionState, message []byte) {
	var event domain.InputAudioBufferCommitEvent
	if err := json.Unmarshal(message, &event); err != nil {
		u.sendError(conn, "", "invalid_request_error", "invalid_event", "Failed to parse input_audio_buffer.commit", nil)
		return
	}

	if state.AudioBuffer.IsEmpty() {
		u.sendError(conn, event.EventID, "invalid_request_error", "empty_buffer", "Audio buffer is empty", nil)
		return
	}

	// Get audio data and commit
	audioData := state.AudioBuffer.Commit()
	itemID := u.idGen.GenerateItemID()

	// Commit and transcribe
	u.commitAndTranscribe(conn, state, itemID, audioData)

	// Clear audio buffer after commit
	state.AudioBuffer.Clear()
}

// commitAndTranscribe handles the commit flow and triggers transcription
func (u *SessionUsecase) commitAndTranscribe(conn Conn, state *domain.SessionState, itemID string, audioData []byte) {
	// Create user message item from audio buffer
	item := domain.NewItem(itemID, "message", "user")
	item.Status = "completed"
	item.Content = []domain.ContentPart{
		{
			Type:   "input_audio",
			Audio:  base64.StdEncoding.EncodeToString(audioData),
			Format: "pcm16",
		},
	}

	// Get previous item ID before adding new item
	var previousItemID *string
	if len(state.Conversation.Order) > 0 {
		prevID := state.Conversation.Order[len(state.Conversation.Order)-1]
		previousItemID = &prevID
	}

	state.Conversation.AddItem(item)

	// Send input_audio_buffer.committed event
	committedEvent := &domain.InputAudioBufferCommittedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventInputAudioBufferCommitted,
		},
		ItemID:         itemID,
		PreviousItemID: previousItemID,
	}
	conn.WriteJSON(committedEvent)

	// Send conversation.item.created event
	itemCreatedEvent := &domain.ConversationItemAddedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventConversationItemCreated,
		},
		Item:           item,
		PreviousItemID: previousItemID,
	}
	conn.WriteJSON(itemCreatedEvent)

	// Trigger transcription asynchronously
	go u.transcribeAudio(conn, state, itemID, audioData)
}

// transcribeAudio performs speech-to-text transcription and sends events
func (u *SessionUsecase) transcribeAudio(conn Conn, state *domain.SessionState, itemID string, audioData []byte) {
	// Check if ASR provider is configured
	if state.ASRProvider == nil {
		failedEvent := &domain.ErrorServerEvent{
			BaseEvent: domain.BaseEvent{
				EventID: u.idGen.GenerateEventID(),
				Type:    domain.EventConversationItemInputAudioTranscriptionFailed,
			},
			Error: &domain.ErrorDetail{
				Type:    "transcription_error",
				Code:    "provider_not_configured",
				Message: "ASR provider not configured. Send session.update with audio.input.transcription.model and audio.input.transcription.language first.",
			},
		}
		conn.WriteJSON(failedEvent)
		return
	}

	// Get transcription config from session
	var transcriptionConfig *domain.TranscriptionConfig
	if state.Config.Audio != nil && state.Config.Audio.Input != nil {
		transcriptionConfig = state.Config.Audio.Input.Transcription
	}

	// Use default config if not specified
	if transcriptionConfig == nil {
		transcriptionConfig = &domain.TranscriptionConfig{
			Model:    "default",
			Language: "en",
		}
	}

	// Create context with timeout for transcription
	ctx, cancel := context.WithTimeout(context.Background(), u.transcriptionTimeout)
	defer cancel()

	// Call ASR provider
	resultChan, err := state.ASRProvider.Transcribe(ctx, audioData, transcriptionConfig)
	if err != nil {
		// Send transcription failed event
		failedEvent := &domain.ErrorServerEvent{
			BaseEvent: domain.BaseEvent{
				EventID: u.idGen.GenerateEventID(),
				Type:    domain.EventConversationItemInputAudioTranscriptionFailed,
			},
			Error: &domain.ErrorDetail{
				Type:    "transcription_error",
				Code:    "transcription_failed",
				Message: err.Error(),
			},
		}
		conn.WriteJSON(failedEvent)
		return
	}

	// Stream transcription results
	var fullTranscript string
	contentIndex := 0

	for {
		select {
		case <-ctx.Done():
			// Timeout or cancellation
			log.Printf("Transcription timeout for item %s", itemID)
			failedEvent := &domain.ErrorServerEvent{
				BaseEvent: domain.BaseEvent{
					EventID: u.idGen.GenerateEventID(),
					Type:    domain.EventConversationItemInputAudioTranscriptionFailed,
				},
				Error: &domain.ErrorDetail{
					Type:    "transcription_error",
					Code:    "transcription_timeout",
					Message: "Transcription timed out",
				},
			}
			conn.WriteJSON(failedEvent)
			return

		case chunk, ok := <-resultChan:
			if !ok {
				// Channel closed, transcription complete
				goto done
			}

			fullTranscript += chunk.Text

			// Send delta event for each chunk
			deltaEvent := &domain.ConversationItemInputAudioTranscriptionDeltaEvent{
				BaseEvent: domain.BaseEvent{
					EventID: u.idGen.GenerateEventID(),
					Type:    domain.EventConversationItemInputAudioTranscriptionDelta,
				},
				ItemID:       itemID,
				ContentIndex: contentIndex,
				Delta:        chunk.Text,
			}
			conn.WriteJSON(deltaEvent)
			log.Printf("Transcription delta: %s", chunk.Text)
		}
	}

done:
	// Send completed event
	completedEvent := &domain.ConversationItemInputAudioTranscriptionCompletedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventConversationItemInputAudioTranscriptionCompleted,
		},
		ItemID:       itemID,
		ContentIndex: contentIndex,
		Transcript:   fullTranscript,
	}
	conn.WriteJSON(completedEvent)
	log.Printf("Transcription completed: %s", fullTranscript)

	// Update item with transcript
	if item := state.Conversation.GetItem(itemID); item != nil && len(item.Content) > 0 {
		item.Content[0].Transcript = fullTranscript
	}
}

func (u *SessionUsecase) handleInputAudioBufferClear(conn Conn, state *domain.SessionState, message []byte) {
	var event domain.InputAudioBufferClearEvent
	if err := json.Unmarshal(message, &event); err != nil {
		u.sendError(conn, "", "invalid_request_error", "invalid_event", "Failed to parse input_audio_buffer.clear", nil)
		return
	}

	state.AudioBuffer.Clear()

	// Send input_audio_buffer.cleared event
	clearedEvent := &domain.InputAudioBufferClearedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventInputAudioBufferCleared,
		},
	}

	conn.WriteJSON(clearedEvent)
}

// ============================================================================
// CONVERSATION ITEM HANDLERS
// ============================================================================

func (u *SessionUsecase) handleConversationItemCreate(conn Conn, state *domain.SessionState, message []byte) {
	var event domain.ConversationItemCreateClientEvent
	if err := json.Unmarshal(message, &event); err != nil {
		u.sendError(conn, "", "invalid_request_error", "invalid_event", "Failed to parse conversation.item.create", nil)
		return
	}

	if event.Item == nil {
		u.sendError(conn, event.EventID, "invalid_request_error", "missing_field", "item field is required", "item")
		return
	}

	// Generate ID if not provided
	if event.Item.ID == "" {
		event.Item.ID = u.idGen.GenerateItemID()
	}
	event.Item.Object = "realtime.item"
	event.Item.Status = "completed"

	// Handle insertion position
	if event.PreviousItemID != nil && *event.PreviousItemID != "root" && *event.PreviousItemID != "" {
		// Insert after specified item (not implemented in simple version)
		// In production, find the item and insert after it
	}

	state.Conversation.AddItem(event.Item)

	// Send conversation.item.created event
	createdEvent := &domain.ConversationItemAddedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventConversationItemCreated,
		},
		Item:           event.Item,
		PreviousItemID: event.PreviousItemID,
	}

	conn.WriteJSON(createdEvent)
}

func (u *SessionUsecase) handleConversationItemDelete(conn Conn, state *domain.SessionState, message []byte) {
	var event domain.ConversationItemDeleteEvent
	if err := json.Unmarshal(message, &event); err != nil {
		u.sendError(conn, "", "invalid_request_error", "invalid_event", "Failed to parse conversation.item.delete", nil)
		return
	}

	if event.ItemID == "" {
		u.sendError(conn, event.EventID, "invalid_request_error", "missing_field", "item_id field is required", "item_id")
		return
	}

	if !state.Conversation.DeleteItem(event.ItemID) {
		u.sendError(conn, event.EventID, "invalid_request_error", "item_not_found",
			fmt.Sprintf("Item not found: %s", event.ItemID), nil)
		return
	}

	// Send conversation.item.deleted event
	deletedEvent := &domain.ConversationItemDeletedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventConversationItemDeleted,
		},
		ItemID: event.ItemID,
	}

	conn.WriteJSON(deletedEvent)
}

func (u *SessionUsecase) handleConversationItemTruncate(conn Conn, state *domain.SessionState, message []byte) {
	var event domain.ConversationItemTruncateEvent
	if err := json.Unmarshal(message, &event); err != nil {
		u.sendError(conn, "", "invalid_request_error", "invalid_event", "Failed to parse conversation.item.truncate", nil)
		return
	}

	if event.ItemID == "" {
		u.sendError(conn, event.EventID, "invalid_request_error", "missing_field", "item_id field is required", "item_id")
		return
	}

	item := state.Conversation.GetItem(event.ItemID)
	if item == nil {
		u.sendError(conn, event.EventID, "invalid_request_error", "item_not_found",
			fmt.Sprintf("Item not found: %s", event.ItemID), nil)
		return
	}

	// Send conversation.item.truncated event
	truncatedEvent := &domain.ConversationItemTruncatedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventConversationItemTruncated,
		},
		ItemID:       event.ItemID,
		ContentIndex: event.ContentIndex,
		AudioEndMs:   event.AudioEndMs,
	}

	conn.WriteJSON(truncatedEvent)
}

// ============================================================================
// RESPONSE HANDLERS
// ============================================================================

func (u *SessionUsecase) handleResponseCreate(conn Conn, state *domain.SessionState, message []byte) {
	var event domain.ResponseCreateClientEvent
	if err := json.Unmarshal(message, &event); err != nil {
		u.sendError(conn, "", "invalid_request_error", "invalid_event", "Failed to parse response.create", nil)
		return
	}

	// Create response
	responseID := u.idGen.GenerateResponseID()
	response := domain.NewResponse(responseID, state.Conversation.ID, state.Config.OutputModalities)

	// Apply overrides if provided
	if event.Response != nil {
		if event.Response.Instructions != "" {
			response.Usage = nil // Reset for demo
		}
		if len(event.Response.OutputModalities) > 0 {
			response.OutputModalities = event.Response.OutputModalities
		}
	}

	state.CurrentResponse = response

	// Send response.created event
	createdEvent := &domain.ResponseCreatedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventResponseCreated,
		},
		Response: response,
	}

	conn.WriteJSON(createdEvent)

	// Create mock assistant message
	assistantItemID := u.idGen.GenerateItemID()
	assistantItem := domain.NewItem(assistantItemID, "message", "assistant")

	// Send response.output_item.added
	itemAddedEvent := &domain.ResponseOutputItemAddedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventResponseOutputItemAdded,
		},
		ResponseID:  responseID,
		OutputIndex: 0,
		Item:        assistantItem,
	}

	conn.WriteJSON(itemAddedEvent)

	// Send mock text content part
	textPart := &domain.ContentPart{
		Type: "text",
		Text: "",
	}

	contentPartAddedEvent := &domain.ResponseContentPartAddedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventResponseContentPartAdded,
		},
		ResponseID:   responseID,
		ItemID:       assistantItemID,
		ContentIndex: 0,
		OutputIndex:  0,
		Part:         textPart,
	}

	conn.WriteJSON(contentPartAddedEvent)

	// Send mock text delta
	textDeltaEvent := &domain.ResponseOutputTextDeltaEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventResponseOutputTextDelta,
		},
		ResponseID:   responseID,
		ItemID:       assistantItemID,
		ContentIndex: 0,
		OutputIndex:  0,
		Delta:        "This is a mock response from the speech-to-text API.",
	}

	conn.WriteJSON(textDeltaEvent)

	// Send text done
	textDoneEvent := &domain.ResponseOutputTextDoneEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventResponseOutputTextDone,
		},
		ResponseID:   responseID,
		ItemID:       assistantItemID,
		ContentIndex: 0,
		OutputIndex:  0,
		Text:         "This is a mock response from the speech-to-text API.",
	}

	conn.WriteJSON(textDoneEvent)

	// Update item status
	assistantItem.Status = "completed"
	assistantItem.Content = []domain.ContentPart{
		{
			Type: "text",
			Text: "This is a mock response from the speech-to-text API.",
		},
	}

	// Send response.output_item.done
	itemDoneEvent := &domain.ResponseOutputItemDoneEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventResponseOutputItemDone,
		},
		ResponseID:  responseID,
		OutputIndex: 0,
		Item:        assistantItem,
	}

	conn.WriteJSON(itemDoneEvent)

	// Mark response as completed
	response.Status = "completed"
	response.Output = []domain.Item{*assistantItem}
	response.Usage = &domain.Usage{
		TotalTokens:  50,
		InputTokens:  20,
		OutputTokens: 30,
		InputTokenDetails: &domain.TokenDetails{
			TextTokens:  10,
			AudioTokens: 10,
		},
		OutputTokenDetails: &domain.TokenDetails{
			TextTokens: 30,
		},
	}

	// Send response.done
	doneEvent := &domain.ResponseDoneEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventResponseDone,
		},
		Response: response,
	}

	conn.WriteJSON(doneEvent)

	// Add assistant item to conversation
	state.Conversation.AddItem(assistantItem)
}

func (u *SessionUsecase) handleResponseCancel(conn Conn, state *domain.SessionState, message []byte) {
	var event domain.ResponseCancelEvent
	if err := json.Unmarshal(message, &event); err != nil {
		u.sendError(conn, "", "invalid_request_error", "invalid_event", "Failed to parse response.cancel", nil)
		return
	}

	if state.CurrentResponse == nil {
		u.sendError(conn, event.EventID, "invalid_request_error", "no_active_response", "No active response to cancel", nil)
		return
	}

	// Cancel the response
	state.CurrentResponse.Status = "cancelled"
	state.CurrentResponse = nil

	// Send response.done with cancelled status
	doneEvent := &domain.ResponseDoneEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventResponseDone,
		},
		Response: state.CurrentResponse,
	}

	conn.WriteJSON(doneEvent)
}

// ============================================================================
// ERROR HANDLING
// ============================================================================

func (u *SessionUsecase) sendError(conn Conn, clientEventID, errType, code, message string, param interface{}) {
	errorEvent := &domain.ErrorServerEvent{
		BaseEvent: domain.BaseEvent{
			EventID: u.idGen.GenerateEventID(),
			Type:    domain.EventError,
		},
		Error: &domain.ErrorDetail{
			Type:    errType,
			Code:    code,
			Message: message,
			Param:   param,
			EventID: clientEventID,
		},
	}

	if err := conn.WriteJSON(errorEvent); err != nil {
		log.Printf("Failed to send error event: %v", err)
	}
}

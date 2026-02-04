package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/aira-id/griber/internal/domain"
	"github.com/google/uuid"
)

// SessionManager handles session lifecycle and state
type SessionManager struct {
	sessions map[string]*domain.SessionState
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*domain.SessionState),
	}
}

// CreateSession creates a new session
func (sm *SessionManager) CreateSession(sessionID, model, conversationID string) *domain.SessionState {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state := &domain.SessionState{
		ID:              sessionID,
		Config:          domain.NewSession(sessionID, model),
		Conversation:    domain.NewConversationState(conversationID),
		AudioBuffer:     NewAudioBuffer(),
		CurrentResponse: nil,
		CreatedAt:       time.Now(),
		LastActivity:    time.Now(),
	}

	sm.sessions[sessionID] = state
	return state
}

// CreateTranscriptionSession creates a new transcription-only (STT) session
func (sm *SessionManager) CreateTranscriptionSession(sessionID, model, conversationID, language string) *domain.SessionState {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state := &domain.SessionState{
		ID:              sessionID,
		Config:          domain.NewTranscriptionSession(sessionID, model, language),
		Conversation:    domain.NewConversationState(conversationID),
		AudioBuffer:     NewAudioBuffer(),
		CurrentResponse: nil,
		CreatedAt:       time.Now(),
		LastActivity:    time.Now(),
	}

	sm.sessions[sessionID] = state
	return state
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*domain.SessionState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Update last activity
	state.LastActivity = time.Now()
	return state, nil
}

// UpdateSession updates session configuration
func (sm *SessionManager) UpdateSession(sessionID string, updates *domain.Session) (*domain.SessionState, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Merge updates (non-empty fields override)
	if updates.Type != "" {
		state.Config.Type = updates.Type
	}
	if updates.Instructions != "" {
		state.Config.Instructions = updates.Instructions
	}
	if updates.Tools != nil {
		state.Config.Tools = updates.Tools
	}
	if updates.ToolChoice != "" {
		state.Config.ToolChoice = updates.ToolChoice
	}
	if updates.MaxOutputTokens != nil {
		state.Config.MaxOutputTokens = updates.MaxOutputTokens
	}
	if updates.Temperature > 0 {
		state.Config.Temperature = updates.Temperature
	}
	if updates.Audio != nil {
		// Deep merge audio config
		if state.Config.Audio == nil {
			state.Config.Audio = updates.Audio
		} else {
			if updates.Audio.Input != nil {
				if state.Config.Audio.Input == nil {
					state.Config.Audio.Input = updates.Audio.Input
				} else {
					// Merge input config
					if updates.Audio.Input.Format != nil {
						state.Config.Audio.Input.Format = updates.Audio.Input.Format
					}
					if updates.Audio.Input.Transcription != nil {
						state.Config.Audio.Input.Transcription = updates.Audio.Input.Transcription
					}
					if updates.Audio.Input.NoiseReduction != nil {
						state.Config.Audio.Input.NoiseReduction = updates.Audio.Input.NoiseReduction
					}
					if updates.Audio.Input.TurnDetection != nil {
						state.Config.Audio.Input.TurnDetection = updates.Audio.Input.TurnDetection
					}
				}
			}
			if updates.Audio.Output != nil {
				state.Config.Audio.Output = updates.Audio.Output
			}
		}
	}
	if len(updates.OutputModalities) > 0 {
		state.Config.OutputModalities = updates.OutputModalities
	}
	if len(updates.Include) > 0 {
		state.Config.Include = updates.Include
	}

	state.LastActivity = time.Now()
	return state, nil
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, sessionID)
}

// ============================================================================
// AUDIO BUFFER
// ============================================================================

// NewAudioBuffer creates a new audio buffer with default settings
func NewAudioBuffer() *domain.AudioBuffer {
	return &domain.AudioBuffer{
		Data: make([]byte, 0),
		Lock: make(chan struct{}, 1),
	}
}

// NewAudioBufferWithMaxSize creates a new audio buffer with a size limit
func NewAudioBufferWithMaxSize(maxSize int) *domain.AudioBuffer {
	ab := &domain.AudioBuffer{
		Data: make([]byte, 0),
		Lock: make(chan struct{}, 1),
	}
	ab.SetMaxSize(maxSize)
	return ab
}

// ============================================================================
// ID GENERATORS
// ============================================================================

// IDGenerator generates unique IDs using UUIDs
// This ensures globally unique IDs even across server restarts and distributed systems
type IDGenerator struct{}

// NewIDGenerator creates a new ID generator
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{}
}

// generateShortUUID generates a shortened UUID (first 12 chars)
func generateShortUUID() string {
	return uuid.New().String()[:12]
}

// GenerateSessionID generates a unique session ID
func (gen *IDGenerator) GenerateSessionID() string {
	return "sess_" + generateShortUUID()
}

// GenerateConversationID generates a unique conversation ID
func (gen *IDGenerator) GenerateConversationID() string {
	return "conv_" + generateShortUUID()
}

// GenerateItemID generates a unique item ID
func (gen *IDGenerator) GenerateItemID() string {
	return "item_" + generateShortUUID()
}

// GenerateResponseID generates a unique response ID
func (gen *IDGenerator) GenerateResponseID() string {
	return "resp_" + generateShortUUID()
}

// GenerateEventID generates a unique event ID
func (gen *IDGenerator) GenerateEventID() string {
	return "evt_" + generateShortUUID()
}

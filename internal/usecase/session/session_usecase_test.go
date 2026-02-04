package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aira-id/griber/internal/domain"
	"github.com/aira-id/griber/internal/usecase/asr"
	"github.com/aira-id/griber/internal/usecase/vad"
)

// TestSessionManager tests
func TestSessionManagerCreateSession(t *testing.T) {
	sm := NewSessionManager()

	sessionID := "test_session"
	conversationID := "test_conv"
	model := "gpt-realtime-2025-08-28"

	state := sm.CreateSession(sessionID, model, conversationID)

	if state.ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, state.ID)
	}

	if state.Config.Model != model {
		t.Errorf("Expected model %s, got %s", model, state.Config.Model)
	}

	if state.Conversation.ID != conversationID {
		t.Errorf("Expected conversation ID %s, got %s", conversationID, state.Conversation.ID)
	}
}

func TestAudioBufferAppend(t *testing.T) {
	ab := NewAudioBuffer()

	data1 := []byte("chunk1")
	data2 := []byte("chunk2")

	if err := ab.Append(data1); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if err := ab.Append(data2); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if ab.GetSize() != len(data1)+len(data2) {
		t.Errorf("Expected buffer size %d, got %d", len(data1)+len(data2), ab.GetSize())
	}
}

func TestAudioBufferSizeLimit(t *testing.T) {
	ab := NewAudioBufferWithMaxSize(100)

	// Should succeed - within limit
	data := make([]byte, 50)
	if err := ab.Append(data); err != nil {
		t.Errorf("Unexpected error for data within limit: %v", err)
	}

	// Should succeed - still within limit
	if err := ab.Append(data); err != nil {
		t.Errorf("Unexpected error for second append within limit: %v", err)
	}

	// Should fail - would exceed limit
	if err := ab.Append(data); err == nil {
		t.Error("Expected error when exceeding buffer limit, got nil")
	} else if err != domain.ErrBufferFull {
		t.Errorf("Expected ErrBufferFull, got %v", err)
	}

	// Verify size hasn't changed after failed append
	if ab.GetSize() != 100 {
		t.Errorf("Expected buffer size 100 after failed append, got %d", ab.GetSize())
	}
}

func TestNewSessionConfiguration(t *testing.T) {
	session := domain.NewSession("sess_1", "model")

	if session.ID != "sess_1" {
		t.Errorf("Expected session ID 'sess_1', got %s", session.ID)
	}

	if session.Type != "realtime" {
		t.Errorf("Expected type 'realtime', got %s", session.Type)
	}

	if session.Audio == nil {
		t.Error("Expected audio configuration to be set")
	}

	if session.Audio.Input == nil || session.Audio.Input.Format.Rate != 24000 {
		t.Error("Expected audio format rate 24000")
	}
}

func TestNewResponse(t *testing.T) {
	response := domain.NewResponse("resp_1", "conv_1", []string{"audio", "text"})

	if response.ID != "resp_1" {
		t.Errorf("Expected response ID 'resp_1', got %s", response.ID)
	}

	if response.Status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got %s", response.Status)
	}

	if response.Object != "realtime.response" {
		t.Errorf("Expected object 'realtime.response', got %s", response.Object)
	}
}

func TestEventSerializationSessionCreated(t *testing.T) {
	session := domain.NewSession("sess_1", "model")

	event := &domain.SessionCreatedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: "evt_1",
			Type:    domain.EventSessionCreated,
		},
		Session: session,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var unmarshalled domain.SessionCreatedEvent
	err = json.Unmarshal(data, &unmarshalled)
	if err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if unmarshalled.BaseEvent.Type != domain.EventSessionCreated {
		t.Error("Event type mismatch after serialization")
	}
}

// ============================================================================
// STT-SPECIFIC TESTS
// ============================================================================

func TestNewTranscriptionSession(t *testing.T) {
	session := domain.NewTranscriptionSession("sess_1", "model-1", "en")

	if session.ID != "sess_1" {
		t.Errorf("Expected session ID 'sess_1', got %s", session.ID)
	}

	if session.Type != "transcription" {
		t.Errorf("Expected type 'transcription', got %s", session.Type)
	}

	if session.Model != "model-1" {
		t.Errorf("Expected model 'model-1', got %s", session.Model)
	}

	// Check output modalities - should only be text for STT
	if len(session.OutputModalities) != 1 || session.OutputModalities[0] != "text" {
		t.Errorf("Expected output modalities ['text'], got %v", session.OutputModalities)
	}

	// Check transcription config
	if session.Audio == nil || session.Audio.Input == nil {
		t.Error("Expected audio input configuration to be set")
	}

	if session.Audio.Input.Transcription == nil {
		t.Error("Expected transcription configuration to be set")
	}

	if session.Audio.Input.Transcription.Language != "en" {
		t.Errorf("Expected language 'en', got %s", session.Audio.Input.Transcription.Language)
	}

	// No audio output for STT mode
	if session.Audio.Output != nil {
		t.Error("Expected no audio output for transcription mode")
	}
}

func TestSessionManagerCreateTranscriptionSession(t *testing.T) {
	sm := NewSessionManager()

	sessionID := "test_stt_session"
	conversationID := "test_conv"
	model := "model-1"
	language := "en"

	state := sm.CreateTranscriptionSession(sessionID, model, conversationID, language)

	if state.ID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, state.ID)
	}

	if state.Config.Type != "transcription" {
		t.Errorf("Expected type 'transcription', got %s", state.Config.Type)
	}

	if state.Config.Audio.Input.Transcription.Model != model {
		t.Errorf("Expected model %s, got %s", model, state.Config.Audio.Input.Transcription.Model)
	}
}

func TestMockASRProvider(t *testing.T) {
	a := asr.NewMockProvider()

	// Test GetSupportedModels
	models := a.GetSupportedModels()
	if len(models) == 0 {
		t.Error("Expected at least one supported model")
	}

	// Test GetSupportedLanguages
	languages := a.GetSupportedLanguages()
	if len(languages) == 0 {
		t.Error("Expected at least one supported language")
	}
}

func TestTranscriptionConfigSerialization(t *testing.T) {
	config := &domain.TranscriptionConfig{
		Model:    "model-1",
		Language: "en",
		Prompt:   "Test prompt",
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	var unmarshalled domain.TranscriptionConfig
	err = json.Unmarshal(data, &unmarshalled)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if unmarshalled.Model != config.Model {
		t.Errorf("Expected model %s, got %s", config.Model, unmarshalled.Model)
	}

	if unmarshalled.Language != config.Language {
		t.Errorf("Expected language %s, got %s", config.Language, unmarshalled.Language)
	}
}

func TestTranscriptionEventTypes(t *testing.T) {
	// Verify event types are correctly defined
	if domain.EventConversationItemInputAudioTranscriptionDelta != "conversation.item.input_audio_transcription.delta" {
		t.Error("Transcription delta event type mismatch")
	}

	if domain.EventConversationItemInputAudioTranscriptionCompleted != "conversation.item.input_audio_transcription.completed" {
		t.Error("Transcription completed event type mismatch")
	}

	if domain.EventConversationItemInputAudioTranscriptionFailed != "conversation.item.input_audio_transcription.failed" {
		t.Error("Transcription failed event type mismatch")
	}
}

func TestVADConfig(t *testing.T) {
	config := domain.NewDefaultVADConfig()

	if config.Type != "server_vad" {
		t.Errorf("Expected default VAD type 'server_vad', got %s", config.Type)
	}

	if config.Threshold != 0.5 {
		t.Errorf("Expected default threshold 0.5, got %f", config.Threshold)
	}

	if config.SampleRate != 24000 {
		t.Errorf("Expected default sample rate 24000, got %d", config.SampleRate)
	}
}

func TestVADConfigFromTurnDetection(t *testing.T) {
	td := &domain.TurnDetection{
		Type:              "server_vad",
		Threshold:         0.7,
		PrefixPaddingMs:   200,
		SilenceDurationMs: 400,
		IdleTimeoutMs:     5000,
	}

	config := domain.VADConfigFromTurnDetection(td)

	if config.Type != td.Type {
		t.Errorf("Expected type %s, got %s", td.Type, config.Type)
	}

	if config.Threshold != td.Threshold {
		t.Errorf("Expected threshold %f, got %f", td.Threshold, config.Threshold)
	}

	if config.PrefixPaddingMs != td.PrefixPaddingMs {
		t.Errorf("Expected prefix padding %d, got %d", td.PrefixPaddingMs, config.PrefixPaddingMs)
	}
}

func TestSimpleVADProvider(t *testing.T) {
	config := domain.NewDefaultVADConfig()
	v := vad.NewSimpleVADProvider(config)
	defer v.Close()

	if v.IsSpeaking() {
		t.Error("VAD should not be speaking initially")
	}

	// Reset should work without panic
	v.Reset()

	if v.GetCurrentMs() != 0 {
		t.Errorf("Expected current ms to be 0 after reset, got %d", v.GetCurrentMs())
	}
}

func TestTranscriptionEventSerialization(t *testing.T) {
	deltaEvent := &domain.ConversationItemInputAudioTranscriptionDeltaEvent{
		BaseEvent: domain.BaseEvent{
			EventID: "evt_1",
			Type:    domain.EventConversationItemInputAudioTranscriptionDelta,
		},
		ItemID:       "item_1",
		ContentIndex: 0,
		Delta:        "Hello",
	}

	data, err := json.Marshal(deltaEvent)
	if err != nil {
		t.Fatalf("Failed to marshal delta event: %v", err)
	}

	var unmarshalled domain.ConversationItemInputAudioTranscriptionDeltaEvent
	err = json.Unmarshal(data, &unmarshalled)
	if err != nil {
		t.Fatalf("Failed to unmarshal delta event: %v", err)
	}

	if unmarshalled.Delta != "Hello" {
		t.Errorf("Expected delta 'Hello', got %s", unmarshalled.Delta)
	}

	// Test completed event
	completedEvent := &domain.ConversationItemInputAudioTranscriptionCompletedEvent{
		BaseEvent: domain.BaseEvent{
			EventID: "evt_2",
			Type:    domain.EventConversationItemInputAudioTranscriptionCompleted,
		},
		ItemID:       "item_1",
		ContentIndex: 0,
		Transcript:   "Hello, world!",
	}

	data, err = json.Marshal(completedEvent)
	if err != nil {
		t.Fatalf("Failed to marshal completed event: %v", err)
	}

	var unmarshalledCompleted domain.ConversationItemInputAudioTranscriptionCompletedEvent
	err = json.Unmarshal(data, &unmarshalledCompleted)
	if err != nil {
		t.Fatalf("Failed to unmarshal completed event: %v", err)
	}

	if unmarshalledCompleted.Transcript != "Hello, world!" {
		t.Errorf("Expected transcript 'Hello, world!', got %s", unmarshalledCompleted.Transcript)
	}
}

func TestIDGeneratorUUID(t *testing.T) {
	gen := NewIDGenerator()

	// Test session ID format
	sessionID := gen.GenerateSessionID()
	if !strings.HasPrefix(sessionID, "sess_") {
		t.Errorf("Session ID should start with 'sess_', got %s", sessionID)
	}
	if len(sessionID) != 17 { // "sess_" (5) + 12 chars UUID
		t.Errorf("Session ID should be 17 chars, got %d: %s", len(sessionID), sessionID)
	}

	// Test conversation ID format
	convID := gen.GenerateConversationID()
	if !strings.HasPrefix(convID, "conv_") {
		t.Errorf("Conversation ID should start with 'conv_', got %s", convID)
	}

	// Test item ID format
	itemID := gen.GenerateItemID()
	if !strings.HasPrefix(itemID, "item_") {
		t.Errorf("Item ID should start with 'item_', got %s", itemID)
	}

	// Test response ID format
	respID := gen.GenerateResponseID()
	if !strings.HasPrefix(respID, "resp_") {
		t.Errorf("Response ID should start with 'resp_', got %s", respID)
	}

	// Test event ID format
	evtID := gen.GenerateEventID()
	if !strings.HasPrefix(evtID, "evt_") {
		t.Errorf("Event ID should start with 'evt_', got %s", evtID)
	}

	// Test uniqueness - generate multiple IDs and ensure they're different
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := gen.GenerateSessionID()
		if ids[id] {
			t.Errorf("Generated duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

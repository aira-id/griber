package domain

// EventType represents the type of event for both client and server.
type EventType string

// BaseEvent is the base structure for all events
type BaseEvent struct {
	EventID string    `json:"event_id"`
	Type    EventType `json:"type"`
}

const (
	// Client Events
	EventSessionUpdate            EventType = "session.update"
	EventInputAudioBufferAppend   EventType = "input_audio_buffer.append"
	EventInputAudioBufferCommit   EventType = "input_audio_buffer.commit"
	EventInputAudioBufferClear    EventType = "input_audio_buffer.clear"
	EventConversationItemCreate   EventType = "conversation.item.create"
	EventConversationItemTruncate EventType = "conversation.item.truncate"
	EventConversationItemDelete   EventType = "conversation.item.delete"
	EventResponseCreate           EventType = "response.create"
	EventResponseCancel           EventType = "response.cancel"

	// Server Events
	EventSessionCreated                EventType = "session.created"
	EventSessionUpdated                EventType = "session.updated"
	EventError                         EventType = "error"
	EventInputAudioBufferCommitted     EventType = "input_audio_buffer.committed"
	EventInputAudioBufferCleared       EventType = "input_audio_buffer.cleared"
	EventInputAudioBufferSpeechStarted EventType = "input_audio_buffer.speech_started"
	EventInputAudioBufferSpeechStopped EventType = "input_audio_buffer.speech_stopped"
	EventConversationItemCreated       EventType = "conversation.item.created"
	EventConversationItemDeleted       EventType = "conversation.item.deleted"
	EventConversationItemTruncated     EventType = "conversation.item.truncated"
	EventResponseCreated               EventType = "response.created"
	EventResponseDone                  EventType = "response.done"
	EventResponseOutputItemAdded       EventType = "response.output_item.added"
	EventResponseOutputItemDone        EventType = "response.output_item.done"
	EventResponseContentPartAdded      EventType = "response.content_part.added"
	EventResponseContentPartDone       EventType = "response.content_part.done"
	EventResponseOutputTextDelta       EventType = "response.output_text.delta"
	EventResponseOutputTextDone        EventType = "response.output_text.done"
	EventResponseAudioTranscriptDelta  EventType = "response.output_audio_transcript.delta"
	EventResponseAudioTranscriptDone   EventType = "response.output_audio_transcript.done"
	EventResponseOutputAudioDelta      EventType = "response.output_audio.delta"
	EventResponseOutputAudioDone       EventType = "response.output_audio.done"

	// Transcription Events (STT-specific)
	EventConversationItemInputAudioTranscriptionDelta     EventType = "conversation.item.input_audio_transcription.delta"
	EventConversationItemInputAudioTranscriptionCompleted EventType = "conversation.item.input_audio_transcription.completed"
	EventConversationItemInputAudioTranscriptionFailed    EventType = "conversation.item.input_audio_transcription.failed"

	// Rate Limits
	EventRateLimitsUpdated EventType = "rate_limits.updated"

	// Transcription Session Events (for OpenAI Realtime Transcription API compatibility)
	EventTranscriptionSessionUpdate  EventType = "transcription_session.update"  // Client event
	EventTranscriptionSessionCreated EventType = "transcription_session.created" // Server event
	EventTranscriptionSessionUpdated EventType = "transcription_session.updated" // Server event
)

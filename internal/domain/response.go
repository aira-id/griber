package domain

import "time"

// Response represents a model response
type Response struct {
	Object           string         `json:"object"` // "realtime.response"
	ID               string         `json:"id"`
	Status           string         `json:"status"` // "in_progress", "completed", "incomplete", "cancelled", "failed"
	StatusDetails    interface{}    `json:"status_details"`
	Output           []Item         `json:"output"`
	ConversationID   string         `json:"conversation_id"`
	OutputModalities []string       `json:"output_modalities"`
	MaxOutputTokens  interface{}    `json:"max_output_tokens"`
	Audio            *ResponseAudio `json:"audio,omitempty"`
	Usage            *Usage         `json:"usage"`
	Metadata         interface{}    `json:"metadata"`
	CreatedAt        int64          `json:"created_at"`
}

// ResponseAudio represents audio configuration in response
type ResponseAudio struct {
	Output *AudioOutput `json:"output"`
}

// Usage represents token usage
type Usage struct {
	TotalTokens        int           `json:"total_tokens"`
	InputTokens        int           `json:"input_tokens"`
	OutputTokens       int           `json:"output_tokens"`
	InputTokenDetails  *TokenDetails `json:"input_token_details,omitempty"`
	OutputTokenDetails *TokenDetails `json:"output_token_details,omitempty"`
}

// TokenDetails represents detailed token information
type TokenDetails struct {
	TextTokens          int                 `json:"text_tokens"`
	AudioTokens         int                 `json:"audio_tokens"`
	ImageTokens         int                 `json:"image_tokens,omitempty"`
	CachedTokens        int                 `json:"cached_tokens,omitempty"`
	CachedTokensDetails *CachedTokenDetails `json:"cached_tokens_details,omitempty"`
}

// CachedTokenDetails represents cached token breakdown
type CachedTokenDetails struct {
	TextTokens  int `json:"text_tokens"`
	AudioTokens int `json:"audio_tokens"`
	ImageTokens int `json:"image_tokens,omitempty"`
}

// NewResponse creates a new response
func NewResponse(responseID, conversationID string, modalities []string) *Response {
	return &Response{
		Object:           "realtime.response",
		ID:               responseID,
		Status:           "in_progress",
		StatusDetails:    nil,
		Output:           []Item{},
		ConversationID:   conversationID,
		OutputModalities: modalities,
		MaxOutputTokens:  "inf",
		Audio: &ResponseAudio{
			Output: &AudioOutput{
				Format: &AudioFormat{
					Type: "audio/pcm",
					Rate: 24000,
				},
				Voice: "alloy",
				Speed: 1.0,
			},
		},
		Usage:     nil,
		Metadata:  nil,
		CreatedAt: time.Now().Unix(),
	}
}

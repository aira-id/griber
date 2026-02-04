package domain

// Tool represents a function tool
type Tool struct {
	Type        string      `json:"type"` // "function"
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"` // JSON schema
}

// ErrorDetail represents error information
type ErrorDetail struct {
	Type    string      `json:"type"`               // "invalid_request_error", "invalid_api_key_error", etc
	Code    string      `json:"code"`               // Specific error code
	Message string      `json:"message"`            // Human-readable message
	Param   interface{} `json:"param"`              // Related parameter if applicable
	EventID string      `json:"event_id,omitempty"` // Echo back client event_id
}

// RateLimit represents rate limit information
type RateLimit struct {
	Name         string `json:"name"` // "requests", "tokens"
	Limit        int    `json:"limit"`
	Remaining    int    `json:"remaining"`
	ResetSeconds int    `json:"reset_seconds"`
}

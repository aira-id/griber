package websocket

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/aira-id/griber/internal/config"
	"github.com/aira-id/griber/internal/middleware"
	"github.com/aira-id/griber/internal/usecase/session"
	"github.com/gorilla/websocket"
)

// Handler handles WebSocket connections
type Handler struct {
	UseCase     *session.SessionUsecase
	Config      *config.Config
	RateLimiter *middleware.RateLimiter
	upgrader    websocket.Upgrader
}

// WebSocket buffer sizes optimized for audio streaming
const (
	// ReadBufferSize is sized for typical audio chunks (4-8KB at 16kHz)
	// 32KB allows efficient handling of larger chunks without excessive syscalls
	wsReadBufferSize = 32 * 1024

	// WriteBufferSize is sized for JSON events with potential base64 audio
	// 64KB handles serialized events without multiple flushes
	wsWriteBufferSize = 64 * 1024
)

// NewHandler creates a new WebSocket handler
func NewHandler(uc *session.SessionUsecase, cfg *config.Config) *Handler {
	h := &Handler{
		UseCase:     uc,
		Config:      cfg,
		RateLimiter: middleware.NewRateLimiter(&cfg.Rate),
	}

	h.upgrader = websocket.Upgrader{
		CheckOrigin:     h.checkOrigin,
		ReadBufferSize:  wsReadBufferSize,
		WriteBufferSize: wsWriteBufferSize,
	}

	return h
}

// checkOrigin validates the request origin against allowed origins
func (h *Handler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	// If no origin header, allow (same-origin request)
	if origin == "" {
		return true
	}

	return h.Config.IsOriginAllowed(origin)
}

// ServeHTTP implements http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientIP := middleware.GetClientIP(r)

	// Check rate limit for connection attempts
	if !h.RateLimiter.Allow(clientIP) {
		log.Printf("Rate limit exceeded for IP: %s", clientIP)
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}

	// Check connection limit per IP
	if !h.RateLimiter.AddConnection(clientIP) {
		log.Printf("Connection limit exceeded for IP: %s", clientIP)
		http.Error(w, "Too many connections", http.StatusTooManyRequests)
		return
	}

	// Validate API key
	if !h.validateAPIKey(r) {
		h.RateLimiter.RemoveConnection(clientIP)
		log.Printf("Invalid API key from IP: %s", clientIP)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade connection
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.RateLimiter.RemoveConnection(clientIP)
		log.Println("Upgrade error:", err)
		return
	}

	// Wrap connection with thread-safe writer
	safeConn := NewSafeConn(conn)

	// Parse intent from query parameter (OpenAI compatible: ?intent=transcription)
	intent := session.IntentRealtime
	intentParam := r.URL.Query().Get("intent")
	if intentParam == "transcription" {
		intent = session.IntentTranscription
		log.Printf("Starting transcription session for IP: %s", clientIP)
	}

	// Handle connection in goroutine and track cleanup
	go func() {
		defer h.RateLimiter.RemoveConnection(clientIP)
		defer safeConn.Close()
		h.UseCase.HandleNewConnectionWithIntent(safeConn, intent)
	}()
}

// validateAPIKey checks if the request has a valid API key
func (h *Handler) validateAPIKey(r *http.Request) bool {
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

	// Check query parameter (for WebSocket clients that can't set headers)
	apiKey = r.URL.Query().Get("api_key")
	if apiKey != "" {
		return h.Config.IsAPIKeyValid(apiKey)
	}

	// If no API keys configured, allow without auth
	return h.Config.IsAPIKeyValid("")
}

// Close cleans up handler resources
func (h *Handler) Close() {
	h.RateLimiter.Close()
}

// SafeConn wraps a WebSocket connection with thread-safe write operations
type SafeConn struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

// NewSafeConn creates a new thread-safe WebSocket connection wrapper
func NewSafeConn(conn *websocket.Conn) *SafeConn {
	return &SafeConn{conn: conn}
}

// WriteJSON writes JSON data in a thread-safe manner
func (sc *SafeConn) WriteJSON(v interface{}) error {
	sc.writeMu.Lock()
	defer sc.writeMu.Unlock()
	return sc.conn.WriteJSON(v)
}

// ReadMessage reads a message from the connection
func (sc *SafeConn) ReadMessage() (messageType int, p []byte, err error) {
	return sc.conn.ReadMessage()
}

// Close closes the underlying connection
func (sc *SafeConn) Close() error {
	return sc.conn.Close()
}

// Conn returns the underlying websocket connection (use with caution)
func (sc *SafeConn) Conn() *websocket.Conn {
	return sc.conn
}

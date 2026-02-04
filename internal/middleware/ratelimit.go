package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/aira-id/griber/internal/config"
)

// RateLimiter implements IP-based rate limiting
type RateLimiter struct {
	config      *config.RateLimitConfig
	connections map[string]*clientState
	mu          sync.RWMutex
	stopCleanup chan struct{}
}

type clientState struct {
	connections int
	tokens      float64
	lastUpdate  time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cfg *config.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:      cfg,
		connections: make(map[string]*clientState),
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	state, exists := rl.connections[ip]
	if !exists {
		state = &clientState{
			connections: 0,
			tokens:      float64(rl.config.BurstSize),
			lastUpdate:  time.Now(),
		}
		rl.connections[ip] = state
	}

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(state.lastUpdate).Seconds()
	state.tokens += elapsed * float64(rl.config.RequestsPerSecond)
	if state.tokens > float64(rl.config.BurstSize) {
		state.tokens = float64(rl.config.BurstSize)
	}
	state.lastUpdate = now

	// Check if we have tokens available
	if state.tokens < 1 {
		return false
	}

	state.tokens--
	return true
}

// AddConnection tracks a new connection from an IP
// Returns false if the connection limit is exceeded
func (rl *RateLimiter) AddConnection(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	state, exists := rl.connections[ip]
	if !exists {
		state = &clientState{
			connections: 0,
			tokens:      float64(rl.config.BurstSize),
			lastUpdate:  time.Now(),
		}
		rl.connections[ip] = state
	}

	if state.connections >= rl.config.MaxConnectionsPerIP {
		return false
	}

	state.connections++
	return true
}

// RemoveConnection removes a connection tracking for an IP
func (rl *RateLimiter) RemoveConnection(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if state, exists := rl.connections[ip]; exists {
		state.connections--
		if state.connections < 0 {
			state.connections = 0
		}
	}
}

// cleanupLoop periodically removes stale entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCleanup:
			return
		}
	}
}

// cleanup removes entries with no connections and full token buckets
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	staleThreshold := 5 * time.Minute

	for ip, state := range rl.connections {
		// Remove if no connections and hasn't been used recently
		if state.connections == 0 && now.Sub(state.lastUpdate) > staleThreshold {
			delete(rl.connections, ip)
		}
	}
}

// Close stops the rate limiter
func (rl *RateLimiter) Close() {
	close(rl.stopCleanup)
}

// GetClientIP extracts the client IP from an HTTP request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if idx := len(xff); idx > 0 {
			for i, c := range xff {
				if c == ',' {
					return xff[:i]
				}
			}
			return xff
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

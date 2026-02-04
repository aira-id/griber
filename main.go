package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aira-id/griber/internal/config"
	"github.com/aira-id/griber/internal/delivery/websocket"
	"github.com/aira-id/griber/internal/usecase/session"
)

func main() {
	// Load configuration from environment and YAML
	cfg := config.LoadWithYAML("config.yaml")

	// Log configuration (without sensitive data)
	log.Printf("Starting Griber STT Server")
	log.Printf("Port: %s", cfg.Server.Port)
	log.Printf("Max audio buffer size: %d bytes", cfg.Audio.MaxBufferSize)
	log.Printf("Max connections per IP: %d", cfg.Rate.MaxConnectionsPerIP)

	if len(cfg.Server.AllowedOrigins) == 0 {
		log.Println("Allowed origins: * (all)")
	} else {
		log.Printf("Allowed origins: %v", cfg.Server.AllowedOrigins)
	}

	if len(cfg.Auth.APIKeys) == 0 {
		log.Println("Authentication: disabled (no API keys configured)")
	} else {
		log.Printf("Authentication: enabled (%d API key(s) configured)", len(cfg.Auth.APIKeys))
	}

	// Initialize Usecase with configuration
	sessionUsecase := session.NewSessionUsecaseWithConfig(cfg)

	// Initialize Delivery Handler
	wsHandler := websocket.NewHandler(sessionUsecase, cfg)

	// Set up routes
	http.Handle("/v1/realtime", wsHandler)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server in a goroutine
	addr := ":" + cfg.Server.Port
	server := &http.Server{
		Addr:    addr,
		Handler: nil, // Uses http.DefaultServeMux
	}

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-quit
	log.Printf("Received signal: %v, shutting down...", sig)

	// Context for shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server and cleanup resources
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server force shutdown: %v", err)
	}

	wsHandler.Close()
	log.Println("Server stopped")
}

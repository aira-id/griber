package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadWithYAML(t *testing.T) {
	// Create a temporary YAML file
	yamlContent := `
server:
  port: "9090"
  allowed_origins:
    - "http://localhost:3000"
auth:
  api_keys:
    - "test-key-1"
audio:
  max_audio_buffer_size: 1000000
  transcription_timeout: "15s"
rate:
  max_connections_per_ip: 5
  requests_per_second: 50
  burst_size: 25
  cleanup_interval: "30s"
asr:
  provider: "gpu"
  num_threads: 8
`
	tmpFile, err := os.CreateTemp("", "config*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(yamlContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Set some environment variables to test precedence (YAML should win for specific fields)
	os.Setenv("GRIBE_PORT", "8888")
	defer os.Unsetenv("GRIBE_PORT")

	cfg := LoadWithYAML(tmpFile.Name())

	// Check if values are correctly loaded and overridden
	if cfg.Server.Port != "9090" {
		t.Errorf("Expected Port 9090 (from YAML), got %s", cfg.Server.Port)
	}

	if len(cfg.Server.AllowedOrigins) != 1 || cfg.Server.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("Expected AllowedOrigins [http://localhost:3000], got %v", cfg.Server.AllowedOrigins)
	}

	if len(cfg.Auth.APIKeys) != 1 || cfg.Auth.APIKeys[0] != "test-key-1" {
		t.Errorf("Expected APIKeys [test-key-1], got %v", cfg.Auth.APIKeys)
	}

	if cfg.Audio.MaxBufferSize != 1000000 {
		t.Errorf("Expected MaxBufferSize 1000000, got %d", cfg.Audio.MaxBufferSize)
	}

	if cfg.Audio.TranscriptionTimeout != 15*time.Second {
		t.Errorf("Expected TranscriptionTimeout 15s, got %v", cfg.Audio.TranscriptionTimeout)
	}

	if cfg.Rate.MaxConnectionsPerIP != 5 {
		t.Errorf("Expected MaxConnectionsPerIP 5, got %d", cfg.Rate.MaxConnectionsPerIP)
	}

	if cfg.Rate.CleanupInterval != 30*time.Second {
		t.Errorf("Expected CleanupInterval 30s, got %v", cfg.Rate.CleanupInterval)
	}

	if cfg.ASR.Provider != "gpu" {
		t.Errorf("Expected ASR Provider gpu, got %s", cfg.ASR.Provider)
	}

	if cfg.ASR.NumThreads != 8 {
		t.Errorf("Expected ASR NumThreads 8, got %d", cfg.ASR.NumThreads)
	}
}

func TestLoadDefaults(t *testing.T) {
	// Test loading when YAML is missing
	os.Unsetenv("GRIBE_PORT")
	cfg := LoadWithYAML("non_existent.yaml")

	if cfg.Server.Port != "8080" {
		t.Errorf("Expected default Port 8080, got %s", cfg.Server.Port)
	}

	if cfg.ASR.Provider != "cpu" {
		t.Errorf("Expected default ASR Provider cpu, got %s", cfg.ASR.Provider)
	}
}

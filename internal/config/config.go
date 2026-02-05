package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application
type Config struct {
	Server ServerConfig
	Auth   AuthConfig
	Audio  AudioConfig
	Rate   RateLimitConfig
	ASR    ASRConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port           string   `yaml:"port"`
	AllowedOrigins []string `yaml:"allowed_origins"` // Empty means allow all (wildcard)
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	APIKeys []string `yaml:"api_keys"` // List of valid API keys, empty means no auth required
}

// AudioConfig holds audio processing limits
type AudioConfig struct {
	MaxBufferSize        int           `yaml:"max_audio_buffer_size"` // Maximum audio buffer size in bytes (default 15MB)
	TranscriptionTimeout time.Duration `yaml:"transcription_timeout"` // Timeout for transcription calls (default 30s)
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	MaxConnectionsPerIP int           `yaml:"max_connections_per_ip"`
	RequestsPerSecond   int           `yaml:"requests_per_second"`
	BurstSize           int           `yaml:"burst_size"`
	CleanupInterval     time.Duration `yaml:"cleanup_interval"`
}

// ASRConfig holds ASR provider configuration loaded from YAML
type ASRConfig struct {
	Provider     string                 `yaml:"provider"`      // cpu or gpu
	NumThreads   int                    `yaml:"num_threads"`   // Number of threads for inference
	ModelsDir    string                 `yaml:"models_dir"`    // Base directory for models
	DefaultModel string                 `yaml:"default_model"` // Default model to use
	Models       map[string]ModelConfig `yaml:"models"`        // Model configurations
}

// RecognizerType represents the type of recognizer (online/offline)
type RecognizerType string

const (
	RecognizerOnline  RecognizerType = "online"  // Streaming/realtime recognition
	RecognizerOffline RecognizerType = "offline" // Batch/non-streaming recognition
)

// ModelConfig holds configuration for a specific ASR model
type ModelConfig struct {
	Provider   string         `yaml:"provider"`   // Provider type (e.g., "sherpa-onnx")
	Recognizer RecognizerType `yaml:"recognizer"` // Recognizer type: "online" (streaming) or "offline" (batch)
	Encoder    string         `yaml:"encoder"`    // Path to encoder model file
	Decoder    string         `yaml:"decoder"`    // Path to decoder model file
	Joiner     string         `yaml:"joiner"`     // Path to joiner model file
	Tokens     string         `yaml:"tokens"`     // Path to tokens file
	Languages  []string       `yaml:"languages"`  // Supported languages
}

// YAMLConfig holds configuration loaded from YAML file
type YAMLConfig struct {
	Server ServerConfig    `yaml:"server"`
	Auth   AuthConfig      `yaml:"auth"`
	Audio  AudioConfig     `yaml:"audio"`
	Rate   RateLimitConfig `yaml:"rate"`
	ASR    ASRConfig       `yaml:"asr"`
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:           getEnv("GRIBE_PORT", "8080"),
			AllowedOrigins: getEnvSlice("GRIBE_ALLOWED_ORIGINS", nil), // nil = wildcard
		},
		Auth: AuthConfig{
			APIKeys: getEnvSlice("GRIBE_API_KEYS", nil), // nil = no auth required
		},
		Audio: AudioConfig{
			MaxBufferSize:        getEnvInt("GRIBE_MAX_AUDIO_BUFFER_SIZE", 15*1024*1024), // 15MB default
			TranscriptionTimeout: time.Duration(getEnvInt("GRIBE_TRANSCRIPTION_TIMEOUT_SECONDS", 30)) * time.Second,
		},
		Rate: RateLimitConfig{
			MaxConnectionsPerIP: getEnvInt("GRIBE_MAX_CONNECTIONS_PER_IP", 10),
			RequestsPerSecond:   getEnvInt("GRIBE_REQUESTS_PER_SECOND", 100),
			BurstSize:           getEnvInt("GRIBE_RATE_BURST_SIZE", 50),
			CleanupInterval:     time.Duration(getEnvInt("GRIBE_RATE_CLEANUP_SECONDS", 60)) * time.Second,
		},
	}
}

// IsOriginAllowed checks if the given origin is allowed
func (c *Config) IsOriginAllowed(origin string) bool {
	// If no origins configured, allow all (wildcard)
	if len(c.Server.AllowedOrigins) == 0 {
		return true
	}

	// Check if origin matches any allowed origin
	for _, allowed := range c.Server.AllowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
	}
	return false
}

// IsAPIKeyValid checks if the given API key is valid
func (c *Config) IsAPIKeyValid(apiKey string) bool {
	// If no API keys configured, allow all (no auth required)
	if len(c.Auth.APIKeys) == 0 {
		return true
	}

	// Check if key matches any configured key
	for _, validKey := range c.Auth.APIKeys {
		if validKey == apiKey {
			return true
		}
	}
	return false
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	// Split by comma and trim whitespace
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}
	return result
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intVal
}

// LoadYAML loads the configuration from a YAML file
func LoadYAML(path string) (*YAMLConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg YAMLConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadWithYAML loads configuration from environment variables and YAML file
func LoadWithYAML(yamlPath string) *Config {
	// 1. Start with environment variables (and defaults)
	cfg := Load()

	// 2. Try to load YAML config
	yamlCfg, err := LoadYAML(yamlPath)
	if err != nil {
		log.Printf("Warning: Could not load YAML config from %s: %v", yamlPath, err)
		// Set defaults for ASR config if YAML fails
		cfg.ASR = ASRConfig{
			Provider:   "cpu",
			NumThreads: 4,
			ModelsDir:  "./models",
			Models:     make(map[string]ModelConfig),
		}
		return cfg
	}

	// 3. Override with YAML values if present
	if yamlCfg.Server.Port != "" {
		cfg.Server.Port = yamlCfg.Server.Port
	}
	if len(yamlCfg.Server.AllowedOrigins) > 0 {
		cfg.Server.AllowedOrigins = yamlCfg.Server.AllowedOrigins
	}

	if len(yamlCfg.Auth.APIKeys) > 0 {
		cfg.Auth.APIKeys = yamlCfg.Auth.APIKeys
	}

	if yamlCfg.Audio.MaxBufferSize > 0 {
		cfg.Audio.MaxBufferSize = yamlCfg.Audio.MaxBufferSize
	}
	if yamlCfg.Audio.TranscriptionTimeout > 0 {
		cfg.Audio.TranscriptionTimeout = yamlCfg.Audio.TranscriptionTimeout
	}

	if yamlCfg.Rate.MaxConnectionsPerIP > 0 {
		cfg.Rate.MaxConnectionsPerIP = yamlCfg.Rate.MaxConnectionsPerIP
	}
	if yamlCfg.Rate.RequestsPerSecond > 0 {
		cfg.Rate.RequestsPerSecond = yamlCfg.Rate.RequestsPerSecond
	}
	if yamlCfg.Rate.BurstSize > 0 {
		cfg.Rate.BurstSize = yamlCfg.Rate.BurstSize
	}
	if yamlCfg.Rate.CleanupInterval > 0 {
		cfg.Rate.CleanupInterval = yamlCfg.Rate.CleanupInterval
	}

	// ASR section is mostly YAML-only anyway
	cfg.ASR = yamlCfg.ASR

	// Set ASR defaults if missing in YAML
	if cfg.ASR.Provider == "" {
		cfg.ASR.Provider = "cpu"
	}
	if cfg.ASR.NumThreads == 0 {
		cfg.ASR.NumThreads = 4
	}
	if cfg.ASR.ModelsDir == "" {
		cfg.ASR.ModelsDir = "./models"
	}

	return cfg
}

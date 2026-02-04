package asr

import (
	"context"
	"log"

	"github.com/aira-id/griber/internal/config"
	"github.com/aira-id/griber/internal/domain"
)

// Example functions demonstrating modular ASR provider usage with the registry pattern

// ExampleUsingRegistry demonstrates using the ASR model registry
func ExampleUsingRegistry(cfg *config.ASRConfig) {
	registry := NewASRModelRegistry(cfg)
	defer registry.Close()

	// List available models
	models := registry.GetAvailableModels()
	log.Printf("Available models: %v", models)

	// Get a model (lazy loading - first call loads, subsequent calls reuse)
	provider, err := registry.GetModel("sherpa-onnx-streaming-zipformer2-id", "id")
	if err != nil {
		log.Fatal("Failed to get model:", err)
	}

	log.Printf("Provider loaded: %T", provider)
	log.Printf("Supported models: %v", provider.GetSupportedModels())
	log.Printf("Supported languages: %v", provider.GetSupportedLanguages())

	// Check loaded models (should show the model we just loaded)
	loaded := registry.GetLoadedModels()
	log.Printf("Loaded models: %v", loaded)
}

// ExampleSingletonPattern demonstrates the singleton pattern for model loading
func ExampleSingletonPattern(cfg *config.ASRConfig) {
	registry := NewASRModelRegistry(cfg)
	defer registry.Close()

	modelName := "sherpa-onnx-streaming-zipformer2-id"
	language := "id"

	// First call - loads the model
	log.Println("First call - loading model...")
	provider1, err := registry.GetModel(modelName, language)
	if err != nil {
		log.Fatal("Failed to get model:", err)
	}
	log.Printf("Provider 1: %p", provider1)

	// Second call - reuses the already loaded model
	log.Println("Second call - should reuse loaded model...")
	provider2, err := registry.GetModel(modelName, language)
	if err != nil {
		log.Fatal("Failed to get model:", err)
	}
	log.Printf("Provider 2: %p", provider2)

	// Both should point to the same instance
	if provider1 == provider2 {
		log.Println("✓ Same provider instance reused (singleton pattern working)")
	} else {
		log.Println("✗ Different instances (unexpected)")
	}
}

// ExampleMultipleModels demonstrates loading multiple models
func ExampleMultipleModels(cfg *config.ASRConfig) {
	registry := NewASRModelRegistry(cfg)
	defer registry.Close()

	// Load multiple models
	models := registry.GetAvailableModels()
	for _, modelName := range models {
		// Get supported languages for this model
		languages, err := registry.GetModelLanguages(modelName)
		if err != nil {
			log.Printf("Failed to get languages for %s: %v", modelName, err)
			continue
		}

		if len(languages) == 0 {
			log.Printf("Model %s has no languages defined", modelName)
			continue
		}

		// Load model with first supported language
		provider, err := registry.GetModel(modelName, languages[0])
		if err != nil {
			log.Printf("Failed to load model %s: %v", modelName, err)
			continue
		}

		log.Printf("✓ Loaded model: %s (language: %s)", modelName, languages[0])
		log.Printf("  Provider type: %T", provider)
	}

	// Check all loaded models
	loaded := registry.GetLoadedModels()
	log.Printf("Total loaded models: %d", len(loaded))
}

// ExampleTranscribeWithRegistry demonstrates transcribing with registry-managed providers
func ExampleTranscribeWithRegistry(cfg *config.ASRConfig, audioBytes []byte) {
	if len(audioBytes) == 0 {
		log.Println("No audio data provided")
		return
	}

	registry := NewASRModelRegistry(cfg)
	defer registry.Close()

	ctx := context.Background()

	// Get provider for Indonesian model
	provider, err := registry.GetModel("sherpa-onnx-streaming-zipformer2-id", "id")
	if err != nil {
		log.Fatal("Failed to get model:", err)
	}

	transcConfig := &domain.TranscriptionConfig{
		Model:    "sherpa-onnx-streaming-zipformer2-id",
		Language: "id",
	}

	log.Println("Transcribing with Indonesian model...")
	results, err := provider.Transcribe(ctx, audioBytes, transcConfig)
	if err != nil {
		log.Fatal("Transcription error:", err)
	}

	for chunk := range results {
		log.Printf("  [%v] %s", chunk.IsFinal, chunk.Text)
	}
}

// ExampleProviderCapabilities demonstrates checking provider capabilities via registry
func ExampleProviderCapabilities(cfg *config.ASRConfig) {
	registry := NewASRModelRegistry(cfg)
	defer registry.Close()

	models := registry.GetAvailableModels()

	for _, modelName := range models {
		languages, err := registry.GetModelLanguages(modelName)
		if err != nil {
			log.Printf("⚠️  %s: Failed to get languages (%v)", modelName, err)
			continue
		}

		if len(languages) == 0 {
			log.Printf("⚠️  %s: No languages defined", modelName)
			continue
		}

		// Try to load the model
		provider, err := registry.GetModel(modelName, languages[0])
		if err != nil {
			log.Printf("⚠️  %s: Failed to load (%v)", modelName, err)
			continue
		}

		supportedModels := provider.GetSupportedModels()
		supportedLanguages := provider.GetSupportedLanguages()

		log.Printf("✓ %s", modelName)
		log.Printf("  Languages: %v", languages)
		log.Printf("  Provider models: %d", len(supportedModels))
		log.Printf("  Provider languages: %d", len(supportedLanguages))
	}
}

// Helper function
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

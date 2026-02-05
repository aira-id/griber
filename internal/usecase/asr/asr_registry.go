package asr

import (
	"fmt"
	"log"
	"sync"

	"github.com/aira-id/griber/internal/config"
	"github.com/aira-id/griber/internal/domain"
	"github.com/aira-id/griber/internal/pkg/sherpa"
)

// ASRModelRegistry manages ASR provider instances with singleton pattern.
// Models are loaded lazily on first request and reused across sessions.
type ASRModelRegistry struct {
	mu            sync.RWMutex
	globalConfig  *config.ASRConfig
	loadedModels  map[string]domain.ASRProvider // modelName -> provider instance
	providerTypes map[ASRProviderType]ProviderCreator
}

// ProviderCreator is a function that creates an ASR provider from config
type ProviderCreator func(globalConfig *config.ASRConfig, modelName string, modelConfig *config.ModelConfig) (domain.ASRProvider, error)

// NewASRModelRegistry creates a new registry with the given config
func NewASRModelRegistry(cfg *config.ASRConfig) *ASRModelRegistry {
	registry := &ASRModelRegistry{
		globalConfig:  cfg,
		loadedModels:  make(map[string]domain.ASRProvider),
		providerTypes: make(map[ASRProviderType]ProviderCreator),
	}

	// Register built-in provider creators
	registry.RegisterProviderType(ProviderSherpaOnnx, createSherpaProvider)

	return registry
}

// RegisterProviderType registers a provider creator for a given type
func (r *ASRModelRegistry) RegisterProviderType(providerType ASRProviderType, creator ProviderCreator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providerTypes[providerType] = creator
}

// GetModel returns an ASR provider for the given model and language.
// If the model is already loaded, returns the existing instance.
// If not, loads the model lazily.
func (r *ASRModelRegistry) GetModel(modelName, language string) (domain.ASRProvider, error) {
	if r.globalConfig == nil {
		return nil, fmt.Errorf("ASR configuration not available")
	}

	// Validate model exists
	modelConfig, exists := r.globalConfig.Models[modelName]
	if !exists {
		availableModels := r.GetAvailableModels()
		return nil, fmt.Errorf("model '%s' not found. Available models: %v", modelName, availableModels)
	}

	// Validate language is supported
	if language == "" {
		return nil, fmt.Errorf("language is required")
	}

	languageSupported := false
	for _, lang := range modelConfig.Languages {
		if lang == language {
			languageSupported = true
			break
		}
	}
	if !languageSupported {
		return nil, fmt.Errorf("language '%s' is not supported by model '%s'. Supported languages: %v",
			language, modelName, modelConfig.Languages)
	}

	// Check if model is already loaded (read lock)
	r.mu.RLock()
	if provider, loaded := r.loadedModels[modelName]; loaded {
		r.mu.RUnlock()
		log.Printf("[INFO] Reusing already loaded model: %s", modelName)
		return provider, nil
	}
	r.mu.RUnlock()

	// Model not loaded, need to load it (write lock)
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine might have loaded it)
	if provider, loaded := r.loadedModels[modelName]; loaded {
		log.Printf("[INFO] Reusing already loaded model (after lock): %s", modelName)
		return provider, nil
	}

	// Get provider type from model config
	providerType := ASRProviderType(modelConfig.Provider)
	if providerType == "" {
		return nil, fmt.Errorf("model '%s' does not specify a provider type", modelName)
	}

	// Get creator for this provider type
	creator, exists := r.providerTypes[providerType]
	if !exists {
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	// Load the model
	log.Printf("[INFO] Loading model: %s (provider: %s)", modelName, providerType)
	provider, err := creator(r.globalConfig, modelName, &modelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load model '%s': %w", modelName, err)
	}

	// Cache the loaded provider
	r.loadedModels[modelName] = provider
	log.Printf("[INFO] Successfully loaded and cached model: %s", modelName)

	return provider, nil
}

// PreloadModel triggers loading of a specific model
func (r *ASRModelRegistry) PreloadModel(modelName string) error {
	// Get first supported language for this model
	langs, err := r.GetModelLanguages(modelName)
	if err != nil {
		return err
	}
	if len(langs) == 0 {
		return fmt.Errorf("model '%s' has no supported languages", modelName)
	}

	_, err = r.GetModel(modelName, langs[0])
	return err
}

// PreloadDefaultModel preloads the default model specified in global config
func (r *ASRModelRegistry) PreloadDefaultModel() error {
	if r.globalConfig == nil || r.globalConfig.DefaultModel == "" {
		return nil
	}
	return r.PreloadModel(r.globalConfig.DefaultModel)
}

// GetAvailableModels returns a list of available model names
func (r *ASRModelRegistry) GetAvailableModels() []string {
	if r.globalConfig == nil {
		return nil
	}

	models := make([]string, 0, len(r.globalConfig.Models))
	for name := range r.globalConfig.Models {
		models = append(models, name)
	}
	return models
}

// GetModelLanguages returns supported languages for a model
func (r *ASRModelRegistry) GetModelLanguages(modelName string) ([]string, error) {
	if r.globalConfig == nil {
		return nil, fmt.Errorf("ASR configuration not available")
	}

	modelConfig, exists := r.globalConfig.Models[modelName]
	if !exists {
		return nil, fmt.Errorf("model '%s' not found", modelName)
	}

	return modelConfig.Languages, nil
}

// IsModelOnline returns true if the model is configured for online (streaming) recognition
func (r *ASRModelRegistry) IsModelOnline(modelName string) (bool, error) {
	if r.globalConfig == nil {
		return false, fmt.Errorf("ASR configuration not available")
	}

	modelConfig, exists := r.globalConfig.Models[modelName]
	if !exists {
		return false, fmt.Errorf("model '%s' not found", modelName)
	}

	return modelConfig.Recognizer == config.RecognizerOnline || modelConfig.Recognizer == "", nil
}

// IsModelOffline returns true if the model is configured for offline (batch) recognition
func (r *ASRModelRegistry) IsModelOffline(modelName string) (bool, error) {
	if r.globalConfig == nil {
		return false, fmt.Errorf("ASR configuration not available")
	}

	modelConfig, exists := r.globalConfig.Models[modelName]
	if !exists {
		return false, fmt.Errorf("model '%s' not found", modelName)
	}

	return modelConfig.Recognizer == config.RecognizerOffline, nil
}

// IsModelLoaded checks if a model is already loaded
func (r *ASRModelRegistry) IsModelLoaded(modelName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, loaded := r.loadedModels[modelName]
	return loaded
}

// GetLoadedModels returns a list of currently loaded model names
func (r *ASRModelRegistry) GetLoadedModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]string, 0, len(r.loadedModels))
	for name := range r.loadedModels {
		models = append(models, name)
	}
	return models
}

// Close closes all loaded models and clears the registry
func (r *ASRModelRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	for name, provider := range r.loadedModels {
		if closer, ok := provider.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Printf("[WARN] Failed to close model '%s': %v", name, err)
				lastErr = err
			}
		}
	}

	r.loadedModels = make(map[string]domain.ASRProvider)
	return lastErr
}

// Provider creator functions

func createSherpaProvider(globalConfig *config.ASRConfig, modelName string, modelConfig *config.ModelConfig) (domain.ASRProvider, error) {
	// Map config recognizer type to sherpa recognizer type
	recognizerType := sherpa.RecognizerOnline // Default to online
	if modelConfig.Recognizer == config.RecognizerOffline {
		recognizerType = sherpa.RecognizerOffline
	}

	sherpaConfig := &sherpa.Config{
		Provider:   globalConfig.Provider,
		NumThreads: globalConfig.NumThreads,
		ModelsDir:  globalConfig.ModelsDir,
		ModelName:  modelName,
		Recognizer: recognizerType,
		Encoder:    modelConfig.Encoder,
		Decoder:    modelConfig.Decoder,
		Joiner:     modelConfig.Joiner,
		Tokens:     modelConfig.Tokens,
		Languages:  modelConfig.Languages,
		// Note: Language is set per-transcription, not per-model
		Language: modelConfig.Languages[0], // Default to first language
	}

	return sherpa.New(sherpaConfig)
}

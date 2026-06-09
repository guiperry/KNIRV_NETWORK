// /home/gperry/Documents/GitHub/Inc-Line/Wordpress-Inference-Engine/inference/inference_service.go
package inferencer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/guiperry/text-embedder/pkg/embed"
	gollm "github.com/guiperry/gollm_cerebras"
	"github.com/guiperry/gollm_cerebras/config"
	"github.com/guiperry/gollm_cerebras/llm"
)

var promptSimilarityThreshold = func() float32 {
	if v := os.Getenv("KNIRV_PROMPT_CACHE_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 32); err == nil {
			return float32(f)
		}
	}
	return 0.97
}()

type cachedResponse struct {
	vec      []float32
	response string
}

// LLMAttemptConfig defines the configuration for a single LLM attempt.
type LLMAttemptConfig struct {
	ProviderName string
	ModelName    string
	APIKeyEnvVar string // Environment variable name for the API key
	BaseURL      string // Optional custom base URL for the provider
	MaxTokens    int
	IsPrimary    bool // True if part of initial attempts, false for fallback
}

// LLMAttempt holds an initialized LLM instance and its config.
type LLMAttempt struct {
	Instance llm.LLM
	Config   LLMAttemptConfig
	Opts     []config.ConfigOption // ADDED: Store the options used to create this instance
}

// InferenceService manages the interaction with the gollm library and its providers.
type InferenceService struct {
	// Store lists of attempts instead of single instances
	primaryAttempts   []LLMAttempt
	fallbackAttempts  []LLMAttempt
	delegator         *DelegatorService
	orchestrator      TaskOrchestratorInterface // ADDED: Orchestrator for complex, multi-step tasks
	db                DatabaseAccessor          // ADDED: Use the DatabaseAccessor interface
	contextStrategist *ContextStrategist        // ADDED: Context Manager instance
	isRunning         bool
	mutex             sync.Mutex
	moa               *gollm.MOA
	// Store names/config options for MOA defaults, separate from execution attempts
	moaPrimaryModelName  string
	moaFallbackModelName string
	moaPrimaryOpts       []config.ConfigOption
	moaFallbackOpts      []config.ConfigOption
	// Inference switcher for internal transformer routing
	switcher *InferenceSwitcher
	// Semantic prompt response cache
	responseCacheMu sync.Mutex
	responseCache   []cachedResponse
	// Chat session management
	chatSessions *ChatSessions
}

// NewInferenceService creates a new instance of InferenceService.
func NewInferenceService(db DatabaseAccessor) (*InferenceService, error) {
	return &InferenceService{
		// Initialize slices
		primaryAttempts:  make([]LLMAttempt, 0),
		fallbackAttempts: make([]LLMAttempt, 0),
		// Initialize ContextStrategist with default strategy
		contextStrategist: NewContextStrategist(
			nil,                                      // Orchestrator is not yet created, will be set later.
			ChunkByTokenCount,                        // Use token count for better splitting
			WithProcessingMode(SequentialProcessing), // Default to sequential
		),
		db: db, // Store the provided database accessor
	}, nil
}

// StartWithConfig configures the service with dynamic LLM configurations and starts it.
func (s *InferenceService) StartWithConfig(attemptConfigs []LLMAttemptConfig, plannerModel string, executorModels []string, finalizerModel string, verifierModel string) error {
	log.Println("InferenceService: Starting with dynamic configuration...")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.primaryAttempts = make([]LLMAttempt, 0)
	s.fallbackAttempts = make([]LLMAttempt, 0)
	var primaryOptsList [][]config.ConfigOption
	var fallbackOptsList [][]config.ConfigOption

	// Initialize LLM instances based on provided config
	for _, attemptConf := range attemptConfigs {
		log.Printf("InferenceService: Configuring LLM attempt: Provider=%s, Model=%s, Primary=%t", attemptConf.ProviderName, attemptConf.ModelName, attemptConf.IsPrimary)
		apiKey := os.Getenv(attemptConf.APIKeyEnvVar)
		if apiKey == "" {
			log.Printf("[WARN] InferenceService: API Key from env var '%s' not found for model '%s'. Skipping this attempt.", attemptConf.APIKeyEnvVar, attemptConf.ModelName)
			continue
		}

		opts := []config.ConfigOption{
			config.SetProvider(attemptConf.ProviderName),
			config.SetAPIKey(apiKey),
			config.SetModel(attemptConf.ModelName),
			config.SetMaxTokens(attemptConf.MaxTokens),
			// Set longer timeout to prevent context deadline exceeded errors
			config.SetTimeout(90 * time.Second),
		}

		// Add custom base URL if provided (for CloudFlare and other custom endpoints)
		if attemptConf.BaseURL != "" {
			// Note: gollm_cerebras config package doesn't have SetEndpoint function
			// The BaseURL is handled internally by the provider implementations
			log.Printf("InferenceService: Custom BaseURL '%s' provided but SetEndpoint not available in config package", attemptConf.BaseURL)
		}

		llmInstance, err := gollm.NewLLM(opts...)
		if err != nil {
			log.Printf("[ERROR] InferenceService: Failed to create LLM instance for model '%s': %v. Skipping this attempt.", attemptConf.ModelName, err)
			continue
		}

		if initializedLLM, ok := llmInstance.(llm.LLM); ok {
			attempt := LLMAttempt{
				Instance: initializedLLM,
				Config:   attemptConf,
				Opts:     opts,
			}
			if attemptConf.IsPrimary {
				s.primaryAttempts = append(s.primaryAttempts, attempt)
				primaryOptsList = append(primaryOptsList, opts)
			} else {
				s.fallbackAttempts = append(s.fallbackAttempts, attempt)
				fallbackOptsList = append(fallbackOptsList, opts)
			}
			log.Printf("InferenceService: Successfully configured LLM instance for model '%s'", attemptConf.ModelName)
		} else {
			log.Printf("[ERROR] InferenceService: Initialized instance for model '%s' is not of type llm.LLM. Skipping.", attemptConf.ModelName)
		}
	}

	// Validate that we have at least one primary and one fallback
	if len(s.primaryAttempts) == 0 {
		return fmt.Errorf("inference service configuration error: no primary LLM attempts were successfully initialized")
	}
	if len(s.fallbackAttempts) == 0 {
		return fmt.Errorf("inference service configuration error: no fallback LLM attempts were successfully initialized")
	}

	// Set initial MOA defaults based on the first primary and last fallback attempt
	s.moaPrimaryModelName = s.primaryAttempts[0].Config.ModelName
	s.moaFallbackModelName = s.fallbackAttempts[len(s.fallbackAttempts)-1].Config.ModelName
	s.moaPrimaryOpts = primaryOptsList[0]
	s.moaFallbackOpts = fallbackOptsList[len(fallbackOptsList)-1]

	// Create the initial MOA instance
	if err := s.reconfigureMOAInternal(); err != nil {
		log.Printf("[WARN] InferenceService: Initial MOA configuration failed: %v. MOA features disabled.", err)
	}

	// Create the Delegator Service first (required for TaskOrchestrator)
	delegatorTokenLimit := s.primaryAttempts[0].Config.MaxTokens
	delegatorTokenModel := s.primaryAttempts[0].Config.ModelName
	s.delegator = NewDelegatorService(s.primaryAttempts, s.fallbackAttempts, delegatorTokenLimit, delegatorTokenModel, s.moa, s.contextStrategist)
	if s.delegator == nil {
		log.Println("[ERROR] InferenceService: Failed to create DelegatorService.")
		s.isRunning = false
		s.moa = nil
		return fmt.Errorf("failed to create delegator service")
	}
	log.Println("InferenceService: DelegatorService created.")

	// --- Create the Task Orchestrator ---
	s.orchestrator = NewTaskOrchestrator(s.delegator, plannerModel, executorModels, finalizerModel, verifierModel)
	if s.orchestrator == nil {
		return fmt.Errorf("failed to create task orchestrator")
	}
	log.Println("InferenceService: TaskOrchestrator created.")

	// Now that the orchestrator exists, link it to the strategist.
	s.contextStrategist.orchestrator = s.orchestrator

	s.isRunning = true
	log.Println("InferenceService: Started successfully with dynamic configuration.")
	return nil
}

// providerChain defines the ordered fallback chain of LLM providers.
// Each entry specifies the provider name, model, env var for the API key, and max tokens.
// When a provider returns a 429 (rate-limit/quota) error, the system falls through
// to the next provider in the chain. If no API key is found for a provider, it's skipped.
var providerChain = []struct {
	ProviderName string
	ModelName    string
	APIKeyEnvVar string
	MaxTokens    int
}{
	{ProviderName: "deepseek", ModelName: "deepseek-chat", APIKeyEnvVar: "DEEPSEEK_API_KEY", MaxTokens: 8000},
	{ProviderName: "gemini", ModelName: "gemini-2.5-flash", APIKeyEnvVar: "GEMINI_API_KEY", MaxTokens: 100000},
	{ProviderName: "cerebras", ModelName: "cerebras/Llama-3.3-70B", APIKeyEnvVar: "CEREBRAS_API_KEY", MaxTokens: 8000},
	{ProviderName: "openai", ModelName: "gpt-4o", APIKeyEnvVar: "OPENAI_API_KEY", MaxTokens: 8000},
	{ProviderName: "anthropic", ModelName: "claude-sonnet-4-20250514", APIKeyEnvVar: "ANTHROPIC_API_KEY", MaxTokens: 8000},
}

// Start configures the service with the ordered provider fallback chain.
// Deepseek is the primary; all others are fallbacks tried in order on 429/quota errors.
func (s *InferenceService) Start() error {
	log.Println("InferenceService: Starting with provider fallback chain...")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// --- Build the attempt configs from the provider chain ---
	// The first available provider is the primary; the rest are fallbacks.
	var attemptConfigs []LLMAttemptConfig
	foundPrimary := false

	for _, provider := range providerChain {
		apiKey := os.Getenv(provider.APIKeyEnvVar)
		if apiKey == "" {
			log.Printf("[WARN] InferenceService: API Key from env var '%s' not found. Skipping provider '%s'.", provider.APIKeyEnvVar, provider.ProviderName)
			continue
		}

		isPrimary := !foundPrimary
		foundPrimary = true

		attemptConfigs = append(attemptConfigs, LLMAttemptConfig{
			ProviderName: provider.ProviderName,
			ModelName:    provider.ModelName,
			APIKeyEnvVar: provider.APIKeyEnvVar,
			MaxTokens:    provider.MaxTokens,
			IsPrimary:    isPrimary,
		})
		log.Printf("InferenceService: Will configure %s (%s) as %s", provider.ProviderName, provider.ModelName, map[bool]string{true: "PRIMARY", false: "FALLBACK"}[isPrimary])
	}

	if len(attemptConfigs) == 0 {
		return fmt.Errorf("inference service configuration error: no LLM providers could be configured (no API keys found)")
	}

	s.primaryAttempts = make([]LLMAttempt, 0)
	s.fallbackAttempts = make([]LLMAttempt, 0)
	var primaryOptsList [][]config.ConfigOption  // For MOA
	var fallbackOptsList [][]config.ConfigOption // For MOA (aggregator might use last fallback)

	// --- Initialize LLM instances based on config ---
	for _, attemptConf := range attemptConfigs {
		log.Printf("InferenceService: Configuring LLM attempt: Provider=%s, Model=%s, Primary=%t", attemptConf.ProviderName, attemptConf.ModelName, attemptConf.IsPrimary)
		apiKey := os.Getenv(attemptConf.APIKeyEnvVar)
		if apiKey == "" {
			log.Printf("[WARN] InferenceService: API Key from env var '%s' not found for model '%s'. Skipping this attempt.", attemptConf.APIKeyEnvVar, attemptConf.ModelName)
			continue // Skip this attempt if key is missing
		}

		opts := []config.ConfigOption{
			config.SetProvider(attemptConf.ProviderName),
			config.SetAPIKey(apiKey),
			config.SetModel(attemptConf.ModelName),
			config.SetMaxTokens(attemptConf.MaxTokens),
			// Set longer timeout to prevent context deadline exceeded errors
			config.SetTimeout(90 * time.Second),
		}

		llmInstance, err := gollm.NewLLM(opts...)
		if err != nil {
			log.Printf("[ERROR] InferenceService: Failed to create LLM instance for model '%s': %v. Skipping this attempt.", attemptConf.ModelName, err)
			continue // Skip this attempt on error
		}

		if initializedLLM, ok := llmInstance.(llm.LLM); ok {
			attempt := LLMAttempt{
				Instance: initializedLLM,
				Config:   attemptConf,
				Opts:     opts, // STORE THE OPTS
			}
			if attemptConf.IsPrimary {
				s.primaryAttempts = append(s.primaryAttempts, attempt)
				primaryOptsList = append(primaryOptsList, opts)
			} else {
				s.fallbackAttempts = append(s.fallbackAttempts, attempt)
				fallbackOptsList = append(fallbackOptsList, opts)
			}
			log.Printf("InferenceService: Successfully configured LLM instance: Provider=%s, Model=%s, Role=%s", attemptConf.ProviderName, attemptConf.ModelName, map[bool]string{true: "Primary", false: "Fallback"}[attemptConf.IsPrimary])
		} else {
			log.Printf("[ERROR] InferenceService: Initialized instance for model '%s' is not of type llm.LLM. Skipping.", attemptConf.ModelName)
		}
	}

	// --- Validate that we have at least one primary and one fallback ---
	if len(s.primaryAttempts) == 0 {
		return fmt.Errorf("inference service configuration error: no primary LLM attempts were successfully initialized")
	}
	if len(s.fallbackAttempts) == 0 {
		return fmt.Errorf("inference service configuration error: no fallback LLM attempts were successfully initialized")
	}

	// --- Initial MOA Configuration ---
	// Set initial MOA defaults based on the first primary and last fallback attempt
	s.moaPrimaryModelName = s.primaryAttempts[0].Config.ModelName
	s.moaFallbackModelName = s.fallbackAttempts[len(s.fallbackAttempts)-1].Config.ModelName
	s.moaPrimaryOpts = primaryOptsList[0]
	s.moaFallbackOpts = fallbackOptsList[len(fallbackOptsList)-1]

	// Attempt to create the initial MOA instance
	if err := s.reconfigureMOAInternal(); err != nil {
		log.Printf("[WARN] InferenceService: Initial MOA configuration failed: %v. MOA features disabled.", err)
	} // Removed incorrect 'else' block that was setting s.moa = nil on success
	// --- End MOA Creation ---

	// --- Create the Delegator Service ---
	// Pass the lists of attempts and the MOA instance
	// The first primary attempt's config determines the initial token limit check
	delegatorTokenLimit := s.primaryAttempts[0].Config.MaxTokens
	delegatorTokenModel := s.primaryAttempts[0].Config.ModelName // Model used for token estimation // Pass contextStrategist to DelegatorService
	s.delegator = NewDelegatorService(s.primaryAttempts, s.fallbackAttempts, delegatorTokenLimit, delegatorTokenModel, s.moa, s.contextStrategist)
	if s.delegator == nil {
		log.Println("[ERROR] InferenceService: Failed to create DelegatorService.") // Corrected log message
		s.isRunning = false
		// Clear attempts?
		s.moa = nil
		return fmt.Errorf("failed to create delegator service")
	}
	log.Println("InferenceService: DelegatorService created.")

	s.isRunning = true
	log.Println("InferenceService: Started successfully.")
	return nil
}

// Stop cleans up the clients and delegator
func (s *InferenceService) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.isRunning {
		return nil
	}
	s.isRunning = false
	s.primaryAttempts = nil // Clear attempts
	s.fallbackAttempts = nil
	s.moa = nil // Clear MOA instance
	s.moaPrimaryOpts = nil
	s.moaFallbackOpts = nil
	s.delegator = nil    // Clear delegator
	s.orchestrator = nil // Clear orchestrator // s.contextStrategist = nil // Keep context manager? Or re-init on Start? Let's keep it.
	// s.contextManager = nil // Keep context manager? Or re-init on Start? Let's keep it.
	log.Println("InferenceService stopped.")
	return nil
}

// GenerateTextWithContext delegates to the DelegatorService or the internal
// transformer (via InferenceSwitcher) when the internal model is ready.
func (s *InferenceService) GenerateTextWithContext(ctx context.Context, modelName string, promptText string, instructionText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning {
		s.mutex.Unlock()
		return "", errors.New("inference service is not running")
	}
	switcher := s.switcher
	delegatorInstance := s.delegator
	s.mutex.Unlock()

	// Check prompt cache first
	if cachedResponse, found := s.lookupCache(promptText); found {
		log.Println("InferenceService: cache hit, returning cached response")
		return cachedResponse, nil
	}

	// Route through internal transformer when ready and no specific model requested
	if modelName == "" && instructionText == "" && switcher != nil && switcher.IsReady() {
		log.Println("InferenceService: routing to internal transformer via InferenceSwitcher")
		response, err := switcher.GenerateText(ctx, promptText)
		if err != nil {
			return "", err
		}
		s.storeCache(promptText, response)
		return response, nil
	}

	if delegatorInstance == nil {
		return "", errors.New("inference service: delegator not configured")
	}

	log.Printf("InferenceService: Delegating generation request to DelegatorService. Model: '%s', Instruction: '%s'", modelName, instructionText)
	response, err := delegatorInstance.GenerateSimple(ctx, modelName, promptText, instructionText)
	if err != nil {
		return "", err
	}

	// Store in response cache
	s.storeCache(promptText, response)

	log.Println("InferenceService: Generation successful via DelegatorService.")
	return response, nil
}

// GenerateTextWithoutContext implements the validation.InferenceClient interface (without context)
func (s *InferenceService) GenerateText(modelName string, promptText string, instructionText string) (string, error) {
	return s.GenerateTextWithContext(context.Background(), modelName, promptText, instructionText)
}

// Generate implements the validation.InferenceClient interface
func (s *InferenceService) Generate(ctx context.Context, prompt string, options interface{}) (string, error) {
	// For now, we'll use a default model and ignore options
	// This can be enhanced later to handle options properly
	defaultModel := "default"
	return s.GenerateTextWithContext(ctx, defaultModel, prompt, "")
}

// --- ADDED: GenerateTextWithProvider ---
// GenerateTextWithProvider sends a prompt directly to the first configured instance of a specific provider.
func (s *InferenceService) GenerateTextWithProvider(providerName string, promptText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning {
		s.mutex.Unlock()
		return "", errors.New("inference service is not running")
	}
	// Find the specific LLM instance
	llmInstance := s.findLLMInstance(providerName)
	if llmInstance == nil {
		s.mutex.Unlock()
		return "", fmt.Errorf("provider '%s' not found or not configured", providerName)
	}
	s.mutex.Unlock() // Unlock before making the potentially long call

	ctx := context.Background() // Consider allowing context passing
	log.Printf("InferenceService: Delegating direct generation request to provider '%s'...", providerName)

	// Use the llm.NewPrompt helper from the gollm library
	prompt := llm.NewPrompt(promptText)

	return llmInstance.Generate(ctx, prompt)
}

// --- ADDED: GenerateTextWithMOA ---
// GenerateTextWithMOA directly delegates to the MOA instance.
func (s *InferenceService) GenerateTextWithMOA(promptText string, instructionText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning {
		s.mutex.Unlock()
		return "", errors.New("inference service is not running")
	}
	if s.moa == nil {
		s.mutex.Unlock()
		return "", errors.New("MOA (Mixture of Agents) is not configured or failed to initialize")
	}
	moaInstance := s.moa // Capture instance under lock
	s.mutex.Unlock()

	ctx := context.Background() // Consider allowing context passing
	log.Printf("InferenceService: Delegating generation request to MOA. Instruction: '%s'", instructionText)

	combinedPrompt := promptText
	if instructionText != "" {
		combinedPrompt = "Instructions:\n" + instructionText + "\n\n---\n\n" + promptText
	}

	// Note: MOA's Generate might have its own internal timeouts based on AgentTimeout
	response, err := moaInstance.Generate(ctx, combinedPrompt)
	if err != nil {
		log.Printf("InferenceService: Direct MOA generation failed: %v", err)
		return "", fmt.Errorf("MOA generation failed: %w", err)
	}
	log.Println("InferenceService: Direct generation successful via MOA.")
	return response, nil
}

// --- ADDED: GenerateTextWithContextStrategist ---
// Explicitly trigger context manager processing (useful for testing or specific UI actions)
func (s *InferenceService) GenerateTextWithContextStrategist(promptText, instruction string, llmProviderName string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.contextStrategist == nil {
		s.mutex.Unlock()
		return "", errors.New("service not running or context manager not configured")
	}
	// Find the LLM instance to use (e.g., Deepseek) - simplified lookup
	llmInstance := s.findLLMInstance(llmProviderName) // Need to implement findLLMInstance
	if llmInstance == nil {
		s.mutex.Unlock()
		return "", fmt.Errorf("LLM provider '%s' not found or configured", llmProviderName)
	}
	ctxMgr := s.contextStrategist
	s.mutex.Unlock()

	ctx := context.Background()
	log.Printf("InferenceService: Explicitly calling ContextStrategist with provider %s", llmProviderName)
	// Adapt llmInstance to TextGenerator interface if needed
	// Wrap the LLM in our adapter to implement TextGenerator
	wrappedLLM := &LLMAdapter{LLM: llmInstance, ProviderName: llmProviderName} // Pass ProviderName
	return ctxMgr.executeCompaction(ctx, wrappedLLM, promptText, instruction)
}

// ExecuteComplexTask delegates a complex task to the orchestrator.
func (s *InferenceService) ExecuteComplexTask(ctx context.Context, complexPrompt string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.orchestrator == nil {
		s.mutex.Unlock()
		return "", errors.New("inference service is not running or orchestrator not configured")
	}
	orchestratorInstance := s.orchestrator
	s.mutex.Unlock()

	log.Println("InferenceService: Delegating complex task to TaskOrchestrator...")
	response, err := orchestratorInstance.ExecuteComplexTask(ctx, complexPrompt)

	return response, err
}

// --- Update other generation methods to use DelegatorService ---

func (s *InferenceService) GenerateTextWithCoT(ctx context.Context, promptText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.delegator == nil {
		s.mutex.Unlock()
		return "", errors.New("service not running")
	}
	delegatorInstance := s.delegator
	s.mutex.Unlock()
	log.Println("InferenceService: Delegating CoT generation to DelegatorService...")
	return delegatorInstance.GenerateWithCoT(ctx, promptText) // Call delegator
}

func (s *InferenceService) GenerateTextWithReflection(ctx context.Context, promptText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.delegator == nil {
		s.mutex.Unlock()
		return "", errors.New("service not running")
	}
	delegatorInstance := s.delegator
	s.mutex.Unlock()
	log.Println("InferenceService: Delegating Reflection generation to DelegatorService...")
	return delegatorInstance.GenerateWithReflection(ctx, promptText) // Call delegator
}

func (s *InferenceService) GenerateStructuredOutput(content string, schema string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.delegator == nil {
		s.mutex.Unlock()
		return "", errors.New("service not running")
	}
	delegatorInstance := s.delegator
	s.mutex.Unlock()
	ctx := context.Background()
	log.Println("InferenceService: Delegating structured output generation to DelegatorService...")
	return delegatorInstance.GenerateStructuredOutput(ctx, content, schema) // Call delegator
}

// --- Model Setting Methods ---
// SetMOAPrimaryModel sets the default primary model used for MOA configuration.
// This does NOT change the primary execution/fallback list.
func (s *InferenceService) SetMOAPrimaryModel(modelName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return errors.New("service is not running")
	}

	// Find the config options for the requested model from the loaded primary attempts
	var foundOpts []config.ConfigOption
	for _, attempt := range s.primaryAttempts {
		if attempt.Config.ModelName == modelName {
			foundOpts = attempt.Opts // Use stored opts
			break
		}
	}

	if foundOpts == nil {
		return fmt.Errorf("model '%s' not found in the configured primary attempts", modelName)
	}

	s.moaPrimaryModelName = modelName
	s.moaPrimaryOpts = foundOpts
	log.Printf("InferenceService: MOA primary model default set to '%s'. Reconfiguring MOA...", modelName)

	// Reconfigure MOA
	if err := s.reconfigureMOAInternal(); err != nil {
		log.Printf("[ERROR] Failed to reconfigure MOA after setting primary model: %v", err)
		return fmt.Errorf("failed to reconfigure MOA: %w", err)
	}

	log.Println("InferenceService: MOA reconfigured successfully.")
	return nil
}

// SetMOAFallbackModel sets the default fallback model used for MOA configuration (including aggregator).
// This does NOT change the primary execution/fallback list.
func (s *InferenceService) SetMOAFallbackModel(modelName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return errors.New("service is not running")
	}

	// Find the config options for the requested model from the loaded fallback attempts
	var foundOpts []config.ConfigOption
	for _, attempt := range s.fallbackAttempts {
		if attempt.Config.ModelName == modelName {
			foundOpts = attempt.Opts // Use stored opts
			break
		}
	}

	if foundOpts == nil {
		return fmt.Errorf("model '%s' not found in the configured fallback attempts", modelName)
	}

	s.moaFallbackModelName = modelName
	s.moaFallbackOpts = foundOpts
	log.Printf("InferenceService: MOA fallback model default set to '%s'. Reconfiguring MOA...", modelName)

	// Reconfigure MOA
	if err := s.reconfigureMOAInternal(); err != nil {
		log.Printf("[ERROR] Failed to reconfigure MOA after setting fallback model: %v", err)
		return fmt.Errorf("failed to reconfigure MOA: %w", err)
	}

	log.Println("InferenceService: MOA reconfigured successfully.")
	return nil
}

// GetPrimaryModels returns the names of the configured primary models.
func (s *InferenceService) GetPrimaryModels() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	models := make([]string, 0, len(s.primaryAttempts))
	for _, attempt := range s.primaryAttempts {
		models = append(models, attempt.Config.ModelName)
	}
	return models
}

// GetFallbackModels returns the names of the configured fallback models.
func (s *InferenceService) GetFallbackModels() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	models := make([]string, 0, len(s.fallbackAttempts))
	for _, attempt := range s.fallbackAttempts {
		models = append(models, attempt.Config.ModelName)
	}
	return models
}

// GetProxyModel returns the name of the proxy model.
// Returns the currently selected default model for MOA's primary role.
func (s *InferenceService) GetProxyModel() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.moaPrimaryModelName
}

// GetBaseModel returns the name of the base model.
// Returns the currently selected default model for MOA's fallback/aggregator role.
func (s *InferenceService) GetBaseModel() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.moaFallbackModelName
}

// IsRunning checks the client status
func (s *InferenceService) IsRunning() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.isRunning
}

// SetInferenceSwitcher attaches a switcher that routes text generation
// to the internal transformer (HEART/Gorgonite) when pre-training completes.
func (s *InferenceService) SetInferenceSwitcher(switcher *InferenceSwitcher) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.switcher = switcher
	log.Println("InferenceService: InferenceSwitcher attached")
}

// GetName identifies the service structure
func (s *InferenceService) GetName() string {
	return "InferenceService(Delegator+MOA)" // Updated name
}

// ClearConversationHistory clears the memory in the delegator.
func (s *InferenceService) ClearConversationHistory() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning || s.delegator == nil {
		return errors.New("inference service is not running or delegator not configured")
	}

	s.delegator.memory.Clear()
	return nil
}

// reconfigureMOAInternal handles the creation or recreation of the MOA instance.
// Assumes lock is already held.
func (s *InferenceService) reconfigureMOAInternal() error {
	log.Println("InferenceService: Reconfiguring MOA...")

	// Check if required opts are set
	if s.moaPrimaryOpts == nil || s.moaFallbackOpts == nil {
		s.moa = nil // Ensure MOA is nil if config is incomplete
		return fmt.Errorf("cannot configure MOA, primary or fallback options missing")
	}

	// --- END DEBUG ---
	// --- Create the MOA Service ---
	moaCfg := gollm.MOAConfig{
		Iterations: 2, // Or make configurable
		Models: []config.ConfigOption{
			// Use the currently selected MOA primary options
			func(cfg *config.Config) {
				for _, opt := range s.moaPrimaryOpts {
					opt(cfg)
				}
			},
			// Use the currently selected MOA fallback options
			func(cfg *config.Config) {
				for _, opt := range s.moaFallbackOpts {
					opt(cfg)
				}
			},
		},
		MaxParallel:  2,                // Or make configurable
		AgentTimeout: 60 * time.Second, // Or make configurable
	}
	// Aggregator uses the options of the currently selected MOA fallback model
	aggregatorOpts := s.moaFallbackOpts
	moaInstance, moaErr := gollm.NewMOA(moaCfg, aggregatorOpts...)
	if moaErr != nil {
		log.Printf("[ERROR] InferenceService: Failed to create/recreate MOA instance: %v", moaErr)
		s.moa = nil // Ensure it's nil on error
		return moaErr
	}

	s.moa = moaInstance // Store the new MOA instance
	log.Printf("InferenceService: MOA instance created/recreated successfully (Primary: %s, Fallback: %s).", s.moaPrimaryModelName, s.moaFallbackModelName)

	// Update the delegator with the new MOA instance
	if s.delegator != nil {
		s.delegator.UpdateMOA(s.moa)
	}
	return nil
}

// findLLMInstance searches primary and fallback attempts for a provider name.
// NOTE: This is a simplified lookup, might need refinement if multiple models
// from the same provider exist. Returns the first match.
func (s *InferenceService) findLLMInstance(providerName string) llm.LLM {
	for _, attempt := range s.primaryAttempts {
		if attempt.Config.ProviderName == providerName {
			return attempt.Instance
		}
	}
	for _, attempt := range s.fallbackAttempts {
		if attempt.Config.ProviderName == providerName {
			return attempt.Instance
		}
	}
	return nil
}

func (s *InferenceService) lookupCache(prompt string) (string, bool) {
	queryVec := embed.Embed(prompt)
	s.responseCacheMu.Lock()
	defer s.responseCacheMu.Unlock()
	for _, entry := range s.responseCache {
		if float32(embed.CosineSimilarity(queryVec, entry.vec)) >= promptSimilarityThreshold {
			return entry.response, true
		}
	}
	return "", false
}

func (s *InferenceService) storeCache(prompt, response string) {
	vec := embed.Embed(prompt)
	s.responseCacheMu.Lock()
	defer s.responseCacheMu.Unlock()
	s.responseCache = append(s.responseCache, cachedResponse{vec: vec, response: response})
}

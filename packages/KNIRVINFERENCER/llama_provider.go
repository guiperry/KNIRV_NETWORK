package inferencer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/guiperry/gollm_cerebras/config"
	"github.com/guiperry/gollm_cerebras/providers"
	gollm_types "github.com/guiperry/gollm_cerebras/types"
	"github.com/guiperry/gollm_cerebras/utils"
	"github.com/guiperry/knirv-sdk-go/models/frame"
)

const llamaSocketHost = "knirv-llama"

// DefaultHTTPTimeout is the dream-task HTTP client timeout. It was 90s in the
// original implementation, but single-slot embedded-CPU dream passes were
// observed exceeding that against TinyLlama-1.1B (up to ~280s in production
// logs). Phase C of the llama cognitive-engine plan lifts this default to 10
// minutes and makes it configurable per-process.
const DefaultHTTPTimeout = 10 * time.Minute

var llamaSocketTransportMu sync.Mutex

// LlamaProvider implements providers.Provider for the embedded local llama.cpp service.
type LlamaProvider struct {
	apiKey       string
	model        string
	maxTokens    int
	extraHeaders map[string]string
	logger       utils.Logger
	client       *http.Client
	endpoint     string
	options      map[string]interface{}
	mutex        sync.Mutex
}

func init() {
	registry := providers.GetDefaultRegistry()
	registry.Register("llama", NewLlamaProvider)
	log.Println("Registered llama provider constructor with gollm registry")
}

func NewLlamaProvider(apiKey, model string, extraHeaders map[string]string) providers.Provider {
	socketPath := os.Getenv("KNIRV_LLAMA_SOCKET")
	if socketPath == "" {
		socketPath = "/var/lib/knirvserver/sockets/llama.sock"
	}
	configureLlamaSocketTransport(socketPath)

	timeout := DefaultHTTPTimeout
	if v := os.Getenv("KNIRV_LLAMA_HTTP_TIMEOUT"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	provider := &LlamaProvider{
		apiKey:       apiKey,
		model:        model,
		maxTokens:    1000,
		extraHeaders: make(map[string]string),
		logger:       utils.NewLogger(utils.LogLevelInfo),
		client:       &http.Client{Timeout: timeout},
		endpoint:     "http://" + llamaSocketHost,
		options:      make(map[string]interface{}),
	}
	if model != "" {
		provider.model = model
	}
	for k, v := range extraHeaders {
		provider.extraHeaders[k] = v
	}
	return provider
}

// configureLlamaSocketTransport teaches gollm's internally-created default
// HTTP client to route only the synthetic knirv-llama host over the private
// Unix socket. No TCP listener is opened for the llama chat API.
func configureLlamaSocketTransport(socketPath string) {
	llamaSocketTransportMu.Lock()
	defer llamaSocketTransportMu.Unlock()

	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		base = (&http.Transport{Proxy: http.ProxyFromEnvironment})
	}
	transport := base.Clone()
	defaultDial := transport.DialContext
	if defaultDial == nil {
		defaultDial = (&net.Dialer{}).DialContext
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		if err == nil && host == llamaSocketHost {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		}
		return defaultDial(ctx, network, address)
	}
	http.DefaultTransport = transport
}

var _ providers.Provider = (*LlamaProvider)(nil)

func (p *LlamaProvider) Name() string {
	return "llama"
}

func (p *LlamaProvider) Endpoint() string {
	return p.endpoint + "/v1/chat/completions"
}

func (p *LlamaProvider) Headers() map[string]string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if p.apiKey != "" {
		headers["Authorization"] = "Bearer " + p.apiKey
	}
	for k, v := range p.extraHeaders {
		headers[k] = v
	}
	return headers
}

// SetAPIKey swaps the bearer token used for llama-server requests.  The
// wrapper that owns llama-server generates a fresh token per launch and hands
// it to the provider via this setter so every dream-task call carries the
// matching `Authorization: Bearer …` header (Phase A defense-in-depth).
func (p *LlamaProvider) SetAPIKey(key string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.apiKey = key
}

// HTTPTimeout returns the configured HTTP client timeout.  Exposed for tests
// and for the dream-task queue (Phase C) so it can size its per-task budget.
func (p *LlamaProvider) HTTPTimeout() time.Duration {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.client.Timeout
}

func (p *LlamaProvider) PrepareRequest(prompt string, options map[string]interface{}) ([]byte, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	model := p.model
	maxTokens := p.maxTokens
	if m, ok := options["max_tokens"].(int); ok && m > 0 {
		maxTokens = m
	}
	if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	fr, _ := p.stageFrame(options, model)
	messages := []map[string]interface{}{
		{"role": "system", "content": fr.System},
		{"role": "user", "content": prompt},
	}
	req := map[string]interface{}{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxTokens,
	}
	applyFrameFields(req, fr, true, options)
	if t, ok := options["temperature"].(float64); ok {
		req["temperature"] = t
	}
	if t, ok := options["top_p"].(float64); ok {
		req["top_p"] = t
	}
	if s, ok := options["stop"].([]string); ok && len(s) > 0 {
		req["stop"] = s
	}
	if stream, ok := options["stream"].(bool); ok && stream {
		req["stream"] = true
	}
	// A skill adapter registered with the llama wrapper (see SkillAdapters). The
	// wrapper answers 404 for a name it has not loaded; it never falls back to the
	// base model, so a missing adapter surfaces as an error the caller must handle.
	if a, ok := options["adapter"].(string); ok && a != "" {
		req["adapter"] = a
	}

	return json.Marshal(req)
}

func (p *LlamaProvider) PrepareRequestWithSchema(prompt string, options map[string]interface{}, schema interface{}) ([]byte, error) {
	if options == nil {
		options = map[string]interface{}{}
	}
	// A schema is sent as a constraint llama.cpp compiles to a grammar. Without
	// one this is plain JSON mode, as before.
	options["response_format"] = responseFormat(schema)
	return p.PrepareRequest(prompt, options)
}

func responseFormat(schema interface{}) map[string]interface{} {
	if schema == nil {
		return map[string]interface{}{"type": "json_object"}
	}
	return map[string]interface{}{
		"type":        "json_schema",
		"json_schema": map[string]interface{}{"name": "output", "strict": true, "schema": schema},
	}
}

// DefaultLlamaStage is the stage of calls that reach the provider without
// naming one: dream passes and server tasks. Every llama-bound request carries
// a frame; there is no unframed path.
const DefaultLlamaStage = "inferencer.dream"

// StageHeader, set as an extra header on a provider instance, is that instance's
// default stage (the gollm provider interface has no per-call stage). The wrapper
// reads the same header, so it also reaches the frame check.
const StageHeader = "X-KNIRV-Stage"

// stageFrame builds the role frame for the named stage, or the default stage
// when none (or an unregistered one) is named.
func (p *LlamaProvider) stageFrame(options map[string]interface{}, model string) (frame.Frame, bool) {
	reg := frame.Default()
	facts, ok := options["frame_facts"].(frame.LiveFacts)
	if !ok {
		facts = frame.LiveFacts{Model: model}
	}
	stage, _ := options["stage"].(string)
	if stage == "" {
		// An instance can carry its own default (callers hold p.mutex here). The
		// orchestrator's fallback chain registers one with "inferencer.task", so its
		// llama calls are not mislabelled as dream passes.
		stage = p.extraHeaders[StageHeader]
	}
	if stage == "" {
		stage = DefaultLlamaStage
	}
	fr, err := reg.Build(stage, facts)
	if err != nil {
		log.Printf("[KNIRVLLAMA] no frame for stage %q (%v); using %s", stage, err, DefaultLlamaStage)
		fr, err = reg.Build(DefaultLlamaStage, facts)
	}
	return fr, err == nil
}

// applyFrameFields adds what the llama wrapper reads on top of the OpenAI
// body: the frame claim, structured-output and caching hints, and the
// admission priority class.
func applyFrameFields(req map[string]interface{}, fr frame.Frame, framed bool, options map[string]interface{}) {
	if framed {
		req["knirv_stage"] = fr.Stage
		req["knirv_frame_hash"] = fr.Hash
	}
	if rf, ok := options["response_format"]; ok {
		req["response_format"] = rf
	}
	if c, ok := options["cache_prompt"].(bool); ok {
		req["cache_prompt"] = c
	} else if framed {
		req["cache_prompt"] = true // the frame is a stable prefix; reuse its KV cache
	}
	if pr, ok := options["priority"].(int); ok {
		req["knirv_priority"] = pr
	}
}

func (p *LlamaProvider) PrepareRequestWithMessages(messages []gollm_types.MemoryMessage, options map[string]interface{}) ([]byte, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	model := p.model
	maxTokens := p.maxTokens
	if m, ok := options["max_tokens"].(int); ok && m > 0 {
		maxTokens = m
	}
	if m, ok := options["model"].(string); ok && m != "" {
		model = m
	}

	fr, _ := p.stageFrame(options, model)
	apiMessages := make([]map[string]interface{}, 0, len(messages)+1)
	apiMessages = append(apiMessages, map[string]interface{}{"role": "system", "content": fr.System})
	for _, msg := range messages {
		if msg.Role == "system" {
			continue // a caller's own system text must not compete with the frame
		}
		apiMessages = append(apiMessages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	req := map[string]interface{}{
		"model":      model,
		"messages":   apiMessages,
		"max_tokens": maxTokens,
	}
	applyFrameFields(req, fr, true, options)
	if t, ok := options["temperature"].(float64); ok {
		req["temperature"] = t
	}
	if t, ok := options["top_p"].(float64); ok {
		req["top_p"] = t
	}
	if stream, ok := options["stream"].(bool); ok && stream {
		req["stream"] = true
	}

	return json.Marshal(req)
}

func (p *LlamaProvider) ParseResponse(body []byte) (string, error) {
	var response struct {
		Choices []struct {
			Message struct {
				Content      string `json:"content"`
				FunctionCall struct {
					Arguments string `json:"arguments"`
				} `json:"function_call"`
				ToolCalls []struct {
					Function struct {
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Error.Message != "" {
		return "", fmt.Errorf("API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	if response.Choices[0].Message.FunctionCall.Arguments != "" {
		completion := response.Choices[0].Message.FunctionCall.Arguments
		log.Printf("[KNIRVLLAMA] completion: %s", completion)
		return completion, nil
	}

	if tc := response.Choices[0].Message.ToolCalls; len(tc) > 0 && tc[0].Function.Arguments != "" && response.Choices[0].Message.Content == "" {
		return tc[0].Function.Arguments, nil
	}

	completion := response.Choices[0].Message.Content
	log.Printf("[KNIRVLLAMA] completion: %s", completion)
	return completion, nil
}

func (p *LlamaProvider) SetExtraHeaders(extraHeaders map[string]string) {
	if extraHeaders == nil {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for k, v := range extraHeaders {
		p.extraHeaders[k] = v
	}
}

func (p *LlamaProvider) HandleFunctionCalls(body []byte) ([]byte, error) {
	return body, nil
}

func (p *LlamaProvider) SupportsJSONSchema() bool {
	return true
}

func (p *LlamaProvider) SetDefaultOptions(cfg *config.Config) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.options["temperature"] = cfg.Temperature
	p.options["max_tokens"] = cfg.MaxTokens
	if cfg.MaxTokens > 0 {
		p.maxTokens = cfg.MaxTokens
	}
	if cfg.Seed != nil {
		p.options["seed"] = *cfg.Seed
	}
}

func (p *LlamaProvider) SetOption(key string, value interface{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.options[key] = value
}

func (p *LlamaProvider) SetLogger(logger utils.Logger) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.logger = logger
}

func (p *LlamaProvider) SupportsStreaming() bool {
	return true
}

func (p *LlamaProvider) PrepareStreamRequest(prompt string, options map[string]interface{}) ([]byte, error) {
	streamOptions := make(map[string]interface{})
	for k, v := range options {
		streamOptions[k] = v
	}
	streamOptions["stream"] = true
	return p.PrepareRequest(prompt, streamOptions)
}

func (p *LlamaProvider) ParseStreamResponse(chunk []byte) (string, error) {
	trimmed := strings.TrimSpace(string(chunk))
	if trimmed == "[DONE]" {
		return "", io.EOF
	}

	var streamChunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(chunk, &streamChunk); err != nil {
		return "", err
	}

	if len(streamChunk.Choices) == 0 {
		return "", nil
	}

	return streamChunk.Choices[0].Delta.Content, nil
}

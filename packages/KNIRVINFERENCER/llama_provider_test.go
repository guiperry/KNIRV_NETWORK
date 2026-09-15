package inferencer

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLlamaProviderInjectsDreamSystemPrompt(t *testing.T) {
	provider := NewLlamaProvider("", "local-llama", nil).(*LlamaProvider)
	if provider.Endpoint() != "http://knirv-llama/v1/chat/completions" {
		t.Fatalf("Endpoint() = %q, want synthetic Unix-socket host", provider.Endpoint())
	}
	if strings.Contains(provider.Endpoint(), "127.0.0.1") {
		t.Fatalf("Endpoint() must not use TCP: %q", provider.Endpoint())
	}
	body, err := provider.PrepareRequest("summarize this telemetry", nil)
	if err != nil {
		t.Fatalf("PrepareRequest() error = %v", err)
	}

	var request struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if len(request.Messages) < 2 {
		t.Fatalf("messages = %#v, want system and user messages", request.Messages)
	}
	if request.Messages[0].Role != "system" || request.Messages[0].Content != DreamSystemPrompt {
		t.Fatalf("first message = %#v, want DreamSystemPrompt system message", request.Messages[0])
	}
	if request.Messages[1].Role != "user" {
		t.Fatalf("second message role = %q, want user", request.Messages[1].Role)
	}
}

func TestRegisterDreamProviderDoesNotChangeInteractiveRouting(t *testing.T) {
	service, err := NewInferenceService(nil)
	if err != nil {
		t.Fatalf("NewInferenceService() error = %v", err)
	}
	if err := service.RegisterDreamProvider("llama", "local-llama", "/var/lib/knirvserver/sockets/llama.sock"); err != nil {
		t.Fatalf("RegisterDreamProvider() error = %v", err)
	}
	if len(service.dreamAttempts) != 1 {
		t.Fatalf("dream attempts = %d, want 1", len(service.dreamAttempts))
	}
	if len(service.primaryAttempts) != 0 || len(service.fallbackAttempts) != 0 {
		t.Fatalf("interactive attempts changed: primary=%d fallback=%d", len(service.primaryAttempts), len(service.fallbackAttempts))
	}
}

func TestLlamaProviderDefaultTimeoutLiftedAboveWorstCase(t *testing.T) {
	provider := NewLlamaProvider("", "local-llama", nil).(*LlamaProvider)
	if provider.HTTPTimeout() < 5*time.Minute {
		t.Fatalf("HTTPTimeout() = %v, want at least 5m (Phase C: observed 280s completion times in production)", provider.HTTPTimeout())
	}
	if provider.HTTPTimeout() != DefaultHTTPTimeout {
		t.Fatalf("HTTPTimeout() = %v, want default %v", provider.HTTPTimeout(), DefaultHTTPTimeout)
	}
}

func TestLlamaProviderHonoursTimeoutEnvOverride(t *testing.T) {
	t.Setenv("KNIRV_LLAMA_HTTP_TIMEOUT", "3m")
	provider := NewLlamaProvider("", "local-llama", nil).(*LlamaProvider)
	if provider.HTTPTimeout() != 3*time.Minute {
		t.Fatalf("HTTPTimeout() = %v, want 3m from env override", provider.HTTPTimeout())
	}
}

func TestLlamaProviderSetsAuthorizationHeaderFromAPIKey(t *testing.T) {
	provider := NewLlamaProvider("super-secret", "local-llama", nil).(*LlamaProvider)
	headers := provider.Headers()
	if got := headers["Authorization"]; got != "Bearer super-secret" {
		t.Fatalf("Authorization = %q, want %q", got, "Bearer super-secret")
	}
}

func TestLlamaProviderOmitsAuthorizationHeaderWithoutAPIKey(t *testing.T) {
	provider := NewLlamaProvider("", "local-llama", nil).(*LlamaProvider)
	headers := provider.Headers()
	if _, ok := headers["Authorization"]; ok {
		t.Fatalf("Authorization header must not be sent when apiKey is empty: %#v", headers)
	}
}

func TestLlamaProviderSetAPIKeyUpdatesHeaders(t *testing.T) {
	provider := NewLlamaProvider("", "local-llama", nil).(*LlamaProvider)
	if _, ok := provider.Headers()["Authorization"]; ok {
		t.Fatal("Authorization header should be absent before SetAPIKey")
	}
	provider.SetAPIKey("rotated-token")
	if got := provider.Headers()["Authorization"]; got != "Bearer rotated-token" {
		t.Fatalf("Authorization = %q, want %q", got, "Bearer rotated-token")
	}
}

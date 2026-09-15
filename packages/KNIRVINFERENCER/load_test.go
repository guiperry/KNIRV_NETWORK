package inferencer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guiperry/gollm_cerebras/llm"
	"github.com/guiperry/gollm_cerebras/utils"
)

// uniquePromptCounter fuels distinct, high-entropy prompt suffixes so the
// semantic response cache (CosineSimilarity >= 0.97) never collapses distinct
// requests into cache hits. Without this the load test would measure the cache,
// not the generation path.
var uniquePromptCounter atomic.Int64

// uniquePrompt builds a prompt whose embedding is far from every other prompt
// in the run, forcing the delegator path — not the response cache — to serve
// it. It needs to be distinct *across runs too*: a math/rand uint32 (seeded
// globally) plus a shared counter desyncs under concurrency and occasionally
// produces two prompts whose cosine similarity exceeds the 0.97 semantic-cache
// threshold, turning a generation assertion into a flaky cache hit. crypto/rand
// gives true per-prompt entropy with no shared global state, and a long hex
// payload keeps every pair well below the threshold (measured max ~0.65).
func uniquePrompt(prefix string, wID, iter int) string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(fmt.Sprintf("uniquePrompt: crypto/rand failed: %v", err))
	}
	return fmt.Sprintf("%s worker-%d iter-%d telemetry-id-%d-%s", prefix, wID, iter, uniquePromptCounter.Add(1), hex.EncodeToString(raw[:]))
}

// stubLLM is an in-memory llm.LLM implementation used to load-test the
// InferenceService without touching the network. It returns a deterministic
// canned response, optionally after a simulated latency.
type stubLLM struct {
	response string
	latency  time.Duration
	calls    atomic.Int64
}

func (s *stubLLM) Generate(ctx context.Context, prompt *llm.Prompt, opts ...llm.GenerateOption) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if s.latency > 0 {
		select {
		case <-time.After(s.latency):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	s.calls.Add(1)
	return s.response, nil
}

func (s *stubLLM) GenerateWithSchema(ctx context.Context, prompt *llm.Prompt, schema interface{}, opts ...llm.GenerateOption) (string, error) {
	return s.Generate(ctx, prompt, opts...)
}

func (s *stubLLM) Stream(ctx context.Context, prompt *llm.Prompt, opts ...llm.StreamOption) (llm.TokenStream, error) {
	return &stubTokenStream{token: &llm.StreamToken{Text: s.response}},
		nil
}

func (s *stubLLM) SupportsStreaming() bool { return false }

func (s *stubLLM) SetOption(key string, value interface{}) {}

func (s *stubLLM) SetLogLevel(level utils.LogLevel) {}

func (s *stubLLM) SetEndpoint(endpoint string) {}

func (s *stubLLM) NewPrompt(input string) *llm.Prompt { return llm.NewPrompt(input) }

func (s *stubLLM) GetLogger() utils.Logger { return stubLogger{} }

func (s *stubLLM) SupportsJSONSchema() bool { return true }

// stubTokenStream trivially satisfies llm.TokenStream.
type stubTokenStream struct {
	token *llm.StreamToken
	sent  bool
}

func (s *stubTokenStream) Next(ctx context.Context) (*llm.StreamToken, error) {
	if s.sent {
		return nil, io.EOF
	}
	s.sent = true
	return s.token, nil
}

func (s *stubTokenStream) Close() error { return nil }

// stubLogger trivially satisfies utils.Logger.
type stubLogger struct{}

func (stubLogger) Debug(msg string, kv ...interface{})  {}
func (stubLogger) Info(msg string, kv ...interface{})   {}
func (stubLogger) Warn(msg string, kv ...interface{})   {}
func (stubLogger) Error(msg string, kv ...interface{})  {}
func (stubLogger) SetLevel(level utils.LogLevel)        {}

// newLoadTestService builds an InferenceService routed entirely through
// deterministic stub LLMs: a primary set that always succeeds and a fallback
// set that would carry failed primary traffic.
func newLoadTestService(primaryLatency, fallbackLatency time.Duration) (*InferenceService, *stubLLM, *stubLLM) {
	primary := &stubLLM{response: "primary-stub-response", latency: primaryLatency}
	fallback := &stubLLM{response: "fallback-stub-response", latency: fallbackLatency}

	svc, _ := NewInferenceService(&MockDatabaseAccessor{})
	svc.primaryAttempts = make([]LLMAttempt, 0)
	svc.fallbackAttempts = make([]LLMAttempt, 0)
	svc.primaryAttempts = append(svc.primaryAttempts, LLMAttempt{
		Instance: primary,
		Config: LLMAttemptConfig{
			ProviderName: "stub",
			ModelName:    "local-llama",
			MaxTokens:    256,
			IsPrimary:    true,
		},
	})
	svc.fallbackAttempts = append(svc.fallbackAttempts, LLMAttempt{
		Instance: fallback,
		Config: LLMAttemptConfig{
			ProviderName: "stub",
			ModelName:    "stub-fallback",
			MaxTokens:    256,
			IsPrimary:    false,
		},
	})

	svc.delegator = NewDelegatorService(
		svc.primaryAttempts,
		svc.fallbackAttempts,
		4096,
		"local-llama",
		nil,
		svc.contextStrategist,
	)
	svc.isRunning = true
	return svc, primary, fallback
}

// TestLoadConcurrentGenerationsLoad is the core load test: a fixed pool of
// workers hammers GenerateTextWithContext concurrently and every response must
// be the primary stub's canned text. Running under -race this is a data race
// net for the InferenceService/DelegatorService/response-cache stack. It also
// asserts the primary attempts saturate: because the primary never errors, the
// fallback attempt must never be reached.
//
// Request count is deliberately bounded (16 workers x 6 iterations): every
// request re-walks the shared sliding window via GetMessagesForContext and
// re-estimates ~273 window messages at ~1ms/encode, so O(requests x window)
// time. 96 requests keeps the test under ~30s at -race while still hammering
// with 16-way concurrency.
func TestLoadConcurrentGenerationsLoad(t *testing.T) {
	const (
		workers    = 16
		iterations = 6
		expected   = workers * iterations
	)

	svc, primary, fallback := newLoadTestService(0, 0)
	defer svc.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, expected)
	start := time.Now()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				prompt := uniquePrompt("telemetry payload", workerID, i)
				got, err := svc.GenerateTextWithContext(ctx, "local-llama", prompt, "Extract operational status")
				if err != nil {
					errCh <- fmt.Errorf("request failed: %w", err)
					continue
				}
				if got != "primary-stub-response" {
					errCh <- fmt.Errorf("unexpected response %q", got)
				}
			}
		}(w)
	}

	wg.Wait()
	close(errCh)
	elapsed := time.Since(start)

	for err := range errCh {
		t.Error(err)
	}

	if n := primary.calls.Load(); n != expected {
		t.Fatalf("primary attempts = %d calls, want %d", n, expected)
	}
	if n := fallback.calls.Load(); n != 0 {
		t.Fatalf("fallback attempts = %d calls, want 0 (primary must never fail under load)", n)
	}
	if len(svc.responseCache) > expected {
		t.Fatalf("response cache = %d entries, want <= %d", len(svc.responseCache), expected)
	}

	t.Logf("load: %d total requests, %v elapsed, %.1f req/s", expected, elapsed.Round(time.Millisecond), float64(expected)/elapsed.Seconds())
}

// TestLoadConcurrentIdenticalPrompts hammers a single prompt value to force
// the semantic response-cache write and read paths to overlap. Without the
// responseCacheMu guard this would race; under -race it is a targeted check.
func TestLoadConcurrentIdenticalPrompts(t *testing.T) {
	const (
		workers    = 8
		iterations = 30
	)

	svc, _, _ := newLoadTestService(0, 0)
	defer svc.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, workers*iterations)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				got, err := svc.GenerateTextWithContext(ctx, "local-llama", "identical prompt", "instruction")
				if err != nil {
					errCh <- err
					continue
				}
				if got != "primary-stub-response" {
					errCh <- fmt.Errorf("unexpected response %q", got)
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}

// TestLoadFallbackUnderPrimaryFailure proves the fallback attempt absorbs load
// when every primary call fails (realistic load-resilience path, no network).
func TestLoadFallbackUnderPrimaryFailure(t *testing.T) {
	const (
		workers    = 4
		iterations = 12
		expected   = workers * iterations
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	failingPrimary := &stubLLM{response: "never-returned"}
	primaryFailures := atomic.Int64{}
	fallback := &stubLLM{response: "fallback-stub-response"}

	svc, _ := NewInferenceService(&MockDatabaseAccessor{})
	svc.primaryAttempts = append(svc.primaryAttempts, LLMAttempt{
		Instance: failOnDemandLLM{LLM: failingPrimary, fail: &primaryFailures},
		Config: LLMAttemptConfig{
			ProviderName: "stub-fail",
			ModelName:    "failing-primary",
			MaxTokens:    256,
			IsPrimary:    true,
		},
	})
	svc.fallbackAttempts = append(svc.fallbackAttempts, LLMAttempt{
		Instance: fallback,
		Config: LLMAttemptConfig{
			ProviderName: "stub",
			ModelName:    "stub-fallback",
			MaxTokens:    256,
			IsPrimary:    false,
		},
	})
	svc.delegator = NewDelegatorService(
		svc.primaryAttempts,
		svc.fallbackAttempts,
		4096,
		"local-llama",
		nil,
		svc.contextStrategist,
	)
	svc.isRunning = true
	defer svc.Stop()

	var wg sync.WaitGroup
	errCh := make(chan error, expected)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				// modelName "" -> no specific model, so the delegator walks
				// primary then fallback (the load-resilience path).
				got, err := svc.GenerateTextWithContext(ctx, "", uniquePrompt("retry", wID, i), "int")
				if err != nil {
					errCh <- err
					continue
				}
				if got != "fallback-stub-response" {
					errCh <- fmt.Errorf("expected fallback response, got %q", got)
				}
			}
		}(w)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}

	if n := primaryFailures.Load(); n != expected {
		t.Fatalf("primary failures = %d, want %d (every request must hit the failing primary first)", n, expected)
	}
	if n := fallback.calls.Load(); n != expected {
		t.Fatalf("fallback attempts = %d calls, want %d (every failing primary request must fall back)", n, expected)
	}
}

// failOnDemandLLM wraps a stub and reports an error from Generate on every call.
type failOnDemandLLM struct {
	llm.LLM
	fail *atomic.Int64
}

func (f failOnDemandLLM) Generate(ctx context.Context, prompt *llm.Prompt, opts ...llm.GenerateOption) (string, error) {
	f.fail.Add(1)
	return "", errors.New("stub primary failure")
}

// BenchmarkInferenceServiceThroughput measures requests/second through the
// full InferenceService -> DelegatorService path using a zero-latency stub.
func BenchmarkInferenceServiceThroughput(b *testing.B) {
	svc, _, _ := newLoadTestService(0, 0)
	defer svc.Stop()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.GenerateTextWithContext(ctx, "local-llama", fmt.Sprintf("bench-prompt-%d", i), "int"); err != nil {
			b.Fatal(err)
		}
	}
}
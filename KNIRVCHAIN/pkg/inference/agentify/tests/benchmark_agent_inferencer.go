// benchmark_agent_inferencer.go
package agentify_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"KNIRVCHAIN/pkg/inference/agentify"
)

func BenchmarkAgentInferencer(b *testing.B) {
	// Create a new Agent Inferencer
	inferencer := agentify.NewAgentInferencer("./plugins")

	// Create a session ID
	sessionID := "benchmark-session"

	// Activate the agent
	ctx := context.Background()
	if err := inferencer.ActivateAgent(ctx, "example", "1.0", sessionID, nil); err != nil {
		b.Fatalf("Failed to activate agent: %v", err)
	}
	defer inferencer.DeactivateAgent(ctx, sessionID)

	// Benchmark processing an inference request
	b.Run("ProcessInference", func(b *testing.B) {
		// Reset the timer
		b.ResetTimer()

		// Run the benchmark
		for i := 0; i < b.N; i++ {
			// Create an inference request
			request := &agentify.InferenceRequest{
				Input:     "Hello, world!",
				SessionID: sessionID,
			}

			// Process the inference request
			_, err := inferencer.ProcessInference(ctx, sessionID, request)
			if err != nil {
				b.Fatalf("Failed to process inference: %v", err)
			}
		}
	})

	// Benchmark setting and getting memory
	b.Run("Memory", func(b *testing.B) {
		// Reset the timer
		b.ResetTimer()

		// Run the benchmark
		for i := 0; i < b.N; i++ {
			// Set a memory value
			key := "benchmark-key"
			value := "benchmark-value"
			if err := inferencer.SetAgentMemory(ctx, sessionID, key, value); err != nil {
				b.Fatalf("Failed to set memory: %v", err)
			}

			// Get the memory value
			_, err := inferencer.GetAgentMemory(ctx, sessionID, key)
			if err != nil {
				b.Fatalf("Failed to get memory: %v", err)
			}
		}
	})
}

func BenchmarkAgentInferencerParallel(b *testing.B) {
	// Create a new Agent Inferencer
	inferencer := agentify.NewAgentInferencer("./plugins")

	// Benchmark processing inference requests in parallel
	b.Run("ProcessInferenceParallel", func(b *testing.B) {
		// Reset the timer
		b.ResetTimer()

		// Run the benchmark in parallel
		b.RunParallel(func(pb *testing.PB) {
			// Create a session ID for this goroutine
			sessionID := "benchmark-session-" + time.Now().String()

			// Activate the agent
			ctx := context.Background()
			if err := inferencer.ActivateAgent(ctx, "example", "1.0", sessionID, nil); err != nil {
				b.Fatalf("Failed to activate agent: %v", err)
			}
			defer inferencer.DeactivateAgent(ctx, sessionID)

			// Run the benchmark
			for pb.Next() {
				// Create an inference request
				request := &agentify.InferenceRequest{
					Input:     "Hello, world!",
					SessionID: sessionID,
				}

				// Process the inference request
				_, err := inferencer.ProcessInference(ctx, sessionID, request)
				if err != nil {
					b.Fatalf("Failed to process inference: %v", err)
				}
			}
		})
	})
}

func BenchmarkAgentHTTPAPI(b *testing.B) {
	// Create a new Agent Inferencer
	inferencer := agentify.NewAgentInferencer("./plugins")

	// Create a new HTTP API
	api := agentify.NewAgentHTTPAPI(inferencer)

	// Create a test server
	mux := http.NewServeMux()
	api.RegisterHandlers(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	// Create a session ID
	sessionID := "benchmark-session"

	// Activate the agent
	activateBody := map[string]interface{}{
		"agentId":   "example",
		"version":   "1.0",
		"sessionId": sessionID,
	}
	activateBodyBytes, err := json.Marshal(activateBody)
	if err != nil {
		b.Fatalf("Failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("POST", server.URL+"/v1/agents/activate", bytes.NewBuffer(activateBodyBytes))
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		b.Fatalf("Failed to send request: %v", err)
	}
	resp.Body.Close()

	// Benchmark processing an inference request
	b.Run("ProcessInference", func(b *testing.B) {
		// Create the request body
		body := map[string]interface{}{
			"sessionId": sessionID,
			"input":     "Hello, world!",
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			b.Fatalf("Failed to marshal request body: %v", err)
		}

		// Reset the timer
		b.ResetTimer()

		// Run the benchmark
		for i := 0; i < b.N; i++ {
			// Create a request
			req, err := http.NewRequest("POST", server.URL+"/v1/inference", bytes.NewBuffer(bodyBytes))
			if err != nil {
				b.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Authorization", "Bearer test-api-key")
			req.Header.Set("Content-Type", "application/json")

			// Send the request
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				b.Fatalf("Failed to send request: %v", err)
			}
			resp.Body.Close()
		}
	})

	// Deactivate the agent
	deactivateBody := map[string]interface{}{
		"sessionId": sessionID,
	}
	deactivateBodyBytes, err := json.Marshal(deactivateBody)
	if err != nil {
		b.Fatalf("Failed to marshal request body: %v", err)
	}

	req, err = http.NewRequest("POST", server.URL+"/v1/agents/deactivate", bytes.NewBuffer(deactivateBodyBytes))
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		b.Fatalf("Failed to send request: %v", err)
	}
	resp.Body.Close()
}

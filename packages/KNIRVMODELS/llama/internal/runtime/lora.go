package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LoRA adapter control.
//
// llama-server loads adapters at startup via repeated --lora/--lora-scaled, and
// adjusts them at runtime:
//
//	GET  /lora-adapters   -> [{"id":0,"path":"...","scale":1.0}]
//	POST /lora-adapters   <- [{"id":0,"scale":0.2}]   sets the server-wide scale
//
// A request body may also carry a `lora` field, which overrides the server-wide
// scale for that request alone. That is what makes per-request adapter switching
// possible: load every adapter once (optionally inert, via
// --lora-init-without-apply) and select per call.
//
// IMPORTANT (verified against the provisioned build, llama.cpp 9e0e220):
// POST /lora-adapters can only change the scale of adapters that were loaded at
// startup. It does NOT load a new adapter file: a body naming a "path" returns
// {"success":true} and changes nothing. Adding an adapter therefore needs a
// llama-server restart with the new adapter set (see the adapters package).

// LoRAAdapterInfo describes one adapter the server has loaded.
type LoRAAdapterInfo struct {
	ID    int     `json:"id"`
	Path  string  `json:"path"`
	Scale float64 `json:"scale"`
}

// LoRASelection is an adapter scale override.
//
// It is used both as the body of POST /lora-adapters and as the per-request
// `lora` field, which is why the JSON shape is a plain array of these.
type LoRASelection struct {
	ID    int     `json:"id"`
	Scale float64 `json:"scale"`
}

// SetLoRAAdapters sets the server-wide adapter scales.
//
// A scale of 0 disables an adapter. Note this changes the default for subsequent
// requests only where they do not carry their own `lora` field.
func SetLoRAAdapters(ctx context.Context, address string, apiKey string, selections []LoRASelection) error {
	if len(selections) == 0 {
		// llama-server requires a JSON array; an empty one is accepted and
		// leaves the adapter set untouched.
		return fmt.Errorf("no adapter selections supplied")
	}
	body, err := json.Marshal(selections)
	if err != nil {
		return fmt.Errorf("marshal adapter selections: %w", err)
	}
	return doLoRARequest(ctx, address, apiKey, http.MethodPost, body, nil)
}

// ZeroLoRAAdapters sets every loaded adapter's server-wide scale to 0, so a request
// that carries no `lora` field runs the plain base model. Verified against the
// provisioned llama-server: --lora-init-without-apply alone leaves requests
// without a `lora` field running the adapters at scale 1. It returns the number of
// adapters zeroed (0 when none are loaded, which is not an error).
func ZeroLoRAAdapters(ctx context.Context, address string, apiKey string) (int, error) {
	adapters, err := GetLoRAAdapters(ctx, address, apiKey)
	if err != nil {
		return 0, err
	}
	if len(adapters) == 0 {
		return 0, nil
	}
	zero := make([]LoRASelection, len(adapters))
	for i, a := range adapters {
		zero[i] = LoRASelection{ID: a.ID, Scale: 0}
	}
	return len(zero), SetLoRAAdapters(ctx, address, apiKey, zero)
}

// GetLoRAAdapters lists the adapters the server has loaded.
func GetLoRAAdapters(ctx context.Context, address string, apiKey string) ([]LoRAAdapterInfo, error) {
	var adapters []LoRAAdapterInfo
	if err := doLoRARequest(ctx, address, apiKey, http.MethodGet, nil, &adapters); err != nil {
		return nil, err
	}
	return adapters, nil
}

// doLoRARequest performs the HTTP exchange and decodes a response when out is
// non-nil.
func doLoRARequest(ctx context.Context, address, apiKey, method string, body []byte, out any) error {
	address = strings.TrimSpace(address)
	if address == "" {
		return fmt.Errorf("llama address is required")
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://"+address+"/lora-adapters", reader)
	if err != nil {
		return fmt.Errorf("build adapter request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key := strings.TrimSpace(apiKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("adapter request to %s: %w", address, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		// A server too old to know /lora-adapters lands here, which is worth
		// distinguishing from a transient failure.
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("llama-server at %s does not expose /lora-adapters (HTTP 404): runtime adapter switching is unavailable: %s",
				address, strings.TrimSpace(string(raw)))
		}
		return fmt.Errorf("adapter request to %s failed (HTTP %d): %s", address, resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode adapter response: %w", err)
	}
	return nil
}

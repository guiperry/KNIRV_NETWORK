// Package teacher is the client for batch work done by the strong model (Colibri):
// synthetic training data, suggested labels and retrieval judgments.
//
// Rules this package enforces, from the inference plan:
//   - Teacher requests carry the teacher role, so the harness answers them only from
//     Colibri. A response that names another backend is rejected: what the caller
//     records about where its data came from has to be true.
//   - Every result carries provenance (backend, model id, catalog model, prompt hash),
//     and results are cached by their inputs and model, so a rerun is a no-op and an
//     interrupted job resumes.
//   - Output is data for a human or a trainer to look at. Nothing here writes ground
//     truth: the teacher never grades.
package teacher

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Header names shared with the harness router.
const (
	roleHeader    = "X-Knirv-Role"
	backendHeader = "X-Knirv-Backend"
	modelHeader   = "X-Knirv-Model"
)

// Version is part of every cache key and provenance record; change it when prompts
// built by this package change meaning.
const Version = "teacher/v1"

// ErrUnavailable means no teacher backend answered within the client's patience.
var ErrUnavailable = errors.New("teacher backend unavailable")

// Provenance says where a piece of data came from.
type Provenance struct {
	Teacher    string  `json:"teacher"`           // always the harness backend name
	ModelID    string  `json:"model_id"`          // the id the engine reported
	Catalog    string  `json:"catalog,omitempty"` // Colibri's catalog model (e.g. qwen36-35b)
	PromptHash string  `json:"prompt_hash"`       // sha256 of system+user+parameters
	Version    string  `json:"version"`           // this package's Version
	Samples    int     `json:"samples,omitempty"` // votes behind a distribution
	Temp       float64 `json:"temperature"`       // sampling temperature
}

// Request is one chat call.
type Request struct {
	System      string
	User        string
	MaxTokens   int
	Temperature float64
}

// Reply is the model's answer and where it came from.
type Reply struct {
	Text             string
	CompletionTokens int
	Prov             Provenance
}

// Client talks to the inference endpoint (the harness router).
type Client struct {
	HTTP *http.Client
	Base string
	// Patience is how long a call waits through "no teacher backend yet" (503) answers
	// before giving up (default 30m: Colibri may still be loading its model).
	Patience time.Duration
	Logf     func(format string, args ...any)

	catalog   string
	retryWait time.Duration // wait between attempts while unavailable (default 15s)
}

// NewUnix builds a client for the inference socket (llama.sock). perRequest bounds a
// single generation; teacher jobs are slow, so it defaults to 15 minutes.
func NewUnix(socket string, perRequest time.Duration) *Client {
	if perRequest <= 0 {
		perRequest = 15 * time.Minute
	}
	return &Client{
		Base: "http://teacher",
		HTTP: &http.Client{
			Timeout: perRequest,
			Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socket)
			}, DisableKeepAlives: true},
		},
	}
}

func (c *Client) logf(f string, a ...any) {
	if c.Logf != nil {
		c.Logf(f, a...)
	}
}

// Status is the harness status document, used to learn the catalog model and whether
// Colibri is qualified.
func (c *Client) Status(ctx context.Context) (map[string]any, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+"/harness/status", nil)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var doc map[string]any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&doc); err != nil {
		return nil, err
	}
	return doc, nil
}

var thinkRE = regexp.MustCompile(`(?s)<think>.*?</think>`)

// Chat sends one teacher-role request and returns the answer with its provenance.
func (c *Client) Chat(ctx context.Context, r Request) (Reply, error) {
	if c.catalog == "" {
		if st, err := c.Status(ctx); err == nil {
			if m, ok := st["colibri_model"].(string); ok {
				c.catalog = m
			}
		}
	}
	patience := c.Patience
	if patience <= 0 {
		patience = 30 * time.Minute
	}
	deadline := time.Now().Add(patience)
	body, _ := json.Marshal(map[string]any{
		"model":       "teacher", // the router substitutes the engine's own id
		"messages":    messages(r),
		"max_tokens":  r.MaxTokens,
		"temperature": r.Temperature,
	})
	for {
		rep, retry, err := c.once(ctx, body, r)
		if err == nil {
			return rep, nil
		}
		if !retry || time.Now().After(deadline) || ctx.Err() != nil {
			if retry {
				return Reply{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
			}
			return Reply{}, err
		}
		c.logf("teacher not ready (%v); retrying", err)
		wait := c.retryWait
		if wait <= 0 {
			wait = 15 * time.Second
		}
		select {
		case <-ctx.Done():
			return Reply{}, ctx.Err()
		case <-time.After(wait):
		}
	}
}

func messages(r Request) []map[string]string {
	var m []map[string]string
	if r.System != "" {
		m = append(m, map[string]string{"role": "system", "content": r.System})
	}
	return append(m, map[string]string{"role": "user", "content": r.User})
}

// once makes a single attempt. retry is true for answers that mean "try again later".
func (c *Client) once(ctx context.Context, body []byte, r Request) (Reply, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Reply{}, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(roleHeader, "teacher")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Reply{}, ctx.Err() == nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	switch {
	case resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests:
		return Reply{}, true, fmt.Errorf("HTTP %d: %s", resp.StatusCode, firstLine(string(raw)))
	case resp.StatusCode != http.StatusOK:
		return Reply{}, false, fmt.Errorf("HTTP %d: %s", resp.StatusCode, firstLine(string(raw)))
	}
	backend := resp.Header.Get(backendHeader)
	if backend != "colibri" {
		return Reply{}, false, fmt.Errorf("the answer came from %q, not from the teacher backend; refusing to record it", backend)
	}
	var doc struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			Completion int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil || len(doc.Choices) == 0 {
		return Reply{}, false, fmt.Errorf("unusable response: %s", firstLine(string(raw)))
	}
	return Reply{
		Text:             strings.TrimSpace(thinkRE.ReplaceAllString(doc.Choices[0].Message.Content, "")),
		CompletionTokens: doc.Usage.Completion,
		Prov: Provenance{
			Teacher: backend, ModelID: resp.Header.Get(modelHeader), Catalog: c.catalog,
			PromptHash: PromptHash(r), Version: Version, Temp: r.Temperature,
		},
	}, false, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 300 {
		s = s[:300]
	}
	return s
}

// PromptHash identifies a request's content and parameters.
func PromptHash(r Request) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%d\x00%g", Version, r.System, r.User, r.MaxTokens, r.Temperature)
	return hex.EncodeToString(h.Sum(nil))
}

// sampleKey identifies one sample of a request: the cache key.
func sampleKey(model string, r Request, sample int) string {
	h := sha256.Sum256([]byte(model + "\x00" + PromptHash(r) + "\x00" + strconv.Itoa(sample)))
	return hex.EncodeToString(h[:])
}

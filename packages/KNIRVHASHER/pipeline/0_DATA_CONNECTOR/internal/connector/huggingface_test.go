package connector

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestHuggingFaceRetriesRateLimit(t *testing.T) {
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		if calls.Add(1) <= 2 {
			return testResponse(http.StatusTooManyRequests, "limited", "0"), nil
		}
		return testResponse(http.StatusOK, `{"rows":[{"row":{"text":"a usable training record"}}]}`, ""), nil
	})}

	connector := HuggingFaceConnector{
		BaseURL:    "http://huggingface.test/rows",
		Client:     client,
		MaxRetries: 2,
		Sleep:      func(time.Duration) {},
	}
	records, exhausted, err := connector.Page("example/data", "train", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if exhausted || len(records) != 1 {
		t.Fatalf("records=%d exhausted=%v", len(records), exhausted)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("calls=%d, want 3", got)
	}
}

func TestHuggingFaceRateLimitFallsBack(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return testResponse(http.StatusTooManyRequests, "limited", "0"), nil
	})}

	connector := HuggingFaceConnector{
		BaseURL:    "http://huggingface.test/rows",
		Client:     client,
		MaxRetries: 1,
		Sleep:      func(time.Duration) {},
	}
	records, exhausted, err := connector.Page("example/data", "train", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !exhausted || len(records) != 0 {
		t.Fatalf("records=%d exhausted=%v", len(records), exhausted)
	}
}

func testResponse(status int, body, retryAfter string) *http.Response {
	headers := make(http.Header)
	if retryAfter != "" {
		headers.Set("Retry-After", retryAfter)
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

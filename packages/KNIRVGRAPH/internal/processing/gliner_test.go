package processing

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type glinerRoundTrip func(*http.Request) (*http.Response, error)

func (f glinerRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestGLiNERClient(t *testing.T) {
	c := NewGLiNERClient("http://gliner", "test", 1)
	c.client = &http.Client{Transport: glinerRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"entities":[{"text":"KNIRV","label":"org","score":0.98,"start":0,"end":5}]}`)), Header: make(http.Header)}, nil
	})}
	entities, err := c.Extract(context.Background(), "doc", "KNIRV", []string{"ORG"}, .5)
	if err != nil || len(entities) != 1 || entities[0].Type != "ORG" {
		t.Fatalf("entities=%v err=%v", entities, err)
	}
}

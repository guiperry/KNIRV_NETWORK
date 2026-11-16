package launcher

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The MiniCPM server may start only once the text harness has made its decision.
// llama.cpp is the primary, Colibri the backup.
func TestHarnessSettledFollowsLlamaPrimaryRule(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		settled bool
	}{
		{"llama starting: wait", `{"primary_state":"starting","colibri_state":"stopped"}`, false},
		{"llama serving: go", `{"primary_state":"serving","colibri_state":"stopped"}`, true},
		{"llama failed, colibri starting: wait", `{"primary_state":"failed","colibri_state":"starting"}`, false},
		{"llama failed, colibri serving: go", `{"primary_state":"failed","colibri_state":"serving"}`, true},
		{"llama out for good, colibri out for good: go", `{"primary_state":"unavailable","colibri_state":"unavailable"}`, true},
		{"llama out for good, colibri installing: wait", `{"primary_state":"unavailable","colibri_state":"installing"}`, false},
		{"unreadable: wait", `not json`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/harness/status" {
					http.NotFound(w, r)
					return
				}
				fmt.Fprint(w, c.body)
			}))
			defer srv.Close()
			ok, why := harnessSettled(srv.Client(), srv.URL)
			if ok != c.settled {
				t.Fatalf("settled=%v (%s), want %v", ok, why, c.settled)
			}
		})
	}
}

func TestHarnessNotAnsweringIsNotSettled(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // nothing listening
	if ok, _ := harnessSettled(http.DefaultClient, srv.URL); ok {
		t.Fatal("a harness that is not answering must not open the gate")
	}
}

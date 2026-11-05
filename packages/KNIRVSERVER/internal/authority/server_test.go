package authority

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
)

const testToken = "internal-test-token"

type apiClient struct {
	http  *http.Client
	token string
}

func (c apiClient) do(t *testing.T, method, path, body string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, "http://unix"+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s %s: non-JSON body %q", method, path, raw)
	}
	return resp.StatusCode, out
}

// startServer serves svc on a fresh socket (short path: sun_path is ~108 bytes).
func startServer(t *testing.T, svc *Service, anchor *Anchor) (string, apiClient) {
	t.Helper()
	dir, err := os.MkdirTemp("", "auth")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "authority.sock")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	srv := &Server{Svc: svc, Anchor: anchor, Token: testToken}
	go func() { done <- srv.Serve(ctx, sock) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("Serve did not stop after its context ended")
		}
	})
	deadline := time.Now().Add(5 * time.Second)
	for {
		if c, err := net.Dial("unix", sock); err == nil {
			c.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("server did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	client := apiClient{token: testToken, http: &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", sock)
		},
	}}}
	return sock, client
}

func TestHTTPAuthorizeFetchAndStatusForRootAndBoot(t *testing.T) {
	for _, tc := range []struct {
		kind string
		seed byte
	}{{KindRoot, 1}, {KindBoot, 2}} {
		t.Run(tc.kind, func(t *testing.T) {
			h := newHarness(t, tc.kind, tc.seed)
			_, c := startServer(t, h.svc, nil)

			code, body := c.do(t, "POST", "/authority/authorize", `{"tier":1,"domain":"evidence.commit","action":{"event":"e1","n":1}}`)
			if code != 200 || body["verified"] != true || body["mode"] != "observe" {
				t.Fatalf("authorize: %d %v", code, body)
			}
			hash, _ := body["hash"].(string)
			rec := body["record"].(map[string]any)
			if rec["hash"] != hash || rec["operator"] != h.signer.Address() || rec["witness_kind"] != "sw" {
				t.Fatalf("record %v", rec)
			}
			// The returned JSON is the core's wire shape: it converts back and verifies.
			raw, _ := json.Marshal(rec)
			var wire RecordJSON
			if err := json.Unmarshal(raw, &wire); err != nil {
				t.Fatal(err)
			}
			back, err := FromJSON(wire)
			if err != nil {
				t.Fatal(err)
			}
			if err := sdk.Verify(back, registryFor(h.signer, tc.kind), h.clock.Now()); err != nil {
				t.Fatalf("record returned over HTTP does not verify: %v", err)
			}

			code, got := c.do(t, "GET", "/authority/records/"+hash, "")
			if code != 200 || got["verified"] != true || got["issued_at"] == nil || got["note"] == nil {
				t.Fatalf("get record: %d %v", code, got)
			}
			code, st := c.do(t, "GET", "/authority/status", "")
			if code != 200 || st["state"] != "ok" || st["mode"] != "observe" || st["kind"] != tc.kind || st["records"] != float64(1) || st["chain_id"] != "authority/"+h.signer.Address() {
				t.Fatalf("status: %d %v", code, st)
			}
		})
	}
}

func TestHTTPBootNodeForeignScopeIsRefused(t *testing.T) {
	h := newHarness(t, KindBoot, 2)
	_, c := startServer(t, h.svc, nil)
	code, body := c.do(t, "POST", "/authority/authorize", `{"tier":1,"domain":"evidence.commit","scope":"knirv1someoneelse","action":{}}`)
	if code != http.StatusForbidden || body["code"] != RefusedScope {
		t.Fatalf("want 403 scope refusal, got %d %v", code, body)
	}
	if h.store.Total() != 0 {
		t.Fatal("a refused request stored a record")
	}
}

func TestHTTPRejectsBadRequests(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	_, c := startServer(t, h.svc, nil)
	cases := []struct {
		name, method, path, body string
		want                     int
	}{
		{"unknown field", "POST", "/authority/authorize", `{"tier":1,"domain":"a.b","action":{},"surprise":1}`, 400},
		{"not JSON", "POST", "/authority/authorize", `nope`, 400},
		{"non-canonical action", "POST", "/authority/authorize", `{"tier":1,"domain":"a.b","action":{"x":0.5}}`, 400},
		{"wrong method on authorize", "GET", "/authority/authorize", ``, 405},
		{"wrong method on status", "POST", "/authority/status", `{}`, 405},
		{"hash too short", "GET", "/authority/records/abcd", ``, 400},
		{"hash upper case", "GET", "/authority/records/" + strings.Repeat("AB", 32), ``, 400},
		{"hash not hex", "GET", "/authority/records/" + strings.Repeat("zz", 32), ``, 400},
		{"unknown record", "GET", "/authority/records/" + strings.Repeat("ab", 32), ``, 404},
		{"unknown route", "GET", "/authority/nothing", ``, 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if code, body := c.do(t, tc.method, tc.path, tc.body); code != tc.want {
				t.Fatalf("got %d %v, want %d", code, body, tc.want)
			}
		})
	}
	big := `{"tier":1,"domain":"a.b","action":"` + strings.Repeat("x", maxRequestBytes) + `"}`
	if code, _ := c.do(t, "POST", "/authority/authorize", big); code != 400 {
		t.Fatalf("an oversized body should be refused, got %d", code)
	}
}

func TestHTTPRequiresTheInternalToken(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	_, c := startServer(t, h.svc, nil)
	for _, token := range []string{"", "wrong", testToken + "x"} {
		bad := apiClient{http: c.http, token: token}
		if code, _ := bad.do(t, "GET", "/authority/status", ""); code != http.StatusUnauthorized {
			t.Errorf("token %q: got %d", token, code)
		}
		if code, _ := bad.do(t, "POST", "/authority/authorize", `{"tier":1,"domain":"a.b","action":{}}`); code != http.StatusUnauthorized {
			t.Errorf("token %q authorize: got %d", token, code)
		}
	}
	if h.store.Total() != 0 {
		t.Fatal("an unauthorized request stored a record")
	}
}

func TestHTTPDegradedService(t *testing.T) {
	svc, err := New(Config{DegradedReason: "root.key present but unusable (no fallback to boot.key): ORACLE_KEY_PASSWORD is not set"})
	if err != nil {
		t.Fatal(err)
	}
	_, c := startServer(t, svc, nil)
	if code, body := c.do(t, "POST", "/authority/authorize", `{"tier":1,"domain":"a.b","action":{}}`); code != 503 || body["code"] != RefusedDegraded {
		t.Fatalf("authorize: %d %v", code, body)
	}
	if code, body := c.do(t, "GET", "/authority/records/"+strings.Repeat("ab", 32), ""); code != 503 {
		t.Fatalf("records: %d %v", code, body)
	}
	code, st := c.do(t, "GET", "/authority/status", "")
	if code != 200 || st["state"] != "degraded" || !strings.Contains(st["reason"].(string), "ORACLE_KEY_PASSWORD") {
		t.Fatalf("status: %d %v", code, st)
	}
}

func TestHTTPStatusIncludesAnchor(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	_, poster := oracleHarness(t)
	a := newAnchor(t, h, poster, 0, "")
	mustAuthorize(t, h, act(1, "evidence.commit", "", `{}`))
	if err := a.Tick(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, c := startServer(t, h.svc, a)
	_, st := c.do(t, "GET", "/authority/status", "")
	anchor, ok := st["anchor"].(map[string]any)
	if !ok || anchor["registered"] != true || anchor["last_end"] != float64(1) {
		t.Fatalf("anchor in status: %v", st["anchor"])
	}
}

func TestSocketIsOwnerOnlyAndNotStolen(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	sock, _ := startServer(t, h.svc, nil)
	info, err := os.Stat(sock)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("socket mode %v gives group or other access", info.Mode().Perm())
	}
	// A second server on a socket that still answers must refuse, not take it over.
	other := &Server{Svc: h.svc}
	err = other.Serve(context.Background(), sock)
	if err == nil || !strings.Contains(err.Error(), "already served") {
		t.Fatalf("want an already-served error, got %v", err)
	}
}

func TestStaleSocketFileIsReplaced(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	dir, _ := os.MkdirTemp("", "auth")
	defer os.RemoveAll(dir)
	sock := filepath.Join(dir, "a.sock")
	// Leave a dead socket file behind: listen, then close without unlinking.
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	l.(*net.UnixListener).SetUnlinkOnClose(false)
	l.Close()
	if _, err := os.Stat(sock); err != nil {
		t.Fatal("test setup: no stale socket left")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- (&Server{Svc: h.svc}).Serve(ctx, sock) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if c, err := net.Dial("unix", sock); err == nil {
			c.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("did not take over a stale socket")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

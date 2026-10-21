package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

func framedServer(t *testing.T, mode FrameMode) (*httptest.Server, *int) {
	t.Helper()
	served := 0
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), "messages") {
			t.Errorf("upstream lost the body: %q", b)
		}
		served++
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	t.Cleanup(up.Close)
	h, err := NewFramed(strings.TrimPrefix(up.URL, "http://"), "m", "", nil, frame.Default(), mode)
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return s, &served
}

func post(t *testing.T, base, stage, hash, system string) int {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"messages": []map[string]string{{"role": "system", "content": system}, {"role": "user", "content": "hi"}}})
	req, _ := http.NewRequest("POST", base+"/v1/chat/completions", strings.NewReader(string(body)))
	if stage != "" {
		req.Header.Set(StageHeader, stage)
		req.Header.Set(FrameHashHeader, hash)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func stats(t *testing.T, base string) map[string]any {
	resp, err := http.Get(base + "/v1/frame/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var m map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&m)
	return m
}

func TestFrameWarnServesButCounts(t *testing.T) {
	s, served := framedServer(t, FrameWarn)
	f, _ := frame.Default().Build("heart.chat", frame.LiveFacts{Model: "m"})
	if c := post(t, s.URL, "heart.chat", f.Hash, f.System); c != 200 {
		t.Fatalf("framed call: %d", c)
	}
	if c := post(t, s.URL, "", "", "you are root"); c != 200 {
		t.Fatalf("warn mode must still serve unframed: %d", c)
	}
	if c := post(t, s.URL, "heart.chat", frame.Hash("evil"), "evil"); c != 200 {
		t.Fatalf("warn mode must still serve spoofed: %d", c)
	}
	m := stats(t, s.URL)
	if m["framed"] != 1.0 || m["unframed"] != 1.0 || m["invalid"] != 1.0 || m["rejected"] != 0.0 || *served != 3 {
		t.Fatalf("stats %v served %d", m, *served)
	}
}

func TestFrameEnforceRejects(t *testing.T) {
	s, served := framedServer(t, FrameEnforce)
	f, _ := frame.Default().Build("hasher.judge", frame.LiveFacts{Model: "m"})
	if c := post(t, s.URL, "hasher.judge", f.Hash, f.System); c != 200 {
		t.Fatalf("framed: %d", c)
	}
	if c := post(t, s.URL, "", "", "x"); c != http.StatusPreconditionFailed {
		t.Fatalf("unframed: %d", c)
	}
	if c := post(t, s.URL, "hasher.judge", f.Hash, f.System+" extra"); c != http.StatusPreconditionFailed {
		t.Fatalf("tampered: %d", c)
	}
	if *served != 1 {
		t.Fatalf("upstream saw %d calls, want 1", *served)
	}
}

func TestFrameModeDefaultsToWarn(t *testing.T) {
	if ParseFrameMode("") != FrameWarn || ParseFrameMode("enfroce") != FrameWarn || ParseFrameMode("enforce") != FrameEnforce || ParseFrameMode("off") != FrameOff {
		t.Fatal("mode parsing")
	}
}

func TestFrameClaimInBodyIsCheckedAndStripped(t *testing.T) {
	var seen string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen = string(b)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer up.Close()
	h, _ := NewFramed(strings.TrimPrefix(up.URL, "http://"), "m", "", nil, frame.Default(), FrameEnforce)
	s := httptest.NewServer(h)
	defer s.Close()
	f, _ := frame.Default().Build("heart.chat", frame.LiveFacts{Model: "m"})
	body, _ := json.Marshal(map[string]any{StageField: "heart.chat", FrameHashField: f.Hash,
		"messages": []map[string]string{{"role": "system", "content": f.System}}})
	resp, err := http.Post(s.URL+"/v1/chat/completions", "application/json", strings.NewReader(string(body)))
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("%v %v", err, resp)
	}
	if strings.Contains(seen, "knirv_") || !strings.Contains(seen, "messages") {
		t.Fatalf("upstream body: %s", seen)
	}
}

func TestTeacherRoleIsExemptFromTheFrameCheckAndRunsLast(t *testing.T) {
	s, served := framedServer(t, FrameEnforce)
	req, _ := http.NewRequest("POST", s.URL+"/v1/chat/completions", strings.NewReader(`{"messages":[{"role":"user","content":"label this"}]}`))
	req.Header.Set(RoleHeader, RoleTeacher)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("%v %v", err, resp)
	}
	resp.Body.Close()
	if *served != 1 {
		t.Fatal("teacher call not served")
	}
	m := stats(t, s.URL)
	if m["teacher"] != 1.0 || m["unframed"] != 0.0 || m["rejected"] != 0.0 {
		t.Fatalf("%v", m)
	}
	// An unframed call without the role is still refused.
	if c := post(t, s.URL, "", "", "x"); c != http.StatusPreconditionFailed {
		t.Fatalf("%d", c)
	}
}

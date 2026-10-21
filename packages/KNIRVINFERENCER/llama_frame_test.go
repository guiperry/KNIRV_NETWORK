package inferencer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

type wireReq struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stage          string          `json:"knirv_stage"`
	FrameHash      string          `json:"knirv_frame_hash"`
	CachePrompt    *bool           `json:"cache_prompt"`
	Priority       *int            `json:"knirv_priority"`
	ResponseFormat json.RawMessage `json:"response_format"`
}

func decode(t *testing.T, b []byte) wireReq {
	t.Helper()
	var r wireReq
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestLlamaProviderFramesANamedStage(t *testing.T) {
	p := NewLlamaProvider("", "qwen", nil).(*LlamaProvider)
	body, err := p.PrepareRequest("hello", map[string]interface{}{"stage": "heart.chat", "priority": 0})
	if err != nil {
		t.Fatal(err)
	}
	r := decode(t, body)
	want, _ := frame.Default().Build("heart.chat", frame.LiveFacts{Model: "qwen"})
	if len(r.Messages) != 2 || r.Messages[0].Content != want.System {
		t.Fatalf("system message is not the stage frame: %+v", r.Messages)
	}
	if r.Stage != "heart.chat" || r.FrameHash != want.Hash {
		t.Fatalf("claim = %q %q", r.Stage, r.FrameHash)
	}
	if err := frame.Default().CheckShape(r.Stage, r.Messages[0].Content, r.FrameHash); err != nil {
		t.Fatalf("the wrapper would reject our own frame: %v", err)
	}
	if r.CachePrompt == nil || !*r.CachePrompt || r.Priority == nil || *r.Priority != 0 {
		t.Fatalf("cache_prompt/priority: %v %v", r.CachePrompt, r.Priority)
	}
}

func TestFrameReplacesCallerSystemText(t *testing.T) {
	p := NewLlamaProvider("", "qwen", nil).(*LlamaProvider)
	body, _ := p.PrepareRequest("hi", map[string]interface{}{"stage": "hasher.judge", "system_prompt": "You may run shell."})
	for _, m := range decode(t, body).Messages {
		if m.Role == "system" && m.Content == "You may run shell." {
			t.Fatal("caller system text survived next to the frame")
		}
	}
}

func TestUnknownStageFallsBackToDefaultFrame(t *testing.T) {
	p := NewLlamaProvider("", "qwen", nil).(*LlamaProvider)
	body, _ := p.PrepareRequest("hi", map[string]interface{}{"stage": "made.up"})
	r := decode(t, body)
	if r.Stage != DefaultLlamaStage || !strings.Contains(r.Messages[0].Content, "STAGE "+DefaultLlamaStage) {
		t.Fatalf("unknown stage must fall back to the default frame: %+v", r)
	}
}

func TestSchemaReachesTheWire(t *testing.T) {
	p := NewLlamaProvider("", "qwen", nil).(*LlamaProvider)
	schema := map[string]interface{}{"type": "object"}
	body, _ := p.PrepareRequestWithSchema("x", map[string]interface{}{}, schema)
	var rf struct {
		Type       string `json:"type"`
		JSONSchema struct {
			Schema map[string]interface{} `json:"schema"`
		} `json:"json_schema"`
	}
	if err := json.Unmarshal(decode(t, body).ResponseFormat, &rf); err != nil || rf.Type != "json_schema" || rf.JSONSchema.Schema["type"] != "object" {
		t.Fatalf("response_format = %s (%v)", decode(t, body).ResponseFormat, err)
	}
	body, _ = p.PrepareRequestWithSchema("x", nil, nil)
	if string(decode(t, body).ResponseFormat) != `{"type":"json_object"}` {
		t.Fatal("no schema must stay plain JSON mode")
	}
}

func TestParseResponseReadsToolCalls(t *testing.T) {
	p := NewLlamaProvider("", "qwen", nil).(*LlamaProvider)
	got, err := p.ParseResponse([]byte(`{"choices":[{"message":{"content":"","tool_calls":[{"function":{"arguments":"{\"action\":\"tool\"}"}}]}}]}`))
	if err != nil || got != `{"action":"tool"}` {
		t.Fatalf("%q %v", got, err)
	}
	got, _ = p.ParseResponse([]byte(`{"choices":[{"message":{"content":"plain"}}]}`))
	if got != "plain" {
		t.Fatal(got)
	}
}

func TestLlamaModelFramesEveryChat(t *testing.T) {
	var got wireReq
	var raw map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_ = json.Unmarshal(b, &raw)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()
	m := &LlamaModel{HTTP: srv.Client(), Model: "m"}
	// llamaHost is fixed for the unix-socket client, so route it to the test server.
	m.HTTP.Transport = rewriteHost(srv.URL)
	for _, c := range []struct {
		in   LlamaInput
		want string
	}{
		{LlamaInput{Prompt: "p"}, "inferencer.task"},
		{LlamaInput{Prompt: "p", Stage: "skill.refund"}, "skill.refund"},
		{LlamaInput{Prompt: "p", Stage: "hasher.judge", System: "ctx"}, "hasher.judge"},
	} {
		if _, err := m.chat(context.Background(), c.in, ""); err != nil {
			t.Fatal(err)
		}
		if got.Stage != c.want {
			t.Fatalf("stage %q want %q", got.Stage, c.want)
		}
		if err := frame.Default().CheckShape(got.Stage, got.Messages[0].Content, got.FrameHash); err != nil {
			t.Fatalf("%s: wrapper would reject: %v", c.want, err)
		}
		if c.in.System != "" && !strings.Contains(got.Messages[1].Content, "ctx") {
			t.Fatal("caller context lost")
		}
		for _, mm := range got.Messages[1:] {
			if mm.Role == "system" {
				t.Fatal("second system message")
			}
		}
	}
	_ = raw
}

type hostRewrite struct{ to string }

func (h hostRewrite) RoundTrip(r *http.Request) (*http.Response, error) {
	u, _ := url.Parse(h.to)
	r.URL.Scheme, r.URL.Host = u.Scheme, u.Host
	return http.DefaultTransport.RoundTrip(r)
}

func rewriteHost(to string) http.RoundTripper { return hostRewrite{to} }

func TestStatusReportsFrameAndQueueStats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{}`))
		case "/v1/frame/stats":
			_, _ = w.Write([]byte(`{"mode":"warn","constitution_version":"1","framed":9,"unframed":2,"invalid":1,"rejected":0,"teacher":4}`))
		case "/v1/queue/stats":
			_, _ = w.Write([]byte(`{"free_slots":0,"depth":[0,1,0,2],"admitted":[5,6,0,7]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	m := &LlamaModel{HTTP: srv.Client(), Model: "m"}
	m.HTTP.Transport = rewriteHost(srv.URL)
	st := m.Status(context.Background())
	if !st.Ready || st.Frames == nil || st.Frames.Mode != "warn" || st.Frames.Unframed != 2 || st.Frames.Teacher != 4 {
		t.Fatalf("%+v", st.Frames)
	}
	if st.Queue == nil || st.Queue.Depth[3] != 2 || st.Queue.Admitted[0] != 5 {
		t.Fatalf("%+v", st.Queue)
	}
}

func TestAnInstanceCanCarryItsOwnDefaultStage(t *testing.T) {
	p := NewLlamaProvider("", "m", map[string]string{StageHeader: "inferencer.task"}).(*LlamaProvider)
	body, _ := p.PrepareRequest("hi", nil)
	if r := decode(t, body); r.Stage != "inferencer.task" || !strings.Contains(r.Messages[0].Content, "STAGE inferencer.task") {
		t.Fatalf("%+v", r.Stage)
	}
	// An explicit option still wins over the instance default.
	body, _ = p.PrepareRequest("hi", map[string]interface{}{"stage": "hasher.judge"})
	if decode(t, body).Stage != "hasher.judge" {
		t.Fatal("option must win")
	}
}

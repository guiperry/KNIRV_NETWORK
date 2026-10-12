package inferencer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHRMReasoner(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/heart/hrm/reason" || r.Method != "POST" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"plan_tokens":[104,105,259,104],"halting_steps":2,"confidence":0.5,"model_version":"v1",
			"decisions":[{"name":"action","kind":"choice","selected":"allow","confidence":0.9,"abstain":false}]}`))
	}))
	defer srv.Close()
	r, err := NewHRMReasoner(srv.URL).Reason(context.Background(), []uint32{257, 1})
	if err != nil {
		t.Fatal(err)
	}
	if r.PlanText() != "hi" || r.HaltingSteps != 2 {
		t.Fatalf("%+v %q", r, r.PlanText())
	}
	if d := r.Decision("action"); d == nil || *d.Selected != "allow" {
		t.Fatalf("%+v", d)
	}
	if r.Decision("nope") != nil {
		t.Fatal("unknown head")
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer bad.Close()
	if _, err := NewHRMReasoner(bad.URL).Reason(context.Background(), []uint32{1}); err == nil {
		t.Fatal("503 must be an error")
	}
	if _, err := NewHRMReasoner("http://127.0.0.1:1").Reason(context.Background(), []uint32{1}); err == nil {
		t.Fatal("unreachable must be an error")
	}
}

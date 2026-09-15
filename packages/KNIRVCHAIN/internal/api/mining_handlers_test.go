package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The mining handlers used to return {"status":"ok"} without doing anything —
// a fabricated success for a pipeline that cannot run on this node. These tests
// exist so a "status: ok" response cannot silently return.
func TestMiningHandlersDoNotFabricateSuccess(t *testing.T) {
	// The mining handlers touch no configuration, so an empty API is enough.
	api := &UnifiedAPI{}

	for name, handler := range map[string]http.HandlerFunc{
		"propose":  api.handleMiningProposal,
		"validate": api.handleMiningValidation,
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler(rec, httptest.NewRequest(http.MethodPost, "/api/mining/"+name, nil))

			if rec.Code != http.StatusNotImplemented {
				t.Fatalf("status = %d, want 501: an unimplemented pipeline must not report success",
					rec.Code)
			}

			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["status"] == "ok" {
				t.Fatal(`the handler must not return status "ok": the mining pipeline is not wired`)
			}
			if body["status"] != "unavailable" {
				t.Fatalf("status = %q, want %q", body["status"], "unavailable")
			}
			// The response must name a reason, not just refuse.
			if body["reason"] == "" {
				t.Fatal("an unavailable response must explain why")
			}
			// And it must point the caller at the route that really mints.
			if body["message"] == "" {
				t.Fatal("an unavailable response must direct the caller to the real entry point")
			}
		})
	}
}

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/d1"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// mockD1Server returns an httptest server that answers D1 query requests by
// matching on a substring of the SQL text, in order — the first matching
// responder wins. This lets each test wire up exactly the sequence of
// queries a handler is expected to issue without a real database.
type d1Responder struct {
	sqlContains string
	respond     func(w http.ResponseWriter, sql string, params []any)
}

func newMockD1Client(t *testing.T, responders []d1Responder) *d1.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SQL    string `json:"sql"`
			Params []any  `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("mock D1: decode request: %v", err)
		}
		for _, resp := range responders {
			if strings.Contains(body.SQL, resp.sqlContains) {
				resp.respond(w, body.SQL, body.Params)
				return
			}
		}
		t.Fatalf("mock D1: no responder matched SQL: %s", body.SQL)
	}))
	t.Cleanup(srv.Close)
	return &d1.Client{AccountID: "acct", DatabaseID: "db", APIToken: "tok", BaseURL: srv.URL, HTTPClient: srv.Client()}
}

func writeD1Rows(w http.ResponseWriter, rows []map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	resultBytes, _ := json.Marshal(rows)
	fmt.Fprintf(w, `{"success":true,"errors":[],"result":[{"results":%s,"success":true,"meta":{"changes":0,"last_row_id":0}}]}`, resultBytes)
}

func writeD1Changes(w http.ResponseWriter, changes int64) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success":true,"errors":[],"result":[{"results":[],"success":true,"meta":{"changes":%d,"last_row_id":0}}]}`, changes)
}

func newTestServer(d1Client *d1.Client) *Server {
	logger, _ := zap.NewDevelopment()
	return &Server{logger: logger, d1Client: d1Client}
}

func withMuxVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}

func TestRequireD1_NoCredentials(t *testing.T) {
	s := newTestServer(nil)
	rec := httptest.NewRecorder()
	s.handleAdminOperatorsList(rec, httptest.NewRequest(http.MethodGet, "/api/admin/operators", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleAdminOperatorsList_JoinsRegistrationStatus(t *testing.T) {
	client := newMockD1Client(t, []d1Responder{
		{sqlContains: "FROM operator_applications", respond: func(w http.ResponseWriter, sql string, params []any) {
			writeD1Rows(w, []map[string]any{
				{"id": "app-1", "legal_name": "Acme Co", "kyc_status": "pending", "created_at": "2026-01-01T00:00:00Z", "status": "validated"},
			})
		}},
	})
	s := newTestServer(client)

	rec := httptest.NewRecorder()
	s.handleAdminOperatorsList(rec, httptest.NewRequest(http.MethodGet, "/api/admin/operators", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var parsed struct {
		Data struct {
			Applications []operatorApplicationRow `json:"applications"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(parsed.Data.Applications) != 1 || parsed.Data.Applications[0].Status != "validated" {
		t.Errorf("applications = %+v", parsed.Data.Applications)
	}
}

func TestHandleAdminOperatorReview_RejectsApprovalWhenNotValidated(t *testing.T) {
	client := newMockD1Client(t, []d1Responder{
		{sqlContains: "SELECT registration_id, status", respond: func(w http.ResponseWriter, sql string, params []any) {
			writeD1Rows(w, []map[string]any{{"registration_id": "reg-1", "status": "pending"}})
		}},
	})
	s := newTestServer(client)

	body, _ := json.Marshal(map[string]string{"decision": "approved"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/operators/app-1/review", bytes.NewReader(body))
	req = withMuxVars(req, map[string]string{"id": "app-1"})
	rec := httptest.NewRecorder()

	s.handleAdminOperatorReview(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s, want %d", rec.Code, rec.Body.String(), http.StatusConflict)
	}
}

func TestHandleAdminOperatorReview_ApprovesWhenValidated(t *testing.T) {
	var updateSQL string
	var updateParams []any
	client := newMockD1Client(t, []d1Responder{
		{sqlContains: "SELECT registration_id, status", respond: func(w http.ResponseWriter, sql string, params []any) {
			writeD1Rows(w, []map[string]any{{"registration_id": "reg-1", "status": "validated"}})
		}},
		{sqlContains: "UPDATE node_registrations", respond: func(w http.ResponseWriter, sql string, params []any) {
			updateSQL = sql
			updateParams = params
			writeD1Changes(w, 1)
		}},
	})
	s := newTestServer(client)

	body, _ := json.Marshal(map[string]string{"decision": "approved"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/operators/app-1/review", bytes.NewReader(body))
	req = withMuxVars(req, map[string]string{"id": "app-1"})
	rec := httptest.NewRecorder()

	s.handleAdminOperatorReview(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if updateSQL == "" {
		t.Fatal("expected an UPDATE statement to be issued")
	}
	if len(updateParams) != 3 || updateParams[0] != "reviewed" || updateParams[2] != "reg-1" {
		t.Errorf("updateParams = %v, want [reviewed, <timestamp>, reg-1]", updateParams)
	}
}

func TestHandleAdminOperatorReview_RejectsInvalidDecision(t *testing.T) {
	s := newTestServer(newMockD1Client(t, nil))
	body, _ := json.Marshal(map[string]string{"decision": "maybe"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/operators/app-1/review", bytes.NewReader(body))
	req = withMuxVars(req, map[string]string{"id": "app-1"})
	rec := httptest.NewRecorder()

	s.handleAdminOperatorReview(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleAdminUserRole_RejectsUnknownRole(t *testing.T) {
	s := newTestServer(newMockD1Client(t, nil))
	body, _ := json.Marshal(map[string]string{"role": "superuser"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users/u1/role", bytes.NewReader(body))
	req = withMuxVars(req, map[string]string{"id": "u1"})
	rec := httptest.NewRecorder()

	s.handleAdminUserRole(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleAdminUserRole_UpdatesRole(t *testing.T) {
	client := newMockD1Client(t, []d1Responder{
		{sqlContains: "UPDATE accounts SET role", respond: func(w http.ResponseWriter, sql string, params []any) {
			writeD1Changes(w, 1)
		}},
	})
	s := newTestServer(client)

	body, _ := json.Marshal(map[string]string{"role": "validator"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users/u1/role", bytes.NewReader(body))
	req = withMuxVars(req, map[string]string{"id": "u1"})
	rec := httptest.NewRecorder()

	s.handleAdminUserRole(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAdminUserStatus_NotFoundWhenNoRowsChanged(t *testing.T) {
	client := newMockD1Client(t, []d1Responder{
		{sqlContains: "UPDATE accounts SET account_status", respond: func(w http.ResponseWriter, sql string, params []any) {
			writeD1Changes(w, 0)
		}},
	})
	s := newTestServer(client)

	body, _ := json.Marshal(map[string]string{"status": "suspended"})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users/nonexistent/status", bytes.NewReader(body))
	req = withMuxVars(req, map[string]string{"id": "nonexistent"})
	rec := httptest.NewRecorder()

	s.handleAdminUserStatus(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

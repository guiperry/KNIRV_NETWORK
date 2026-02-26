// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"backend_server/internal/database"
	"backend_server/internal/services/agent"
	"backend_server/internal/web/middleware"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newAgentTestDB returns a file-backed BuntDB in a temp directory.
func newAgentTestDB(t *testing.T) *database.BuntDBManager {
	t.Helper()
	tmpDir := t.TempDir()
	db, err := database.NewBuntDB(filepath.Join(tmpDir, "agent_handler_test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

// newStartedAgentService creates and starts an AgentService backed by a real
// DB but with a nil UCM (container manager). This is sufficient for all
// handler-level tests that do not exercise the container provisioning path.
func newStartedAgentService(t *testing.T) *agent.AgentService {
	t.Helper()
	db := newAgentTestDB(t)
	svc := agent.NewAgentService(db, nil, nil)
	require.NoError(t, svc.Start())
	t.Cleanup(func() { svc.Stop() })
	return svc
}

// buildReq creates an httptest.Request and wires gorilla/mux URL vars.
func buildReq(method, url string, body []byte, vars map[string]string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	if len(vars) > 0 {
		req = mux.SetURLVars(req, vars)
	}
	return req
}

// decodeResp decodes the JSON response body into a generic map.
func decodeResp(t *testing.T, body *bytes.Buffer) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(body).Decode(&resp), "response must be valid JSON")
	return resp
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewAgentHandlers(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)
	require.NotNil(t, h)
	assert.Equal(t, svc, h.agentService)
}

func TestNewAgentHandlers_Nil(t *testing.T) {
	// Constructing with a nil service must not panic.
	h := NewAgentHandlers(nil)
	require.NotNil(t, h)
}

// ---------------------------------------------------------------------------
// GET /api/dve/{id}/agent/status  → 200 with stopped status
// ---------------------------------------------------------------------------

func TestAgentHandlers_GetAgentStatus_Stopped(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	req := buildReq("GET", "/api/dve/dve-no-agent/agent/status", nil,
		map[string]string{"id": "dve-no-agent"})
	w := httptest.NewRecorder()

	h.GetAgentStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeResp(t, w.Body)
	assert.Equal(t, true, resp["success"])

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok, "data field must be an object")
	assert.Equal(t, false, data["running"])
	assert.Equal(t, "dve-no-agent", data["dve_id"])
}

func TestAgentHandlers_GetAgentStatus_ContentType(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	req := buildReq("GET", "/api/dve/dve-ct/agent/status", nil,
		map[string]string{"id": "dve-ct"})
	w := httptest.NewRecorder()

	h.GetAgentStatus(w, req)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestAgentHandlers_GetAgentStatus_TimestampPresent(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	req := buildReq("GET", "/api/dve/dve-ts/agent/status", nil,
		map[string]string{"id": "dve-ts"})
	w := httptest.NewRecorder()

	h.GetAgentStatus(w, req)

	resp := decodeResp(t, w.Body)
	ts, _ := resp["timestamp"].(string)
	assert.NotEmpty(t, ts, "response must include a non-empty timestamp")
}

// ---------------------------------------------------------------------------
// POST /api/dve/{id}/agent/launch  (nil UCM → error)
// ---------------------------------------------------------------------------

func TestAgentHandlers_LaunchAgent_NilUCM(t *testing.T) {
	// Service has a nil UCM; LaunchAgent must return an error → 500.
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	req := buildReq("POST", "/api/dve/dve-launch/agent/launch", nil,
		map[string]string{"id": "dve-launch"})
	w := httptest.NewRecorder()

	h.LaunchAgent(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	resp := decodeResp(t, w.Body)
	errMsg, _ := resp["error"].(string)
	assert.NotEmpty(t, errMsg, "response must contain an error message")
}

// ---------------------------------------------------------------------------
// GET /api/dve/{id}/agent/tasks  → empty list
// ---------------------------------------------------------------------------

func TestAgentHandlers_GetAgentTasks_EmptyList(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	req := buildReq("GET", "/api/dve/dve-tasks/agent/tasks", nil,
		map[string]string{"id": "dve-tasks"})
	w := httptest.NewRecorder()

	h.GetAgentTasks(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeResp(t, w.Body)
	assert.Equal(t, true, resp["success"])

	// Even when no tasks exist the data field must be an array, not null.
	data, ok := resp["data"].([]interface{})
	require.True(t, ok, "data must be a JSON array even when empty")
	assert.Empty(t, data)
}

// ---------------------------------------------------------------------------
// POST /api/dve/{id}/agent/tasks  – validation errors
// ---------------------------------------------------------------------------

func TestAgentHandlers_SubmitAgentTask_MissingTitle(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	body, _ := json.Marshal(map[string]string{"description": "no title here"})
	req := buildReq("POST", "/api/dve/dve-notitle/agent/tasks", body,
		map[string]string{"id": "dve-notitle"})
	w := httptest.NewRecorder()

	h.SubmitAgentTask(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := decodeResp(t, w.Body)
	errMsg, _ := resp["error"].(string)
	assert.Contains(t, errMsg, "title is required")
}

func TestAgentHandlers_SubmitAgentTask_InvalidJSON(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	req := buildReq("POST", "/api/dve/dve-badjson/agent/tasks",
		[]byte("not-json!!"), map[string]string{"id": "dve-badjson"})
	w := httptest.NewRecorder()

	h.SubmitAgentTask(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := decodeResp(t, w.Body)
	errMsg, _ := resp["error"].(string)
	assert.Contains(t, errMsg, "invalid request body")
}

func TestAgentHandlers_SubmitAgentTask_NonRunningDVE(t *testing.T) {
	// The DVE has no running agent → SubmitTask returns error → 500.
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	body, _ := json.Marshal(map[string]string{
		"title":       "My Task",
		"description": "Do something",
	})
	req := buildReq("POST", "/api/dve/dve-no-agent/agent/tasks", body,
		map[string]string{"id": "dve-no-agent"})
	w := httptest.NewRecorder()

	h.SubmitAgentTask(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	resp := decodeResp(t, w.Body)
	errMsg, _ := resp["error"].(string)
	assert.Contains(t, errMsg, "no running agent")
}

// ---------------------------------------------------------------------------
// GET /api/dve/{id}/agent/tasks/{taskID}
// ---------------------------------------------------------------------------

func TestAgentHandlers_GetAgentTask_NotFound(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	req := buildReq("GET", "/api/dve/dve-x/agent/tasks/ghost-id", nil,
		map[string]string{"id": "dve-x", "taskID": "ghost-id"})
	w := httptest.NewRecorder()

	h.GetAgentTask(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := decodeResp(t, w.Body)
	errMsg, _ := resp["error"].(string)
	assert.Contains(t, errMsg, "task not found")
}

// ---------------------------------------------------------------------------
// DELETE /api/dve/{id}/agent  (StopAgent)
// ---------------------------------------------------------------------------

func TestAgentHandlers_StopAgent_NoAgent(t *testing.T) {
	// Stopping a non-existent agent is a no-op and must succeed.
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	req := buildReq("DELETE", "/api/dve/dve-ghost/agent", nil,
		map[string]string{"id": "dve-ghost"})
	w := httptest.NewRecorder()

	h.StopAgent(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeResp(t, w.Body)
	assert.Equal(t, true, resp["success"])
}

// ---------------------------------------------------------------------------
// RegisterRoutes
// ---------------------------------------------------------------------------

func TestAgentHandlers_RegisterRoutes(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)
	router := mux.NewRouter()
	authMiddleware := &middleware.AuthMiddleware{}

	assert.NotPanics(t, func() {
		h.RegisterRoutes(router, authMiddleware)
	})
}

func TestAgentHandlers_RegisterRoutes_NilAuth(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)
	router := mux.NewRouter()

	assert.NotPanics(t, func() {
		h.RegisterRoutes(router, nil)
	})
}

// ---------------------------------------------------------------------------
// Full route integration (via mux router)
// ---------------------------------------------------------------------------

func TestAgentHandlers_FullRoute_StatusViaRouter(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	router := mux.NewRouter()
	h.RegisterRoutes(router, nil)

	req := httptest.NewRequest("GET", "/api/dve/dve-router-test/agent/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAgentHandlers_FullRoute_TasksViaRouter(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	router := mux.NewRouter()
	h.RegisterRoutes(router, nil)

	req := httptest.NewRequest("GET", "/api/dve/dve-router-tasks/agent/tasks", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAgentHandlers_FullRoute_LaunchViaRouter(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	router := mux.NewRouter()
	h.RegisterRoutes(router, nil)

	req := httptest.NewRequest("POST", "/api/dve/dve-launch-router/agent/launch", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// UCM is nil → error is expected, but route must resolve (not 404/405).
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAgentHandlers_FullRoute_SubmitTaskMissingTitle(t *testing.T) {
	svc := newStartedAgentService(t)
	h := NewAgentHandlers(svc)

	router := mux.NewRouter()
	h.RegisterRoutes(router, nil)

	body, _ := json.Marshal(map[string]string{"description": "missing title"})
	req := httptest.NewRequest("POST", "/api/dve/dve-rt/agent/tasks",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

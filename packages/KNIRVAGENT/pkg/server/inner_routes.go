package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/knirvcorp/knirvagent/pkg/inner"
)

func RegisterInnerRoutes(mux *http.ServeMux, mgr *inner.InnerAgentManager) {
	mux.HandleFunc("/api/inner/spawn", handleSpawn(mgr))
	mux.HandleFunc("/api/inner/install", handleInstall(mgr))
	mux.HandleFunc("/api/inner/sessions", handleListSessions(mgr))
	mux.HandleFunc("/api/inner/tools", handleListTools(mgr))
	mux.HandleFunc("/api/inner/", handleSessionRouter(mgr))
}

func handleSpawn(mgr *inner.InnerAgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Tool    string   `json:"tool"`
			ExtraEnv []string `json:"extra_env,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.Tool == "" {
			http.Error(w, "tool is required", http.StatusBadRequest)
			return
		}
		sess, err := mgr.Spawn(req.Tool, req.ExtraEnv)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":             sess.ID,
			"tool":           sess.ToolName,
			"status":         sess.Status,
			"started_at":     sess.StartedAt,
			"workspace_dir":  sess.WorkspaceDir,
		})
	}
}

func handleInstall(mgr *inner.InnerAgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Tool string `json:"tool"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := mgr.Install(req.Tool); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleListSessions(mgr *inner.InnerAgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		sessions := mgr.ListSessions()
		result := make([]map[string]interface{}, 0, len(sessions))
		for _, s := range sessions {
			result = append(result, map[string]interface{}{
				"id":            s.ID,
				"tool":          s.ToolName,
				"status":        s.Status,
				"started_at":    s.StartedAt,
				"workspace_dir": s.WorkspaceDir,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func handleListTools(mgr *inner.InnerAgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mgr.ListTools())
	}
}

func handleSessionRouter(mgr *inner.InnerAgentManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/api/inner/")
		parts := strings.SplitN(rest, "/", 2)
		sessionID := parts[0]
		action := ""
		if len(parts) == 2 {
			action = parts[1]
		}

		switch {
		case r.Method == "DELETE" && action == "":
			mgr.Terminate(sessionID)
			w.WriteHeader(http.StatusNoContent)
		case action == "input" && r.Method == "POST":
			data, err := io.ReadAll(io.LimitReader(r.Body, 4096))
			if err != nil {
				http.Error(w, "failed to read input: "+err.Error(), http.StatusBadRequest)
				return
			}
			mgr.SendInput(sessionID, data)
			w.WriteHeader(http.StatusNoContent)
		case action == "stream" && r.Method == "GET":
			handleStream(mgr, sessionID, w, r)
		case action == "log" && r.Method == "GET":
			handleGetLog(mgr, sessionID, w)
		default:
			http.NotFound(w, r)
		}
	}
}

func handleStream(mgr *inner.InnerAgentManager, sessionID string, w http.ResponseWriter, r *http.Request) {
	sess, err := mgr.GetSession(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)
	ch := sess.Subscribe()
	defer sess.Unsubscribe(ch)

	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				return
			}
			if _, err := w.Write(chunk); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func handleGetLog(mgr *inner.InnerAgentManager, sessionID string, w http.ResponseWriter) {
	sess, err := mgr.GetSession(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	logPath := sess.WorkspaceDir + "/.supervisor-logs/" + sessionID + ".log"
	data, err := os.ReadFile(logPath)
	if err != nil {
		http.Error(w, "log not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(data)
}

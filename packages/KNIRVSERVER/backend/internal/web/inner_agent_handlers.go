package web

import (
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type InnerAgentManagerProxy interface {
	ForwardToInnerAgent(dveID, method, path string, body io.Reader) (*http.Response, error)
	GetSocketPath(dveID string) (string, error)
}

type InnerAgentHandlers struct {
	proxy InnerAgentManagerProxy
}

func NewInnerAgentHandlers(proxy InnerAgentManagerProxy) *InnerAgentHandlers {
	return &InnerAgentHandlers{proxy: proxy}
}

func (h *InnerAgentHandlers) RegisterRoutes(r *mux.Router) {
	sub := r.PathPrefix("/api/v1/dve/{dveId}/inner").Subrouter()
	sub.HandleFunc("/spawn", h.HandleSpawn).Methods("POST", "OPTIONS")
	sub.HandleFunc("/install", h.HandleInstall).Methods("POST", "OPTIONS")
	sub.HandleFunc("/sessions", h.HandleListSessions).Methods("GET", "OPTIONS")
	sub.HandleFunc("/tools", h.HandleListTools).Methods("GET", "OPTIONS")
	sub.HandleFunc("/sessions/{sessionId}", h.HandleSessionAction).Methods("DELETE", "GET", "OPTIONS")
	sub.HandleFunc("/sessions/{sessionId}/input", h.HandleSessionInput).Methods("POST", "OPTIONS")
	sub.HandleFunc("/sessions/{sessionId}/log", h.HandleSessionLog).Methods("GET", "OPTIONS")
}

func (h *InnerAgentHandlers) forward(w http.ResponseWriter, r *http.Request, dveID, innerPath string) {
	resp, err := h.proxy.ForwardToInnerAgent(dveID, r.Method, innerPath, r.Body)
	if err != nil {
		log.Printf("[InnerAgent] Forward error DVE=%s path=%s: %v", dveID, innerPath, err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.Header().Del("Access-Control-Allow-Origin")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (h *InnerAgentHandlers) HandleSpawn(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["dveId"]
	h.forward(w, r, dveID, "/api/inner/spawn")
}

func (h *InnerAgentHandlers) HandleInstall(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["dveId"]
	h.forward(w, r, dveID, "/api/inner/install")
}

func (h *InnerAgentHandlers) HandleListSessions(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["dveId"]
	h.forward(w, r, dveID, "/api/inner/sessions")
}

func (h *InnerAgentHandlers) HandleListTools(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["dveId"]
	h.forward(w, r, dveID, "/api/inner/tools")
}

func (h *InnerAgentHandlers) HandleSessionAction(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["dveId"]
	sessionID := mux.Vars(r)["sessionId"]
	innerPath := "/api/inner/" + sessionID
	h.forward(w, r, dveID, innerPath)
}

func (h *InnerAgentHandlers) HandleSessionInput(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["dveId"]
	sessionID := mux.Vars(r)["sessionId"]
	innerPath := "/api/inner/" + sessionID + "/input"
	h.forward(w, r, dveID, innerPath)
}

func (h *InnerAgentHandlers) HandleSessionLog(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["dveId"]
	sessionID := mux.Vars(r)["sessionId"]
	innerPath := "/api/inner/" + sessionID + "/log"
	h.forward(w, r, dveID, innerPath)
}

func (h *InnerAgentHandlers) HandleSessionStream(w http.ResponseWriter, r *http.Request) {
	dveID := mux.Vars(r)["dveId"]
	sessionID := mux.Vars(r)["sessionId"]

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)

	resp, err := h.proxy.ForwardToInnerAgent(dveID, "GET", "/api/inner/"+sessionID+"/stream", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			return
		}
	}
}



package bridge

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// loginAttempt tracks one in-flight WhatsApp QR pairing flow, polled by
// KNIRVCONTROLLER's existing QR-pairing UI (Phase 4's "real interface
// synergy" reuse of the phone-pairing screen customers already know).
type loginAttempt struct {
	mu       sync.Mutex
	status   string // "pending", "success", "error", "timeout"
	code     string
	errorMsg string
}

func (a *loginAttempt) snapshot() (status, code, errMsg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status, a.code, a.errorMsg
}

func (a *loginAttempt) set(status, code, errMsg string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status, a.code, a.errorMsg = status, code, errMsg
}

// bridgeHandlers implements /api/bridge/*. Every route here is bridge
// *management* API (status, pairing, the internal outbound-relay callback)
// — never the appservice's own transaction endpoint, which Tuwunel calls
// directly over a private Unix socket (see appservice.go) and is
// deliberately never exposed on the public gateway router.
type bridgeHandlers struct {
	svc *Service

	mu     sync.Mutex
	logins map[string]*loginAttempt
}

func newBridgeHandlers(svc *Service) http.Handler {
	h := &bridgeHandlers{svc: svc, logins: make(map[string]*loginAttempt)}
	r := mux.NewRouter()
	r.HandleFunc("/api/bridge/status", h.handleStatus).Methods(http.MethodGet)
	r.HandleFunc("/api/bridge/whatsapp/login", h.handleStartLogin).Methods(http.MethodPost)
	r.HandleFunc("/api/bridge/whatsapp/login/{sessionId}", h.handlePollLogin).Methods(http.MethodGet)
	r.HandleFunc("/api/bridge/whatsapp/sessions", h.handleSessions).Methods(http.MethodGet)
	r.HandleFunc("/api/bridge/relay/outbound", h.handleOutbound).Methods(http.MethodPost)
	return r
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (h *bridgeHandlers) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.status(r.Context()))
}

func (h *bridgeHandlers) handleSessions(w http.ResponseWriter, r *http.Request) {
	if !h.svc.cfg.Enabled {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "bridge disabled"})
		return
	}
	writeJSON(w, http.StatusOK, h.svc.whatsapp.Sessions())
}

func (h *bridgeHandlers) handleStartLogin(w http.ResponseWriter, r *http.Request) {
	if !h.svc.cfg.Enabled {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "bridge disabled"})
		return
	}

	var body struct {
		CustomerID string `json:"customer_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&body); err != nil || strings.TrimSpace(body.CustomerID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "customer_id is required"})
		return
	}

	qrChan, err := h.svc.whatsapp.StartLogin(r.Context(), body.CustomerID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	sessionID, err := newSessionID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to allocate session id"})
		return
	}
	attempt := &loginAttempt{status: "pending"}
	h.mu.Lock()
	h.logins[sessionID] = attempt
	h.mu.Unlock()

	go func() {
		for item := range qrChan {
			switch item.Event {
			case "code":
				attempt.set("pending", item.Code, "")
			case "success":
				attempt.set("success", "", "")
			case "timeout":
				attempt.set("timeout", "", "")
			default:
				if item.Error != nil {
					attempt.set("error", "", item.Error.Error())
				} else {
					attempt.set("error", "", "pairing ended: "+item.Event)
				}
			}
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"session_id": sessionID})
}

func (h *bridgeHandlers) handlePollLogin(w http.ResponseWriter, r *http.Request) {
	sessionID := mux.Vars(r)["sessionId"]
	h.mu.Lock()
	attempt, ok := h.logins[sessionID]
	h.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown login session"})
		return
	}
	status, code, errMsg := attempt.snapshot()
	resp := map[string]string{"status": status}
	if code != "" {
		resp["qr_code"] = code
	}
	if errMsg != "" {
		resp["error"] = errMsg
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleOutbound is the callback KNIRVSERVER's backend calls once the
// Expert-Advisor/Supervisor pipeline has a response ready for a bridged
// conversation. Gated by the same shared internal service-to-service token
// used elsewhere in KNIRVGATEWAY (e.g. the event-bundle mint proxy) — never
// reachable with an arbitrary bearer value.
func (h *bridgeHandlers) handleOutbound(w http.ResponseWriter, r *http.Request) {
	if !h.svc.cfg.Enabled {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "bridge disabled"})
		return
	}
	if h.svc.cfg.InternalAuthToken == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "bridge outbound relay requires KNIRV_INTERNAL_AUTH_TOKEN"})
		return
	}
	supplied := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(supplied), []byte(h.svc.cfg.InternalAuthToken)) != 1 {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var body struct {
		Channel    string `json:"channel"`
		CustomerID string `json:"customer_id"`
		ChatID     string `json:"chat_id"`
		Text       string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if body.Channel == "" || body.CustomerID == "" || body.ChatID == "" || body.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "channel, customer_id, chat_id, and text are required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := h.svc.deliverOutbound(ctx, body.Channel, body.CustomerID, body.ChatID, body.Text); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "delivered"})
}

func newSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

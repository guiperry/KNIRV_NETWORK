package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

// Headers (or, for clients that cannot set per-request headers, body fields)
// a framed caller sends. The claim is checked against the system message,
// never trusted. Body fields are removed before the request reaches llama-server.
const (
	StageHeader     = "X-KNIRV-Stage"
	FrameHashHeader = "X-KNIRV-Frame-Hash"
	StageField      = "knirv_stage"
	FrameHashField  = "knirv_frame_hash"
	PriorityField   = "knirv_priority"
	PriorityHeader  = "X-KNIRV-Priority"
	// RoleHeader marks batch teacher work (synthetic data, suggested labels,
	// judging). The model harness sends it to llama.cpp only when Colibri is not
	// enabled; it carries no role frame by design and runs at the lowest priority.
	RoleHeader  = "X-Knirv-Role"
	RoleTeacher = "teacher"
)

// FrameMode is how the wrapper treats a call whose frame does not check out.
type FrameMode string

const (
	FrameOff     FrameMode = "off"
	FrameWarn    FrameMode = "warn"    // count and log, still serve
	FrameEnforce FrameMode = "enforce" // reject with 412
)

// ParseFrameMode reads KNIRV_FRAME_MODE; anything unknown is warn, so a typo
// can never silently turn the check off.
func ParseFrameMode(s string) FrameMode {
	switch FrameMode(s) {
	case FrameOff, FrameEnforce:
		return FrameMode(s)
	}
	return FrameWarn
}

// FrameStats count calls by outcome since the wrapper started.
type FrameStats struct {
	Framed   atomic.Uint64
	Unframed atomic.Uint64
	Invalid  atomic.Uint64
	Rejected atomic.Uint64
	// Teacher counts role=teacher calls, which are exempt from the frame check.
	Teacher atomic.Uint64
}

func (s *FrameStats) snapshot(mode FrameMode) map[string]any {
	return map[string]any{
		"mode": mode, "constitution_version": frame.ConstitutionVersion,
		"framed": s.Framed.Load(), "unframed": s.Unframed.Load(),
		"invalid": s.Invalid.Load(), "rejected": s.Rejected.Load(), "teacher": s.Teacher.Load(),
	}
}

// withFrame verifies the role frame on chat completions. It reads the body to
// find the first system message, then restores it for the next handler.
func withFrame(next http.Handler, reg *frame.Registry, mode FrameMode, stats *FrameStats) http.Handler {
	if mode == FrameOff || reg == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBytes))
		if err != nil {
			jsonError(w, http.StatusRequestEntityTooLarge, "request body too large or unreadable")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
		if strings.EqualFold(r.Header.Get(RoleHeader), RoleTeacher) {
			stats.Teacher.Add(1)
			next.ServeHTTP(w, r)
			return
		}
		stage, hash := r.Header.Get(StageHeader), r.Header.Get(FrameHashHeader)
		verr := reg.CheckShape(stage, firstSystemMessage(raw), hash)
		switch {
		case verr == nil:
			stats.Framed.Add(1)
		case errors.Is(verr, frame.ErrUnframed):
			stats.Unframed.Add(1)
		default:
			stats.Invalid.Add(1)
		}
		if verr != nil {
			log.Printf("[KNIRVLLAMA] frame check failed (mode=%s stage=%q): %v", mode, stage, verr)
			if mode == FrameEnforce {
				stats.Rejected.Add(1)
				jsonError(w, http.StatusPreconditionFailed, "role frame check failed: "+verr.Error())
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// firstSystemMessage returns the content of the first system message, or "".
func firstSystemMessage(raw []byte) string {
	var req struct {
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
	}
	if json.Unmarshal(raw, &req) != nil {
		return ""
	}
	for _, m := range req.Messages {
		if m.Role == "system" {
			s, _ := m.Content.(string)
			return s
		}
	}
	return ""
}

func registerFrameRoutes(mux *http.ServeMux, mode FrameMode, stats *FrameStats) {
	mux.HandleFunc("GET /v1/frame/stats", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats.snapshot(mode))
	})
}

// withClaims moves the knirv_* body fields into request headers and removes
// them from the body, so llama-server never sees them and the later layers
// (frame check, admission) read one place. Headers already set win.
func withClaims(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBytes))
		if err != nil {
			jsonError(w, http.StatusRequestEntityTooLarge, "request body too large or unreadable")
			return
		}
		raw, stage, hash, prio := stripFrameFields(raw)
		setIfEmpty(r.Header, StageHeader, stage)
		setIfEmpty(r.Header, FrameHashHeader, hash)
		setIfEmpty(r.Header, PriorityHeader, prio)
		r.Body = io.NopCloser(bytes.NewReader(raw))
		r.ContentLength = int64(len(raw))
		next.ServeHTTP(w, r)
	})
}

func setIfEmpty(h http.Header, k, v string) {
	if v != "" && h.Get(k) == "" {
		h.Set(k, v)
	}
}

// stripFrameFields removes the frame claim from a JSON object body and returns
// the body to forward plus the claim. A body that is not an object is returned as is.
func stripFrameFields(raw []byte) (out []byte, stage, hash, prio string) {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return raw, "", "", ""
	}
	_, hasStage := m[StageField]
	_, hasHash := m[FrameHashField]
	_, hasPrio := m[PriorityField]
	if !hasStage && !hasHash && !hasPrio {
		return raw, "", "", ""
	}
	_ = json.Unmarshal(m[StageField], &stage)
	_ = json.Unmarshal(m[FrameHashField], &hash)
	if p, ok := m[PriorityField]; ok {
		var n int
		if json.Unmarshal(p, &n) == nil {
			prio = strconv.Itoa(n)
		}
	}
	delete(m, StageField)
	delete(m, FrameHashField)
	delete(m, PriorityField)
	b, err := json.Marshal(m)
	if err != nil {
		return raw, stage, hash, prio
	}
	return b, stage, hash, prio
}

package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/guiperry/knirv/llama/internal/adapters"
)

const (
	maxRequestBytes = 32 << 20
	// AdapterHeader tells the caller which adapter actually served the request
	// ("base" when none): a trace must say what ran, not what was asked for.
	AdapterHeader = "X-KNIRV-Adapter"
	gateWait      = 30 * time.Second
)

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": msg, "code": code}})
}

// withAdapters resolves the `adapter` field to llama's per-request `lora` list.
func withAdapters(next http.Handler, mgr *adapters.Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBytes))
		if err != nil {
			jsonError(w, http.StatusRequestEntityTooLarge, "request body too large or unreadable")
			return
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(raw, &body); err != nil {
			jsonError(w, http.StatusBadRequest, "request body must be a JSON object")
			return
		}
		if _, has := body["lora"]; has {
			jsonError(w, http.StatusBadRequest, `the "lora" field is not accepted; name an adapter with "adapter"`)
			return
		}
		var name string
		if v, ok := body["adapter"]; ok {
			if err := json.Unmarshal(v, &name); err != nil {
				jsonError(w, http.StatusBadRequest, `"adapter" must be a string`)
				return
			}
			delete(body, "adapter")
		}

		leave, err := mgr.Gate().Enter(r.Context(), gateWait)
		if err != nil {
			w.Header().Set("Retry-After", "5")
			jsonError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		defer leave()

		// Resolve inside the gate: the loaded set cannot change between lookup and use.
		loaded := mgr.Loaded()
		served := "base"
		if name != "" {
			id, ok := mgr.Resolve(name)
			if !ok {
				jsonError(w, http.StatusNotFound, "adapter "+quote(name)+" is not loaded; the request was not run on the base model instead")
				return
			}
			served = loaded[id].Name
		}
		if len(loaded) > 0 {
			sel := make([]map[string]any, len(loaded))
			for i, e := range loaded {
				scale := 0.0
				if e.Name == name {
					scale = 1.0
				}
				sel[i] = map[string]any{"id": i, "scale": scale}
			}
			lora, _ := json.Marshal(sel)
			body["lora"] = lora
		}
		out, _ := json.Marshal(body)
		r.Body = io.NopCloser(bytes.NewReader(out))
		r.ContentLength = int64(len(out))
		r.Header.Set("Content-Length", itoa(len(out)))
		w.Header().Set(AdapterHeader, served)
		next.ServeHTTP(w, r)
	})
}

func quote(s string) string { b, _ := json.Marshal(s); return string(b) }
func itoa(n int) string     { b, _ := json.Marshal(n); return string(b) }

type adapterView struct {
	Name       string  `json:"name"`
	ID         *int    `json:"id,omitempty"`
	Loaded     bool    `json:"loaded"`
	Path       string  `json:"path"`
	BaseSHA256 string  `json:"base_sha256"`
	BundleHash string  `json:"bundle_hash,omitempty"`
	Skill      string  `json:"skill,omitempty"`
	Alpha      float64 `json:"alpha,omitempty"`
}

func registerAdapterRoutes(mux *http.ServeMux, mgr *adapters.Manager, model string) {
	mux.HandleFunc("GET /v1/adapters", func(w http.ResponseWriter, _ *http.Request) {
		sha, err := mgr.Base().SHA256()
		resp := map[string]any{"base": map[string]any{"model": model, "path": mgr.Base().Path, "sha256": sha}}
		if err != nil {
			resp["base"].(map[string]any)["sha256_error"] = err.Error()
		}
		views := []adapterView{}
		loaded := map[string]int{}
		for i, e := range mgr.Loaded() {
			loaded[e.Name] = i
		}
		for _, e := range mgr.Registered() {
			v := adapterView{Name: e.Name, Path: e.Path, BaseSHA256: e.BaseSHA256, BundleHash: e.BundleHash, Skill: e.Skill, Alpha: e.Alpha}
			if id, ok := loaded[e.Name]; ok {
				v.Loaded, v.ID = true, &id
			}
			views = append(views, v)
		}
		resp["adapters"] = views
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("POST /v1/adapters", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Name, Path string }
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "body must be {\"name\":..., \"path\":...}")
			return
		}
		e, err := mgr.Register(r.Context(), req.Name, req.Path)
		if err != nil {
			jsonError(w, adapterStatus(err), err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(e)
	})
	mux.HandleFunc("DELETE /v1/adapters/{name}", func(w http.ResponseWriter, r *http.Request) {
		if err := mgr.Remove(r.Context(), r.PathValue("name")); err != nil {
			jsonError(w, adapterStatus(err), err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func adapterStatus(err error) int {
	var busy *adapters.StillBusyError
	switch {
	case errors.Is(err, adapters.ErrInvalid):
		return http.StatusUnprocessableEntity
	case errors.Is(err, adapters.ErrBaseMismatch):
		return http.StatusConflict
	case errors.Is(err, adapters.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, adapters.ErrUnmanaged):
		return http.StatusNotImplemented
	case errors.As(err, &busy), errors.Is(err, adapters.ErrReloading):
		return http.StatusServiceUnavailable
	}
	return http.StatusBadGateway
}

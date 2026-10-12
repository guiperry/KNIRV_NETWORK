package head

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// Routes mounts the model's HTTP surface on mux under sdk.HeadPrefix
// (/models/head) and, for one release, the previous prefix /heart/head:
//
//	POST {prefix}/reason          {"tokens":[...]} -> Reasoning
//	GET  {prefix}/models          -> Manifest
//	GET  {prefix}/weights/{ver}   safetensors, ETag = sha256, range requests
//	GET  {prefix}/matrix          a d×d attention projection (KNIRVULORA connectors)
//
// The routes carry no CORS headers and no authentication: they are meant to be
// reached only through backend_server, over the models unix socket or loopback.
func Routes(svc *Service, mux *http.ServeMux) {
	for _, prefix := range []string{sdk.HeadPrefix, sdk.LegacyHeadPrefix} {
		RoutesAt(svc, mux, prefix)
	}
}

// RoutesAt mounts the HEAD surface under one prefix.
func RoutesAt(svc *Service, mux *http.ServeMux, prefix string) {
	mux.HandleFunc(prefix+"/reason", func(w http.ResponseWriter, r *http.Request) { reason(svc, w, r) })
	mux.HandleFunc(prefix+"/models", func(w http.ResponseWriter, r *http.Request) { models(svc, w, r) })
	mux.HandleFunc(prefix+"/weights/", func(w http.ResponseWriter, r *http.Request) { weights(svc, prefix, w, r) })
	mux.HandleFunc(prefix+"/matrix", func(w http.ResponseWriter, r *http.Request) { matrix(svc, w, r) })
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func httpStatus(err error) int {
	if errors.Is(err, ErrNoModel) {
		return http.StatusServiceUnavailable
	}
	return http.StatusInternalServerError
}

func matrix(svc *Service, w http.ResponseWriter, r *http.Request) {
	m, err := svc.Matrix(r.URL.Query().Get("version"), r.URL.Query().Get("name"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func reason(svc *Service, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Tokens []uint32 `json:"tokens"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request: " + err.Error()})
		return
	}
	raw, err := svc.Reason(r.Context(), req.Tokens)
	if err != nil {
		status := httpStatus(err)
		// Input the model rejects is the caller's error, not a server fault.
		if status == http.StatusInternalServerError && (strings.Contains(err.Error(), "tokens") || strings.Contains(err.Error(), "vocab") || strings.Contains(err.Error(), "input")) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

func models(svc *Service, w http.ResponseWriter, _ *http.Request) {
	m, err := svc.Manifest()
	if err != nil {
		writeJSON(w, httpStatus(err), map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func weights(svc *Service, prefix string, w http.ResponseWriter, r *http.Request) {
	version := strings.TrimPrefix(r.URL.Path, prefix+"/weights/")
	path, err := svc.WeightsPath(version)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown version"})
		return
	}
	etag, err := svc.ETag(version)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	f, err := os.Open(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown version"})
		return
	}
	defer f.Close()
	w.Header().Set("ETag", `"`+etag+`"`)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable") // content-addressed by version
	http.ServeContent(w, r, "", time.Time{}, f)
}

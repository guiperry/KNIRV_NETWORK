package supervision

import (
	"encoding/json"
	"net/http"
)

// NewHTTPHandler exposes only the local, typed advisory API. Deployment owns
// listener binding and authentication; this handler never executes advice.
func NewHTTPHandler(retriever Retriever, modelVersion, manifestID string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/supervision/advise", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var in SuperviseContext
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			http.Error(w, "invalid advisory context", http.StatusBadRequest)
			return
		}
		out, err := retriever.Advise(in)
		if err != nil {
			http.Error(w, "advisory unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("/supervision/version", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"model_version": modelVersion, "dataset_manifest_hash": manifestID})
	})
	return mux
}

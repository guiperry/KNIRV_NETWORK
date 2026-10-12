package models

import (
	"encoding/json"
	"net/http"
	"time"

	"knirvmodels/pkg/head"

	sdk "github.com/guiperry/knirv-sdk-go/models"
)

// Head is the registry facade over the wazero HEAD runtime.
type Head struct {
	svc   *head.Service
	start time.Time
}

func NewHead(svc *head.Service) *Head { return &Head{svc: svc, start: time.Now()} }

func (h *Head) Name() string { return "head" }

func (h *Head) Status() sdk.ModelStatus {
	st := h.svc.Status()
	ready, _ := st["ready"].(bool)
	version, _ := st["version"].(string)
	loaded, _ := st["loaded"].(bool)
	detail, _ := json.Marshal(st)
	out := sdk.ModelStatus{
		Name: "head", Role: sdk.RoleReasoner, Version: version, Ready: ready, Loaded: loaded,
		Trained: ready, Backend: "wazero", Detail: detail,
	}
	if loaded {
		out.Since = since(h.start)
	}
	if reason, _ := st["reason"].(string); !ready && reason != "" {
		out.Error = reason
	}
	return out
}

// Mount serves the HEAD under prefix and, for one release, under the previous
// /heart/head prefix so an older backend keeps working.
func (h *Head) Mount(mux *http.ServeMux, prefix string) {
	head.RoutesAt(h.svc, mux, prefix)
	head.RoutesAt(h.svc, mux, sdk.LegacyHeadPrefix)
}

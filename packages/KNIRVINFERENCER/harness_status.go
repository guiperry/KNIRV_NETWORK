package inferencer

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	models "github.com/guiperry/knirv-sdk-go/models"
)

// This file turns the harness's internals into the wire document KNIRVSDK defines
// (models.PlatformStatus), so the backend, CLI, dashboard and ARENA all show the same
// facts. Nothing here carries prompt or output text.

func clip(s string) string {
	if len(s) > 240 {
		return s[:240] + "…"
	}
	return s
}

// RoutingEvents returns up to limit recent decisions, newest first.
func (h *Harness) RoutingEvents(limit int) []models.RoutingEvent {
	traces := h.Traces(limit)
	out := make([]models.RoutingEvent, 0, len(traces))
	for _, t := range traces {
		ev := models.RoutingEvent{
			ID: t.ID, Time: t.Time.Format(time.RFC3339), Kind: string(t.Kind), Skill: t.Skill,
			Chosen: t.Chosen, Version: t.Version, Adapter: t.Adapter, Degraded: t.Degraded,
			Reason: t.Reason, LatencyMs: t.LatencyMs, Error: clip(t.Error),
		}
		for _, f := range t.Fallbacks {
			ev.Fallbacks = append(ev.Fallbacks, f.From)
		}
		for _, n := range t.Notes {
			ev.Notes = append(ev.Notes, clip(n))
		}
		out = append(out, ev)
	}
	return out
}

// RouteStats returns per-model counters and breaker state.
func (h *Harness) RouteStats() map[string]models.RouteStats {
	in := h.Stats()
	out := make(map[string]models.RouteStats, len(in))
	for name, s := range in {
		out[name] = models.RouteStats{Calls: s.Calls, Failures: s.Failures, P50Ms: s.P50Ms, P95Ms: s.P95Ms,
			Circuit: s.Circuit, LastError: clip(s.LastError)}
	}
	return out
}

// Status reads llama's health and adapter registry. It never changes anything.
func (l *LlamaModel) Status(ctx context.Context) *models.LlamaStatus {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	st := &models.LlamaStatus{}
	code, _, err := l.get(ctx, "/health")
	if err != nil {
		st.Error = clip(err.Error())
		return st
	}
	if code != 200 {
		st.Error = "health check failed"
		return st
	}
	st.Ready = true
	var ad llamaAdapters
	if code, body, err := l.get(ctx, "/v1/adapters"); err == nil && code == 200 && json.Unmarshal(body, &ad) == nil {
		st.BaseModel = filepath.Base(strings.TrimSpace(ad.Base.Path))
		if st.BaseModel == "." || st.BaseModel == string(filepath.Separator) {
			st.BaseModel = ""
		}
		st.BaseSHA256 = ad.Base.SHA256
		for _, a := range ad.Adapters {
			st.Adapters = append(st.Adapters, models.LlamaAdapter{Name: a.Name, Loaded: a.Loaded, BaseSHA256: a.BaseSHA256, BundleHash: a.BundleHash})
		}
	}
	return st
}

// BuildPlatformStatus assembles the document. Any part that cannot be read is reported
// as such in the document rather than failing it: a status page that is down when
// something is down is useless. llama and h may be nil.
func BuildPlatformStatus(ctx context.Context, c *models.Client, llama *LlamaModel, h *Harness, routingLimit int) models.PlatformStatus {
	s := models.PlatformStatus{Time: time.Now().UTC().Format(time.RFC3339), Daemon: []models.ModelStatus{}, Routing: []models.RoutingEvent{}, Stats: map[string]models.RouteStats{}}
	if c != nil {
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		rows, err := c.Registry(cctx)
		cancel()
		if err != nil {
			s.DaemonError = clip(err.Error())
		}
		for _, r := range rows {
			r.Detail = nil // internal engine detail, not part of the contract
			r.Error = clip(r.Error)
			s.Daemon = append(s.Daemon, r)
		}
	}
	if llama != nil {
		s.Llama = llama.Status(ctx)
	}
	if h != nil {
		s.Routing = h.RoutingEvents(routingLimit)
		s.Stats = h.RouteStats()
	}
	return s
}

package launcher

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

func replicationHealth(ctx context.Context, socket string) map[string]any {
	out := map[string]any{"available": false}
	transport := unixSocketTransport(socket)
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/oracle/v3/status", nil)
	if err != nil {
		return out
	}
	resp, err := client.Do(req)
	if err != nil {
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out
	}
	var status struct {
		Mode       string   `json:"mode"`
		HeadSeq    *uint64  `json:"head_seq"`
		LagSeq     *uint64  `json:"lag_seq"`
		LagSeconds *int64   `json:"lag_seconds"`
		Diverged   *bool    `json:"diverged"`
		Paused     *bool    `json:"replication_paused"`
		Unverified []string `json:"unverified_classes"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 65536)).Decode(&status) != nil || status.Mode == "" || status.HeadSeq == nil || status.LagSeq == nil || status.LagSeconds == nil || status.Diverged == nil || status.Paused == nil {
		return out
	}
	return map[string]any{"available": true, "mode": status.Mode, "head_seq": *status.HeadSeq, "lag_seq": *status.LagSeq, "lag_seconds": *status.LagSeconds, "diverged": *status.Diverged, "paused": *status.Paused, "unverified_classes": status.Unverified}
}

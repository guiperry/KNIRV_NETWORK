package knirvoracle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ReplicaStatus is the local Oracle's verified replication position.
type ReplicaStatus struct {
	Mode              string   `json:"mode"`
	HeadSeq           uint64   `json:"head_seq"`
	LagSeq            uint64   `json:"lag_seq"`
	LagSeconds        int64    `json:"lag_seconds"`
	Diverged          bool     `json:"diverged"`
	Paused            bool     `json:"replication_paused"`
	UnverifiedClasses []string `json:"unverified_classes"`
}

func (c *Client) ReplicaStatus(ctx context.Context) (ReplicaStatus, error) {
	var result ReplicaStatus
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/oracle/v3/status", nil)
	if err != nil {
		return result, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("oracle status: %s", resp.Status)
	}
	var wire struct {
		ReplicaStatus
		Mode       *string `json:"mode"`
		Paused     *bool   `json:"replication_paused"`
		Diverged   *bool   `json:"diverged"`
		Head       *uint64 `json:"head_seq"`
		Lag        *uint64 `json:"lag_seq"`
		LagSeconds *int64  `json:"lag_seconds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return result, err
	}
	if wire.Mode == nil || wire.Paused == nil || wire.Diverged == nil || wire.Head == nil || wire.Lag == nil || wire.LagSeconds == nil {
		return result, fmt.Errorf("oracle replica status incomplete")
	}
	result = wire.ReplicaStatus
	result.Mode, result.Paused, result.Diverged = *wire.Mode, *wire.Paused, *wire.Diverged
	result.HeadSeq, result.LagSeq, result.LagSeconds = *wire.Head, *wire.Lag, *wire.LagSeconds
	return result, nil
}

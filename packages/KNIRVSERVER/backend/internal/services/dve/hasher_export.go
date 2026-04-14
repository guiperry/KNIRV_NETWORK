package dve

import (
	"context"
	"fmt"
	"time"

	hasherpb "github.com/knirvcorp/knirvserver/backend/internal/proto"
)

// HasherExporter sends DVE snapshots to a running KNIRVHASHER instance for
// dataset collection. Safe to disable; KNIRVSERVER operation is unaffected.
type HasherExporter struct {
	grpcClient hasherpb.HasherServiceClient
	enabled    bool // false until KNIRVHASHER is out of stealth
}

// NewHasherExporter creates a new HasherExporter instance.
func NewHasherExporter(client hasherpb.HasherServiceClient) *HasherExporter {
	return &HasherExporter{
		grpcClient: client,
		enabled:    false, // Start disabled (stealth mode)
	}
}

// Enable activates the exporter when KNIRVHASHER is ready.
func (he *HasherExporter) Enable() {
	he.enabled = true
}

// Disable deactivates the exporter (stealth mode).
func (he *HasherExporter) Disable() {
	he.enabled = false
}

// ExportOnboardingData streams user onboarding data to KNIRVHASHER.
// This is a fire-and-forget operation that doesn't block KNIRVSERVER.
func (he *HasherExporter) ExportOnboardingData(orgID, userID string) {
	if !he.enabled {
		return // stealth mode — no-op
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := he.grpcClient.ExportSecurityData(ctx, &hasherpb.ExportRequest{
			OrgId:     orgID,
			UserId:    userID,
			DataType:  hasherpb.DataType_ALL,
			Encrypted: true,
		})
		if err != nil {
			// Log error but don't fail - this is advisory
			fmt.Printf("HasherExporter: failed to export data for user %s: %v\n", userID, err)
		}
	}()
}

// ExportGuardrailViolation exports guardrail violation data for training.
func (he *HasherExporter) ExportGuardrailViolation(orgID, userID string) {
	if !he.enabled {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := he.grpcClient.ExportSecurityData(ctx, &hasherpb.ExportRequest{
			OrgId:     orgID,
			UserId:    userID,
			DataType:  hasherpb.DataType_GUARDRAILS,
			Encrypted: true,
		})
		if err != nil {
			fmt.Printf("HasherExporter: failed to export violation data for user %s: %v\n", userID, err)
		}
	}()
}

// IsEnabled returns whether the exporter is active.
func (he *HasherExporter) IsEnabled() bool {
	return he.enabled
}

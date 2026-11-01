package drq

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFitKindManifest(t *testing.T, fitKind string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	doc := map[string]any{
		"$schema": "https://ulora.org/schema/v1/manifest.json",
		"training": map[string]any{
			"fit_kind": fitKind,
		},
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateBundleRejectsLinearEmbeddingFit(t *testing.T) {
	v := NewDVEBundleValidator("http://validation.invalid", "token")
	result := &ULoRACompileResult{
		ContentHash:  "abc123",
		ManifestPath: writeFitKindManifest(t, "linear_embedding_fit"),
	}
	err := v.ValidateBundle(context.Background(), &ErrorCluster{ClusterID: "c1"}, result, nil)
	if err == nil {
		t.Fatal("linear embedding fit bundle was accepted")
	}
	if !strings.Contains(err.Error(), "fit_kind linear_embedding_fit cannot be published") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateBundleAllowsGradientSFTFitKindPastThisGate(t *testing.T) {
	v := NewDVEBundleValidator("http://validation.invalid", "token")
	result := &ULoRACompileResult{
		ContentHash:  "abc123",
		ManifestPath: writeFitKindManifest(t, "gradient_sft"),
	}
	err := v.ValidateBundle(context.Background(), &ErrorCluster{ClusterID: "c1"}, result, nil)
	// The bundle passes the honesty gate and then fails at submit because the
	// endpoint is unreachable; the important part is that it is not rejected as
	// an unfit artifact.
	if err == nil {
		t.Fatal("expected submission to fail against an unreachable endpoint")
	}
	if strings.Contains(err.Error(), "cannot be published") {
		t.Fatalf("gradient_sft was rejected by the fit-kind gate: %v", err)
	}
}

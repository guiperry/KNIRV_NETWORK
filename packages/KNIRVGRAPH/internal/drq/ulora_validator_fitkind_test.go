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

// Fail closed (PI_Q40): every case where the fit kind cannot be established must
// refuse the bundle with an error naming the reason, never pass it through.

func writeRawManifest(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckPublishableFitKindFailsClosed(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "does-not-exist.json")
	cases := []struct {
		name     string
		path     string
		contains string
	}{
		{"missing manifest path", "", "has no manifest path"},
		{"whitespace-only manifest path", "   ", "has no manifest path"},
		{"unreadable manifest file", missingPath, "cannot read manifest"},
		{"invalid JSON", writeRawManifest(t, `{"training": `), "is not valid JSON"},
		{"missing training object", writeRawManifest(t, `{"$schema": "x"}`), "declares no training.fit_kind"},
		{"empty fit_kind", writeFitKindManifest(t, ""), "declares no training.fit_kind"},
		{"unknown fit_kind", writeFitKindManifest(t, "made_up_fit"), `fit_kind "made_up_fit" is not a publishable fit kind`},
		{"linear_embedding_fit", writeFitKindManifest(t, "linear_embedding_fit"), "fit_kind linear_embedding_fit cannot be published"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkPublishableFitKind(tc.path)
			if err == nil {
				t.Fatalf("bundle was accepted; want refusal containing %q", tc.contains)
			}
			if !strings.Contains(err.Error(), tc.contains) {
				t.Fatalf("error %q does not name the reason %q", err.Error(), tc.contains)
			}
		})
	}
}

func TestCheckPublishableFitKindPassesGradientSFT(t *testing.T) {
	if err := checkPublishableFitKind(writeFitKindManifest(t, "gradient_sft")); err != nil {
		t.Fatalf("gradient_sft manifest refused: %v", err)
	}
}

func TestValidateBundleRefusesMissingManifestPath(t *testing.T) {
	v := NewDVEBundleValidator("http://validation.invalid", "token")
	result := &ULoRACompileResult{ContentHash: "abc123"}
	err := v.ValidateBundle(context.Background(), &ErrorCluster{ClusterID: "c1"}, result, nil)
	if err == nil {
		t.Fatal("bundle with no manifest path was accepted")
	}
	if !strings.Contains(err.Error(), "has no manifest path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

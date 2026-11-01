package bundling

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ulora/internal/api"
)

func TestFitKindManifestRoundTrip(t *testing.T) {
	cases := []string{api.FitKindGradientSFT, api.FitKindLinearEmbeddingFit, ""}
	for _, kind := range cases {
		manifest := api.Manifest{
			Schema:  api.SchemaURL,
			Name:    "round-trip",
			Version: "1.0.0",
			Training: api.Training{
				FitKind: kind,
			},
		}
		data, err := json.Marshal(manifest)
		if err != nil {
			t.Fatalf("marshal manifest with fit kind %q: %v", kind, err)
		}
		got, err := FitKindOfManifestJSON(data)
		if err != nil {
			t.Fatalf("read fit kind %q: %v", kind, err)
		}
		if got != kind {
			t.Fatalf("fit kind round trip = %q, want %q", got, kind)
		}
	}
}

func TestFitKindOfBundleDirectory(t *testing.T) {
	dir := t.TempDir()
	manifest := api.Manifest{
		Schema:   api.SchemaURL,
		Name:     "bundle",
		Version:  "1.0.0",
		Training: api.Training{FitKind: api.FitKindLinearEmbeddingFit},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := FitKindOfBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != api.FitKindLinearEmbeddingFit {
		t.Fatalf("fit kind = %q", got)
	}
}

func TestPublishableFitKind(t *testing.T) {
	if err := PublishableFitKind(api.FitKindLinearEmbeddingFit); err == nil {
		t.Fatal("linear embedding fit must not be publishable")
	} else if !strings.Contains(err.Error(), "cannot be published") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := PublishableFitKind(api.FitKindGradientSFT); err != nil {
		t.Fatalf("gradient_sft must be publishable: %v", err)
	}
	if err := PublishableFitKind(""); err != nil {
		t.Fatalf("unknown kind is left to the caller: %v", err)
	}
}

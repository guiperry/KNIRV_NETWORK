package bundling

import (
	"encoding/json"
	"fmt"
	"os"

	"ulora/internal/api"
)

// FitKindOfManifestJSON extracts training.fit_kind from a manifest document.
// An absent field returns "" so callers can distinguish "unknown" from a kind
// they deliberately refuse.
func FitKindOfManifestJSON(data []byte) (string, error) {
	var doc struct {
		Training struct {
			FitKind string `json:"fit_kind"`
		} `json:"training"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("decode manifest: %w", err)
	}
	return doc.Training.FitKind, nil
}

// FitKindOfManifestFile reads training.fit_kind from a manifest file on disk.
func FitKindOfManifestFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read manifest: %w", err)
	}
	return FitKindOfManifestJSON(data)
}

// FitKindOfBundle reads training.fit_kind from the manifest inside a bundle
// archive or bundle directory.
func FitKindOfBundle(bundleOrDirPath string) (string, error) {
	data, err := ReadFile(bundleOrDirPath, ManifestFile)
	if err != nil {
		return "", err
	}
	return FitKindOfManifestJSON(data)
}

// PublishableFitKind refuses any fit kind that is not real gradient training.
// It is the single definition of the honesty gate, shared by the compiler, the
// daemon and the chain-side validator.
func PublishableFitKind(kind string) error {
	if kind == api.FitKindLinearEmbeddingFit {
		return fmt.Errorf("fit_kind %s cannot be published: it is a linear embedding fit, not trained LoRA", kind)
	}
	return nil
}

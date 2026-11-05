package models

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Migration flags (spec §10.1). Centralised here so every daemon package reads
// the same defaults; the legacy layout stays the default until Phase 5.

const (
	// ModelLayout selects the route/package layout. "legacy" keeps HEAD as the
	// decision surface at /models/head; "migrated" re-cuts the roles (HEART
	// decides, HEAD becomes the VLM + mirrored text generator).
	ModelLayoutLegacy   = "legacy"
	ModelLayoutMigrated = "migrated"

	// HandsEnabledEnv gates whether /models/hands/* is mounted.
	HandsEnabledEnv = "KNIRV_HANDS_ENABLED"

	// VLM backend values for HEAD. "none" (default) makes every HEAD route
	// return a structured 503 with no fabricated output.
	VLMBackendNone   = "none"
	VLMBackendLLAMA  = "llama"
	VLMBackendONNX   = "onnx"
	VLMBackendOpenAI = "openai"

	// HeartBackend selects HEART's generation backend. "gpt" is accepted as an
	// alias for "hands" for one release (spec §3.3).
	HeartBackendQwen  = "qwen"
	HeartBackendHands = "hands"
	HeartBackendGPT   = "gpt"
	HeartBackendExt   = "external"
)

// legacyLayoutSupported reports whether the legacy model layout can still be
// served. The legacy layout depended on the `pkg/head` (Go) and `head/` (Rust)
// packages, both removed in Phase 5; with those gone there is nothing legacy to
// mount, so legacy is unsupported. This is the single switch point that enforces
// the plan rule "legacy still works only if the old files are present".
func legacyLayoutSupported() bool { return false }

// ModelLayout returns the active layout, defaulting to migrated. The legacy
// layout is only honoured when explicitly requested via KNIRV_MODEL_LAYOUT=legacy
// AND legacyLayoutSupported() — since the legacy-only files were removed, an
// explicit legacy request silently degrades to migrated rather than mounting a
// route that no longer exists.
func ModelLayout() string {
	if v := trimSpace(os.Getenv("KNIRV_MODEL_LAYOUT")); v == ModelLayoutLegacy && legacyLayoutSupported() {
		return ModelLayoutLegacy
	}
	return ModelLayoutMigrated
}

// IsMigrated reports whether the migrated layout is active.
func IsMigrated() bool { return ModelLayout() == ModelLayoutMigrated }

// HandsEnabled reports whether HANDS routes should be mounted.
func HandsEnabled() bool {
	return trimSpace(os.Getenv(HandsEnabledEnv)) == "1"
}

// VLMBackend returns HEAD's configured vision backend, defaulting to none.
func VLMBackend() string {
	if v := trimSpace(os.Getenv("KNIRV_VLM_BACKEND")); v != "" {
		return v
	}
	return VLMBackendNone
}

// HeartBackend returns HEART's configured generation backend, defaulting to qwen.
func HeartBackend() string {
	if v := trimSpace(os.Getenv("KNIRV_HEART_BACKEND")); v != "" {
		return v
	}
	return HeartBackendQwen
}

// HeartBackendIsHands reports whether HEART's backend is the HANDS GPT (or the
// legacy "gpt" alias).
func HeartBackendIsHands() bool {
	switch HeartBackend() {
	case HeartBackendHands, HeartBackendGPT:
		return true
	}
	return false
}

// MirroredGeneration reports whether KNIRVINFERENCER runs dual HEART+HEAD mirrored
// generation and HASHER weighting (spec §10.1).
func MirroredGeneration() bool {
	return trimSpace(os.Getenv("KNIRV_MIRRORED_GENERATION")) == "1"
}

// MirroredWeighting returns the arbitration weighting mode.
func MirroredWeighting() string {
	if v := trimSpace(os.Getenv("KNIRV_MIRRORED_WEIGHTING")); v != "" {
		return v
	}
	return "hasher"
}

// MirroredTimeoutMS returns the per-side mirrored-generation timeout.
func MirroredTimeoutMS() int {
	if v := trimSpace(os.Getenv("KNIRV_MIRRORED_TIMEOUT_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 30000
}

// MirrorTieMargin returns the score margin within which HEART wins ties.
func MirrorTieMargin() float64 {
	if v := trimSpace(os.Getenv("KNIRV_MIRROR_TIE_MARGIN")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0.05
}

// HeartDecisionDir returns the directory HEART's decision weights live in.
func HeartDecisionDir() string {
	if d := trimSpace(os.Getenv("KNIRV_HEART_DECISION_DIR")); d != "" {
		return d
	}
	if root := trimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); root != "" {
		return filepath.Join(root, "knirvmodels", "data", "heart", "decision")
	}
	return "/var/lib/knirvserver/knirvmodels/data/heart/decision"
}

// HandsDir returns the directory HANDS checkpoints live in.
func HandsDir() string {
	if d := trimSpace(os.Getenv("KNIRV_HANDS_DIR")); d != "" {
		return d
	}
	if root := trimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); root != "" {
		return filepath.Join(root, "knirvmodels", "data", "hands")
	}
	return "/var/lib/knirvserver/knirvmodels/data/hands"
}

// VLMDir returns the directory HEAD's VLM weights/adapters live in.
func VLMDir() string {
	if d := trimSpace(os.Getenv("KNIRV_VLM_DIR")); d != "" {
		return d
	}
	if root := trimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); root != "" {
		return filepath.Join(root, "knirvmodels", "data", "vlm")
	}
	return "/var/lib/knirvserver/knirvmodels/data/vlm"
}

// HeartHandsFallbackQwen reports whether the Qwen hands.assistant fallback stays
// on while HANDS is untrained (spec §10.1).
func HeartHandsFallbackQwen() bool {
	v := trimSpace(os.Getenv("KNIRV_HANDS_FALLBACK_QWEN"))
	if v == "" {
		return true
	}
	return v == "1"
}

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}

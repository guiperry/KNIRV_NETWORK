// Package compat holds one-release shims for the HRM -> HEAD and
// KNIRVHASHER -> KNIRVMODELS renames (models_untangling.md).
package compat

import (
	"log"
	"os"
	"strings"
)

// PromoteLegacyEnv copies each KNIRV_HRM_<X> variable to KNIRV_HEAD_<X> when the
// new name is unset, so every reader can use only the new name. A warning names
// each deprecated variable that is still in use.
func PromoteLegacyEnv() {
	for _, kv := range os.Environ() {
		name, val, ok := strings.Cut(kv, "=")
		if !ok || !strings.HasPrefix(name, "KNIRV_HRM_") {
			continue
		}
		newName := "KNIRV_HEAD_" + strings.TrimPrefix(name, "KNIRV_HRM_")
		if os.Getenv(newName) != "" {
			continue
		}
		log.Printf("Warning: %s is deprecated; use %s", name, newName)
		_ = os.Setenv(newName, val)
	}
}

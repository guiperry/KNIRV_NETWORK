package controller

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"knirvmodels/internal/cli/embedded"
)

// fakeLookup returns a path for each name in have and an error for the rest, and
// records the names it was asked for, in order.
func fakeLookup(have map[string]string, asked *[]string) func(string) (string, error) {
	return func(name string) (string, error) {
		*asked = append(*asked, name)
		if p, ok := have[name]; ok {
			return p, nil
		}
		return "", errors.New(name + " not embedded")
	}
}

func TestTrainerLookupPrefersCurrentName(t *testing.T) {
	var asked []string
	lookup := fakeLookup(map[string]string{
		"heart-decision-trainer": "/bin/heart-decision-trainer",
		"head-trainer":           "/bin/head-trainer",
	}, &asked)
	got, err := trainerBinaryPath(lookup)
	if err != nil || got != "/bin/heart-decision-trainer" {
		t.Fatalf("got %q, %v; want the current name", got, err)
	}
	if len(asked) != 1 || asked[0] != "heart-decision-trainer" {
		t.Fatalf("asked %v; the legacy name must not be tried when the current one exists", asked)
	}
}

func TestTrainerLookupFallsBackToLegacyName(t *testing.T) {
	var asked []string
	lookup := fakeLookup(map[string]string{"head-trainer": "/bin/head-trainer"}, &asked)
	got, err := trainerBinaryPath(lookup)
	if err != nil || got != "/bin/head-trainer" {
		t.Fatalf("got %q, %v; want the legacy binary", got, err)
	}
	if len(asked) != 2 || asked[0] != "heart-decision-trainer" || asked[1] != "head-trainer" {
		t.Fatalf("asked %v; want current then legacy", asked)
	}
}

func TestTrainerLookupErrorNamesCurrentTrainer(t *testing.T) {
	var asked []string
	_, err := trainerBinaryPath(fakeLookup(map[string]string{}, &asked))
	if err == nil {
		t.Fatal("expected an error when neither name is embedded")
	}
	if got := err.Error(); got != "heart-decision-trainer not embedded" {
		t.Fatalf("error %q does not name the current trainer", got)
	}
}

// TestTrainerLookupRealEmbeddedGetBinaryPath runs the trainer lookup through the real
// embedded.GetBinaryPath, not a fake. KNIRV_APP_DATA_DIR points at a temp dir, so the
// extraction never touches the real app data directory.
func TestTrainerLookupRealEmbeddedGetBinaryPath(t *testing.T) {
	if !embedded.IsEmbedded(trainerName) {
		t.Skipf("%s is not embedded in this build; run make build-heart-decision-trainer first", trainerName)
	}
	appData := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", appData)

	got, err := trainerBinaryPath(embedded.GetBinaryPath)
	if err != nil {
		t.Fatalf("trainerBinaryPath(embedded.GetBinaryPath): %v", err)
	}
	wantDir := filepath.Join(appData, "bin")
	if !strings.HasPrefix(got, wantDir+string(os.PathSeparator)) {
		t.Fatalf("trainer path %q is not under %q", got, wantDir)
	}
	if filepath.Base(got) != trainerName {
		t.Fatalf("resolved %q; want the current trainer name %q", filepath.Base(got), trainerName)
	}
	st, err := os.Stat(got)
	if err != nil {
		t.Fatalf("extracted trainer missing: %v", err)
	}
	if st.Mode().Perm()&0111 == 0 {
		t.Fatalf("extracted trainer is not executable: mode %s", st.Mode())
	}
}

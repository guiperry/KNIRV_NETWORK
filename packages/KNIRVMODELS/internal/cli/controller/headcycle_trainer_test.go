package controller

import (
	"errors"
	"testing"
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

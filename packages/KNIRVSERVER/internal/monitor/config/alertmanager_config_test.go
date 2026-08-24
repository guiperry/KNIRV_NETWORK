package config

import (
	"testing"
)

func TestResolveEnvVars_Substitute(t *testing.T) {
	result, err := resolveVars("hello ${NAME}", func(s string) string {
		if s == "NAME" {
			return "world"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", result)
	}
}

func TestResolveEnvVars_DefaultValue(t *testing.T) {
	result, err := resolveVars("hello ${NAME:-friend}", func(s string) string {
		return ""
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello friend" {
		t.Fatalf("expected 'hello friend', got %q", result)
	}
}

func TestResolveEnvVars_MissingRequired(t *testing.T) {
	result, err := resolveVars("hello ${NAME}", func(s string) string {
		return ""
	})
	if err == nil {
		t.Fatal("expected error for missing required var")
	}
	if result != "hello ${NAME}" {
		t.Fatalf("expected 'hello ${NAME}', got %q", result)
	}
}

func TestResolveEnvVars_MultipleVars(t *testing.T) {
	result, err := resolveVars("${A}/${B:-default}", func(s string) string {
		if s == "A" {
			return "alpha"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "alpha/default" {
		t.Fatalf("expected 'alpha/default', got %q", result)
	}
}

func TestResolveEnvVars_NoPlaceholders(t *testing.T) {
	result, err := resolveVars("plain text without vars", func(s string) string { return "" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "plain text without vars" {
		t.Fatalf("expected input unchanged, got %q", result)
	}
}

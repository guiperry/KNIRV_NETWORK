package dvepod

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderHTMLTemplate(t *testing.T) {
	wasmB64 := "dGVzdC1kYXRh"
	dockURL := "http://localhost:8084"

	html := renderHTMLTemplate(wasmB64, dockURL)

	if !strings.Contains(html, "DVE Pod") {
		t.Error("HTML should contain DVE Pod title")
	}
	if !strings.Contains(html, wasmB64) {
		t.Error("HTML should contain base64 WASM data")
	}
	if !strings.Contains(html, dockURL) {
		t.Error("HTML should contain dock URL")
	}
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML should start with DOCTYPE")
	}
	if !strings.Contains(html, "</html>") {
		t.Error("HTML should end with closing html tag")
	}
}

func TestRenderHTMLTemplateNoDock(t *testing.T) {
	wasmB64 := "dGVzdC1kYXRh"
	html := renderHTMLTemplate(wasmB64, "")

	if strings.Contains(html, "localhost") {
		t.Error("HTML should not contain a dock URL when none provided")
	}
	if !strings.Contains(html, "null") {
		t.Error("DOCK_URL should be null when no dock URL provided")
	}
}

func TestRenderHTMLTemplateContainsJS(t *testing.T) {
	wasmB64 := "dGVzdC1kYXRh"
	html := renderHTMLTemplate(wasmB64, "")

	expectedElements := []string{
		"const WASM_B64",
		"terminal",
		"boot()",
		"help()",
		"ls(args)",
		"cat(args)",
		"dock(args)",
		"tee(args)",
		"agent(args)",
	}
	for _, el := range expectedElements {
		if !strings.Contains(html, el) {
			t.Errorf("HTML should contain JS function/constant: %s", el)
		}
	}
}

func TestRenderHTMLTemplateHasCSS(t *testing.T) {
	html := renderHTMLTemplate("dGVzdA==", "")

	cssSelectors := []string{
		"#terminal-wrap",
		".term-bar",
		".term-input-row",
		".spinner",
		":root",
		"@keyframes spin",
	}
	for _, sel := range cssSelectors {
		if !strings.Contains(html, sel) {
			t.Errorf("HTML should contain CSS selector: %s", sel)
		}
	}
}

func TestEmbeddedWASMHash(t *testing.T) {
	if !HasEmbeddedWASM() {
		t.Skip("dvepod.wasm not embedded — run 'make build/dvepod-wasm' first")
	}

	hash := EmbeddedWASMHash()
	if len(hash) != 64 {
		t.Errorf("expected 64 hex chars, got %d: %s", len(hash), hash)
	}

	size := EmbeddedWASMSize()
	if size <= 0 {
		t.Error("expected positive WASM size")
	}
}

func TestManagerInit(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}

	if mgr.runtimeName == "" {
		t.Error("runtime name should not be empty")
	}
	if mgr.dataDir == "" {
		t.Error("data dir should not be empty")
	}
}

func TestManagerSetDataDir(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	tmpDir := t.TempDir()
	mgr.SetDataDir(tmpDir)
	if mgr.dataDir != tmpDir {
		t.Errorf("expected dataDir %s, got %s", tmpDir, mgr.dataDir)
	}
}

func TestManagerListPodsEmpty(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	tmpDir := t.TempDir()
	mgr.SetDataDir(tmpDir)

	err = mgr.ListPods()
	if err != nil {
		t.Errorf("ListPods() on empty dir should not error: %v", err)
	}
}

func TestManagerExtractWASM(t *testing.T) {
	if !HasEmbeddedWASM() {
		t.Skip("dvepod.wasm not embedded — run 'make build/dvepod-wasm' first")
	}

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	tmpDir := t.TempDir()
	mgr.SetDataDir(tmpDir)

	wasmPath, err := mgr.extractWASM()
	if err != nil {
		t.Fatalf("extractWASM() failed: %v", err)
	}

	if _, err := os.Stat(wasmPath); os.IsNotExist(err) {
		t.Errorf("extracted WASM file not found at %s", wasmPath)
	}

	podsDir := filepath.Join(tmpDir, "pods")
	entries, err := os.ReadDir(podsDir)
	if err != nil {
		t.Fatalf("failed to read pods dir: %v", err)
	}

	metaFound := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			metaFound = true
			break
		}
	}
	if !metaFound {
		t.Error("expected a .json metadata file in pods dir")
	}
}

func TestManagerStatus(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	err = mgr.Status()
	if err != nil {
		t.Errorf("Status() should not error: %v", err)
	}
}

func TestManagerBundleWithoutWASM(t *testing.T) {
	if HasEmbeddedWASM() {
		t.Skip("WASM is embedded — can't test without it here")
	}

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test.html")
	err = mgr.Bundle(tmpFile, "")
	if err == nil {
		t.Error("expected error when WASM is not embedded")
	}
}

func TestBase64Encode(t *testing.T) {
	cases := []struct {
		input    []byte
		expected string
	}{
		{[]byte("hello"), "aGVsbG8="},
		{[]byte(""), ""},
		{[]byte("f"), "Zg=="},
		{[]byte("fo"), "Zm8="},
		{[]byte("foo"), "Zm9v"},
	}

	for _, c := range cases {
		result := base64Encode(c.input)
		if result != c.expected {
			t.Errorf("base64Encode(%q) = %q, want %q", string(c.input), result, c.expected)
		}
	}
}

func TestBase64EncodeDecode(t *testing.T) {
	input := []byte("Hello, DVE Pod! This is a test.")
	encoded := base64Encode(input)
	if encoded == "" {
		t.Fatal("base64Encode returned empty string")
	}

	for _, ch := range encoded {
		if ch != '=' && !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '+' || ch == '/') {
			t.Errorf("invalid base64 character: %c", ch)
		}
	}
}

func TestManagerDockNoEmbed(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	if !HasEmbeddedWASM() {
		err = mgr.Dock(nil, "http://localhost:8084", "")
		if err == nil {
			t.Error("expected error when WASM is not embedded")
		}
	}
}

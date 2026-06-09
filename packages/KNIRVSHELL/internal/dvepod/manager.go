package dvepod

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Manager struct {
	runtimePath string
	runtimeName string
	dataDir     string
}

func defaultDataDir() string {
	usr, err := user.Current()
	if err != nil {
		return filepath.Join(os.TempDir(), "knirv", "dvepod")
	}
	return filepath.Join(usr.HomeDir, ".local", "share", "knirv", "dvepod")
}

func NewManager() (*Manager, error) {
	m := &Manager{dataDir: defaultDataDir()}

	for _, name := range []string{"wasmer", "wasmtime", "wasmedge"} {
		if p, err := exec.LookPath(name); err == nil {
			m.runtimePath = p
			m.runtimeName = name
			return m, nil
		}
	}

	managed := filepath.Join(m.dataDir, "runtime", "wasmer")
	if _, err := os.Stat(managed); err == nil {
		m.runtimePath = managed
		m.runtimeName = "wasmer"
		return m, nil
	}

	fmt.Println("knirv: WASM runtime not found — installing Wasmer...")
	if err := m.installWasmer(); err != nil {
		return nil, fmt.Errorf("failed to install WASM runtime: %w", err)
	}
	m.runtimePath = managed
	m.runtimeName = "wasmer"
	fmt.Println("knirv: Wasmer installed at", managed)
	return m, nil
}

func (m *Manager) SetDataDir(dir string) {
	if dir != "" {
		m.dataDir = dir
	}
}

func (m *Manager) extractWASM() (string, error) {
	if !HasEmbeddedWASM() {
		return "", fmt.Errorf("dvepod.wasm is not embedded in this binary — rebuild with 'tinygo build -target=wasi -o dvepod.wasm ./cmd/dvepod/'")
	}

	podsDir := filepath.Join(m.dataDir, "pods")
	if err := os.MkdirAll(podsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create pods directory: %w", err)
	}

	podID := "dvepod-" + hex.EncodeToString(sha256.New().Sum(nil)[:4])
	wasmPath := filepath.Join(podsDir, podID+".wasm")

	if err := os.WriteFile(wasmPath, embeddedWASM, 0644); err != nil {
		return "", fmt.Errorf("failed to extract WASM: %w", err)
	}

	meta := map[string]interface{}{
		"pod_id":      podID,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
		"wasm_hash":   EmbeddedWASMHash(),
		"wasm_size":   EmbeddedWASMSize(),
		"runtime":     m.runtimeName,
	}
	metaPath := filepath.Join(podsDir, podID+".json")
	metaData, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(metaPath, metaData, 0644)

	return wasmPath, nil
}

func (m *Manager) New(ctx context.Context) error {
	wasmPath, err := m.extractWASM()
	if err != nil {
		return err
	}
	return m.runInPTY(ctx, wasmPath, nil)
}

func (m *Manager) Run(ctx context.Context, podID string) error {
	if podID == "" {
		podsDir := filepath.Join(m.dataDir, "pods")
		entries, err := os.ReadDir(podsDir)
		if err != nil {
			return fmt.Errorf("no pods found and no pod-id specified: %w", err)
		}
		var wasmFiles []string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".wasm") {
				wasmFiles = append(wasmFiles, e.Name())
			}
		}
		if len(wasmFiles) == 0 {
			return fmt.Errorf("no pods found — run 'knirv dve pod new' first")
		}
		podID = strings.TrimSuffix(wasmFiles[len(wasmFiles)-1], ".wasm")
	}

	wasmPath := filepath.Join(m.dataDir, "pods", podID+".wasm")
	if _, err := os.Stat(wasmPath); os.IsNotExist(err) {
		return fmt.Errorf("pod %s not found at %s", podID, wasmPath)
	}

	return m.runInPTY(ctx, wasmPath, nil)
}

func (m *Manager) Bundle(outPath, dockURL string) error {
	if !HasEmbeddedWASM() {
		return fmt.Errorf("dvepod.wasm is not embedded in this binary")
	}

	wasmB64 := base64Encode(embeddedWASM)
	html := renderHTMLTemplate(wasmB64, dockURL)

	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write bundle: %w", err)
	}

	fmt.Printf("knirv: DVE Pod bundle written to %s (%d bytes)\n", outPath, EmbeddedWASMSize())
	return nil
}

func (m *Manager) Dock(ctx context.Context, server, token string) error {
	attestation := map[string]interface{}{
		"node_id":   EmbeddedWASMHash()[:16],
		"tee_type":  "wasmer",
		"wasm_hash": EmbeddedWASMHash(),
		"timestamp": time.Now().Unix(),
		"version":   "dvepod/1.0",
	}

	if token != "" {
		attestation["auth_token"] = token
	}

	body, _ := json.Marshal(map[string]interface{}{
		"attestation": attestation,
	})

	url := strings.TrimRight(server, "/") + "/api/dve/pod/register"
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to dock to %s: %w", server, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("dock failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		SessionID string `json:"session_id"`
		WSURL     string `json:"ws_url"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("knirv: docked to %s\n", server)
	fmt.Printf("  Session ID: %s\n", result.SessionID)
	if result.WSURL != "" {
		fmt.Printf("  WebSocket:  %s\n", result.WSURL)
	}
	return nil
}

func (m *Manager) ListPods() error {
	podsDir := filepath.Join(m.dataDir, "pods")
	if err := os.MkdirAll(podsDir, 0755); err != nil {
		return fmt.Errorf("failed to access pods directory: %w", err)
	}

	entries, err := os.ReadDir(podsDir)
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	type podInfo struct {
		ID        string `json:"id"`
		CreatedAt string `json:"created_at"`
		WasmHash  string `json:"wasm_hash"`
		Runtime   string `json:"runtime"`
	}

	var pods []podInfo
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			data, err := os.ReadFile(filepath.Join(podsDir, e.Name()))
			if err != nil {
				continue
			}
			var p podInfo
			if err := json.Unmarshal(data, &p); err != nil {
				continue
			}
			p.ID = strings.TrimSuffix(e.Name(), ".json")
			pods = append(pods, p)
		}
	}

	if len(pods) == 0 {
		fmt.Println("No DVE Pods found.")
		return nil
	}

	fmt.Printf("%-24s %-24s %-16s %s\n", "POD ID", "CREATED", "WASM HASH", "RUNTIME")
	fmt.Println(strings.Repeat("-", 80))
	for _, p := range pods {
		hash := p.WasmHash
		if len(hash) > 16 {
			hash = hash[:16]
		}
		created := p.CreatedAt
		if len(created) > 19 {
			created = created[:19]
		}
		fmt.Printf("%-24s %-24s %-16s %s\n", p.ID, created, hash, p.Runtime)
	}

	return nil
}

func (m *Manager) Status() error {
	fmt.Printf("DVE Pod Status\n")
	fmt.Printf("  Runtime:     %s (%s)\n", m.runtimeName, m.runtimePath)
	fmt.Printf("  Data Dir:    %s\n", m.dataDir)
	fmt.Printf("  WASM Hash:   %s\n", EmbeddedWASMHash())
	fmt.Printf("  WASM Size:   %d bytes\n", EmbeddedWASMSize())
	fmt.Printf("  GOOS:        %s\n", runtime.GOOS)
	fmt.Printf("  GOARCH:      %s\n", runtime.GOARCH)

	if HasEmbeddedWASM() {
		fmt.Printf("  Embedded:    \033[32m✓\033[0m\n")
	} else {
		fmt.Printf("  Embedded:    \033[33m✗ (rebuild with tinygo)\033[0m\n")
	}

	podsDir := filepath.Join(m.dataDir, "pods")
	if entries, err := os.ReadDir(podsDir); err == nil {
		podCount := 0
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".wasm") {
				podCount++
			}
		}
		fmt.Printf("  Local Pods:  %d\n", podCount)
	} else {
		fmt.Printf("  Local Pods:  0\n")
	}

	return nil
}

func (m *Manager) runInPTY(ctx context.Context, wasmPath string, extraArgs []string) error {
	var args []string
	switch m.runtimeName {
	case "wasmer":
		args = append([]string{"run", wasmPath, "--"}, extraArgs...)
	case "wasmtime":
		args = append([]string{wasmPath, "--"}, extraArgs...)
	case "wasmedge":
		args = append([]string{wasmPath}, extraArgs...)
	default:
		return fmt.Errorf("unknown runtime: %s", m.runtimeName)
	}

	cmd := exec.CommandContext(ctx, m.runtimePath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("WASM runtime error: %w", err)
	}
	return nil
}

func (m *Manager) installWasmer() error {
	dest := filepath.Join(m.dataDir, "runtime")
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create runtime directory: %w", err)
	}

	url := wasmerReleaseURL(runtime.GOOS, runtime.GOARCH)
	fmt.Printf("knirv: downloading Wasmer from %s\n", url)

	return downloadAndExtract(url, dest)
}

func wasmerReleaseURL(goos, goarch string) string {
	archMap := map[string]string{
		"amd64": "amd64",
		"arm64": "aarch64",
		"arm":   "armv7",
	}
	arch, ok := archMap[goarch]
	if !ok {
		arch = "amd64"
	}

	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("https://github.com/wasmerio/wasmer/releases/download/v5.0.0/wasmer-linux-%s.%s", arch, ext)
}

func downloadAndExtract(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed (HTTP %d)", resp.StatusCode)
	}

	tmpFile := filepath.Join(dest, "wasmer.tar.gz")
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to download: %w", err)
	}

	wasmerBin := filepath.Join(dest, "wasmer")
	if _, err := os.Stat(wasmerBin); err == nil {
		os.Remove(tmpFile)
		return nil
	}

	if err := os.Rename(tmpFile, wasmerBin); err != nil {
		return fmt.Errorf("failed to install wasmer: %w", err)
	}

	if err := os.Chmod(wasmerBin, 0755); err != nil {
		return fmt.Errorf("failed to make wasmer executable: %w", err)
	}

	os.Remove(tmpFile)
	return nil
}

func base64Encode(data []byte) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result strings.Builder
	for i := 0; i < len(data); i += 3 {
		var b [3]byte
		var n int
		for j := 0; j < 3 && i+j < len(data); j++ {
			b[j] = data[i+j]
			n = j + 1
		}
		val := uint(b[0])<<16 | uint(b[1])<<8 | uint(b[2])
		result.WriteByte(chars[(val>>18)&0x3F])
		result.WriteByte(chars[(val>>12)&0x3F])
		if n > 1 {
			result.WriteByte(chars[(val>>6)&0x3F])
		} else {
			result.WriteByte('=')
		}
		if n > 2 {
			result.WriteByte(chars[val&0x3F])
		} else {
			result.WriteByte('=')
		}
	}
	return result.String()
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"ulora/internal/api"
	"ulora/internal/config"
)

func unixHTTPClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
}

func callULORA(cfg *config.Config, method, route string, body io.Reader) (*http.Response, error) {
	client := unixHTTPClient(cfg.SocketPath)
	url := "http://unix" + route
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequestWithContext(context.Background(), method, url, body)
	} else {
		req, err = http.NewRequestWithContext(context.Background(), method, url, nil)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
	return client.Do(req)
}

func cmdValidateManifest(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ulora validate-manifest <manifest.json>")
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	manifest, err := api.ManifestFromJSON(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "INVALID: %v\n", err)
		return err
	}

	fmt.Printf("VALID: %s v%s\n", manifest.Name, manifest.Version)
	return nil
}

func cmdCompileCluster(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var req api.CompileRequest
	if len(args) > 0 {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("read request: %w", err)
		}
		if err := json.Unmarshal(data, &req); err != nil {
			return fmt.Errorf("parse request: %w", err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "error: compile-cluster requires a JSON request file")
		fmt.Fprintln(os.Stderr, "usage: ulora compile-cluster <request.json>")
		return fmt.Errorf("missing request file")
	}

	resp, err := callULORA(cfg, "POST", "/ulora/v1/compile-cluster", bytes.NewReader(mustJSON(req)))
	if err != nil {
		return fmt.Errorf("call ulorad: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("compile failed: %s", strings.TrimSpace(string(body)))
	}

	var result api.CompileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Bundle ID:    %s\n", result.BundleID)
	fmt.Printf("Content Hash: %s\n", result.ContentHash)
	fmt.Printf("Path Used:    %s\n", result.PathUsed)
	fmt.Printf("Targets:      %s\n", strings.Join(result.TargetModels, ", "))
	for _, p := range []struct{ name, path string }{
		{"Manifest", result.ManifestPath},
		{"Weights", result.WeightsPath},
		{"Anchors", result.AnchorsPath},
	} {
		if p.path != "" {
			fmt.Printf("%-12s %s\n", p.name, p.path)
		}
	}
	return nil
}

func cmdTransfer(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(args) < 2 {
		return fmt.Errorf("usage: ulora transfer <source.ulora> <target_request.json>")
	}

	sourceBundle := args[0]
	data, err := os.ReadFile(args[1])
	if err != nil {
		return fmt.Errorf("read target request: %w", err)
	}

	var req api.TransferRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("parse request: %w", err)
	}
	req.SourceBundlePath = sourceBundle

	resp, err := callULORA(cfg, "POST", "/ulora/v1/transfer", bytes.NewReader(mustJSON(req)))
	if err != nil {
		return fmt.Errorf("call ulorad: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("transfer failed: %s", strings.TrimSpace(string(body)))
	}

	var result api.TransferResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Printf("Bundle ID:    %s\n", result.BundleID)
	fmt.Printf("Content Hash: %s\n", result.ContentHash)
	fmt.Printf("Path Used:    %s\n", result.PathUsed)
	fmt.Printf("Targets:      %s\n", strings.Join(result.TargetModels, ", "))
	return nil
}

func cmdHealth(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	resp, err := callULORA(cfg, "GET", "/ulora/v1/health", nil)
	if err != nil {
		return fmt.Errorf("call ulorad: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var health api.HealthResponse
	_ = json.Unmarshal(body, &health)
	fmt.Printf("Status:    %s\n", health.Status)
	fmt.Printf("Version:   %s\n", health.Version)
	fmt.Printf("Timestamp: %d\n", health.Timestamp)
	return nil
}

func mustJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ulora <command> [args]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  validate-manifest <manifest.json>    Validate a ULoRA manifest")
		fmt.Fprintln(os.Stderr, "  compile-cluster <request.json>       Compile a .ulora bundle from a dataset corpus")
		fmt.Fprintln(os.Stderr, "  transfer <source.ulora> <request.json>  Transfer an existing bundle to new targets")
		fmt.Fprintln(os.Stderr, "  health                               Check ulorad health")
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "validate-manifest":
		err = cmdValidateManifest(args)
	case "compile-cluster", "compile":
		err = cmdCompileCluster(args)
	case "transfer":
		err = cmdTransfer(args)
	case "health":
		err = cmdHealth(args)
	case "help", "-h", "--help":
		fmt.Fprintln(os.Stderr, "ULoRA CLI - Universal LoRA Protocol")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  validate-manifest <manifest.json>    Validate a ULoRA manifest")
		fmt.Fprintln(os.Stderr, "  compile-cluster <request.json>       Compile a .ulora bundle")
		fmt.Fprintln(os.Stderr, "  transfer <source.ulora> <request.json>  Transfer bundle to new targets")
		fmt.Fprintf(os.Stderr, "  health                               Check ulorad health\n")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		fmt.Fprintln(os.Stderr, "Run 'ulora help' for usage.")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

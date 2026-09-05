// Package install discovers or provisions the llama.cpp server and a default model.
package install

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/guiperry/knirv/llama/internal/config"
)

type Options struct {
	DataDir, ServerPath, ModelPath, ModelURL, ModelName string
	NoInstall                                           bool
}

type Result struct{ ServerPath, ModelPath, ModelName string }

type Installer struct {
	Run func(context.Context, string, ...string) error
	Get func(string) (*http.Response, error)
}

func New() *Installer {
	return &Installer{
		Run: func(ctx context.Context, name string, args ...string) error {
			return exec.CommandContext(ctx, name, args...).Run()
		},
		Get: http.Get,
	}
}

func (i *Installer) Ensure(ctx context.Context, o Options) (Result, error) {
	if o.DataDir == "" {
		return Result{}, fmt.Errorf("data directory is required")
	}
	if o.ModelName == "" {
		o.ModelName = config.DefaultModelName
	}
	if o.ModelURL == "" {
		o.ModelURL = config.DefaultModelURL
	}

	serverPaths := append([]string{o.ServerPath, filepath.Join(o.DataDir, "llama.cpp", "build", "bin", "llama-server")}, legacyServerPaths()...)
	server := firstFile(serverPaths...)
	if server == "" {
		if o.NoInstall {
			return Result{}, fmt.Errorf("llama-server was not found and installation is disabled")
		}
		var err error
		server, err = i.installServer(ctx, o.DataDir)
		if err != nil {
			return Result{}, err
		}
	}
	modelPaths := []string{o.ModelPath, filepath.Join(o.DataDir, "models", o.ModelName)}
	// Legacy installations only contain the bundled TinyLlama model. Do not let
	// one override an explicitly selected model name or URL.
	if o.ModelName == config.DefaultModelName && o.ModelURL == config.DefaultModelURL {
		modelPaths = append(modelPaths, legacyModelPaths()...)
	}
	model := firstFile(modelPaths...)
	if model == "" {
		if o.NoInstall {
			return Result{}, fmt.Errorf("model was not found and installation is disabled")
		}
		var err error
		model, err = i.downloadModel(o.DataDir, o.ModelName, o.ModelURL)
		if err != nil {
			return Result{}, err
		}
	}
	return Result{ServerPath: server, ModelPath: model, ModelName: strings.TrimSuffix(filepath.Base(model), filepath.Ext(model))}, nil
}

func (i *Installer) installServer(ctx context.Context, dataDir string) (string, error) {
	dir := filepath.Join(dataDir, "llama.cpp")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := i.Run(ctx, "git", "clone", "--depth", "1", "https://github.com/ggml-org/llama.cpp.git", dir); err != nil {
			return "", fmt.Errorf("clone llama.cpp: %w", err)
		}
	}
	build := filepath.Join(dir, "build")
	if err := i.Run(ctx, "cmake", "-S", dir, "-B", build, "-DLLAMA_BUILD_SERVER=ON"); err != nil {
		return "", fmt.Errorf("configure llama.cpp: %w", err)
	}
	if err := i.Run(ctx, "cmake", "--build", build, "--target", "llama-server", "--config", "Release"); err != nil {
		return "", fmt.Errorf("build llama.cpp server: %w", err)
	}
	path := firstFile(filepath.Join(build, "bin", "llama-server"), filepath.Join(build, "bin", "Release", "llama-server"))
	if path == "" {
		return "", fmt.Errorf("llama.cpp build completed but llama-server was not produced")
	}
	return path, nil
}

func (i *Installer) downloadModel(dataDir, name, url string) (string, error) {
	dir := filepath.Join(dataDir, "models")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create models directory: %w", err)
	}
	path := filepath.Join(dir, name)
	tmp := path + ".part"
	resp, err := i.Get(url)
	if err != nil {
		return "", fmt.Errorf("download model: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download model: unexpected HTTP status %s", resp.Status)
	}
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		return "", fmt.Errorf("write model: %w", copyErr)
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", fmt.Errorf("finalize model download: %w", err)
	}
	return path, nil
}

func firstFile(paths ...string) string {
	for _, path := range paths {
		if path == "" {
			continue
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}
func legacyServerPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, "llama.cpp", "build", "bin", "llama-server"), filepath.Join(home, "models", "llama.cpp", "build", "bin", "llama-server")}
}
func legacyModelPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{filepath.Join(home, "models", "tinyllama-1.1b-chat-v1.0.Q4_0.gguf"), filepath.Join(home, "tinyllama-1.1b-chat-v1.0.Q4_0.gguf")}
}

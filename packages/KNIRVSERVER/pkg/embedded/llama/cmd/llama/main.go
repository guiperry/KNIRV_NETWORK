// llama provisions llama.cpp when needed and exposes local chat endpoints.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guiperry/knirv/llama/internal/config"
	"github.com/guiperry/knirv/llama/internal/httpapi"
	"github.com/guiperry/knirv/llama/internal/install"
	"github.com/guiperry/knirv/llama/internal/runtime"
)

func main() {
	var dataDir, listen, llamaAddress, serverPath, modelPath, modelURL string
	var noInstall bool
	var unixSocket string
	flag.StringVar(&dataDir, "data-dir", "", "directory for llama.cpp and models")
	flag.StringVar(&listen, "listen", "127.0.0.1:8080", "address for chat API")
	flag.StringVar(&unixSocket, "unix-socket", "", "path for chat API Unix socket (overrides -listen)")
	flag.StringVar(&llamaAddress, "llama-address", "127.0.0.1:8000", "llama-server address")
	flag.StringVar(&serverPath, "server-path", "", "path to llama-server")
	flag.StringVar(&modelPath, "model-path", "", "path to a GGUF model")
	flag.StringVar(&modelURL, "model-url", "", "URL for the model downloaded on first run")
	flag.BoolVar(&noInstall, "no-install", false, "fail instead of installing missing dependencies")
	flag.Parse()

	dataDir, configPath, err := config.Paths(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	if saved, err := config.Load(configPath); err == nil {
		if serverPath == "" {
			serverPath = saved.ServerPath
		}
		if modelPath == "" {
			modelPath = saved.ModelPath
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	result, err := install.New().Ensure(ctx, install.Options{DataDir: dataDir, ServerPath: serverPath, ModelPath: modelPath, ModelURL: modelURL, NoInstall: noInstall})
	if err != nil {
		log.Fatal(err)
	}
	if err := config.Save(configPath, config.Config{ServerPath: result.ServerPath, ModelPath: result.ModelPath, ModelName: result.ModelName}); err != nil {
		log.Fatal(err)
	}

	var child *runtime.Server
	if !runtime.Healthy(context.Background(), llamaAddress) {
		child, err = runtime.Start(result.ServerPath, result.ModelPath, llamaAddress)
		if err != nil {
			log.Fatal(err)
		}
		defer child.Stop()
		deadline := time.Now().Add(30 * time.Second)
		for !runtime.Healthy(context.Background(), llamaAddress) && time.Now().Before(deadline) {
			time.Sleep(250 * time.Millisecond)
		}
		if !runtime.Healthy(context.Background(), llamaAddress) {
			log.Fatal("llama-server did not become healthy")
		}
	}
	handler, err := httpapi.New(llamaAddress, result.ModelName)
	if err != nil {
		log.Fatal(err)
	}

	var listener net.Listener
	if unixSocket != "" {
		if err := os.RemoveAll(unixSocket); err != nil {
			log.Fatalf("failed to remove stale unix socket: %v", err)
		}
		listener, err = net.Listen("unix", unixSocket)
		if err != nil {
			log.Fatalf("failed to create unix socket listener: %v", err)
		}
	} else {
		listener, err = net.Listen("tcp", listen)
		if err != nil {
			log.Fatalf("failed to listen on %s: %v", listen, err)
		}
	}

	httpServer := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() { <-signals; _ = httpServer.Shutdown(context.Background()) }()

	if unixSocket != "" {
		fmt.Printf("llama chat API listening on unix socket %s (POST /v1/chat/completions)\n", unixSocket)
	} else {
		fmt.Printf("llama chat API listening on http://%s (POST /v1/chat/completions)\n", listen)
	}
	if err := httpServer.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

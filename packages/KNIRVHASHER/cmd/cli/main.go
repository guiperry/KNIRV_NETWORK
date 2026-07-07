// Hasher: Neural Inference Engine Powered by SHA-256 ASICs
// Copyright (C) 2026  Guillermo Perry
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"knirvhasher/internal/analyzer"
	"knirvhasher/internal/cli/embedded"
	"knirvhasher/internal/cli/headless"
	"knirvhasher/internal/cli/ui"

	tea "github.com/charmbracelet/bubbletea"
)

var pipelineState struct {
	Cmd     *exec.Cmd
	Running bool
	Mu      sync.Mutex
}

var (
	monitorLogs  = flag.Bool("monitor-logs", true, "enable server log monitoring")
	headlessMode = flag.Bool("headless", false, "run in headless mode with HTTP API server over Unix socket")
	socketPath   = flag.String("socket-path", "/var/run/knirvhasher.sock", "Unix socket path for headless API server")
	socketPerm   = flag.String("socket-perm", "0660", "Unix socket permissions (octal)")
	goatMode     = flag.Bool("goat", false, "MAPPER mode: use Hugging Face API as data source; passes -goat to data-miner and skips the data-connector stage")
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC: %v\n", r)
		}
	}()

	flag.Parse()

	initEmbeddedBinaries()

	if *headlessMode {
		runHeadlessMode(*goatMode)
		return
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	model := ui.NewModel()

	model.CheckExistingHasherHost()

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseAllMotion(), tea.WithInputTTY())

	go func() {
		<-sigChan
		fmt.Println("\nReceived shutdown signal.")
		cleanupASICDevice(model.Deployer)
		if model.ServerCmd != nil && model.ServerCmd.Process != nil {
			shutdownHasherHost(model.ServerCmd, true, 8080)
		}
		pipelineState.Mu.Lock()
		shutdownPipelineProcess(pipelineState.Cmd)
		pipelineState.Running = false
		pipelineState.Mu.Unlock()
		os.Exit(0)
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		cleanupASICDevice(model.Deployer)
		pipelineState.Mu.Lock()
		shutdownPipelineProcess(pipelineState.Cmd)
		pipelineState.Running = false
		pipelineState.Mu.Unlock()
		os.Exit(1)
	}

	cleanupASICDevice(model.Deployer)
	pipelineState.Mu.Lock()
	shutdownPipelineProcess(pipelineState.Cmd)
	pipelineState.Running = false
	pipelineState.Mu.Unlock()
}

func initEmbeddedBinaries() {
	binaries := embedded.CheckEmbeddedBinaries()
	embeddedCount := 0
	for _, b := range binaries {
		if b.Embedded {
			embeddedCount++
		}
	}

	if embeddedCount > 0 {
		if err := embedded.EnsureExtracted(); err != nil {
		}
	}
}

func cleanupASICDevice(deployer *analyzer.Deployer) {
	if deployer == nil {
		return
	}

	device := deployer.GetActiveDevice()
	if device == nil {
		return
	}

	fmt.Printf("Cleaning up binaries from ASIC device %s...\n", device.IPAddress)

	if err := deployer.Connect(); err != nil {
		fmt.Printf("Could not connect to device for cleanup: %v\n", err)
		return
	}
	defer deployer.Disconnect()

	if err := deployer.Cleanup(); err != nil {
		fmt.Printf("Cleanup warning: %v\n", err)
	} else {
		fmt.Println("ASIC device cleanup complete.")
	}
}

func shutdownHasherHost(cmd *exec.Cmd, started bool, port int) {
	if !started {
		return
	}

	fmt.Println("Shutting down hasher-host...")

	if cmd != nil && cmd.Process != nil {
		shutdownURL := fmt.Sprintf("http://localhost:%d/api/v1/shutdown", port)
		http.Post(shutdownURL, "application/json", nil)

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			fmt.Println("hasher-host shut down successfully.")
		case <-time.After(15 * time.Second):
			fmt.Println("hasher-host shutdown timeout, force killing...")
			cmd.Process.Kill()
		}
	} else if port > 0 {
		shutdownURL := fmt.Sprintf("http://localhost:%d/api/v1/shutdown", port)
		resp, err := http.Post(shutdownURL, "application/json", nil)
		if err == nil {
			resp.Body.Close()
			fmt.Println("Shutdown request sent to pre-existing hasher-host.")
		}
	}
}

func shutdownPipelineProcess(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		fmt.Println("Shutting down pipeline process...")

		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			fmt.Println("Pipeline process SIGTERM failed, force killing...")
			cmd.Process.Kill()
		} else {
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()

			select {
			case <-done:
				fmt.Println("Pipeline process shut down successfully.")
				return
			case <-time.After(5 * time.Second):
				fmt.Println("Pipeline process shutdown timeout, force killing...")
				cmd.Process.Kill()
			}
		}
	}

	pipelineBinaries := []string{"data-miner", "data-encoder", "data-trainer"}
	for _, bin := range pipelineBinaries {
		exec.Command("pkill", "-TERM", "-f", bin).Run()
		time.Sleep(100 * time.Millisecond)
		exec.Command("pkill", "-KILL", "-f", bin).Run()
	}
}

func runHeadlessMode(goat bool) {
	perm, err := strconv.ParseUint(*socketPerm, 8, 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid socket permissions '%s': %v\n", *socketPerm, err)
		os.Exit(1)
	}

	fmt.Printf("Starting headless mode with socket: %s (perm: %s)\n", *socketPath, *socketPerm)

	if err := headless.RunHeadless(*socketPath, uint32(perm), goat); err != nil {
		fmt.Fprintf(os.Stderr, "Headless mode error: %v\n", err)
		os.Exit(1)
	}
}

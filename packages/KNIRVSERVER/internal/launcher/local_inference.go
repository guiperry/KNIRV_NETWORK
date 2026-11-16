package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"knirv-server/internal/vlmserver"
)

// Local inference start order (on by default; -no-llama turns all of it off):
//
//  1. backend_server starts the model harness (knirvmodels --inference). llama.cpp is the
//     primary and starts first. Colibri is the backup: it starts only while llama.cpp is
//     down or out for good, and is released once llama.cpp has been stable again.
//  2. The MiniCPM-V server (HEAD's vision backend) is started by vlmserver once the
//     harness has settled: llama.cpp serving, or Colibri serving as the backup. Its memory plan is then made against the RAM that is
//     actually free, not the RAM that was free before the text models loaded.
//  3. Model files are downloaded in parallel with step 1 (no RAM or GPU is used by a
//     download), so the server can start the moment the gate opens.

// harnessSettled reports whether the local text harness has made its decision. llama.cpp
// is the primary and Colibri the backup. The gate opens once something is serving, or once
// both are out for good. It reads /harness/status on the inference socket.
func harnessSettled(client *http.Client, base string) (bool, string) {
	resp, err := client.Get(base + "/harness/status")
	if err != nil {
		return false, "the model harness is not answering yet"
	}
	defer resp.Body.Close()
	var doc struct {
		PrimaryState string `json:"primary_state"`
		ColibriState string `json:"colibri_state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return false, "unreadable harness status"
	}
	switch {
	case doc.PrimaryState == "serving":
		return true, "llama.cpp serving"
	case doc.ColibriState == "serving":
		return true, "colibri serving as backup"
	case doc.PrimaryState == "unavailable" && (doc.ColibriState == "unavailable" || doc.ColibriState == "stopped" || doc.ColibriState == ""):
		return true, "no local model can serve on this host"
	default:
		return false, fmt.Sprintf("llama.cpp is %s, colibri is %s", doc.PrimaryState, doc.ColibriState)
	}
}

// harnessGate returns a gate that waits on the harness socket until it has settled or ctx
// ends. It logs the wait once per state change, not every poll.
func harnessGate(socket string) vlmserver.Gate {
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socket)
			},
		},
	}
	return func(ctx context.Context) bool {
		last := ""
		for {
			ok, why := harnessSettled(client, "http://harness")
			if ok {
				log.Printf("[vlm] local text harness settled (%s); starting the MiniCPM server", why)
				return true
			}
			if why != last {
				log.Printf("[vlm] waiting for the local text harness: %s", why)
				last = why
			}
			select {
			case <-ctx.Done():
				return false
			case <-time.After(5 * time.Second):
			}
		}
	}
}

// startLocalInference prepares the MiniCPM server and exports the variables HEAD needs.
// It must run before backend_server is spawned, since children inherit the environment.
// Nothing here blocks: the supervisor runs in the background and never fails startup.
func (app *ServerApp) startLocalInference() {
	if !app.config.AutoStartLlama {
		log.Println("Local inference disabled (-no-llama): model harness and MiniCPM server not started")
		return
	}
	appData := mustAppDataDir()
	socket := filepath.Join(appData, "sockets", "llama.sock")

	cfg := vlmserver.Config{
		DataDir:  filepath.Join(appData, "vlm"),
		ModelDir: strings.TrimSpace(os.Getenv("KNIRV_VLM_MODEL_DIR")),
		Gate:     harnessGate(socket),
		Logf: func(format string, args ...any) {
			log.Printf("[vlm] "+format, args...)
		},
	}
	if p := strings.TrimSpace(os.Getenv("KNIRV_VLM_MINICPM_PORT")); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			cfg.Port = n
		}
	}
	sup := vlmserver.New(cfg)
	for _, kv := range sup.Env() {
		k, v, _ := strings.Cut(kv, "=")
		if err := os.Setenv(k, v); err != nil {
			log.Printf("Warning: could not export %s: %v", k, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	app.vlm, app.vlmCancel = sup, cancel
	go func() {
		if err := sup.Run(ctx); err != nil && ctx.Err() == nil {
			log.Printf("[vlm] MiniCPM server stopped: %v", err)
		}
	}()
	log.Printf("Local inference: Colibri first, llama.cpp on failure; MiniCPM server at %s once the harness settles", sup.Endpoint())
}

// stopLocalInference stops the MiniCPM server and its llama-server process group.
func (app *ServerApp) stopLocalInference() {
	if app.vlmCancel != nil {
		app.vlmCancel()
	}
	if app.vlm != nil {
		app.vlm.Stop()
	}
}

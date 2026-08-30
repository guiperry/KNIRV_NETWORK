package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/apache/arrow/go/v15/arrow/flight"
	"github.com/knirvcorp/knirvbase/internal/monitoring"
	"github.com/knirvcorp/knirvbase/pkg/knirvbase"
	"github.com/knirvcorp/knirvbase/pkg/nrv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"net"
)

type appendRequest struct {
	Domain  string `json:"domain"`
	Bracket string `json:"bracket"`
}
type bracketResponse struct {
	Bracket string `json:"bracket"`
}

func main() {
	addr := flag.String("addr", envOr("KNIRVBASE_ADDR", ":50052"), "HTTP client API address")
	dataDir := flag.String("data-dir", defaultDataDir(), "database data directory")
	networkID := flag.String("network-id", envOr("KNIRVBASE_NETWORK_ID", "knirvbase-default"), "distributed network ID")
	bootstrap := flag.String("bootstrap-peers", envOr("KNIRVBASE_BOOTSTRAP_PEERS", ""), "comma-separated bootstrap peers")
	flightAddr := flag.String("flight-addr", envOr("KNIRVBASE_FLIGHT_ADDR", ":8815"), "Arrow Flight address (reserved for Flight adapter)")
	flag.Parse()
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatal(err)
	}
	peers := []string{}
	if strings.TrimSpace(*bootstrap) != "" {
		peers = strings.Split(*bootstrap, ",")
	}
	db, err := knirvbase.NewNRV(context.Background(), knirvbase.Options{DataDir: *dataDir, DistributedEnabled: true, DistributedNetworkID: *networkID, DistributedBootstrapPeers: peers}, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Shutdown()
	flightListener, err := net.Listen("tcp", *flightAddr)
	if err != nil {
		log.Fatal(err)
	}
	flightGRPC := grpc.NewServer()
	flight.RegisterFlightServiceServer(flightGRPC, newFlightAdapter(db))
	go func() {
		if err := flightGRPC.Serve(flightListener); err != nil {
			log.Printf("Arrow Flight stopped: %v", err)
		}
	}()
	// metrics registers into the default Prometheus registry, the same one
	// promhttp.Handler() below serves — instantiating it here is what makes
	// the knirvbase_* counters/gauges actually appear on /metrics instead of
	// being dead code exercised only by monitoring_test.go. Only the metrics
	// observable from this HTTP handler layer are wired (store/retrieve ops,
	// errors, active connections); blocks_committed/cache_hits/cache_misses/
	// nrn_balance/index_size require instrumentation inside pkg/knirvbase's
	// dataset/collection internals and are intentionally left unwired here —
	// see the network_monitor metrics_alignment.md plan for that follow-up.
	metrics := monitoring.NewMetrics()

	instrumented := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			metrics.ActiveConnections.Inc()
			defer metrics.ActiveConnections.Dec()
			next(w, r)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"status": "ok", "network_id": *networkID, "flight_addr": *flightAddr})
	})
	// NetworkStats are deliberately JSON rather than Prometheus-only: the
	// embedded KNIRVMONITOR needs the coherent peer/traffic snapshot, not a
	// lossy reconstruction from counters.
	mux.HandleFunc("/network/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := db.NetworkStats(*networkID)
		if stats == nil {
			http.Error(w, "network not found", http.StatusNotFound)
			return
		}
		writeJSON(w, map[string]any{
			"network_id":          stats.NetworkID,
			"connected_peers":     stats.ConnectedPeers,
			"total_peers":         stats.TotalPeers,
			"collections_shared":  stats.CollectionsShared,
			"operations_sent":     stats.OperationsSent,
			"operations_received": stats.OperationsReceived,
			"bytes_transferred":   stats.BytesTransferred,
			"average_latency_ms":  stats.AverageLatency.Milliseconds(),
		})
	})
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/append", instrumented(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		var req appendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Domain == "" || req.Bracket == "" {
			http.Error(w, "domain and bracket are required", 400)
			return
		}
		raw, err := base64.StdEncoding.DecodeString(req.Bracket)
		if err != nil || len(raw) != nrv.BracketSize {
			http.Error(w, "bracket must be 80-byte base64", 400)
			return
		}
		var encoded [nrv.BracketSize]byte
		copy(encoded[:], raw)
		b := nrv.DecodeBracket(encoded)
		if err := db.Dataset(req.Domain).AppendBracket(r.Context(), &b, nrv.ThermoAtmosphere{}); err != nil {
			metrics.ErrorCount.Inc()
			http.Error(w, err.Error(), 500)
			return
		}
		metrics.MemoryStoreOps.Inc()
		writeJSON(w, map[string]any{"ok": true})
	}))
	mux.HandleFunc("/document", instrumented(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Collection string         `json:"collection"`
			Document   map[string]any `json:"document"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Collection == "" || req.Document == nil {
			http.Error(w, "collection and document are required", http.StatusBadRequest)
			return
		}
		stored, err := db.Collection(req.Collection).Insert(r.Context(), req.Document)
		if err != nil {
			metrics.ErrorCount.Inc()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		metrics.MemoryStoreOps.Inc()
		writeJSON(w, stored)
	}))
	mux.HandleFunc("/stream", instrumented(func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("domain")
		if domain == "" {
			http.Error(w, "domain is required", 400)
			return
		}
		ch, err := db.Dataset(domain).StreamBrackets(r.Context(), false)
		if err != nil {
			metrics.ErrorCount.Inc()
			http.Error(w, err.Error(), 500)
			return
		}
		out := []bracketResponse{}
		for b := range ch {
			raw := nrv.EncodeBracket(b)
			out = append(out, bracketResponse{Bracket: base64.StdEncoding.EncodeToString(raw[:])})
			metrics.MemoryRetrieveOps.Inc()
		}
		writeJSON(w, out)
	}))
	log.Printf("KNIRVBASE listening on %s (network=%s, flight=%s)", *addr, *networkID, *flightAddr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func defaultDataDir() string {
	if v := os.Getenv("KNIRVBASE_DATA_DIR"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "knirvbase")
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/routes"
	"go.uber.org/zap"
)

var (
	socketPath = flag.String("socket", "/var/run/knirv/oracle.sock", "Unix socket path")
	dataDir    = flag.String("data-dir", "", "Oracle data directory")
	configFile = flag.String("config", "", "Config file path")
)

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	flag.Parse()

	fmt.Printf("KNIRVORACLE v%s (built %s, commit %s)\n", Version, BuildTime, GitCommit)

	if *configFile != "" {
		log.Printf("Config file specified: %s (loading from file not yet implemented)", *configFile)
	}
	if *dataDir != "" {
		if err := os.Setenv("ORACLE_DATA_DIR", *dataDir); err != nil {
			log.Fatalf("Failed to set ORACLE_DATA_DIR: %v", err)
		}
	}

	resolvedSocketPath, err := resolveSocketPath(*socketPath)
	if err != nil {
		log.Fatalf("Failed to resolve oracle socket path: %v", err)
	}
	*socketPath = resolvedSocketPath

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	oracleInst, err := initOracleFromKeyFile(logger)
	if err != nil {
		log.Fatalf("Failed to initialize Oracle from root.key: %v", err)
	}
	if oracleInst == nil {
		log.Println("Oracle not activated: no usable root.key found")
		return
	}

	socketDir := filepath.Dir(*socketPath)
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		log.Fatalf("Failed to create socket directory: %v", err)
	}

	if err := os.Remove(*socketPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to remove existing socket: %v", err)
	}

	listener, err := net.Listen("unix", *socketPath)
	if err != nil {
		log.Fatalf("Failed to create socket listener: %v", err)
	}
	if err := os.Chmod(*socketPath, 0600); err != nil {
		log.Printf("Warning: failed to set socket permissions: %v", err)
	}

	log.Printf("Oracle listening on %s", *socketPath)

	mux := http.NewServeMux()
	oracleRoutes := routes.NewOracleRoutes(oracleInst, logger)
	oracleRoutes.RegisterRoutes(mux)
	mux.HandleFunc("/health", handleHealth(oracleInst))

	server := &http.Server{Handler: mux}

	if err := oracleInst.Start(); err != nil {
		log.Fatalf("Failed to start Oracle: %v", err)
	}

	log.Println("Oracle services started")

	if err := updateCloudflareDNS(logger); err != nil {
		log.Printf("Warning: failed to update Cloudflare DNS: %v", err)
	} else {
		log.Println("Cloudflare DNS updated for oracle.knirv.network")
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down Oracle...")

	if err := oracleInst.Stop(); err != nil {
		log.Printf("Error stopping Oracle: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	listener.Close()
}

func resolveSocketPath(currentFlag string) (string, error) {
	if currentFlag != "" && currentFlag != "/var/run/knirv/oracle.sock" {
		return currentFlag, nil
	}

	if envPath := os.Getenv("ORACLE_SOCKET_PATH"); envPath != "" {
		return envPath, nil
	}

	appDataDir, err := getOSAppDataDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(appDataDir, "sockets", "oracle.sock"), nil
}

func handleHealth(o *oracle.Oracle) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","oracle":"active","version":"` + Version + `"}`))
	}
}

type cloudflareResponse struct {
	Success bool        `json:"success"`
	Errors  interface{} `json:"errors"`
	Result  interface{} `json:"result"`
}

func updateCloudflareDNS(logger *zap.Logger) error {
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	if apiToken == "" {
		return fmt.Errorf("CLOUDFLARE_API_TOKEN not set")
	}

	zoneID := os.Getenv("CLOUDFLARE_ZONE_ID")
	zoneName := os.Getenv("CLOUDFLARE_ZONE_NAME")

	if zoneID == "" && zoneName == "" {
		zoneName = "knirv.network"
	}

	recordName := os.Getenv("CLOUDFLARE_RECORD_NAME")
	if recordName == "" {
		recordName = "oracle"
	}

	ip, err := getOutboundIP()
	if err != nil {
		return fmt.Errorf("failed to get outbound IP: %w", err)
	}

	if zoneID == "" && zoneName != "" {
		zoneID, err = getZoneID(apiToken, zoneName)
		if err != nil {
			return fmt.Errorf("failed to get zone ID from zone name: %w", err)
		}
		log.Printf("Resolved zone ID %s from zone name %s", zoneID, zoneName)
	}

	log.Printf("Updating Cloudflare DNS: %s -> %s", recordName, ip)

	recordID, _ := getRecordID(apiToken, zoneID, recordName, "A")

	if recordID != "" {
		if _, err := updateDNSRecord(apiToken, zoneID, recordID, "A", recordName, ip, 60, false); err != nil {
			return fmt.Errorf("failed to update DNS record: %w", err)
		}
		log.Printf("Updated DNS record %s", recordName)
	} else {
		fullRecordName := recordName
		if zoneName != "" {
			fullRecordName = recordName + "." + zoneName
		}
		if _, err := createDNSRecord(apiToken, zoneID, "A", fullRecordName, ip, 60, false); err != nil {
			return fmt.Errorf("failed to create DNS record: %w", err)
		}
		log.Printf("Created DNS record %s", fullRecordName)
	}

	return nil
}

func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func getZoneID(apiToken, zoneName string) (string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", zoneName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Result  []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.Success || len(result.Result) == 0 {
		return "", fmt.Errorf("zone not found: %s", zoneName)
	}
	return result.Result[0].ID, nil
}

func getRecordID(apiToken, zoneID, recordName, recordType string) (string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?type=%s&name=%s", zoneID, recordType, recordName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Result  []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.Success || len(result.Result) == 0 {
		return "", fmt.Errorf("record not found")
	}
	return result.Result[0].ID, nil
}

func updateDNSRecord(apiToken, zoneID, recordID, recordType, recordName, content string, ttl int, proxied bool) (*cloudflareResponse, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)

	data := map[string]interface{}{
		"type":    recordType,
		"name":    recordName,
		"content": content,
		"ttl":     ttl,
		"proxied": proxied,
	}
	body, _ := json.Marshal(data)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result cloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func createDNSRecord(apiToken, zoneID, recordType, recordName, content string, ttl int, proxied bool) (*cloudflareResponse, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", zoneID)

	data := map[string]interface{}{
		"type":    recordType,
		"name":    recordName,
		"content": content,
		"ttl":     ttl,
		"proxied": proxied,
	}
	body, _ := json.Marshal(data)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result cloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

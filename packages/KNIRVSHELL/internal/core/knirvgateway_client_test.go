package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestKNIRVGatewayClientNetworkEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/balance/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if err := json.NewEncoder(w).Encode(map[string]string{
			"balance": "12345",
		}); err != nil {
			t.Errorf("encode balance response: %v", err)
		}
	})
	mux.HandleFunc("/api/faucet/request", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": GatewayFaucetResponse{
				Success: true,
				Transaction: &GatewayFaucetTransaction{
					ID:        "tx-1",
					Amount:    "1000",
					Token:     "NRN",
					Recipient: "0x1234567890abcdef1234567890abcdef12345678",
					Network:   "public-testnet",
					Timestamp: time.Unix(123, 0).UTC(),
				},
				Message: "ok",
			},
		}); err != nil {
			t.Errorf("encode faucet response: %v", err)
		}
	})
	mux.HandleFunc("/api/faucet/status/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"canRequest": true,
				"timeLeft":   0,
				"hoursLeft":  0,
			},
		}); err != nil {
			t.Errorf("encode faucet status: %v", err)
		}
	})
	mux.HandleFunc("/api/economics/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"status":    "healthy",
				"timestamp": "current",
			},
		}); err != nil {
			t.Errorf("encode health response: %v", err)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewKNIRVGatewayClient(&config.ServiceConfig{
		URL:     server.URL,
		Timeout: 5 * time.Second,
		Retries: 0,
	}, logrus.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	balance, err := client.GetNRNBalance(ctx, "0x1234567890abcdef1234567890abcdef12345678")
	require.NoError(t, err)
	require.Equal(t, "12345", balance)

	faucet, err := client.RequestNRNFromFaucet(ctx, "0x1234567890abcdef1234567890abcdef12345678", "1000", "public-testnet")
	require.NoError(t, err)
	require.True(t, faucet.Success)
	require.NotNil(t, faucet.Transaction)
	require.Equal(t, "tx-1", faucet.Transaction.ID)

	status, err := client.CheckFaucetStatus(ctx, "0x1234567890abcdef1234567890abcdef12345678", "public-testnet")
	require.NoError(t, err)
	require.Equal(t, true, status["canRequest"])

	health, err := client.CheckFaucetHealth(ctx)
	require.NoError(t, err)
	require.Equal(t, "healthy", health["status"])
}

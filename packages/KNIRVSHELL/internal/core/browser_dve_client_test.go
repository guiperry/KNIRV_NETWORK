package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveBrowserDVEIdentity(t *testing.T) {
	identity := DeriveBrowserDVEIdentity("gno1testwallet", "extension-abc", "knirvshell/1.2.3")

	assert.Equal(t, "gno1testwallet", identity.WalletAddress)
	assert.Equal(t, "extension-abc", identity.ExtensionID)
	assert.Equal(t, "knirv://dve/gno1testwallet/browser", identity.DVEURI)
	assert.Equal(t, "dve-", identity.NodeID[:4])
	assert.Len(t, identity.NodeID, 20)
	assert.Equal(t, "knirvshell/1.2.3", identity.BrowserVersion)
}

func TestBrowserDVEClientConnectAndHeartbeat(t *testing.T) {
	registerReceived := make(chan BrowserDVEMessage, 1)
	heartbeatReceived := make(chan BrowserDVEMessage, 1)
	taskReceived := make(chan BrowserDVEMessage, 1)
	heartbeatAckReceived := make(chan BrowserDVEMessage, 1)

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dve/browser/ws" {
			http.NotFound(w, r)
			return
		}

		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("unexpected auth header: %q", got)
			return
		}
		if got := r.URL.Query().Get("wallet"); got != "gno1testwallet" {
			t.Errorf("unexpected wallet query param: %q", got)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("failed to upgrade websocket: %v", err)
			return
		}
		defer conn.Close()

		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("failed to read register message: %v", err)
			return
		}

		var register BrowserDVEMessage
		if err := json.Unmarshal(raw, &register); err != nil {
			t.Errorf("failed to unmarshal register message: %v", err)
			return
		}
		registerReceived <- register

		if err := conn.WriteJSON(BrowserDVEMessage{
			Type:    "task_assigned",
			Payload: json.RawMessage(`{"task_id":"task-1","status":"assigned"}`),
		}); err != nil {
			t.Errorf("failed to write task_assigned: %v", err)
			return
		}

		_, raw, err = conn.ReadMessage()
		if err != nil {
			t.Errorf("failed to read heartbeat message: %v", err)
			return
		}

		var heartbeat BrowserDVEMessage
		if err := json.Unmarshal(raw, &heartbeat); err != nil {
			t.Errorf("failed to unmarshal heartbeat message: %v", err)
			return
		}
		heartbeatReceived <- heartbeat

		if err := conn.WriteJSON(BrowserDVEMessage{
			Type:    "heartbeat_ack",
			Payload: json.RawMessage(`{"timestamp":123456789}`),
		}); err != nil {
			t.Errorf("failed to write heartbeat ack: %v", err)
			return
		}

		_, raw, err = conn.ReadMessage()
		if err != nil {
			return
		}

		var final BrowserDVEMessage
		if err := json.Unmarshal(raw, &final); err != nil {
			t.Errorf("failed to unmarshal final message: %v", err)
			return
		}
		if final.Type == "task_result" {
			taskReceived <- final
		}
	}))
	defer server.Close()

	identity := DeriveBrowserDVEIdentity("gno1testwallet", "extension-abc", "knirvshell/1.2.3")
	client := NewBrowserDVEClient(
		server.URL,
		"test-token",
		identity,
		WithBrowserDVEHeartbeatInterval(25*time.Millisecond),
		WithBrowserDVECapabilities([]string{"policy-check", "signature-verify"}),
		WithBrowserDVEBadgeNFTIDs([]string{"badge-1", "badge-2"}),
	)

	taskHandlerHit := make(chan struct{}, 1)
	ackHandlerHit := make(chan struct{}, 1)

	client.On("task_assigned", func(message BrowserDVEMessage) {
		taskReceived <- message
		taskHandlerHit <- struct{}{}
	})
	client.On("heartbeat_ack", func(message BrowserDVEMessage) {
		heartbeatAckReceived <- message
		ackHandlerHit <- struct{}{}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, client.Connect(ctx))
	defer client.Disconnect()

	select {
	case register := <-registerReceived:
		var payload map[string]any
		require.NoError(t, json.Unmarshal(register.Payload, &payload))
		assert.Equal(t, "ws_register", register.Type)
		assert.Equal(t, identity.NodeID, payload["node_id"])
		assert.Equal(t, identity.ExtensionID, payload["extension_id"])
		assert.Equal(t, identity.BrowserVersion, payload["browser_version"])
		assert.Equal(t, []any{"policy-check", "signature-verify"}, payload["capabilities"])
		assert.Equal(t, []any{"badge-1", "badge-2"}, payload["badge_nft_ids"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for register message")
	}

	select {
	case <-taskHandlerHit:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for task_assigned handler")
	}

	select {
	case heartbeat := <-heartbeatReceived:
		assert.Equal(t, "heartbeat", heartbeat.Type)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for heartbeat message")
	}

	select {
	case <-ackHandlerHit:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for heartbeat_ack handler")
	}

	select {
	case ack := <-heartbeatAckReceived:
		var payload map[string]any
		require.NoError(t, json.Unmarshal(ack.Payload, &payload))
		assert.Equal(t, float64(123456789), payload["timestamp"])
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for heartbeat_ack message")
	}

	assert.NoError(t, client.Disconnect())
}

func TestBuildDialTarget(t *testing.T) {
	identity := DeriveBrowserDVEIdentity("gno1abc", "extension-x", "")
	client := NewBrowserDVEClient("http://localhost:8084", "token", identity)

	dialURL, headers, err := client.buildDialTarget()
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(dialURL, "ws://localhost:8084/api/dve/browser/ws"))
	assert.Contains(t, dialURL, "wallet=gno1abc")
	assert.Equal(t, "Bearer token", headers.Get("Authorization"))
}

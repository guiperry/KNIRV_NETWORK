package channels

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestControllerChannelAppendsHashChainedEvidence(t *testing.T) {
	sessionDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(sessionDir, "session.json"), []byte(`{"session_id":"local-session"}`), 0o600))
	channel := &ControllerChannel{sessionID: "network-session", evidenceDir: sessionDir}

	require.NoError(t, channel.appendEvidence("chat.message.received", controllerChatFrame{
		ID: "frame-1", SessionID: "network-session", Sender: "mobile:phone-1",
		Content: "inspect the build", Timestamp: time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC),
		TrustLevel: "signed_supervised", SignatureVerified: true,
	}))
	require.NoError(t, channel.appendEvidence("chat.message.sent", controllerChatFrame{
		ID: "frame-2", SessionID: "network-session", Sender: "agent",
		Content: "build is healthy", Timestamp: time.Date(2026, 7, 25, 12, 0, 1, 0, time.UTC),
		TrustLevel: "locally_supervised",
	}))

	file, err := os.Open(filepath.Join(sessionDir, "events", "eventlog.jsonl"))
	require.NoError(t, err)
	defer file.Close()
	var events []controllerEvidenceEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event controllerEvidenceEvent
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &event))
		events = append(events, event)
	}
	require.NoError(t, scanner.Err())
	require.Len(t, events, 2)
	require.Equal(t, 0, events[0].Index)
	require.Equal(t, 1, events[1].Index)
	require.Empty(t, events[0].PrevHash)
	require.Equal(t, events[0].Hash, events[1].PrevHash)
	require.NotEmpty(t, events[1].Hash)

	hashes, err := os.ReadFile(filepath.Join(sessionDir, "events", "eventlog.hashchain"))
	require.NoError(t, err)
	require.Contains(t, string(hashes), events[0].Hash+"\n")
	require.Contains(t, string(hashes), events[1].Hash+"\n")
}

func TestControllerEvidenceRequiresSupervisedSession(t *testing.T) {
	channel := &ControllerChannel{evidenceDir: t.TempDir()}
	err := channel.appendEvidence("chat.message.received", controllerChatFrame{Content: "hello"})
	require.ErrorContains(t, err, "session evidence directory is unavailable")
}

func TestControllerEvidenceDirUsesCLIActiveSession(t *testing.T) {
	workspace := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(workspace, ".knirv"), 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(workspace, ".knirv", "active-session"),
		[]byte("session_local_1\n"),
		0o600,
	))
	t.Setenv("KNIRV_WORKSPACE", workspace)
	t.Setenv("KNIRV_DVE_SESSION_DIR", "")
	t.Setenv("KNIRV_SESSION_DIR", "")
	t.Setenv("KNIRV_EVIDENCE_SESSION_ID", "")

	require.Equal(t,
		filepath.Join(workspace, ".knirv", "sessions", "session_local_1"),
		controllerEvidenceDir("sess-network-1"),
	)
}

func TestControllerChannelUsesInternalSessionDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/dve/controller-session", r.URL.Path)
		require.Equal(t, "env-1", r.URL.Query().Get("environment_id"))
		require.Equal(t, "internal-secret", r.Header.Get("X-KNIRV-Agent-Token"))
		_ = json.NewEncoder(w).Encode(map[string]string{"session_id": "sess-1"})
	}))
	defer server.Close()

	channel := &ControllerChannel{
		dveID:         "env-1",
		serverURL:     server.URL,
		internalToken: "internal-secret",
		client:        server.Client(),
	}
	require.NoError(t, channel.resolveSession(t.Context()))
	require.Equal(t, "sess-1", channel.sessionID)
}

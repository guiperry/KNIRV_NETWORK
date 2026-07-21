package blockchain

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckpointStatusProvider(t *testing.T) {
	server := &BlockchainServer{}
	server.SetCheckpointStatusProvider(func() interface{} { return map[string]interface{}{"enabled": true, "last_end_height": 64} })
	recorder := httptest.NewRecorder()
	server.handleCheckpointStatus(recorder, httptest.NewRequest(http.MethodGet, "/checkpoint/status", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"last_end_height":64`) {
		t.Fatalf("unexpected response %d: %s", recorder.Code, recorder.Body.String())
	}
}

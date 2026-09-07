package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	ulorastore "KNIRVCHAIN/internal/ulora"
)

func TestHandleULoRABundleStreamsContentAddressedBytes(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("KNIRV_APP_DATA_DIR", dataDir)
	store, err := ulorastore.NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := store.Put(bytes.NewReader([]byte("bundle bytes")))
	if err != nil {
		t.Fatal(err)
	}

	api := &UnifiedAPI{}
	req := httptest.NewRequest(http.MethodGet, "/api/ulora/"+hash, nil)
	req = mux.SetURLVars(req, map[string]string{"hash": hash})
	res := httptest.NewRecorder()
	api.handleULoRABundle(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if got := res.Body.String(); got != "bundle bytes" {
		t.Fatalf("body = %q", got)
	}

}

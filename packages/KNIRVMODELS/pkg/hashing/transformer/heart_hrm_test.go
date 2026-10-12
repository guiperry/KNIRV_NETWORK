package transformer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"knirvmodels/pkg/hashing/hrm"
)

func hrmServer(t *testing.T, publish bool) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	const v = "v3-cafef00d"
	if publish {
		cfg, _ := os.ReadFile("../hrm/testdata/golden_config.json")
		var m map[string]any
		json.Unmarshal(cfg, &m)
		m["model_version"] = v
		cfg, _ = json.Marshal(m)
		w, _ := os.ReadFile("../hrm/testdata/golden.safetensors")
		os.WriteFile(filepath.Join(dir, "hrm_config.json"), cfg, 0o644)
		os.WriteFile(filepath.Join(dir, "model-"+v+".safetensors"), w, 0o644)
		os.Symlink("model-"+v+".safetensors", filepath.Join(dir, "model_latest"))
	}
	hs := &HEARTService{}
	svc := hrm.NewService(dir)
	hs.SetHRMService(svc)
	t.Cleanup(func() { svc.Close(context.Background()) })
	mux := http.NewServeMux()
	hs.registerHRM(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, v
}

func TestHeartHRMEndpoints(t *testing.T) {
	srv, v := hrmServer(t, true)

	res, _ := http.Get(srv.URL + "/heart/hrm/models")
	var m hrm.Manifest
	json.NewDecoder(res.Body).Decode(&m)
	if res.StatusCode != 200 || m.Version != v || len(m.SHA256) != 64 {
		t.Fatalf("models: %d %+v", res.StatusCode, m)
	}

	res, _ = http.Get(srv.URL + "/heart/hrm/weights/" + v)
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 || res.Header.Get("ETag") != `"`+m.SHA256+`"` || int64(len(body)) != m.Size {
		t.Fatalf("weights: %d etag=%s len=%d", res.StatusCode, res.Header.Get("ETag"), len(body))
	}
	req, _ := http.NewRequest("GET", srv.URL+"/heart/hrm/weights/"+v, nil)
	req.Header.Set("Range", "bytes=0-7")
	res, _ = http.DefaultClient.Do(req)
	part, _ := io.ReadAll(res.Body)
	if res.StatusCode != 206 || len(part) != 8 {
		t.Fatalf("range: %d len=%d", res.StatusCode, len(part))
	}
	for _, bad := range []string{"../../etc/passwd", "latest", "v1-zzzzzzzz"} {
		res, _ = http.Get(srv.URL + "/heart/hrm/weights/" + bad)
		if res.StatusCode != 404 {
			t.Errorf("weights/%s -> %d", bad, res.StatusCode)
		}
	}

	res, _ = http.Post(srv.URL+"/heart/hrm/reason", "application/json", strings.NewReader(`{"tokens":[3,1,4,1,5,9,2,6]}`))
	var r struct {
		ModelVersion string `json:"model_version"`
		Decisions    []any  `json:"decisions"`
	}
	json.NewDecoder(res.Body).Decode(&r)
	if res.StatusCode != 200 || r.ModelVersion != v || len(r.Decisions) != 2 {
		t.Fatalf("reason: %d %+v", res.StatusCode, r)
	}
	for body, want := range map[string]int{`{"tokens":[]}`: 400, `{"tokens":[99999]}`: 400, `not json`: 400} {
		res, _ = http.Post(srv.URL+"/heart/hrm/reason", "application/json", strings.NewReader(body))
		if res.StatusCode != want {
			t.Errorf("reason %q -> %d, want %d", body, res.StatusCode, want)
		}
	}
	res, _ = http.Get(srv.URL + "/heart/hrm/reason")
	if res.StatusCode != 405 {
		t.Errorf("GET reason -> %d", res.StatusCode)
	}
}

func TestHeartHRMWithoutModel(t *testing.T) {
	srv, _ := hrmServer(t, false)
	for _, c := range []struct{ method, path, body string }{
		{"GET", "/heart/hrm/models", ""}, {"POST", "/heart/hrm/reason", `{"tokens":[1]}`},
	} {
		req, _ := http.NewRequest(c.method, srv.URL+c.path, strings.NewReader(c.body))
		res, _ := http.DefaultClient.Do(req)
		if res.StatusCode != 503 {
			t.Errorf("%s %s -> %d, want 503", c.method, c.path, res.StatusCode)
		}
	}
}

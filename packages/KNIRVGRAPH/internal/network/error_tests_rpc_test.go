package network

import (
	"KNIRVGRAPH/internal/nrv"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type testKV struct{ m map[string][]byte }

func (k *testKV) Put(key, value []byte) error { k.m[string(key)] = value; return nil }
func (k *testKV) Get(key []byte) ([]byte, error) {
	if v, ok := k.m[string(key)]; ok {
		return v, nil
	}
	return nil, errors.New("not found")
}

func errorTestRouter() http.Handler {
	rpc := &RPCServer{logger: zap.NewNop()}
	rpc.SetErrorTestSuiteStore(nrv.NewErrorTestSuiteStore(&testKV{m: map[string][]byte{}}))
	r := mux.NewRouter()
	rpc.registerErrorTestRoutes(r)
	return r
}

func doJSON(t *testing.T, h http.Handler, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("X-KNIRV-Internal-Token", token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestErrorTestRoutesContributeSealGrade(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	h := errorTestRouter()

	tc := map[string]string{"id": "t1", "input": "in", "expected": "out", "author_id": "dev-1"}
	if rec := doJSON(t, h, "POST", "/nrv/errors/e1/tests", "", tc); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
	for i := 1; i <= nrv.ErrorTestSuiteSize; i++ {
		tc := map[string]string{"id": fmt.Sprintf("t%d", i), "input": fmt.Sprintf("in %d", i), "expected": fmt.Sprintf("out %d", i), "author_id": "dev-1"}
		if rec := doJSON(t, h, "POST", "/nrv/errors/e1/tests", "secret", tc); rec.Code != http.StatusCreated {
			t.Fatalf("contribution %d: %d %s", i, rec.Code, rec.Body.String())
		}
	}

	rec := doJSON(t, h, "GET", "/nrv/errors/e1/tests", "", nil)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "out 1") {
		t.Fatalf("public view must succeed and omit expected values: %d %s", rec.Code, rec.Body.String())
	}
	var pub nrv.PublicErrorTestSuite
	_ = json.Unmarshal(rec.Body.Bytes(), &pub)
	if pub.Status != nrv.SuiteSealed || pub.SuiteHash == "" {
		t.Fatalf("expected a sealed suite, got %+v", pub)
	}

	outputs := map[string]string{}
	for i := 1; i <= nrv.ErrorTestSuiteSize; i++ {
		outputs[fmt.Sprintf("t%d", i)] = fmt.Sprintf("out %d", i)
	}
	if rec := doJSON(t, h, "POST", "/nrv/errors/e1/tests/grade", "", map[string]interface{}{"outputs": outputs}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("grading must be internal-only, got %d", rec.Code)
	}
	rec = doJSON(t, h, "POST", "/nrv/errors/e1/tests/grade", "secret", map[string]interface{}{"suite_hash": "stale", "outputs": outputs})
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected a stale suite hash to conflict, got %d", rec.Code)
	}
	rec = doJSON(t, h, "POST", "/nrv/errors/e1/tests/grade", "secret", map[string]interface{}{"suite_hash": pub.SuiteHash, "outputs": outputs})
	var report nrv.GradeReport
	_ = json.Unmarshal(rec.Body.Bytes(), &report)
	if rec.Code != http.StatusOK || !report.AllPassed {
		t.Fatalf("expected all 8 to pass: %d %s", rec.Code, rec.Body.String())
	}
}

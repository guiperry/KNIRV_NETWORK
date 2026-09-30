package network

import (
	"KNIRVGRAPH/internal/nrv"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Error-node test suites (see nrv.ErrorTestSuiteSize).
//
//	GET  /nrv/errors/{errorID}/tests          public, redacted view (no expected values)
//	POST /nrv/errors/{errorID}/tests          contribute one KNIRVARENA-authored test   [internal]
//	PUT  /nrv/errors/{errorID}/tests          replace with a full 8-test version        [internal]
//	POST /nrv/errors/{errorID}/tests/grade    grade outputs against the sealed suite    [internal]
//
// Writes and grading are internal-only (X-KNIRV-Internal-Token, the shared
// KNIRV_INTERNAL_AUTH_TOKEN service key used across KNIRV's service-to-service
// calls). Contributions arrive via KNIRVSERVER on behalf of an authenticated
// KNIRVARENA user; grading is gated so the grader cannot be used as an
// oracle to brute-force expected values.

// SetErrorTestSuiteStore wires persistence for error-node test suites. Until
// it is set the routes answer 503.
func (rpc *RPCServer) SetErrorTestSuiteStore(store *nrv.ErrorTestSuiteStore) {
	rpc.errorTests = store
}

func (rpc *RPCServer) registerErrorTestRoutes(router *mux.Router) {
	router.HandleFunc("/nrv/errors/{errorID}/tests", rpc.getErrorTests).Methods("GET", "OPTIONS")
	router.HandleFunc("/nrv/errors/{errorID}/tests", rpc.contributeErrorTest).Methods("POST")
	router.HandleFunc("/nrv/errors/{errorID}/tests", rpc.replaceErrorTests).Methods("PUT")
	router.HandleFunc("/nrv/errors/{errorID}/tests/grade", rpc.gradeErrorTests).Methods("POST")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// requireInternalToken reports whether r carries the internal service token.
// With no token configured every internal call is refused rather than
// silently allowed.
func requireInternalToken(w http.ResponseWriter, r *http.Request) bool {
	expected := strings.TrimSpace(os.Getenv("KNIRV_INTERNAL_AUTH_TOKEN"))
	if expected == "" {
		writeJSONError(w, http.StatusServiceUnavailable, "internal auth token is not configured")
		return false
	}
	got := strings.TrimSpace(r.Header.Get("X-KNIRV-Internal-Token"))
	if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
		writeJSONError(w, http.StatusUnauthorized, "internal token required")
		return false
	}
	return true
}

func (rpc *RPCServer) errorTestStore(w http.ResponseWriter) *nrv.ErrorTestSuiteStore {
	if rpc.errorTests == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "error test suites are not configured on this node")
	}
	return rpc.errorTests
}

func (rpc *RPCServer) getErrorTests(w http.ResponseWriter, r *http.Request) {
	store := rpc.errorTestStore(w)
	if store == nil {
		return
	}
	suite, err := store.Get(mux.Vars(r)["errorID"])
	if errors.Is(err, nrv.ErrTestSuiteNotFound) {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, suite.Public())
}

func (rpc *RPCServer) contributeErrorTest(w http.ResponseWriter, r *http.Request) {
	if !requireInternalToken(w, r) {
		return
	}
	store := rpc.errorTestStore(w)
	if store == nil {
		return
	}
	var tc nrv.ErrorTestCase
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&tc); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid test case")
		return
	}
	errorID := mux.Vars(r)["errorID"]
	suite, err := store.ContributeTest(errorID, tc)
	switch {
	case errors.Is(err, nrv.ErrSuiteSealed):
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	rpc.logger.Info("Test contributed to error node",
		zap.String("error_id", errorID), zap.String("test_id", tc.ID),
		zap.String("author_id", tc.AuthorID), zap.Int("tests", len(suite.Tests)),
		zap.String("status", string(suite.Status)))
	writeJSON(w, http.StatusCreated, suite.Public())
}

func (rpc *RPCServer) replaceErrorTests(w http.ResponseWriter, r *http.Request) {
	if !requireInternalToken(w, r) {
		return
	}
	store := rpc.errorTestStore(w)
	if store == nil {
		return
	}
	var body struct {
		Tests []nrv.ErrorTestCase `json:"tests"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid suite")
		return
	}
	suite, err := store.Replace(mux.Vars(r)["errorID"], body.Tests)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, suite.Public())
}

func (rpc *RPCServer) gradeErrorTests(w http.ResponseWriter, r *http.Request) {
	if !requireInternalToken(w, r) {
		return
	}
	store := rpc.errorTestStore(w)
	if store == nil {
		return
	}
	var body struct {
		// SuiteHash, when set, pins grading to the suite version the outputs
		// were produced against; a mismatch means the suite was replaced
		// mid-attempt and the grade would be meaningless.
		SuiteHash string            `json:"suite_hash,omitempty"`
		Outputs   map[string]string `json:"outputs"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid grade request")
		return
	}
	suite, err := store.Get(mux.Vars(r)["errorID"])
	if errors.Is(err, nrv.ErrTestSuiteNotFound) {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if body.SuiteHash != "" && body.SuiteHash != suite.SuiteHash {
		writeJSONError(w, http.StatusConflict, "the error node's test suite changed since these outputs were produced")
		return
	}
	report, err := suite.GradeOutputs(body.Outputs)
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

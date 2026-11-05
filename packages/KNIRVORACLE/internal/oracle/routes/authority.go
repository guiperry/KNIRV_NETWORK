package routes

import (
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/guiperry/knirv-sdk-go/authority"
	"github.com/knirvcorp/knirvoracle/internal/oracle/registry"
)

// Authority records on the wire (AUTHORITY_SPEC section 2): byte fields are hex.
type wireAuthorityCert struct {
	SessionPub string `json:"session_pub"`
	Operator   string `json:"operator"`
	Epoch      uint64 `json:"epoch"`
	NotBefore  int64  `json:"not_before"`
	NotAfter   int64  `json:"not_after"`
	Scope      string `json:"scope"`
	Entropy    string `json:"entropy"`
	Signature  string `json:"sig_by_longterm"`
}

type wireAuthorityRecord struct {
	Tier                uint8             `json:"tier"`
	Domain              string            `json:"domain"`
	Scope               string            `json:"scope"`
	Operator            string            `json:"operator"`
	Epoch               uint64            `json:"epoch"`
	CanonicalActionHash string            `json:"canonical_action_hash"`
	PrevRecordHash      string            `json:"prev_record_hash"`
	Header              string            `json:"header"`
	Nonce               uint32            `json:"nonce"`
	Hash                string            `json:"hash"`
	TargetBits          uint32            `json:"target_bits"`
	WitnessKind         string            `json:"witness_kind"`
	DeviceClaim         string            `json:"device_claim,omitempty"`
	SessionCert         wireAuthorityCert `json:"session_cert"`
	Signature           string            `json:"signature"`
}

func hexN(field, s string, n int) ([]byte, error) {
	b, err := hex.DecodeString(s)
	if err != nil || (n > 0 && len(b) != n) {
		return nil, fmt.Errorf("%s must be %d bytes of hex", field, n)
	}
	return b, nil
}

// decodeAuthorityRecord turns the wire form into the SDK type. It rejects
// malformed hex and wrong lengths; whether the record is *valid* is Verify's job.
func decodeAuthorityRecord(w wireAuthorityRecord) (*authority.Record, error) {
	rec := &authority.Record{Tier: authority.Tier(w.Tier), Domain: w.Domain, Scope: w.Scope, Operator: w.Operator,
		Epoch: w.Epoch, Nonce: w.Nonce, TargetBits: w.TargetBits, WitnessKind: w.WitnessKind, DeviceClaim: w.DeviceClaim}
	for _, f := range []struct {
		name, val string
		dst       []byte
		n         int
	}{
		{"canonical_action_hash", w.CanonicalActionHash, rec.CanonicalActionHash[:], 32},
		{"prev_record_hash", w.PrevRecordHash, rec.PrevRecordHash[:], 32},
		{"header", w.Header, rec.Header[:], len(rec.Header)},
		{"hash", w.Hash, rec.Hash[:], 32},
	} {
		b, err := hexN(f.name, f.val, f.n)
		if err != nil {
			return nil, err
		}
		copy(f.dst, b)
	}
	var err error
	if rec.Signature, err = hexN("signature", w.Signature, 0); err != nil {
		return nil, err
	}
	c := w.SessionCert
	rec.SessionCert = authority.SessionCert{Operator: c.Operator, Epoch: c.Epoch, NotBefore: c.NotBefore,
		NotAfter: c.NotAfter, Scope: c.Scope, Entropy: c.Entropy}
	if rec.SessionCert.SessionPub, err = hexN("session_cert.session_pub", c.SessionPub, 0); err != nil {
		return nil, err
	}
	if rec.SessionCert.Signature, err = hexN("session_cert.sig_by_longterm", c.Signature, 0); err != nil {
		return nil, err
	}
	return rec, nil
}

// requireOperatorToken checks the explicitly configured operator token, the same
// one verifier registration uses. Both routes can change what the Oracle holds
// against an operator, so they share one gate (AUTHORITY_SPEC section 10, D14).
func requireOperatorToken(w http.ResponseWriter, req *http.Request) bool {
	token := strings.TrimSpace(os.Getenv("KNIRV_VERIFIER_REGISTRATION_TOKEN"))
	supplied := strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	if token == "" || subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) != 1 {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{"error": "requires the explicitly configured operator token"})
		return false
	}
	return true
}

// handleAuthorityReport is POST /oracle/v3/authority/report: hand the Oracle one
// record from `authority/<operator>` to check ("the record verifies and its scope
// is covered"). A proven violation slashes, queues evidence and holds approval.
func (r *OracleRoutes) handleAuthorityReport(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireOperatorToken(w, req) {
		return
	}
	var body struct {
		ChainID     string              `json:"chain_id"`
		Record      wireAuthorityRecord `json:"record"`
		ReportedPub string              `json:"reported_pub,omitempty"`
		AsOfUnix    int64               `json:"as_of_unix,omitempty"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 1<<20)).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid request"})
		return
	}
	rec, err := decodeAuthorityRecord(body.Record)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	var pub []byte
	if body.ReportedPub != "" {
		if pub, err = hexN("reported_pub", body.ReportedPub, 33); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
			return
		}
	}
	// D3: the verification time is supplied by the reporter; default to now.
	asOf := time.Now()
	if body.AsOfUnix != 0 {
		asOf = time.Unix(body.AsOfUnix, 0)
	}
	f, err := r.oracle.ReportAuthorityRecord(body.ChainID, rec, pub, asOf)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	out := map[string]interface{}{"verified": f.Verified, "chain_id": body.ChainID}
	if f.Err != nil {
		out["reason"] = f.Err.Error()
	}
	if f.Violation != nil {
		out["violation"] = f.Violation
	}
	respondJSON(w, http.StatusOK, out)
}

// enrollAuthorityOperator is the optional authority part of verifier registration.
func (r *OracleRoutes) enrollAuthorityOperator(id string, pub []byte, kind string, scopes []string) error {
	if kind != string(authority.OperatorRoot) && kind != string(authority.OperatorBoot) {
		return fmt.Errorf("authority_kind must be root or boot")
	}
	if len(scopes) == 0 {
		scopes = []string{id} // conservative: own address only
	}
	return r.oracle.SetAuthorityOperator(registry.OperatorEntry{Address: id, Kind: kind, LongTermPub: pub, Scopes: scopes})
}

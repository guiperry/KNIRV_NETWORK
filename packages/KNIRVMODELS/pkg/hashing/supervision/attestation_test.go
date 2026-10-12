package supervision

import "testing"

const hash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func testRequest() AttestationRequest {
	return AttestationRequest{ModelArtifactHash: hash, DatasetManifestHash: hash, PolicyBundleHash: hash, NonceEnd: 100000, Target: 1 << 30}
}

func TestAttestationBindsRequestAndDifficulty(t *testing.T) {
	a := NewAttestor(true)
	req := testRequest()
	r := a.ComputeAttestation(req)
	if !r.Attested || !a.VerifyAttestation(r.Record, req) {
		t.Fatal("valid attestation was rejected")
	}
	changed := req
	changed.PolicyBundleHash = "1123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if a.VerifyAttestation(r.Record, changed) {
		t.Fatal("different request was accepted")
	}
	r.Record.Nonce = req.NonceEnd + 1
	if a.VerifyAttestation(r.Record, req) {
		t.Fatal("out-of-range nonce was accepted")
	}
}

func TestAttestationRejectsNonHex(t *testing.T) {
	r := testRequest()
	r.ModelArtifactHash = "zz23456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if validateAttestationRequest(r) == nil {
		t.Fatal("non-hex hash accepted")
	}
}

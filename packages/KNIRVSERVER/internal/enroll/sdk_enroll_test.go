package enroll

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
)

// The production signer and verifier, end to end. The tests above drive ClientDrive
// with fakeSigner and a fake verifier, so they cannot notice bytes handed to the Rust
// core in the wrong form (the raw-envelope bug fixed in client.go). These tests sign
// with SDKSigner{} through the SDK's core and verify with SDKVerifier{}, which calls
// signing.VerifyBootnodeEnrollment, so the core checks the wire format and the
// approval binding.
//
// They need a core that reports refusal codes. Run them against the core pinned in
// go-package/internal/core/stamp.go once that pin is the new core; against an older
// core the code-specific assertions fail by design.

const (
	sdkChainID  = "knirv-testnet-1"
	sdkOperator = "operator:alice"
	sdkBootHex  = "1a2b3c4d5e6f7081928374655647382910fedcba0987654321fedcba09876543"
	sdkRootHex  = "ffffeeeeddddccccbbbbaaaa9999888877776666555544443333222211110000"
	sdkOtherHex = "2222222222222222222222222222222222222222222222222222222222222222"
)

func sdkKey(t *testing.T, h string) []byte {
	t.Helper()
	k, err := hex.DecodeString(h)
	if err != nil || len(k) != 32 {
		t.Fatalf("bad test key: %v", err)
	}
	return k
}

// sdkRootPublicKey is the root public key the server is configured with: the base64
// public key the core reports for the root key, as the SDK's vector test derives it.
func sdkRootPublicKey(t *testing.T, rootKey []byte) string {
	t.Helper()
	bootKey := sdkKey(t, sdkBootHex)
	const now = uint64(1_700_000_000)
	challenge, err := signing.SignBootnodeEnrollChallenge(bootKey, sdkChainID, "enroll:pubkey", now, now+600, sdkOperator, nil)
	if err != nil {
		t.Fatalf("sign probe challenge: %v", err)
	}
	approval, err := signing.SignBootnodeEnrollApproval(rootKey, sdkChainID, "enroll:pubkey", now, now+600,
		base64.StdEncoding.EncodeToString(challenge.Envelope), base64.StdEncoding.EncodeToString(challenge.PublicKey))
	if err != nil {
		t.Fatalf("sign probe approval: %v", err)
	}
	return base64.StdEncoding.EncodeToString(approval.PublicKey)
}

// newSDKServer serves an enrollment server that verifies with the real core.
func newSDKServer(t *testing.T, store signing.NonceStore, rootPK string) (*Server, *httptest.Server) {
	t.Helper()
	srv := NewServer(SDKVerifier{}, store, rootPK, sdkOperator, nil)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return srv, ts
}

// TestClientDriveSDKSignerVerifiesAgainstCore runs the production path: SDKSigner builds
// the challenge and approval, ClientDrive posts them in order, and the server accepts
// them only if the core verifies the approval over the base64 envelope and boot key.
func TestClientDriveSDKSignerVerifiesAgainstCore(t *testing.T) {
	bootKey := sdkKey(t, sdkBootHex)
	rootKey := sdkKey(t, sdkRootHex)
	rootPK := sdkRootPublicKey(t, rootKey)
	store := signing.NewMemoryNonceStore()
	_, ts := newSDKServer(t, store, rootPK)

	resp, err := ClientDrive(context.Background(), SDKSigner{}, NewStepClient(ts.URL),
		baseReq(sdkChainID, sdkOperator, "enroll:sdk-1"), bootKey, rootKey, []byte("boot-node-claims"), time.Now(), 600)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if !resp.OK {
		t.Fatalf("core-verified enrollment rejected at %s: %s", resp.Step, resp.Error)
	}

	// The same nonce against a second server sharing the store is a replay.
	_, tsReplay := newSDKServer(t, store, rootPK)
	resp, err = ClientDrive(context.Background(), SDKSigner{}, NewStepClient(tsReplay.URL),
		baseReq(sdkChainID, sdkOperator, "enroll:sdk-1"), bootKey, rootKey, []byte("boot-node-claims"), time.Now(), 600)
	if err != nil {
		t.Fatalf("replay enroll: %v", err)
	}
	if resp.OK || resp.Step != StepRegister {
		t.Fatalf("replayed nonce must be refused at register, got ok=%v step=%s", resp.OK, resp.Step)
	}
}

// tamperedChallengeSigner signs the challenge correctly, then corrupts its signature.
type tamperedChallengeSigner struct{ SDKSigner }

func (s tamperedChallengeSigner) SignChallenge(privateKey []byte, chainID, nonce string, issuedAtUnix, expiresAtUnix uint64, operator string, claims []byte) (signing.SignedMessage, error) {
	msg, err := s.SDKSigner.SignChallenge(privateKey, chainID, nonce, issuedAtUnix, expiresAtUnix, operator, claims)
	if err == nil && len(msg.Signature) > 0 {
		msg.Signature[0] ^= 0xff
	}
	return msg, err
}

// substitutingSigner countersigns an approval that names a different boot-node key than
// the one that signed the challenge: the substitution the approval binding must refuse.
type substitutingSigner struct {
	SDKSigner
	other []byte
}

func (s substitutingSigner) SignApproval(rootKey []byte, chainID, nonce string, issuedAtUnix, expiresAtUnix uint64, challengeEnvelope string, _ []byte) (signing.SignedMessage, error) {
	return s.SDKSigner.SignApproval(rootKey, chainID, nonce, issuedAtUnix, expiresAtUnix, challengeEnvelope, s.other)
}

// TestEnrollRejectionCountersSplitByCoreCode drives real core refusals through the
// production path and checks each lands in the counter for its core reason code.
func TestEnrollRejectionCountersSplitByCoreCode(t *testing.T) {
	bootKey := sdkKey(t, sdkBootHex)
	rootKey := sdkKey(t, sdkRootHex)
	rootPK := sdkRootPublicKey(t, rootKey)

	cases := []struct {
		name   string
		signer Signer
		want   string
	}{
		{"tampered challenge signature", tamperedChallengeSigner{}, "core_" + signing.EnrollCodeChallengeSignature},
		{"approval over a substituted boot key", substitutingSigner{other: sdkKey(t, sdkOtherHex)}, "core_" + signing.EnrollCodeApprovalPayloadMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, ts := newSDKServer(t, signing.NewMemoryNonceStore(), rootPK)
			resp, err := ClientDrive(context.Background(), tc.signer, NewStepClient(ts.URL),
				baseReq(sdkChainID, sdkOperator, "enroll:counter-1"), bootKey, rootKey, nil, time.Now(), 600)
			if err != nil {
				t.Fatalf("drive: %v", err)
			}
			if resp.OK {
				t.Fatal("tampered enrollment must be refused")
			}
			stats := srv.Stats()
			if stats.Accepted != 0 {
				t.Fatalf("refused enrollment counted as accepted: %+v", stats)
			}
			if stats.Rejections[tc.want] != 1 {
				t.Fatalf("counter %q = %d, want 1; rejections %+v", tc.want, stats.Rejections[tc.want], stats.Rejections)
			}
		})
	}
}

// TestEnrollReplayCounterHasNoCoreCode checks a replay, which the host store refuses
// without a core code, stays in the verification_rejected bucket.
func TestEnrollReplayCounterHasNoCoreCode(t *testing.T) {
	bootKey := sdkKey(t, sdkBootHex)
	rootKey := sdkKey(t, sdkRootHex)
	rootPK := sdkRootPublicKey(t, rootKey)
	store := signing.NewMemoryNonceStore()

	_, first := newSDKServer(t, store, rootPK)
	if resp, err := ClientDrive(context.Background(), SDKSigner{}, NewStepClient(first.URL),
		baseReq(sdkChainID, sdkOperator, "enroll:replay-counter"), bootKey, rootKey, nil, time.Now(), 600); err != nil || !resp.OK {
		t.Fatalf("first enrollment: ok=%v err=%v", resp.OK, err)
	}

	srv, second := newSDKServer(t, store, rootPK)
	if resp, err := ClientDrive(context.Background(), SDKSigner{}, NewStepClient(second.URL),
		baseReq(sdkChainID, sdkOperator, "enroll:replay-counter"), bootKey, rootKey, nil, time.Now(), 600); err != nil || resp.OK {
		t.Fatalf("replay must be refused: ok=%v err=%v", resp.OK, err)
	}
	if got := srv.Stats().Rejections["verification_rejected"]; got != 1 {
		t.Fatalf("replay counter verification_rejected = %d, want 1; %+v", got, srv.Stats().Rejections)
	}
}

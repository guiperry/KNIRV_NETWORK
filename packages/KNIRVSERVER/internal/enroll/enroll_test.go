package enroll

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
)

// ---- fakes (the local core build does not implement the DEC-2 enroll ops) ----

// fakeSigner produces SignedMessages whose "signedness" is controllable so the
// test can exercise the unsigned-challenge and missing-approval rejection paths.
type fakeSigner struct {
	signChallenge bool // when false, the challenge carries no signature
	signApproval  bool // when false, the approval is empty (missing root approval)
}

func (f fakeSigner) SignChallenge(_ []byte, chainID, nonce string, _, _ uint64, operator string, claims []byte) (signing.SignedMessage, error) {
	msg := signing.SignedMessage{
		Envelope:  []byte("env:" + chainID + ":" + nonce + ":" + operator),
		PublicKey: []byte("boot-pub"),
	}
	if f.signChallenge {
		msg.Signature = []byte("signed-challenge")
	}
	return msg, nil
}

func (f fakeSigner) SignApproval(_ []byte, chainID, nonce string, _, _ uint64, challengeEnvelope string) (signing.SignedMessage, error) {
	if !f.signApproval {
		return signing.SignedMessage{}, nil
	}
	return signing.SignedMessage{
		Envelope:  []byte("approval:" + challengeEnvelope),
		PublicKey: []byte("root-pub"),
		Signature: []byte("signed-approval"),
	}, nil
}

// fakeVerifier enforces the same contract as signing.VerifyBootnodeEnrollment:
// a signed challenge, a present root approval, and replay prevention through the
// real signing.NonceStore (which it consumes with the explicit nonce).
type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, e signing.BootnodeEnrollment, nonce, _, _ string, store signing.NonceStore, _ time.Time) (bool, error) {
	if len(e.Challenge.Signature) == 0 {
		return false, nil // unsigned challenge
	}
	if len(e.Approval.Signature) == 0 {
		return false, nil // missing root approval
	}
	consumed, err := store.Consume(nonce)
	if err != nil {
		return false, err
	}
	return consumed, nil // consumed==false on replay
}

// startServer wires the enrollment Server onto an httptest server.
func startServer(t *testing.T, verifier Verifier, store signing.NonceStore, rootPub, operator string) *httptest.Server {
	t.Helper()
	srv := NewServer(verifier, store, rootPub, operator, nil)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	return httptest.NewServer(mux)
}

func baseReq(chainID, operator, nonce string) StepRequest {
	return StepRequest{ChainID: chainID, Operator: operator, Nonce: nonce}
}

// TestEnrollSequenceOrder proves the CLI builds transfer -> bond -> register and
// the server accepts it in that order.
func TestEnrollSequenceOrder(t *testing.T) {
	store := signing.NewMemoryNonceStore()
	ts := startServer(t, fakeVerifier{}, store, "root-pub", "operator:alice")
	defer ts.Close()

	steps := NewStepClient(ts.URL)
	resp, err := ClientDrive(context.Background(), fakeSigner{signChallenge: true, signApproval: true}, steps, baseReq("chain-x", "operator:alice", "enroll:order-1"), []byte("boot"), []byte("root"), []byte("claims"), time.Now(), 600)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected accepted enrollment, got rejected at %s: %s", resp.Step, resp.Error)
	}
	want := []string{StepTransfer, StepBond, StepRegister}
	if len(resp.Order) != 3 || resp.Order[0] != want[0] || resp.Order[1] != want[1] || resp.Order[2] != want[2] {
		t.Fatalf("server recorded wrong order: %v, want %v", resp.Order, want)
	}
}

// TestEnrollRejectsReplay proves a replayed nonce (already consumed by the
// NonceStore) is rejected even though the signatures are valid.
func TestEnrollRejectsReplay(t *testing.T) {
	store := signing.NewMemoryNonceStore()

	// First, accepted enrollment against server instance A.
	tsA := startServer(t, fakeVerifier{}, store, "root-pub", "operator:alice")
	defer tsA.Close()
	stepsA := NewStepClient(tsA.URL)
	resp, err := ClientDrive(context.Background(), fakeSigner{signChallenge: true, signApproval: true}, stepsA, baseReq("chain-x", "operator:alice", "enroll:replay-1"), []byte("boot"), []byte("root"), nil, time.Now(), 600)
	if err != nil {
		t.Fatalf("first enroll: %v", err)
	}
	if !resp.OK {
		t.Fatalf("first enrollment should succeed: %s", resp.Error)
	}

	// Replay the same nonce against a fresh server instance sharing the store.
	tsB := startServer(t, fakeVerifier{}, store, "root-pub", "operator:alice")
	defer tsB.Close()
	stepsB := NewStepClient(tsB.URL)
	resp, err = ClientDrive(context.Background(), fakeSigner{signChallenge: true, signApproval: true}, stepsB, baseReq("chain-x", "operator:alice", "enroll:replay-1"), []byte("boot"), []byte("root"), nil, time.Now(), 600)
	if err != nil {
		t.Fatalf("replay enroll: %v", err)
	}
	if resp.OK {
		t.Fatal("replayed nonce must be rejected")
	}
	if resp.Step != StepRegister {
		t.Fatalf("replay must be rejected at register, got step %s", resp.Step)
	}
}

// TestEnrollRejectsUnsignedChallenge proves an unsigned challenge is rejected.
func TestEnrollRejectsUnsignedChallenge(t *testing.T) {
	store := signing.NewMemoryNonceStore()
	ts := startServer(t, fakeVerifier{}, store, "root-pub", "operator:alice")
	defer ts.Close()
	steps := NewStepClient(ts.URL)
	resp, err := ClientDrive(context.Background(), fakeSigner{signChallenge: false, signApproval: true}, steps, baseReq("chain-x", "operator:alice", "enroll:unsigned-1"), []byte("boot"), []byte("root"), nil, time.Now(), 600)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if resp.OK {
		t.Fatal("unsigned challenge must be rejected")
	}
}

// TestEnrollRejectsMissingApproval proves a missing root approval is rejected.
func TestEnrollRejectsMissingApproval(t *testing.T) {
	store := signing.NewMemoryNonceStore()
	ts := startServer(t, fakeVerifier{}, store, "root-pub", "operator:alice")
	defer ts.Close()
	steps := NewStepClient(ts.URL)
	resp, err := ClientDrive(context.Background(), fakeSigner{signChallenge: true, signApproval: false}, steps, baseReq("chain-x", "operator:alice", "enroll:noapproval-1"), []byte("boot"), []byte("root"), nil, time.Now(), 600)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if resp.OK {
		t.Fatal("missing root approval must be rejected")
	}
}

// TestEnrollRejectsRegisterBeforeBond proves the server enforces transfer-first
// then bond then register: a register without a prior transfer/bond is rejected.
func TestEnrollRejectsRegisterBeforeBond(t *testing.T) {
	store := signing.NewMemoryNonceStore()
	srv := NewServer(fakeVerifier{}, store, "root-pub", "operator:alice", nil)
	// Attempt register directly, skipping transfer and bond.
	resp := srv.RecordStep(StepRegister, baseReq("chain-x", "operator:alice", "enroll:early-reg"), time.Now())
	if resp.OK {
		t.Fatal("register without transfer/bond must be rejected")
	}
	if resp.Step != StepRegister {
		t.Fatalf("expected rejection at register, got %s", resp.Step)
	}
}

// TestEnrollSocketEndToEnd proves enrollment works end-to-end over the internal
// Unix-domain socket (no public HTTP listener involved), via the real ServeUnix
// and the socket client.
func TestEnrollSocketEndToEnd(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "enroll.sock")
	store := signing.NewMemoryNonceStore()
	srv := NewServer(fakeVerifier{}, store, "root-pub", "operator:alice", nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := srv.ServeUnix(ctx, sock); err != nil && ctx.Err() == nil {
			t.Errorf("ServeUnix: %v", err)
		}
	}()

	// Wait for the socket file to appear before connecting.
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("enrollment socket never came up")
		}
		time.Sleep(10 * time.Millisecond)
	}

	steps := NewStepClientUnix(sock)
	resp, err := ClientDrive(ctx, fakeSigner{signChallenge: true, signApproval: true}, steps,
		baseReq("chain-x", "operator:alice", "enroll:sock-1"), []byte("boot"), []byte("root"), nil, time.Now(), 600)
	if err != nil {
		t.Fatalf("socket enroll: %v", err)
	}
	if !resp.OK {
		t.Fatalf("socket enrollment rejected: %s", resp.Error)
	}
	want := []string{StepTransfer, StepBond, StepRegister}
	if len(resp.Order) != 3 || resp.Order[0] != want[0] || resp.Order[1] != want[1] || resp.Order[2] != want[2] {
		t.Fatalf("socket server recorded wrong order: %v, want %v", resp.Order, want)
	}
}

// TestEnrollPublicRouteMutation documents the failure mode that TestEnrollNotOnPublicMux
// guards against: if someone re-adds /enroll/* to the public mux, the endpoint becomes
// reachable (no 404), so the public-mux guard test must fail. This is the required
// mutation check for 2d2.
func TestEnrollPublicRouteMutation(t *testing.T) {
	srv := NewServer(fakeVerifier{}, signing.NewMemoryNonceStore(), "root-pub", "operator:alice", nil)
	mutatedMux := http.NewServeMux()
	srv.RegisterRoutes(mutatedMux) // MUTATION: enrollment re-added to the public mux
	ts := httptest.NewServer(mutatedMux)
	defer ts.Close()

	steps := NewStepClient(ts.URL)
	resp, err := steps.do(context.Background(), StepTransfer, baseReq("chain-x", "operator:alice", "n"))
	if err != nil {
		t.Fatalf("mutation scenario: /enroll/transfer must be reachable on the (mutated) public mux, got err %v", err)
	}
	if !resp.OK {
		t.Fatalf("mutation scenario: transfer should be accepted on the public mux, got %s", resp.Error)
	}
}

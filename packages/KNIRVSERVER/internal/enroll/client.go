package enroll

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
)

// Signer produces the two signatures that make up a DEC-2 boot-node enrollment:
// the new boot-node key's challenge and the root owner's approval of it. The
// production implementation delegates to the SDK; tests inject a fake.
type Signer interface {
	SignChallenge(privateKey []byte, chainID, nonce string, issuedAtUnix, expiresAtUnix uint64, operator string, claims []byte) (signing.SignedMessage, error)
	SignApproval(rootKey []byte, chainID, nonce string, issuedAtUnix, expiresAtUnix uint64, challengeEnvelope string) (signing.SignedMessage, error)
}

// SDKSigner is the production signer, delegating to the SDK's DEC-2 enrollment
// operations.
type SDKSigner struct{}

func (SDKSigner) SignChallenge(privateKey []byte, chainID, nonce string, issuedAtUnix, expiresAtUnix uint64, operator string, claims []byte) (signing.SignedMessage, error) {
	return signing.SignBootnodeEnrollChallenge(privateKey, chainID, nonce, issuedAtUnix, expiresAtUnix, operator, claims)
}

func (SDKSigner) SignApproval(rootKey []byte, chainID, nonce string, issuedAtUnix, expiresAtUnix uint64, challengeEnvelope string) (signing.SignedMessage, error) {
	return signing.SignBootnodeEnrollApproval(rootKey, chainID, nonce, issuedAtUnix, expiresAtUnix, challengeEnvelope)
}

// StepClient talks to the D14 enrollment server, performing the three steps in
// the mandated order. In production it dials the internal Unix-domain socket; the
// HTTP variant exists only for in-process and local testing.
type StepClient struct {
	base string
	http *http.Client
}

// NewStepClient returns an HTTP client targeting the enrollment server at base
// (e.g. "http://localhost:8084"). Prefer NewStepClientUnix for production.
func NewStepClient(base string) *StepClient {
	return &StepClient{base: base, http: &http.Client{Timeout: 30 * time.Second}}
}

// NewStepClientUnix returns a client that dials the enrollment server over its
// internal Unix-domain socket at socketPath (e.g. /run/knirv/enroll.sock). The
// host in the request URL is ignored; the socket path is used for the dial.
func NewStepClientUnix(socketPath string) *StepClient {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
	}
	return &StepClient{
		base: "http://unix",
		http: &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}
}

func (c *StepClient) do(ctx context.Context, step string, req StepRequest) (stepResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return stepResponse{}, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/enroll/"+step, bytes.NewReader(body))
	if err != nil {
		return stepResponse{}, err
	}
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(r)
	if err != nil {
		return stepResponse{}, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var out stepResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return stepResponse{}, fmt.Errorf("decode %s response: %w", step, err)
	}
	return out, nil
}

// ClientDrive performs one full DEC-2 boot-node enrollment against the server,
// following the order transfer -> bond -> register. It builds the signed
// challenge and root approval, then drives the three steps in order and returns
// the final register response. Callers use ClientDrive so the order is single
// sourced.
func ClientDrive(ctx context.Context, signer Signer, steps *StepClient, req StepRequest, bootKey, rootKey []byte, claims []byte, now time.Time, lifetimeSec uint64) (stepResponse, error) {
	issued := uint64(now.Unix())
	expires := issued + lifetimeSec

	challenge, err := signer.SignChallenge(bootKey, req.ChainID, req.Nonce, issued, expires, req.Operator, claims)
	if err != nil {
		return stepResponse{}, fmt.Errorf("sign challenge: %w", err)
	}
	approval, err := signer.SignApproval(rootKey, req.ChainID, req.Nonce, issued, expires, string(challenge.Envelope))
	if err != nil {
		return stepResponse{}, fmt.Errorf("sign approval: %w", err)
	}
	enrollment := signing.BootnodeEnrollment{Challenge: challenge, Approval: approval}

	// Order: transfer first.
	if r, err := steps.do(ctx, StepTransfer, StepRequest{ChainID: req.ChainID, Operator: req.Operator}); err != nil {
		return stepResponse{}, fmt.Errorf("transfer: %w", err)
	} else if !r.OK {
		return r, fmt.Errorf("transfer rejected: %s", r.Error)
	}

	// Then bond.
	if r, err := steps.do(ctx, StepBond, StepRequest{ChainID: req.ChainID, Operator: req.Operator}); err != nil {
		return stepResponse{}, fmt.Errorf("bond: %w", err)
	} else if !r.OK {
		return r, fmt.Errorf("bond rejected: %s", r.Error)
	}

	// Then register.
	reg := StepRequest{ChainID: req.ChainID, Operator: req.Operator, Nonce: req.Nonce, Enrollment: &enrollment}
	return steps.do(ctx, StepRegister, reg)
}

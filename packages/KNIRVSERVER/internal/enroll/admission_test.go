package enroll

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
)

// All keys in this file are fixed in-source TEST vectors. No file, env var or
// secrets provider is read, and no vault is touched.
const (
	testBootNodeKeyHex = "1a2b3c4d5e6f7081928374655647382910fedcba0987654321fedcba09876543"
	testRootKeyHex     = "ffffeeeeddddccccbbbbaaaa9999888877776666555544443333222211110000"
	testChainID        = "knirv-testnet-1"
	testOperator       = "operator:alice"
	testRegistrationID = "reg-1"
)

type mapRegistrations struct {
	regs map[string]Registration
}

func (m mapRegistrations) Lookup(_ context.Context, registrationID string) (Registration, bool, error) {
	reg, ok := m.regs[registrationID]
	return reg, ok, nil
}

type recordingAdmitter struct {
	calls []ValidatorAdmission
	err   error
}

func (r *recordingAdmitter) Admit(_ context.Context, admission ValidatorAdmission) error {
	r.calls = append(r.calls, admission)
	return r.err
}

type enrollmentFixture struct {
	enrollment  signing.BootnodeEnrollment
	nonce       string
	rootPubKey  string
	validatorID string
	now         time.Time
	regs        mapRegistrations
	admitter    *recordingAdmitter
	store       *signing.MemoryNonceStore
	issuer      *Issuer
}

// signEnrollment signs a challenge with the test boot key and approves it with the
// test root key, for the given nonce and claims.
func signEnrollment(t *testing.T, chainID, operator, nonce string, claims []byte, now time.Time) signing.BootnodeEnrollment {
	t.Helper()
	bootKey, err := hex.DecodeString(testBootNodeKeyHex)
	if err != nil {
		t.Fatalf("decode boot key: %v", err)
	}
	rootKey, err := hex.DecodeString(testRootKeyHex)
	if err != nil {
		t.Fatalf("decode root key: %v", err)
	}
	expires := now.Add(10 * time.Minute).Unix()
	challenge, err := signing.SignBootnodeEnrollChallenge(bootKey, chainID, nonce, uint64(now.Unix()), uint64(expires), operator, claims)
	if err != nil {
		t.Fatalf("sign challenge: %v", err)
	}
	approval, err := signing.SignBootnodeEnrollApproval(rootKey, chainID, nonce, uint64(now.Unix()), uint64(expires),
		base64.StdEncoding.EncodeToString(challenge.Envelope), base64.StdEncoding.EncodeToString(challenge.PublicKey))
	if err != nil {
		t.Fatalf("sign approval: %v", err)
	}
	return signing.BootnodeEnrollment{Challenge: challenge, Approval: approval}
}

// newEnrollmentFixture approves registrationID, issues a server nonce for it, and
// signs the enrollment with that issued nonce. The nonce argument is only a label
// for the test; the value actually used is f.nonce (the issued one).
func newEnrollmentFixture(t *testing.T, chainID, operator, registrationID, _ string) *enrollmentFixture {
	t.Helper()
	now := time.Unix(1_700_000_000, 0)
	regs := mapRegistrations{regs: map[string]Registration{}}
	issuer := NewIssuer(regs, chainID, 2*time.Minute)

	// Derive the validator ID from the test boot key first, then approve it.
	bootKey, err := hex.DecodeString(testBootNodeKeyHex)
	if err != nil {
		t.Fatalf("decode boot key: %v", err)
	}
	probe, err := signing.SignBootnodeEnrollChallenge(bootKey, chainID, "enroll:probe", uint64(now.Unix()), uint64(now.Add(time.Minute).Unix()), operator, nil)
	if err != nil {
		t.Fatalf("probe challenge: %v", err)
	}
	validatorID, err := signing.Address(probe.PublicKey, signing.DefaultAddressPrefix)
	if err != nil {
		t.Fatalf("derive validator address: %v", err)
	}
	regs.regs[registrationID] = Registration{RegistrationID: registrationID, ValidatorID: validatorID, ChainID: chainID}
	issuer.registrations = regs

	issued, err := issuer.Issue(context.Background(), registrationID, now, 0)
	if err != nil {
		t.Fatalf("issue nonce: %v", err)
	}
	enrollment := signEnrollment(t, chainID, operator, issued.Nonce,
		MarshalClaims(EnrollmentClaims{RegistrationID: registrationID}), now)

	return &enrollmentFixture{
		enrollment:  enrollment,
		nonce:       issued.Nonce,
		rootPubKey:  base64.StdEncoding.EncodeToString(enrollment.Approval.PublicKey),
		validatorID: validatorID,
		now:         now,
		regs:        regs,
		admitter:    &recordingAdmitter{},
		store:       signing.NewMemoryNonceStore(),
		issuer:      issuer,
	}
}

func (f *enrollmentFixture) server(chainID, operator string) *Server {
	return NewServer(SDKVerifier{}, f.store, f.rootPubKey, operator, nil).
		ConfigureAdmission(chainID, f.regs, f.admitter).
		WithIssuer(f.issuer)
}

// bondedServer returns a server whose transfer and bond steps are already done
// for the test session, so only register attempts follow.
func (f *enrollmentFixture) bondedServer(t *testing.T, srv *Server, now time.Time) {
	t.Helper()
	req := StepRequest{ChainID: testChainID, Operator: testOperator}
	if resp := srv.RecordStep(StepTransfer, req, now); !resp.OK {
		t.Fatalf("transfer: %s", resp.Error)
	}
	if resp := srv.RecordStep(StepBond, req, now); !resp.OK {
		t.Fatalf("bond: %s", resp.Error)
	}
}

func (f *enrollmentFixture) register(srv *Server, e signing.BootnodeEnrollment, nonce string, now time.Time) stepResponse {
	return srv.RecordStep(StepRegister, StepRequest{ChainID: testChainID, Operator: testOperator, Nonce: nonce, Enrollment: &e}, now)
}

func (f *enrollmentFixture) drive(t *testing.T, srv *Server, chainID, operator, nonce string, now time.Time) stepResponse {
	t.Helper()
	req := StepRequest{ChainID: chainID, Operator: operator, Nonce: nonce}
	if resp := srv.RecordStep(StepTransfer, req, now); !resp.OK {
		t.Fatalf("transfer: %s", resp.Error)
	}
	if resp := srv.RecordStep(StepBond, req, now); !resp.OK {
		t.Fatalf("bond: %s", resp.Error)
	}
	req.Enrollment = &f.enrollment
	return srv.RecordStep(StepRegister, req, now)
}

func TestAdmissionAcceptsBoundEnrollment(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	srv := f.server(testChainID, testOperator)

	resp := f.drive(t, srv, testChainID, testOperator, f.nonce, f.now)
	if !resp.OK {
		t.Fatalf("expected accepted enrollment, got: %s", resp.Error)
	}
	if len(f.admitter.calls) != 1 {
		t.Fatalf("admitter calls = %d, want 1", len(f.admitter.calls))
	}
	got := f.admitter.calls[0]
	if got.ChainID != testChainID || got.RegistrationID != testRegistrationID || got.ValidatorID != f.validatorID {
		t.Fatalf("unexpected admission record: %+v", got)
	}
	if got.PublicKey != hex.EncodeToString(f.enrollment.Challenge.PublicKey) {
		t.Fatalf("admission public key mismatch: %s", got.PublicKey)
	}
	if stats := srv.Stats(); stats.Accepted != 1 {
		t.Fatalf("accepted = %d, want 1", stats.Accepted)
	}
}

// wantRejectionReason maps each invalid-enrollment case to the counter the server
// must increment. A bad signature, expiry or operator mismatch is refused by the
// core with a stable code, counted as core_<code>. A missing approval is reported
// as an error by the binding (verification_error).
var wantRejectionReason = map[string]string{
	"missing countersignature":     "verification_error",
	"bad boot signature":           "core_" + signing.EnrollCodeChallengeSignature,
	"bad root signature":           "core_" + signing.EnrollCodeApprovalSignature,
	"expired":                      "core_" + signing.EnrollCodeChallengeWindow,
	"wrong chain":                  "wrong_chain",
	"wrong operator":               "core_" + signing.EnrollCodeOperatorMismatch,
	"pubkey validator id mismatch": "validator_id_mismatch",
	"unknown registration":         "unknown_registration",
}

func TestAdmissionRejectsInvalidEnrollments(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(f *enrollmentFixture, srvChain, srvOperator *string, now *time.Time)
	}{
		{
			name: "missing countersignature",
			mutate: func(f *enrollmentFixture, _, _ *string, _ *time.Time) {
				f.enrollment.Approval.Signature = nil
			},
		},
		{
			name: "bad boot signature",
			mutate: func(f *enrollmentFixture, _, _ *string, _ *time.Time) {
				f.enrollment.Challenge.Signature[0] ^= 0xff
			},
		},
		{
			name: "bad root signature",
			mutate: func(f *enrollmentFixture, _, _ *string, _ *time.Time) {
				f.enrollment.Approval.Signature[0] ^= 0xff
			},
		},
		{
			name: "expired",
			mutate: func(_ *enrollmentFixture, _, _ *string, now *time.Time) {
				*now = now.Add(11 * time.Minute)
			},
		},
		{
			name: "wrong chain",
			mutate: func(_ *enrollmentFixture, srvChain, _ *string, _ *time.Time) {
				*srvChain = "knirv-other-1"
			},
		},
		{
			name: "wrong operator",
			mutate: func(_ *enrollmentFixture, _, srvOperator *string, _ *time.Time) {
				*srvOperator = "operator:bob"
			},
		},
		{
			name: "pubkey validator id mismatch",
			mutate: func(f *enrollmentFixture, _, _ *string, _ *time.Time) {
				f.regs.regs[testRegistrationID] = Registration{
					RegistrationID: testRegistrationID,
					ValidatorID:    "knirv1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqn0h4h2",
					ChainID:        testChainID,
				}
			},
		},
		{
			name: "unknown registration",
			mutate: func(f *enrollmentFixture, _, _ *string, _ *time.Time) {
				f.regs.regs = map[string]Registration{}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
			chainID, operator := testChainID, testOperator
			now := f.now
			tc.mutate(f, &chainID, &operator, &now)

			srv := f.server(chainID, operator)
			resp := f.drive(t, srv, testChainID, testOperator, f.nonce, now)
			if resp.OK {
				t.Fatal("invalid enrollment must be rejected")
			}
			if len(f.admitter.calls) != 0 {
				t.Fatalf("invalid enrollment must not admit, got %d calls", len(f.admitter.calls))
			}
			stats := srv.Stats()
			if stats.Accepted != 0 {
				t.Fatalf("rejected enrollment must not count as accepted: %+v", stats)
			}
			want, ok := wantRejectionReason[tc.name]
			if !ok {
				t.Fatalf("no expected rejection reason for case %q", tc.name)
			}
			if stats.Rejections[want] == 0 {
				t.Fatalf("rejection counter %q not incremented; got %+v", want, stats.Rejections)
			}
		})
	}
}

func TestAdmissionRejectsReplay(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	srv := f.server(testChainID, testOperator)
	if resp := f.drive(t, srv, testChainID, testOperator, f.nonce, f.now); !resp.OK {
		t.Fatalf("first enrollment rejected: %s", resp.Error)
	}

	f.admitter.calls = nil
	replayServer := f.server(testChainID, testOperator)
	resp := f.drive(t, replayServer, testChainID, testOperator, f.nonce, f.now)
	if resp.OK {
		t.Fatal("replayed enrollment must be rejected")
	}
	if len(f.admitter.calls) != 0 {
		t.Fatalf("replay must not reach admission, got %d calls", len(f.admitter.calls))
	}
	if !strings.Contains(strings.ToLower(resp.Error), "replay") {
		t.Fatalf("expected replay rejection, got %q", resp.Error)
	}
	if replayServer.Stats().Rejections["replayed"] == 0 {
		t.Fatalf("replayed counter not incremented: %+v", replayServer.Stats())
	}
}

func TestAdmissionRequiresApprovedRegistration(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	f.regs.regs = map[string]Registration{}
	resp := f.drive(t, f.server(testChainID, testOperator), testChainID, testOperator, f.nonce, f.now)
	if resp.OK {
		t.Fatal("unknown registration must be rejected")
	}
	if len(f.admitter.calls) != 0 {
		t.Fatal("unknown registration must not admit")
	}
}

// An invalid attempt that carries a valid, issued nonce must not burn that nonce:
// the honest enrollment that follows is still accepted exactly once.
func TestInvalidAttemptDoesNotBurnValidNonce(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	srv := f.server(testChainID, testOperator)
	f.bondedServer(t, srv, f.now)

	bad := f.enrollment
	bad.Challenge.Signature = append([]byte(nil), bad.Challenge.Signature...)
	bad.Challenge.Signature[0] ^= 0xff
	if resp := f.register(srv, bad, f.nonce, f.now); resp.OK {
		t.Fatal("bad boot signature must be rejected")
	}
	if len(f.admitter.calls) != 0 {
		t.Fatal("rejected attempt must not admit")
	}
	resp := f.register(srv, f.enrollment, f.nonce, f.now)
	if !resp.OK {
		t.Fatalf("valid enrollment after invalid attempt was rejected: %s", resp.Error)
	}
	if len(f.admitter.calls) != 1 {
		t.Fatalf("admitter calls = %d, want 1", len(f.admitter.calls))
	}
}

// An admission failure after the consume burns the nonce: the backend write did
// not happen, a retry with the same approval is refused as a replay, and no second
// membership write is attempted.
func TestAdmissionFailureBurnsNonce(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	f.admitter.err = context.DeadlineExceeded
	srv := f.server(testChainID, testOperator)
	f.bondedServer(t, srv, f.now)

	if resp := f.register(srv, f.enrollment, f.nonce, f.now); resp.OK {
		t.Fatal("admission failure must surface as a rejection")
	}
	if len(f.admitter.calls) != 1 {
		t.Fatalf("admitter calls = %d, want 1", len(f.admitter.calls))
	}
	if srv.Stats().Rejections["admission_failed"] != 1 {
		t.Fatalf("admission_failed not counted: %+v", srv.Stats())
	}

	f.admitter.err = nil
	retry := f.server(testChainID, testOperator)
	f.bondedServer(t, retry, f.now)
	resp := f.register(retry, f.enrollment, f.nonce, f.now)
	if resp.OK {
		t.Fatal("retry with a burned nonce must be rejected")
	}
	if len(f.admitter.calls) != 1 {
		t.Fatalf("retry must not reach the admitter, calls = %d", len(f.admitter.calls))
	}
	if retry.Stats().Rejections["replayed"] != 1 {
		t.Fatalf("retry not counted as replay: %+v", retry.Stats())
	}
}

// The challenge's public key can match the registration while the signed claim
// names a different validator ID; that must be rejected as a key/ID mismatch.
func TestChallengePubkeyMismatchesClaimsValidatorID(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	claims := MarshalClaims(EnrollmentClaims{
		RegistrationID: testRegistrationID,
		ValidatorID:    "knirv1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqn0h4h2",
	})
	issued, err := f.issuer.Issue(context.Background(), testRegistrationID, f.now, 0)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	f.enrollment = signEnrollment(t, testChainID, testOperator, issued.Nonce, claims, f.now)

	srv := f.server(testChainID, testOperator)
	f.bondedServer(t, srv, f.now)
	if resp := f.register(srv, f.enrollment, issued.Nonce, f.now); resp.OK {
		t.Fatal("claims validator_id mismatch must be rejected")
	}
	if srv.Stats().Rejections["validator_id_mismatch"] != 1 || len(f.admitter.calls) != 0 {
		t.Fatalf("expected validator_id_mismatch and no admission: %+v calls=%d", srv.Stats(), len(f.admitter.calls))
	}
}

func TestIssuedNonceAcceptedOnce(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	if err := f.issuer.Check(f.nonce, testRegistrationID, testChainID, f.now); err != nil {
		t.Fatalf("freshly issued nonce must check: %v", err)
	}
	// Check must not consume: it can be checked again.
	if err := f.issuer.Check(f.nonce, testRegistrationID, testChainID, f.now); err != nil {
		t.Fatalf("second check of unredeemed nonce: %v", err)
	}
}

func TestUnissuedNonceRejectedAndMembershipUnchanged(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	// Valid signatures, approval and registration, but a nonce the server never issued.
	f.enrollment = signEnrollment(t, testChainID, testOperator, "enroll:not-issued",
		MarshalClaims(EnrollmentClaims{RegistrationID: testRegistrationID}), f.now)
	srv := f.server(testChainID, testOperator)
	if resp := f.drive(t, srv, testChainID, testOperator, "enroll:not-issued", f.now); resp.OK {
		t.Fatal("unissued nonce must be rejected")
	}
	if srv.Stats().Rejections["nonce_not_issued"] != 1 || len(f.admitter.calls) != 0 {
		t.Fatalf("unissued nonce not counted / admitted: %+v calls=%d", srv.Stats(), len(f.admitter.calls))
	}
}

func TestIssuedNonceExpiresBeforeRegister(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	// Issuance TTL is 2 minutes; the challenge itself is still valid at +3 minutes.
	later := f.now.Add(3 * time.Minute)
	srv := f.server(testChainID, testOperator)
	if resp := f.drive(t, srv, testChainID, testOperator, f.nonce, later); resp.OK {
		t.Fatal("expired issued nonce must be rejected")
	}
	if srv.Stats().Rejections["nonce_expired"] != 1 || len(f.admitter.calls) != 0 {
		t.Fatalf("expired nonce not counted / admitted: %+v calls=%d", srv.Stats(), len(f.admitter.calls))
	}
}

func TestIssuedNonceBoundToItsRegistration(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	f.regs.regs["reg-2"] = Registration{RegistrationID: "reg-2", ValidatorID: f.validatorID, ChainID: testChainID}
	issued, err := f.issuer.Issue(context.Background(), "reg-2", f.now, 0)
	if err != nil {
		t.Fatalf("issue for reg-2: %v", err)
	}
	// Enrollment claims reg-1 but carries the nonce issued for reg-2.
	f.enrollment = signEnrollment(t, testChainID, testOperator, issued.Nonce,
		MarshalClaims(EnrollmentClaims{RegistrationID: testRegistrationID}), f.now)
	srv := f.server(testChainID, testOperator)
	if resp := f.drive(t, srv, testChainID, testOperator, issued.Nonce, f.now); resp.OK {
		t.Fatal("nonce issued for another registration must be rejected")
	}
	if srv.Stats().Rejections["nonce_registration_mismatch"] != 1 || len(f.admitter.calls) != 0 {
		t.Fatalf("mismatch not counted / admitted: %+v calls=%d", srv.Stats(), len(f.admitter.calls))
	}
}

func TestIssueRequiresApprovedRegistration(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	before := f.issuer.Outstanding()
	if _, err := f.issuer.Issue(context.Background(), "not-approved", f.now, 0); err == nil {
		t.Fatal("issuing for an unapproved registration must fail")
	}
	if f.issuer.Outstanding() != before {
		t.Fatal("failed issuance must not record a nonce")
	}
}

func TestAdmissionFailsClosedWithoutIssuer(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	srv := NewServer(SDKVerifier{}, f.store, f.rootPubKey, testOperator, nil).
		ConfigureAdmission(testChainID, f.regs, f.admitter)
	if resp := f.drive(t, srv, testChainID, testOperator, f.nonce, f.now); resp.OK {
		t.Fatal("admission without an issuer must be rejected")
	}
	if srv.Stats().Rejections["issuance_unconfigured"] != 1 || len(f.admitter.calls) != 0 {
		t.Fatalf("expected issuance_unconfigured and no admission: %+v", srv.Stats())
	}
}

func TestIssueRouteIsInternalOnly(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	srv := f.server(testChainID, testOperator)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// The gateway proxies only POST /enroll/*; the issue route must not be under it.
	if IssuePath == "" || strings.HasPrefix(IssuePath, "/enroll/") {
		t.Fatalf("issue path %q must sit outside the public /enroll/ prefix", IssuePath)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/enroll/issue", strings.NewReader("{}")))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("POST /enroll/issue must not exist, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, IssuePath, nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET on issue route = %d, want 405", rec.Code)
	}
}

func TestIssueRouteIssuesNonceForApprovedRegistration(t *testing.T) {
	f := newEnrollmentFixture(t, testChainID, testOperator, testRegistrationID, "")
	srv := f.server(testChainID, testOperator)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	body := `{"registration_id":"` + testRegistrationID + `","chain_id":"` + testChainID + `"}`
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, IssuePath, strings.NewReader(body)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"nonce":"enroll:`) {
		t.Fatalf("issue route: code=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	bad := `{"registration_id":"` + testRegistrationID + `","chain_id":"knirv-other-1"}`
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, IssuePath, strings.NewReader(bad)))
	if rec.Code != http.StatusBadRequest || srv.Stats().Rejections["issue_wrong_chain"] != 1 {
		t.Fatalf("wrong chain issue: code=%d stats=%+v", rec.Code, srv.Stats())
	}
}

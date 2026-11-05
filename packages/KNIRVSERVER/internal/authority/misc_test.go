package authority

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
)

func TestFailureCodeNamesEverySDKSentinel(t *testing.T) {
	for want, err := range map[string]error{
		"ErrUnknownOperator":  sdk.ErrUnknownOperator,
		"ErrExpiredSession":   sdk.ErrExpiredSession,
		"ErrSessionSignature": sdk.ErrSessionSignature,
		"ErrRecordSignature":  sdk.ErrRecordSignature,
		"ErrHeader":           sdk.ErrHeader,
		"ErrWitness":          sdk.ErrWitness,
		"ErrTarget":           sdk.ErrTarget,
		"ErrScope":            sdk.ErrScope,
		"ErrTier":             sdk.ErrTier,
		"ErrInvalid":          sdk.ErrInvalid,
	} {
		if got := FailureCode(fmt.Errorf("wrapped: %w", err)); got != want {
			t.Errorf("FailureCode(%v) = %q, want %q", err, got, want)
		}
	}
	if FailureCode(nil) != "" || FailureCode(errors.New("other")) != "ErrOther" {
		t.Error("nil must be empty and unknown errors ErrOther")
	}
}

func TestRefusalMessageAndAccessors(t *testing.T) {
	r := &Refusal{Code: RefusedScope, Reason: "nope", Err: sdk.ErrScope}
	if !strings.Contains(r.Error(), "scope") || !strings.Contains(r.Error(), "nope") || !errors.Is(r, sdk.ErrScope) {
		t.Fatalf("%v", r)
	}
	h := newHarness(t, KindBoot, 2)
	if h.svc.Operator() != h.signer.Address() || h.svc.Signer() == nil || h.svc.Store() == nil {
		t.Fatal("accessors")
	}
	deg, _ := New(Config{DegradedReason: "x"})
	if deg.Operator() != "" || deg.Signer() != nil || deg.Store() != nil {
		t.Fatal("a degraded service exposes no identity")
	}
}

func TestAnchorRunPostsOnAScheduleAndStopsWithItsContext(t *testing.T) {
	h := newHarness(t, KindRoot, 1)
	p := &flakyPoster{}
	a := newAnchor(t, h, p, 0, "")
	mustAuthorize(t, h, act(1, "evidence.commit", "", `{}`))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { a.Run(ctx, 10*time.Millisecond); close(done) }()
	deadline := time.Now().Add(5 * time.Second)
	for a.Status().LastEnd < 1 {
		if time.Now().After(deadline) {
			t.Fatal("Run never posted a checkpoint")
		}
		time.Sleep(5 * time.Millisecond)
	}
	mustAuthorize(t, h, act(1, "evidence.commit", "", `{"n":2}`))
	for a.Status().LastEnd < 2 {
		if time.Now().After(deadline) {
			t.Fatal("Run did not pick up the new record")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not stop")
	}
}

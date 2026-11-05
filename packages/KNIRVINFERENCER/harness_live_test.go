package inferencer

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	models "github.com/guiperry/knirv-sdk-go/models"
)

// TestHarnessVerifyAgainstARunningDaemonAndLlama is the live-path check for Tasks 3.4
// and 3.5: a real KNIRVMODELS daemon whose verifier talks to a real local llama, an
// evidence index that holds the Eiffel Tower facts, and the harness on top. It is
// skipped unless KNIRV_LIVE_MODELS_SOCKET names the daemon's Unix socket (the daemon,
// llama and index are started by the person running the check; see PendingImplementations
// Task 3.4 for the exact setup used on 2026-11-03).
//
// HEART is a stand-in that gives a wrong answer, so the test does not depend on what a
// small local model happens to say; HASHER, the index, the judge and the Trace are real.
func TestHarnessVerifyAgainstARunningDaemonAndLlama(t *testing.T) {
	sock := os.Getenv("KNIRV_LIVE_MODELS_SOCKET")
	if sock == "" {
		t.Skip("KNIRV_LIVE_MODELS_SOCKET not set")
	}
	client := models.NewClient("http://models", models.NewUnixHTTPClient(sock, 5*time.Minute))
	_, _, hasher := DaemonModels(client)
	heart := &fakeModel{name: "heart", ready: Readiness{Ready: true, Version: "stand-in"},
		do: func(Task) (Result, error) { return Result{Text: "The Eiffel Tower is 450 metres tall."}, nil }}

	dir := t.TempDir() + "/traces"
	store, err := NewTraceStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHarness([]HarnessModel{heart, hasher}, WithTraceStore(store),
		WithRoutes(map[Kind][]string{KindGenerate: {"heart"}, KindVerify: {"hasher"}}))
	ctx := context.Background()

	gen, _, err := h.Do(ctx, Task{Kind: KindGenerate, Input: "How tall is the Eiffel Tower?"})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	resp, tr, err := h.Verify(ctx, models.VerifyRequest{Text: gen.Text, Delivery: models.DeliveryInline, BudgetMS: 30000},
		Constraints{NoFlash: true, Budget: 60 * time.Second})
	if err != nil {
		t.Fatalf("verify: %v (trace %+v)", err, tr)
	}
	t.Logf("live verify took %v: gate=%s level=%d verdict=%s claims=%d degraded=%t", time.Since(start), resp.Gate, resp.GateLevel, resp.Verdict, len(resp.Claims), resp.Degraded)
	if err := resp.Check(); err != nil {
		t.Fatalf("the daemon's answer breaks the contract: %v", err)
	}
	if resp.Degraded || resp.Status != models.VerifyDone {
		t.Fatalf("expected a real verdict, got %+v", resp)
	}
	if resp.Gate != "refused" || len(resp.Claims) == 0 {
		t.Fatalf("a wrong HEART answer must be flagged: %+v", resp)
	}
	found := false
	for _, p := range resp.Citations {
		// positions index the concatenated evidence of every claim
		n := 0
		for _, c := range resp.Claims {
			if p < n+len(c.Evidence) {
				found = found || strings.Contains(c.Evidence[p-n].Text, "330 metres")
				break
			}
			n += len(c.Evidence)
		}
	}
	if !found {
		t.Errorf("the flag should cite the 330 metres passage: citations %v", resp.Citations)
	}
	v := tr.Verify
	if v == nil || v.Mode != "index" || v.FlashUsed || v.Gate != "refused" || v.Judge == "" || !tr.NoFlash {
		t.Fatalf("trace verify fields: %+v", tr)
	}

	// The trace is on disk and reloads.
	_ = store.Close()
	store2, err := NewTraceStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := store2.Read(0)
	if len(got) != 2 || got[0].Kind != KindVerify || got[0].Verify == nil {
		t.Fatalf("%+v", got)
	}

	// The status document carries the daemon's manifest.
	s := BuildPlatformStatus(ctx, client, nil, h, 5)
	if s.Manifest == nil || s.Manifest.Error != "" || len(s.Manifest.Hash) != 64 {
		t.Fatalf("manifest: %+v", s.Manifest)
	}
	t.Logf("manifest %s: %d stages, %d tools", s.Manifest.Version, s.Manifest.Stages, s.Manifest.Tools)
}

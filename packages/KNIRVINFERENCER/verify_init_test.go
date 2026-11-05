package inferencer

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	models "github.com/guiperry/knirv-sdk-go/models"
)

// TestVerifyCoreWarmedBeforeBudgetedRequest guards that NewHarness pre-loads the SDK
// Rust-core WASM, so the first budgeted verify does not pay the ~6s cold init inside the
// caller's 3s budget. It runs the production entry point (NewHarness -> Verify) in a fresh
// subprocess so the warmup can only have come from NewHarness — never from a pre-warm in
// TestMain. If the production wiring is removed, the first Verify pays the cold init and this
// test fails (pi_3_5_budget).
func TestVerifyCoreWarmedBeforeBudgetedRequest(t *testing.T) {
	const env = "KNIRV_COLD_WARMUP_SUBTEST"
	if os.Getenv(env) != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestVerifyCoreWarmedBeforeBudgetedRequest$")
		cmd.Env = append(os.Environ(), env+"=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("cold warmup subprocess failed (production NewHarness warmup likely missing): %v\n%s", err, out)
		}
		return
	}

	// Fresh process: TestMain set up KNIRV_SDK_DATA_DIR but did NOT pre-warm. Build a real
	// Harness over a fake daemon and verify the first budgeted request returns in budget.
	d := &verifyDaemon{verify: truthful}
	_, heart, hasher := DaemonModels(d.client(t))
	h := NewHarness([]HarnessModel{heart, hasher})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	if _, _, err := h.Verify(ctx, models.VerifyRequest{Text: "x y", Mode: models.ModeIndex, BudgetMS: 3000}, Constraints{}); err != nil {
		t.Fatalf("budgeted verify failed in cold subprocess: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("first budgeted verify took %s; production NewHarness did not warm the core", elapsed)
	}
}

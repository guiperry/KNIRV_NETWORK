package inferencer

import (
	"fmt"
	"os"
	"sync"

	models "github.com/guiperry/knirv-sdk-go/models"
)

// initVerifyCoreOnce ensures the SDK Rust-core WASM module is loaded exactly once per
// process. The first SDK core call loads and compiles the digest-pinned module; on a cold
// process that is a 6+ second cost. The SDK's Verify runs that contract check unbounded on
// a background context before the daemon POST, so if the load lands inside a caller's
// latency budget the POST is deadline-cancelled. Calling EnsureVerifyCoreWarmed from
// NewHarness loads it at startup, outside any per-request budget (pi_3_5_budget).
var initVerifyCoreOnce sync.Once

// EnsureVerifyCoreWarmed loads the SDK Rust-core WASM module once. It is idempotent and
// safe to call from NewHarness on every construction. The returned error is from the
// empty-fixture validation (the warmup's real product is the module load, a side effect of
// the core call); a genuine load failure surfaces on the first real Verify, so we log here
// rather than silently swallow it, and the guard test fails if the warmup is ever removed.
func EnsureVerifyCoreWarmed() {
	initVerifyCoreOnce.Do(func() {
		if err := (models.VerifyResponse{}).Check(); err != nil {
			fmt.Fprintf(os.Stderr, "InitVerifyCore (warmup) returned: %v\n", err)
		}
	})
}

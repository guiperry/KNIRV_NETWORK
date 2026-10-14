# KNIRVINFERENCER — the reasoning harness

Go module `github.com/guiperry/knirv-inference-go` (name kept). Not a service; `backend_server` imports it.

`Harness.Do(ctx, Task)` sends one typed task (`decide`, `analyze`, `generate`, `validate`, `server_task`, `execute_skill`, `plan`) to the first eligible model in its routing table:

| Kind | Default chain |
|---|---|
| decide | HEAD |
| analyze, generate | HEART → llama → external |
| validate | HASHER |
| server_task, execute_skill | llama |
| plan | external provider chain |

`UserFacingRoutes()` is the production table (llama dropped from analyze/generate). Per call: constraints (`LocalOnly`, `NoSpend`, `Pin`, latency `Budget`), readiness, circuit breaker (opens after 3 failures, 30 s cooldown, one half-open probe), optional HEAD advice for ambiguous kinds (used only if HEAD names an eligible model with confidence >= 0.6), then the chain. Every hop lands in a `Trace`; `Traces`, `Stats`, `RoutingEvents` expose them.

`ExecuteSkill` ensures the skill's ULoRA adapter is loaded in llama and runs with it. If the adapter cannot load, the base model answers and the result is `Degraded` with a note "skill X did NOT run".

`BuildPlatformStatus` builds the document served at `GET /api/models/status` (contract in `KNIRVSDK/models`). `InferenceService` keeps its public methods and routes through the harness (`SetHarness`).

Tests: `go test ./...`; `KNIRV_MODELS_BIN=<daemon binary> go test -run RealDaemon` runs the harness against a real daemon.

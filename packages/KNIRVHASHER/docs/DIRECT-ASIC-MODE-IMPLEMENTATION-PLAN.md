# Direct ASIC Mode: Implementation Plan

**Status:** Ready for implementation. Written for an implementing agent with no
prior context on this codebase's recent history - all grounding facts are
included below, with file:line anchors. Do not re-derive the architecture;
verify these anchors still hold (line numbers drift) and proceed.

**Do not implement this by re-deriving the design from scratch.** Every
decision below was made by reading the actual call graph, the actual wire
protocol, and the actual prior hardware test log. Follow the plan; where it
says "reuse X", reuse X rather than rewriting it.

## 1. Objective

Give the seeder pipeline's evolutionary jitter loop (`pipeline/3_DATA_SEEDER`)
an opt-in path where each of its per-pass hash computations is a genuine,
hardware-backed Bitmain ASIC operation - full caller control over every
header byte except the nonce (which no SHA-256 ASIC, on any interface, can
ever accept as caller-supplied - it is always hardware-searched) - instead of
the current local-software computation.

### Hard constraints from the requester

1. The new mode is server-side, activated by a boolean flag literally named
   `-direct` on `hasher-server`.
2. Full pipeline runs take too long to reach the seeding phase for iteration.
   Build a **new, standalone seeding benchmark program** that exercises only
   the affected code path, so validation doesn't require running stages
   0-2 or loading a real corpus.
3. The existing default behavior (software hash method; CGMiner/stratum ASIC
   mode; the current always-local `ComputeHash`/`ComputeDoubleHash` shortcut)
   **must remain fully operational and unchanged** when `-direct` is not
   used. Everything in this plan is additive.
4. This document is the deliverable. Do not implement it in this
   conversation - hand it to an implementing agent.

## 2. Grounding facts (verified this session, with anchors)

Read this section before touching code. It corrects two wrong assumptions
that would otherwise derail the implementation.

### 2.1 The real hot path is `ComputeDoubleHash`, not `ComputeBatch`

The seeder's actual per-generation call graph (confirmed by reading the live
code, not inferred):

```
pipeline/3_DATA_SEEDER/cmd/data-seeder/main.go: trainRecord/trainToken
  -> harness.EvaluatePopulation(...)                         (evolutionary.go:754)
  -> harness.EvaluatePopulationBatch(...)                    (evolutionary.go:787)
  -> hWrap.GetHashMethod().Execute21PassLoopBatch(headers, commitmentTarget)  (evolutionary.go:819)
       // hWrap.GetHashMethod() is a core.HashMethod - in ASIC mode, *asic.ASICMethod
  -> (*asic.ASICMethod).Execute21PassLoopBatch                (pkg/hashing/methods/asic/device.go:254)
       sets jitterEngine.SetHashMethod(&ASICHASHMethod{client: m.client})
  -> jitterEngine.Execute21PassLoopBatch(headers, targetTokenID)  (pkg/hashing/jitter/jitter_engine.go:254)
       // just loops Execute21PassLoop sequentially per header - NOT a real network batch
  -> jitterEngine.Execute21PassLoop(header, targetTokenID)    (jitter_engine.go:95)
       for pass := 0..20:
         hash, _ := je.hashMethod.ComputeDoubleHash(state.Header)   // <-- THE HOT CALL, once per pass
         ... XORJitterIntoHeader(state.Header, jitter) ...          // mutates header bytes 36:52
  -> ASICHASHMethod.ComputeDoubleHash(data)                   (asic/device.go:376)
  -> ASICClient.ComputeDoubleHash(data)                       (asic/asic_client.go:286)
```

Default population is 256 (`pipeline/3_DATA_SEEDER/cmd/data-seeder/main.go:35`),
pass count is 21 - **5,376 `ComputeDoubleHash` calls per generation**, per
token, with `*maxGenerations` defaulting to 2000. `ComputeBatch` is not on
this path at all. **Do not build anything around `ComputeBatch`.** The
integration point is `ComputeHash`/`ComputeDoubleHash` (the single-item gRPC
RPC), full stop.

### 2.2 A correct direct-hardware primitive already exists - it's just unreachable

`BuildTxTaskFromHeader(header []byte, workID uint8) []byte`
(`internal/driver/device/controller.go:1316-1352`) already builds a
protocol-correct Bitmain `TxTask` packet:

- `midstate[32]` = a **real** SHA-256 compression state over `header[0:64]`,
  via `computeMidstate` (`controller.go:1299-1313`) - not a raw byte copy.
- `data[12]` = `header[64:76]` verbatim (tail of merkle root + ntime + nbits).
- The nonce (`header[76:80]`) is never transmitted - it doesn't need to be;
  it's what the ASIC hardware searches and returns via `RxNonce`.

This gives full, correct control over version/prevhash/**entire merkle
root**/ntime/nbits - exactly what `XORJitterIntoHeader` needs (it writes
into header bytes 36:52, inside the merkle-root region this format fully
covers). `ParseRxNonce` (`controller.go:1355-1386`) parses the response.
`docs/BITMAIN-PROTOCOL.md`'s own hardware test log (Dec 2024/Jan 2026)
recorded a real `0xA2 RxNonce` response with a valid CRC using this exact
packet shape on this exact device.

**This is currently dead code for our purposes.** `BuildTxTaskFromHeader` is
used today only by `Device.MineWork`'s raw-device fallback branch
(`controller.go:809-852`), which is itself unreachable in the deployed
system because `OpenDevice`'s Strategy 0 (CGMiner) always wins first
(`controller.go:180-204`) and CGMiner holds the USB device exclusively, so
Strategy 2 (`controller.go:223-245`, raw USB) never even attempts to open.

A **separate, different, and buggy** packet builder - `buildTxTaskPacket`
(`controller.go:866-923`) - is what `ComputeBatch`'s raw-device branch
currently calls (`controller.go:687`). It copies raw input bytes directly
into the midstate field instead of computing a real midstate. This plan does
**not** touch `buildTxTaskPacket` or `ComputeBatch`'s raw path - out of
scope, since `ComputeBatch` isn't on the hot path (2.1). Leave it as-is.

### 2.3 No server-side `ComputeDoubleHash` RPC exists

`internal/proto/hasher/v1/*.proto` defines `ComputeHash`, `ComputeBatch`,
`StreamCompute`, `MineWork`, etc. - no `ComputeDoubleHash` RPC.
`ASICClient.ComputeDoubleHash` (`asic_client.go:286-290`) is a **client-side**
concept only. The plan must map the jitter engine's double-hash need onto
the existing single-item `ComputeHash` RPC (`server.go:46-64`,
`Device.ComputeHash` at `controller.go:626-631`) - one real hardware round
trip per pass, with the server returning the full `sha256d` of the exact
header it mined (mirroring the pattern `stratum_server.go`'s `handleSubmit`
already uses: never trust a self-reported hash, independently recompute it
from the real header + real nonce - `RxNonce` doesn't return a hash at all,
only a nonce, so this recomputation is mandatory, not optional).

### 2.4 Current default behavior (must not change)

`ASICClient.ComputeHash`/`ComputeDoubleHash` (`asic_client.go:280-290`)
**always** compute locally via `computeHashSoftware`, never touching the
network, per this session's earlier fix (see doc comment there for why).
`Device.ComputeHash` (`controller.go:626-631`) **always** computes
`sha256.Sum256(data)` locally server-side too, for the same reason. Both of
these must remain the default and must not be modified in place - this plan
adds new, separate, opt-in code paths alongside them.

### 2.5 `OpenDevice`'s strategy chain

`OpenDevice(enableTracing bool) (*Device, error)` (`controller.go:171-249`):
Strategy 0 = CGMiner (`ensureCGMinerRunning()` in
`cmd/driver/hasher-server/main.go:157-190` is called first by
`attemptDeviceRecovery`, `main.go:150-166`), Strategy 1 = kernel device
(`/dev/bitmain-asic`), Strategy 2 = raw USB (`usb_device_mips.go` on the
real MIPS target). Direct mode needs Strategy 0 skipped entirely - not
raced, not stopped-and-restarted (a prior attempt this session to
programmatically stop/restart CGMiner via `/etc/init.d/cgminer` left it
non-running and was fully reverted; do not repeat that pattern). CGMiner
must simply never be started when direct mode is requested.

### 2.6 Remote deployment start command

`internal/analyzer/deployer.go:854` is where `hasher-server` is actually
launched on the device during a real deployment:

```go
startCmd := fmt.Sprintf("sh -c '%s --port=8888 --trace=true > %s/hasher-server.log 2>&1 &' && echo 'SERVER_START_PID:$!' && sleep 1", remoteServerPath, d.config.RemoteDir)
```

Any new `-direct` flag must be threaded through here too, or it will never
reach a real deployed device - only local/manual test runs of
`hasher-server` would see it.

## 3. Design

### 3.1 Server side: `-direct` mode

**`cmd/driver/hasher-server/main.go`**

- Add `directMode = flag.Bool("direct", false, "skip CGMiner and drive the ASIC directly over raw USB (no stratum pool, no CGMiner); mutually exclusive in effect with CGMiner mode")`.
- `attemptDeviceRecovery(enableTracing bool)` (`main.go:150-166`) needs a
  `directMode bool` parameter. When true: **skip** the
  `ensureCGMinerRunning()` call and its whole primary-strategy branch
  entirely, and call `device.OpenDevice(enableTracing, true)` directly
  (new second parameter - see below). Do not call `ensureCGMinerRunning` in
  this branch under any circumstance.
- `NewHasherServerWithRecovery(enableTracing bool)` (`main.go:296-303`)
  needs the same `directMode bool` threaded through from `main()`.
- Log the mode clearly at startup (`log.Printf("Direct ASIC mode: %v", *directMode)`)
  since this determines whether CGMiner's GUI/monitoring is available at all
  during this run - operationally important, not just a debug line.

**`internal/driver/device/controller.go`**

- `OpenDevice(enableTracing bool, directMode bool) (*Device, error)` - add
  the parameter (this changes the public signature; update the two other
  callers: `server.go:29`'s `NewHasherServer` should also gain a
  `directMode bool` parameter and pass it through, defaulting call sites
  that don't care to `false`).
- At the top of `OpenDevice`, when `directMode` is true: **skip the entire
  Strategy 0 block** (`controller.go:180-204`) unconditionally - do not
  call `NewCGMinerMiner()`/`miner.IsAvailable()` at all, jump straight to
  Strategy 1. This guarantees CGMiner is never probed or started by this
  process in direct mode, which combined with 3.1's `ensureCGMinerRunning`
  skip means CGMiner is never touched anywhere in the direct-mode boot path.
- Add `useDirectASIC bool` field to the `Device` struct (near `useCGMiner`,
  `controller.go:76`). Set it to the `directMode` parameter's value
  unconditionally at the end of `OpenDevice` (regardless of which strategy
  ultimately succeeded) - `Device.ComputeHash` will branch on this field.
- **`Device.ComputeHash`** (`controller.go:626-631`): add a new branch
  **before** the existing unconditional `sha256.Sum256(data)` line:

  ```go
  func (d *Device) ComputeHash(data []byte) ([32]byte, error) {
      start := time.Now()
      if d.useDirectASIC && len(data) == 80 {
          hash, err := d.computeHashDirectASIC(data)
          if err != nil {
              return [32]byte{}, err
          }
          d.updateStats(1, uint64(len(data)), uint64(time.Since(start).Nanoseconds()))
          return hash, nil
      }
      hash := sha256.Sum256(data)
      d.updateStats(1, uint64(len(data)), uint64(time.Since(start).Nanoseconds()))
      return hash, nil
  }
  ```

  The `len(data) == 80` guard is deliberate and load-bearing: non-jitter
  callers (the gRPC health-check probe in
  `pkg/hashing/methods/asic/asic_client.go`'s `Connect()`, and any other
  caller of the plain `ComputeHash` RPC that isn't the jitter engine) pass
  arbitrary-length data and must keep getting a literal `sha256(data)` -
  only the jitter engine's own 80-byte headers should ever take the
  hardware path. Update the doc comment above `ComputeHash` to explain this
  branch (don't delete the existing explanation of why the software path
  exists - it's still correct for non-80-byte input and for non-direct mode).

- New method `computeHashDirectASIC(header []byte) ([32]byte, error)` on
  `*Device`, new code, roughly:

  ```go
  func (d *Device) computeHashDirectASIC(header []byte) ([32]byte, error) {
      if d.usbDevice == nil {
          return [32]byte{}, fmt.Errorf("direct ASIC mode active but no raw USB device is open")
      }

      txTask := BuildTxTaskFromHeader(header, directASICWorkID())
      d.mu.Lock()
      writeErr := d.usbDevice.SendPacket(txTask)
      d.mu.Unlock()
      if writeErr != nil {
          return [32]byte{}, fmt.Errorf("direct ASIC: send TxTask: %w", writeErr)
      }

      nonce, err := d.pollForDirectNonce(directPollTimeout)
      if err != nil {
          return [32]byte{}, fmt.Errorf("direct ASIC: %w", err)
      }

      var mined [80]byte
      copy(mined[:], header)
      binary.LittleEndian.PutUint32(mined[76:80], nonce)
      // sha256d, computed locally from the real header + real ASIC-found
      // nonce - RxNonce never returns a hash, only a nonce (see plan 2.3),
      // so this recomputation is the only way to get a hash value at all,
      // not a correctness shortcut.
      return sha256d(mined[:]), nil
  }
  ```

  Reuse the existing `sha256d` helper already defined in
  `internal/driver/device/stratum_server.go:92-95` (same package, already
  in scope - do not redefine it).

  `pollForDirectNonce` should be a tightened version of the existing
  `pollForNonce`/`parseRxNonceResponse` pattern (`controller.go:926-1042`),
  but simpler since it doesn't need the `originalInput`/software-recompute
  detour that `parseRxNonceResponse` does - `ParseRxNonce`
  (`controller.go:1355`) already gives back a bare nonce, which is all
  that's needed here. Tighten the poll interval for this path specifically
  (see 3.1's next bullet) - do not change the existing `PollInterval`
  constant (`controller.go:44`, 40ms), since that's still used by
  `ComputeBatch`'s unrelated raw-device loop; add a new
  `directPollInterval = 2 * time.Millisecond` (or similar - tune against
  real hardware; see §5) and `directPollTimeout = 500 * time.Millisecond`
  constants instead, scoped to this new path.

- Add a `directASICWorkID` helper (or an atomic counter field on `Device`)
  so successive `TxTask` packets use distinct `work_id` bytes, matching how
  `RxNonce` responses are correlated back to their request
  (`ParseRxNonce`'s `workID` return value isn't currently checked by the new
  method above for simplicity - since this path is strictly sequential
  (one in-flight request at a time, no concurrent `ComputeHash` calls from
  the same jitter pass), a rolling `uint8` counter is sufficient; do not
  over-engineer this into a request-matching table).

- **Concurrency note:** the jitter engine calls `ComputeDoubleHash`
  sequentially within one header's 21 passes, but `Execute21PassLoopBatch`
  itself just loops `Execute21PassLoop` per header sequentially too (2.1) -
  so at the `Device` level, direct-mode `ComputeHash` calls arrive strictly
  one at a time from a single seeder process. `d.mu` around `SendPacket`
  already exists in the codebase's other call sites for the same reason;
  keep using it, don't introduce new locking primitives.

### 3.2 Client side: opt-in `direct` hash method

**`pkg/hashing/methods/asic/asic_client.go`**

Add two new methods that do the real gRPC round trip - do not modify
`ComputeHash`/`ComputeDoubleHash` (2.4):

```go
// ComputeHashRemote always calls the server's ComputeHash RPC, never
// computing locally. Used only by DirectASICHASHMethod - see its doc
// comment for why this is a distinct method from ComputeHash rather than a
// mode flag on it.
func (c *ASICClient) ComputeHashRemote(data []byte) ([32]byte, error) {
    // same shape as the pre-existing real-RPC implementation this session
    // replaced in ComputeHash - a ComputeHashRequest/ComputeHashResponse
    // round trip via c.invoke, see asic_client.go:170 (invoke) and the
    // proto's ComputeHashRequest{Data: data} / ComputeHashResponse{Hash}.
}
```

Do not add a `ComputeDoubleHashRemote` that makes two round trips. Per 2.3,
one `ComputeHashRemote` call on an 80-byte header already gets back a real
`sha256d` from a direct-mode server (§3.1's `computeHashDirectASIC` always
double-hashes). Two round trips would be both wrong (double-double-hashing a
value that's already been through the ASIC's own internal double-SHA256)
and needlessly slow.

New file **`pkg/hashing/methods/asic/direct_hash_method.go`**:

```go
package asic

// DirectASICHASHMethod implements jitter.HashMethod by always calling the
// remote ComputeHash RPC via ASICClient.ComputeHashRemote - never the
// client's own local-software ComputeHash/ComputeDoubleHash. Correctness
// does not depend on the connected hasher-server actually running in
// -direct mode: if it isn't, the server just returns a plain
// sha256.Sum256(data) instead of a real ASIC round trip (Device.ComputeHash's
// existing default behavior), so this degrades to "slow, correct, network
// round trip per pass" rather than failing - the speed and hardware-backed
// property both come specifically from pairing this with a -direct server.
type DirectASICHASHMethod struct {
    client *ASICClient
}

func (a *DirectASICHASHMethod) ComputeHash(data []byte) ([32]byte, error) {
    if a.client == nil {
        return [32]byte{}, fmt.Errorf("ASIC client not available")
    }
    return a.client.ComputeHashRemote(data)
}

// ComputeDoubleHash issues exactly one remote call - see this file's
// package-level note on why two round trips would be wrong, not just slow.
func (a *DirectASICHASHMethod) ComputeDoubleHash(data []byte) ([32]byte, error) {
    if a.client == nil {
        return [32]byte{}, fmt.Errorf("ASIC client not available")
    }
    return a.client.ComputeHashRemote(data)
}
```

**`pkg/hashing/methods/asic/device.go`**

- Add `direct bool` field to `ASICMethod` (near `client`, `device.go:13`).
- Add `func NewDirectASICMethod(address string) *ASICMethod` that calls the
  existing `NewASICMethod(address)` and then sets `direct: true` on the
  result before returning it. **Do not change `NewASICMethod`'s existing
  signature or behavior** - this must be strictly additive so every
  existing call site (`factory.go`, `hasher_wrapper.go`) is unaffected.
- In `Execute21PassLoop` (`device.go:222-251`), `Execute21PassLoopBatch`
  (`device.go:254-277`), and `ExecuteRecursiveMine` (`device.go:301-326`) -
  the three places that currently do
  `jitterEngine.SetHashMethod(&ASICHASHMethod{client: m.client})` - branch:

  ```go
  if m.direct {
      jitterEngine.SetHashMethod(&DirectASICHASHMethod{client: m.client})
  } else {
      jitterEngine.SetHashMethod(&ASICHASHMethod{client: m.client})
  }
  ```

**`pipeline/3_DATA_SEEDER/pkg/simulator/hasher_wrapper.go`**

- `methodTypeForDevice` (`hasher_wrapper.go:83-95`) does substring matching
  in this order: `software`, `asic`, else `auto`. A naive new value like
  `"asic-direct"` would be caught by the existing `Contains(deviceType,
  "asic")` case before ever reaching a new check - **add the `direct` case
  before** the `asic` case:

  ```go
  case strings.Contains(deviceType, "direct"):
      return "direct"
  case strings.Contains(deviceType, "asic"):
      return "asic"
  ```

- `createHashMethod` (`hasher_wrapper.go:98-137`): add a `case "direct":`
  branch calling a new `createDirectASICMethod()` (mirror of
  `createASICMethod`, `hasher_wrapper.go:139-154`, but calling
  `asic.NewDirectASICMethod(address)` instead of `asic.NewASICMethod(address)`).
  Do **not** add `"direct"` to the `"auto"` case's fallback chain
  (`hasher_wrapper.go:115-131`) - direct mode must always be explicit, never
  silently auto-selected, since it has real operational side effects
  (implies the connected hasher-server should be running with `-direct` for
  any benefit, and changes what's physically happening on the device).

**`pipeline/3_DATA_SEEDER/cmd/data-seeder/main.go`**

- Update the `-hash-method` flag's help text (`main.go:40`) to document the
  new `direct` value: `"Hash method to use: auto, asic, direct, software, cuda"`.
  No other change needed here - `simConfig.DeviceType =
  fmt.Sprintf("hasher_%s", *hashMethod)` (`main.go:318`) already threads
  whatever string is passed straight through to the wrapper.

### 3.3 Deployment wiring

**`internal/analyzer/deployer.go:854`**

The remote start command needs a conditional `--direct` flag, sourced from
wherever this deployment flow's own config/flags currently decide things
like `--trace`. Find the equivalent decision point for an existing boolean
deploy-time option (`--trace=true` is unconditional today, so look instead
at how `d.config` or the deployer's constructor exposes toggles - do not
invent a new configuration mechanism if one already exists for something
comparable elsewhere in this file). Thread a `DirectMode bool` through
`DeploymentConfig` (wherever that struct is defined - likely
`internal/host/deployment.go`, given that's where `ConnectTimeout` and
similar knobs live per this session's prior work) and conditionally append
`--direct` to the format string when set.

This is lower priority than 3.1/3.2 - the new benchmark program (§4) does
not need it (it talks to a manually-started `hasher-server -direct` for
now). Implement it, but validate 3.1-3.2 end-to-end first.

## 4. New standalone seeding benchmark program

**Location:** `pipeline/3_DATA_SEEDER/cmd/seed-bench/main.go` - must live
inside `pipeline/3_DATA_SEEDER` to share its `data-seeder` Go module and
import its internal packages (per this repo's convention: each
`pipeline/N_*` stage is a fully separate module, no cross-package imports -
see this repo's root `CLAUDE.md`).

**Purpose:** exercise exactly `EvolutionaryHarness.EvaluatePopulation` /
`EvaluatePopulationBatch` against a single synthetic target token, with no
corpus loading, no checkpoint DB, no seed-writer, no KNIRVBASE submission,
no config file - so a full run starts in well under a second and every
generation's cost is 100% attributable to the hash method under test. This
directly mirrors what `data-seeder/cmd/data-seeder/main.go`'s
`runSyntheticTraining`/`trainToken`/`createTokenMap` already do for the
"no training data available" fallback path (`main.go:489-524`,
`main.go:720-801`) - **reuse that pattern, don't reinvent it.**

**Flags:**

```
--device-addr string   hasher-server gRPC address, e.g. 192.168.1.50:8888 (required unless --hash-method=software)
--hash-method string   auto | asic | direct | software  (default "software")
--population int       (default 256, matches data-seeder's default)
--generations int      (default 50 - deliberately much lower than data-seeder's
                        2000 default; this program is for latency/throughput
                        measurement per generation, not for actually finding
                        a winning seed)
--difficulty-bits int  (default config.DefaultDifficultyBits, reuse the constant)
--target-token int32   (default 42, or any fixed synthetic value)
--verbose bool
```

**What it does:**

1. Set `DEVICE_IP` env var from `--device-addr` if provided (matches
   `asicAddressFromEnv()`'s existing convention,
   `hasher_wrapper.go:156-165` - don't invent a second config surface for
   the same address).
2. Build a `simulator.SimulatorConfig{DeviceType: fmt.Sprintf("hasher_%s", *hashMethod), ...}`
   exactly as `data-seeder/main.go:317-324` does, and construct/initialize a
   `simulator.NewHasherWrapper(simConfig)` the same way.
3. Build one synthetic `training.TrainingRecord` and one
   `training.NewSeedPopulation(...)`, exactly mirroring
   `data-seeder/main.go:731-743`'s `trainToken` synthetic path.
4. Construct one `training.NewEvolutionaryHarness(*population)`, set its
   difficulty mask exactly as `data-seeder/main.go:415-427` does.
5. Loop `*generations` times calling `harness.EvaluatePopulation(...)`,
   timing each call individually. Do **not** implement win-detection/early
   exit logic - this program measures raw per-generation cost, not
   convergence; always run the full requested generation count.
6. Print per-generation latency (min/max/mean/p50/p95 across the run) and a
   final summary: total generations, total wall time, effective
   `ComputeDoubleHash` calls/sec (= generations × population × 21 / total
   seconds). This last number is the one directly comparable across
   `--hash-method=software` vs `direct` runs and is the actual answer to
   "is this faster."
7. Exit non-zero with a clear message if `HasherWrapper.Initialize` fails
   (e.g. can't reach `--device-addr`) - don't silently fall back to
   software when a hardware method was explicitly requested; that would
   defeat the program's purpose of measuring exactly what was asked for.

**Explicitly out of scope for this program:** checkpoint persistence, seed
write-back, config file loading, KNIRVBASE submission, multi-token runs,
epoch semantics. If any of these would be needed later, extend
`data-seeder`'s real pipeline, not this benchmark tool - keep this tool
single-purpose.

## 5. Validation plan (live hardware, sequential milestones)

This mirrors how the stratum/CGMiner work in this same codebase was
actually validated - do not skip straight to a full run.

1. **Build check first.** `cd pipeline/3_DATA_SEEDER && go build ./...` and
   `cd <repo root of KNIRVHASHER> && go build ./...` after each of §3.1,
   3.2, and 4 - do not batch all changes before the first build attempt.
2. **Server smoke test without hardware dependency.** Run
   `hasher-server -direct` locally (or on the device) and confirm from logs
   that: `ensureCGMinerRunning` is never called, Strategy 0 is skipped, and
   the process either opens the raw USB device or fails clearly (not
   silently falls through to CGMiner - there should be no CGMiner fallback
   path in direct mode at all, by design of §3.1).
3. **Single-call correctness, no seeder involved.** Add a temporary manual
   test (or reuse `cmd/stratum-pool-test`'s pattern of a small throwaway
   `main.go`) that connects an `ASICClient` in `direct` mode to a running
   `hasher-server -direct` and calls `ComputeHashRemote` once with a
   deliberately-constructed 80-byte header. Confirm: a real `TxTask` is
   logged sent, a real `RxNonce` is logged received, and the returned hash
   actually verifies as `sha256d` of that header with the returned nonce
   substituted in (recompute independently in the test to check, exactly
   like `stratum_server_test.go`'s existing pattern for the stratum path).
4. **`seed-bench --hash-method=software` baseline.** Run first to get a
   reference per-generation latency with zero hardware/network involvement.
5. **`seed-bench --hash-method=direct` against real hardware.** Compare.
   Expect real hardware involvement to be audibly confirmable (fan
   spin-up, as observed earlier this session for the CGMiner path) and
   substantially slower than software per §5's own honest estimate below -
   that is expected, not a bug; the goal is genuine hardware use, not
   beating software.
6. **Tune `directPollInterval`** (§3.1) based on step 5's actual measured
   latency distribution - start at 2ms, adjust based on what the real
   round-trip floor turns out to be (USB bulk transfer time dominates, per
   `docs/BITMAIN-PROTOCOL.md`'s own "Latency: 1ms" figure for this exact
   device/driver combination).

**Honest expected performance (do not be surprised by this):** each
`ComputeHashRemote` call in direct mode costs one gRPC round trip plus one
USB bulk write + poll-read cycle. Per `docs/BITMAIN-PROTOCOL.md`'s own
measured figures for this device (1ms USB latency, `EasyTarget` making
search time negligible), expect single-digit-to-low-double-digit
milliseconds per call once polling is tuned. At 5,376 calls/generation
(§2.1), that's roughly 15-60+ seconds per generation - far slower than
software (microseconds), genuinely hardware-backed, and imperceptibly
sequential per-step. This tradeoff was already discussed with and accepted
by the requester; don't attempt to "optimize past" software's speed - that
was never the goal (see §1).

## 6. Safety note - do not skip

Direct mode means CGMiner's own thermal/fan control loop is not running for
the duration. `USBDevice.Initialize()` (`usb_device_mips.go:488-576`, or
its non-MIPS twin) sends one `buildTxConfigPacket` at startup with a fixed
fan PWM (`0x60`/96%, `usb_device_mips.go:600`) and no ongoing supervision
loop after that. Before this mode is used for any extended/unattended run
(the actual seeder pipeline, as opposed to short `seed-bench` runs), add a
periodic `RxStatus` poll that checks reported temperature (the
`bitmain_rxstatus_data.temp[]` field, per `docs/BITMAIN-PROTOCOL.md`'s
packet struct) and aborts/backs off if it exceeds the documented thresholds
(`docs/BITMAIN-PROTOCOL.md`'s "Temperature Management" table: overheat
60°C, critical shutdown 80°C). This plan's §3.1 scope does not include
building that supervisor - flag it explicitly to whoever runs the first
extended direct-mode session, and do not leave direct mode running
unattended until it exists.

## 7. Explicit non-goals

- Do not modify `ComputeBatch`, `computeBatchCGMiner`, `buildTxTaskPacket`,
  or anything under the CGMiner/stratum path (`stratum_server.go`,
  `cgminer_*.go`). None of it is affected by or relevant to this plan.
- Do not modify `ASICClient.ComputeHash`/`ComputeDoubleHash`'s existing
  always-local behavior (§2.4) - add new methods alongside them.
- Do not modify `NewASICMethod`'s signature - add `NewDirectASICMethod`
  alongside it.
- Do not add `"direct"`/`"asic-direct"` to any `"auto"` fallback chain -
  direct mode is always an explicit, deliberate choice.
- Do not attempt to make direct mode and CGMiner mode coexist in the same
  `hasher-server` process/run. They are mutually exclusive per run, by the
  device's own USB exclusivity constraint, not an artificial restriction
  this plan is imposing.

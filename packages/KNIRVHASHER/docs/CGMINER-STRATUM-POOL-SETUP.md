# CGMiner Stratum Pool Setup (Antminer S3)

How `hasher-server`'s CGMiner strategy gets real ASIC hash work, and how to
configure a device for it. Written after getting this working end-to-end on
a real Antminer S3; the steps below are the ones that were actually needed,
not a theoretical checklist.

## Background

`hasher-server` (Strategy 0 in `internal/driver/device/controller.go`) does
not talk to the ASIC chips directly. It talks to the CGMiner process that's
already running on the device, by acting as a **local Stratum V1 mining
pool** that CGMiner connects to (see `internal/driver/device/stratum_server.go`).
CGMiner mines real work against that pool exactly like it would against any
other pool (e.g. `stratum.antpool.com`), and `ComputeBatch`/`MineWork`
submit real jobs and get back real, independently-reverified nonces/hashes.

This only works if two separate things are both true:

1. **CGMiner is pointed at our pool.** `startCGMinerStratumPool` tries this
   automatically via CGMiner's RPC API (`addpool`/`switchpool`), but on at
   least this device's stock `/etc/init.d/cgminer` build, those RPCs return
   `"Access denied"` regardless of anything on our end - the API's write
   privilege group doesn't extend to us. **You need to add the pool by hand
   through the device's own web GUI instead** (see below). This is a
   one-time step per device.
2. **The ASIC chips are actually being driven by CGMiner**, not just
   detected. See "The PIC bootloader trap" below - this is the one that
   actually mattered.

## Adding the pool via the GUI

Open the device's web UI (`http://<device-ip>/`) → **Miner Configuration**.
Add a pool entry:

| Field | Value |
|---|---|
| URL | `127.0.0.1:18332` |
| Worker | `knirvhasher` |
| Password | `knirvhasher` |

These match `stratumFixedPort`/`stratumFixedUser`/`stratumFixedPass` in
`internal/driver/device/stratum_server.go` - fixed rather than randomized
per run specifically so a one-time GUI entry keeps working across every
future `hasher-server` restart. The password carries no real security
weight (the server's `mining.authorize` handler doesn't actually check it);
this pool is only ever reachable on loopback anyway.

You don't need to remove any existing pool (e.g. a real pool like
`stratum.antpool.com`) - CGMiner can have both. Click **Save & Apply**,
which restarts CGMiner and makes it re-evaluate all configured pools fresh.

## The PIC bootloader trap

This is the one that cost the most time to find, so it's worth spelling
out. Everything above can be configured perfectly and CGMiner will still
report `0.00 GH/s` with **zero** ASIC chains detected
(`AntMiner` section on the **Miner Status** GUI page shows "This section
contains no values yet" for Chain#/ASIC#/Temp *and* for Fan#) - with no
error anywhere suggesting why.

Check the **Kernel Log** GUI page for this line near the top of boot:

```
bitmainbl 1-1.1:1.0: USB Bitmain bootloader device now attached to USB Bitmain bootloader-0
PIC microchip bootloader usb OK
```

If you see `bitmainbl`/"bootloader" rather than an operational ASIC driver
attaching, the hash board's PIC microcontroller (the chip that actually
drives the BM1382 ASIC array) is sitting in bootloader mode - it never had
its operational firmware uploaded. CGMiner, the kernel module, and
everything else can be perfectly healthy and this will still produce zero
hashrate, because nothing downstream of CGMiner's stratum layer is actually
computing anything.

**Fix:** in the GUI, there's a "PIC update" option (disabled by default on
at least this unit) - **enable it and Save & Apply**, then reboot the
device. This corresponds to `/etc/init.d/pic-update`, a script (present,
correctly configured, pointing at a valid firmware blob at
`/etc/config/miner_pic.hex`) that was simply never linked into
`/etc/rc.d/` at boot - so it existed but never ran. Enabling it via the GUI
fixes that persistently. After the reboot, the Miner Status page should
show real chain/ASIC/fan entries and a nonzero hashrate, and you should
physically hear the fans spin up under load.

If your device doesn't expose a GUI toggle for this, the equivalent manual
step is running `/etc/init.d/pic-update start` over SSH (uploads
`/etc/config/miner_pic.hex` to the PIC via `/usr/sbin/pic-update`) - but
prefer the GUI path if it's available; it's the device's own sanctioned
mechanism for this and more likely to interact correctly with everything
else than a hand-invoked script.

## Verifying it's actually working

Two independent things to check, since a clean stratum handshake
(subscribe/authorize succeeding) does **not** by itself mean CGMiner is
routing real hash time to the pool:

- **Miner Status GUI page**: nonzero `GH/S(5s)`/`GH/S(avg)`, real
  Chain#/ASIC#/Fan# entries, and `Diff1#` incrementing on the pool you
  care about.
- **`cmd/stratum-pool-test`** (see below): watch for real `mining.submit`
  messages and `JOB N SOLVED` log lines - each one is independently
  reverified against the target server-side, not just trusted from
  CGMiner's own claim.

A pool showing `"Stratum Active":false` in CGMiner's own `pools` API
response, even while connected and subscribed, means CGMiner isn't
currently routing real hash time to it - normal for a moment after
(re)connecting, or if a higher-priority pool is currently the one CGMiner
considers active.

## Standalone diagnostic tool

`cmd/stratum-pool-test` is a small, self-contained program that runs the
same `StratumServer` code `hasher-server` uses, without touching
deployment, CGMiner's config, or the Antminer's filesystem at all. Useful
for testing pool connectivity/mining independent of the full
deploy-to-device flow:

```sh
go run ./cmd/stratum-pool-test --difficulty=1024
```

Point CGMiner's pool config at `<this-host's-LAN-IP>:18332` (worker/pass
`knirvhasher`/`knirvhasher` by default) and watch for `JOB N SOLVED` lines.
See `--help` for tunable difficulty/timeouts - useful for testing whether a
given firmware's difficulty floor (if any) rejects very easy targets.

## What is and isn't ASIC-backed

Worth being precise about, since it's easy to assume "ASIC Hardware mode is
selected" means every hash computation touches real silicon - it doesn't:

- **`ComputeBatch` / `MineWork`** (via CGMiner mode): real, hardware-backed.
  Each item is a genuine stratum job CGMiner mines and submits a real,
  independently-reverified nonce for.
- **`ComputeHash` / `ComputeDoubleHash`** (single-item): **always software**
  (`crypto/sha256`), even when CGMiner/ASIC mode is otherwise active. A
  SHA-256 ASIC cannot compute an arbitrary caller-specified hash on demand -
  it can only search a mining header for a nonce satisfying a target. Code
  that needs "the literal hash of exactly this data" (e.g. the seeder
  pipeline's per-candidate evolutionary search, which calls
  `ComputeDoubleHash` many thousands of times) gets a real answer, but
  never touches the ASIC - selecting `--hash-method=asic` for that kind of
  workload only adds a needless network round trip per call versus just
  using `--hash-method=software` directly. If you don't hear the fans
  spin up despite "ASIC Hardware" being selected, this is almost certainly
  why: the workload in question never routes through `ComputeBatch` at all.

# KNIRVCONTROLLER ↔ KNIRVSERVER/KNIRVCLI Integration Plan

Tracks what's needed to make KNIRVCONTROLLER (the mobile app) a real participant in the
network rather than a standalone client — starting with the flagship use case: **a DVE
session started from KNIRVCLI can be joined from a phone, so the user can send and
receive text with the active agent(s) in that session while away from the terminal.**
Also captures other meaningful ways to use KNIRVCONTROLLER across the suite, grounded in
infrastructure that already exists today.

This doc follows the same pattern as `controller_refactor.md`: state what's already
shipped, be honest about what's mocked vs. real, then lay out phased work.

---

## Part 1: CLI-Initiated DVE Session ↔ Mobile Chat

### 1.1 What already exists (verified in code)

**KNIRVCONTROLLER (mobile/web)**
- `src/react-app/pages/AgentChat.tsx` — a fully-built per-DVE chat UI (`/dves/:dveId/agent`)
  using `ChatThread.tsx` + `VoiceChatBar.tsx`. **It is 100% mocked** — `mockDVEs` and
  `generateMockResponse()` produce canned replies with a `setTimeout` to fake streaming.
  There is no backend call in this page at all today.
- `src/react-app/pages/CognitiveEngineChat.tsx` — same shared chat components, root-only.
- Native QR scanning already works end-to-end: `capacitor-barcode-scanner`,
  `src/react-app/platform/nativeQrScanner.ts`, `QrScannerFrame.tsx`, used today by
  `Onboarding.tsx` and `Scanner.tsx`.
- `src/worker/index.ts` — a thin Cloudflare Worker proxy in front of KNIRVSERVER. Today it
  only forwards four plain HTTP routes (`/worker/dve/list`, `/worker/dve/creations`,
  `/worker/cognitive/status`, `/worker/cognitive/chat` → `/api/knirvshell/execute`). **No
  WebSocket route exists.**
- Vault/DVE-vault identity system (`useVault.ts`, HD derivation) is real and already signs
  locally — relevant later for authenticating a pairing.

**KNIRVAGENT (the process that supervises a DVE)**
- `pkg/bus` + `pkg/channels` is a genuine pluggable channel-bus: `InboundMessage{Channel,
  SenderID, ChatID, Content, SessionKey, Metadata}` / `OutboundMessage{Channel, ChatID,
  Content}`, with Telegram, Discord, Slack, WhatsApp, and a `dve` channel all implementing
  the same `Channel` interface (`Name/Start/Stop/Send/IsRunning/IsAllowed`). Telegram's
  channel (`pkg/channels/telegram.go`) is the best reference implementation: it does
  pairing (`/start` claims the bridge), an allowlist, and bidirectional message relay.
- `pkg/channels/dve.go` already exists but is **outbound-only**: `Send()` POSTs the
  agent's reply to `POST /api/v1/dve/{dveID}/agent/response` on KNIRVSERVER (a real,
  routed endpoint — `internal/web/dve_handlers.go:268`, `PostAgentResponse`). `Start()`
  just flips a running flag; there is no receive-side logic, so nothing today delivers an
  externally-typed message *into* this channel's inbound side.
- This channel is only initialized when the `DVE_ID` env var is set (`manager.go`), i.e.
  one instance runs per DVE container/pod, alongside a `terminal` channel for local piping.

**KNIRVSERVER**
- A full "Controller Integration" feature already exists server-side —
  `internal/web/controller_integration_handlers.go` (635 lines) +
  `internal/services/controllerintegration/controller_integration_service.go` (955 lines)
  in the KNIRV_CORP backend, routed under `/api/controller-integration/*`: QR generation,
  QR scan, pairing confirm, session get/list/delete, `POST .../sessions/{id}/messages`,
  `.../commands`, `.../capabilities`, `.../notifications`. The KNIRVCONTROLLER-facing
  frontend hook (`use-controller-integration.ts`) and QR component (`qr-code-display.tsx`,
  literally says *"Open KNIRVCONTROLLER app on your mobile device"*) live in
  `packages/KNIRVSERVER/frontend/src/components/controller/`.
  **However, this is a generic desktop-pairing model** (`ControllerSession` has
  `desktop_id` / `mobile_device_id`, capabilities like `remote_control`, `file_transfer`,
  `screen_share`) — it has **no DVE or session linkage field at all**. It's real
  infrastructure, but for a different use case (remote-controlling a KNIRVSERVER
  dashboard), not "attach to a specific running DVE session."
- The dashboard's SSH panel (`console-panel.tsx`) already does the pattern we want to
  copy for chat: `POST /api/dve/{nodeId}/ssh-session` → server returns `{ ws_url }` →
  browser opens `new WebSocket(...)`. This is the right shape for a "DVE session chat
  socket" endpoint.

**KNIRVCLI**
- `knirv session start [workspace]` / `status` / `commit` (deprecated) / `verify` —
  `cmd/dve.go`, backed by `internal/dvesessions` + `dve/` packages. This is a
  **local-first, audit-oriented** construct: it writes `.knirv/sessions/<id>/`,
  `session.json`, an append-only hash-chained `eventlog.jsonl`, and produces a signed
  evidence bundle on commit (see `DVE_Alignment_Plan.md` — vocabulary is "supervised
  session," "evidence bundle," "permission decision," not "chat room"). The
  server-side call is `dvesessions.Client.CreateSession()` → `POST /dve/sessions`,
  returning `{ID, EnvironmentID, Status, ...}`.
- Confusingly, **"session" means three different things** in this codebase today, and
  picking the right one matters for this feature:
  1. `dvesessions.Session` — the CLI's supervised dev/compute session (above).
  2. The DVE **evidence-ingestion** session — `POST /api/dve/:dve/sessions/ingest`, `GET
     /api/dve/:dve/sessions/:session[/evidence|/proof|/report]`
     (`internal/dveevidence/ingest.go`) — an audit-trail record keyed by dve+session,
     unrelated to live interaction.
  3. `internal/uievents.Session` — the CLI TUI's own PTY/workspace session model
     (`internal/ui/screens/neon_shell_test.go`, `chathistory` store), used to render the
     multi-agent terminal locally. It already has a `chathistory.Store` with
     `DVESessionID` + `Slot` fields recording PTY input/output as chat records — i.e. the
     CLI *already* logs agent conversation locally in a chat-shaped format
     (`internal/chathistory/store_test.go`).

**Conclusion:** the pieces for a text bridge exist in spirit on all three ends (a
channel-bus on the agent side, a chat UI shell on the mobile side, a WebSocket-session
pattern on the server side, and a "session already logs chat-shaped data" habit on the
CLI side) but none of them are connected, and none of the three "session" types is
currently addressable from a phone.

### 1.2 Gap list

1. No unified, phone-addressable identity for "the DVE session the CLI just started."
2. No inbound transport into KNIRVAGENT's bus for an externally-typed message — the `dve`
   channel is Send-only.
3. No WebSocket (or SSE) route in the Cloudflare Worker — only plain proxied HTTP.
4. `AgentChat.tsx` has zero backend wiring; it needs to be pointed at something real.
5. No pairing mechanism ties "a CLI process on someone's laptop" to "a phone." The
   existing Controller Integration pairing is desktop-dashboard-scoped, not
   session-scoped.
6. Mobile-originated input has no place in the CLI's evidence/trust model yet
   (`DVE_Alignment_Plan.md`'s trust levels — unsupervised / locally supervised / signed
   supervised / hardware-backed supervised / server-executed deterministic — don't
   mention a mobile origin at all).

### 1.3 Proposed design

**a) KNIRVCLI — mint a pairing token for the active session**
Extend `cmd/dve.go`'s `session` command group with `knirv session pair` (or a `--pair`
flag on `session start`) that:
- Reads the active session via the existing `ActiveSessionPath` / `LoadSession` helpers
  in `dve/session_helpers.go`.
- Calls a new server endpoint to mint a short-lived pairing token bound to
  `{session_id, environment_id, user_id}`.
- Prints both a copy-pasteable code and a terminal QR (reuse the same
  `session_id`/`expires_at`/`capabilities` shape already defined in
  `use-controller-integration.ts`'s `QRCodeData` so the payload format is consistent
  across the two pairing features).

**b) KNIRVSERVER — a session-scoped chat transport**
Do **not** bolt this onto `ControllerSession` (wrong semantics, would conflate
"remote-control the desktop dashboard" with "chat with a DVE agent"). Instead, add,
alongside the existing `dvesessions`/`dveevidence` handlers:
- `POST /dve/sessions/{id}/pair` — mint the token described above (server side of 1.3a).
- `POST /dve/sessions/{id}/chat-socket` — mirrors `console-panel.tsx`'s SSH pattern:
  validates the pairing token, returns `{ ws_url }`.
- The WebSocket endpoint fans a single logical stream out to every connected party
  (KNIRVAGENT inside the DVE, and however many phones are paired), tagging each frame
  with `sender: "user" | "agent" | "mobile:<device_id>"` so all clients can render a
  single shared thread.

**c) KNIRVAGENT — a bidirectional `controller` channel**
Add `pkg/channels/controller.go` implementing `Channel`, modeled on `telegram.go` (which
already does pairing + allowlist + bidirectional relay) rather than on the current
`dve.go` stub:
- `Start()` opens the session's chat WebSocket (from 1.3b) and, on each inbound frame
  from a mobile sender, calls `c.HandleMessage(senderID, chatID, content, nil, metadata)`
  exactly like every other channel does — no bus changes needed.
- `Send()` writes the agent's `OutboundMessage` back onto the same socket.
- Register it in `manager.go` next to the existing `if os.Getenv("DVE_ID") != ""` block,
  since it's scoped the same way (one per DVE process).

**d) KNIRVCONTROLLER — join flow + real chat**
- Add a "Join Session" scan flow reusing the existing `nativeQrScanner.ts` /
  `QrScannerFrame.tsx` (same components `Onboarding.tsx`/`Scanner.tsx` already use) to
  read the CLI's QR and store `{session_id, pairing_token}`.
- Replace `AgentChat.tsx`'s `generateMockResponse()` with a real WebSocket client hitting
  the new worker route (below). Keep `ChatThread`/`VoiceChatBar` as-is — they're already
  transport-agnostic.
- Extend `src/worker/index.ts` with a WebSocket passthrough route
  (`app.get("/worker/dve/sessions/:id/socket", ...)`) — Cloudflare Workers can forward an
  `Upgrade: websocket` request directly via `fetch`, so this is a small addition, not a
  new proxy architecture.

**e) Fold it into the existing audit trail**
Every relayed message should append to the session's hash-chained `eventlog.jsonl`
(the same mechanism `dve/hashchain.go` and `chathistory.Store.Append` already provide) so
a mobile-originated command is evidenced exactly like a locally-typed one — not a side
channel that escapes the signed bundle on `knirv session commit`. This is the one place
where "genuinely new" data is entering the session, so it should get a trust-level tag
per `DVE_Alignment_Plan.md`'s vocabulary — see the open question below.

### 1.4 Suggested milestones

| # | Milestone | Depends on | Notes |
|---|---|---|---|
| M0 | Wire `AgentChat.tsx` to a **real** single agent call (drop the mock) over the already-proxied `/api/knirvshell/execute` route | nothing new | Fastest way to prove the chat UI end-to-end against a live backend, no pairing yet |
| M1 | CLI prints a pairing QR/code for the active `knirv session` | 1.3a | |
| M2 | KNIRVSERVER session-scoped pairing + chat-socket endpoints | 1.3b | |
| M3 | KNIRVAGENT `controller` channel (bidirectional) | 1.3c, M2 | |
| M4 | KNIRVCONTROLLER scan-to-join + live session chat | 1.3d, M2 | |
| M5 | Evidence-bundle integration for mobile-originated messages | 1.3e, M3 | |

---

## Part 2: Other meaningful KNIRVCONTROLLER enhancements

These lean on infrastructure that's already built (mostly server-side) but not yet
surfaced on mobile — each is a "wire it up" job more than a "build it" job.

- **Push notifications for DVE events.** `PostSendPushNotification` already exists on the
  Controller Integration API (`.../sessions/{id}/notifications`) but nothing in
  KNIRVCONTROLLER consumes it. Task completion, error-node creation, guardrail
  violations, and badge-mint results are natural triggers.
- **Mobile approval for permission decisions.** `DVE_Alignment_Plan.md` already defines
  "permission decision" as a first-class evidence record. A push notification +
  tap-to-approve/deny screen (vault-signed, using the existing HD key) turns every
  supervised-session permission prompt into something the user can act on away from
  their desk — the mobile equivalent of a GitHub/2FA approval push.
- **Badge Creation Lab companion view.** `badge-lab-panel.tsx` (desktop-only today) mints
  badges via `POST /api/knirvshell/chain/badge/create` / `.../badge/mint`. A read-only
  mobile status view (mint progress, badge attached to which DVE) would close the loop
  for a workflow that already exists.
- **Voice-driven session commands.** Voice packages are installed but `useVoiceIntegration`
  still simulates STT/TTS (per `controller_refactor.md`'s remaining-work list). Once M0
  above makes `AgentChat.tsx` real, the same voice pipeline gives hands-free session
  control ("show pending tasks", "run badge X") for free.
- **SSH companion/second-factor.** `console-panel.tsx`'s SSH-session WebSocket pattern
  (which this doc's chat-socket design deliberately mirrors) could let KNIRVCONTROLLER
  act as a read-only mirror of an active SSH debugging session, or as an out-of-band
  approval step before an SSH session is allowed to open.
- **Fleet digest notifications.** KNIRVCONTROLLER's DVE List view already shows every
  owned DVE sorted by reputation; a scheduled push ("3 DVEs have pending tasks", "DVE-Beta
  reputation dropped below threshold") reuses the same notification endpoint as above.
- **Wallet-triggered mobile confirmation.** Consistent with the sovereign,
  client-side-signing model in `Oracle_Contoller_Alignment_Plan.md`, NRN stake/transfer
  actions initiated elsewhere (CLI, dashboard) could push a confirm-on-phone step instead
  of trusting whatever device happens to hold the session.
- **Treat "controller" as a first-class channel, not a one-off.** KNIRVAGENT already
  treats Telegram/Discord/Slack/WhatsApp as peer channels with the same pairing +
  allowlist conventions (`pkg/channels/base.go`'s `IsAllowed`). Building the mobile bridge
  as a `Channel` (Part 1.3c) rather than a bespoke path means it inherits that scrutiny
  for free and stays consistent with how every other integration in this codebase works.

---

## Open questions (need a product decision, not an engineering guess)

1. **Which "session" does mobile actually pair to?** Recommend `dvesessions.Session` (the
   CLI's supervised session) as the primary key, since "initialized from the CLI first"
   is literally what that object is. The evidence-ingestion session and the TUI's
   `uievents.Session` should stay internal bookkeeping, not phone-addressable.
2. **Pairing auth strength.** A bare QR/token (like today's Controller Integration flow)
   is weaker than everything else in this codebase's security model (vault-signed UDCs,
   hash-chained evidence, ed25519-signed bundles). Recommend requiring the pairing
   confirmation itself be signed by the vault's derived key, not just "whoever scanned
   the code first."
3. **Trust tier for mobile input.** `DVE_Alignment_Plan.md`'s trust vocabulary
   (unsupervised → locally supervised → signed supervised → hardware-backed supervised →
   server-executed deterministic) has no slot for "typed on a paired phone." Recommend
   landing it at "locally supervised" by default, promotable to "signed supervised" once
   1.3e's vault-signature requirement (open question 2) is in place.

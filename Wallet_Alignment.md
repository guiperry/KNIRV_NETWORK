# Wallet Alignment: KNIRVCONTROLLER & Agentic NFT Wallet DVE Identity

**Version:** 1.0
**Date:** April 30, 2026
**Status:** Planning

---

## Overview

This document aligns the current `KNIRVWALLET` package with the Agentic NFT Wallet DVE Identity
paradigm introduced in the KNIRVBRIDGE Browser DVE implementation plan. It formalizes the rename
from `KNIRVWALLET` → `KNIRVCONTROLLER`, defines the multi-DVE wallet model, specifies the
supervisor KNIRVAGENT auto-initialization contract, and designs the voice/text chat surfaces for
both the root Cognitive Engine and per-DVE agents.

The SDD at `packages/KNIRVWALLET/SDD.md` already establishes the KNIRV-CONTROLLER vision. This
plan operationalizes it against the live codebase and the DVE identity work.

---

## 1. Package Rename: KNIRVWALLET → KNIRVCONTROLLER

### 1.1 What Changes

| Before | After |
|---|---|
| `packages/KNIRVWALLET/` | `packages/KNIRVCONTROLLER/` |
| `"KNIRV Wallet"` heading in `Wallet.tsx` | `"Vault"` |
| `WalletPage` component | `VaultPage` component |
| `useWallet` hook | `useVault` hook |
| Route `/wallet` | Route `/vault` |
| `window.knirv` provider (in KNIRVBRIDGE) | unchanged — provider identity is network-facing |

The Cloudflare Worker at `src/worker/index.ts` is renamed internally but its external route
bindings are unchanged (network-facing URLs are infrastructure concerns, not part of this plan).

### 1.2 What Does NOT Change

- The React/Vite/TypeScript stack stays.
- The voice control subsystem (`VoiceProcessor.ts`, `EdgeColoring.tsx`, `useVoiceIntegration.ts`)
  stays and is extended in Phase 4.
- The existing page routes (`/workflows`, `/scanner`, `/skills`, `/udc`, `/onboarding`) are
  unchanged.
- `DVENodeVerifier.tsx` is superseded by the new DVE List (Phase 3) but kept as an internal
  utility for diagnostics.

---

## 2. Identity Model: Vault + DVE Wallets

### 2.1 Core Concept

```
KNIRVCONTROLLER
├── Vault  (the "primary wallet" — renamed from KNIRV Wallet)
│     └── NRN balance, signing keys, HD root, transaction history
│
└── DVE Wallets  (one per owned DVE Creation)
      ├── DVE-A  →  KNIRVAGENT-A (supervisor)
      ├── DVE-B  →  KNIRVAGENT-B (supervisor)
      └── DVE-C  →  KNIRVAGENT-C (supervisor, browser-extension TEE)
```

The **Vault** is the user's master financial account. It holds NRN, signs transactions, and
manages stake. It is never itself a DVE node.

**DVE Wallets** are derived child identities — each is cryptographically linked to the Vault's
HD root at a dedicated derivation path, but has its own address, stake pool, and badge inventory.
Each DVE Wallet maps 1:1 to a `DVECreation` record on KNIRVSERVER. The DVE Wallet's address
is stored in `DVECreation.OwnerAddress` and `DVENode.WalletAddress`.

### 2.2 HD Derivation Path

| Identity | BIP44 Path |
|---|---|
| Vault | `m/44'/60'/0'/0/0` (existing root account) |
| DVE Wallet N | `m/44'/60'/1'/0/N` (new branch, N = DVE index) |

Using a separate account branch (`account=1'`) keeps DVE keys strictly separated from the Vault
keys. The Vault retains the ability to authorize DVE wallet operations via cross-signing.

### 2.3 DVE Wallet Lifecycle

```
User creates DVE  →  KNIRVCONTROLLER derives DVE Wallet at path N
                  →  DVE Wallet address registered as DVECreation.OwnerAddress
                  →  Supervisor KNIRVAGENT provisioned automatically (Phase 5)
                  →  DVE Identity NFT minted on KNIRVCHAIN (from KNIRVBRIDGE plan)
                  →  DVE Wallet appears in DVE List (Phase 3)
```

DVE Wallets are deleted (zeroed, not destroyed — HD keys are deterministic) when a DVE is
decommissioned. The on-chain DVE Identity NFT remains as a permanent record.

---

## 3. DVE List View

### 3.1 New Route: `/dves`

**New file: `src/react-app/pages/DVEList.tsx`**

The DVE List is the primary new screen in KNIRVCONTROLLER. It replaces the current
`DVENodeVerifier.tsx` as the entry-point route (`/`).

#### Layout

```
┌─────────────────────────────────────────────────────────────┐
│  KNIRVCONTROLLER          [Vault: 42,000 NRN]   [⚙] [🎤]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  MY DVEs                                    [+ New DVE]    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 🟢 DVE-Alpha                    sgx · us-east-1     │   │
│  │    g1abc...def    12,000 NRN staked   Rep: 847      │   │
│  │    Badges: [policy-check] [reasoning] [skill-lint]  │   │
│  │    Agent: KNIRVAGENT-Alpha  [Chat ▶]                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 🟡 DVE-Beta                 browser-ext · local     │   │
│  │    g1xyz...uvw     1,000 NRN staked   Rep: 203      │   │
│  │    Badges: [signature-verify]                        │   │
│  │    Agent: KNIRVAGENT-Beta   [Chat ▶]                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ─────────────── NETWORK DVEs (discoverable) ─────────────  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 🔵 Demo DVE Node 3 (TDX)         tdx · eu-central   │   │
│  │    Not owned · Rental available · Rep: 1000          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### DVE Card Data

Each card is backed by:
- `DVENode` object from `GET /api/dve/nodes` (KNIRVSERVER)
- `DVECreation` object (owned DVEs only) from `GET /api/dve/creations?owner={vaultAddress}`
- Badge inventory from wallet's NFT holdings filtered by `dve_badge: true` metadata
- Live status dot via the WebSocket relay established in KNIRVBRIDGE plan (Phase 2)

#### Sections

1. **My DVEs** — DVEs where `DVECreation.OwnerAddress` matches any DVE Wallet derived from the
   current Vault. Sorted by reputation score descending.
2. **Network DVEs** — DVEs discovered via `GET /api/dve/nodes` that are not owned. Shown in a
   collapsed, read-only list for browsing and rental. Paginated, max 10 shown by default.

### 3.2 DVE Detail Slide-Over

Tapping a DVE card slides out a detail panel (using the existing `Panel Manager` pattern from
`Layout.tsx`) with:
- Full DVE node metrics (CPU, memory, latency, uptime)
- Badge inventory for this DVE Wallet
- Policy rules derived from attached badges
- Task history (pending / completed / failed)
- Direct link to the DVE's agent chat (opens `AgentChat`, Phase 4)
- Stake management (add/reduce NRN from Vault)
- Decommission DVE action (guarded by confirmation)

### 3.3 New DVE Creation Flow

**New file: `src/react-app/pages/DVECreate.tsx`**

Steps:
1. **Name & Type** — user picks a name and TEE type (`hardware` or `browser-extension`)
2. **Stake** — user specifies NRN amount from Vault to stake (validated against badge requirements)
3. **Badges** — user selects which NFT badges to attach from their Vault badge inventory
4. **Confirm** — shows derived DVE Wallet address, capabilities summary, and estimated reputation
5. **Submit** — calls `POST /api/dve/creations` (KNIRVSERVER) + mints DVE Identity NFT

After submit, the supervisor KNIRVAGENT is provisioned automatically (Phase 5).

---

## 4. Voice & Text Chat Surfaces

Two distinct chat channels are introduced. They share UI components but differ in who they
connect to and what access level is required.

### 4.1 Cognitive Engine Chat (Root Nodes Only)

**Context:** The `CognitiveEngine` at
`packages/KNIRVSERVER/backend/internal/services/cognitiveengine/cognitive_engine.go` is only
active on root nodes (those with `packages/KNIRVSERVER/bin/root.key` loaded). This chat surface
is only available when the connected KNIRVSERVER instance is a root node.

**New file: `src/react-app/pages/CognitiveEngineChat.tsx`**
**New route:** `/cognitive`

#### Access Gate

On load, KNIRVCONTROLLER calls `GET /api/health` (existing endpoint). If the response includes
`"cognitive_engine": "active"` in the components map, the chat is enabled. Otherwise a
"Root Node Required" notice is shown with a prompt to connect to a root instance.

#### Capabilities

The Cognitive Engine chat is a privileged system-level interface:
- Query DVE fleet health, task queues, and node reputation across the entire network
- Execute guardrail policy evaluations
- Inspect and run oracle operations (root-only, requires password — existing oracle auth flow)
- Submit validation tasks manually
- Review `ValidationResult` proofs submitted by DVE agents
- Voice commands processed via existing `VoiceProcessor.ts` — extend command patterns with:
  - `"Show DVE fleet status"`
  - `"Run validation on skill [name]"`
  - `"What is the current network health?"`
  - `"Attach policy [id] to DVE [name]"`

#### UI Layout

```
┌─────────────────────────────────────────────────────────────┐
│  ◈ COGNITIVE ENGINE              [Root: knirv-1]   [🎤]    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  [SYSTEM]  Cognitive Engine active. Oracle loaded.          │
│            DVE fleet: 12 nodes, 8 online.                   │
│                                                             │
│  [USER]    What's the reputation distribution?             │
│                                                             │
│  [CE]      Top nodes: DVE-Alpha (847), DVE-Gamma (812)...   │
│                                                             │
│  ─────────────────── edge coloring: teal = thinking ───────  │
│                                                             │
│  [_________________ Type or speak a command __________] 🎙  │
└─────────────────────────────────────────────────────────────┘
```

#### Transport

Messages are sent to `POST /api/knirvshell/execute` (existing KNIRVSHELL endpoint, which
already handles chain, validation, TEE, and P2P commands). The response stream is displayed
incrementally. No new backend endpoint is required for basic operation.

For streaming responses, the existing endpoint is extended to support
`Transfer-Encoding: chunked` with a `text/event-stream` fallback.

### 4.2 DVE KNIRVAGENT Chat

**Context:** Each owned DVE has one supervisor KNIRVAGENT provisioned on creation (Phase 5).
This chat connects directly to that agent's session.

**New file: `src/react-app/pages/AgentChat.tsx`**
**New route:** `/dves/:dveId/agent`

Accessible from the DVE List card's `[Chat ▶]` button.

#### Capabilities

The DVE KNIRVAGENT chat is task-and-capability scoped to the owning DVE:
- Assign skill execution tasks to the DVE
- Query task history and validation results
- Inspect badge capabilities and policy constraints
- Request DVE status (CPU, memory, pending tasks)
- Submit error nodes from within the chat context ("The Fabric" workflow)
- Voice commands scoped to the DVE:
  - `"Run skill [name] on this DVE"`
  - `"Show me pending tasks"`
  - `"What badges does this DVE have?"`
  - `"Submit error [description]"`

#### Transport

Messages are sent via the WebSocket relay established in Phase 2 of the KNIRVBRIDGE plan:
```
AgentChat.tsx  →  WS message type "agent_chat"  →  BrowserDVEHub
               →  KNIRVAGENT session (pkg/agent/loop.go AgentLoop)
               ←  WS message type "agent_response"  ←
```

For hardware DVE nodes (sgx/sev-snp/tdx) where the agent runs on the server, messages are
routed via the existing `BroadcastAgentMessage` interface already wired into `DVEHandlers`
(`packages/KNIRVSERVER/backend/internal/web/dve_handlers.go:44`).

For browser-extension DVE nodes, messages are handled locally within the extension's
`ValidationRuntime` (KNIRVBRIDGE plan Phase 4), with responses surfaced back through the WS.

#### UI Layout

```
┌─────────────────────────────────────────────────────────────┐
│  ← DVE-Alpha     KNIRVAGENT-Alpha    [🟢 Active]   [🎤]    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  [AGENT]  I'm KNIRVAGENT-Alpha, supervising DVE-Alpha.     │
│           Current tasks: 2 pending, 14 completed.           │
│           Capabilities: policy-check, reasoning.            │
│                                                             │
│  [USER]   Run skill error_classifier on the pending tasks.  │
│                                                             │
│  [AGENT]  Running error_classifier... (skill task queued)   │
│           Task ID: val_9f3c...  ETA: ~12s                   │
│                                                             │
│  [_________________ Type or speak a command __________] 🎙  │
└─────────────────────────────────────────────────────────────┘
```

### 4.3 Shared Chat Components

Both surfaces share:

**New file: `src/react-app/components/ChatThread.tsx`**
- Message list with role-based styling (`system`, `user`, `agent`, `cognitive`)
- Streaming token render (SSE or WS chunked)
- Timestamp + copy button per message
- Code block formatting for skill output / JSON results

**New file: `src/react-app/components/VoiceChatBar.tsx`**
- Unified input bar: text field + microphone toggle
- Reuses `VoiceProcessor.ts` for STT; connects output to chat input
- Edge coloring reflects chat-specific states:
  - **Teal:** Agent is processing / CE is thinking
  - **Purple:** Agent / CE is responding (TTS active)
  - **Green:** Idle / ready
  - **Red:** Agent error or disconnected

---

## 5. Supervisor KNIRVAGENT Auto-Initialization

### 5.1 Contract

Every DVE, at the moment of creation, automatically receives one supervisor KNIRVAGENT. No
other agent frameworks are integrated. The KNIRVAGENT is the only agent runtime.

### 5.2 Provisioning Flow

```
POST /api/dve/creations  (KNIRVSERVER)
  → DVECreationService.CreateDVE(req)
    → ContainerOrchestrator.ProvisionContainer(rentalID)        [hardware DVEs]
    │   OR  BrowserDVEHub.RegisterBrowserDVE(walletAddress)     [browser DVEs]
    └→ AgentProvisioner.SpawnSupervisorAgent(dveCreation)       ← NEW
         → Creates KNIRVAGENT config bound to DVE:
             - workspace: /var/knirv/dve/{dveID}/agent/
             - model: configured via DVECreation.capabilities
             - session key: DVESession.SessionKey (encrypted)
             - tools: restricted to DVE-scoped operations only
             - identity: workspace/IDENTITY.md populated with DVE metadata
         → Returns AgentInstance { agentID, sessionKey, wsEndpoint }
         → Stored in DVECreation.SupervisorAgentID (new field)
```

### 5.3 KNIRVSERVER Changes

**New file: `packages/KNIRVSERVER/backend/internal/services/agentprovisioner/agent_provisioner.go`**

```go
type AgentProvisioner struct {
    db          *database.BuntDBManager
    config      *config.Config
    agentBinary string  // path to knirvagent binary
}

func (ap *AgentProvisioner) SpawnSupervisorAgent(dve *objects.DVECreation) (*AgentInstance, error)
func (ap *AgentProvisioner) TerminateAgent(agentID string) error
func (ap *AgentProvisioner) GetAgentStatus(agentID string) (*AgentStatus, error)
```

For **hardware DVEs**: the KNIRVAGENT binary runs inside the provisioned container
(injected via `ContainerOrchestrator` at provision time). The agent's workspace is
`/workspace/` inside the container, restricted to DVE-scoped tools only.

For **browser-extension DVEs**: there is no server-side container. The supervisor KNIRVAGENT
runs as a lightweight in-process goroutine within the extension's background service worker
context. A TypeScript port of the core `AgentLoop` (stripped to the essentials: session, bus,
tool dispatch) is created at:

**New file: `packages/KNIRVBRIDGE/src/services/dve/dve-agent-runtime.ts`**

This is NOT a separate agent framework — it is a TypeScript reimplementation of the same
KNIRVAGENT loop contract, using the same KNIRVAGENT configuration schema and workspace layout.
It executes within the existing `ValidationRuntime` Web Worker.

### 5.4 Agent Identity & Workspace

Each supervisor KNIRVAGENT is initialized with workspace files derived from the DVE's identity:

**`workspace/IDENTITY.md`** (auto-generated):
```markdown
# DVE Supervisor Agent

DVE ID: {dveCreation.ID}
DVE Name: {dveCreation.Name}
Owner Address: {dveCreation.OwnerAddress}
TEE Type: {dveCreation.TEEType}
DVE URI: knirv://dve/{ownerAddress}/{teeType}
Capabilities: {joined capabilities from badges}
Attached Policies: {policy IDs}
Initialized At: {timestamp}
```

**`workspace/AGENT.md`** (template, DVE-scoped):
```markdown
# Agent Instructions

You are a supervisor agent for DVE {dveName} on the KNIRV Network.
Your role is to manage validation tasks, report results, and assist the
DVE owner with skill execution and error resolution.

## Constraints
- Only use tools scoped to this DVE's capabilities
- Do not attempt to access other DVEs or wallets
- Report all validation results via the task_result channel
- Escalate anomalies to the Cognitive Engine via the alert bus
```

### 5.5 Agent Scoped Tool Restrictions

The supervisor KNIRVAGENT's `ToolRegistry` is initialized with a restricted set:
```go
// Allowed for supervisor agents
registry.Register(tools.NewReadFileTool(workspace, true))   // restrict=true
registry.Register(tools.NewWriteFileTool(workspace, true))
registry.Register(tools.NewListDirTool(workspace, true))
registry.Register(DVEValidationTool(dveID, taskQueue))       // new: queue tasks
registry.Register(DVEStatusTool(dveID, nodeTracker))         // new: read node metrics
registry.Register(DVEAlertTool(dveID, alertBus))             // new: send alerts to CE
// NOT registered:
// - ExecTool (no shell access)
// - WebSearchTool (no external internet)
// - Cross-DVE tools
```

---

## 6. Updated Route Map

| Route | Component | Description |
|---|---|---|
| `/` | `DVEList` | Primary view — Vault summary + DVE list |
| `/vault` | `VaultPage` | NRN balance, send/receive, tx history |
| `/dves` | `DVEList` | DVE list (same as `/`) |
| `/dves/new` | `DVECreate` | Create a new DVE |
| `/dves/:dveId` | `DVEDetail` | DVE detail slide-over (panel) |
| `/dves/:dveId/agent` | `AgentChat` | Chat with DVE's supervisor KNIRVAGENT |
| `/cognitive` | `CognitiveEngineChat` | Root node Cognitive Engine chat |
| `/workflows` | `HomePage` | Unchanged |
| `/scanner` | `Scanner` | Unchanged |
| `/skills` | `Skills` | Unchanged |
| `/udc` | `UDC` | Unchanged |
| `/onboarding` | `Onboarding` | Unchanged |

**Update: `src/react-app/App.tsx`** — add the new routes above.

---

## 7. Navigation & Layout Changes

### 7.1 Bottom Navigation Bar

The existing `Layout.tsx` bottom nav needs to be extended:

| Tab | Icon | Route |
|---|---|---|
| DVEs | `Cpu` | `/` |
| Vault | `Zap` (existing wallet icon) | `/vault` |
| Skills | `BookOpen` | `/skills` |
| Chat | `MessageSquare` | `/cognitive` (shows badge if root available) |
| Settings | `Settings` | (existing, keep) |

### 7.2 Header Changes

The header in `Layout.tsx` is updated to show:
- **Left:** KNIRVCONTROLLER logo / name
- **Center:** Active DVE indicator (dot + name) when inside a DVE context route
- **Right:** Vault NRN balance (condensed) + voice toggle button

---

## 8. Shared Types

**Update: `src/shared/types.ts`**

Add:

```typescript
export interface DVEWallet {
  dveID: string;
  dveName: string;
  walletAddress: string;          // derived DVE Wallet address
  derivationIndex: number;        // N in m/44'/60'/1'/0/N
  teeType: "sgx" | "sev-snp" | "tdx" | "software" | "browser-extension";
  status: "online" | "offline" | "maintenance" | "error";
  stakeAmount: number;
  reputationScore: number;
  capabilities: string[];
  attachedPolicies: string[];
  badgeNFTIDs: string[];
  supervisorAgentID: string;       // KNIRVAGENT instance ID
  dveURI: string;                  // "knirv://dve/{address}/{teeType}"
}

export interface VaultAccount {
  address: string;
  nrnBalance: number;
  usdValue: number;
  dveWallets: DVEWallet[];
  isRootNode: boolean;             // true if connected server has oracle loaded
}

export interface AgentMessage {
  id: string;
  role: "user" | "agent" | "system" | "cognitive";
  content: string;
  timestamp: string;
  dveID?: string;
  taskID?: string;
  streaming?: boolean;
}
```

---

## 9. Backend Worker Changes

**Update: `src/worker/index.ts`** (Cloudflare Worker)

Add API proxy routes:
- `GET /worker/dve/list` → proxies to `GET /api/dve/nodes` on configured KNIRVSERVER
- `GET /worker/dve/creations` → proxies to `GET /api/dve/creations`
- `POST /worker/dve/create` → proxies to `POST /api/dve/creations`
- `GET /worker/cognitive/status` → proxies to `GET /api/health` (checks CE active)
- `POST /worker/cognitive/chat` → proxies to `POST /api/knirvshell/execute`

The worker holds the KNIRVSERVER base URL in a Cloudflare environment binding
(`KNIRVSERVER_URL`), keeping it out of the client bundle.

---

## 10. Implementation Order

| Phase | Work | Depends On |
|---|---|---|
| 1 | Package rename + Vault page refactor | — |
| 2 | `DVEWallet` type + HD derivation in `useVault` | Phase 1 |
| 3 | DVE List page (read-only, no agent chat yet) | Phase 2 + KNIRVSERVER DVE APIs |
| 4 | Cognitive Engine Chat | Phase 3 + KNIRVSHELL streaming |
| 5 | Supervisor KNIRVAGENT provisioning (server side) | KNIRVBRIDGE Phase 2 WS |
| 6 | DVE KNIRVAGENT Chat | Phase 5 |
| 7 | DVE Create flow + badge attachment | Phase 6 + KNIRVBRIDGE Phase 3 |
| 8 | Browser-extension DVE agent runtime (TS port) | KNIRVBRIDGE Phase 4 |

Phases 4 and 5 can proceed in parallel after Phase 3 is stable.

---

## 11. Files Changed Summary

| File | Change |
|---|---|
| `packages/KNIRVWALLET/` → `packages/KNIRVCONTROLLER/` | Directory rename |
| `src/react-app/App.tsx` | Add new routes |
| `src/react-app/pages/Wallet.tsx` → `VaultPage.tsx` | Rename + header copy update |
| `src/react-app/pages/DVEList.tsx` | New — primary DVE list view |
| `src/react-app/pages/DVECreate.tsx` | New — DVE creation wizard |
| `src/react-app/pages/CognitiveEngineChat.tsx` | New — root CE chat |
| `src/react-app/pages/AgentChat.tsx` | New — per-DVE agent chat |
| `src/react-app/components/ChatThread.tsx` | New — shared chat UI |
| `src/react-app/components/VoiceChatBar.tsx` | New — voice+text input bar |
| `src/react-app/components/Layout.tsx` | Update nav tabs + header |
| `src/react-app/hooks/useWallet.ts` → `useVault.ts` | Rename + multi-DVE wallet support |
| `src/shared/types.ts` | Add `DVEWallet`, `VaultAccount`, `AgentMessage` |
| `src/worker/index.ts` | Add DVE + cognitive proxy routes |
| `KNIRVSERVER/backend/internal/services/agentprovisioner/` | New — supervisor agent spawn |
| `KNIRVSERVER/backend/internal/objects/dve.go` | Add `SupervisorAgentID` to `DVECreation` |
| `KNIRVBRIDGE/src/services/dve/dve-agent-runtime.ts` | New — browser DVE agent TS port |

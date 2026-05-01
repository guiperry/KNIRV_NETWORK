KNIRVCONTROLLER

### 🔷 KNIRVCONTROLLER: The Adaptive Intelligence Interface
Technology: React/TypeScript with Vite + Voice Integration + DVE Management

Purpose: The primary user gateway to the KNIRV D-TEN. Manages the Vault (master account), DVE wallets, supervisor KNIRVAGENTS, and provides voice/text chat surfaces for both the Cognitive Engine and per-DVE agents.

## 🎯 Key Features

### Vault Management
- NRN balance display and transaction history
- Send/receive NRN tokens
- Wallet unlock/lock security
- HD key derivation for DVE wallets (BIP44 `m/44'/60'/1'/0/N`)

### DVE Wallet System
- Multi-DVE identity model: one Vault + N derived DVE wallets
- Each DVE Wallet mapped 1:1 to a DVECreation record on KNIRVSERVER
- HD derivation at `m/44'/60'/1'/0/N` for strict key separation
- DVE Wallet lifecycle: create → derive → register → decommission

### DVE List View
- Primary interface (`/`) showing all owned DVEs with live status
- Sortable by reputation score
- Network DVEs discovery (read-only, paginated)
- Expandable detail panel with metrics, badges, policies, and stake management

### DVE Creation Wizard
- 5-step creation flow: Name & Type → Stake → Badges → Confirm → Submit
- TEE type selection (hardware / browser-extension)
- Badge attachment from vault inventory
- Automatic supervisor KNIRVAGENT provisioning on submit

### Chat Surfaces

**Cognitive Engine Chat** (`/cognitive`)
- Root-node only privileged interface
- DVE fleet health, task queues, guardrail policies
- Oracle operations (root-only)
- Voice commands via Web Speech API

**DVE KNIRVAGENT Chat** (`/dves/:dveId/agent`)
- Per-DVE supervisor agent conversation
- Skill execution, task history, badge capabilities
- Voice + text commands scoped to the DVE

### Voice Integration
- Real-time speech recognition via Web Speech API
- Wake word detection ("KNIRV")
- Edge coloring visual feedback system
- Text-to-speech responses
- Voice commands for navigation, skills, and system control

### Visual Feedback
- Dynamic edge coloring based on voice/agent status
- Smooth animated transitions (500ms)
- Color states: idle (green), listening (teal), processing (blue), speaking (purple), error (red)
- Glowing glass-panel UI components

## 🏗️ Architecture

### Route Map

| Route | Component | Description |
|-------|-----------|-------------|
| `/` | `DVEList` | Primary view — Vault summary + DVE list |
| `/vault` | `VaultPage` | NRN balance, send/receive, tx history |
| `/dves` | `DVEList` | DVE list (same as `/`) |
| `/dves/new` | `DVECreate` | Create a new DVE |
| `/dves/:dveId` | `DVEList` | DVE detail slide-over (panel) |
| `/dves/:dveId/agent` | `AgentChat` | Chat with DVE's supervisor KNIRVAGENT |
| `/cognitive` | `CognitiveEngineChat` | Root node Cognitive Engine chat |
| `/workflows` | `HomePage` | Unchanged from original |
| `/scanner` | `Scanner` | QR scanner |
| `/skills` | `Skills` | Skill management |
| `/udc` | `UDC` | User Delegation Certificate |
| `/onboarding` | `Onboarding` | First-time setup |

### Component Structure

```
src/
├── shared/
│   ├── cognitive-shell/
│   │   ├── EventEmitter.ts       # Browser-compatible event system
│   │   └── VoiceProcessor.ts     # Core voice processing logic
│   └── types.ts                  # Shared type definitions (DVEWallet, VaultAccount, AgentMessage)
├── react-app/
│   ├── components/
│   │   ├── Layout.tsx            # Main layout with voice + nav
│   │   ├── EdgeColoring.tsx      # Visual feedback system
│   │   ├── VoiceControl.tsx      # Voice interface component
│   │   ├── ChatThread.tsx        # Shared chat message list
│   │   └── VoiceChatBar.tsx      # Unified voice+text input bar
│   ├── hooks/
│   │   ├── useVault.ts           # Vault + multi-DVE wallet management
│   │   ├── useBackend.ts         # Backend data fetching
│   │   └── useVoiceIntegration.ts # Voice state management
│   └── pages/
│       ├── DVEList.tsx           # Primary DVE list view
│       ├── DVECreate.tsx         # DVE creation wizard
│       ├── VaultPage.tsx         # Vault balance + transactions
│       ├── CognitiveEngineChat.tsx # Root CE chat
│       ├── AgentChat.tsx         # Per-DVE agent chat
│       └── ...                   # Home, Scanner, Skills, UDC, Onboarding
├── worker/
│   └── index.ts                  # Cloudflare Worker API proxy
└── react-app/__tests__/          # Vitest test files
```

### Backend Integration

```
KNIRVCONTROLLER (Browser)
│
├── Cloudflare Worker (proxy)
│   │
│   └── KNIRVSERVER (Go backend)
│       ├── /api/dve/nodes         → DVE List
│       ├── /api/dve/creations     → DVE Creation + Management
│       ├── /api/health            → Cognitive Engine Status
│       └── /api/knirvshell/execute → Chat commands
│
└── WebSocket (browser DVE)
    │
    └── BrowserDVEHub → DVEAgentRuntime (TS)
```

## 🚀 Getting Started

### Prerequisites
- Node.js 20+
- Modern browser with Web Speech API support
- Microphone for voice features

### Installation
```bash
cd packages/KNIRVCONTROLLER
npm install
npm run dev
```

## 🧪 Testing

```bash
# Run vitest tests
npx vitest run

# Run Go backend tests
cd ../../packages/KNIRVSERVER && go test -v ./backend/internal/services/agentprovisioner/...
```

## 🔮 Future Enhancements
- Multi-language voice support
- Advanced NLP for context-aware commands
- Offline speech recognition
- Voice biometrics for authentication
- DVE rental marketplace integration
- Supervisor KNIRVAGENT performance analytics

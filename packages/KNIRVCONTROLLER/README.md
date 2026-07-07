KNIRVCONTROLLER

### 🔷 KNIRVCONTROLLER: The Adaptive Intelligence Interface
Technology: React/TypeScript with Vite + Voice Integration + DVE Management

Purpose: The primary user gateway to the KNIRV D-TEN. Manages the Vault (master account), DVE vault identities, supervisor KNIRVAGENTS, and provides voice/text chat surfaces for both the Cognitive Engine and per-DVE agents.

This package is currently a Vite web app. For phone sniff testing, use the PWA HTTPS launcher instead of Expo.
It also now ships native wrapper shells for Android and iOS via Capacitor around the same Vite build.

## 🎯 Key Features

### Vault Management
- NRN balance display and transaction history
- Send/receive NRN tokens
- Vault unlock/lock security
- HD key derivation for DVE vault identities (BIP44 `m/44'/60'/1'/0/N`)

### DVE Vault System
- Multi-DVE identity model: one Vault + N derived DVE vault identities
- Each DVE vault mapped 1:1 to a DVECreation record on KNIRVSERVER
- HD derivation at `m/44'/60'/1'/0/N` for strict key separation
- DVE vault lifecycle: create → derive → register → decommission

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
│   └── types.ts                  # Shared type definitions (DVEVault, VaultAccount, AgentMessage)
├── react-app/
│   ├── components/
│   │   ├── Layout.tsx            # Main layout with voice + nav
│   │   ├── EdgeColoring.tsx      # Visual feedback system
│   │   ├── VoiceControl.tsx      # Voice interface component
│   │   ├── ChatThread.tsx        # Shared chat message list
│   │   └── VoiceChatBar.tsx      # Unified voice+text input bar
│   ├── hooks/
│   │   ├── useVault.ts           # Vault + multi-DVE vault management
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
- For phone installability, use an HTTPS local launch

### Installation
```bash
cd packages/KNIRVCONTROLLER
npm install
npm run dev
```

### Local Testing
```bash
# Standard web dev server
npm run start

# LAN-accessible PWA dev server
npm run start:pwa

# HTTPS PWA launcher for phone testing
npm run start:mobile

# HTTPS PWA preview from a built app
npm run preview:pwa:https

# Native wrapper sync
npm run cap:sync

# Open the native shells
npm run cap:open:android
npm run cap:open:ios
```

### Phone Sniff Testing
- Run `npm run start:mobile` from `packages/KNIRVCONTROLLER`
- Open the resulting `https://<lan-ip>:4174` URL on your phone
- If the LAN IP is not detected correctly, set `KNIRV_HTTPS_IPS=<your-phone-reachable-ip>`
- You can also override `KNIRV_PWA_HOST` and `KNIRV_PWA_PORT` if needed
- The app registers a service worker from `/sw.js`, so HTTPS is required for installable PWA behavior on a phone

### Native Wrappers
- `android/` and `ios/` are generated Capacitor wrapper projects around the same controller build
- Run `npm run build` before `npm run cap:sync` to copy the latest web assets into the native shells
- Open Android in Android Studio with `npm run cap:open:android`
- Open iOS in Xcode with `npm run cap:open:ios` on macOS
- The native shells are wrappers for the current controller app, not a full rewrite
- A full native rewrite remains a separate future project

### Release Builds
- `npm run release:mobile` builds the Android APK and iOS IPA, stores the raw artifacts in `build/` as `build/knirvcontroller-android-latest.apk` and `build/knirvcontroller-ios-latest.ipa`, then uploads both files with `rclone`
- `npm run release:mobile:upload` skips rebuilds and uploads the existing `build/knirvcontroller-android-latest.apk` and `build/knirvcontroller-ios-latest.ipa` files with `rclone`
- `npm run release:mobile:upload:android` uploads only the existing Android APK
- `npm run release:mobile:upload:ios` uploads only the existing iOS IPA
- The Android release uses a local signing config if these environment variables are provided: `KNIRV_ANDROID_KEYSTORE_FILE`, `KNIRV_ANDROID_KEYSTORE_PASSWORD`, `KNIRV_ANDROID_KEY_ALIAS`, `KNIRV_ANDROID_KEY_PASSWORD`
- If no Android release keystore is provided, the release script generates a project-local signing key in `build/signing/` so sideload testing still works
- Android release builds require JDK 17 in this workspace
- iOS IPA export requires macOS, Xcode, and valid Apple signing credentials
- Set `KNIRV_IOS_EXPORT_METHOD` to control the IPA export mode if needed; the default is `development`
- The upload targets are `incline:knirv/controller/android/` and `incline:knirv/controller/ios/`

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

# KNIRVCONTROLLER Mobile Refactor Plan

Tracks remaining work to bring the Android and iOS builds to feature parity with the web app.
Capacitor scaffolding and the release pipeline are already in place — this document picks up from there.

---

## What's Already Shipped

| Area | Status | Notes |
|---|---|---|
| Capacitor foundation | ✅ Done | `android/`, `ios/`, `capacitor.config.ts` |
| Android signing | ✅ Done | `build.gradle` reads `KNIRV_ANDROID_*` env vars; debug keystore auto-generated in `build/signing/` |
| Release script | ✅ Done | `scripts/release-native.mjs` — APK + IPA build, uploads via `rclone` to `incline:knirv/controller/{android,ios}/` |
| Android permissions | ✅ Done | `CAMERA`, `RECORD_AUDIO` in `AndroidManifest.xml` |
| iOS usage strings | ✅ Done | Camera, Microphone, SpeechRecognition in `Info.plist` |
| Native QR scanning | ✅ Done | `capacitor-barcode-scanner` + `src/react-app/platform/nativeQrScanner.ts` + `QrScannerFrame.tsx` |
| QR bridge in pages | ✅ Done | `Onboarding.tsx` and `Scanner.tsx` both use `QrScannerFrame` |
| Voice packages | ✅ Done | `@capgo/capacitor-speech-recognition` and `@capacitor-community/text-to-speech` installed |
| Voice platform infra | ✅ Done | `cognitive-shell/VoiceProcessor.ts`, `platform/voiceProcessor.ts` |

---

## Remaining Work

### 1  Wire native voice into `useVoiceIntegration`
**Files:** `src/react-app/hooks/useVoiceIntegration.ts`

`@capgo/capacitor-speech-recognition` and `@capacitor-community/text-to-speech` are installed but `useVoiceIntegration.ts` still simulates STT/TTS with `setTimeout`. The real wiring is missing.

Create `src/react-app/platform/voiceBridge.ts`:

```ts
import { Capacitor } from '@capacitor/core';
import { SpeechRecognition } from '@capgo/capacitor-speech-recognition';
import { TextToSpeech } from '@capacitor-community/text-to-speech';

export async function startNativeListening(onResult: (text: string) => void): Promise<void> {
  const { speechRecognition } = await SpeechRecognition.requestPermissions();
  if (speechRecognition !== 'granted') throw new Error('Microphone permission denied');
  await SpeechRecognition.start({ language: 'en-US', maxResults: 1, partialResults: true, popup: false });
  await SpeechRecognition.addListener('partialResults', (data) => {
    if (data.matches?.[0]) onResult(data.matches[0]);
  });
}

export async function stopNativeListening(): Promise<void> {
  await SpeechRecognition.stop();
  await SpeechRecognition.removeAllListeners();
}

export async function speakNative(text: string): Promise<void> {
  await TextToSpeech.speak({ text, lang: 'en-US', rate: 0.9, pitch: 1.0, volume: 1.0 });
}
```

Then in `useVoiceIntegration.ts`, replace the simulated `toggleVoice` and `speakResponse` with:

```ts
import { Capacitor } from '@capacitor/core';
import { startNativeListening, stopNativeListening, speakNative } from '@/react-app/platform/voiceBridge';

// In toggleVoice:
if (Capacitor.isNativePlatform()) {
  if (active) await startNativeListening(handleVoiceCommand);
  else await stopNativeListening();
} else {
  // existing Web Speech API path
}

// In speakResponse:
if (Capacitor.isNativePlatform()) {
  await speakNative(text);
} else {
  // existing window.speechSynthesis path
}
```

---

### 2  Persistent native storage for vault and app settings
**Files:** `src/react-app/hooks/useVault.ts`, `src/react-app/hooks/useAppSettings.ts`, `src/react-app/platform/browserStorage.ts`, `src/react-app/platform/vaultHandoff.ts`

#### 2a  Install `@capacitor/preferences`
```bash
npm install @capacitor/preferences
npx cap sync
```

#### 2b  Create `src/react-app/platform/nativeStorage.ts`
```ts
import { Capacitor } from '@capacitor/core';
import { Preferences } from '@capacitor/preferences';

// Drop-in replacement for localStorage / sessionStorage calls.
// On web falls through to the existing browserStorage helpers.
export const nativeStorage = {
  async get(key: string): Promise<string | null> {
    if (Capacitor.isNativePlatform()) {
      const { value } = await Preferences.get({ key });
      return value;
    }
    return localStorage.getItem(key);
  },
  async set(key: string, value: string): Promise<void> {
    if (Capacitor.isNativePlatform()) {
      await Preferences.set({ key, value });
    } else {
      localStorage.setItem(key, value);
    }
  },
  async remove(key: string): Promise<void> {
    if (Capacitor.isNativePlatform()) {
      await Preferences.remove({ key });
    } else {
      localStorage.removeItem(key);
    }
  },
};
```

#### 2c  Update `useVault.ts` — four call sites
Replace every `localStorage.getItem(VAULT_STORAGE_KEY)` / `localStorage.setItem(...)` / `localStorage.removeItem(...)` with `await nativeStorage.get/set/remove(...)`.
The initial `checkStorage` `useEffect` must be made async for this. No other logic changes.

#### 2d  Update `browserStorage.ts` — add async native path
`readStorageItem` / `writeStorageItem` / `removeStorageItem` are used by `useAppSettings` and `vaultHandoff`. Add overloads that accept a `native?: boolean` flag (driven by `Capacitor.isNativePlatform()`) and delegate to `nativeStorage`.  
**`vaultHandoff.ts`** currently uses `sessionStorage` — on native, `session` scope should map to in-memory state (session scoped storage has no native equivalent; the handoff payload is ephemeral by design, so an in-memory map is correct).

#### 2e  Future hardening: Keychain / Keystore
After the base adapter ships, swap `Preferences` for `@aparajita/capacitor-secure-storage` on the native path. The vault blob is already AES-encrypted by `knirvwallet-module`, so this adds a second layer: iOS Keychain with `.whenUnlockedThisDeviceOnly` + biometric unlock, Android Keystore with StrongBox.

---

### 3  Vite WebView asset path fix
**File:** `vite.config.ts`

Without `base: './'`, Vite emits absolute asset paths (e.g. `/assets/index-abc.js`) that resolve against the WebView origin but break when the bundle is loaded from the local filesystem. Add to the `build:` block:

```ts
build: {
  target: 'esnext',
  chunkSizeWarningLimit: 5000,
  base: './',   // required for Capacitor WebView
},
```

Verify with `npm run build && npx cap sync` — all JS/CSS hrefs in `dist/index.html` should start with `./`.

---

### 4  Safe area insets and viewport config
**Files:** `index.html`, `src/react-app/components/Layout.tsx`

iOS notch and home-indicator devices clip the fixed header and bottom nav without `viewport-fit=cover`.

**`index.html`** — replace the viewport meta:
```html
<!-- before -->
<meta name="viewport" content="width=device-width, initial-scale=1.0" />

<!-- after -->
<meta name="viewport" content="viewport-fit=cover, width=device-width, initial-scale=1.0, maximum-scale=1.0" />
```

**`Layout.tsx`** — add `style` props to the fixed header and bottom nav:
```tsx
// Fixed header
<header
  className="..."
  style={{ paddingTop: 'env(safe-area-inset-top)' }}
>

// Bottom nav
<nav
  className="..."
  style={{ paddingBottom: 'env(safe-area-inset-bottom)' }}
>
```

---

### 5  Capacitor plugin polish (status bar, haptics, keyboard, back button)

Install remaining Capacitor core plugins:
```bash
npm install @capacitor/app @capacitor/haptics @capacitor/keyboard @capacitor/status-bar @capacitor/splash-screen
npx cap sync
```

**Status bar** — add to `capacitor.config.ts` plugins block:
```ts
plugins: {
  SplashScreen: { launchAutoHide: false, backgroundColor: '#08111f', showSpinner: false },
  StatusBar: { style: 'Dark', backgroundColor: '#08111f', overlaysWebView: false },
},
```

**Android back button** — create `src/react-app/platform/backHandler.ts`:
```ts
import { App } from '@capacitor/app';
import { Capacitor } from '@capacitor/core';

export function registerBackHandler(onBack: () => boolean) {
  if (!Capacitor.isNativePlatform()) return () => {};
  const listener = App.addListener('backButton', () => {
    if (!onBack()) App.exitApp();
  });
  return () => { listener.then(l => l.remove()); };
}
```
Call from `Layout.tsx` — wire to `navigate(-1)`; return `false` (triggering `exitApp`) only when already at the root.

**Haptic feedback** — wrap in `Capacitor.isNativePlatform()` guard; add to:
- Vault creation success → `Haptics.impact({ style: ImpactStyle.Medium })`
- Transaction confirmed → `Haptics.impact({ style: ImpactStyle.Heavy })`
- Error states → `Haptics.notification({ type: NotificationStyle.Error })`

**Keyboard** — add to `src/react-app/main.tsx`:
```ts
import { Keyboard } from '@capacitor/keyboard';
if (Capacitor.isNativePlatform()) {
  Keyboard.setResizeMode({ mode: 'body' });
}
```

---

### 6  App icons and splash screen
**Requires:** Source artwork at `assets/` (does not yet exist)

```bash
npm install --save-dev @capacitor/assets
```

Create:
```
assets/
  icon.png              1024×1024  KNIRV mark on #08111f
  icon-foreground.png   1024×1024  Android adaptive foreground
  icon-background.png   1024×1024  Android adaptive background (#08111f solid)
  splash.png            2732×2732  Centered KNIRV mark on #08111f
```

Generate all required densities:
```bash
npx @capacitor/assets generate --ios --android
```

This replaces the default Capacitor placeholder icons across all iOS `@1x`/`@2x`/`@3x` and Android `mdpi` → `xxxhdpi` slots.

---

### 7  Bluetooth permissions for Ledger (Android manifest)
**File:** `android/app/src/main/AndroidManifest.xml`

CAMERA and RECORD_AUDIO are already present. Ledger BLE support (Phase 8 below) requires:
```xml
<uses-permission android:name="android.permission.BLUETOOTH" />
<uses-permission android:name="android.permission.BLUETOOTH_CONNECT" />
<uses-permission android:name="android.permission.BLUETOOTH_SCAN" />
<uses-permission android:name="android.permission.ACCESS_FINE_LOCATION" />
```
Add these now so store review is not re-triggered later.

iOS Bluetooth usage string should also be added to `Info.plist`:
```xml
<key>NSBluetoothAlwaysUsageDescription</key>
<string>KNIRV Controller uses Bluetooth to connect Ledger hardware wallets.</string>
```

---

### 8  Ledger BLE bridge (advanced, deferrable)

`@ledgerhq/hw-transport-webhid` uses WebHID — unavailable in any mobile WebView. Requires a custom Capacitor plugin wrapping the native Ledger Device SDKs.

**New files:**
```
src/capacitor-plugins/ledger-ble/definitions.ts    — plugin TS interface
src/capacitor-plugins/ledger-ble/index.ts          — registerPlugin('LedgerBLE')
src/react-app/platform/ledgeBleTransport.ts        — getLedgerTransport() returning CapacitorLedgerTransport on native
android/.../LedgerBLEPlugin.java                   — wraps com.ledger.devicesdk:ledger-device-sdk
ios/.../LedgerBLEPlugin.swift                      — wraps LedgerDeviceKit via Swift Package Manager
```

The plugin exposes `scan()`, `connect({ deviceId })`, `disconnect()`, `exchange({ apdu: string })` — enough for `@ledgerhq/hw-app-cosmos` to drive the full signing flow.
All wallet signing logic in `knirvwallet-module/` is pure TypeScript and requires no changes.

**Can ship Phases 1–7 without this.** Ledger BLE is a follow-on sprint.

---

### 9  CI/CD pipelines
**Missing:** `.github/` directory does not yet exist under KNIRVCONTROLLER.

The `release-native.mjs` script handles local builds. GitHub Actions workflows are needed for automated builds on push and pull requests.

Create `.github/workflows/build-android.yml` and `.github/workflows/build-ios.yml`.

**`build-android.yml`** — trigger: push to `main` touching `packages/KNIRVCONTROLLER/**`
- `ubuntu-latest` runner
- `actions/setup-node@v4` (Node 20) + `actions/setup-java@v4` (Java 17, Temurin)
- `npm ci` → `npm run build` (inject `VITE_*` from secrets) → `npx cap sync android`
- Decode `KNIRV_ANDROID_KEYSTORE_FILE` from `ANDROID_KEYSTORE_BASE64` secret
- `./gradlew assembleRelease` → copy `app-release.apk` → `actions/upload-artifact@v4`
- On `main`: upload AAB (`./gradlew bundleRelease`) to Play Store Internal via `r0adkll/upload-google-play@v1`

Required secrets: `ANDROID_KEYSTORE_BASE64`, `KNIRV_ANDROID_KEYSTORE_PASSWORD`, `KNIRV_ANDROID_KEY_ALIAS`, `KNIRV_ANDROID_KEY_PASSWORD`, `VITE_API_URL`, `VITE_XION_RPC_URL`, `VITE_KNIRV_CHAIN_ID`, `PLAY_SERVICE_ACCOUNT_JSON`

**`build-ios.yml`** — trigger: same
- `macos-14` runner (Apple Silicon, Xcode 15)
- `npm ci` → `npm run build` → `npx cap sync ios`
- Import signing cert via `apple-actions/import-codesign-certs@v2`
- Import provisioning profile via `apple-actions/download-provisioning-profiles@v1`
- `xcodebuild -workspace ios/App/App.xcworkspace -scheme App archive`
- `xcodebuild -exportArchive` with `ios/ExportOptions.plist`
- On `main`: upload IPA to TestFlight via `apple-actions/upload-testflight-build@v1`

Required secrets: `IOS_CERTIFICATE_BASE64`, `IOS_CERTIFICATE_PASSWORD`, `APPLE_ISSUER_ID`, `APPLE_API_KEY_ID`, `APPLE_API_PRIVATE_KEY`, `APPLE_TEAM_ID`

Create `ios/ExportOptions.plist`:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>method</key><string>app-store</string>
  <key>teamID</key><string>YOUR_TEAM_ID</string>
  <key>signingStyle</key><string>manual</string>
  <key>uploadBitcode</key><false/>
  <key>uploadSymbols</key><true/>
</dict></plist>
```

Note: the current `release-native.mjs` uploads to `incline:knirv/controller/` via `rclone`. The CI workflows above target Play Store / TestFlight directly. Both paths can coexist — local `npm run release:mobile` for ad-hoc distribution, CI for store submission.

---

### 10  E2E testing (Maestro)
**Missing:** No `.maestro/` directory exists.

Maestro works against Capacitor WebView apps without any framework coupling.

```bash
curl -Ls "https://get.maestro.mobile.dev" | bash
```

Create `.maestro/flows/`:

```
onboarding-create.yaml      happy path: create new vault, back up phrase, sign in
onboarding-import.yaml      restore from mnemonic
vault-view.yaml             unlock vault, check balance display
qr-scan.yaml                requires physical device; scan send: QR, confirm send flow
scanner-send.yaml           paste a send: payload via debug text input
dve-list.yaml               navigate to DVEs, verify list renders
settings-clear.yaml         clear vault from settings, verify onboarding shown
```

Example `onboarding-create.yaml`:
```yaml
appId: com.knirv.controller
---
- launchApp
- assertVisible: "Initialize your Identity"
- tapOn: "Create New Controller"
- assertVisible: "Set Device Password"
- inputText:
    text: "TestPassword123!"
- tapOn:
    id: "confirmPassword"
- inputText:
    text: "TestPassword123!"
- tapOn: "Secure Vault"
- assertVisible: "Secret Recovery Phrase"
- tapOn: "I have saved my phrase"
- assertVisible: "Sign To Enter KNIRV"
```

Run locally:
```bash
maestro test .maestro/flows/onboarding-create.yaml
```

Run in CI (add to both workflow files after the build step):
```bash
maestro cloud --apiKey $MAESTRO_API_KEY --app-file build/knirvcontroller-android-latest.apk .maestro/flows/
```

Required secret: `MAESTRO_API_KEY`

---

## Execution Order

Tasks 1–5 can be worked in parallel by different developers — they touch different files and subsystems.

```
Now:
  Task 1  Wire native voice (useVoiceIntegration + voiceBridge.ts)
  Task 2  Native storage adapter (nativeStorage.ts, useVault, browserStorage)
  Task 3  Vite base path fix
  Task 4  Safe area insets (index.html + Layout.tsx)
  Task 5  Plugin polish (install @capacitor/app etc., status bar, haptics, keyboard, back button)

After artwork is ready:
  Task 6  App icons + splash screen

Alongside Task 5:
  Task 7  Android BLE permissions (manifest + Info.plist)

Sprint 2:
  Task 8  Ledger BLE plugin (native Java + Swift + JS bridge)

Any time:
  Task 9  GitHub Actions CI/CD
  Task 10 Maestro E2E flows
```

**Minimum viable native release** (TestFlight beta + Play Internal): Tasks 2, 3, 4, 5, 6, 9 — Task 1 (voice) and Task 8 (Ledger) can follow.

---

## File Change Summary

| File | Action | Task |
|---|---|---|
| `src/react-app/platform/voiceBridge.ts` | **New** — native STT/TTS bridge | 1 |
| `src/react-app/hooks/useVoiceIntegration.ts` | **Adapt** — wire voiceBridge, guard by `isNativePlatform()` | 1 |
| `src/react-app/platform/nativeStorage.ts` | **New** — Capacitor Preferences adapter | 2 |
| `src/react-app/hooks/useVault.ts` | **Adapt** — 4 `localStorage` calls → `nativeStorage` | 2 |
| `src/react-app/platform/browserStorage.ts` | **Adapt** — add async native path; fix session scope on native | 2 |
| `vite.config.ts` | **Adapt** — add `base: './'` to build block | 3 |
| `index.html` | **Adapt** — add `viewport-fit=cover` | 4 |
| `src/react-app/components/Layout.tsx` | **Adapt** — safe area padding on header + bottom nav | 4 |
| `capacitor.config.ts` | **Adapt** — add SplashScreen + StatusBar plugin config | 5 |
| `src/react-app/platform/backHandler.ts` | **New** — Android back button handler | 5 |
| `src/react-app/main.tsx` | **Adapt** — Keyboard resize mode on native | 5 |
| `assets/` (4 images) | **New** — icon + splash source artwork | 6 |
| `android/app/src/main/AndroidManifest.xml` | **Adapt** — add Bluetooth + Location permissions | 7 |
| `ios/App/App/Info.plist` | **Adapt** — add Bluetooth usage string | 7 |
| `src/capacitor-plugins/ledger-ble/` | **New** — plugin definition + index | 8 |
| `src/react-app/platform/ledgerBleTransport.ts` | **New** — `getLedgerTransport()` | 8 |
| `android/.../LedgerBLEPlugin.java` | **New** — native Ledger Device SDK wrapper | 8 |
| `ios/.../LedgerBLEPlugin.swift` | **New** — native LedgerDeviceKit wrapper | 8 |
| `.github/workflows/build-android.yml` | **New** — CI Android build + Play Store upload | 9 |
| `.github/workflows/build-ios.yml` | **New** — CI iOS build + TestFlight upload | 9 |
| `ios/ExportOptions.plist` | **New** — xcodebuild export configuration | 9 |
| `.maestro/flows/*.yaml` | **New** — E2E test flows (7 files) | 10 |

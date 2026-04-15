# KNIRVSERVER PWA Refactor: Feasibility Report & Implementation Plan

## 1. Executive Summary
This report analyzes the feasibility and provides a detailed plan for refactoring the **KNIRVSERVER Desktop Application** (currently an Electron-based HUD) into a **Progressive Web Application (PWA)**. 

The goal is to allow users to "download" and install the client version of the application directly from their browser, providing a native-like experience with lower overhead, easier updates, and broader platform compatibility.

## 2. Current Architecture Overview
The current KNIRVSERVER desktop application consists of three main parts:
1.  **Go Backend**: A high-performance server providing the core API and orchestration logic.
2.  **Next.js Frontend**: The main user interface, currently served via an iframe inside the desktop wrapper.
3.  **Electron Desktop Wrapper**: A HUD (Heads-Up Display) built with HTML/CSS/JS that provides login, menu transitions, and system monitoring (CPU/Memory) via Node.js `os` module.

## 3. Feasibility Analysis

### 3.1 Advantages of PWA Refactor
-   **Platform Agnostic**: Works on Windows, macOS, Linux, and even mobile without platform-specific builds.
-   **Reduced Overhead**: Eliminates the Electron runtime, significantly reducing memory and disk usage.
-   **Streamlined Updates**: Updates are delivered automatically when the user refreshes or the service worker detects a new version.
-   **Installability**: Users can still "install" the app to their desktop/dock via the browser's "Install" prompt.

### 3.2 Challenges & Mitigations
| Challenge | Impact | Mitigation Strategy |
| :--- | :--- | :--- |
| **System Metrics** | The current HUD uses Node.js `os` module to read CPU/Memory. | Expose a new API endpoint in the Go backend (`/api/v1/system/info`) that provides these metrics to the frontend. |
| **Window Controls** | PWA doesn't have native "Minimize" or "Close" buttons. | Use standard browser/OS window controls. The HUD's "Close" button can be repurposed to "Log Out" or "Close Connection". |
| **Local File Access** | Limited in browsers compared to Electron. | Use the File System Access API where needed, or proxy file operations through the Go backend. |
| **Offline Support** | Essential for a "client" feel. | Implement a robust Service Worker strategy (Cache First) to ensure the UI is always available. |

## 4. Implementation Plan

### Phase 1: HUD & UI Consolidation
-   **Port HUD to React**: Extract the HUD layout from `packages/KNIRVSERVER/desktop/index.html` and `styles.css` and create a `HudLayout` component in Next.js.
-   **Unified Login/Menu**: Integrate the login and constellation menu logic directly into the Next.js app routes (`/login`, `/menu`).
-   **Theme Synchronization**: Ensure the HUD styling matches the existing Next.js frontend design language.

### Phase 2: Backend API Expansion
-   **System Info Endpoint**: Implement a new handler in `packages/KNIRVSERVER/backend/internal/api` that utilizes the existing `SystemInfoCollector` in `internal/utils/host/system_info.go`.
-   **WebSocket Metrics**: Optionally implement a WebSocket stream for real-time performance metrics (CPU, Memory, Network RX/TX) to power the HUD charts.

### Phase 3: PWA Integration
-   **Manifest Generation**: Create `public/manifest.json` with appropriate icons, theme colors, and display modes (`standalone`).
-   **Service Worker**: Implement a service worker (using `next-pwa` or a custom script) to cache the static export files.
-   **Icons**: Generate a set of KNIRV-branded icons (192x192, 512x512) for the PWA.

### Phase 4: Build & Deployment Workflow
-   **Static Export**: Continue using Next.js `output: 'export'` to generate a static site.
-   **Go Embedding**: The Go backend will continue to embed the `frontend/out` directory but will also need to serve the `manifest.json` and service worker with correct MIME types.
-   **Deployment**: The user "downloads" the client by navigating to the KNIRVSERVER instance and clicking "Install".

## 5. Timeline & Milestones
1.  **Milestone 1 (Week 1)**: HUD componentized in Next.js and backend System API active.
2.  **Milestone 2 (Week 2)**: Unified Login/Menu flow and PWA manifest implementation.
3.  **Milestone 3 (Week 2)**: Service worker testing and offline mode validation.
4.  **Milestone 4 (Week 3)**: Final UX polish and documentation update.

## 6. Conclusion
Refactoring KNIRVSERVER to a PWA is not only feasible but highly recommended to reduce complexity and improve user accessibility. The existing Go-based system monitoring capabilities are already sufficient to replace the Node.js-specific logic in the current Electron HUD.

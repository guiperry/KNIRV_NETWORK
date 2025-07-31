

---

**Source**: KNIRVNEXUS/docs/test-system-tray.md

# System Tray Implementation Test Guide

## Overview
The Agentic Engine now includes system tray functionality that allows users to minimize the application to the system tray instead of completely closing it. When the user clicks the close button (X), they are presented with a dialog asking whether they want to minimize to tray or quit the application.

## Features Implemented

### 1. System Tray Icon
- **Location**: System tray area (notification area)
- **Icon**: Uses the application icon (resized to 16x16 for tray)
- **Tooltip**: "Agentic Engine - NFT-Agent Platform"

### 2. Close Dialog
When the user clicks the close button (X), a dialog appears with three options:
- **Minimize to Tray**: Hides the window but keeps the application running
- **Quit Application**: Completely closes the application
- **Cancel**: Keeps the window open

### 3. Tray Context Menu
Right-clicking the tray icon shows a context menu with:
- Show Agentic Engine
- Hide to Tray
- New Agent
- Import Plugin
- Settings
- Quit Agentic Engine

### 4. Tray Interactions
- **Double-click**: Toggle window visibility (show/hide)
- **Single-click** (Windows/Linux): Toggle window visibility
- **Right-click**: Show context menu

### 5. Frontend Integration
- **Settings Tab**: New "System Tray" tab in Settings (only visible in Electron)
- **Controls**: Buttons to test minimize to tray and close dialog
- **Auto-updater**: Integration with Electron's auto-updater system

## Testing Instructions

### 1. Start the Application
```bash
cd electron
./dist/linux-unpacked/agentic-engine-desktop
```

### 2. Test Close Dialog
1. Click the close button (X) in the window title bar
2. Verify the dialog appears with three options
3. Test each option:
   - Click "Minimize to Tray" - window should hide, tray icon should appear
   - Click "Quit Application" - application should close completely
   - Click "Cancel" - dialog should close, window should remain open

### 3. Test Tray Icon
1. Minimize to tray using the close dialog
2. Verify tray icon appears in system tray
3. Test tray interactions:
   - Double-click tray icon - window should restore
   - Right-click tray icon - context menu should appear
   - Test context menu items

### 4. Test Settings Integration
1. Open the application
2. Navigate to Settings
3. Look for "System Tray" tab (should only appear in Electron version)
4. Test the controls in the System Tray settings

### 5. Test Auto-updater (if applicable)
1. In System Tray settings, click "Check for Updates"
2. Verify update checking functionality works

## Technical Implementation Details

### Files Modified/Created:
1. **electron/main.js**: Added system tray functionality
2. **electron/preload.js**: Added IPC handlers for tray operations
3. **gui/src/components/SystemTrayManager.jsx**: Frontend component for tray management
4. **gui/src/components/Settings.jsx**: Added system tray tab

### Key Functions:
- `createTray()`: Creates and configures the system tray
- `showCloseDialog()`: Shows the close confirmation dialog
- `hideToTray()`: Hides window to tray
- `showMainWindow()`: Restores window from tray

### IPC Handlers:
- `minimize-to-tray`: Minimize window to tray
- `show-from-tray`: Restore window from tray
- `is-minimized-to-tray`: Check tray status
- `show-close-dialog`: Show close confirmation dialog

## Platform-Specific Behavior

### Linux
- Single-click and double-click both toggle window visibility
- Tray icon appears in system notification area
- Context menu works with right-click

### Windows (when built for Windows)
- Single-click toggles window visibility
- Double-click also toggles window visibility
- Balloon notifications supported

### macOS (when built for macOS)
- Only double-click toggles window visibility (following macOS conventions)
- Dock icon is hidden when minimized to tray
- Dock icon is shown when restored

## Troubleshooting

### Tray Icon Not Appearing
- Ensure system tray is enabled in your desktop environment
- Check if other applications can show tray icons
- Verify icon file exists and is readable

### Dialog Not Showing
- Check console for JavaScript errors
- Verify IPC communication is working
- Ensure dialog is not being blocked by window manager

### Application Not Responding
- Check backend server is running (should start automatically)
- Verify WebSocket connections are working
- Check for port conflicts (8081 for backend, 3001 for frontend)

## Expected Behavior Summary

1. **Normal Close**: Dialog appears asking user preference
2. **Minimize to Tray**: Window hides, app continues running, tray icon appears
3. **Tray Interaction**: Click/double-click restores window
4. **Context Menu**: Right-click provides quick actions
5. **Quit from Tray**: Completely closes application
6. **Settings Integration**: System tray controls available in settings

The implementation provides a seamless desktop experience where users can keep the Agentic Engine running in the background while freeing up taskbar space.


---

<div class="footer-links">


© 2025 KNIRV Network
</div>

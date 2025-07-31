

---

**Source**: KNIRVROOT/docs/Troubleshooting/completedFixes/rootnode_fix_implementation_plan.md

# Root Node Fix Implementation Plan

This document outlines a comprehensive plan to address two critical issues in the KNIRVCHAIN application:

1. **Port Conflict Resolution**: Ensuring consistent use of dynamically assigned ports when the configured port is already in use
2. **SIGABRT Crash with Systray and Webview**: Resolving conflicts between GTK main loops in the GUI components

## 1. Port Conflict Resolution

### Problem Description

The application correctly detects when the configured HTTP port (e.g., 9999) is in use and attempts to use an alternative port (e.g., 10000). However, subsequent operations still attempt to use the original port:

- The BlockchainServer initializes and attempts to listen on the original port
- The NEXT_PUBLIC_BACKEND_URL for the GUI is updated with the original port

### Root Cause Analysis

- In the `startNode` and `startNodeWithComponents` functions in `main.go`, the `cfg` parameter is a copy
- While `cfg.Port` is updated locally after `blockchainSrv.Prepare()` returns the `actualHTTPPort`, this updated port is not consistently used throughout the application
- The timing/scoping with goroutines may lead to the old port value being used for critical operations
- For the root node, there is no config file (as a security measure), so essential parameters need to be stored in the env.local file for persistence across restarts

### Implementation Plan

#### 1.1. Modify `startNode` function in `main.go`

```diff
// Blockchain HTTP Server
blockchainSrv := NewBlockchainServer(uint64(cfg.Port), bc, db, discoveryMgr, int(cfg.P2PPort))

// Prepare the server first to initialize the server field
- _, err = blockchainSrv.Prepare()
+ actualHTTPPort, err := blockchainSrv.Prepare()
if err != nil {
    log.Printf("[%s] ERROR: Failed to prepare blockchain server: %v", cfg.ChainID, err)
    log.Fatalf("[%s] FATAL: Failed to prepare blockchain server: %v", cfg.ChainID, err)
}
+ // Update the configuration with the actual port that will be used
+ if actualHTTPPort != cfg.Port {
+     log.Printf("[%s] Port %d is in use, using port %d instead", cfg.ChainID, cfg.Port, actualHTTPPort)
+     cfg.Port = actualHTTPPort
+     
+     // If this is the root node, save the new port to env.local
+     if nodeRole == config.Root {
+         rootDataDir, err := config.GetDataDir(config.Root)
+         if err == nil {
+             envLocalPath := filepath.Join(rootDataDir, "env.local")
+             if err := UpdateEnvVariable(envLocalPath, "HTTP_PORT", fmt.Sprintf("%d", actualHTTPPort)); err != nil {
+                 log.Printf("[%s] WARNING: Failed to update HTTP_PORT in env.local: %v", cfg.ChainID, err)
+             } else {
+                 log.Printf("[%s] Successfully saved HTTP_PORT=%d to env.local", cfg.ChainID, actualHTTPPort)
+             }
+         }
+     }
+ }
```

#### 1.2. Ensure consistent port usage in GUI backend URL updates

```diff
// When updating the GUI backend URL
backendURL = fmt.Sprintf("http://%s:%d", publicIP, cfg.Port)

log.Printf("[%s][%s] Attempting to update NEXT_PUBLIC_BACKEND_URL in %s to %s", nodeRole.String(), cfg.ChainID, altguiEnvPath, backendURL)
```

#### 1.3. Ensure the dynamic IP address is preserved

- The current implementation in `fetchAndStorePublicIPInfo` correctly fetches the public IP dynamically at runtime
- Ensure this function is called before any operations that require the IP address
- Maintain the existing code that updates `ROOTCHAIN_URL` with the dynamic IP:

```go
// If this is the Root node, update ROOTCHAIN_URL
if role == config.Root && utils.LastIPInfoResponse.IP != "" {
    newRootURL := fmt.Sprintf("http://%s:%d", utils.LastIPInfoResponse.IP, cfg.Port)
    SetRootchainURL(newRootURL) // This function is in constants.go
    log.Printf("Updated ROOTCHAIN_URL to: %s based on fetched public IP.", newRootURL)
}
```

#### 1.4. Enhance Root Node Configuration Persistence

For the root node, implement a comprehensive approach to store all essential parameters in the env.local file:

```go
// Function to save essential root node parameters to env.local
func saveRootNodeParameters(cfg *config.Config, rootDataDir string) error {
    envLocalPath := filepath.Join(rootDataDir, "env.local")
    log.Printf("Saving essential root node parameters to %s", envLocalPath)
    
    // Create a map of all essential parameters to save
    // Note: CHAIN_ID and MINERS_ADDRESS are constants and should not be saved in env.local
    params := map[string]string{
        "HTTP_PORT":           fmt.Sprintf("%d", cfg.Port),
        "P2P_PORT":            fmt.Sprintf("%d", cfg.P2PPort),
        "WALLET_PORT":         fmt.Sprintf("%d", cfg.WalletPort),
        "ALT_GUI_PORT":        fmt.Sprintf("%d", cfg.AltGUIPort),
        "DATABASE_PATH":       cfg.DatabasePath,
        "NEXT_PUBLIC_IP_INFO": utils.LastIPInfoResponse.IP,
    }
    
    // Save each parameter to env.local
    for key, value := range params {
        if value != "" {
            if err := UpdateEnvVariable(envLocalPath, key, value); err != nil {
                log.Printf("WARNING: Failed to update %s in env.local: %v", key, err)
                // Continue with other parameters even if one fails
            }
        }
    }
    
    log.Printf("Successfully saved essential root node parameters to env.local")
    return nil
}
```

This function should be called:
1. After successful port conflict resolution
2. At the end of the node startup process
3. Whenever a critical parameter changes during runtime

## 2. SIGABRT Crash with Systray and Webview

### Problem Description

The application crashes with SIGABRT when the webview GUI is enabled, with the stack trace pointing to `github.com/getlantern/systray`.

### Root Cause Analysis

- **webview_go** (github.com/webview/webview_go): Takes control of the main GUI event loop (GTK on Linux) when `webview.Run()` is called
- **systray** (github.com/getlantern/systray): The `systray.Run()` function also attempts to run its own GTK main loop on Linux
- Running two GTK main loops in the same process, especially from different threads, is not supported and leads to crashes

### Implementation Plan

#### 2.1. Modify `createWebViewWindow` in `altgui.go`

```diff
// createWebViewWindow creates and runs the WebView window
func (g *GUI) createWebViewWindow() {
    w := webview_go.New(true)
    g.webviewWindow = w
    defer w.Destroy()

    w.SetTitle("KNIRVCHAIN")
    w.SetSize(1200, 800, 0)
    w.Navigate(fmt.Sprintf("http://localhost:%d/", g.apiPort)) // Navigate to the root served by Go

    // Initialize the systray manager
    g.systrayManager = NewSystrayManager(w, g.cancelFunc, g.wg)

    // Run the systray manager in a goroutine since it's blocking
-   go g.systrayManager.Run()
+   // systray.Run() calls gtk_main() on Linux.
+   // webview.Run() also manages the GTK main loop on Linux.
+   // Running two GTK main loops in the same process, especially from different goroutines,
+   // is a common cause of crashes (like SIGABRT).
+   // To prevent this crash, we should not call g.systrayManager.Run() when the webview
+   // is also going to run its GTK loop.
+   log.Println("INFO: Systray manager run explicitly disabled in webview mode to prevent GTK main loop conflict.")
+   // Consider enabling this only if webview is not active or if a compatible systray library is used.
+   // go g.systrayManager.Run()

    // Bind functions for minimizing to tray and quitting
    w.Bind("go.minimizeToTray", func() {
        log.Println("Minimize to tray requested from UI")
        if g.systrayManager != nil {
            // ...
        }
    })
    
    // ...
}
```

#### 2.2. Ensure proper thread handling in `Run` method

```diff
// Run starts the alternative GUI
func (g *GUI) Run() {
+   // Lock this goroutine to the main OS thread for proper GTK handling
+   runtime.LockOSThread()
+   // webview.Run() is blocking, so UnlockOSThread might be deferred or handled on exit.
    
    // Start the API server in a goroutine
    go g.startAPIServer()

    // Wait for the API server to be ready
    <-g.apiServerReady

    // Check if the blockchain server is running
    if g.isBlockchainServerRunning() {
        log.Printf("Blockchain server detected at port %d", g.cfg.Port)
    } else {
        log.Printf("Blockchain server not detected, using fallback implementations")
    }

    // Create and run the WebView window
    g.createWebViewWindow()
}
```

## 3. Testing Plan

### 3.1. Port Conflict Testing

1. **Simulate Port Conflict**:
   - Start another service on port 9999 before launching the application
   - Verify that the application detects the conflict and uses an alternative port

2. **Verify Consistent Port Usage**:
   - Check logs to confirm that all components use the same alternative port
   - Verify that the BlockchainServer listens on the alternative port
   - Confirm that the GUI backend URL is updated with the alternative port

3. **Test Dynamic IP Address**:
   - Verify that the application correctly fetches and uses the public IP at runtime
   - Confirm that the ROOTCHAIN_URL is updated with the correct IP and port

### 3.2. GUI Crash Testing

1. **Test GUI Stability**:
   - Launch the application with the GUI enabled
   - Verify that the application does not crash with SIGABRT
   - Test basic GUI functionality to ensure it works without the systray

2. **Test GUI Performance**:
   - Monitor memory and CPU usage to ensure the GUI performs efficiently
   - Test responsiveness of the GUI interface

## 4. Root Node Parameter Loading

To ensure that the root node can properly load its configuration from env.local at startup, we need to implement a function to read these parameters during initialization:

```go
// Function to load essential root node parameters from env.local
func loadRootNodeParameters(cfg *config.Config) error {
    rootDataDir, err := config.GetDataDir(config.Root)
    if err != nil {
        return fmt.Errorf("failed to get root data directory: %w", err)
    }
    
    envLocalPath := filepath.Join(rootDataDir, "env.local")
    log.Printf("Loading essential root node parameters from %s", envLocalPath)
    
    // Check if env.local exists
    if _, err := os.Stat(envLocalPath); os.IsNotExist(err) {
        log.Printf("env.local file not found at %s, will use default parameters", envLocalPath)
        return nil
    }
    
    // Read env.local file
    content, err := os.ReadFile(envLocalPath)
    if err != nil {
        return fmt.Errorf("failed to read env.local file: %w", err)
    }
    
    // Parse env.local file
    lines := strings.Split(string(content), "\n")
    for _, line := range lines {
        if line == "" || strings.HasPrefix(line, "#") {
            continue // Skip empty lines and comments
        }
        
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue // Skip malformed lines
        }
        
        key := strings.TrimSpace(parts[0])
        value := strings.TrimSpace(parts[1])
        
        // Remove quotes if present
        value = strings.Trim(value, "\"'")
        
        // Update config based on key
        switch key {
        case "HTTP_PORT":
            if port, err := strconv.ParseUint(value, 10, 64); err == nil {
                cfg.Port = int(port)
                log.Printf("Loaded HTTP_PORT=%d from env.local", cfg.Port)
            }
        case "P2P_PORT":
            if port, err := strconv.ParseUint(value, 10, 64); err == nil {
                cfg.P2PPort = int(port)
                log.Printf("Loaded P2P_PORT=%d from env.local", cfg.P2PPort)
            }
        case "WALLET_PORT":
            if port, err := strconv.ParseUint(value, 10, 64); err == nil {
                cfg.WalletPort = int(port)
                log.Printf("Loaded WALLET_PORT=%d from env.local", cfg.WalletPort)
            }
        case "ALT_GUI_PORT":
            if port, err := strconv.ParseUint(value, 10, 64); err == nil {
                cfg.AltGUIPort = uint64(port)
                log.Printf("Loaded ALT_GUI_PORT=%d from env.local", cfg.AltGUIPort)
            }
        case "DATABASE_PATH":
            cfg.DatabasePath = value
            log.Printf("Loaded DATABASE_PATH=%s from env.local", cfg.DatabasePath)
        // Note: CHAIN_ID and MINERS_ADDRESS are constants and should not be loaded from env.local
        }
    }
    
    log.Printf("Successfully loaded root node parameters from env.local")
    return nil
}
```

This function should be called early in the initialization process for the root node, before any services are started:

```diff
// In main.go, after determining that this is a root node
if nodeRole == config.Root {
+   // Load parameters from env.local for root node
+   if err := loadRootNodeParameters(cfg); err != nil {
+       log.Printf("WARNING: Failed to load parameters from env.local: %v", err)
+       // Continue with default or CLI-provided parameters
+   }
    
    // Proceed with node initialization
    // ...
}
```

## 5. Implementation Timeline

1. **Day 1**: Implement port conflict resolution changes
   - Modify `startNode` and related functions
   - Implement `saveRootNodeParameters` function
   - Test port conflict handling

2. **Day 2**: Implement root node parameter persistence
   - Implement `loadRootNodeParameters` function
   - Integrate parameter loading into the startup process
   - Test parameter persistence across restarts

3. **Day 3**: Implement GUI crash fix
   - Modify `createWebViewWindow` and `Run` methods
   - Test GUI stability

4. **Day 4**: Integration testing and documentation
   - Perform comprehensive testing of all fixes together
   - Update documentation to reflect changes

## 6. Future Considerations

### 6.1. Alternative Systray Implementation

For a more robust solution to the systray issue, consider:

1. **Using a systray library designed to integrate with an existing GTK main loop**
2. **Implementing a custom systray solution that works with webview's GTK loop**
3. **Creating a configuration option to choose between webview GUI and systray functionality**

### 6.2. Enhanced Port Conflict Resolution

Consider implementing:

1. **A more sophisticated port selection algorithm**
2. **A user-configurable port range for automatic selection**
3. **Fallback mechanisms if all ports in a range are occupied**

### 6.3. Enhanced Root Node Configuration Management

For more robust root node configuration:

1. **Implement encryption for sensitive parameters in env.local**
2. **Add version tracking for the env.local format**
3. **Create a backup/restore mechanism for env.local**
4. **Develop a UI for viewing and editing root node configuration**

## 7. Conclusion

This implementation plan addresses the critical issues while significantly improving the root node's configuration management:

1. **Port Conflict Resolution**: Ensures consistent port usage throughout the application when the configured port is already in use.

2. **Root Node Parameter Persistence**: Stores all essential parameters in the env.local file for the root node, ensuring configuration persistence across restarts without relying on a config file (maintaining the security measure).

3. **Parameter Loading at Startup**: Implements a mechanism to load the saved parameters from env.local during root node initialization, ensuring consistent configuration.

4. **SIGABRT Crash Fix**: Prevents the crash by avoiding conflicting GTK main loops between the systray and webview components.

5. **Dynamic IP Address Handling**: Preserves the existing functionality that dynamically sets the root node IP at runtime.

By following this plan, the application will be more robust, stable, and user-friendly. The root node will maintain consistent configuration across restarts, even when port conflicts occur, and the GUI will operate without crashes, providing a better experience for all users.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>

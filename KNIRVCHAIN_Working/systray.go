//go:build !no_webview
// +build !no_webview

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"

	"KNIRVCHAIN/config"

	"github.com/getlantern/systray"

	"github.com/skratchdot/open-golang/open"
	webview_go "github.com/webview/webview_go"
)

// iconData will be loaded from the altgui/out directory at runtime
var iconData []byte

// SystrayManager handles the system tray functionality
type SystrayManager struct {
	webview         webview_go.WebView
	isWebviewHidden bool
	cancelFunc      context.CancelFunc
	wg              *sync.WaitGroup
	quitChan        chan struct{}
	cfg             *config.Config
	menuItems       map[string]*systray.MenuItem
}

// NewSystrayManager creates a new SystrayManager and binds JS functions.
// It takes a webview instance, a context cancel function for application shutdown,
// and a wait group for coordinating shutdown.
func NewSystrayManager(webview webview_go.WebView, cancelFunc context.CancelFunc, wg *sync.WaitGroup, cfg ...*config.Config) *SystrayManager {
	sm := &SystrayManager{
		webview:         webview,
		isWebviewHidden: false,
		cancelFunc:      cancelFunc,
		wg:              wg,
		quitChan:        make(chan struct{}),
		menuItems:       make(map[string]*systray.MenuItem),
	}

	if len(cfg) > 0 && cfg[0] != nil {
		sm.cfg = cfg[0]
	}

	// Bind Go methods to JavaScript
	webview.Bind("minimizeToTray", sm.minimizeToTray)
	webview.Bind("quitApplication", sm.quitApplication)

	return sm
}

// Run starts the system tray. This function is blocking and should typically be run in a goroutine.
func (sm *SystrayManager) Run() {
	// Create a context with cancellation for managing goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure all goroutines are cancelled when this function returns

	// Goroutine to listen on quitChan. When quitChan is closed, it calls systray.Quit().
	go func() {
		select {
		case <-sm.quitChan:
			log.Println("SystrayManager: Received quit signal from application logic. Telling systray to quit.")
			systray.Quit() // This will trigger the onExit callback passed to systray.Run below.
		case <-ctx.Done():
			// Context was cancelled, clean exit
			return
		}
	}()

	// We're now calling showConfirmationDialog directly from HandleCloseAttempt
	// so we don't need a goroutine to listen on showDialogChan anymore

	// systray.Run is blocking and must be called directly.
	// It takes an onReady and onExit callback.
	// The onExit callback is executed when systray.Quit() is called and the native event loop terminates.
	systray.Run(sm.onReady, func() { // This is systray's onExit
		log.Println("SystrayManager: systray.Run onExit callback triggered (systray is quitting).")

		// Cancel the context to clean up goroutines
		cancel()

		// Clear the webview reference to prevent further access
		sm.webview = nil

		if sm.cancelFunc != nil {
			log.Println("SystrayManager: Calling main context cancelFunc to signal application shutdown.")
			sm.cancelFunc() // Signal the rest of the application to shut down.
		}
	})
	log.Println("SystrayManager.Run: systray.Run has completed. Systray has exited.")
}

// onReady is called when the system tray is ready
func (sm *SystrayManager) onReady() {
	// Try to load icon from the altgui/out directory
	var err error
	iconData, err = loadIconFromAltGUI()
	if err != nil {
		log.Printf("Failed to load icon from altgui/out: %v", err)
	}

	// Set icon (fallback to a default if icon is not available)
	if len(iconData) > 0 {
		systray.SetIcon(iconData)
	}

	// Set title and tooltip
	systray.SetTitle("KNIRVCHAIN")
	systray.SetTooltip("KNIRVCHAIN is running")

	// Add menu items
	mShow := systray.AddMenuItem("Show", "Show the application window")
	mHide := systray.AddMenuItem("Hide", "Hide the application window")
	systray.AddSeparator()
	mViewLogs := systray.AddMenuItem("View Logs", "View application logs")
	mOpenDataDir := systray.AddMenuItem("Open Data Directory", "Open the data directory")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Handle menu clicks
	// Goroutine to handle menu item clicks. This is the standard pattern.
	// systray.Run() handles dispatching events to these channels.
	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				log.Println("Show clicked")
				if sm.webview != nil {
					sm.showWebview()
				}
			case <-mHide.ClickedCh:
				log.Println("Hide clicked")
				if sm.webview != nil {
					sm.hideWebview()
				}
			case <-mViewLogs.ClickedCh:
				log.Println("View Logs clicked")
				sm.viewLogs()
			case <-mOpenDataDir.ClickedCh:
				log.Println("Open Data Directory clicked")
				sm.openDataDirectory()
			case <-mQuit.ClickedCh:
				log.Println("Quit clicked")
				// Closing quitChan signals the goroutine in sm.Run() to call systray.Quit()
				close(sm.quitChan)
				return // Exit this goroutine as systray is shutting down
			}
		}
	}()
}

// Note: We're using an inline function for the onExit callback in systray.Run
// instead of a separate method

// showWebview shows the webview window
func (sm *SystrayManager) showWebview() {
	if sm.webview != nil {
		// Store a local reference to avoid race conditions
		webview := sm.webview
		webview.Dispatch(func() {
			// Check again inside the dispatch to ensure webview is still valid
			if sm.webview != nil {
				webview.SetSize(1200, 800, 0)
				sm.isWebviewHidden = false
				log.Println("SystrayManager: WebView shown")
			}
		})
	} else {
		log.Println("SystrayManager: Cannot show WebView - it is nil")
	}
}

// hideWebview hides the webview window
func (sm *SystrayManager) hideWebview() {
	if sm.webview != nil {
		// Store a local reference to avoid race conditions
		webview := sm.webview
		webview.Dispatch(func() {
			// Check again inside the dispatch to ensure webview is still valid
			if sm.webview != nil {
				webview.SetSize(0, 0, 0)
				sm.isWebviewHidden = true
				log.Println("SystrayManager: WebView hidden")
			}
		})
	} else {
		log.Println("SystrayManager: Cannot hide WebView - it is nil")
	}
}

// Note: The showConfirmationDialog method has been removed as it's no longer used.
// Dialog functionality has been moved to the UI layer in altgui.go

// minimizeToTray hides the webview window and updates state
func (sm *SystrayManager) minimizeToTray() {
	log.Println("SystrayManager: Minimizing to tray")
	if sm.webview != nil {
		sm.hideWebview()
		sm.isWebviewHidden = true
	} else {
		log.Println("SystrayManager: Cannot minimize - WebView is nil")
	}
}

// quitApplication triggers application shutdown
func (sm *SystrayManager) quitApplication() {
	log.Println("SystrayManager: Quitting application")

	// Only close the channel if it's not already closed
	select {
	case <-sm.quitChan:
		// Channel is already closed
		log.Println("SystrayManager: Quit channel already closed")
	default:
		// Channel is still open, close it
		close(sm.quitChan)
	}
}

// HandleCloseAttempt handles when the user tries to close the application
// This method is kept for backward compatibility but is no longer used
// as we now handle close events directly in the UI
func (sm *SystrayManager) HandleCloseAttempt() {
	log.Println("SystrayManager: HandleCloseAttempt called (deprecated)")

	// We no longer need this method as we're handling close events in the UI
	// But we'll keep it for backward compatibility
}

// loadIconFromAltGUI loads the icon from the altgui/out directory
func loadIconFromAltGUI() ([]byte, error) {
	// Try to find the icon in the altgui/out directory
	// Common icon paths in the Next.js output
	iconPaths := []string{
		"altgui/out/favicon.ico",
		"altgui/out/icon.png",
		"altgui/out/images/icon.png",
		"altgui/out/assets/icon.png",
		"altgui/out/public/favicon.ico",
		"altgui/out/public/icon.png",
	}

	// Try each path
	for _, path := range iconPaths {
		if _, err := os.Stat(path); err == nil {
			// File exists, read it
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			log.Printf("Loaded icon from %s", path)
			return data, nil
		}
	}

	// If no icon found, try to find any .ico or .png file in the altgui/out directory
	var iconData []byte
	err := filepath.Walk("altgui/out", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is an icon or png
		ext := filepath.Ext(path)
		if ext == ".ico" || ext == ".png" {
			// Found an icon, read it
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			iconData = data
			log.Printf("Found and loaded icon from %s", path)
			return filepath.SkipDir // Stop walking, we found an icon
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(iconData) == 0 {
		return nil, os.ErrNotExist
	}

	return iconData, nil
}

func (sm *SystrayManager) viewLogs() {
	// Get the log directory - for now, just use a default location
	logDir := filepath.Join(".", "logs")
	logFilePath := filepath.Join(logDir, "KNIRVCHAIN.log")

	// Check if the log file exists
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		log.Printf("Systray: Log file does not exist at %s", logFilePath)
		return
	}

	if err := open.Run(logFilePath); err != nil {
		log.Printf("Systray: Failed to open log file %s: %v", logFilePath, err)
	}
}

func (sm *SystrayManager) openDataDirectory() {
	// Use a default data directory location
	dataDir := filepath.Join(".", "data")

	// Check if the directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		log.Printf("Systray: Data directory does not exist at %s", dataDir)
		return
	}

	if err := open.Run(dataDir); err != nil {
		log.Printf("Systray: Failed to open data directory %s: %v", dataDir, err)
	}
}

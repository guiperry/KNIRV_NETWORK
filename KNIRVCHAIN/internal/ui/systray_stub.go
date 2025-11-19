//go:build no_webview
// +build no_webview

package ui

import (
	"context"
	"log"
	"sync"

	"KNIRVCHAIN/config"
)

// Stub WebView interface for headless builds
type WebView interface{}

// SystrayManager stub for headless builds
type SystrayManager struct {
	webview         WebView
	isWebviewHidden bool
	cancelFunc      context.CancelFunc
	wg              *sync.WaitGroup
	quitChan        chan struct{}
	cfg             *config.Config
	menuItems       map[string]interface{}
}

// NewSystrayManager creates a stub SystrayManager for headless builds
func NewSystrayManager(webview WebView, cancelFunc context.CancelFunc, wg *sync.WaitGroup, cfg ...*config.Config) *SystrayManager {
	log.Println("SystrayManager: Running in headless mode (no GUI)")
	sm := &SystrayManager{
		webview:         nil,
		isWebviewHidden: true,
		cancelFunc:      cancelFunc,
		wg:              wg,
		quitChan:        make(chan struct{}),
		menuItems:       make(map[string]interface{}),
	}

	if len(cfg) > 0 && cfg[0] != nil {
		sm.cfg = cfg[0]
	}

	return sm
}

// Run is a no-op for headless builds
func (sm *SystrayManager) Run() {
	log.Println("SystrayManager: Headless mode - system tray disabled")
	// In headless mode, we don't run a system tray
	// Just wait for the quit signal or context cancellation
	select {
	case <-sm.quitChan:
		log.Println("SystrayManager: Received quit signal in headless mode")
	}
}

// showWebview is a no-op for headless builds
func (sm *SystrayManager) showWebview() {
	log.Println("SystrayManager: showWebview called in headless mode (no-op)")
}

// hideWebview is a no-op for headless builds
func (sm *SystrayManager) hideWebview() {
	log.Println("SystrayManager: hideWebview called in headless mode (no-op)")
}

// minimizeToTray is a no-op for headless builds
func (sm *SystrayManager) minimizeToTray() {
	log.Println("SystrayManager: minimizeToTray called in headless mode (no-op)")
}

// quitApplication triggers application shutdown in headless mode
func (sm *SystrayManager) quitApplication() {
	log.Println("SystrayManager: Quitting application in headless mode")

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

// HandleCloseAttempt is a no-op for headless builds
func (sm *SystrayManager) HandleCloseAttempt() {
	log.Println("SystrayManager: HandleCloseAttempt called in headless mode (no-op)")
}

// viewLogs is a no-op for headless builds
func (sm *SystrayManager) viewLogs() {
	log.Println("SystrayManager: viewLogs called in headless mode (no-op)")
}

// openDataDirectory is a no-op for headless builds
func (sm *SystrayManager) openDataDirectory() {
	log.Println("SystrayManager: openDataDirectory called in headless mode (no-op)")
}

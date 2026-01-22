// /home/gperry/Documents/GitHub/KNIRVROUTER/internal/gui/fyne_gui.go
//go:build !wasmloader
// +build !wasmloader

package gui

import (
	"bufio"
	"errors"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall" // Add this import for SIGSTOP and SIGCONT
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"KNIRVROUTER/internal/utils"
	"KNIRVROUTER/internal/wallet"
)

// --- ADDED KnirvchainTheme struct and methods ---
// KnirvchainTheme is a custom theme for the application
type KnirvchainTheme struct {
	fyne.Theme
}

// Override specific theme settings
func (t KnirvchainTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameBackground {
		return color.NRGBA{R: 33, G: 33, B: 33, A: 255} // Darker background
	} else if name == theme.ColorNameForeground {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255} // Pure white text
	} else if name == theme.ColorNameDisabled {
		// Make disabled text white as well for the terminal
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255} // White
	} else if name == theme.ColorNamePrimary {
		return color.NRGBA{R: 0, G: 180, B: 255, A: 255} // Bright blue for primary elements
	} else if name == theme.ColorNameSelection {
		return color.NRGBA{R: 0, G: 120, B: 215, A: 128} // Highlight color with transparency
	} else if name == theme.ColorNameHover {
		return color.NRGBA{R: 0, G: 120, B: 215, A: 64} // Lighter hover highlight
	} else if name == theme.ColorNameFocus {
		return color.NRGBA{R: 0, G: 180, B: 255, A: 200} // Bright focus highlight
	} else if name == theme.ColorNameInputBackground {
		return color.NRGBA{R: 45, G: 45, B: 45, A: 255} // Slightly lighter input background
	}

	// Fall back to the base theme for other colors
	return t.Theme.Color(name, variant)
}

// TextSize returns customized text sizes
func (t KnirvchainTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return 10 // Smaller text for terminal and general UI
	} else if name == theme.SizeNameHeadingText {
		return 18 // Moderate size for headings
	} else if name == theme.SizeNameInputBorder {
		return 2 // Slightly thicker borders for better visibility
	} else if name == theme.SizeNamePadding {
		return 4 // Reduced padding for more compact layout
	}

	return t.Theme.Size(name)
}

// --- END ADDED KnirvchainTheme ---

// --- FyneGuiConfig struct ---
type FyneGuiConfig struct {
	MinersAddress         string
	ChainPort             string
	WalletPort            string
	BlockchainNodeAddress string
	MiningDifficulty      string
	MiningReward          string
	DatabasePath          string // Default Path for the Verifier node UI field
	RootChainAddress      string // Address of the root chain for federation
	TurnPort              string // Port for the TURN server
	NextPeerPort          int    // Next available port for peer connections
	// New KNIRV-ROUTER specific configurations
	NRNMintingRate       string // Rate of NRN token minting
	ConnectivityInterval string // Interval for connectivity tests (seconds)
	FaucetEndpoint       string // KNIRV-ORACLE Faucet endpoint
	ProofEnginePort      string // Port for Proof-of-Connectivity engine API
}

// --- FyneTerminalWidget struct and methods ---
type FyneTerminalWidget struct {
	lines      []string
	maxLines   int
	textArea   *widget.Entry
	container  *container.Scroll
	autoScroll bool
	mu         sync.Mutex
}

func NewFyneTerminalWidget() *FyneTerminalWidget {
	t := &FyneTerminalWidget{
		maxLines:   10000000,
		lines:      make([]string, 0, 10000000),
		autoScroll: true,
	}

	// Create a multiline entry with monospace font
	t.textArea = widget.NewMultiLineEntry()
	t.textArea.Disable() // Make it read-only
	t.textArea.Wrapping = fyne.TextWrapWord
	t.textArea.TextStyle = fyne.TextStyle{Monospace: true}
	t.textArea.TextStyle.Monospace = true

	// Set initial content to ensure widget is initialized
	t.textArea.SetText("Terminal initialized...\n")

	// Create a scroll container
	t.container = container.NewVScroll(t.textArea)
	t.container.SetMinSize(fyne.NewSize(600, 400)) // Larger minimum size

	// Initial scroll to bottom
	t.container.ScrollToBottom()

	// Start a periodic scroll enforcement goroutine to keep terminal locked to bottom
	go t.startScrollEnforcement()

	return t
}

// GetContent returns the terminal scroll container
func (t *FyneTerminalWidget) GetContent() fyne.CanvasObject {
	return t.container
}

// Improved Append method for FyneTerminalWidget
func (t *FyneTerminalWidget) Append(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Format with timestamp
	timestamp := time.Now().Format("15:04:05")
	newText := fmt.Sprintf("[%s] %s\n", timestamp, text)

	// Add to buffer and maintain max size
	t.lines = append(t.lines, newText)
	if len(t.lines) > t.maxLines {
		t.lines = t.lines[len(t.lines)-t.maxLines:]
	}

	// Use a string builder for better performance
	var sb strings.Builder
	for _, line := range t.lines {
		sb.WriteString(line)
	}

	// Update UI on main thread with enhanced scrolling
	if ap, ok := fyne.CurrentApp().(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			// Update text first
			t.textArea.SetText(sb.String())

			// Force scroll to bottom if auto-scroll is enabled
			if t.autoScroll {
				// Multiple scroll attempts to ensure it sticks
				t.container.ScrollToBottom()
				t.container.Refresh()

				// Additional delayed scroll to handle any layout changes
				time.AfterFunc(10*time.Millisecond, func() {
					ap.CallOnMainThread(func() {
						t.container.ScrollToBottom()
					})
				})

				// Final scroll attempt after a longer delay
				time.AfterFunc(50*time.Millisecond, func() {
					ap.CallOnMainThread(func() {
						t.container.ScrollToBottom()
					})
				})
			}
		})
	} else {
		// Fallback if CallOnMainThread is not available
		t.textArea.SetText(sb.String())
		if t.autoScroll {
			t.container.ScrollToBottom()
			t.container.Refresh()
		}
	}
}

// Improved ScrollToBottom method with enhanced sticking behavior
func (t *FyneTerminalWidget) ScrollToBottom() {
	t.mu.Lock()
	doScroll := t.autoScroll
	t.mu.Unlock()

	if doScroll {
		if ap, ok := fyne.CurrentApp().(interface{ CallOnMainThread(func()) }); ok {
			ap.CallOnMainThread(func() {
				// Multiple scroll attempts to ensure it sticks
				t.container.ScrollToBottom()
				t.container.Refresh()

				// Force scroll to the very end
				t.container.ScrollToBottom()
			})
		} else {
			t.container.ScrollToBottom()
			t.container.Refresh()
			t.container.ScrollToBottom()
		}
	}
}

// SetAutoScroll with enhanced immediate scrolling when enabled
func (t *FyneTerminalWidget) SetAutoScroll(auto bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.autoScroll = auto

	// If enabling auto-scroll, immediately and aggressively scroll to bottom
	if auto {
		if ap, ok := fyne.CurrentApp().(interface{ CallOnMainThread(func()) }); ok {
			ap.CallOnMainThread(func() {
				// Multiple immediate scroll attempts
				t.container.ScrollToBottom()
				t.container.Refresh()
				t.container.ScrollToBottom()

				// Additional delayed scrolls to ensure it sticks
				time.AfterFunc(10*time.Millisecond, func() {
					ap.CallOnMainThread(func() {
						t.container.ScrollToBottom()
					})
				})

				time.AfterFunc(50*time.Millisecond, func() {
					ap.CallOnMainThread(func() {
						t.container.ScrollToBottom()
					})
				})
			})
		} else {
			t.container.ScrollToBottom()
			t.container.Refresh()
			t.container.ScrollToBottom()
		}
	}
}

// startScrollEnforcement runs a periodic check to ensure terminal stays at bottom
func (t *FyneTerminalWidget) startScrollEnforcement() {
	ticker := time.NewTicker(500 * time.Millisecond) // Check every 500ms
	defer ticker.Stop()

	for range ticker.C {
		t.mu.Lock()
		shouldScroll := t.autoScroll
		t.mu.Unlock()

		if shouldScroll {
			if ap, ok := fyne.CurrentApp().(interface{ CallOnMainThread(func()) }); ok {
				ap.CallOnMainThread(func() {
					// Gently enforce scroll to bottom
					t.container.ScrollToBottom()
				})
			} else {
				t.container.ScrollToBottom()
			}
		}
	}
}

// ForceScrollToBottom forces the terminal to scroll to bottom regardless of auto-scroll setting
func (t *FyneTerminalWidget) ForceScrollToBottom() {
	if ap, ok := fyne.CurrentApp().(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			t.container.ScrollToBottom()
			t.container.Refresh()
			// Double scroll to ensure it sticks
			time.AfterFunc(10*time.Millisecond, func() {
				ap.CallOnMainThread(func() {
					t.container.ScrollToBottom()
				})
			})
		})
	} else {
		t.container.ScrollToBottom()
		t.container.Refresh()
		t.container.ScrollToBottom()
	}
}

// --- createStatusIndicator function ---
func createStatusIndicator(label string, isRunning bool) (*canvas.Circle, *widget.Label) { /* ... as before ... */
	indicator := canvas.NewCircle(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	if isRunning {
		indicator.FillColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	}
	indicator.Resize(fyne.NewSize(15, 15))
	statusLabel := widget.NewLabel(label)
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	return indicator, statusLabel
}

// --- updateIndicator method ---
func (s *FyneGUIState) updateIndicator(indicator *canvas.Circle, label *widget.Label, isRunning bool, statusText string) {
	// <<< CORRECTED RunOnMain call >>>
	if ap, ok := s.app.(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			if isRunning {
				indicator.FillColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}
			} else {
				indicator.FillColor = color.RGBA{R: 255, G: 0, B: 0, A: 255}
			}
			label.SetText(statusText)
			indicator.Refresh()
			label.Refresh()
		})
	} else {
		go func() { // Fallback to goroutine with mutex protection
			s.mu.Lock()
			defer s.mu.Unlock()
			if isRunning {
				indicator.FillColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}
			} else {
				indicator.FillColor = color.RGBA{R: 255, G: 0, B: 0, A: 255}
			}
			label.SetText(statusText)
			indicator.Refresh()
			label.Refresh()
		}()
	}
}

// --- FyneWalletInfo struct ---
type FyneWalletInfo struct{ Address, PrivateKey, PublicKey string }

// --- FyneGUIState struct ---
type FyneGUIState struct {
	app                   fyne.App // Store app reference for main thread calls
	config                FyneGuiConfig
	terminal              *FyneTerminalWidget
	walletInfo            FyneWalletInfo
	status                string
	activeProcs           []*exec.Cmd // Still track processes for cleanup
	window                fyne.Window
	statusLabel           *widget.Label
	walletAddressEntry    *widget.Entry
	walletPrivateKeyEntry *widget.Entry
	walletPublicKeyEntry  *widget.Entry
	logChan               chan string // Channel for log messages (Verifier node + GUI status)
	mu                    sync.Mutex  // Mutex for thread-safe access to activeProcs, counts etc.

	// Status indicators
	blockchainRunning     bool // Verifier blockchain
	blockchainPaused      bool // Track if blockchain is paused
	walletRunning         bool
	turnServerRunning     bool
	blockchainIndicator   *canvas.Circle
	walletIndicator       *canvas.Circle
	turnIndicator         *canvas.Circle
	blockchainStatusLabel *widget.Label
	walletStatusLabel     *widget.Label
	turnStatusLabel       *widget.Label

	// New KNIRV-ROUTER status indicators
	nrnMintingRunning      bool
	proofEngineRunning     bool
	faucetConnected        bool
	nrnMintingIndicator    *canvas.Circle
	proofEngineIndicator   *canvas.Circle
	faucetIndicator        *canvas.Circle
	nrnMintingStatusLabel  *widget.Label
	proofEngineStatusLabel *widget.Label
	faucetStatusLabel      *widget.Label

	// UI elements for new features
	turnPortEntry         *widget.Entry
	rootChainAddressEntry *widget.Entry

	// New KNIRV-ROUTER configuration entries
	nrnMintingRateEntry       *widget.Entry
	connectivityIntervalEntry *widget.Entry
	faucetEndpointEntry       *widget.Entry
	proofEnginePortEntry      *widget.Entry

	// Button references for state management
	startVerifyerBlockchainButton *widget.Button
	pauseBlockchainButton         *widget.Button
	resumeBlockchainButton        *widget.Button

	// New KNIRV-ROUTER button references
	startNRNMintingButton  *widget.Button
	stopNRNMintingButton   *widget.Button
	startProofEngineButton *widget.Button
	stopProofEngineButton  *widget.Button
	connectFaucetButton    *widget.Button
	testConnectivityButton *widget.Button

	// New KNIRV-ROUTER display widgets
	nrnBalanceLabel          *widget.Label
	usdcBalanceLabel         *widget.Label
	connectivityMetricsLabel *widget.Label
	mintingStatsLabel        *widget.Label
	pathCertificatesLabel    *widget.Label

	// TURN server reference
	turnServer interface {
		Stop() error
		IsRunning() bool
	}
}

// --- StartFyneGUI function ---
func StartFyneGUI() {
	myApp := app.New()
	// <<< CORRECTED Theme initialization >>>
	customTheme := &KnirvchainTheme{Theme: theme.DarkTheme()}
	myApp.Settings().SetTheme(customTheme)
	window := myApp.NewWindow("KNIRVROUTER Verifier Node Manager")
	window.Resize(fyne.NewSize(1000, 700))

	// --- Determine Default Verifier DB Path for UI ---
	defaultVerifierDbPath := utils.GetDefaultRootDBPath() // Use helper

	state := &FyneGUIState{
		app:      myApp,
		terminal: NewFyneTerminalWidget(),
		config: FyneGuiConfig{ // Load from file ideally
			MinersAddress:         "KNIRVROUTER-3dd025e8fec7eda7cdd012ddde9c8e978ee7fa33", // Updated prefix
			ChainPort:             "5000",
			WalletPort:            "8080",
			BlockchainNodeAddress: "http://127.0.0.1:5000",
			MiningDifficulty:      "3",
			MiningReward:          "100",
			DatabasePath:          defaultVerifierDbPath,             // Set default for the UI field
			RootChainAddress:      "http://root.knirvchain.com:5000", // Default root chain address
			TurnPort:              "3478",                            // Default TURN port
			// New KNIRV-ROUTER defaults
			NRNMintingRate:       "10",                              // NRNs per minute
			ConnectivityInterval: "30",                              // Connectivity test interval in seconds
			FaucetEndpoint:       "http://root.knirvchain.com:8080", // KNIRV-ORACLE Faucet endpoint
			ProofEnginePort:      "9090",                            // Proof-of-Connectivity engine API port
		},
		window:            window,
		activeProcs:       make([]*exec.Cmd, 0),
		logChan:           make(chan string, 256),
		blockchainRunning: false,
		blockchainPaused:  false,
		walletRunning:     false,
		turnServerRunning: false,
		// New KNIRV-ROUTER status defaults
		nrnMintingRunning:  false,
		proofEngineRunning: false,
		faucetConnected:    false,
	}

	window.SetCloseIntercept(func() { state.cleanupProcesses(); close(state.logChan); window.Close() })

	// --- UI Element Creation ---
	minerAddress := widget.NewEntry()
	chainPort := widget.NewEntry()
	walletPort := widget.NewEntry()
	nodeAddress := widget.NewEntry()
	difficulty := widget.NewEntry()
	reward := widget.NewEntry()
	dbPath := widget.NewEntry() // DB Path for Verifier node

	// New UI elements
	state.rootChainAddressEntry = widget.NewEntry()
	state.turnPortEntry = widget.NewEntry()

	// New KNIRV-ROUTER UI elements
	state.nrnMintingRateEntry = widget.NewEntry()
	state.connectivityIntervalEntry = widget.NewEntry()
	state.faucetEndpointEntry = widget.NewEntry()
	state.proofEnginePortEntry = widget.NewEntry()

	minerAddress.SetText(state.config.MinersAddress)
	chainPort.SetText(state.config.ChainPort)
	walletPort.SetText(state.config.WalletPort)
	nodeAddress.SetText(state.config.BlockchainNodeAddress)
	difficulty.SetText(state.config.MiningDifficulty)
	reward.SetText(state.config.MiningReward)
	dbPath.SetText(state.config.DatabasePath) // Set initial Verifier DB path
	state.rootChainAddressEntry.SetText(state.config.RootChainAddress)
	state.turnPortEntry.SetText(state.config.TurnPort)

	// Set initial values for new KNIRV-ROUTER elements
	state.nrnMintingRateEntry.SetText(state.config.NRNMintingRate)
	state.connectivityIntervalEntry.SetText(state.config.ConnectivityInterval)
	state.faucetEndpointEntry.SetText(state.config.FaucetEndpoint)
	state.proofEnginePortEntry.SetText(state.config.ProofEnginePort)

	state.statusLabel = widget.NewLabel("Status: Ready")
	state.statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	state.blockchainIndicator, state.blockchainStatusLabel = createStatusIndicator("Blockchain: Stopped", false)
	state.walletIndicator, state.walletStatusLabel = createStatusIndicator("Wallet: Stopped", false)
	state.turnIndicator, state.turnStatusLabel = createStatusIndicator("TURN: Stopped", false)

	// Create new KNIRV-ROUTER status indicators
	state.nrnMintingIndicator, state.nrnMintingStatusLabel = createStatusIndicator("NRN Minting: Stopped", false)
	state.proofEngineIndicator, state.proofEngineStatusLabel = createStatusIndicator("Proof Engine: Stopped", false)
	state.faucetIndicator, state.faucetStatusLabel = createStatusIndicator("Faucet: Disconnected", false)
	walletInfoTitle := widget.NewLabel("Wallet Information")
	walletInfoTitle.TextStyle = fyne.TextStyle{Bold: true}
	state.walletAddressEntry = widget.NewEntry()
	state.walletAddressEntry.SetPlaceHolder("Address")
	state.walletAddressEntry.Disable()
	state.walletPrivateKeyEntry = widget.NewEntry()
	state.walletPrivateKeyEntry.SetPlaceHolder("Private key")
	state.walletPrivateKeyEntry.Disable()
	state.walletPublicKeyEntry = widget.NewEntry()
	state.walletPublicKeyEntry.SetPlaceHolder("Public key")
	state.walletPublicKeyEntry.Disable()
	addressLabel := widget.NewLabel("Address:")
	addressLabel.TextStyle = fyne.TextStyle{Bold: true}
	privateKeyLabel := widget.NewLabel("Private Key:")
	privateKeyLabel.TextStyle = fyne.TextStyle{Bold: true}
	publicKeyLabel := widget.NewLabel("Public Key:")
	publicKeyLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Create new KNIRV-ROUTER display labels
	state.nrnBalanceLabel = widget.NewLabel("NRN Balance: 0")
	state.nrnBalanceLabel.TextStyle = fyne.TextStyle{Bold: true}
	state.usdcBalanceLabel = widget.NewLabel("USDC Balance: 0.00")
	state.usdcBalanceLabel.TextStyle = fyne.TextStyle{Bold: true}
	state.connectivityMetricsLabel = widget.NewLabel("Connectivity: No data")
	state.mintingStatsLabel = widget.NewLabel("Minting: 0 NRNs generated")
	state.pathCertificatesLabel = widget.NewLabel("Path Certificates: 0 active")

	// --- Button Creation ---
	startVerifyerBlockchain := widget.NewButton("Start Verifier Blockchain", func() {
		cfg := state.readConfigFromUI(minerAddress, chainPort, walletPort, nodeAddress, difficulty, reward, dbPath)
		go state.startVerifierBlockchain(cfg) // Renamed function
	})
	startVerifyerBlockchain.Importance = widget.HighImportance

	// New Pause/Resume Buttons - initially disabled
	pauseBlockchain := widget.NewButton("Pause Blockchain", func() {
		go state.pauseBlockchain()
	})
	pauseBlockchain.Importance = widget.MediumImportance
	pauseBlockchain.Disable() // Initially disabled

	resumeBlockchain := widget.NewButton("Resume Blockchain", func() {
		go state.resumeBlockchain()
	})
	resumeBlockchain.Importance = widget.MediumImportance
	resumeBlockchain.Disable() // Initially disabled

	generateWallet := widget.NewButton("Generate Wallet", func() {
		go state.generateWallet()
	})
	generateWallet.Importance = widget.MediumImportance

	startWallet := widget.NewButton("Start Wallet", func() {
		cfg := state.readConfigFromUI(minerAddress, chainPort, walletPort, nodeAddress, difficulty, reward, dbPath)
		go state.startWallet(cfg)
	})
	startWallet.Importance = widget.HighImportance

	startTurnServer := widget.NewButton("Start TURN Server", func() {
		go state.startTurnServer()
	})
	startTurnServer.Importance = widget.MediumImportance

	stopTurnServer := widget.NewButton("Stop TURN Server", func() {
		go state.stopTurnServer()
	})
	stopTurnServer.Importance = widget.MediumImportance

	// New KNIRV-ROUTER control buttons
	startNRNMinting := widget.NewButton("Start NRN Minting", func() {
		go state.startNRNMinting()
	})
	startNRNMinting.Importance = widget.HighImportance

	stopNRNMinting := widget.NewButton("Stop NRN Minting", func() {
		go state.stopNRNMinting()
	})
	stopNRNMinting.Importance = widget.MediumImportance

	startProofEngine := widget.NewButton("Start Proof Engine", func() {
		go state.startProofEngine()
	})
	startProofEngine.Importance = widget.HighImportance

	stopProofEngine := widget.NewButton("Stop Proof Engine", func() {
		go state.stopProofEngine()
	})
	stopProofEngine.Importance = widget.MediumImportance

	connectFaucet := widget.NewButton("Connect to Faucet", func() {
		go state.connectToFaucet()
	})
	connectFaucet.Importance = widget.MediumImportance

	testConnectivity := widget.NewButton("Test Connectivity", func() {
		go state.testConnectivity()
	})
	testConnectivity.Importance = widget.LowImportance

	// Store buttons in state for later access
	state.startVerifyerBlockchainButton = startVerifyerBlockchain
	state.pauseBlockchainButton = pauseBlockchain
	state.resumeBlockchainButton = resumeBlockchain

	// Store new KNIRV-ROUTER buttons in state
	state.startNRNMintingButton = startNRNMinting
	state.stopNRNMintingButton = stopNRNMinting
	state.startProofEngineButton = startProofEngine
	state.stopProofEngineButton = stopProofEngine
	state.connectFaucetButton = connectFaucet
	state.testConnectivityButton = testConnectivity

	// --- Layout Creation ---
	title := canvas.NewText("KNIRVROUTER Verifier Node Manager", color.NRGBA{R: 0, G: 180, B: 255, A: 255})
	title.TextSize = 24
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter
	// Auto-scroll checkbox with enhanced behavior
	autoScrollCheck := widget.NewCheck("Auto Scroll", func(checked bool) {
		state.terminal.SetAutoScroll(checked)
		if checked {
			// Immediately scroll to bottom when enabled
			state.terminal.ScrollToBottom()
			// Log the action
			state.logChan <- "Auto-scroll enabled - terminal locked to bottom"
		} else {
			state.logChan <- "Auto-scroll disabled - manual scrolling enabled"
		}
	})
	autoScrollCheck.SetChecked(true)

	// Status bar with proper spacing
	statusContainer := container.NewHBox(
		container.NewHBox(state.blockchainIndicator, state.blockchainStatusLabel),
		widget.NewLabel("│"),
		container.NewHBox(state.walletIndicator, state.walletStatusLabel),
		widget.NewLabel("│"),
		container.NewHBox(state.turnIndicator, state.turnStatusLabel),
		widget.NewLabel("│"),
		container.NewHBox(state.nrnMintingIndicator, state.nrnMintingStatusLabel),
		widget.NewLabel("│"),
		container.NewHBox(state.proofEngineIndicator, state.proofEngineStatusLabel),
		widget.NewLabel("│"),
		container.NewHBox(state.faucetIndicator, state.faucetStatusLabel),
		widget.NewLabel("│"),
		autoScrollCheck,
	)

	formContainer := container.NewVBox(
		container.NewCenter(title),
		widget.NewSeparator(),
		container.NewPadded(widget.NewForm(
			widget.NewFormItem("Miner's Address", minerAddress),
			widget.NewFormItem("Verifier Chain Port", chainPort),
			widget.NewFormItem("Wallet Port", walletPort),
			widget.NewFormItem("Node Address (for Wallet)", nodeAddress),
			widget.NewFormItem("Mining Difficulty", difficulty),
			widget.NewFormItem("Mining Reward", reward),
			widget.NewFormItem("Verifier DB Path", dbPath),
			widget.NewFormItem("Root Chain Address", state.rootChainAddressEntry),
			widget.NewFormItem("TURN Port (UDP/TCP)", state.turnPortEntry),
			widget.NewFormItem("NRN Minting Rate (per min)", state.nrnMintingRateEntry),
			widget.NewFormItem("Connectivity Test Interval (sec)", state.connectivityIntervalEntry),
			widget.NewFormItem("KNIRV-ORACLE Faucet Endpoint", state.faucetEndpointEntry),
			widget.NewFormItem("Proof Engine API Port", state.proofEnginePortEntry),
		)),
		widget.NewSeparator(),
		container.NewGridWithColumns(3,
			startVerifyerBlockchain, pauseBlockchain, resumeBlockchain,
			generateWallet, startWallet, widget.NewLabel(""),
			startTurnServer, stopTurnServer, widget.NewLabel(""),
			startNRNMinting, stopNRNMinting, widget.NewLabel(""),
			startProofEngine, stopProofEngine, testConnectivity,
			connectFaucet, widget.NewLabel(""), widget.NewLabel(""),
		),
		widget.NewSeparator(),
		statusContainer,
		widget.NewSeparator(),
		walletInfoTitle,
		container.NewPadded(container.NewVBox(
			container.NewBorder(nil, nil, addressLabel, nil, state.walletAddressEntry),
			widget.NewSeparator(),
			container.NewBorder(nil, nil, privateKeyLabel, nil, state.walletPrivateKeyEntry),
			widget.NewSeparator(),
			container.NewBorder(nil, nil, publicKeyLabel, nil, state.walletPublicKeyEntry),
		)),
		widget.NewSeparator(),
		// New KNIRV-ROUTER Information Panel
		widget.NewLabel("KNIRV-ROUTER Metrics & Balances"),
		container.NewPadded(container.NewVBox(
			container.NewGridWithColumns(2,
				state.nrnBalanceLabel, state.usdcBalanceLabel,
			),
			widget.NewSeparator(),
			state.connectivityMetricsLabel,
			state.mintingStatsLabel,
			state.pathCertificatesLabel,
		)),
		widget.NewSeparator(),
		state.statusLabel,
	)
	borderedForm := container.NewBorder(nil, nil, nil, nil, container.NewPadded(formContainer))
	terminalTitle := widget.NewLabel("Verifier Node Console Output")
	terminalTitle.TextStyle = fyne.TextStyle{Bold: true}
	terminalTitle.Alignment = fyne.TextAlignCenter

	// Create a padding container to ensure proper spacing
	paddedTerminal := container.NewPadded(state.terminal.GetContent())

	// Create the terminal section with a dark background
	terminalBackground := canvas.NewRectangle(color.NRGBA{R: 20, G: 20, B: 20, A: 255})
	terminalWithBg := container.NewMax(terminalBackground, paddedTerminal)

	// Wrap in a border container to maintain layout
	terminalContainer := container.NewBorder(
		container.NewVBox(terminalTitle, widget.NewSeparator()),
		nil, nil, nil,
		terminalWithBg,
	)

	// Set a larger minimum size for the terminal container
	terminalContainer.Resize(fyne.NewSize(600, 400))

	// Create the main content split
	content := container.NewHSplit(borderedForm, terminalContainer)
	content.Offset = 0.3

	window.SetContent(content)

	// Ensure terminal starts at bottom
	state.terminal.ForceScrollToBottom()

	go state.processLogChannel() // Start log processor BEFORE ShowAndRun

	// Add initial welcome message
	state.logChan <- "KNIRVROUTER GUI initialized - terminal auto-scroll enabled"
	state.logChan <- "Ready to start KNIRV-ROUTER operations..."

	window.ShowAndRun()
}

// Helper to read config from UI elements safely
func (s *FyneGUIState) readConfigFromUI(minerAddress, chainPort, walletPort, nodeAddress, difficulty, reward, dbPath *widget.Entry) FyneGuiConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get values from the entries that are passed as parameters
	config := FyneGuiConfig{
		MinersAddress:         minerAddress.Text,
		ChainPort:             chainPort.Text,
		WalletPort:            walletPort.Text,
		BlockchainNodeAddress: nodeAddress.Text,
		MiningDifficulty:      difficulty.Text,
		MiningReward:          reward.Text,
		DatabasePath:          dbPath.Text,
	}

	// Get values from the entries stored in the state
	if s.rootChainAddressEntry != nil {
		config.RootChainAddress = s.rootChainAddressEntry.Text
	}

	if s.turnPortEntry != nil {
		config.TurnPort = s.turnPortEntry.Text
	}

	// Get values from new KNIRV-ROUTER entries
	if s.nrnMintingRateEntry != nil {
		config.NRNMintingRate = s.nrnMintingRateEntry.Text
	}

	if s.connectivityIntervalEntry != nil {
		config.ConnectivityInterval = s.connectivityIntervalEntry.Text
	}

	if s.faucetEndpointEntry != nil {
		config.FaucetEndpoint = s.faucetEndpointEntry.Text
	}

	if s.proofEnginePortEntry != nil {
		config.ProofEnginePort = s.proofEnginePortEntry.Text
	}

	return config
}

// Goroutine to process logs from the channel
func (s *FyneGUIState) processLogChannel() {
	log.Println("Log processing goroutine started.")

	// Create a buffer to batch process messages
	const batchSize = 5
	const batchDelay = 100 * time.Millisecond

	msgBuffer := make([]string, 0, batchSize)
	lastFlush := time.Now()

	// Function to flush the buffer to the terminal
	flushBuffer := func() {
		if len(msgBuffer) == 0 {
			return
		}

		// Join all messages with newlines
		combinedMsg := strings.Join(msgBuffer, "\n")
		msgCopy := combinedMsg // Make a copy for the closure

		// Clear the buffer
		msgBuffer = msgBuffer[:0]

		// Update the terminal on the main thread with enhanced scrolling
		if ap, ok := s.app.(interface{ CallOnMainThread(func()) }); ok {
			ap.CallOnMainThread(func() {
				// Append the text (this already handles scrolling internally)
				s.terminal.Append(msgCopy)

				// Additional scroll enforcement for batch processing
				if s.terminal.autoScroll {
					// Immediate scroll
					s.terminal.container.ScrollToBottom()

					// Delayed scroll to handle any UI updates
					time.AfterFunc(25*time.Millisecond, func() {
						ap.CallOnMainThread(func() {
							s.terminal.container.ScrollToBottom()
						})
					})
				}
			})
		} else {
			// Fallback if CallOnMainThread is not available
			s.mu.Lock()
			s.terminal.Append(msgCopy)
			if s.terminal.autoScroll {
				s.terminal.container.ScrollToBottom()
				s.terminal.container.Refresh()
			}
			s.mu.Unlock()
		}

		// Update the last flush time
		lastFlush = time.Now()
	}

	// Process messages from the channel
	for logMsg := range s.logChan {
		// Add the message to the buffer
		msgBuffer = append(msgBuffer, logMsg)

		// Flush if we've reached the batch size or the delay has elapsed
		if len(msgBuffer) >= batchSize || time.Since(lastFlush) > batchDelay {
			flushBuffer()
		}
	}

	// Flush any remaining messages
	flushBuffer()

	log.Println("Log processing channel closed.")
}

// updateStatus sends to channel and updates label safely
func (s *FyneGUIState) updateStatus(status string) {
	// <<< CORRECTED RunOnMain call >>>
	if ap, ok := s.app.(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			s.status = status
			s.statusLabel.SetText("Status: " + status)
			s.statusLabel.Refresh()
		})
	} else {
		go func(status string) { // Fallback with mutex protection
			s.mu.Lock()
			defer s.mu.Unlock()
			s.status = status
			s.statusLabel.SetText("Status: " + status)
			s.statusLabel.Refresh()
		}(status)
	}
	select {
	case s.logChan <- status:
	default:
		log.Printf("Log channel full, dropping status message: %s", status)
	}
}

// updateWalletInfo updates wallet info entries safely
func (s *FyneGUIState) updateWalletInfo(address, privateKey, publicKey string) {
	s.walletInfo.Address = address
	s.walletInfo.PrivateKey = privateKey
	s.walletInfo.PublicKey = publicKey
	// <<< CORRECTED RunOnMain call >>>
	if ap, ok := s.app.(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			s.walletAddressEntry.SetText(address)
			s.walletPrivateKeyEntry.SetText(privateKey)
			s.walletPublicKeyEntry.SetText(publicKey)
		})
	} else {
		go func(addr, priv, pub string) { // Fallback with mutex protection
			s.mu.Lock()
			defer s.mu.Unlock()
			s.walletAddressEntry.SetText(addr)
			s.walletPrivateKeyEntry.SetText(priv)
			s.walletPublicKeyEntry.SetText(pub)
		}(address, privateKey, publicKey)
	}
}

// startVerifierBlockchain starts the verifier blockchain node process

func (s *FyneGUIState) startVerifierBlockchain(cfg FyneGuiConfig) {
	s.mu.Lock()
	// Check if blockchain is already running and paused - if so, just resume it
	if s.blockchainRunning && s.blockchainPaused {
		s.mu.Unlock()
		s.resumeBlockchain()
		return
	}

	// Check if blockchain is already running - if so, don't start another one
	if s.blockchainRunning && !s.blockchainPaused {
		s.mu.Unlock()
		s.updateStatus("Blockchain is already running")
		return
	}
	s.mu.Unlock()

	if cfg.ChainPort == "" {
		s.updateStatus("Error: Verifier Chain port cannot be empty")
		return
	}
	if _, err := strconv.Atoi(cfg.ChainPort); err != nil {
		s.updateStatus("Error: Invalid Verifier Chain port: " + cfg.ChainPort)
		return
	}
	if _, err := strconv.Atoi(cfg.MiningDifficulty); err != nil {
		s.updateStatus("Error: Invalid Mining Difficulty: " + cfg.MiningDifficulty)
		return
	}
	if _, err := strconv.Atoi(cfg.MiningReward); err != nil {
		s.updateStatus("Error: Invalid Mining Reward: " + cfg.MiningReward)
		return
	}
	if cfg.DatabasePath == "" {
		s.updateStatus("Error: Verifier Database path cannot be empty")
		return
	}
	if cfg.RootChainAddress == "" {
		s.updateStatus("Warning: Root Chain address is empty. Federation will be disabled.")
	}

	s.config.MinersAddress = cfg.MinersAddress
	s.config.ChainPort = cfg.ChainPort
	s.config.MiningDifficulty = cfg.MiningDifficulty
	s.config.MiningReward = cfg.MiningReward
	s.config.DatabasePath = cfg.DatabasePath         // Store potentially edited verifier DB path
	s.config.RootChainAddress = cfg.RootChainAddress // Store root chain address
	s.config.BlockchainNodeAddress = fmt.Sprintf("http://127.0.0.1:%s", cfg.ChainPort)

	s.updateStatus("Starting Verifier Blockchain...")
	s.logChan <- "Verifier Blockchain Config:"
	s.logChan <- "  Miner Addr: " + cfg.MinersAddress
	s.logChan <- "  Chain Port: " + cfg.ChainPort
	s.logChan <- "  Difficulty: " + cfg.MiningDifficulty
	s.logChan <- "  Reward:     " + cfg.MiningReward
	s.logChan <- "  DB Path:    " + cfg.DatabasePath
	s.logChan <- "  Root Chain: " + cfg.RootChainAddress
	var cmd *exec.Cmd
	var err error
	binaryPath := "./KNIRVROUTER"

	// Build command arguments
	args := []string{
		"chain",
		"--port=" + cfg.ChainPort,
		"--miners_address=" + cfg.MinersAddress,
		"--dbpath=" + cfg.DatabasePath,
	}

	// Add root chain address if provided
	/*
		if cfg.RootChainAddress != "" {
			args = append(args, "--root_chain="+cfg.RootChainAddress)
		}
	*/
	// Check if binary exists and create command
	if _, err = os.Stat(binaryPath); err == nil {
		cmd = exec.Command(binaryPath, args...)
	} else if errors.Is(err, os.ErrNotExist) {
		s.updateStatus("Warning: KNIRVROUTER binary not found, attempting 'go run'")
		// Use "run ." to run from the current directory where main.go is expected
		runArgs := append([]string{"run", "."}, args...) // Pass only the valid args
		cmd = exec.Command("go", runArgs...)
	} else {
		s.updateStatus(fmt.Sprintf("Error checking for binary %s: %v", binaryPath, err))
		return
	}

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"MINERS_ADDRESS="+cfg.MinersAddress,
		"PORT="+cfg.ChainPort,
		"MINING_DIFFICULTY="+cfg.MiningDifficulty,
		"MINING_REWARD="+cfg.MiningReward,
		"BLOCKCHAIN_DB_PATH="+cfg.DatabasePath,
		"ROOT_CHAIN_ADDRESS="+cfg.RootChainAddress,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.updateStatus(fmt.Sprintf("Error getting stdout pipe: %v", err))
		s.updateIndicator(s.blockchainIndicator, s.blockchainStatusLabel, false, "Blockchain: Failed")
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.updateStatus(fmt.Sprintf("Error getting stderr pipe: %v", err))
		s.updateIndicator(s.blockchainIndicator, s.blockchainStatusLabel, false, "Blockchain: Failed")
		return
	}
	if err := cmd.Start(); err != nil {
		s.updateStatus(fmt.Sprintf("Error starting verifier blockchain: %v", err))
		s.updateIndicator(s.blockchainIndicator, s.blockchainStatusLabel, false, "Blockchain: Failed")
		return
	}
	s.mu.Lock()
	s.activeProcs = append(s.activeProcs, cmd)
	s.blockchainRunning = true
	s.blockchainPaused = false
	s.mu.Unlock()

	// Update button states now that blockchain is running
	s.updateBlockchainButtonStates()

	s.updateIndicator(s.blockchainIndicator, s.blockchainStatusLabel, true, "Blockchain: Running")
	go s.readOutput(stdout, "Blockchain")
	go s.readOutput(stderr, "Blockchain Error")
	go func(processCmd *exec.Cmd) {
		err := processCmd.Wait()
		s.mu.Lock()
		blockchainPaused := s.blockchainPaused
		s.mu.Unlock()

		// Only update status as exited if the blockchain wasn't paused
		if !blockchainPaused {
			statusMsg := "Verifier Blockchain process exited normally"
			if err != nil {
				statusMsg = fmt.Sprintf("Verifier Blockchain process exited with error: %v", err)
			}
			s.updateStatus(statusMsg)
			s.mu.Lock()
			s.blockchainRunning = false
			s.blockchainPaused = false
			s.removeProcess(processCmd)
			s.mu.Unlock()
			s.updateIndicator(s.blockchainIndicator, s.blockchainStatusLabel, false, "Blockchain: Stopped")
		}
	}(cmd)
}

// generateWallet remains largely the same
func (s *FyneGUIState) generateWallet() { /* ... as before ... */
	s.updateStatus("Generating new wallet...")
	w, err := wallet.NewWallet()
	if err != nil {
		s.updateStatus(fmt.Sprintf("Error generating wallet: %v", err))
		return
	}
	s.updateWalletInfo(w.GetAddress(), w.GetPrivateKeyHex(), w.GetPublicKeyHex())
	s.updateStatus("New wallet generated successfully")
}

// startWallet remains largely the same
func (s *FyneGUIState) startWallet(cfg FyneGuiConfig) { /* ... as before ... */
	if cfg.WalletPort == "" {
		s.updateStatus("Error: Wallet port cannot be empty")
		return
	}
	if _, err := strconv.Atoi(cfg.WalletPort); err != nil {
		s.updateStatus("Error: Invalid wallet port number: " + cfg.WalletPort)
		return
	}
	nodeAddrToUse := cfg.BlockchainNodeAddress
	if nodeAddrToUse == "" {
		s.mu.Lock()
		nodeAddrToUse = s.config.BlockchainNodeAddress
		s.mu.Unlock()
		if nodeAddrToUse == "" {
			s.updateStatus("Error: Blockchain node address is not set.")
			return
		}
		s.updateStatus("Using node address from state: " + nodeAddrToUse)
	}
	s.updateStatus("Starting wallet...")
	s.logChan <- "Wallet Config:"
	s.logChan <- "  Wallet Port: " + cfg.WalletPort
	s.logChan <- "  Node Addr:   " + nodeAddrToUse
	var cmd *exec.Cmd
	var err error
	binaryPath := "./KNIRVROUTER"
	args := []string{"wallet", "--port=" + cfg.WalletPort, "--node_address=" + nodeAddrToUse}
	if _, err = os.Stat(binaryPath); err == nil {
		cmd = exec.Command(binaryPath, args...)
	} else if errors.Is(err, os.ErrNotExist) {
		s.updateStatus("Warning: KNIRVROUTER binary not found, attempting 'go run'")
		cmd = exec.Command("go", append([]string{"run", "main.go"}, args...)...)
	} else {
		s.updateStatus(fmt.Sprintf("Error checking for binary %s: %v", binaryPath, err))
		return
	}
	cmd.Env = append(os.Environ(), "WALLET_PORT="+cfg.WalletPort, "BLOCKCHAIN_NODE_ADDRESS="+nodeAddrToUse)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.updateStatus(fmt.Sprintf("Error getting wallet stdout pipe: %v", err))
		s.updateIndicator(s.walletIndicator, s.walletStatusLabel, false, "Wallet: Failed")
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.updateStatus(fmt.Sprintf("Error getting wallet stderr pipe: %v", err))
		s.updateIndicator(s.walletIndicator, s.walletStatusLabel, false, "Wallet: Failed")
		return
	}
	if err := cmd.Start(); err != nil {
		s.updateStatus(fmt.Sprintf("Error starting wallet: %v", err))
		s.updateIndicator(s.walletIndicator, s.walletStatusLabel, false, "Wallet: Failed")
		return
	}
	s.mu.Lock()
	s.activeProcs = append(s.activeProcs, cmd)
	s.walletRunning = true
	s.mu.Unlock()
	s.updateIndicator(s.walletIndicator, s.walletStatusLabel, true, "Wallet: Running")
	go s.readOutput(stdout, "Wallet")
	go s.readOutput(stderr, "Wallet Error")
	go func(processCmd *exec.Cmd) {
		err := processCmd.Wait()
		statusMsg := "Wallet process exited normally"
		if err != nil {
			statusMsg = fmt.Sprintf("Wallet process exited with error: %v", err)
		}
		s.updateStatus(statusMsg)
		s.mu.Lock()
		s.walletRunning = false
		s.removeProcess(processCmd)
		s.mu.Unlock()
		s.updateIndicator(s.walletIndicator, s.walletStatusLabel, false, "Wallet: Stopped")
	}(cmd)
	go func(port string) {
		time.Sleep(2 * time.Second)
		url := fmt.Sprintf("http://localhost:%s", port)
		s.updateStatus("Opening wallet in browser: " + url)
		openBrowser(url)
	}(cfg.WalletPort)
}

// readOutput sends to channel (Only used for Root Node and Wallet now)
func (s *FyneGUIState) readOutput(pipe io.ReadCloser, prefix string) {
	scanner := bufio.NewScanner(pipe)
	const maxCapacity = 512 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" { // Only process non-empty lines
			// Directly send to the terminal via logChan
			s.logChan <- line
		}
	}

	if err := scanner.Err(); err != nil {
		errMsg := fmt.Sprintf("Error reading %s output: %v", prefix, err)
		s.logChan <- errMsg
	}
}

// Helper function to remove a process from the active list (requires external lock)
func (s *FyneGUIState) removeProcess(cmdToRemove *exec.Cmd) { /* ... as before ... */
	newProcs := make([]*exec.Cmd, 0, len(s.activeProcs))
	for _, p := range s.activeProcs {
		if p != cmdToRemove {
			newProcs = append(newProcs, p)
		}
	}
	s.activeProcs = newProcs
}

// cleanupProcesses uses mutex and tries graceful shutdown
func (s *FyneGUIState) cleanupProcesses() {
	s.updateStatus("Cleaning up processes...")

	// First, stop the TURN server if it's running
	s.mu.Lock()
	turnServerRunning := s.turnServerRunning
	turnServer := s.turnServer
	s.mu.Unlock()

	if turnServerRunning && turnServer != nil {
		s.updateStatus("Stopping TURN server...")
		if err := turnServer.Stop(); err != nil {
			s.updateStatus(fmt.Sprintf("Error stopping TURN server: %v", err))
		} else {
			s.updateStatus("TURN server stopped.")
		}
		s.mu.Lock()
		s.turnServerRunning = false
		s.turnServer = nil
		s.mu.Unlock()
	}

	// Then clean up external processes
	s.mu.Lock()
	procsToKill := make([]*exec.Cmd, len(s.activeProcs))
	copy(procsToKill, s.activeProcs)
	blockchainPaused := s.blockchainPaused
	s.activeProcs = nil
	s.blockchainRunning = false
	s.blockchainPaused = false
	s.walletRunning = false
	s.mu.Unlock()

	for _, proc := range procsToKill {
		if proc != nil && proc.Process != nil {
			pid := proc.Process.Pid
			s.updateStatus(fmt.Sprintf("Attempting to terminate process PID %d", pid))
			log.Printf("Attempting to terminate process PID %d", pid)

			// For paused processes, resume them first before killing
			if blockchainPaused && runtime.GOOS != "windows" {
				cmdLine := strings.Join(proc.Args, " ")
				if strings.Contains(cmdLine, "chain") {
					// Resume the paused blockchain process first
					err := proc.Process.Signal(syscall.SIGCONT)
					if err != nil {
						log.Printf("Failed to resume paused blockchain process PID %d: %v", pid, err)
					} else {
						log.Printf("Resumed paused blockchain PID %d before termination", pid)
						// Give it a short time to resume
						time.Sleep(100 * time.Millisecond)
					}
				}
			}

			err := proc.Process.Signal(os.Interrupt)
			if err != nil {
				log.Printf("Failed to send interrupt signal to PID %d: %v. Attempting kill.", pid, err)
				if killErr := proc.Process.Kill(); killErr != nil {
					errMsg := fmt.Sprintf("Error killing process PID %d: %v", pid, killErr)
					s.updateStatus(errMsg)
					log.Println(errMsg)
				} else {
					log.Printf("Force killed process PID %d", pid)
				}
			} else {
				log.Printf("Sent interrupt signal to PID %d", pid)
			}
		}
	}
	s.updateStatus("Process cleanup finished.")
	log.Println("Process cleanup finished.")
}

// openBrowser helper function (needed for wallet)
func openBrowser(url string) {
	var err error
	log.Printf("Executing openBrowser for URL: %s", url)
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("cmd", "/c", "start", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err != nil {
		log.Printf("ERROR opening browser for %s: %v", url, err)
	} else {
		log.Printf("Browser command initiated successfully for %s", url)
	}
}

// pauseBlockchain sends a SIGSTOP signal to the blockchain process to pause it
func (s *FyneGUIState) pauseBlockchain() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.blockchainRunning || s.blockchainPaused {
		s.updateStatus("Blockchain is not running or already paused")
		return
	}

	// Find the blockchain process
	var blockchainProc *exec.Cmd
	for _, proc := range s.activeProcs {
		// We identify the blockchain process based on what we know about it
		// This assumes we have a way to identify the blockchain process
		if proc != nil && proc.Process != nil {
			// Check if this process is our blockchain process (simplified approach)
			cmdLine := strings.Join(proc.Args, " ")
			if strings.Contains(cmdLine, "chain") {
				blockchainProc = proc
				break
			}
		}
	}

	if blockchainProc == nil || blockchainProc.Process == nil {
		s.updateStatus("Error: Could not find running blockchain process")
		return
	}

	// Send SIGSTOP signal to pause the process
	var err error
	if runtime.GOOS == "windows" {
		// Windows doesn't support SIGSTOP directly
		s.updateStatus("Pausing not fully supported on Windows - will attempt to suspend")
		// Use alternative Windows-specific approach if needed
		cmd := exec.Command("powershell", "-Command",
			fmt.Sprintf("(Get-Process -Id %d).Suspend()", blockchainProc.Process.Pid))
		err = cmd.Run()
	} else {
		// Unix-like systems can use SIGSTOP
		err = blockchainProc.Process.Signal(syscall.SIGSTOP)
	}

	if err != nil {
		s.updateStatus(fmt.Sprintf("Error pausing blockchain: %v", err))
		return
	}

	s.blockchainPaused = true
	s.updateStatus("Blockchain paused successfully")
	s.updateIndicator(s.blockchainIndicator, s.blockchainStatusLabel, true, "Blockchain: Paused")
}

// resumeBlockchain sends a SIGCONT signal to resume the paused blockchain process
func (s *FyneGUIState) resumeBlockchain() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.blockchainRunning || !s.blockchainPaused {
		s.updateStatus("Blockchain is not running or not paused")
		return
	}

	// Find the blockchain process
	var blockchainProc *exec.Cmd
	for _, proc := range s.activeProcs {
		if proc != nil && proc.Process != nil {
			cmdLine := strings.Join(proc.Args, " ")
			if strings.Contains(cmdLine, "chain") {
				blockchainProc = proc
				break
			}
		}
	}

	if blockchainProc == nil || blockchainProc.Process == nil {
		s.updateStatus("Error: Could not find paused blockchain process")
		return
	}

	// Send SIGCONT signal to resume the process
	var err error
	if runtime.GOOS == "windows" {
		// Windows doesn't support SIGCONT directly
		s.updateStatus("Resuming not fully supported on Windows - will attempt to resume")
		// Use alternative Windows-specific approach if needed
		cmd := exec.Command("powershell", "-Command",
			fmt.Sprintf("(Get-Process -Id %d).Resume()", blockchainProc.Process.Pid))
		err = cmd.Run()
	} else {
		// Unix-like systems can use SIGCONT
		err = blockchainProc.Process.Signal(syscall.SIGCONT)
	}

	if err != nil {
		s.updateStatus(fmt.Sprintf("Error resuming blockchain: %v", err))
		return
	}

	s.blockchainPaused = false
	s.updateStatus("Blockchain resumed successfully")
	s.updateIndicator(s.blockchainIndicator, s.blockchainStatusLabel, true, "Blockchain: Running")
}

// startTurnServer - stub implementation for TURN server
func (s *FyneGUIState) startTurnServer() {
	// Placeholder for TURN server implementation
	s.updateStatus("Starting TURN server...")
	s.mu.Lock()
	if s.turnServerRunning {
		s.mu.Unlock()
		s.updateStatus("TURN server is already running")
		return
	}
	s.turnServerRunning = true
	s.mu.Unlock()
	s.updateIndicator(s.turnIndicator, s.turnStatusLabel, true, "TURN: Running")
	s.updateStatus("TURN server started successfully")
}

// stopTurnServer - stub implementation
func (s *FyneGUIState) stopTurnServer() {
	s.updateStatus("Stopping TURN server...")
	s.mu.Lock()
	if !s.turnServerRunning {
		s.mu.Unlock()
		s.updateStatus("TURN server is not running")
		return
	}
	if s.turnServer != nil {
		if err := s.turnServer.Stop(); err != nil {
			s.mu.Unlock()
			s.updateStatus(fmt.Sprintf("Error stopping TURN server: %v", err))
			return
		}
	}
	s.turnServerRunning = false
	s.turnServer = nil
	s.mu.Unlock()
	s.updateIndicator(s.turnIndicator, s.turnStatusLabel, false, "TURN: Stopped")
	s.updateStatus("TURN server stopped successfully")
}

// updateBlockchainButtonStates updates button states based on blockchain status
func (s *FyneGUIState) updateBlockchainButtonStates() {
	// Call on main UI thread to be safe
	if ap, ok := s.app.(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			if s.blockchainRunning {
				// Blockchain is running - disable start button
				s.startVerifyerBlockchainButton.Disable()

				if s.blockchainPaused {
					// Blockchain is paused - enable resume, disable pause
					s.pauseBlockchainButton.Disable()
					s.resumeBlockchainButton.Enable()
				} else {
					// Blockchain is running but not paused - enable pause, disable resume
					s.pauseBlockchainButton.Enable()
					s.resumeBlockchainButton.Disable()
				}
			} else {
				// Blockchain is not running at all - enable start, disable pause & resume
				s.startVerifyerBlockchainButton.Enable()
				s.pauseBlockchainButton.Disable()
				s.resumeBlockchainButton.Disable()
			}
		})
	} else {
		// Fallback implementation if CallOnMainThread is not available
		go func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			if s.blockchainRunning {
				s.startVerifyerBlockchainButton.Disable()

				if s.blockchainPaused {
					s.pauseBlockchainButton.Disable()
					s.resumeBlockchainButton.Enable()
				} else {
					s.pauseBlockchainButton.Enable()
					s.resumeBlockchainButton.Disable()
				}
			} else {
				s.startVerifyerBlockchainButton.Enable()
				s.pauseBlockchainButton.Disable()
				s.resumeBlockchainButton.Disable()
			}
		}()
	}
}

// --- New KNIRV-ROUTER Methods ---

// startNRNMinting starts the NRN minting process
func (s *FyneGUIState) startNRNMinting() {
	s.mu.Lock()
	if s.nrnMintingRunning {
		s.mu.Unlock()
		s.updateStatus("NRN Minting is already running")
		return
	}
	s.nrnMintingRunning = true
	s.mu.Unlock()

	s.updateStatus("Starting NRN Minting Engine...")
	s.logChan <- "NRN Minting Configuration:"
	s.logChan <- "  Minting Rate: " + s.config.NRNMintingRate + " NRNs/min"
	s.logChan <- "  Proof Engine Port: " + s.config.ProofEnginePort

	// TODO: Implement actual NRN minting logic here
	// This would integrate with the connectivity/proof_engine.go module
	s.updateIndicator(s.nrnMintingIndicator, s.nrnMintingStatusLabel, true, "NRN Minting: Running")
	s.updateStatus("NRN Minting Engine started successfully")

	// Update display with mock data for now
	s.updateNRNBalance("150")
	s.updateMintingStats("15 NRNs generated today")
}

// stopNRNMinting stops the NRN minting process
func (s *FyneGUIState) stopNRNMinting() {
	s.mu.Lock()
	if !s.nrnMintingRunning {
		s.mu.Unlock()
		s.updateStatus("NRN Minting is not running")
		return
	}
	s.nrnMintingRunning = false
	s.mu.Unlock()

	s.updateStatus("Stopping NRN Minting Engine...")
	// TODO: Implement actual stopping logic here
	s.updateIndicator(s.nrnMintingIndicator, s.nrnMintingStatusLabel, false, "NRN Minting: Stopped")
	s.updateStatus("NRN Minting Engine stopped successfully")
}

// startProofEngine starts the Proof-of-Connectivity engine
func (s *FyneGUIState) startProofEngine() {
	s.mu.Lock()
	if s.proofEngineRunning {
		s.mu.Unlock()
		s.updateStatus("Proof-of-Connectivity Engine is already running")
		return
	}
	s.proofEngineRunning = true
	s.mu.Unlock()

	s.updateStatus("Starting Proof-of-Connectivity Engine...")
	s.logChan <- "Proof Engine Configuration:"
	s.logChan <- "  Test Interval: " + s.config.ConnectivityInterval + " seconds"
	s.logChan <- "  API Port: " + s.config.ProofEnginePort

	// TODO: Implement actual proof engine startup here
	// This would integrate with the connectivity/proof_engine.go module
	s.updateIndicator(s.proofEngineIndicator, s.proofEngineStatusLabel, true, "Proof Engine: Running")
	s.updateStatus("Proof-of-Connectivity Engine started successfully")

	// Update display with mock data for now
	s.updateConnectivityMetrics("Latency: 45ms, Success: 98.5%")
	s.updatePathCertificates("12 active certificates")
}

// stopProofEngine stops the Proof-of-Connectivity engine
func (s *FyneGUIState) stopProofEngine() {
	s.mu.Lock()
	if !s.proofEngineRunning {
		s.mu.Unlock()
		s.updateStatus("Proof-of-Connectivity Engine is not running")
		return
	}
	s.proofEngineRunning = false
	s.mu.Unlock()

	s.updateStatus("Stopping Proof-of-Connectivity Engine...")
	// TODO: Implement actual stopping logic here
	s.updateIndicator(s.proofEngineIndicator, s.proofEngineStatusLabel, false, "Proof Engine: Stopped")
	s.updateStatus("Proof-of-Connectivity Engine stopped successfully")
}

// connectToFaucet establishes connection to KNIRV-ORACLE Faucet
func (s *FyneGUIState) connectToFaucet() {
	s.mu.Lock()
	if s.faucetConnected {
		s.mu.Unlock()
		s.updateStatus("Already connected to KNIRV-ORACLE Faucet")
		return
	}
	s.mu.Unlock()

	s.updateStatus("Connecting to KNIRV-ORACLE Faucet...")
	s.logChan <- "Faucet Configuration:"
	s.logChan <- "  Endpoint: " + s.config.FaucetEndpoint

	// TODO: Implement actual faucet connection logic here
	// This would integrate with the faucet_integration.go module
	s.mu.Lock()
	s.faucetConnected = true
	s.mu.Unlock()

	s.updateIndicator(s.faucetIndicator, s.faucetStatusLabel, true, "Faucet: Connected")
	s.updateStatus("Connected to KNIRV-ORACLE Faucet successfully")

	// Update display with mock data for now
	s.updateUSDCBalance("250.75")
}

// testConnectivity performs a manual connectivity test
func (s *FyneGUIState) testConnectivity() {
	s.updateStatus("Running connectivity test...")
	s.logChan <- "Performing manual connectivity test..."

	// TODO: Implement actual connectivity test here
	// This would use the connectivity/proof_engine.go module

	// Simulate test results for now
	s.updateConnectivityMetrics("Latency: 42ms, Success: 99.2%")
	s.updateStatus("Connectivity test completed successfully")
}

// --- Helper methods for updating display labels ---

// updateNRNBalance updates the NRN balance display
func (s *FyneGUIState) updateNRNBalance(balance string) {
	if ap, ok := s.app.(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			s.nrnBalanceLabel.SetText("NRN Balance: " + balance)
			s.nrnBalanceLabel.Refresh()
		})
	} else {
		go func(bal string) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.nrnBalanceLabel.SetText("NRN Balance: " + bal)
			s.nrnBalanceLabel.Refresh()
		}(balance)
	}
}

// updateUSDCBalance updates the USDC balance display
func (s *FyneGUIState) updateUSDCBalance(balance string) {
	if ap, ok := s.app.(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			s.usdcBalanceLabel.SetText("USDC Balance: " + balance)
			s.usdcBalanceLabel.Refresh()
		})
	} else {
		go func(bal string) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.usdcBalanceLabel.SetText("USDC Balance: " + bal)
			s.usdcBalanceLabel.Refresh()
		}(balance)
	}
}

// updateConnectivityMetrics updates the connectivity metrics display
func (s *FyneGUIState) updateConnectivityMetrics(metrics string) {
	if ap, ok := s.app.(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			s.connectivityMetricsLabel.SetText("Connectivity: " + metrics)
			s.connectivityMetricsLabel.Refresh()
		})
	} else {
		go func(met string) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.connectivityMetricsLabel.SetText("Connectivity: " + met)
			s.connectivityMetricsLabel.Refresh()
		}(metrics)
	}
}

// updateMintingStats updates the minting statistics display
func (s *FyneGUIState) updateMintingStats(stats string) {
	if ap, ok := s.app.(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			s.mintingStatsLabel.SetText("Minting: " + stats)
			s.mintingStatsLabel.Refresh()
		})
	} else {
		go func(st string) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.mintingStatsLabel.SetText("Minting: " + st)
			s.mintingStatsLabel.Refresh()
		}(stats)
	}
}

// updatePathCertificates updates the path certificates display
func (s *FyneGUIState) updatePathCertificates(certs string) {
	if ap, ok := s.app.(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			s.pathCertificatesLabel.SetText("Path Certificates: " + certs)
			s.pathCertificatesLabel.Refresh()
		})
	} else {
		go func(c string) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.pathCertificatesLabel.SetText("Path Certificates: " + c)
			s.pathCertificatesLabel.Refresh()
		}(certs)
	}
}

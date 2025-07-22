//go:build fyne_gui
// +build fyne_gui

package gui

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	//"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"KNIRVCHAIN/config"
	"KNIRVCHAIN/types"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// guiState represents the centralized data displayed in the GUI
type guiState struct {
	// Bound data for automatic UI updates
	status       binding.String
	chainHeight  binding.Int
	peerCount    binding.Int
	latestBlock  binding.String
	miningStatus binding.String
	IsRoot       binding.Bool   // Whether this node is in root mode
	roleDisplay  binding.String // Display of the node's role

	// Lists that need manual synchronization
	blockListData      binding.UntypedList
	txnListData        binding.UntypedList
	peerListData       binding.StringList
	reflectionsList    binding.StringList
	contextRecordsData binding.UntypedList // For context records
	capabilitiesData   binding.UntypedList // For capabilities

	// References to original data sources
	blockchain       *BlockchainStruct
	db               *LevelDB
	discoveryMgr     *DiscoveryManager
	p2pConsensusMgr  *P2PConsensusManager
	paymentProcessor *PaymentProcessor // Reference to payment processor if enabled
	config           *config.Config    // Reference to the node configuration
	walletServer     *WalletServer     // Reference to the wallet server
	nodeRole         config.Role       // The role of this node

	// --- Wallet Information ---
	walletAddress string  // The address of this node's wallet
	wallet        *Wallet // The loaded wallet object (contains private key)

	// --- Terminal Output ---
	terminalOutput binding.String   // For chain activity terminal output
	logBuffer      *strings.Builder // Buffer for log capture

	// Mutex for thread-safe operations on non-binding data
	mu sync.RWMutex
}

// newGuiState creates and initializes a new GUI state
func newGuiState(blockchain *BlockchainStruct, db *LevelDB, discoveryMgr *DiscoveryManager, p2pConsensusMgr *P2PConsensusManager, cfg *config.Config, paymentProcessor *PaymentProcessor, role config.Role) *guiState {
	state := &guiState{
		status:             binding.NewString(),
		chainHeight:        binding.NewInt(),
		peerCount:          binding.NewInt(),
		latestBlock:        binding.NewString(),
		miningStatus:       binding.NewString(),
		IsRoot:             binding.NewBool(),
		roleDisplay:        binding.NewString(),
		blockListData:      binding.NewUntypedList(),
		txnListData:        binding.NewUntypedList(),
		peerListData:       binding.NewStringList(),
		reflectionsList:    binding.NewStringList(),
		contextRecordsData: binding.NewUntypedList(),
		capabilitiesData:   binding.NewUntypedList(),
		terminalOutput:     binding.NewString(),
		logBuffer:          new(strings.Builder),

		blockchain:       blockchain,
		db:               db,
		discoveryMgr:     discoveryMgr,
		p2pConsensusMgr:  p2pConsensusMgr,
		config:           cfg,
		paymentProcessor: paymentProcessor,
		walletServer:     NewWalletServer(cfg.WalletPort, cfg.MasterAddress), // Create a new wallet server for wallet operations
		nodeRole:         role,

		// Initialize Wallet Info
		walletAddress: "N/A", // Default if not found
		wallet:        nil,   // Default if not loaded
	}

	// Set the role display
	state.roleDisplay.Set(role.String())

	// Set initial values
	state.status.Set("Initializing...")
	state.chainHeight.Set(len(blockchain.Blocks))
	state.miningStatus.Set("Not Mining")
	state.peerCount.Set(0)
	state.IsRoot.Set(cfg != nil && cfg.IsRoot)
	state.terminalOutput.Set("Terminal ready. Chain activity will appear here...\n")

	if len(blockchain.Blocks) > 0 {
		latestBlock := blockchain.Blocks[len(blockchain.Blocks)-1]
		state.latestBlock.Set(hex.EncodeToString(latestBlock.BlockHash))
	}

	// Load Wallet Address and Private Key
	if blockchain.WalletAddress != "" {
		state.walletAddress = blockchain.WalletAddress
		log.Printf("GUI: Using wallet address from blockchain struct: %s", state.walletAddress)

		// Determine wallet path (needs refinement based on node mode)
		walletPath := ""
		peerWalletPath, _ := config.GetPeerWalletPath(config.RolePeer)
		rootWalletPath, _ := config.GetWalletPath(config.Root)

		if _, err := os.Stat(peerWalletPath); err == nil {
			walletPath = peerWalletPath
			log.Printf("GUI: Found peer wallet file at %s", walletPath)
		} else if _, err := os.Stat(rootWalletPath); err == nil {
			walletPath = rootWalletPath
			log.Printf("GUI: Found root wallet file at %s", walletPath)
		} else {
			log.Printf("GUI: Could not find wallet file at default paths (%s, %s)", peerWalletPath, rootWalletPath)
		}

		if walletPath != "" {
			// Use the global walletManager which handles decryption correctly
			// and uses the consistent encryption key.
			// walletManager is initialized in install.go's init() or main.go
			if walletManager == nil {
				log.Printf("GUI Error: global walletManager is not initialized.")
				state.status.Set("Error: Wallet Manager Missing")
			} else {
				loadedWallet, err := walletManager.LoadWalletFromFile(walletPath)
				if err != nil {
					log.Printf("GUI Error: Failed to load wallet from %s: %v", walletPath, err)
					state.status.Set("Error: Wallet Load Failed")
				} else {
					if loadedWallet.GetAddress() == state.walletAddress {
						state.wallet = loadedWallet // Wallet loaded successfully
						log.Printf("GUI: Successfully loaded wallet for address %s", state.walletAddress)
						state.status.Set("Ready")
					} else {
						log.Printf("GUI Error: Loaded wallet address (%s) does not match expected address (%s)",
							loadedWallet.GetAddress(), state.walletAddress)
						state.status.Set("Error: Wallet Address Mismatch")
					}
				}
			}
		} else {
			log.Printf("GUI: Wallet file for address %s not found. Private key operations will be unavailable.", state.walletAddress)
			state.status.Set("Ready (Wallet Read-Only)") // Indicate wallet is known but private key not loaded
		}
	} else {
		log.Println("GUI Warning: Wallet address not set in blockchain struct.")
		state.status.Set("Error: Wallet Address Missing")
	}

	return state
}

// Methods to safely update state (can be called from any goroutine)
func (s *guiState) SetStatus(newStatus string) {
	s.status.Set(newStatus)
}

func (s *guiState) SetChainHeight(height int) {
	s.chainHeight.Set(height)
}

func (s *guiState) SetPeerCount(count int) {
	s.peerCount.Set(count)
}

func (s *guiState) SetLatestBlock(hash string) {
	s.latestBlock.Set(hash)
}

func (s *guiState) SetMiningStatus(status string) {
	s.miningStatus.Set(status)
}

// UpdateBlockList updates the block list data
func (s *guiState) UpdateBlockList(blocks []*Block) {
	items := make([]interface{}, len(blocks))
	for i, block := range blocks {
		items[i] = block
	}
	s.blockListData.Set(items)
}

// UpdateTransactionList updates the transaction list data
func (s *guiState) UpdateTransactionList(txns []*Transaction) {
	items := make([]interface{}, len(txns))
	for i, txn := range txns {
		items[i] = txn
	}
	s.txnListData.Set(items)
}

// UpdatePeerList updates the peer list data
func (s *guiState) UpdatePeerList() {
	if s.discoveryMgr == nil || s.discoveryMgr.host == nil {
		return
	}

	// Get connected peers from the host
	connectedPeers := s.discoveryMgr.host.Network().Peers()
	items := make([]string, len(connectedPeers))
	for i, p := range connectedPeers {
		items[i] = p.String()
	}
	s.peerListData.Set(items)
}

// updateSelectedPeerCapabilities updates the capabilities list for the selected peer
func (g *GUI) updateSelectedPeerCapabilities() {
	// In a real implementation, this would fetch capabilities from the selected peer
	// For now, we'll just show some placeholder data

	// Create sample capabilities data
	capabilities := []interface{}{
		map[string]interface{}{
			"id":          "cap-1",
			"type":        "inference",
			"description": "Text generation capability",
		},
		map[string]interface{}{
			"id":          "cap-2",
			"type":        "plugin",
			"description": "Data processing plugin",
		},
		map[string]interface{}{
			"id":          "cap-3",
			"type":        "service",
			"description": "API integration service",
		},
	}

	// Update the capabilities list
	g.state.capabilitiesData.Set(capabilities)
}

// GUI represents the graphical user interface for the blockchain application
type GUI struct {
	app    fyne.App
	window fyne.Window
	state  *guiState
}

// NewGUI creates a new GUI instance
func NewGUI(blockchain *BlockchainStruct, db *LevelDB, discoveryMgr *DiscoveryManager, p2pConsensusMgr *P2PConsensusManager, cfg *config.Config, paymentProcessor *PaymentProcessor, role config.Role, wallet *Wallet) *GUI {
	gui := &GUI{
		app:   app.NewWithID("com.knirvchain.gui"),
		state: newGuiState(blockchain, db, discoveryMgr, p2pConsensusMgr, cfg, paymentProcessor, role),
	}

	// Set the wallet if provided
	if wallet != nil {
		gui.state.wallet = wallet
		gui.state.walletAddress = wallet.GetAddress()
	}

	gui.window = gui.app.NewWindow("KNIRVCHAIN - Blockchain Explorer")
	gui.window.Resize(fyne.NewSize(1200, 800)) // Larger window for more content

	// Initialize UI components
	gui.initUI()

	return gui
}

// initUI initializes the user interface components
func (g *GUI) initUI() {
	// Create a modern header with logo and status information
	logoText := canvas.NewText("KNIRVCHAIN", theme.PrimaryColor())
	logoText.TextSize = 24
	logoText.TextStyle = fyne.TextStyle{Bold: true}

	// Status information in header
	statusLabel := widget.NewLabelWithData(g.state.status)
	chainHeightLabel := widget.NewLabel("")
	chainHeightLabel.Bind(binding.IntToStringWithFormat(g.state.chainHeight, "Chain Height: %d"))

	peerCountLabel := widget.NewLabel("")
	peerCountLabel.Bind(binding.IntToStringWithFormat(g.state.peerCount, "Peers: %d"))

	latestBlockLabel := widget.NewLabel("")
	latestBlockText := binding.NewString()
	latestBlockLabel.Bind(latestBlockText)

	// Update the latest block label with prefix
	go func() {
		for {
			val, _ := g.state.latestBlock.Get()
			latestBlockText.Set("Latest Block: " + val)
			time.Sleep(1 * time.Second)
		}
	}()

	// Role badge
	roleBadge := widget.NewLabel("")
	roleText, _ := g.state.roleDisplay.Get()

	// Set badge based on role
	switch g.state.nodeRole {
	case config.Root:
		roleBadge = widget.NewLabelWithStyle(roleText, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		roleBadge.Importance = widget.HighImportance
	case config.RoleBootnode:
		roleBadge = widget.NewLabelWithStyle(roleText, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		roleBadge.Importance = widget.HighImportance
	case config.RolePeer:
		roleBadge = widget.NewLabelWithStyle(roleText, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		roleBadge.Importance = widget.MediumImportance
	case config.RoleClient:
		roleBadge = widget.NewLabelWithStyle(roleText, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		roleBadge.Importance = widget.LowImportance
	}

	// Header layout
	header := container.NewVBox(
		container.NewHBox(
			logoText,
			layout.NewSpacer(),
			roleBadge,
		),
		widget.NewSeparator(),
		container.NewHBox(
			widget.NewIcon(theme.InfoIcon()),
			statusLabel,
			widget.NewSeparator(),
			chainHeightLabel,
			widget.NewSeparator(),
			peerCountLabel,
			widget.NewSeparator(),
			latestBlockLabel,
		),
	)

	// Create tabs for different sections
	tabs := container.NewAppTabs(
		container.NewTabItem("Dashboard", g.createDashboardTab()),
		container.NewTabItem("Blockchain", g.createBlockchainTab()),
		container.NewTabItem("Transactions", g.createTransactionsTab()),
		container.NewTabItem("Network", g.createNetworkTab()),
		container.NewTabItem("Context Records", g.createContextRecordsTab()),
	)

	// Add role-specific tabs
	switch g.state.nodeRole {
	case config.Root:
		// Root-specific tabs
		if g.state.config.PaymentProcessor.Enabled && g.state.paymentProcessor != nil {
			tabs.Append(container.NewTabItem("Payment Processor", g.createPaymentProcessorTab()))
		}
		// Always show root settings tab for Root role, even if payments are disabled
		tabs.Append(container.NewTabItem("Root Settings", g.createRootSettingsTab()))
		// Always add mining tab for Root
		tabs.Append(container.NewTabItem("Mining", g.createMiningTab()))

	case config.RoleBootnode:
		// Bootnode-specific tabs
		if g.state.config.PaymentProcessor.Enabled && g.state.paymentProcessor != nil {
			tabs.Append(container.NewTabItem("Payment Processor", g.createPaymentProcessorTab()))
		}
		// Add mining tab if wallet is available
		if g.state.wallet != nil {
			tabs.Append(container.NewTabItem("Mining", g.createMiningTab()))
		}

	case config.RolePeer:
		// Peer-specific tabs
		// Add mining tab if wallet is available
		if g.state.wallet != nil {
			tabs.Append(container.NewTabItem("Mining", g.createMiningTab()))
		}

	case config.RoleClient:
		// Client-specific tabs
		// No mining or payment processing for clients
	}

	tabs.SetTabLocation(container.TabLocationTop)

	// Main layout
	mainContent := container.NewBorder(
		header, nil, nil, nil,
		tabs,
	)

	g.window.SetContent(mainContent)
}

// createDashboardTab creates the dashboard tab with overview and terminal
func (g *GUI) createDashboardTab() fyne.CanvasObject {
	// Node information card
	nodeInfoCard := widget.NewCard(
		"Node Information",
		"",
		container.NewVBox(
			widget.NewLabelWithStyle("Node Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithData(g.state.status),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Node Role", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithData(g.state.roleDisplay),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Wallet Address", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(g.state.walletAddress),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Chain Height", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithData(binding.IntToString(g.state.chainHeight)),
		),
	)

	// Network information card
	networkInfoCard := widget.NewCard(
		"Network Information",
		"",
		container.NewVBox(
			widget.NewLabelWithStyle("Connected Peers", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithData(binding.IntToString(g.state.peerCount)),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Latest Block", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithData(g.state.latestBlock),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Mining Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabelWithData(g.state.miningStatus),
		),
	)

	// Terminal window for chain activity
	terminalOutput := widget.NewLabelWithData(g.state.terminalOutput)
	terminalScroll := container.NewScroll(terminalOutput)
	// Set a fixed height of 500px for the terminal
	terminalScroll.SetMinSize(fyne.NewSize(0, 500))

	// Bind the terminal output to the label
	// The log capture mechanism will update g.state.terminalOutput
	terminalOutputLabel := widget.NewLabelWithData(g.state.terminalOutput)
	terminalOutputLabel.Wrapping = fyne.TextWrapWord
	terminalCard := widget.NewCard(
		"Chain Activity Terminal",
		"Tailing application log (knirvchain_app.log)",
		terminalScroll,
	)

	// Layout with cards in a grid
	topRow := container.NewGridWithColumns(2,
		nodeInfoCard,
		networkInfoCard,
	)

	// Start the terminal process
	//go g.startTerminalProcess()

	return container.NewBorder(
		topRow,
		nil,
		nil,
		nil,
		terminalCard,
	)
}

// createBlockchainTab creates the blockchain tab content
func (g *GUI) createBlockchainTab() fyne.CanvasObject {
	// Block list
	blockList := widget.NewListWithData(
		g.state.blockListData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Number"),
				widget.NewLabel("Hash"),
				widget.NewLabel("Timestamp"),
				widget.NewLabel("Txns"),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			val, err := item.(binding.Untyped).Get()
			if err != nil {
				return
			}
			block, ok := val.(*Block)
			if !ok {
				return
			}

			items := obj.(*fyne.Container).Objects
			items[0].(*widget.Label).SetText(fmt.Sprintf("%d", block.BlockNumber))
			items[1].(*widget.Label).SetText(fmt.Sprintf("%.8s...", hex.EncodeToString(block.BlockHash)))
			items[2].(*widget.Label).SetText(time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"))
			items[3].(*widget.Label).SetText(fmt.Sprintf("%d", len(block.Transactions)))
		},
	)

	// Block details
	blockDetails := widget.NewLabel("Select a block to view details")

	// Handle block selection
	blockList.OnSelected = func(id widget.ListItemID) {
		item, err := g.state.blockListData.GetValue(id)
		if err != nil {
			return
		}

		block, ok := item.(*Block)
		if !ok {
			return
		}

		details := fmt.Sprintf("Block #%d\nHash: %s\nPrevious Hash: %s\nTimestamp: %s\nNonce: %d\nTransactions: %d\nProposer: %s",
			block.BlockNumber,
			hex.EncodeToString(block.BlockHash),
			hex.EncodeToString(block.PrevHash),
			time.Unix(block.Timestamp, 0).Format("2006-01-02 15:04:05"),
			block.Nonce,
			len(block.Transactions),
			block.ProposerAddress,
		)
		blockDetails.SetText(details)
	}

	// Refresh button
	refreshBtn := widget.NewButton("Refresh Blocks", func() {
		g.state.UpdateBlockList(g.state.blockchain.Blocks)
	})

	// Initial load
	g.state.UpdateBlockList(g.state.blockchain.Blocks)

	// Layout
	return container.NewHSplit(
		container.NewBorder(
			refreshBtn, nil, nil, nil,
			container.NewScroll(blockList),
		),
		container.NewBorder(
			widget.NewLabel("Block Details"), nil, nil, nil,
			container.NewScroll(blockDetails),
		),
	)
}

// createTransactionsTab creates the transactions tab content
func (g *GUI) createTransactionsTab() fyne.CanvasObject {
	// Transaction list
	txnList := widget.NewListWithData(
		g.state.txnListData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Hash"),
				widget.NewLabel("Type"),
				widget.NewLabel("From"),
				widget.NewLabel("To"),
				widget.NewLabel("Value"),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			val, err := item.(binding.Untyped).Get()
			if err != nil {
				return
			}
			txn, ok := val.(*Transaction)
			if !ok {
				return
			}

			items := obj.(*fyne.Container).Objects
			items[0].(*widget.Label).SetText(fmt.Sprintf("%.8s...", txn.TransactionHash))
			items[1].(*widget.Label).SetText(txn.Type)
			items[2].(*widget.Label).SetText(fmt.Sprintf("%.8s...", txn.From))
			items[3].(*widget.Label).SetText(fmt.Sprintf("%.8s...", txn.To))
			items[4].(*widget.Label).SetText(fmt.Sprintf("%d", txn.Value))
		},
	)

	// Transaction details
	txnDetails := widget.NewLabel("Select a transaction to view details")

	// Handle transaction selection
	txnList.OnSelected = func(id widget.ListItemID) {
		item, err := g.state.txnListData.GetValue(id)
		if err != nil {
			return
		}

		txn, ok := item.(*Transaction)
		if !ok {
			return
		}

		// Format data as JSON for better readability
		dataJSON, err := json.MarshalIndent(txn.Data, "", "  ")
		if err != nil {
			dataJSON = []byte("Error formatting data")
		}

		details := fmt.Sprintf("Transaction Hash: %s\nType: %s\nFrom: %s\nTo: %s\nValue: %d\nFee: %d\nTimestamp: %s\nStatus: %s\nData:\n%s",
			txn.TransactionHash,
			txn.Type,
			txn.From,
			txn.To,
			txn.Value,
			txn.Fee,
			time.Unix(txn.Timestamp, 0).Format("2006-01-02 15:04:05"),
			txn.Status,
			string(dataJSON),
		)
		txnDetails.SetText(details)
	}

	// Refresh button
	refreshBtn := widget.NewButton("Refresh Transactions", func() {
		// Get all transactions from all blocks
		var allTxns []*Transaction
		for _, block := range g.state.blockchain.Blocks {
			allTxns = append(allTxns, block.Transactions...)
		}
		g.state.UpdateTransactionList(allTxns)
	})

	// Initial load
	var allTxns []*Transaction
	for _, block := range g.state.blockchain.Blocks {
		allTxns = append(allTxns, block.Transactions...)
	}
	g.state.UpdateTransactionList(allTxns)

	// Layout
	return container.NewHSplit(
		container.NewBorder(
			refreshBtn, nil, nil, nil,
			container.NewScroll(txnList),
		),
		container.NewBorder(
			widget.NewLabel("Transaction Details"), nil, nil, nil,
			container.NewScroll(txnDetails),
		),
	)
}

// createNetworkTab creates the network tab content
func (g *GUI) createNetworkTab() fyne.CanvasObject {
	// Peer list
	peerList := widget.NewListWithData(
		g.state.peerListData,
		func() fyne.CanvasObject {
			return widget.NewLabel("Peer ID")
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			val, err := item.(binding.String).Get()
			if err != nil {
				return
			}
			obj.(*widget.Label).SetText(val)
		},
	)

	// Peer capabilities and plugins list
	peerCapabilitiesList := widget.NewListWithData(
		g.state.capabilitiesData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Capability ID"),
				widget.NewLabel("Type"),
				widget.NewLabel("Description"),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			val, err := item.(binding.Untyped).Get()
			if err != nil {
				return
			}

			// This would be a proper capability object in a real implementation
			// For now, we'll just display placeholder data
			capability, ok := val.(map[string]interface{})
			if !ok {
				return
			}

			items := obj.(*fyne.Container).Objects
			items[0].(*widget.Label).SetText(fmt.Sprintf("%v", capability["id"]))
			items[1].(*widget.Label).SetText(fmt.Sprintf("%v", capability["type"]))
			items[2].(*widget.Label).SetText(fmt.Sprintf("%v", capability["description"]))
		},
	)

	// Refresh button for peers
	refreshPeersBtn := widget.NewButton("Refresh Peers", func() {
		if g.state.discoveryMgr != nil {
			g.state.UpdatePeerList()
		}
	})

	// Refresh button for peer capabilities
	refreshCapabilitiesBtn := widget.NewButton("Refresh Capabilities", func() {
		// This would call a method to update capabilities for the selected peer
		// For now, we'll just show a placeholder
		g.updateSelectedPeerCapabilities()
	})

	// Initial load
	if g.state.discoveryMgr != nil {
		g.state.UpdatePeerList()
	}

	// Add selection handler for peer list
	peerList.OnSelected = func(id widget.ListItemID) {
		// When a peer is selected, update the capabilities view
		g.updateSelectedPeerCapabilities()
	}

	// Layout
	peersContainer := container.NewBorder(
		refreshPeersBtn, nil, nil, nil,
		container.NewScroll(peerList),
	)

	capabilitiesContainer := container.NewBorder(
		refreshCapabilitiesBtn,
		nil, nil, nil,
		container.NewScroll(peerCapabilitiesList),
	)

	return container.NewHSplit(
		container.NewBorder(
			widget.NewLabel("Connected Peers"), nil, nil, nil,
			peersContainer,
		),
		container.NewBorder(
			widget.NewLabel("Peer Plugins & Capabilities"), nil, nil, nil,
			capabilitiesContainer,
		),
	)
}

// createContextRecordsTab creates the context records tab with search functionality
func (g *GUI) createContextRecordsTab() fyne.CanvasObject {
	// Search form
	searchTypeSelect := widget.NewSelect([]string{
		"By ID",
		"By Capability ID",
		"By Interaction Type",
		"By Initiator",
	}, nil)
	searchTypeSelect.SetSelectedIndex(0) // Default to search by ID

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Enter search term...")

	// Context records list
	contextRecordsList := widget.NewListWithData(
		g.state.contextRecordsData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("ID"),
				widget.NewLabel("Capability"),
				widget.NewLabel("Type"),
				widget.NewLabel("Initiator"),
				widget.NewLabel("Timestamp"),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			val, err := item.(binding.Untyped).Get()
			if err != nil {
				return
			}
			record, ok := val.(*types.ContextRecord)
			if !ok {
				return
			}

			items := obj.(*fyne.Container).Objects

			// Format timestamp
			timeStr := time.Unix(record.Timestamp, 0).Format("2006-01-02 15:04:05")

			// Update labels
			items[0].(*widget.Label).SetText(fmt.Sprintf("%.8s...", record.ID))
			items[1].(*widget.Label).SetText(fmt.Sprintf("%.8s...", record.CapabilityID))
			items[2].(*widget.Label).SetText(string(record.InteractionType))
			items[3].(*widget.Label).SetText(fmt.Sprintf("%.8s...", record.Initiator))
			items[4].(*widget.Label).SetText(timeStr)
		},
	)

	// Context record details
	recordDetails := widget.NewLabel("Select a context record to view details")

	// Handle record selection
	contextRecordsList.OnSelected = func(id widget.ListItemID) {
		item, err := g.state.contextRecordsData.GetValue(id)
		if err != nil {
			return
		}

		record, ok := item.(*types.ContextRecord)
		if !ok {
			return
		}

		// Format details as JSON for better readability
		detailsJSON, err := json.MarshalIndent(record.Details, "", "  ")
		if err != nil {
			detailsJSON = []byte("Error formatting details")
		}

		details := fmt.Sprintf("ID: %s\nCapability ID: %s\nInteraction Type: %s\nInitiator: %s\nTimestamp: %s\nInput Hash: %s\nOutput Hash: %s\n\nDetails:\n%s",
			record.ID,
			record.CapabilityID,
			record.InteractionType,
			record.Initiator,
			time.Unix(record.Timestamp, 0).Format("2006-01-02 15:04:05"),
			record.InputHash,
			record.OutputHash,
			string(detailsJSON),
		)
		recordDetails.SetText(details)
	}

	// Search button
	searchBtn := widget.NewButton("Search", func() {
		if searchEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("search term is required"), g.window)
			return
		}

		var records []*types.ContextRecord
		// Perform search based on selected type
		switch searchTypeSelect.Selected {
		case "By ID":
			// Search by ID
			protoRecord, err := g.state.db.GetContextRecord(searchEntry.Text)
			if err == nil && protoRecord != nil {
				// Convert proto to ContextRecord
				record, err := ConvertProtoToContextRecord(protoRecord)
				if err == nil {
					records = []*types.ContextRecord{&record}
				}
			}
		case "By Capability ID":
			// Search by capability ID
			protoRecords, err := g.state.db.GetContextRecordsForCapability(searchEntry.Text)
			if err == nil {
				records = ConvertProtoContextRecordsToContextRecords(protoRecords)
			}
		case "By Interaction Type":
			// This would require a new method in the database
			dialog.ShowInformation("Not Implemented", "Search by Interaction Type is not yet implemented", g.window)
			return
		case "By Initiator":
			// This would require a new method in the database
			dialog.ShowInformation("Not Implemented", "Search by Initiator is not yet implemented", g.window)
			return
		}

		if len(records) == 0 {
			dialog.ShowInformation("No Results", "No context records found matching your search criteria", g.window)
			return
		}

		// Update the list with search results
		items := make([]interface{}, len(records))
		for i, record := range records {
			items[i] = record
		}
		g.state.contextRecordsData.Set(items)
	})

	// Clear button
	clearBtn := widget.NewButton("Clear", func() {
		searchEntry.SetText("")
		g.state.contextRecordsData.Set([]interface{}{})
		recordDetails.SetText("Select a context record to view details")
	})

	// Search form layout
	searchForm := container.NewVBox(
		widget.NewLabel("Search Context Records"),
		searchTypeSelect,
		searchEntry,
		container.NewHBox(
			searchBtn,
			clearBtn,
		),
	)

	// Layout
	return container.NewHSplit(
		container.NewBorder(
			searchForm, nil, nil, nil,
			container.NewScroll(contextRecordsList),
		),
		container.NewBorder(
			widget.NewLabel("Context Record Details"), nil, nil, nil,
			container.NewScroll(recordDetails),
		),
	)
}

// createPaymentProcessorTab creates the payment processor management tab (root mode only)
func (g *GUI) createPaymentProcessorTab() fyne.CanvasObject {
	// Check if payment processor is available
	if g.state.paymentProcessor == nil {
		return widget.NewLabel("Payment processor is not initialized. Start the node with --root flag to enable payment processing.")
	}

	// Payment processor status
	statusCard := widget.NewCard(
		"Payment Processor Status",
		"",
		container.NewVBox(
			widget.NewLabelWithStyle("Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Running"),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Webhook Port", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(fmt.Sprintf("%d", g.state.config.PaymentProcessor.WebhookPort)),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Master Wallet", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(g.state.paymentProcessor.disbursementWallet.GetAddress()),
		),
	)

	// Payment gateway configuration
	stripeSecretEntry := widget.NewPasswordEntry()
	stripeSecretEntry.SetPlaceHolder("Stripe Secret Key")
	stripeSecretEntry.SetText(g.state.config.PaymentProcessor.StripeSecretKey)

	stripeWebhookSecretEntry := widget.NewPasswordEntry()
	stripeWebhookSecretEntry.SetPlaceHolder("Stripe Webhook Secret")
	stripeWebhookSecretEntry.SetText(g.state.config.PaymentProcessor.StripeWebhookSecret)

	coinbaseAPIKeyEntry := widget.NewPasswordEntry()
	coinbaseAPIKeyEntry.SetPlaceHolder("Coinbase API Key")
	coinbaseAPIKeyEntry.SetText(g.state.config.PaymentProcessor.CoinbaseAPIKey)

	coinbaseWebhookSecretEntry := widget.NewPasswordEntry()
	coinbaseWebhookSecretEntry.SetPlaceHolder("Coinbase Webhook Secret")
	coinbaseWebhookSecretEntry.SetText(g.state.config.PaymentProcessor.CoinbaseWebhookSecret)

	// Token pricing configuration
	usdPerTokenEntry := widget.NewEntry()
	usdPerTokenEntry.SetPlaceHolder("USD per Token")
	usdPerTokenEntry.SetText(fmt.Sprintf("%.4f", g.state.config.PaymentProcessor.USDPerToken))

	ethPerTokenEntry := widget.NewEntry()
	ethPerTokenEntry.SetPlaceHolder("ETH per Token")
	ethPerTokenEntry.SetText(fmt.Sprintf("%.8f", g.state.config.PaymentProcessor.ETHPerToken))

	// Save configuration button
	saveConfigBtn := widget.NewButton("Save Configuration", func() {
		// Update config with form values
		g.state.config.PaymentProcessor.StripeSecretKey = stripeSecretEntry.Text
		g.state.config.PaymentProcessor.StripeWebhookSecret = stripeWebhookSecretEntry.Text
		g.state.config.PaymentProcessor.CoinbaseAPIKey = coinbaseAPIKeyEntry.Text
		g.state.config.PaymentProcessor.CoinbaseWebhookSecret = coinbaseWebhookSecretEntry.Text

		// Parse and update token pricing
		usdPerToken, err := strconv.ParseFloat(usdPerTokenEntry.Text, 64)
		if err == nil && usdPerToken > 0 {
			g.state.config.PaymentProcessor.USDPerToken = usdPerToken
		}

		ethPerToken, err := strconv.ParseFloat(ethPerTokenEntry.Text, 64)
		if err == nil && ethPerToken > 0 {
			g.state.config.PaymentProcessor.ETHPerToken = ethPerToken
		}

		// Save config to file
		configPath, err := config.GetConfigPath()
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to get config path: %v", err), g.window)
			return
		}

		if _, err := config.SaveConfig(configPath, g.state.config); err != nil {
			dialog.ShowError(fmt.Errorf("failed to save config to %s: %v", configPath, err), g.window)
			return
		}

		dialog.ShowInformation("Success", "Payment processor configuration saved", g.window)
	})

	// Restart payment processor button
	restartBtn := widget.NewButton("Restart Payment Processor", func() {
		// Stop the current payment processor
		if g.state.paymentProcessor != nil {
			if err := g.state.paymentProcessor.Stop(); err != nil {
				dialog.ShowError(fmt.Errorf("failed to stop payment processor: %v", err), g.window)
				return
			}
		}

		// Initialize a new payment processor with updated config
		paymentProcessor, err := initPaymentProcessor(g.state.config, g.state.db, g.state.nodeRole)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to initialize payment processor: %v", err), g.window)
			return
		}

		g.state.paymentProcessor = paymentProcessor
		dialog.ShowInformation("Success", "Payment processor restarted with new configuration", g.window)
	})

	// Layout
	configForm := widget.NewCard(
		"Payment Gateway Configuration",
		"Configure payment gateways and token pricing",
		container.NewVBox(
			widget.NewLabelWithStyle("Stripe Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			stripeSecretEntry,
			stripeWebhookSecretEntry,
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Coinbase Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			coinbaseAPIKeyEntry,
			coinbaseWebhookSecretEntry,
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Token Pricing", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewGridWithColumns(2,
				container.NewHBox(widget.NewLabel("USD per Token:"), usdPerTokenEntry),
				container.NewHBox(widget.NewLabel("ETH per Token:"), ethPerTokenEntry),
			),
			widget.NewSeparator(),
			container.NewHBox(
				saveConfigBtn,
				restartBtn,
			),
		),
	)

	// Layout with cards in a grid
	return container.NewVBox(
		statusCard,
		configForm,
	)
}

// createRootSettingsTab creates the root settings tab (root mode only)
func (g *GUI) createRootSettingsTab() fyne.CanvasObject {
	// Node configuration
	nodeRPCEntry := widget.NewEntry()
	nodeRPCEntry.SetPlaceHolder("Node RPC URL")
	nodeRPCEntry.SetText(g.state.config.PaymentProcessor.NodeRPC)

	webhookPortEntry := widget.NewEntry()
	webhookPortEntry.SetPlaceHolder("Webhook Port")
	webhookPortEntry.SetText(fmt.Sprintf("%d", g.state.config.PaymentProcessor.WebhookPort))

	// Token configuration
	tokenSymbolEntry := widget.NewEntry()
	tokenSymbolEntry.SetPlaceHolder("Token Symbol")
	tokenSymbolEntry.SetText(g.state.config.PaymentProcessor.TokenSymbol)

	tokenDecimalsEntry := widget.NewEntry()
	tokenDecimalsEntry.SetPlaceHolder("Token Decimals")
	tokenDecimalsEntry.SetText(fmt.Sprintf("%d", g.state.config.PaymentProcessor.TokenDecimals))

	// Save configuration button
	saveConfigBtn := widget.NewButton("Save Configuration", func() {
		// Update config with form values
		g.state.config.PaymentProcessor.NodeRPC = nodeRPCEntry.Text

		// Parse and update webhook port
		webhookPort, err := strconv.Atoi(webhookPortEntry.Text)
		if err == nil && webhookPort > 0 {
			g.state.config.PaymentProcessor.WebhookPort = webhookPort
		}

		// Update token configuration
		g.state.config.PaymentProcessor.TokenSymbol = tokenSymbolEntry.Text

		// Parse and update token decimals
		tokenDecimals, err := strconv.Atoi(tokenDecimalsEntry.Text)
		if err == nil && tokenDecimals >= 0 {
			g.state.config.PaymentProcessor.TokenDecimals = tokenDecimals
		}

		// Save config to file
		configPath, err := config.GetConfigPath()
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to get config path: %v", err), g.window)
			return
		}

		if _, err := config.SaveConfig(configPath, g.state.config); err != nil {
			dialog.ShowError(fmt.Errorf("failed to save config to %s: %v", configPath, err), g.window)
			return
		}

		dialog.ShowInformation("Success", "Root settings saved", g.window)
	})

	// Layout
	configForm := widget.NewCard(
		"Root Node Configuration",
		"Configure root node settings",
		container.NewVBox(
			widget.NewLabelWithStyle("Node Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewGridWithColumns(2,
				container.NewHBox(widget.NewLabel("Node RPC URL:"), nodeRPCEntry),
				container.NewHBox(widget.NewLabel("Webhook Port:"), webhookPortEntry),
			),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Token Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewGridWithColumns(2,
				container.NewHBox(widget.NewLabel("Token Symbol:"), tokenSymbolEntry),
				container.NewHBox(widget.NewLabel("Token Decimals:"), tokenDecimalsEntry),
			),
			widget.NewSeparator(),
			saveConfigBtn,
		),
	)

	// Network settings
	enableRootModeCheck := widget.NewCheck("Enable Root Mode", func(enabled bool) {
		g.state.config.IsRoot = enabled
		g.state.config.PaymentProcessor.Enabled = enabled
	})
	enableRootModeCheck.SetChecked(g.state.config.IsRoot)

	applyNetworkSettingsBtn := widget.NewButton("Apply Network Settings", func() {
		// Save config to file
		configPath, err := config.GetConfigPath()
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to get config path: %v", err), g.window)
			return
		}

		if _, err := config.SaveConfig(configPath, g.state.config); err != nil {
			dialog.ShowError(fmt.Errorf("failed to save config to %s: %v", configPath, err), g.window)
			return
		}

		dialog.ShowInformation("Success", "Network settings saved. Restart the node for changes to take effect.", g.window)
	})

	networkSettingsForm := widget.NewCard(
		"Network Settings",
		"Configure network mode settings",
		container.NewVBox(
			enableRootModeCheck,
			widget.NewSeparator(),
			applyNetworkSettingsBtn,
		),
	)

	// Layout with cards in a grid
	return container.NewVBox(
		configForm,
		networkSettingsForm,
	)
}

// createMiningTab creates the mining tab content
func (g *GUI) createMiningTab() fyne.CanvasObject {
	// Mining status
	miningStatusLabel := widget.NewLabelWithData(g.state.miningStatus)

	// Mining controls
	difficultyEntry := widget.NewEntry()
	difficultyEntry.SetPlaceHolder("Difficulty (1-5)")
	difficultyEntry.SetText("1")

	miningBtn := widget.NewButton("START MINING", nil)

	miningBtn.OnTapped = func() {
		g.state.mu.Lock()
		isCurrentlyMining := g.state.blockchain.IsActivelyMining()
		g.state.mu.Unlock()

		if isCurrentlyMining {
			// Stop mining
			g.state.blockchain.StopMining = true // Signal the mining loop to stop
			// Status will be updated by refreshDataLoop
		} else {
			// Start mining
			// Difficulty is handled internally by the blockchain's mining logic for now
			// If you want to set difficulty from GUI, you'd need a method in BlockchainStruct
			g.state.blockchain.StopMining = false // Ensure stop flag is clear
			go g.state.blockchain.StartMining()   // Start the actual mining process
			// Status will be updated by refreshDataLoop
		}
	}

	// Periodically update button text based on actual mining state
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			g.state.mu.RLock()
			isCurrentlyMining := g.state.blockchain.IsActivelyMining()
			g.state.mu.RUnlock()

			if isCurrentlyMining {
				miningBtn.SetText("STOP MINING")
			} else {
				miningBtn.SetText("START MINING")
			}
		}
	}()

	// Mining rewards
	rewardsCard := widget.NewCard(
		"Mining Rewards",
		"",
		container.NewVBox(
			widget.NewLabelWithStyle("Wallet Address", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(g.state.walletAddress),
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Total Mined", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel("0 blocks"), // This would be dynamic in a real implementation
		),
	)

	// Layout
	controlsContainer := container.NewVBox(
		widget.NewLabelWithStyle("Mining Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		miningStatusLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Mining Controls", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(
			widget.NewLabel("Difficulty:"),
			difficultyEntry,
		),
		miningBtn,
	)

	return container.NewGridWithColumns(2,
		widget.NewCard("Mining Controls", "", controlsContainer),
		rewardsCard,
	)
}

// Start starts the GUI
func (g *GUI) Start() {
	// Start data refresh goroutine *after* UI is built but *before* blocking
	go g.refreshDataLoop()

	// Show the window and start the event loop (this blocks)
	g.window.ShowAndRun()
}

// refreshDataLoop periodically updates the GUI data
func (g *GUI) refreshDataLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Update chain height
			g.state.SetChainHeight(len(g.state.blockchain.Blocks))

			// Update latest block
			if len(g.state.blockchain.Blocks) > 0 {
				latestBlock := g.state.blockchain.Blocks[len(g.state.blockchain.Blocks)-1]
				g.state.SetLatestBlock(hex.EncodeToString(latestBlock.BlockHash))
				log.Printf("GUI Refresh: Chain Height: %d, Latest Block: %.8s...", len(g.state.blockchain.Blocks), hex.EncodeToString(latestBlock.BlockHash))
				// Update the block list displayed in the GUI
				// Make a copy to avoid race conditions if bc.Blocks is modified elsewhere
				g.state.UpdateBlockList(append([]*Block(nil), g.state.blockchain.Blocks...))
			} else {
				log.Printf("GUI Refresh: Chain Height: %d (Genesis only or empty)", len(g.state.blockchain.Blocks))
			}

			if g.state.blockchain.IsActivelyMining() {
				g.state.SetMiningStatus("Mining Active")
			} else {
				g.state.SetMiningStatus("Not Mining")
			}

			// Update peer count
			if g.state.discoveryMgr != nil && g.state.discoveryMgr.host != nil {
				g.state.SetPeerCount(len(g.state.discoveryMgr.host.Network().Peers()))
			}
		}
	}
}

// RunGUI starts the GUI with the given components
func RunGUI(bc *BlockchainStruct, db *LevelDB, discoveryMgr *DiscoveryManager, p2pConsensusMgr *P2PConsensusManager, cfg *config.Config, paymentProcessor *PaymentProcessor, logBinding binding.String, role config.Role, wallet *Wallet) {
	gui := NewGUI(bc, db, discoveryMgr, p2pConsensusMgr, cfg, paymentProcessor, role, wallet)

	// If a log binding is provided, use it instead of creating our own
	if logBinding != nil {
		gui.state.terminalOutput = logBinding
	} else {
		// Otherwise set up our own log capture
		setupLogCapture(gui)
	}

	// Run the GUI
	gui.Run()
}

// --- Log Capture ---
const maxLogLines = 200 // Max lines to keep in GUI terminal

type boundWriter struct {
	binding  binding.String
	buffer   *strings.Builder
	mu       sync.Mutex
	maxLines int
	//lineCount int
}

// newBoundWriter creates a writer that updates a binding.String with log output
// It implements io.Writer interface so it can be used with log.SetOutput
// maxLines controls how many lines of history to keep in the terminal
func newBoundWriter(b binding.String, maxLines int) *boundWriter {
	return &boundWriter{binding: b, buffer: new(strings.Builder), maxLines: maxLines}
}

// Write implements the io.Writer interface for boundWriter
// It updates the binding.String with the written content and limits the number of lines
func (bw *boundWriter) Write(p []byte) (n int, err error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	// Write the new content to the buffer
	bw.buffer.Write(p)

	// Implement line limiting to prevent the terminal from growing too large
	content := bw.buffer.String()
	lines := strings.Split(content, "\n")

	// If we have more lines than our maximum, trim the oldest lines
	if len(lines) > bw.maxLines {
		// Keep only the most recent maxLines
		startIdx := len(lines) - bw.maxLines

		// Reset the buffer and write only the lines we want to keep
		bw.buffer.Reset()
		bw.buffer.WriteString(strings.Join(lines[startIdx:], "\n"))
	}

	// Update the binding with the current buffer content
	bw.binding.Set(bw.buffer.String())

	return len(p), nil
}

// Run starts the Fyne GUI
func (g *GUI) Run() {
	// Set up window close event
	g.window.SetOnClosed(func() {
		log.Println("GUI window closed, shutting down application...")
	})

	// Show and run the window (this is blocking)
	g.window.ShowAndRun()
}

// setupLogCapture configures log output to be captured in the GUI terminal
func setupLogCapture(gui *GUI) {
	// Create a bound writer that will update the terminal output binding
	writer := newBoundWriter(gui.state.terminalOutput, maxLogLines)

	// Create a multi-writer to send logs to both the original output and our bound writer
	multiWriter := io.MultiWriter(os.Stdout, writer)

	// Configure the standard logger to use our multi-writer
	log.SetOutput(multiWriter)

	// Log a message to confirm setup
	log.Println("Log capture initialized for GUI terminal")
}

// InitializeGUI is the entry point for the Fyne GUI
func InitializeGUI(bc *BlockchainStruct, db *LevelDB, discoveryMgr *DiscoveryManager, p2pConsensusMgr *P2PConsensusManager, cfg *config.Config, paymentProcessor *PaymentProcessor, role config.Role, wallet *Wallet, ws *WalletServer, chromemManager *ChromemManager) *GUI {
	// Create the GUI
	gui := NewGUI(bc, db, discoveryMgr, p2pConsensusMgr, cfg, paymentProcessor, role, wallet)

	// Set up log capture for the terminal
	setupLogCapture(gui)

	// Run the GUI (blocking call)
	gui.Run()
	return gui
}

package screens

import (
	"fmt"
	"strings"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/core"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/ui"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/internal/ui/components"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WalletItem represents a wallet item
type WalletItem struct {
	address    string
	path       string
	balance    string
	lastUsed   string
	isSelected bool
}

// Title returns the item title
func (i WalletItem) Title() string { return i.address }

// Description returns the item description
func (i WalletItem) Description() string {
	return fmt.Sprintf("Balance: %s • Last Used: %s", i.balance, i.lastUsed)
}

// FilterValue returns the filter value
func (i WalletItem) FilterValue() string { return i.address }

// WalletKeyMap defines keybindings for the wallet screen
type WalletKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	New      key.Binding
	Import   key.Binding
	Export   key.Binding
	Delete   key.Binding
	Back     key.Binding
	Refresh  key.Binding
	Select   key.Binding
	Password key.Binding
}

// DefaultWalletKeyMap returns the default keybindings
func DefaultWalletKeyMap() WalletKeyMap {
	return WalletKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new wallet"),
		),
		Import: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "import wallet"),
		),
		Export: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "export wallet"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete wallet"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Password: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "change password"),
		),
	}
}

// WalletScreen represents the wallet management screen
type WalletScreen struct {
	styles        ui.Styles
	list          list.Model
	keyMap        WalletKeyMap
	walletManager *core.WalletManager
	width         int
	height        int
	loading       bool
	error         string

	// Modals
	newWalletForm      components.Form
	importWalletForm   components.Form
	exportWalletForm   components.Form
	passwordForm       components.Form
	confirmDeleteModal components.ConfirmModal

	// State
	showNewWallet     bool
	showImportWallet  bool
	showExportWallet  bool
	showPasswordForm  bool
	showConfirmDelete bool
	selectedWallet    string

	// Parent screen
	parent ui.Screen
}

// NewWalletScreen creates a new wallet management screen
func NewWalletScreen(styles ui.Styles, walletManager *core.WalletManager, parent ui.Screen) *WalletScreen {
	keyMap := DefaultWalletKeyMap()

	// Create list
	listDelegate := list.NewDefaultDelegate()
	listDelegate.Styles.SelectedTitle = listDelegate.Styles.SelectedTitle.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary).
		Bold(true)
	listDelegate.Styles.SelectedDesc = listDelegate.Styles.SelectedDesc.
		Foreground(styles.Theme.Text).
		Background(styles.Theme.Primary)

	walletList := list.New([]list.Item{}, listDelegate, 80, 20)
	walletList.Title = "Wallet Management"
	walletList.SetShowStatusBar(false)
	walletList.SetFilteringEnabled(false)
	walletList.Styles.Title = styles.Title
	walletList.Styles.TitleBar = styles.Header

	// Create forms
	newWalletForm := components.NewForm(styles, 60)
	newWalletForm.AddField(components.NewPasswordField("Password", "Enter password", "", true))
	newWalletForm.AddField(components.NewPasswordField("Confirm Password", "Confirm password", "", true))
	newWalletForm.SetSubmitLabel("Create")
	newWalletForm.SetCancelLabel("Cancel")

	importWalletForm := components.NewForm(styles, 60)
	importWalletForm.AddField(components.NewFormField("Private Key", "Enter private key", "", true))
	importWalletForm.AddField(components.NewPasswordField("Password", "Enter password", "", true))
	importWalletForm.AddField(components.NewPasswordField("Confirm Password", "Confirm password", "", true))
	importWalletForm.SetSubmitLabel("Import")
	importWalletForm.SetCancelLabel("Cancel")

	exportWalletForm := components.NewForm(styles, 60)
	exportWalletForm.AddField(components.NewPasswordField("Password", "Enter password", "", true))
	exportWalletForm.SetSubmitLabel("Export")
	exportWalletForm.SetCancelLabel("Cancel")

	passwordForm := components.NewForm(styles, 60)
	passwordForm.AddField(components.NewPasswordField("Current Password", "Enter current password", "", true))
	passwordForm.AddField(components.NewPasswordField("New Password", "Enter new password", "", true))
	passwordForm.AddField(components.NewPasswordField("Confirm New Password", "Confirm new password", "", true))
	passwordForm.SetSubmitLabel("Change")
	passwordForm.SetCancelLabel("Cancel")

	// Create modals
	confirmDeleteModal := components.NewConfirmModal(
		styles,
		"Confirm Delete",
		"Are you sure you want to delete this wallet? This action cannot be undone.",
		60,
		10,
	)

	return &WalletScreen{
		styles:             styles,
		list:               walletList,
		keyMap:             keyMap,
		walletManager:      walletManager,
		width:              80,
		height:             24,
		loading:            false,
		error:              "",
		newWalletForm:      newWalletForm,
		importWalletForm:   importWalletForm,
		exportWalletForm:   exportWalletForm,
		passwordForm:       passwordForm,
		confirmDeleteModal: confirmDeleteModal,
		showNewWallet:      false,
		showImportWallet:   false,
		showExportWallet:   false,
		showPasswordForm:   false,
		showConfirmDelete:  false,
		selectedWallet:     "",
		parent:             parent,
	}
}

// Init initializes the screen
func (w *WalletScreen) Init() tea.Cmd {
	return w.loadWallets()
}

// loadWallets loads the wallets
func (w *WalletScreen) loadWallets() tea.Cmd {
	return func() tea.Msg {
		w.loading = true

		// In a real implementation, this would load wallets from the wallet manager
		// For now, we'll create some sample wallets
		items := []list.Item{
			WalletItem{
				address:  "0x1234567890abcdef1234567890abcdef12345678",
				path:     "/home/user/.knirvchain/wallets/wallet1.json",
				balance:  "100 KNRV",
				lastUsed: "2025-06-16 14:30:45",
			},
			WalletItem{
				address:  "0xabcdef1234567890abcdef1234567890abcdef12",
				path:     "/home/user/.knirvchain/wallets/wallet2.json",
				balance:  "250 KNRV",
				lastUsed: "2025-06-15 09:12:33",
			},
			WalletItem{
				address:  "0x7890abcdef1234567890abcdef1234567890abcd",
				path:     "/home/user/.knirvchain/wallets/wallet3.json",
				balance:  "75 KNRV",
				lastUsed: "2025-06-14 18:45:21",
			},
		}

		return WalletsLoadedMsg{items: items}
	}
}

// WalletsLoadedMsg is sent when wallets are loaded
type WalletsLoadedMsg struct {
	items []list.Item
}

// Update handles user input
func (w *WalletScreen) Update(msg tea.Msg) (ui.Screen, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case WalletsLoadedMsg:
		w.loading = false
		w.list.SetItems(msg.items)
		return w, nil

	case tea.KeyMsg:
		// Handle form and modal inputs first
		if w.showNewWallet {
			form, cmd := w.newWalletForm.Update(msg)
			w.newWalletForm = form

			switch msg.String() {
			case "enter":
				if w.newWalletForm.Validate() {
					// Create new wallet
					values := w.newWalletForm.GetValues()
					password := values["Password"]
					confirmPassword := values["Confirm Password"]

					if password != confirmPassword {
						w.error = "Passwords do not match"
						return w, nil
					}

					// In a real implementation, this would create a new wallet
					w.showNewWallet = false
					return w, w.loadWallets()
				}
			case "esc":
				w.showNewWallet = false
				return w, nil
			}

			return w, cmd
		}

		if w.showImportWallet {
			form, cmd := w.importWalletForm.Update(msg)
			w.importWalletForm = form

			switch msg.String() {
			case "enter":
				if w.importWalletForm.Validate() {
					// Import wallet
					values := w.importWalletForm.GetValues()
					privateKey := values["Private Key"]
					password := values["Password"]
					confirmPassword := values["Confirm Password"]

					if password != confirmPassword {
						w.error = "Passwords do not match"
						return w, nil
					}

					// In a real implementation, this would import a wallet
					_ = privateKey // Unused for now

					w.showImportWallet = false
					return w, w.loadWallets()
				}
			case "esc":
				w.showImportWallet = false
				return w, nil
			}

			return w, cmd
		}

		if w.showExportWallet {
			form, cmd := w.exportWalletForm.Update(msg)
			w.exportWalletForm = form

			switch msg.String() {
			case "enter":
				if w.exportWalletForm.Validate() {
					// Export wallet
					values := w.exportWalletForm.GetValues()
					password := values["Password"]

					// In a real implementation, this would export the wallet
					_ = password // Unused for now

					w.showExportWallet = false
					return w, nil
				}
			case "esc":
				w.showExportWallet = false
				return w, nil
			}

			return w, cmd
		}

		if w.showPasswordForm {
			form, cmd := w.passwordForm.Update(msg)
			w.passwordForm = form

			switch msg.String() {
			case "enter":
				if w.passwordForm.Validate() {
					// Change password
					values := w.passwordForm.GetValues()
					currentPassword := values["Current Password"]
					newPassword := values["New Password"]
					confirmNewPassword := values["Confirm New Password"]

					if newPassword != confirmNewPassword {
						w.error = "New passwords do not match"
						return w, nil
					}

					// In a real implementation, this would change the wallet password
					_ = currentPassword // Unused for now

					w.showPasswordForm = false
					return w, nil
				}
			case "esc":
				w.showPasswordForm = false
				return w, nil
			}

			return w, cmd
		}

		if w.showConfirmDelete {
			modal, cmd := w.confirmDeleteModal.Update(msg)
			w.confirmDeleteModal = modal

			if !w.confirmDeleteModal.IsVisible() {
				w.showConfirmDelete = false

				// If confirmed, delete the wallet
				if w.confirmDeleteModal.SelectedButton() == "Confirm" {
					// In a real implementation, this would delete the wallet
					return w, w.loadWallets()
				}
			}

			return w, cmd
		}

		// Handle main screen inputs
		switch {
		case key.Matches(msg, w.keyMap.Back):
			return w.parent, nil

		case key.Matches(msg, w.keyMap.New):
			w.showNewWallet = true
			w.newWalletForm.Focus()
			return w, nil

		case key.Matches(msg, w.keyMap.Import):
			w.showImportWallet = true
			w.importWalletForm.Focus()
			return w, nil

		case key.Matches(msg, w.keyMap.Export):
			i, ok := w.list.SelectedItem().(WalletItem)
			if ok {
				w.selectedWallet = i.address
				w.showExportWallet = true
				w.exportWalletForm.Focus()
			}
			return w, nil

		case key.Matches(msg, w.keyMap.Delete):
			i, ok := w.list.SelectedItem().(WalletItem)
			if ok {
				w.selectedWallet = i.address
				w.showConfirmDelete = true
				w.confirmDeleteModal.Show()
			}
			return w, nil

		case key.Matches(msg, w.keyMap.Password):
			i, ok := w.list.SelectedItem().(WalletItem)
			if ok {
				w.selectedWallet = i.address
				w.showPasswordForm = true
				w.passwordForm.Focus()
			}
			return w, nil

		case key.Matches(msg, w.keyMap.Refresh):
			return w, w.loadWallets()
		}

	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height
		w.list.SetWidth(msg.Width)
		w.list.SetHeight(msg.Height - 10)
	}

	// Update the list
	var cmd tea.Cmd
	w.list, cmd = w.list.Update(msg)
	cmds = append(cmds, cmd)

	return w, tea.Batch(cmds...)
}

// View renders the screen
func (w WalletScreen) View() string {
	if w.loading {
		spinner := components.NewSpinner(w.styles)
		spinner.SetLabel("Loading wallets...")
		return lipgloss.Place(
			w.width,
			w.height,
			lipgloss.Center,
			lipgloss.Center,
			spinner.View(),
		)
	}

	if w.showNewWallet {
		return lipgloss.Place(
			w.width,
			w.height,
			lipgloss.Center,
			lipgloss.Center,
			w.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					w.styles.DialogTitle.Render("Create New Wallet"),
					"",
					w.newWalletForm.View(),
					"",
					w.styles.Error.Render(w.error),
				),
			),
		)
	}

	if w.showImportWallet {
		return lipgloss.Place(
			w.width,
			w.height,
			lipgloss.Center,
			lipgloss.Center,
			w.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					w.styles.DialogTitle.Render("Import Wallet"),
					"",
					w.importWalletForm.View(),
					"",
					w.styles.Error.Render(w.error),
				),
			),
		)
	}

	if w.showExportWallet {
		return lipgloss.Place(
			w.width,
			w.height,
			lipgloss.Center,
			lipgloss.Center,
			w.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					w.styles.DialogTitle.Render(fmt.Sprintf("Export Wallet: %s", w.selectedWallet)),
					"",
					w.exportWalletForm.View(),
					"",
					w.styles.Error.Render(w.error),
				),
			),
		)
	}

	if w.showPasswordForm {
		return lipgloss.Place(
			w.width,
			w.height,
			lipgloss.Center,
			lipgloss.Center,
			w.styles.Panel.Render(
				lipgloss.JoinVertical(
					lipgloss.Left,
					w.styles.DialogTitle.Render(fmt.Sprintf("Change Password: %s", w.selectedWallet)),
					"",
					w.passwordForm.View(),
					"",
					w.styles.Error.Render(w.error),
				),
			),
		)
	}

	if w.showConfirmDelete {
		return lipgloss.Place(
			w.width,
			w.height,
			lipgloss.Center,
			lipgloss.Center,
			w.confirmDeleteModal.View(),
		)
	}

	// Main view
	var sb strings.Builder

	// Title
	sb.WriteString(w.list.View())

	// Help
	help := w.styles.HelpText.Render(fmt.Sprintf(
		"%s: navigate • %s: new • %s: import • %s: export • %s: delete • %s: change password • %s: refresh • %s: back",
		w.styles.KeyBinding.Render("↑/↓"),
		w.styles.KeyBinding.Render("n"),
		w.styles.KeyBinding.Render("i"),
		w.styles.KeyBinding.Render("e"),
		w.styles.KeyBinding.Render("d"),
		w.styles.KeyBinding.Render("p"),
		w.styles.KeyBinding.Render("r"),
		w.styles.KeyBinding.Render("esc"),
	))

	sb.WriteString("\n\n")
	sb.WriteString(help)

	// Error
	if w.error != "" {
		sb.WriteString("\n\n")
		sb.WriteString(w.styles.Error.Render(w.error))
	}

	return sb.String()
}

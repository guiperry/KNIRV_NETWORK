//go:build !wasmloader
// +build !wasmloader

package gui

import (
	"fmt"
	"image/color"
	"log"

	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"KNIRVROUTER_GO_Verifyer/starter"
)

type InstallGUI struct {
	app         fyne.App
	window      fyne.Window
	terminal    *FyneTerminalWidget
	currentStep int
	mu          sync.Mutex

	// UI elements
	rootEndpointEntry *widget.Entry
	desiredURIEntry   *widget.Entry
	progressBar       *widget.ProgressBar
	statusLabel       *widget.Label
	nextButton        *widget.Button
	backButton        *widget.Button
	cancelButton      *widget.Button
	contentContainer  *fyne.Container

	// Installation data
	rootEndpoint string
	desiredURI   string
	chainURI     string
	hashID       string
}

func StartInstallGUI() {
	myApp := app.New()
	myApp.Settings().SetTheme(&KnirvchainTheme{Theme: theme.DarkTheme()})
	window := myApp.NewWindow("KNIRVROUTER Verifier Node Installation")
	window.Resize(fyne.NewSize(800, 600))

	installer := &InstallGUI{
		app:         myApp,
		window:      window,
		terminal:    NewFyneTerminalWidget(),
		currentStep: 0,
	}

	// Create UI elements
	installer.createUI()

	window.SetContent(installer.createMainLayout())
	window.ShowAndRun()
}

func (i *InstallGUI) createUI() {
	// Root endpoint entry
	i.rootEndpointEntry = widget.NewEntry()
	i.rootEndpointEntry.SetText("http://localhost:5000") // Default value
	i.rootEndpointEntry.SetPlaceHolder("Enter KNIRVROUTER Root endpoint")

	// Desired URI entry
	i.desiredURIEntry = widget.NewEntry()
	i.desiredURIEntry.SetPlaceHolder("Enter desired URI (optional)")
	i.desiredURIEntry.SetText("") // Empty by default

	// Progress bar
	i.progressBar = widget.NewProgressBar()
	i.progressBar.Min = 0
	i.progressBar.Max = 6 // Number of steps
	i.progressBar.Value = 0

	// Status label
	i.statusLabel = widget.NewLabel("Welcome to KNIRVROUTER Verifier Node Installation")
	i.statusLabel.Wrapping = fyne.TextWrapWord

	// Navigation buttons
	i.nextButton = widget.NewButton("Next", i.nextStep)
	i.backButton = widget.NewButton("Back", i.previousStep)
	i.cancelButton = widget.NewButton("Cancel", func() {
		i.window.Close()
	})

	// Content container (will be updated per step)
	i.contentContainer = container.NewVBox()
}

func (i *InstallGUI) createMainLayout() fyne.CanvasObject {
	title := canvas.NewText("KNIRVROUTER Verifier Node Installation", color.NRGBA{R: 0, G: 180, B: 255, A: 255})
	title.TextSize = 20
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// Button bar
	buttonBar := container.NewHBox(
		layout.NewSpacer(),
		i.backButton,
		i.nextButton,
		i.cancelButton,
	)

	// Main layout
	return container.NewBorder(
		container.NewVBox(title, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), i.progressBar, buttonBar),
		nil,
		nil,
		container.NewVScroll(container.NewVBox(
			i.statusLabel,
			widget.NewSeparator(),
			i.contentContainer,
			widget.NewSeparator(),
			widget.NewLabel("Installation Log:"),
			i.terminal.GetContent(),
		)),
	)
}

func (i *InstallGUI) nextStep() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.currentStep++
	i.progressBar.SetValue(float64(i.currentStep))
	i.updateStepContent()
}

func (i *InstallGUI) previousStep() {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.currentStep > 0 {
		i.currentStep--
		i.progressBar.SetValue(float64(i.currentStep))
		i.updateStepContent()
	}
}

func (i *InstallGUI) updateStepContent() {
	i.contentContainer.RemoveAll()

	switch i.currentStep {
	case 0:
		i.showWelcomeStep()
	case 1:
		i.showRootEndpointStep()
	case 2:
		i.showGenerateURIStep()
	case 3:
		i.showOSDetectionStep()
	case 4:
		i.showURIHandlerStep()
	case 5:
		i.showConfigurationStep()
	case 6:
		i.showCompletionStep()
	}

	i.updateButtonStates()
}

func (i *InstallGUI) showWelcomeStep() {
	i.statusLabel.SetText("Welcome to KNIRVROUTER Verifier Node Installation")
	i.contentContainer.Add(widget.NewLabel("This installer will guide you through setting up a KNIRVROUTER Verifier Node."))
	i.contentContainer.Add(widget.NewLabel("\nThe installation will:"))
	i.contentContainer.Add(widget.NewLabel("1. Connect to the KNIRVROUTER Root"))
	i.contentContainer.Add(widget.NewLabel("2. Generate a unique chain URI for this verifier"))
	i.contentContainer.Add(widget.NewLabel("3. Detect host operating system"))
	i.contentContainer.Add(widget.NewLabel("4. Register URI handler for knirv:// protocol"))
	i.contentContainer.Add(widget.NewLabel("5. Update the application configuration"))
	i.contentContainer.Add(widget.NewLabel("6. Start the verifier node"))
}

func (i *InstallGUI) showRootEndpointStep() {
	i.statusLabel.SetText("Connect to KNIRVROUTER Root")
	i.contentContainer.Add(widget.NewLabel("Enter the KNIRVROUTER Root endpoint:"))
	i.contentContainer.Add(i.rootEndpointEntry)
	i.contentContainer.Add(widget.NewLabel("\nEnter your desired URI (optional):"))
	i.contentContainer.Add(widget.NewLabel("This will be used to request a specific URI from the server."))
	i.contentContainer.Add(widget.NewLabel("Leave empty for a randomly generated URI."))
	i.contentContainer.Add(i.desiredURIEntry)
}

func (i *InstallGUI) showGenerateURIStep() {
	i.statusLabel.SetText("Generating Chain URI")
	i.contentContainer.Add(widget.NewLabel("Connecting to KNIRVROUTER Root to generate unique chain URI..."))

	go func() {
		i.terminal.Append("Connecting to KNIRVROUTER Root...")

		// Store the desired URI
		i.desiredURI = i.desiredURIEntry.Text

		// Pass both the root endpoint and desired URI
		uri, hashID, err := starter.GenerateChainURI(i.rootEndpointEntry.Text, i.desiredURI)
		if err != nil {
			i.terminal.Append(fmt.Sprintf("Error: %v", err))
			i.updateStatus(fmt.Sprintf("Error: %v", err))
			return
		}

		i.mu.Lock()
		i.chainURI = uri
		i.hashID = hashID
		i.mu.Unlock()

		i.terminal.Append(fmt.Sprintf("Generated chain URI: %s", uri))
		i.terminal.Append(fmt.Sprintf("Transaction HashID: %s", hashID))

		// Update main content area with the generated values
		if ap, ok := i.app.(interface{ CallOnMainThread(func()) }); ok {
			ap.CallOnMainThread(func() {
				i.contentContainer.RemoveAll()
				i.contentContainer.Add(widget.NewLabel("Chain URI generated successfully!"))

				// Make URI more prominent
				uriLabel := widget.NewLabel(fmt.Sprintf("Chain URI: %s", uri))
				uriLabel.TextStyle = fyne.TextStyle{Bold: true}
				uriLabel.Alignment = fyne.TextAlignCenter

				hashLabel := widget.NewLabel(fmt.Sprintf("Transaction HashID: %s", hashID))
				hashLabel.TextStyle = fyne.TextStyle{Italic: true}

				i.contentContainer.Add(uriLabel)
				i.contentContainer.Add(hashLabel)
			})
		}

		i.updateStatus("Chain URI generated successfully")
		i.nextStep()
	}()
}

func (i *InstallGUI) showOSDetectionStep() {
	i.statusLabel.SetText("Operating System Detection")
	i.contentContainer.Add(widget.NewLabel(fmt.Sprintf("Detected operating system: %s", runtime.GOOS)))
}

func (i *InstallGUI) showURIHandlerStep() {
	i.statusLabel.SetText("Registering URI Handlers")
	i.contentContainer.Add(widget.NewLabel("Registering knirv:// protocol handler..."))

	if runtime.GOOS == "windows" && !starter.CheckAdminPrivileges() {
		warning := widget.NewLabel("Warning: Administrator privileges are required on Windows to register URI handlers automatically.")
		warning.Wrapping = fyne.TextWrapWord
		i.contentContainer.Add(warning)
	}

	go func() {
		err := starter.RegisterURIHandlers(i.chainURI)
		if err != nil {
			i.terminal.Append(fmt.Sprintf("Warning: Failed to register URI handlers: %v", err))
			i.updateStatus("URI handler registration failed - may need admin privileges")
		} else {
			i.terminal.Append("URI handlers registered successfully")
			i.updateStatus("URI handlers registered successfully")
		}
		i.nextStep()
	}()
}

func (i *InstallGUI) showConfigurationStep() {
	i.statusLabel.SetText("Updating Configuration")
	i.contentContainer.Add(widget.NewLabel("Updating verifier node configuration..."))

	go func() {
		// Extract the chain ID from the new URI format (knirv://<ID>.chain/)
		var serviceAddress string
		if strings.HasPrefix(i.chainURI, "knirv://") {
			// Extract the ID part from knirv://<ID>.chain/
			parts := strings.Split(strings.TrimPrefix(i.chainURI, "knirv://"), ".")
			if len(parts) > 0 {
				serviceAddress = parts[0]
			}
		} else if strings.HasPrefix(i.chainURI, "chain://") {
			// Handle legacy format for backward compatibility
			serviceAddress = strings.TrimPrefix(i.chainURI, "chain://")
		} else {
			// If the format is unexpected, use the whole URI
			serviceAddress = i.chainURI
		}

		err := starter.UpdateConfiguration(serviceAddress, i.rootEndpointEntry.Text)
		if err != nil {
			i.terminal.Append(fmt.Sprintf("Error updating configuration: %v", err))
			i.updateStatus("Configuration update failed")
			return
		}

		i.terminal.Append("Configuration updated successfully")
		i.updateStatus("Configuration updated successfully")
		i.nextStep()
	}()
}

func (i *InstallGUI) showCompletionStep() {
	i.statusLabel.SetText("Installation Complete")
	i.contentContainer.Add(widget.NewLabel("Your KNIRVROUTER Verifier Node is now configured with a unique chain URI."))
	i.contentContainer.Add(widget.NewLabel(fmt.Sprintf("Chain URI: %s", i.chainURI)))
	i.contentContainer.Add(widget.NewLabel(fmt.Sprintf("Transaction HashID: %s", i.hashID)))
	i.contentContainer.Add(widget.NewLabel("\nClick Finish to launch the Verifyer Application..."))

	// Change next button to "Finish" and make it launch the GUI
	i.nextButton.SetText("Finish")
	i.nextButton.OnTapped = func() {
		// Get working directory

		// Use the LaunchAfterInstall function from starter package
		go func() {
			if err := starter.LaunchAfterInstall(); err != nil {
				log.Printf("Failed to launch application: %v", err)
				if ap, ok := i.app.(interface{ CallOnMainThread(func()) }); ok {
					ap.CallOnMainThread(func() {
						dialog.ShowInformation("Launch Failed",
							"Please manually run the application",
							i.window)
					})
				}
			}
		}()

		// Small delay before closing to ensure process starts
		time.Sleep(500 * time.Millisecond)
		i.window.Close()
	}
	// Force enable the button regardless of step
	i.nextButton.Enable()
	i.updateButtonStates() // Refresh all button states
}

func (i *InstallGUI) updateButtonStates() {
	i.backButton.Disable()
	i.nextButton.Disable()

	if i.currentStep > 0 {
		i.backButton.Enable()
	}

	// Enable next button if not on completion step OR if it's the Finish button
	if i.currentStep < 6 || i.nextButton.Text == "Finish" {
		i.nextButton.Enable()
	}
}

func (i *InstallGUI) updateStatus(status string) {
	if ap, ok := i.app.(interface{ CallOnMainThread(func()) }); ok {
		ap.CallOnMainThread(func() {
			i.statusLabel.SetText(status)
		})
		return
	}
	i.statusLabel.SetText(status)
}

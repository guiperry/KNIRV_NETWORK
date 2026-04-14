package gui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/knirv/network-monitor/internal/config"
	"github.com/knirv/network-monitor/internal/logging"
	"github.com/knirv/network-monitor/internal/monitoring"
	"github.com/sirupsen/logrus"
)

// MainWindow represents the main GUI window
type MainWindow struct {
	app        fyne.App
	window     fyne.Window
	config     *config.Config
	monitor    *monitoring.Monitor
	logManager *logging.Manager
	logger     *logrus.Logger

	// UI Components
	statusCard      *widget.Card
	servicesGrid    *fyne.Container
	metricsChart    *widget.Card
	logsContainer   *container.Scroll
	alertsContainer *fyne.Container

	// Status indicators
	networkStatus  *widget.Label
	servicesStatus *widget.Label
	lastUpdate     *widget.Label

	// Refresh timer
	refreshTimer *time.Ticker
}

// NewMainWindow creates a new main window
func NewMainWindow(app fyne.App, cfg *config.Config, monitor *monitoring.Monitor, logManager *logging.Manager, logger *logrus.Logger) *MainWindow {
	window := app.NewWindow("KNIRV Network Monitor")
	window.SetIcon(theme.ComputerIcon())
	window.Resize(fyne.NewSize(float32(cfg.GUI.WindowSize.Width), float32(cfg.GUI.WindowSize.Height)))
	window.CenterOnScreen()

	mw := &MainWindow{
		app:        app,
		window:     window,
		config:     cfg,
		monitor:    monitor,
		logManager: logManager,
		logger:     logger,
	}

	mw.setupUI()
	mw.startRefreshTimer()

	return mw
}

// setupUI initializes the user interface
func (mw *MainWindow) setupUI() {
	// Create main menu
	mw.setupMenu()

	// Create status overview card
	mw.createStatusCard()

	// Create services grid
	mw.createServicesGrid()

	// Create metrics chart
	mw.createMetricsChart()

	// Create logs viewer
	mw.createLogsViewer()

	// Create alerts panel
	mw.createAlertsPanel()

	// Create main layout
	mainContent := container.NewBorder(
		mw.statusCard, // top
		nil,           // bottom
		nil,           // left
		nil,           // right
		container.NewVSplit(
			container.NewHSplit(
				mw.servicesGrid,
				mw.metricsChart,
			),
			container.NewHSplit(
				mw.logsContainer,
				mw.alertsContainer,
			),
		),
	)

	mw.window.SetContent(mainContent)
}

// setupMenu creates the application menu
func (mw *MainWindow) setupMenu() {
	// File menu
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Settings", mw.showSettings),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Export Logs", mw.exportLogs),
		fyne.NewMenuItem("Export Metrics", mw.exportMetrics),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() { mw.app.Quit() }),
	)

	// View menu
	viewMenu := fyne.NewMenu("View",
		fyne.NewMenuItem("Refresh", mw.refreshData),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Production Network", func() { mw.switchNetwork("production") }),
		fyne.NewMenuItem("Testnet", func() { mw.switchNetwork("testnet") }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Full Screen", func() { mw.window.SetFullScreen(!mw.window.FullScreen()) }),
	)

	// Tools menu
	toolsMenu := fyne.NewMenu("Tools",
		fyne.NewMenuItem("Service Details", mw.showServiceDetails),
		fyne.NewMenuItem("Metrics Dashboard", mw.openMetricsDashboard),
		fyne.NewMenuItem("Log Search", mw.showLogSearch),
		fyne.NewMenuItem("Alert Rules", mw.showAlertRules),
	)

	// Help menu
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", mw.showAbout),
		fyne.NewMenuItem("Documentation", mw.openDocumentation),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, viewMenu, toolsMenu, helpMenu)
	mw.window.SetMainMenu(mainMenu)
}

// createStatusCard creates the network status overview card
func (mw *MainWindow) createStatusCard() {
	network, _ := mw.config.GetActiveNetwork()

	mw.networkStatus = widget.NewLabel("Unknown")
	mw.servicesStatus = widget.NewLabel("0/0")
	mw.lastUpdate = widget.NewLabel("Never")

	statusContent := container.NewGridWithColumns(3,
		container.NewVBox(
			widget.NewLabel("Network Status"),
			mw.networkStatus,
		),
		container.NewVBox(
			widget.NewLabel("Services Up"),
			mw.servicesStatus,
		),
		container.NewVBox(
			widget.NewLabel("Last Update"),
			mw.lastUpdate,
		),
	)

	mw.statusCard = widget.NewCard(
		fmt.Sprintf("KNIRV Network Monitor - %s", network.Name),
		"Real-time network status and health monitoring",
		statusContent,
	)
}

// createServicesGrid creates the services status grid
func (mw *MainWindow) createServicesGrid() {
	mw.servicesGrid = container.NewGridWithColumns(2)
	mw.updateServicesGrid()
}

// createMetricsChart creates the metrics visualization
func (mw *MainWindow) createMetricsChart() {
	// Placeholder for metrics chart
	chartContent := container.NewVBox(
		widget.NewLabel("Real-time Metrics"),
		widget.NewProgressBar(),
		widget.NewLabel("CPU: 45%"),
		widget.NewLabel("Memory: 62%"),
		widget.NewLabel("Network: 1.2 MB/s"),
		widget.NewLabel("IPFS Storage: 15.3 GB"),
	)

	mw.metricsChart = widget.NewCard(
		"System Metrics",
		"Performance and resource utilization",
		chartContent,
	)
}

// createLogsViewer creates the logs viewing panel
func (mw *MainWindow) createLogsViewer() {
	logsContent := container.NewVBox(
		widget.NewLabel("Recent Logs"),
		widget.NewSeparator(),
	)

	mw.logsContainer = container.NewScroll(logsContent)
	mw.logsContainer.SetMinSize(fyne.NewSize(400, 200))
}

// createAlertsPanel creates the alerts management panel
func (mw *MainWindow) createAlertsPanel() {
	mw.alertsContainer = container.NewVBox(
		widget.NewCard(
			"Active Alerts",
			"Current system alerts and notifications",
			widget.NewLabel("No active alerts"),
		),
	)
}

// startRefreshTimer starts the automatic refresh timer
func (mw *MainWindow) startRefreshTimer() {
	mw.refreshTimer = time.NewTicker(mw.config.GUI.RefreshInterval)

	go func() {
		for range mw.refreshTimer.C {
			mw.refreshData()
		}
	}()
}

// refreshData updates all displayed data
func (mw *MainWindow) refreshData() {
	// Update network status
	status := mw.monitor.GetNetworkStatus()

	mw.networkStatus.SetText(status.OverallStatus)
	mw.servicesStatus.SetText(fmt.Sprintf("%d/%d", status.ServicesUp, status.ServicesTotal))
	mw.lastUpdate.SetText(status.LastUpdate.Format("15:04:05"))

	// Update services grid
	mw.updateServicesGrid()

	// Update logs
	mw.updateLogs()

	// Update alerts
	mw.updateAlerts()
}

// updateServicesGrid updates the services status grid
func (mw *MainWindow) updateServicesGrid() {
	mw.servicesGrid.RemoveAll()

	status := mw.monitor.GetNetworkStatus()

	for _, service := range status.Services {
		serviceCard := mw.createServiceCard(service)
		mw.servicesGrid.Add(serviceCard)
	}
}

// createServiceCard creates a card for a single service
func (mw *MainWindow) createServiceCard(service monitoring.ServiceStatus) *widget.Card {
	var statusIcon fyne.Resource

	switch service.Status {
	case "up":
		statusIcon = theme.ConfirmIcon()
	case "down":
		statusIcon = theme.ErrorIcon()
	case "degraded":
		statusIcon = theme.WarningIcon()
	default:
		statusIcon = theme.QuestionIcon()
	}

	statusLabel := widget.NewLabelWithStyle(service.Status, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	content := container.NewVBox(
		container.NewHBox(
			widget.NewIcon(statusIcon),
			statusLabel,
		),
		widget.NewLabel(fmt.Sprintf("Response: %v", service.ResponseTime)),
		widget.NewLabel(fmt.Sprintf("Last Check: %s", service.LastCheck.Format("15:04:05"))),
	)

	if service.Error != "" {
		content.Add(widget.NewLabel(fmt.Sprintf("Error: %s", service.Error)))
	}

	return widget.NewCard(service.Name, service.URL, content)
}

// updateLogs updates the logs display
func (mw *MainWindow) updateLogs() {
	// Placeholder for log updates
	// In a real implementation, this would fetch recent logs from the log manager
}

// updateAlerts updates the alerts display
func (mw *MainWindow) updateAlerts() {
	// Placeholder for alert updates
	// In a real implementation, this would fetch current alerts
}

// ShowAndRun displays the window and runs the application
func (mw *MainWindow) ShowAndRun() {
	mw.window.ShowAndRun()

	// Cleanup
	if mw.refreshTimer != nil {
		mw.refreshTimer.Stop()
	}
}

// Menu action handlers
func (mw *MainWindow) showSettings() {
	// Placeholder for settings dialog
	dialog := widget.NewModalPopUp(
		widget.NewCard("Settings", "Configuration options",
			widget.NewLabel("Settings dialog coming soon...")),
		mw.window.Canvas(),
	)
	dialog.Show()
}

func (mw *MainWindow) exportLogs() {
	// Placeholder for log export
	mw.logger.Info("Export logs requested")
}

func (mw *MainWindow) exportMetrics() {
	// Placeholder for metrics export
	mw.logger.Info("Export metrics requested")
}

func (mw *MainWindow) switchNetwork(network string) {
	mw.config.ActiveNetwork = network
	mw.refreshData()
	mw.logger.Infof("Switched to network: %s", network)
}

func (mw *MainWindow) showServiceDetails() {
	// Placeholder for service details dialog
}

func (mw *MainWindow) openMetricsDashboard() {
	// Placeholder for opening Grafana dashboard
}

func (mw *MainWindow) showLogSearch() {
	// Placeholder for log search dialog
}

func (mw *MainWindow) showAlertRules() {
	// Placeholder for alert rules dialog
}

func (mw *MainWindow) showAbout() {
	about := widget.NewCard(
		"About KNIRV Network Monitor",
		"Version 1.0.0",
		widget.NewLabel("A comprehensive monitoring solution for KNIRV networks.\n\nBuilt with Go and Fyne."),
	)

	dialog := widget.NewModalPopUp(about, mw.window.Canvas())
	dialog.Show()
}

func (mw *MainWindow) openDocumentation() {
	// Placeholder for opening documentation
}

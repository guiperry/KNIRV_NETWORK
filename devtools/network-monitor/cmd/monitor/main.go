package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
	"github.com/knirv/network-monitor/internal/config"
	"github.com/knirv/network-monitor/internal/gui"
	"github.com/knirv/network-monitor/internal/logging"
	"github.com/knirv/network-monitor/internal/monitoring"
	"github.com/sirupsen/logrus"
)

var (
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"
)

// initializeServices sets up and starts the core monitoring and logging services.
func initializeServices(ctx context.Context, cfg *config.Config, logger *logrus.Logger) (*monitoring.Monitor, *logging.Manager, error) {
	// Initialize monitoring system
	monitor, err := monitoring.New(cfg, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize monitoring: %w", err)
	}

	// Start monitoring services
	if err := monitor.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Initialize logging system
	logManager, err := logging.New(cfg, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize logging: %w", err)
	}

	if err := logManager.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start log manager: %w", err)
	}

	return monitor, logManager, nil
}

func main() {
	var (
		configPath  = flag.String("config", "config/config.yaml", "Path to configuration file")
		network     = flag.String("network", "production", "Network to monitor (production/testnet)")
		debug       = flag.Bool("debug", false, "Enable debug logging")
		headless    = flag.Bool("headless", false, "Run without GUI (server mode)")
		port        = flag.Int("port", 8090, "Port for web interface in headless mode")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("KNIRV Network Monitor\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
		os.Exit(0)
	}

	// Setup logging
	logger := logrus.New()
	if *debug {
		logger.SetLevel(logrus.DebugLevel)
	}
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}

	// Override network from command line
	if *network != "" {
		cfg.ActiveNetwork = *network
	}

	logger.Infof("Starting KNIRV Network Monitor v%s", version)
	logger.Infof("Monitoring network: %s", cfg.ActiveNetwork)
	logger.Infof("Debug mode: %v", *debug)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var monitor *monitoring.Monitor
	var logManager *logging.Manager

	if *headless {
		// Run in headless mode with web interface
		logger.Infof("Starting headless mode on port %d", *port)

		monitor, logManager, err = initializeServices(ctx, cfg, logger)
		if err != nil {
			logger.Fatalf("Failed to initialize services: %v", err)
		}

		webServer := gui.NewWebServer(cfg, monitor, logManager, logger)

		go func() {
			if err := webServer.Start(*port); err != nil {
				logger.Errorf("Web server error: %v", err)
			}
		}()

		// Wait for shutdown signal
		<-sigChan
		logger.Info("Received shutdown signal, stopping services...")

	} else {
		// Run with GUI
		logger.Info("Starting GUI application...")

		// Create Fyne application
		myApp := app.NewWithID("com.knirv.network-monitor")
		
		// Set cobalt blue theme
		myApp.Settings().SetTheme(gui.NewCobaltTheme())

		// Show splash screen immediately - this should appear first
		splashWin := showSplash(myApp)

		// Initialize services
		monitor, logManager, err = initializeServices(ctx, cfg, logger)
		if err != nil {
			splashWin.Close()
			logger.Fatalf("Failed to initialize services: %v", err)
		}

		// Initialization complete, hide splash screen
		splashWin.Close()

		// Create main window
		mainWindow := gui.NewMainWindow(myApp, cfg, monitor, logManager, logger)

		// Setup graceful shutdown for GUI
		go func() {
			<-sigChan
			logger.Info("Received shutdown signal, stopping services...")
			cancel()

			// Give services time to shutdown gracefully
			time.Sleep(2 * time.Second)
			myApp.Quit()
		}()

		// Show window and run
		mainWindow.ShowAndRun()
	}

	// Cleanup
	logger.Info("Shutting down monitoring services...")
	if monitor != nil {
		monitor.Stop()
	}
	if logManager != nil {
		logManager.Stop()
	}
	logger.Info("KNIRV Network Monitor stopped")
}

// showSplash creates and shows a splash screen window.
// It returns the window so it can be closed later.
func showSplash(a fyne.App) fyne.Window {
	w := a.NewWindow("KNIRV Network Monitor")
	w.SetContent(widget.NewCard(
		"KNIRV Network Monitor",
		"Initializing monitoring systems...",
		widget.NewProgressBarInfinite(),
	))
	w.CenterOnScreen()
	w.SetFixedSize(true)
	w.Resize(fyne.NewSize(320, 120))
	w.Show()
	return w
}

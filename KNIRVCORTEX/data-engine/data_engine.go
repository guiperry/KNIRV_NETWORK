package dataengine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"backend-server/internal/messages"
)

// DataEngine is the main manager for all data engineering components
type DataEngine struct {
	producer   *EventProducer
	processor  *StreamProcessor
	aggregator *WindowedAggregator
	chromaDB   *ChromaDB
	alerting   *AlertingSystem

	config    DataEngineConfig
	isRunning bool
	ctx       context.Context
	cancel    context.CancelFunc
	mutex     sync.RWMutex

	// Channels for communication with UI
	alertChan   chan Alert
	metricsChan chan *MetricsSnapshot
}

// DataEngineConfig contains configuration for the data engine
type DataEngineConfig struct {
	KafkaBrokers     []string
	KafkaClientID    string
	ChromaDBURL      string
	ChromaCollection string
	EnableKafka      bool
	EnableChromaDB   bool
	EnableWebSocket  bool
	EnableRESTAPI    bool
	WebSocketPort    int
	RESTAPIPort      int
	WindowSize       time.Duration
	MetricsInterval  time.Duration
}

// NewDataEngine creates a new data engine
func NewDataEngine(config DataEngineConfig) *DataEngine {
	ctx, cancel := context.WithCancel(context.Background())

	return &DataEngine{
		config:      config,
		ctx:         ctx,
		cancel:      cancel,
		alertChan:   make(chan Alert, 100),
		metricsChan: make(chan *MetricsSnapshot, 10),
	}
}

// GetAlertChannel returns the alert channel
func (d *DataEngine) GetAlertChannel() <-chan Alert {
	return d.alertChan
}

// GetMetricsChannel returns the metrics channel
func (d *DataEngine) GetMetricsChannel() <-chan *MetricsSnapshot {
	return d.metricsChan
}

// Start starts the data engine
func (d *DataEngine) Start() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.isRunning {
		return fmt.Errorf("data engine is already running")
	}

	// Initialize components
	if d.config.EnableKafka {
		// Create event producer
		d.producer = NewEventProducer(EventProducerConfig{
			KafkaBrokers: d.config.KafkaBrokers,
			ClientID:     d.config.KafkaClientID,
			BatchSize:    100,
			BatchTimeout: 1 * time.Second,
			Async:        true,
		})

		// Connect to Kafka
		err := d.producer.Connect(d.ctx)
		if err != nil {
			return fmt.Errorf("failed to connect to Kafka: %w", err)
		}

		// Create stream processor
		d.processor = NewStreamProcessor(StreamProcessorConfig{
			KafkaBrokers:   d.config.KafkaBrokers,
			ConsumerGroup:  d.config.KafkaClientID + "-consumer",
			Topics:         []string{"KNIRVORACLE-events"},
			BatchSize:      100,
			CommitInterval: 1 * time.Second,
		})

		// Start stream processor
		err = d.processor.Start()
		if err != nil {
			return fmt.Errorf("failed to start stream processor: %w", err)
		}
	}

	// Create windowed aggregator
	d.aggregator = NewWindowedAggregator(SlidingWindow, d.config.WindowSize)

	// Create ChromaDB client
	if d.config.EnableChromaDB {
		d.chromaDB = NewChromaDB(d.config.ChromaDBURL, d.config.ChromaCollection)

		// Connect to ChromaDB
		err := d.chromaDB.Connect(d.ctx)
		if err != nil {
			return fmt.Errorf("failed to connect to ChromaDB: %w", err)
		}
	}

	// Create alerting system
	d.alerting = NewAlertingSystem(1000)

	// Register alert handler
	d.alerting.RegisterHandler(d.handleAlert)

	// Register default alert rules
	d.registerDefaultAlertRules()

	// Note: WebSocket and REST API endpoints are now handled by the unified server
	// Data-engine registers its routes with the main server instead of starting its own servers

	// Start metrics reporting
	go d.reportMetrics()

	d.isRunning = true
	return nil
}

// Stop stops the data engine
func (d *DataEngine) Stop() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if !d.isRunning {
		return nil
	}

	// Cancel context
	d.cancel()

	// Stop components
	if d.producer != nil {
		d.producer.Close()
	}

	if d.processor != nil {
		d.processor.Stop()
	}

	if d.chromaDB != nil {
		d.chromaDB.Close()
	}

	if d.alerting != nil {
		d.alerting.Close()
	}

	// Note: WebSocket and REST API endpoints are handled by the unified server

	d.isRunning = false
	return nil
}

// ProcessEvent processes an event through the data engine
func (d *DataEngine) ProcessEvent(event Event) error {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if !d.isRunning {
		return fmt.Errorf("data engine is not running")
	}

	// Process through windowed aggregator
	err := d.aggregator.ProcessEvent(event)
	if err != nil {
		return fmt.Errorf("failed to process event through aggregator: %w", err)
	}

	// Process through alerting system
	d.alerting.ProcessEvent(event)

	// Send to Kafka if enabled
	if d.config.EnableKafka && d.producer != nil && d.producer.IsConnected() {
		err := d.producer.ProduceEvent(d.ctx, event)
		if err != nil {
			return fmt.Errorf("failed to produce event to Kafka: %w", err)
		}
	}

	// Store in ChromaDB if enabled
	if d.config.EnableChromaDB && d.chromaDB != nil && d.chromaDB.IsConnected() {
		err := d.chromaDB.AddEvent(d.ctx, event)
		if err != nil {
			return fmt.Errorf("failed to add event to ChromaDB: %w", err)
		}
	}

	// Note: WebSocket broadcasting is handled by the unified server

	return nil
}

// ProcessBlockchainEvent processes a blockchain event
func (d *DataEngine) ProcessBlockchainEvent(msg messages.BlockchainEventMsg) error {
	event := ConvertBlockchainEventMsg(msg)
	return d.ProcessEvent(event)
}

// ProcessNetworkUpdate processes a network update
func (d *DataEngine) ProcessNetworkUpdate(msg messages.NetworkUpdateMsg) error {
	event := ConvertNetworkUpdateMsg(msg)
	return d.ProcessEvent(event)
}

// ProcessLogMsg processes a log message
func (d *DataEngine) ProcessLogMsg(msg messages.LogMsg) error {
	event := ConvertLogMsg(msg)
	return d.ProcessEvent(event)
}

// handleAlert handles an alert
func (d *DataEngine) handleAlert(alert Alert) {
	// Send to alert channel
	select {
	case d.alertChan <- alert:
		// Alert sent successfully
	default:
		// Channel is full, log and continue
		fmt.Printf("Alert channel is full, dropping alert: %s\n", alert.Title)
	}
}

// reportMetrics periodically reports metrics
func (d *DataEngine) reportMetrics() {
	ticker := time.NewTicker(d.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			if d.processor != nil {
				// Get metrics from processor
				metrics := d.processor.GetMetrics()

				// Send to metrics channel
				select {
				case d.metricsChan <- metrics:
					// Metrics sent successfully
				default:
					// Channel is full, skip this update
				}
			}
		}
	}
}

// registerDefaultAlertRules registers default alert rules
func (d *DataEngine) registerDefaultAlertRules() {
	// Register error rate alert
	d.alerting.RegisterRule(AlertRule{
		ID:          "error-rate",
		Name:        "High Error Rate",
		Description: "Error rate exceeds threshold",
		EventType:   ErrorEvent,
		Condition:   ErrorRateCondition(10), // 10 errors per minute
		Level:       ErrorAlert,
		Cooldown:    5 * time.Minute,
	})

	// Register transaction spike alert
	d.alerting.RegisterRule(AlertRule{
		ID:          "tx-spike",
		Name:        "Transaction Spike",
		Description: "Unusual spike in transaction volume",
		EventType:   TxSubmittedEvent,
		Condition:   SpikeCondition("count", 200), // 200% increase
		Level:       WarningAlert,
		Cooldown:    10 * time.Minute,
	})

	// Register network latency alert
	d.alerting.RegisterRule(AlertRule{
		ID:          "high-latency",
		Name:        "High Network Latency",
		Description: "Network latency exceeds threshold",
		EventType:   NetworkEventType,
		Condition:   ThresholdCondition("latency_ms", 500, true), // >500ms
		Level:       WarningAlert,
		Cooldown:    5 * time.Minute,
	})

	// Register dev count alert
	d.alerting.RegisterRule(AlertRule{
		ID:          "low-dev-count",
		Name:        "Low Peer Count",
		Description: "Number of connected devs is too low",
		EventType:   NetworkEventType,
		Condition:   ThresholdCondition("dev_count", 3, false), // <3 devs
		Level:       WarningAlert,
		Cooldown:    15 * time.Minute,
	})

	// Register block inactivity alert
	d.alerting.RegisterRule(AlertRule{
		ID:          "block-inactivity",
		Name:        "Block Creation Inactivity",
		Description: "No new blocks have been created recently",
		EventType:   BlockCreatedEvent,
		Condition:   InactivityCondition(BlockCreatedEvent, 300), // 5 minutes
		Level:       WarningAlert,
		Cooldown:    10 * time.Minute,
	})
}

// GetActiveAlerts returns all active alerts
func (d *DataEngine) GetActiveAlerts() []Alert {
	if d.alerting == nil {
		return nil
	}

	return d.alerting.GetActiveAlerts()
}

// ResolveAlert resolves an alert
func (d *DataEngine) ResolveAlert(alertID string) bool {
	if d.alerting == nil {
		return false
	}

	resolved := d.alerting.ResolveAlert(alertID)

	return resolved
}

// IsRunning returns whether the data engine is running
func (d *DataEngine) IsRunning() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.isRunning
}

// GetProducer returns the event producer
func (d *DataEngine) GetProducer() *EventProducer {
	return d.producer
}

// GetChromaDB returns the ChromaDB instance
func (d *DataEngine) GetChromaDB() *ChromaDB {
	return d.chromaDB
}

// GetAlerting returns the alerting system
func (d *DataEngine) GetAlerting() *AlertingSystem {
	return d.alerting
}

// GetAggregator returns the windowed aggregator
func (d *DataEngine) GetAggregator() *WindowedAggregator {
	return d.aggregator
}

// GetMetrics returns the current metrics
func (d *DataEngine) GetMetrics() *MetricsSnapshot {
	if d.processor == nil {
		return nil
	}

	return d.processor.GetMetrics()
}

// Note: WebSocket and REST API methods are handled by the unified server

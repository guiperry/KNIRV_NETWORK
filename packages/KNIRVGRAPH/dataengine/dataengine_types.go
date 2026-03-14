package dataengine

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketServer handles real-time updates (simplified for KNIRVGRAPH)
type WebSocketServer struct {
	clients      map[*websocket.Conn]bool
	clientsMutex sync.RWMutex
	broadcast    chan interface{}
	server       *http.Server
	dataEngine   *DataEngine
	isRunning    bool
	ctx          context.Context
	cancel       context.CancelFunc
	eventLog     []interface{}
	upgrader     websocket.Upgrader
	config       WebSocketConfig
}

// WebSocketConfig contains configuration for the WebSocket server
type WebSocketConfig struct {
	Port            int
	ReadBufferSize  int	
	WriteBufferSize int
	CheckOrigin     bool
}

// DataEngine is the main manager for all data engineering components
type DataEngine struct {
	producer   *EventProducer
	processor  *StreamProcessor
	aggregator *WindowedAggregator
	chromaDB   *ChromaDB
	alerting   *AlertingSystem
	websocket  *WebSocketServer
	restAPI    *RESTAPIServer

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

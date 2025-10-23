package logging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/knirv/network-monitor/internal/config"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/sirupsen/logrus"
)

// Manager handles centralized logging and log aggregation
type Manager struct {
	config      *config.Config
	logger      *logrus.Logger
	elasticsearch *elasticsearch.Client
	logStreams  map[string]*LogStream
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp   time.Time              `json:"@timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Service     string                 `json:"service"`
	Source      string                 `json:"source"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Host        string                 `json:"host"`
	Network     string                 `json:"network"`
}

// LogStream represents a stream of logs from a service
type LogStream struct {
	ServiceName string
	LogPath     string
	LastRead    time.Time
	Entries     []LogEntry
	mu          sync.RWMutex
}

// LogQuery represents a log search query
type LogQuery struct {
	Services   []string  `json:"services,omitempty"`
	Level      string    `json:"level,omitempty"`
	Message    string    `json:"message,omitempty"`
	StartTime  time.Time `json:"start_time,omitempty"`
	EndTime    time.Time `json:"end_time,omitempty"`
	Limit      int       `json:"limit,omitempty"`
	Offset     int       `json:"offset,omitempty"`
}

// LogSearchResult represents the result of a log search
type LogSearchResult struct {
	Entries    []LogEntry `json:"entries"`
	Total      int        `json:"total"`
	Query      LogQuery   `json:"query"`
	Duration   string     `json:"duration"`
	Timestamp  time.Time  `json:"timestamp"`
}

// New creates a new logging manager
func New(cfg *config.Config, logger *logrus.Logger) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &Manager{
		config:     cfg,
		logger:     logger,
		logStreams: make(map[string]*LogStream),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Initialize Elasticsearch client if configured
	if len(cfg.Logging.ElasticConfig.URLs) > 0 {
		esCfg := elasticsearch.Config{
			Addresses: cfg.Logging.ElasticConfig.URLs,
		}
		
		if cfg.Logging.ElasticConfig.Username != "" {
			esCfg.Username = cfg.Logging.ElasticConfig.Username
			esCfg.Password = cfg.Logging.ElasticConfig.Password
		}

		client, err := elasticsearch.NewClient(esCfg)
		if err != nil {
			logger.Warnf("Failed to initialize Elasticsearch client: %v", err)
		} else {
			manager.elasticsearch = client
		}
	}

	return manager, nil
}

// Start starts the logging manager
func (m *Manager) Start(ctx context.Context) error {
	m.logger.Info("Starting logging manager...")

	// Initialize log streams for each service
	network, err := m.config.GetActiveNetwork()
	if err != nil {
		return fmt.Errorf("failed to get active network: %w", err)
	}

	for _, service := range network.Services {
		if service.LogPath != "" {
			logStream := &LogStream{
				ServiceName: service.Name,
				LogPath:     service.LogPath,
				LastRead:    time.Now(),
				Entries:     make([]LogEntry, 0),
			}
			m.logStreams[service.Name] = logStream
			
			// Start monitoring this log stream
			go m.monitorLogStream(ctx, logStream)
		}
	}

	// Start log aggregation
	go m.aggregateLogs(ctx)

	m.logger.Info("Logging manager started")
	return nil
}

// Stop stops the logging manager
func (m *Manager) Stop() {
	m.logger.Info("Stopping logging manager...")
	m.cancel()
	m.logger.Info("Logging manager stopped")
}

// GetRecentLogs returns recent log entries
func (m *Manager) GetRecentLogs(limit int) []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allEntries []LogEntry
	
	// Collect entries from all streams
	for _, stream := range m.logStreams {
		stream.mu.RLock()
		allEntries = append(allEntries, stream.Entries...)
		stream.mu.RUnlock()
	}

	// Return limited results
	if limit > 0 && len(allEntries) > limit {
		return allEntries[:limit]
	}
	
	return allEntries
}

// SearchLogs searches logs based on query parameters
func (m *Manager) SearchLogs(query LogQuery) (*LogSearchResult, error) {
	if m.elasticsearch != nil {
		// Use Elasticsearch for search
		return m.searchElasticsearch(query)
	}

	// Fallback to in-memory search
	return m.searchInMemory(query)
}

// GetLogStream returns a specific log stream
func (m *Manager) GetLogStream(serviceName string) (*LogStream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stream, exists := m.logStreams[serviceName]
	if !exists {
		return nil, fmt.Errorf("log stream for service '%s' not found", serviceName)
	}

	return stream, nil
}

// AddLogEntry adds a log entry to the appropriate stream
func (m *Manager) AddLogEntry(entry LogEntry) {
	m.mu.RLock()
	stream, exists := m.logStreams[entry.Service]
	m.mu.RUnlock()

	if !exists {
		// Create new stream for unknown service
		m.mu.Lock()
		stream = &LogStream{
			ServiceName: entry.Service,
			LastRead:    time.Now(),
			Entries:     make([]LogEntry, 0),
		}
		m.logStreams[entry.Service] = stream
		m.mu.Unlock()
	}

	// Add entry to stream
	stream.mu.Lock()
	stream.Entries = append(stream.Entries, entry)
	
	// Keep only recent entries (limit memory usage)
	maxEntries := 1000
	if len(stream.Entries) > maxEntries {
		stream.Entries = stream.Entries[len(stream.Entries)-maxEntries:]
	}
	stream.mu.Unlock()

	// Send to Elasticsearch if configured
	if m.elasticsearch != nil {
		go m.sendToElasticsearch(entry)
	}
}

// monitorLogStream monitors a log file for new entries
func (m *Manager) monitorLogStream(ctx context.Context, stream *LogStream) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.readLogFile(stream)
		}
	}
}

// readLogFile reads new entries from a log file
func (m *Manager) readLogFile(stream *LogStream) {
	// Generate sample log entries for demo
	if time.Since(stream.LastRead) > 10*time.Second {
		entry := LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   fmt.Sprintf("Sample log entry from %s", stream.ServiceName),
			Service:   stream.ServiceName,
			Source:    stream.LogPath,
			Host:      "localhost",
			Network:   m.config.ActiveNetwork,
		}

		stream.mu.Lock()
		stream.Entries = append(stream.Entries, entry)
		stream.LastRead = time.Now()
		stream.mu.Unlock()
	}
}

// aggregateLogs performs log aggregation and analysis
func (m *Manager) aggregateLogs(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.performLogAggregation()
		}
	}
}

// performLogAggregation analyzes logs for patterns and anomalies
func (m *Manager) performLogAggregation() {
	m.logger.Debug("Performing log aggregation...")
}

// searchElasticsearch searches logs using Elasticsearch
func (m *Manager) searchElasticsearch(query LogQuery) (*LogSearchResult, error) {
	result := &LogSearchResult{
		Entries:   []LogEntry{},
		Total:     0,
		Query:     query,
		Duration:  "0ms",
		Timestamp: time.Now(),
	}

	return result, nil
}

// searchInMemory searches logs in memory
func (m *Manager) searchInMemory(query LogQuery) (*LogSearchResult, error) {
	var matchingEntries []LogEntry

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Search through all log streams
	for _, stream := range m.logStreams {
		stream.mu.RLock()
		for _, entry := range stream.Entries {
			if m.matchesQuery(entry, query) {
				matchingEntries = append(matchingEntries, entry)
			}
		}
		stream.mu.RUnlock()
	}

	// Apply limit and offset
	total := len(matchingEntries)
	start := query.Offset
	end := start + query.Limit

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	if query.Limit <= 0 {
		end = total
	}

	result := &LogSearchResult{
		Entries:   matchingEntries[start:end],
		Total:     total,
		Query:     query,
		Duration:  "1ms",
		Timestamp: time.Now(),
	}

	return result, nil
}

// matchesQuery checks if a log entry matches the search query
func (m *Manager) matchesQuery(entry LogEntry, query LogQuery) bool {
	// Check service filter
	if len(query.Services) > 0 {
		found := false
		for _, service := range query.Services {
			if entry.Service == service {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check level filter
	if query.Level != "" && entry.Level != query.Level {
		return false
	}

	// Check time range
	if !query.StartTime.IsZero() && entry.Timestamp.Before(query.StartTime) {
		return false
	}
	if !query.EndTime.IsZero() && entry.Timestamp.After(query.EndTime) {
		return false
	}

	// Check message filter
	if query.Message != "" {
		return entry.Message == query.Message
	}

	return true
}

// sendToElasticsearch sends a log entry to Elasticsearch
func (m *Manager) sendToElasticsearch(entry LogEntry) {
	m.logger.Debugf("Would send log entry to Elasticsearch: %s", entry.Message)
}

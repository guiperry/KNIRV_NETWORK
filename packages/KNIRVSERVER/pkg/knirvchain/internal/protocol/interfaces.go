package protocol

import (
	"context"
	"net/http"
	"time"
)

// ProtocolAdapter defines the interface for protocol adaptation
type ProtocolAdapter interface {
	// Protocol operations
	AdaptRequest(request *ProtocolRequest) (*AdaptedRequest, error)
	AdaptResponse(response *ProtocolResponse) (*AdaptedResponse, error)
	
	// Protocol conversion
	ConvertToProtocol(data interface{}, targetProtocol string) (interface{}, error)
	ConvertFromProtocol(data interface{}, sourceProtocol string) (interface{}, error)
	
	// Protocol validation
	ValidateProtocol(data interface{}, protocol string) error
	GetSupportedProtocols() []string
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
}

// ProtocolConverter defines the interface for protocol conversion
type ProtocolConverter interface {
	// Conversion operations
	Convert(input *ConversionInput) (*ConversionOutput, error)
	ConvertBatch(inputs []*ConversionInput) ([]*ConversionOutput, error)
	
	// Format operations
	RegisterFormat(format *ProtocolFormat) error
	UnregisterFormat(formatName string) error
	GetFormat(formatName string) (*ProtocolFormat, error)
	ListFormats() ([]*ProtocolFormat, error)
	
	// Validation
	ValidateFormat(data interface{}, formatName string) error
	DetectFormat(data interface{}) (string, error)
}

// RelayManager defines the interface for message relay operations
type RelayManager interface {
	// Relay operations
	RelayMessage(message *RelayMessage) error
	RelayBatch(messages []*RelayMessage) error
	
	// Route management
	AddRoute(route *RelayRoute) error
	RemoveRoute(routeID string) error
	GetRoute(routeID string) (*RelayRoute, error)
	ListRoutes() ([]*RelayRoute, error)
	
	// Relay monitoring
	GetRelayStats() (*RelayStats, error)
	GetRouteStats(routeID string) (*RouteStats, error)
	
	// Event handling
	OnMessageRelayed(handler RelayHandler) error
	OnRelayError(handler ErrorHandler) error
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
}

// ProxyManager defines the interface for reverse proxy operations
type ProxyManager interface {
	// Proxy operations
	HandleRequest(request *http.Request) (*http.Response, error)
	
	// Target management
	AddTarget(target *ProxyTarget) error
	RemoveTarget(targetID string) error
	GetTarget(targetID string) (*ProxyTarget, error)
	ListTargets() ([]*ProxyTarget, error)
	
	// Load balancing
	SetLoadBalancer(balancer LoadBalancer) error
	GetLoadBalancer() LoadBalancer
	
	// Health checking
	CheckTargetHealth(targetID string) (*HealthStatus, error)
	GetHealthyTargets() ([]*ProxyTarget, error)
	
	// Middleware
	AddMiddleware(middleware Middleware) error
	RemoveMiddleware(middlewareID string) error
	
	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
}

// ProtocolRequest represents a protocol request
type ProtocolRequest struct {
	ID        string                 `json:"id"`
	Protocol  string                 `json:"protocol"`
	Method    string                 `json:"method"`
	Path      string                 `json:"path"`
	Headers   map[string]string      `json:"headers"`
	Body      []byte                 `json:"body"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ProtocolResponse represents a protocol response
type ProtocolResponse struct {
	ID        string                 `json:"id"`
	RequestID string                 `json:"request_id"`
	Protocol  string                 `json:"protocol"`
	Status    int                    `json:"status"`
	Headers   map[string]string      `json:"headers"`
	Body      []byte                 `json:"body"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// AdaptedRequest represents an adapted request
type AdaptedRequest struct {
	OriginalRequest *ProtocolRequest       `json:"original_request"`
	AdaptedData     interface{}            `json:"adapted_data"`
	TargetProtocol  string                 `json:"target_protocol"`
	Transformations []string               `json:"transformations"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// AdaptedResponse represents an adapted response
type AdaptedResponse struct {
	OriginalResponse *ProtocolResponse      `json:"original_response"`
	AdaptedData      interface{}            `json:"adapted_data"`
	TargetProtocol   string                 `json:"target_protocol"`
	Transformations  []string               `json:"transformations"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ConversionInput represents input for protocol conversion
type ConversionInput struct {
	Data           interface{}            `json:"data"`
	SourceFormat   string                 `json:"source_format"`
	TargetFormat   string                 `json:"target_format"`
	Options        map[string]interface{} `json:"options,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ConversionOutput represents output from protocol conversion
type ConversionOutput struct {
	Data         interface{}            `json:"data"`
	SourceFormat string                 `json:"source_format"`
	TargetFormat string                 `json:"target_format"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ProtocolFormat represents a protocol format definition
type ProtocolFormat struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	MimeType    string                 `json:"mime_type"`
	Schema      interface{}            `json:"schema,omitempty"`
	Validator   FormatValidator        `json:"-"`
	Converter   FormatConverter        `json:"-"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RelayMessage represents a message to be relayed
type RelayMessage struct {
	ID          string                 `json:"id"`
	Source      string                 `json:"source"`
	Destination string                 `json:"destination"`
	Type        string                 `json:"type"`
	Payload     []byte                 `json:"payload"`
	Headers     map[string]string      `json:"headers"`
	Priority    int                    `json:"priority"`
	TTL         time.Duration          `json:"ttl"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// RelayRoute represents a relay route
type RelayRoute struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Source      string                 `json:"source"`
	Destination string                 `json:"destination"`
	Conditions  []RouteCondition       `json:"conditions"`
	Transform   *MessageTransform      `json:"transform,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// RouteCondition represents a condition for routing
type RouteCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// MessageTransform represents a message transformation
type MessageTransform struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// RelayStats represents relay statistics
type RelayStats struct {
	TotalMessages    int64         `json:"total_messages"`
	SuccessfulRelays int64         `json:"successful_relays"`
	FailedRelays     int64         `json:"failed_relays"`
	AverageLatency   time.Duration `json:"average_latency"`
	ActiveRoutes     int           `json:"active_routes"`
	LastRelay        time.Time     `json:"last_relay"`
}

// RouteStats represents statistics for a specific route
type RouteStats struct {
	RouteID         string        `json:"route_id"`
	MessageCount    int64         `json:"message_count"`
	SuccessCount    int64         `json:"success_count"`
	ErrorCount      int64         `json:"error_count"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastMessage     time.Time     `json:"last_message"`
}

// ProxyTarget represents a proxy target
type ProxyTarget struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	URL      string            `json:"url"`
	Weight   int               `json:"weight"`
	Enabled  bool              `json:"enabled"`
	Headers  map[string]string `json:"headers,omitempty"`
	Timeout  time.Duration     `json:"timeout"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// HealthStatus represents the health status of a target
type HealthStatus struct {
	TargetID    string        `json:"target_id"`
	Healthy     bool          `json:"healthy"`
	LastCheck   time.Time     `json:"last_check"`
	ResponseTime time.Duration `json:"response_time"`
	Error       string        `json:"error,omitempty"`
}

// LoadBalancer defines the interface for load balancing
type LoadBalancer interface {
	SelectTarget(targets []*ProxyTarget, request *http.Request) (*ProxyTarget, error)
	GetAlgorithm() string
	Configure(config map[string]interface{}) error
}

// Middleware defines the interface for proxy middleware
type Middleware interface {
	Process(request *http.Request, response *http.Response) error
	GetID() string
	GetPriority() int
}

// FormatValidator defines the interface for format validation
type FormatValidator interface {
	Validate(data interface{}) error
}

// FormatConverter defines the interface for format conversion
type FormatConverter interface {
	Convert(data interface{}, targetFormat string) (interface{}, error)
}

// ProtocolEvent represents protocol events
type ProtocolEvent struct {
	Type      string                 `json:"type"`
	Protocol  string                 `json:"protocol"`
	Operation string                 `json:"operation"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ProtocolMetrics represents protocol metrics
type ProtocolMetrics struct {
	RequestCount      int64         `json:"request_count"`
	ResponseCount     int64         `json:"response_count"`
	ErrorCount        int64         `json:"error_count"`
	AverageLatency    time.Duration `json:"average_latency"`
	ConversionCount   int64         `json:"conversion_count"`
	RelayCount        int64         `json:"relay_count"`
	ProxyCount        int64         `json:"proxy_count"`
	LastActivity      time.Time     `json:"last_activity"`
}

// ProtocolConfig represents protocol configuration
type ProtocolConfig struct {
	DefaultProtocol   string                 `json:"default_protocol"`
	SupportedProtocols []string              `json:"supported_protocols"`
	ConversionEnabled bool                   `json:"conversion_enabled"`
	RelayEnabled      bool                   `json:"relay_enabled"`
	ProxyEnabled      bool                   `json:"proxy_enabled"`
	MaxMessageSize    int64                  `json:"max_message_size"`
	Timeout           time.Duration          `json:"timeout"`
	RetryAttempts     int                    `json:"retry_attempts"`
	Options           map[string]interface{} `json:"options,omitempty"`
}

// Event handler function types
type RelayHandler func(message *RelayMessage) error
type ErrorHandler func(error) error
type EventHandler func(event *ProtocolEvent) error

// Error types for protocol operations
var (
	ErrUnsupportedProtocol = NewProtocolError("unsupported protocol")
	ErrInvalidFormat       = NewProtocolError("invalid format")
	ErrConversionFailed    = NewProtocolError("conversion failed")
	ErrRelayFailed         = NewProtocolError("relay failed")
	ErrProxyFailed         = NewProtocolError("proxy failed")
	ErrTargetUnhealthy     = NewProtocolError("target unhealthy")
	ErrRouteNotFound       = NewProtocolError("route not found")
)

// ProtocolError represents a protocol-specific error
type ProtocolError struct {
	Message string
	Code    string
}

func (e *ProtocolError) Error() string {
	return e.Message
}

func NewProtocolError(message string) *ProtocolError {
	return &ProtocolError{Message: message}
}

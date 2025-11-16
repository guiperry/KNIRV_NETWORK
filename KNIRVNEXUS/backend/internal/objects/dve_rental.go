package objects

import (
	"time"
)

// DVERental represents a DVE rental transaction
type DVERental struct {
	ID                  string         `json:"id"`
	UserID              string         `json:"user_id"`
	DVENodeID           string         `json:"dve_node_id"`
	NRNAmount           int64          `json:"nrn_amount"`         // Amount of NRN tokens paid
	RentalDuration      int64          `json:"rental_duration"`    // Duration in seconds
	StartTime           time.Time      `json:"start_time"`
	EndTime             time.Time      `json:"end_time"`
	Status              string         `json:"status"`             // "active", "expired", "cancelled"
	PaymentTxHash       string         `json:"payment_tx_hash"`    // Transaction hash for NRN payment
	CDEEnvironmentID    string         `json:"cde_environment_id"` // Associated CDE environment
	ResourceLimits      ResourceLimits `json:"resource_limits"`
	UsageMetrics        UsageMetrics   `json:"usage_metrics"`
	AutoRenewalEnabled  bool           `json:"auto_renewal_enabled"`  // Whether automatic renewal is enabled
	RenewalPaymentTxHash string        `json:"renewal_payment_tx_hash,omitempty"` // Transaction hash for renewal payment
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`

	// ⭐ NEW CONTAINER AND SESSION FIELDS
	ContainerID         string         `json:"container_id"`         // ID of provisioned container
	SSHUsername         string         `json:"ssh_username"`         // SSH username for container access
	SSHPort             int            `json:"ssh_port"`             // SSH port for container access
	SSHSessionID        string         `json:"ssh_session_id"`       // ID of SSH session
	AccessToken         string         `json:"access_token"`         // JWT access token for API access
	ValidationSessionID string         `json:"validation_session_id"` // ID of reasoning validation session
	ErrorResSessionID   string         `json:"error_resolution_session_id"` // ID of error resolution session
	ProvisionedAt       time.Time      `json:"provisioned_at"`       // When container was provisioned
	ProvisioningStatus  string         `json:"provisioning_status"`  // "pending", "provisioned", "failed"

	// ⭐ NEW PAYMENT FIELDS
	PaymentMethodID     string         `json:"payment_method_id"`    // Link to PaymentMethod
	PaymentProvider     string         `json:"payment_provider"`     // "stripe" or "paypal"
	PaymentAmount       int64          `json:"payment_amount"`       // Amount in cents
	PaymentCurrency     string         `json:"payment_currency"`     // "USD", "EUR"
	PaymentStatus       string         `json:"payment_status"`       // "pending", "completed", "failed"
	StripeChargeID      string         `json:"stripe_charge_id,omitempty"`
	PayPalOrderID       string         `json:"paypal_order_id,omitempty"`
	PayPalCaptureID     string         `json:"paypal_capture_id,omitempty"`
	ReceiptURL          string         `json:"receipt_url,omitempty"`
	InvoiceID           string         `json:"invoice_id,omitempty"`
	RefundRequested     bool           `json:"refund_requested"`
	RefundedAmount      int64          `json:"refunded_amount"`      // Amount refunded in cents
	PaymentFailureReason string        `json:"payment_failure_reason,omitempty"`
}

// ResourceLimits defines the resource limits for a rented DVE
type ResourceLimits struct {
	MaxCPU       float64 `json:"max_cpu"`       // CPU cores
	MaxMemory    int64   `json:"max_memory"`    // Memory in bytes
	MaxDisk      int64   `json:"max_disk"`      // Disk space in bytes
	MaxBandwidth int64   `json:"max_bandwidth"` // Network bandwidth in bytes/sec
}

// UsageMetrics tracks actual resource usage during rental
type UsageMetrics struct {
	CPUUsage     float64   `json:"cpu_usage"`     // Current CPU usage percentage
	MemoryUsage  int64     `json:"memory_usage"`  // Current memory usage in bytes
	DiskUsage    int64     `json:"disk_usage"`    // Current disk usage in bytes
	NetworkUsage int64     `json:"network_usage"` // Network usage in bytes
	LastUpdated  time.Time `json:"last_updated"`
}

// NRNPayment represents an NRN token payment for DVE rental
type NRNPayment struct {
	ID            string     `json:"id"`
	RentalID      string     `json:"rental_id"`
	UserID        string     `json:"user_id"`
	Amount        int64      `json:"amount"`        // Amount in NRN tokens
	TxHash        string     `json:"tx_hash"`       // Blockchain transaction hash
	Status        string     `json:"status"`        // "pending", "confirmed", "failed"
	BlockHeight   int64      `json:"block_height"`  // Block height when confirmed
	Confirmations int        `json:"confirmations"` // Number of confirmations
	CreatedAt     time.Time  `json:"created_at"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
}

// RentalPlan defines different rental pricing tiers
type RentalPlan struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	PricePerHour   int64          `json:"price_per_hour"` // NRN tokens per hour
	ResourceLimits ResourceLimits `json:"resource_limits"`
	MaxDuration    int64          `json:"max_duration"` // Maximum rental duration in seconds
	MinDuration    int64          `json:"min_duration"` // Minimum rental duration in seconds
	Features       []string       `json:"features"`     // List of included features
	IsActive       bool           `json:"is_active"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// DVERentalStats provides statistics for the rental system
type DVERentalStats struct {
	TotalRentals      int64       `json:"total_rentals"`
	ActiveRentals     int64       `json:"active_rentals"`
	TotalNRNCollected int64       `json:"total_nrn_collected"`
	AverageRentalTime int64       `json:"average_rental_time"`
	PopularPlans      []PlanUsage `json:"popular_plans"`
	RevenueToday      int64       `json:"revenue_today"`
	Revenue7Days      int64       `json:"revenue_7days"`
	Revenue30Days     int64       `json:"revenue_30days"`
	Timestamp         time.Time   `json:"timestamp"`
}

// PlanUsage tracks usage statistics for rental plans
type PlanUsage struct {
	PlanID     string  `json:"plan_id"`
	PlanName   string  `json:"plan_name"`
	UsageCount int64   `json:"usage_count"`
	Percentage float64 `json:"percentage"`
}

// RentalRequest represents a request to rent a DVE
type RentalRequest struct {
	UserID             string `json:"user_id"`
	PlanID             string `json:"plan_id"`
	Duration           int64  `json:"duration"`                // Duration in seconds
	PaymentTxHash      string `json:"payment_tx_hash"`         // NRN payment transaction hash
	PreferredDVE       string `json:"preferred_dve,omitempty"` // Optional preferred DVE node
	AutoRenewalEnabled bool   `json:"auto_renewal_enabled"`    // Whether to enable automatic renewal
}

// RentalResponse represents the response to a rental request
type RentalResponse struct {
	Success        bool           `json:"success"`
	RentalID       string         `json:"rental_id,omitempty"`
	DVENodeID      string         `json:"dve_node_id,omitempty"`
	CDEAccessURL   string         `json:"cde_access_url,omitempty"`
	CDECredentials CDECredentials `json:"cde_credentials,omitempty"`
	ExpiresAt      time.Time      `json:"expires_at,omitempty"`
	Error          string         `json:"error,omitempty"`
	Message        string         `json:"message,omitempty"`
}

// CDECredentials provides access credentials for the provisioned CDE
type CDECredentials struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	SSHKey      string `json:"ssh_key,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
}

// ⭐ NEW SESSION AND ENDPOINT OBJECTS

// SSHSession represents an SSH session to a DVE container
type SSHSession struct {
	ID              string    `json:"id"`
	RentalID        string    `json:"rental_id"`
	ContainerID     string    `json:"container_id"`
	Username        string    `json:"username"`
	PublicKeyHash   string    `json:"public_key_hash"`
	PrivateKeyURL   string    `json:"private_key_url"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
	LastUsed        time.Time `json:"last_used"`
}

// ValidationSession represents a reasoning validation session
type ValidationSession struct {
	ID              string    `json:"id"`
	RentalID        string    `json:"rental_id"`
	SessionToken    string    `json:"session_token"`
	EndpointURL     string    `json:"endpoint_url"`
	Port            int       `json:"port"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
	ValidationType  string    `json:"validation_type"`
}

// ErrorResolutionSession for error resolution access
type ErrorResolutionSession struct {
	ID              string    `json:"id"`
	RentalID        string    `json:"rental_id"`
	SessionToken    string    `json:"session_token"`
	EndpointURL     string    `json:"endpoint_url"`
	Port            int       `json:"port"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
	SupportedTypes  []string  `json:"supported_error_types"`
}

// TEEEndpoint represents a TEE endpoint (SSH, validation, error resolution)
type TEEEndpoint struct {
	ID              string    `json:"id"`
	RentalID        string    `json:"rental_id"`
	ContainerID     string    `json:"container_id"`
	EndpointType    string    `json:"endpoint_type"` // "ssh", "validation", "error-resolution"
	Host            string    `json:"host"`
	Port            int       `json:"port"`
	Protocol        string    `json:"protocol"` // "ssh", "http", "https", "ws"
	Credentials     Credentials `json:"credentials"`
	Status          string    `json:"status"` // "active", "inactive", "terminated"
	CreatedAt       time.Time `json:"created_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// Credentials contains authentication details
type Credentials struct {
	Username        string    `json:"username,omitempty"`
	PrivateKey      string    `json:"private_key,omitempty"` // Only in single-use responses
	Token           string    `json:"token,omitempty"`
	KeyFingerprint  string    `json:"key_fingerprint,omitempty"`
}

// ⭐ NEW PAYMENT OBJECTS

// PaymentMethod represents payment configuration and transaction details
type PaymentMethod struct {
	ID              string    `json:"id"`
	RentalID        string    `json:"rental_id"`
	UserID          string    `json:"user_id"`
	Provider        string    `json:"provider"` // "stripe" or "paypal"
	Status          string    `json:"status"` // "pending", "completed", "failed", "refunded"
	Amount          int64     `json:"amount"` // in cents
	Currency        string    `json:"currency"` // "USD", "EUR", etc.
	ProviderRefID   string    `json:"provider_ref_id"` // Stripe charge ID or PayPal transaction ID
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"created_at"`
	CompletedAt     time.Time `json:"completed_at"`
	FailureReason   string    `json:"failure_reason,omitempty"`
}

// PaymentTransaction represents a financial transaction record
type PaymentTransaction struct {
	ID              string    `json:"id"`
	PaymentMethodID string    `json:"payment_method_id"`
	RentalID        string    `json:"rental_id"`
	UserID          string    `json:"user_id"`
	Amount          int64     `json:"amount"` // in cents
	Currency        string    `json:"currency"`
	Status          string    `json:"status"` // "initiated", "processing", "completed", "failed"
	Provider        string    `json:"provider"` // "stripe" or "paypal"
	ProviderTxID    string    `json:"provider_tx_id"`
	WebhookReceived bool      `json:"webhook_received"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ReceiptURL      string    `json:"receipt_url,omitempty"`
}

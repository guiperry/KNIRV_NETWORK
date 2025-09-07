package models

import (
	"time"
)

// DVERental represents a DVE rental transaction
type DVERental struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	DVENodeID         string    `json:"dve_node_id"`
	NRNAmount         int64     `json:"nrn_amount"`         // Amount of NRN tokens paid
	RentalDuration    int64     `json:"rental_duration"`    // Duration in seconds
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Status            string    `json:"status"`             // "active", "expired", "cancelled"
	PaymentTxHash     string    `json:"payment_tx_hash"`    // Transaction hash for NRN payment
	CDEEnvironmentID  string    `json:"cde_environment_id"` // Associated CDE environment
	ResourceLimits    ResourceLimits `json:"resource_limits"`
	UsageMetrics      UsageMetrics   `json:"usage_metrics"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ResourceLimits defines the resource limits for a rented DVE
type ResourceLimits struct {
	MaxCPU    float64 `json:"max_cpu"`    // CPU cores
	MaxMemory int64   `json:"max_memory"` // Memory in bytes
	MaxDisk   int64   `json:"max_disk"`   // Disk space in bytes
	MaxBandwidth int64 `json:"max_bandwidth"` // Network bandwidth in bytes/sec
}

// UsageMetrics tracks actual resource usage during rental
type UsageMetrics struct {
	CPUUsage      float64 `json:"cpu_usage"`      // Current CPU usage percentage
	MemoryUsage   int64   `json:"memory_usage"`   // Current memory usage in bytes
	DiskUsage     int64   `json:"disk_usage"`     // Current disk usage in bytes
	NetworkUsage  int64   `json:"network_usage"`  // Network usage in bytes
	LastUpdated   time.Time `json:"last_updated"`
}

// NRNPayment represents an NRN token payment for DVE rental
type NRNPayment struct {
	ID            string    `json:"id"`
	RentalID      string    `json:"rental_id"`
	UserID        string    `json:"user_id"`
	Amount        int64     `json:"amount"`        // Amount in NRN tokens
	TxHash        string    `json:"tx_hash"`       // Blockchain transaction hash
	Status        string    `json:"status"`        // "pending", "confirmed", "failed"
	BlockHeight   int64     `json:"block_height"`  // Block height when confirmed
	Confirmations int       `json:"confirmations"` // Number of confirmations
	CreatedAt     time.Time `json:"created_at"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
}

// RentalPlan defines different rental pricing tiers
type RentalPlan struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	PricePerHour    int64          `json:"price_per_hour"`    // NRN tokens per hour
	ResourceLimits  ResourceLimits `json:"resource_limits"`
	MaxDuration     int64          `json:"max_duration"`      // Maximum rental duration in seconds
	MinDuration     int64          `json:"min_duration"`      // Minimum rental duration in seconds
	Features        []string       `json:"features"`          // List of included features
	IsActive        bool           `json:"is_active"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// DVERentalStats provides statistics for the rental system
type DVERentalStats struct {
	TotalRentals      int64   `json:"total_rentals"`
	ActiveRentals     int64   `json:"active_rentals"`
	TotalNRNCollected int64   `json:"total_nrn_collected"`
	AverageRentalTime int64   `json:"average_rental_time"`
	PopularPlans      []PlanUsage `json:"popular_plans"`
	RevenueToday      int64   `json:"revenue_today"`
	Revenue7Days      int64   `json:"revenue_7days"`
	Revenue30Days     int64   `json:"revenue_30days"`
	Timestamp         time.Time `json:"timestamp"`
}

// PlanUsage tracks usage statistics for rental plans
type PlanUsage struct {
	PlanID      string `json:"plan_id"`
	PlanName    string `json:"plan_name"`
	UsageCount  int64  `json:"usage_count"`
	Percentage  float64 `json:"percentage"`
}

// RentalRequest represents a request to rent a DVE
type RentalRequest struct {
	UserID         string `json:"user_id"`
	PlanID         string `json:"plan_id"`
	Duration       int64  `json:"duration"`        // Duration in seconds
	PaymentTxHash  string `json:"payment_tx_hash"` // NRN payment transaction hash
	PreferredDVE   string `json:"preferred_dve,omitempty"` // Optional preferred DVE node
}

// RentalResponse represents the response to a rental request
type RentalResponse struct {
	Success       bool      `json:"success"`
	RentalID      string    `json:"rental_id,omitempty"`
	DVENodeID     string    `json:"dve_node_id,omitempty"`
	CDEAccessURL  string    `json:"cde_access_url,omitempty"`
	CDECredentials CDECredentials `json:"cde_credentials,omitempty"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	Error         string    `json:"error,omitempty"`
	Message       string    `json:"message,omitempty"`
}

// CDECredentials provides access credentials for the provisioned CDE
type CDECredentials struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	SSHKey      string `json:"ssh_key,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
}

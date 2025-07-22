package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AIAgent struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Name            string    `gorm:"not null" json:"name"`
	Description     string    `json:"description"`
	Category        string    `gorm:"not null" json:"category"` // defi, trading, nft, analytics
	Version         string    `gorm:"not null" json:"version"`
	Status          string    `gorm:"default:'inactive'" json:"status"` // active, inactive, installing, error
	CodeHash        string    `gorm:"not null" json:"code_hash"`
	EncryptedCode   []byte    `gorm:"not null" json:"-"`
	Permissions     []string  `gorm:"type:text[]" json:"permissions"`
	Configuration   string    `gorm:"type:jsonb" json:"configuration"`
	Performance     decimal.Decimal `gorm:"type:decimal(10,4);default:0" json:"performance"`
	RiskLevel       string    `gorm:"default:'medium'" json:"risk_level"` // low, medium, high
	IsPublic        bool      `gorm:"default:false" json:"is_public"`
	InstallCount    int       `gorm:"default:0" json:"install_count"`
	Rating          decimal.Decimal `gorm:"type:decimal(3,2);default:0" json:"rating"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relationships
	User           User                `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Executions     []AgentExecution    `gorm:"foreignKey:AgentID" json:"executions,omitempty"`
	Trades         []AgentTrade        `gorm:"foreignKey:AgentID" json:"trades,omitempty"`
	Reviews        []AgentReview       `gorm:"foreignKey:AgentID" json:"reviews,omitempty"`
}

type AgentExecution struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	AgentID     uuid.UUID `gorm:"type:uuid;not null" json:"agent_id"`
	Status      string    `gorm:"not null" json:"status"` // running, completed, failed, timeout
	StartTime   time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	Duration    int64     `json:"duration_ms"`
	MemoryUsed  int64     `json:"memory_used"`
	CPUUsed     float64   `json:"cpu_used"`
	Input       string    `gorm:"type:jsonb" json:"input"`
	Output      string    `gorm:"type:jsonb" json:"output"`
	ErrorMessage string   `json:"error_message"`
	CreatedAt   time.Time `json:"created_at"`

	Agent AIAgent `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
}

type AgentTrade struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	AgentID         uuid.UUID       `gorm:"type:uuid;not null" json:"agent_id"`
	TransactionID   uuid.UUID       `gorm:"type:uuid" json:"transaction_id"`
	Type            string          `gorm:"not null" json:"type"` // buy, sell, swap
	FromSymbol      string          `json:"from_symbol"`
	ToSymbol        string          `json:"to_symbol"`
	Amount          decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"amount"`
	Price           decimal.Decimal `gorm:"type:decimal(36,18)" json:"price"`
	USDValue        decimal.Decimal `gorm:"type:decimal(36,18)" json:"usd_value"`
	Profit          decimal.Decimal `gorm:"type:decimal(36,18)" json:"profit"`
	Status          string          `gorm:"not null" json:"status"` // pending, executed, failed
	Reason          string          `json:"reason"`
	ExecutedAt      *time.Time      `json:"executed_at"`
	CreatedAt       time.Time       `json:"created_at"`

	Agent       AIAgent      `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	Transaction *Transaction `gorm:"foreignKey:TransactionID" json:"transaction,omitempty"`
}

type AgentReview struct {
	ID        uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	AgentID   uuid.UUID       `gorm:"type:uuid;not null" json:"agent_id"`
	UserID    uuid.UUID       `gorm:"type:uuid;not null" json:"user_id"`
	Rating    decimal.Decimal `gorm:"type:decimal(3,2);not null" json:"rating"`
	Comment   string          `json:"comment"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`

	Agent AIAgent `gorm:"foreignKey:AgentID" json:"agent,omitempty"`
	User  User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type AgentPermission struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RiskLevel   string `json:"risk_level"`
}

var DefaultPermissions = []AgentPermission{
	{Name: "read_portfolio", Description: "Read portfolio balance and holdings", RiskLevel: "low"},
	{Name: "read_market_data", Description: "Access real-time market data", RiskLevel: "low"},
	{Name: "execute_trades", Description: "Execute buy/sell orders", RiskLevel: "high"},
	{Name: "access_defi", Description: "Interact with DeFi protocols", RiskLevel: "high"},
	{Name: "send_notifications", Description: "Send push notifications", RiskLevel: "low"},
	{Name: "network_access", Description: "Make external API calls", RiskLevel: "medium"},
	{Name: "storage_access", Description: "Store and retrieve data", RiskLevel: "medium"},
}
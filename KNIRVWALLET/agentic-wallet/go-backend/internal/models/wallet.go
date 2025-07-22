package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Wallet struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Name          string    `gorm:"not null" json:"name"`
	Network       string    `gorm:"not null" json:"network"` // ethereum, bitcoin, solana
	Address       string    `gorm:"not null" json:"address"`
	EncryptedPrivateKey string `gorm:"not null" json:"-"`
	PublicKey     string    `json:"public_key"`
	IsHardware    bool      `gorm:"default:false" json:"is_hardware"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Relationships
	User         User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Assets       []Asset       `gorm:"foreignKey:WalletID" json:"assets,omitempty"`
	Transactions []Transaction `gorm:"foreignKey:WalletID" json:"transactions,omitempty"`
}

type Asset struct {
	ID            uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	WalletID      uuid.UUID       `gorm:"type:uuid;not null" json:"wallet_id"`
	Symbol        string          `gorm:"not null" json:"symbol"`
	Name          string          `gorm:"not null" json:"name"`
	ContractAddress string        `json:"contract_address"`
	Balance       decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"balance"`
	USDValue      decimal.Decimal `gorm:"type:decimal(36,18)" json:"usd_value"`
	LastUpdated   time.Time       `json:"last_updated"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`

	Wallet Wallet `gorm:"foreignKey:WalletID" json:"wallet,omitempty"`
}

type Transaction struct {
	ID              uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	WalletID        uuid.UUID       `gorm:"type:uuid;not null" json:"wallet_id"`
	Hash            string          `gorm:"uniqueIndex;not null" json:"hash"`
	Type            string          `gorm:"not null" json:"type"` // send, receive, swap, stake
	Status          string          `gorm:"not null" json:"status"` // pending, confirmed, failed
	FromAddress     string          `json:"from_address"`
	ToAddress       string          `json:"to_address"`
	Amount          decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"amount"`
	Symbol          string          `gorm:"not null" json:"symbol"`
	USDValue        decimal.Decimal `gorm:"type:decimal(36,18)" json:"usd_value"`
	GasFee          decimal.Decimal `gorm:"type:decimal(36,18)" json:"gas_fee"`
	BlockNumber     int64           `json:"block_number"`
	Confirmations   int             `json:"confirmations"`
	Timestamp       time.Time       `json:"timestamp"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`

	Wallet Wallet `gorm:"foreignKey:WalletID" json:"wallet,omitempty"`
}

type PriceData struct {
	ID          uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Symbol      string          `gorm:"uniqueIndex;not null" json:"symbol"`
	Name        string          `gorm:"not null" json:"name"`
	Price       decimal.Decimal `gorm:"type:decimal(36,18);not null" json:"price"`
	Change24h   decimal.Decimal `gorm:"type:decimal(10,4)" json:"change_24h"`
	Volume24h   decimal.Decimal `gorm:"type:decimal(36,18)" json:"volume_24h"`
	MarketCap   decimal.Decimal `gorm:"type:decimal(36,18)" json:"market_cap"`
	LastUpdated time.Time       `json:"last_updated"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}
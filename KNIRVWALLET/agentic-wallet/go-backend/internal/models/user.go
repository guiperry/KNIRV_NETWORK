package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email             string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash      string    `gorm:"not null" json:"-"`
	FirstName         string    `json:"first_name"`
	LastName          string    `json:"last_name"`
	IsEmailVerified   bool      `gorm:"default:false" json:"is_email_verified"`
	IsTwoFactorEnabled bool     `gorm:"default:false" json:"is_two_factor_enabled"`
	TwoFactorSecret   string    `json:"-"`
	BiometricEnabled  bool      `gorm:"default:false" json:"biometric_enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Wallets   []Wallet   `gorm:"foreignKey:UserID" json:"wallets,omitempty"`
	AIAgents  []AIAgent  `gorm:"foreignKey:UserID" json:"ai_agents,omitempty"`
	Sessions  []Session  `gorm:"foreignKey:UserID" json:"-"`
}

type Session struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Token     string    `gorm:"uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type UserPreferences struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID            uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	Currency          string    `gorm:"default:'USD'" json:"currency"`
	Language          string    `gorm:"default:'en'" json:"language"`
	Theme             string    `gorm:"default:'dark'" json:"theme"`
	NotificationsEnabled bool   `gorm:"default:true" json:"notifications_enabled"`
	PriceAlertsEnabled bool    `gorm:"default:true" json:"price_alerts_enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
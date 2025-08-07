package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	Name      string    `json:"name" gorm:"not null"`
	Email     string    `json:"email" gorm:"unique;not null"`
	Wallet    *Wallet   `json:"wallet" gorm:"foreignKey:UserID"` // 👈 Use pointer here
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Wallet represents a user's wallet
type Wallet struct {
	ID        uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:char(36);not null;unique"`
	Balance   int64     `json:"balance" gorm:"default:0"` // Store in smallest currency unit (cents)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      *User     `json:"user" gorm:"foreignKey:UserID"` // 👈 Use pointer here
}

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeCredit   TransactionType = "credit"
	TransactionTypeDebit    TransactionType = "debit"
	TransactionTypeTransfer TransactionType = "transfer"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
)

// Transaction represents a transaction record
type Transaction struct {
	ID          uuid.UUID         `json:"id" gorm:"type:char(36);primary_key"`
	UserID      uuid.UUID         `json:"user_id" gorm:"type:char(36);not null"`
	Type        TransactionType   `json:"type" gorm:"not null"`
	Amount      int64             `json:"amount" gorm:"not null"` // Store in smallest currency unit
	Description string            `json:"description"`
	Status      TransactionStatus `json:"status" gorm:"default:'pending'"`
	Reference   string            `json:"reference" gorm:"unique;not null"` // For idempotency
	FromUserID  *uuid.UUID        `json:"from_user_id,omitempty" gorm:"type:char(36)"`
	ToUserID    *uuid.UUID        `json:"to_user_id,omitempty" gorm:"type:char(36)"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	User        User              `json:"user" gorm:"foreignKey:UserID"`
	FromUser    *User             `json:"from_user,omitempty" gorm:"foreignKey:FromUserID"`
	ToUser      *User             `json:"to_user,omitempty" gorm:"foreignKey:ToUserID"`
}

// BeforeCreate hook for User model
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for Wallet model
func (w *Wallet) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for Transaction model
func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

// ReconciliationResult represents the result of a reconciliation process
type ReconciliationResult struct {
	UserID            uuid.UUID `json:"user_id"`
	StoredBalance     int64     `json:"stored_balance"`
	CalculatedBalance int64     `json:"calculated_balance"`
	Difference        int64     `json:"difference"`
	HasMismatch       bool      `json:"has_mismatch"`
	CheckedAt         time.Time `json:"checked_at"`
}

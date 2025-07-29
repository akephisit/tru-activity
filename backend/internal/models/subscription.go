package models

import (
	"time"

	"gorm.io/gorm"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusExpired  SubscriptionStatus = "expired"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
)

type SubscriptionType string

const (
	SubscriptionTypeBasic   SubscriptionType = "basic"
	SubscriptionTypePremium SubscriptionType = "premium"
	SubscriptionTypeVIP     SubscriptionType = "vip"
)

type Subscription struct {
	ID        uint               `json:"id" gorm:"primaryKey"`
	UserID    uint               `json:"user_id"`
	User      User               `json:"user"`
	Type      SubscriptionType   `json:"type" gorm:"type:varchar(20);not null"`
	Status    SubscriptionStatus `json:"status" gorm:"type:varchar(20);default:'active'"`
	StartDate time.Time          `json:"start_date"`
	EndDate   time.Time          `json:"end_date"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	DeletedAt gorm.DeletedAt     `json:"deleted_at" gorm:"index"`
}

// IsExpired checks if the subscription has expired
func (s *Subscription) IsExpired() bool {
	return time.Now().After(s.EndDate)
}

// DaysUntilExpiry returns the number of days until expiry
func (s *Subscription) DaysUntilExpiry() int {
	if s.IsExpired() {
		return 0
	}
	return int(s.EndDate.Sub(time.Now()).Hours() / 24)
}
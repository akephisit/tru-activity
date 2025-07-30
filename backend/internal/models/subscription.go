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
	SubscriptionTypeBasic      SubscriptionType = "basic"
	SubscriptionTypePremium    SubscriptionType = "premium"
	SubscriptionTypeEnterprise SubscriptionType = "enterprise"
)

type Subscription struct {
	ID                    uint               `json:"id" gorm:"primaryKey"`
	FacultyID             uint               `json:"faculty_id" gorm:"not null"`
	Faculty               Faculty            `json:"faculty"`
	Type                  SubscriptionType   `json:"type" gorm:"type:varchar(20);not null"`
	Status                SubscriptionStatus `json:"status" gorm:"type:varchar(20);default:'active'"`
	StartDate             time.Time          `json:"start_date"`
	EndDate               time.Time          `json:"end_date"`
	NotificationSent7Days bool               `json:"notification_sent_7_days" gorm:"default:false"`
	NotificationSent1Day  bool               `json:"notification_sent_1_day" gorm:"default:false"`
	LastNotificationAt    *time.Time         `json:"last_notification_at"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`
	DeletedAt             gorm.DeletedAt     `json:"deleted_at" gorm:"index"`
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

// NeedsNotification checks if subscription needs notification (7 days or 1 day before expiry)
func (s *Subscription) NeedsNotification() bool {
	daysLeft := s.DaysUntilExpiry()
	
	// Check if needs 7-day notification
	if daysLeft <= 7 && daysLeft > 1 && !s.NotificationSent7Days {
		return true
	}
	
	// Check if needs 1-day notification
	if daysLeft <= 1 && daysLeft > 0 && !s.NotificationSent1Day {
		return true
	}
	
	return false
}

// GetNotificationType returns which type of notification is needed
func (s *Subscription) GetNotificationType() string {
	daysLeft := s.DaysUntilExpiry()
	
	if daysLeft <= 1 && daysLeft > 0 && !s.NotificationSent1Day {
		return "1_day"
	}
	
	if daysLeft <= 7 && daysLeft > 1 && !s.NotificationSent7Days {
		return "7_days"
	}
	
	return ""
}

// IsSubscriptionData implements the GraphQL union interface for Subscription
func (s *Subscription) IsSubscriptionData() {}
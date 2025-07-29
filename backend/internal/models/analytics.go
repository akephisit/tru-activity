package models

import (
	"time"

	"gorm.io/gorm"
)

type FacultyMetrics struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	FacultyID           uint      `json:"faculty_id" gorm:"not null"`
	Faculty             Faculty   `json:"faculty"`
	TotalStudents       int       `json:"total_students"`
	ActiveStudents      int       `json:"active_students"`
	TotalActivities     int       `json:"total_activities"`
	CompletedActivities int       `json:"completed_activities"`
	TotalParticipants   int       `json:"total_participants"`
	AverageAttendance   float64   `json:"average_attendance"`
	Date                time.Time `json:"date" gorm:"type:date"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type SystemMetrics struct {
	ID                    uint           `json:"id" gorm:"primaryKey"`
	TotalFaculties        int            `json:"total_faculties"`
	TotalDepartments      int            `json:"total_departments"`
	TotalStudents         int            `json:"total_students"`
	TotalActivities       int            `json:"total_activities"`
	TotalParticipations   int            `json:"total_participations"`
	ActiveSubscriptions   int            `json:"active_subscriptions"`
	ExpiredSubscriptions  int            `json:"expired_subscriptions"`
	Date                  time.Time      `json:"date" gorm:"type:date"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type NotificationLog struct {
	ID               uint                 `json:"id" gorm:"primaryKey"`
	SubscriptionID   uint                 `json:"subscription_id" gorm:"not null"`
	Subscription     Subscription         `json:"subscription"`
	Type             NotificationType     `json:"type" gorm:"type:varchar(20);not null"`
	Status           NotificationStatus   `json:"status" gorm:"type:varchar(20);default:'pending'"`
	Email            string               `json:"email" gorm:"size:255"`
	Subject          string               `json:"subject" gorm:"size:255"`
	Message          string               `json:"message" gorm:"type:text"`
	SentAt           *time.Time           `json:"sent_at"`
	ErrorMessage     string               `json:"error_message" gorm:"type:text"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
	DeletedAt        gorm.DeletedAt       `json:"deleted_at" gorm:"index"`
}

type NotificationType string

const (
	NotificationTypeExpiry7Days NotificationType = "expiry_7_days"
	NotificationTypeExpiry1Day  NotificationType = "expiry_1_day"
	NotificationTypeExpired     NotificationType = "expired"
)

type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)
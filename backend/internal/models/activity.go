package models

import (
	"time"

	"gorm.io/gorm"
)

type ActivityStatus string

const (
	ActivityStatusDraft     ActivityStatus = "draft"
	ActivityStatusActive    ActivityStatus = "active"
	ActivityStatusCompleted ActivityStatus = "completed"
	ActivityStatusCancelled ActivityStatus = "cancelled"
)

type ActivityType string

const (
	ActivityTypeWorkshop   ActivityType = "workshop"
	ActivityTypeSeminar    ActivityType = "seminar"
	ActivityTypeCompetition ActivityType = "competition"
	ActivityTypeVolunteer  ActivityType = "volunteer"
	ActivityTypeOther      ActivityType = "other"
)

type Activity struct {
	ID             uint             `json:"id" gorm:"primaryKey"`
	Title          string           `json:"title" gorm:"size:200;not null"`
	Description    string           `json:"description" gorm:"type:text"`
	Type           ActivityType     `json:"type" gorm:"type:varchar(20);not null"`
	Status         ActivityStatus   `json:"status" gorm:"type:varchar(20);default:'draft'"`
	StartDate      time.Time        `json:"start_date"`
	EndDate        time.Time        `json:"end_date"`
	Location       string           `json:"location" gorm:"size:200"`
	MaxParticipants *int            `json:"max_participants"`
	RequireApproval bool            `json:"require_approval" gorm:"default:false"`
	Points         int              `json:"points" gorm:"default:0"`
	FacultyID      *uint            `json:"faculty_id"`
	Faculty        *Faculty         `json:"faculty,omitempty"`
	DepartmentID   *uint            `json:"department_id"`
	Department     *Department      `json:"department,omitempty"`
	CreatedByID    uint             `json:"created_by_id"`
	CreatedBy      User             `json:"created_by"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	DeletedAt      gorm.DeletedAt   `json:"deleted_at" gorm:"index"`

	// Associations
	Participations []Participation `json:"participations"`
}

type ParticipationStatus string

const (
	ParticipationStatusPending   ParticipationStatus = "pending"
	ParticipationStatusApproved  ParticipationStatus = "approved"
	ParticipationStatusRejected  ParticipationStatus = "rejected"
	ParticipationStatusAttended  ParticipationStatus = "attended"
	ParticipationStatusAbsent    ParticipationStatus = "absent"
)

type Participation struct {
	ID           uint                `json:"id" gorm:"primaryKey"`
	UserID       uint                `json:"user_id"`
	User         User                `json:"user"`
	ActivityID   uint                `json:"activity_id"`
	Activity     Activity            `json:"activity"`
	Status       ParticipationStatus `json:"status" gorm:"type:varchar(20);default:'pending'"`
	RegisteredAt time.Time           `json:"registered_at"`
	ApprovedAt   *time.Time          `json:"approved_at"`
	AttendedAt   *time.Time          `json:"attended_at"`
	Notes        string              `json:"notes" gorm:"type:text"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}
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
	ID               uint             `json:"id" gorm:"primaryKey"`
	Title            string           `json:"title" gorm:"size:200;not null"`
	Description      string           `json:"description" gorm:"type:text"`
	Type             ActivityType     `json:"type" gorm:"type:varchar(20);not null"`
	Status           ActivityStatus   `json:"status" gorm:"type:varchar(20);default:'draft'"`
	StartDate        time.Time        `json:"start_date"`
	EndDate          time.Time        `json:"end_date"`
	Location         string           `json:"location" gorm:"size:200"`
	MaxParticipants  *int             `json:"max_participants"`
	RequireApproval  bool             `json:"require_approval" gorm:"default:false"`
	Points           int              `json:"points" gorm:"default:0"`
	FacultyID        *uint            `json:"faculty_id"`
	Faculty          *Faculty         `json:"faculty,omitempty"`
	DepartmentID     *uint            `json:"department_id"`
	Department       *Department      `json:"department,omitempty"`
	CreatedByID      uint             `json:"created_by_id"`
	CreatedBy        User             `json:"created_by"`
	TemplateID       *uint            `json:"template_id"`
	Template         *ActivityTemplate `json:"template,omitempty"`
	IsRecurring      bool             `json:"is_recurring" gorm:"default:false"`
	RecurrenceRule   string           `json:"recurrence_rule" gorm:"type:text"`
	ParentActivityID *uint            `json:"parent_activity_id"`
	ParentActivity   *Activity        `json:"parent_activity,omitempty"`
	QRCodeRequired   bool             `json:"qr_code_required" gorm:"default:true"`
	AutoApprove      bool             `json:"auto_approve" gorm:"default:false"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	DeletedAt        gorm.DeletedAt   `json:"deleted_at" gorm:"index"`

	// Associations
	Participations   []Participation   `json:"participations"`
	Assignments      []ActivityAssignment `json:"assignments"`
	ChildActivities  []Activity        `json:"child_activities" gorm:"foreignKey:ParentActivityID"`
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
	QRScannedAt  *time.Time          `json:"qr_scanned_at"`
	ScannedByID  *uint               `json:"scanned_by_id"`
	ScannedBy    *User               `json:"scanned_by,omitempty"`
	ScanLocation string              `json:"scan_location" gorm:"size:200"`
	Notes        string              `json:"notes" gorm:"type:text"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

// ActivityTemplate represents reusable activity templates
type ActivityTemplate struct {
	ID              uint             `json:"id" gorm:"primaryKey"`
	Name            string           `json:"name" gorm:"size:200;not null"`
	Description     string           `json:"description" gorm:"type:text"`
	Type            ActivityType     `json:"type" gorm:"type:varchar(20);not null"`
	DefaultDuration int              `json:"default_duration"` // in minutes
	Location        string           `json:"location" gorm:"size:200"`
	MaxParticipants *int             `json:"max_participants"`
	RequireApproval bool             `json:"require_approval" gorm:"default:false"`
	Points          int              `json:"points" gorm:"default:0"`
	QRCodeRequired  bool             `json:"qr_code_required" gorm:"default:true"`
	AutoApprove     bool             `json:"auto_approve" gorm:"default:false"`
	FacultyID       *uint            `json:"faculty_id"`
	Faculty         *Faculty         `json:"faculty,omitempty"`
	CreatedByID     uint             `json:"created_by_id"`
	CreatedBy       User             `json:"created_by"`
	IsActive        bool             `json:"is_active" gorm:"default:true"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DeletedAt       gorm.DeletedAt   `json:"deleted_at" gorm:"index"`

	// Associations
	Activities []Activity `json:"activities"`
}

// ActivityAssignment represents assignment of activities to regular admins
type ActivityAssignment struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	ActivityID uint           `json:"activity_id"`
	Activity   Activity       `json:"activity"`
	AdminID    uint           `json:"admin_id"`
	Admin      User           `json:"admin"`
	AssignedByID uint         `json:"assigned_by_id"`
	AssignedBy User           `json:"assigned_by"`
	CanScanQR  bool           `json:"can_scan_qr" gorm:"default:true"`
	CanApprove bool           `json:"can_approve" gorm:"default:true"`
	Notes      string         `json:"notes" gorm:"type:text"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// QRScanLog tracks all QR code scan attempts
type QRScanLog struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	StudentID      string         `json:"student_id" gorm:"size:20;not null"`
	UserID         *uint          `json:"user_id"`
	User           *User          `json:"user,omitempty"`
	ActivityID     uint           `json:"activity_id"`
	Activity       Activity       `json:"activity"`
	ScannedByID    uint           `json:"scanned_by_id"`
	ScannedBy      User           `json:"scanned_by"`
	ScanTimestamp  time.Time      `json:"scan_timestamp"`
	QRTimestamp    time.Time      `json:"qr_timestamp"`
	Valid          bool           `json:"valid"`
	ErrorMessage   string         `json:"error_message" gorm:"type:text"`
	ScanLocation   string         `json:"scan_location" gorm:"size:200"`
	IPAddress      string         `json:"ip_address" gorm:"size:45"`
	UserAgent      string         `json:"user_agent" gorm:"type:text"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}
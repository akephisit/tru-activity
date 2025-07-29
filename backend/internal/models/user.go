package models

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

const (
	UserRoleStudent      UserRole = "student"
	UserRoleSuperAdmin   UserRole = "super_admin"
	UserRoleFacultyAdmin UserRole = "faculty_admin"
	UserRoleRegularAdmin UserRole = "regular_admin"
)

type User struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	StudentID      string         `json:"student_id" gorm:"uniqueIndex;size:20"`
	Email          string         `json:"email" gorm:"uniqueIndex;size:100"`
	FirstName      string         `json:"first_name" gorm:"size:50;not null"`
	LastName       string         `json:"last_name" gorm:"size:50;not null"`
	Password       string         `json:"-" gorm:"not null"`
	Role           UserRole       `json:"role" gorm:"type:varchar(20);default:'student'"`
	QRSecret       string         `json:"qr_secret" gorm:"size:32;not null"`
	FacultyID      *uint          `json:"faculty_id"`
	Faculty        *Faculty       `json:"faculty,omitempty"`
	DepartmentID   *uint          `json:"department_id"`
	Department     *Department    `json:"department,omitempty"`
	IsActive       bool           `json:"is_active" gorm:"default:true"`
	LastLoginAt    *time.Time     `json:"last_login_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Associations
	Participations []Participation `json:"participations"`
	Subscriptions  []Subscription  `json:"subscriptions"`
}

func (u *User) IsAdmin() bool {
	return u.Role == UserRoleSuperAdmin || u.Role == UserRoleFacultyAdmin || u.Role == UserRoleRegularAdmin
}

func (u *User) CanManageFaculty(facultyID uint) bool {
	if u.Role == UserRoleSuperAdmin {
		return true
	}
	if u.Role == UserRoleFacultyAdmin && u.FacultyID != nil && *u.FacultyID == facultyID {
		return true
	}
	return false
}
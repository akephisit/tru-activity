package models

import (
	"time"

	"gorm.io/gorm"
)

type Faculty struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:100;not null"`
	Code        string         `json:"code" gorm:"uniqueIndex;size:10;not null"`
	Description string         `json:"description" gorm:"type:text"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Associations
	Departments []Department `json:"departments"`
	Users       []User       `json:"users"`
	Activities  []Activity   `json:"activities"`
}

type Department struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"size:100;not null"`
	Code      string         `json:"code" gorm:"size:10;not null"`
	FacultyID uint           `json:"faculty_id" gorm:"not null"`
	Faculty   Faculty        `json:"faculty"`
	IsActive  bool           `json:"is_active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Associations
	Users      []User     `json:"users"`
	Activities []Activity `json:"activities"`
}

// Add unique index for department code within faculty
func (Department) TableName() string {
	return "departments"
}
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

func (u *User) CanManageDepartment(departmentID uint) bool {
	if u.Role == UserRoleSuperAdmin {
		return true
	}
	if u.Role == UserRoleFacultyAdmin && u.FacultyID != nil {
		return true // Faculty admin สามารถจัดการ department ในคณะตัวเองได้
	}
	if u.Role == UserRoleRegularAdmin && u.DepartmentID != nil && *u.DepartmentID == departmentID {
		return true // Regular admin สามารถดู department ตัวเองได้
	}
	return false
}

func (u *User) CanViewUser(targetUser *User) bool {
	if u.Role == UserRoleSuperAdmin {
		return true
	}
	if u.Role == UserRoleFacultyAdmin {
		// Faculty admin สามารถดู user ในคณะเดียวกัน
		return u.FacultyID != nil && targetUser.FacultyID != nil && *u.FacultyID == *targetUser.FacultyID
	}
	if u.Role == UserRoleRegularAdmin {
		// Regular admin สามารถดู student ในคณะเดียวกัน
		return targetUser.Role == UserRoleStudent &&
			u.FacultyID != nil && targetUser.FacultyID != nil && 
			*u.FacultyID == *targetUser.FacultyID
	}
	// Student สามารถดูตัวเองได้เท่านั้น
	return u.ID == targetUser.ID
}

func (u *User) CanCreateActivity() bool {
	return u.Role == UserRoleSuperAdmin || u.Role == UserRoleFacultyAdmin
}

func (u *User) CanManageActivity(activity *Activity) bool {
	if u.Role == UserRoleSuperAdmin {
		return true
	}
	if u.Role == UserRoleFacultyAdmin {
		// Faculty admin สามารถจัดการกิจกรรมในคณะตัวเอง หรือกิจกรรมที่ตนสร้าง
		if u.ID == activity.CreatedByID {
			return true
		}
		if activity.FacultyID != nil && u.FacultyID != nil && *activity.FacultyID == *u.FacultyID {
			return true
		}
	}
	// เจ้าของกิจกรรมสามารถจัดการได้
	return u.ID == activity.CreatedByID
}

func (u *User) CanApproveParticipation(participation *Participation) bool {
	if u.Role == UserRoleSuperAdmin {
		return true
	}
	if u.Role == UserRoleFacultyAdmin || u.Role == UserRoleRegularAdmin {
		// Admin สามารถ approve participation ได้
		return true
	}
	return false
}

func (u *User) CanMarkAttendance() bool {
	return u.Role == UserRoleSuperAdmin || u.Role == UserRoleFacultyAdmin || u.Role == UserRoleRegularAdmin
}

func (u *User) GetAccessibleFacultyIDs() []uint {
	if u.Role == UserRoleSuperAdmin {
		return []uint{} // Empty slice หมายถึงทุก faculty
	}
	if u.FacultyID != nil {
		return []uint{*u.FacultyID}
	}
	return []uint{}
}

func (u *User) IsInSameFaculty(otherUser *User) bool {
	if u.FacultyID == nil || otherUser.FacultyID == nil {
		return false
	}
	return *u.FacultyID == *otherUser.FacultyID
}

func (u *User) HasHigherRoleThan(otherUser *User) bool {
	roleHierarchy := map[UserRole]int{
		UserRoleStudent:      1,
		UserRoleRegularAdmin: 2,
		UserRoleFacultyAdmin: 3,
		UserRoleSuperAdmin:   4,
	}
	
	currentLevel, exists1 := roleHierarchy[u.Role]
	otherLevel, exists2 := roleHierarchy[otherUser.Role]
	
	if !exists1 || !exists2 {
		return false
	}
	
	return currentLevel > otherLevel
}

// IsSubscriptionData implements the GraphQL union interface
func (u *User) IsSubscriptionData() {}
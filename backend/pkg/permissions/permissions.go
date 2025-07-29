package permissions

import (
	"github.com/kruakemaths/tru-activity/backend/internal/models"
)

type Permission string

const (
	// Faculty Management Permissions
	PermCreateFaculty Permission = "create_faculty"
	PermUpdateFaculty Permission = "update_faculty"
	PermDeleteFaculty Permission = "delete_faculty"
	PermViewAllFaculties Permission = "view_all_faculties"

	// Department Management Permissions
	PermCreateDepartment Permission = "create_department"
	PermUpdateDepartment Permission = "update_department"
	PermDeleteDepartment Permission = "delete_department"
	PermViewDepartments Permission = "view_departments"

	// User Management Permissions
	PermCreateUser Permission = "create_user"
	PermUpdateUser Permission = "update_user"
	PermDeleteUser Permission = "delete_user"
	PermViewUsers Permission = "view_users"
	PermViewUserDetails Permission = "view_user_details"

	// Activity Management Permissions
	PermCreateActivity Permission = "create_activity"
	PermUpdateActivity Permission = "update_activity"
	PermDeleteActivity Permission = "delete_activity"
	PermViewAllActivities Permission = "view_all_activities"
	PermManageParticipations Permission = "manage_participations"

	// Participation Permissions
	PermJoinActivity Permission = "join_activity"
	PermLeaveActivity Permission = "leave_activity"
	PermApproveParticipation Permission = "approve_participation"
	PermMarkAttendance Permission = "mark_attendance"

	// Report Permissions
	PermViewSystemReports Permission = "view_system_reports"
	PermViewFacultyReports Permission = "view_faculty_reports"

	// QR and Subscription Permissions
	PermScanQRCode Permission = "scan_qr_code"
	PermViewSubscriptions Permission = "view_subscriptions"
)

type PermissionChecker struct{}

func NewPermissionChecker() *PermissionChecker {
	return &PermissionChecker{}
}

// HasPermission ตรวจสอบว่า user มีสิทธิ์ในการทำ permission หนึ่งๆ หรือไม่
func (pc *PermissionChecker) HasPermission(user *models.User, permission Permission) bool {
	switch user.Role {
	case models.UserRoleSuperAdmin:
		return pc.superAdminPermissions(permission)
	case models.UserRoleFacultyAdmin:
		return pc.facultyAdminPermissions(permission)
	case models.UserRoleRegularAdmin:
		return pc.regularAdminPermissions(permission)
	case models.UserRoleStudent:
		return pc.studentPermissions(permission)
	default:
		return false
	}
}

// HasFacultyPermission ตรวจสอบสิทธิ์ที่เกี่ยวข้องกับ faculty
func (pc *PermissionChecker) HasFacultyPermission(user *models.User, permission Permission, facultyID uint) bool {
	if !pc.HasPermission(user, permission) {
		return false
	}

	switch user.Role {
	case models.UserRoleSuperAdmin:
		return true // Super admin สามารถจัดการทุก faculty
	case models.UserRoleFacultyAdmin:
		return user.FacultyID != nil && *user.FacultyID == facultyID
	case models.UserRoleRegularAdmin:
		return user.FacultyID != nil && *user.FacultyID == facultyID
	default:
		return false
	}
}

// HasDepartmentPermission ตรวจสอบสิทธิ์ที่เกี่ยวข้องกับ department
func (pc *PermissionChecker) HasDepartmentPermission(user *models.User, permission Permission, departmentID uint) bool {
	if !pc.HasPermission(user, permission) {
		return false
	}

	switch user.Role {
	case models.UserRoleSuperAdmin:
		return true // Super admin สามารถจัดการทุก department
	case models.UserRoleFacultyAdmin:
		// Faculty admin สามารถจัดการ department ในคณะของตนเอง
		return user.FacultyID != nil
	case models.UserRoleRegularAdmin:
		// Regular admin สามารถดู department ในคณะของตนเองเท่านั้น
		return user.DepartmentID != nil && *user.DepartmentID == departmentID
	default:
		return false
	}
}

// CanManageUser ตรวจสอบว่าสามารถจัดการ user คนหนึ่งๆ ได้หรือไม่
func (pc *PermissionChecker) CanManageUser(manager *models.User, targetUser *models.User) bool {
	switch manager.Role {
	case models.UserRoleSuperAdmin:
		return true // Super admin สามารถจัดการทุกคน
	case models.UserRoleFacultyAdmin:
		// Faculty admin สามารถจัดการคนในคณะเดียวกัน (ยกเว้น super admin และ faculty admin คนอื่น)
		if targetUser.Role == models.UserRoleSuperAdmin {
			return false
		}
		if targetUser.Role == models.UserRoleFacultyAdmin && targetUser.ID != manager.ID {
			return false
		}
		return manager.FacultyID != nil && targetUser.FacultyID != nil && *manager.FacultyID == *targetUser.FacultyID
	case models.UserRoleRegularAdmin:
		// Regular admin สามารถดูข้อมูล student เท่านั้น
		return targetUser.Role == models.UserRoleStudent &&
			manager.FacultyID != nil && targetUser.FacultyID != nil && 
			*manager.FacultyID == *targetUser.FacultyID
	default:
		return false
	}
}

// Super Admin Permissions - สามารถทำทุกอย่างได้
func (pc *PermissionChecker) superAdminPermissions(permission Permission) bool {
	allowedPermissions := []Permission{
		PermCreateFaculty, PermUpdateFaculty, PermDeleteFaculty, PermViewAllFaculties,
		PermCreateDepartment, PermUpdateDepartment, PermDeleteDepartment, PermViewDepartments,
		PermCreateUser, PermUpdateUser, PermDeleteUser, PermViewUsers, PermViewUserDetails,
		PermCreateActivity, PermUpdateActivity, PermDeleteActivity, PermViewAllActivities, PermManageParticipations,
		PermJoinActivity, PermLeaveActivity, PermApproveParticipation, PermMarkAttendance,
		PermViewSystemReports, PermViewFacultyReports,
		PermScanQRCode, PermViewSubscriptions,
	}

	for _, perm := range allowedPermissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// Faculty Admin Permissions - จัดการคณะตัวเอง
func (pc *PermissionChecker) facultyAdminPermissions(permission Permission) bool {
	allowedPermissions := []Permission{
		PermViewAllFaculties, // สามารถดู faculty ทั้งหมดได้
		PermCreateDepartment, PermUpdateDepartment, PermDeleteDepartment, PermViewDepartments,
		PermCreateUser, PermUpdateUser, PermDeleteUser, PermViewUsers, PermViewUserDetails,
		PermCreateActivity, PermUpdateActivity, PermDeleteActivity, PermViewAllActivities, PermManageParticipations,
		PermJoinActivity, PermLeaveActivity, PermApproveParticipation, PermMarkAttendance,
		PermViewFacultyReports,
		PermScanQRCode, PermViewSubscriptions,
	}

	for _, perm := range allowedPermissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// Regular Admin Permissions - ดำเนินการระดับกิจกรรม
func (pc *PermissionChecker) regularAdminPermissions(permission Permission) bool {
	allowedPermissions := []Permission{
		PermViewAllFaculties, PermViewDepartments,
		PermViewUsers, PermViewUserDetails, // สามารถดูข้อมูล user ได้
		PermViewAllActivities, // สามารถดูกิจกรรมทั้งหมดได้
		PermJoinActivity, PermLeaveActivity, PermApproveParticipation, PermMarkAttendance,
		PermScanQRCode,
	}

	for _, perm := range allowedPermissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// Student Permissions - สิทธิ์พื้นฐาน
func (pc *PermissionChecker) studentPermissions(permission Permission) bool {
	allowedPermissions := []Permission{
		PermViewAllFaculties, PermViewDepartments,
		PermViewAllActivities,
		PermJoinActivity, PermLeaveActivity,
	}

	for _, perm := range allowedPermissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// GetUserPermissions ส่งคืนรายการ permission ทั้งหมดที่ user มี
func (pc *PermissionChecker) GetUserPermissions(user *models.User) []Permission {
	var permissions []Permission
	allPermissions := []Permission{
		PermCreateFaculty, PermUpdateFaculty, PermDeleteFaculty, PermViewAllFaculties,
		PermCreateDepartment, PermUpdateDepartment, PermDeleteDepartment, PermViewDepartments,
		PermCreateUser, PermUpdateUser, PermDeleteUser, PermViewUsers, PermViewUserDetails,
		PermCreateActivity, PermUpdateActivity, PermDeleteActivity, PermViewAllActivities, PermManageParticipations,
		PermJoinActivity, PermLeaveActivity, PermApproveParticipation, PermMarkAttendance,
		PermViewSystemReports, PermViewFacultyReports,
		PermScanQRCode, PermViewSubscriptions,
	}

	for _, perm := range allPermissions {
		if pc.HasPermission(user, perm) {
			permissions = append(permissions, perm)
		}
	}

	return permissions
}
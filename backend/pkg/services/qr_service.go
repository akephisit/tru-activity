package services

import (
	"fmt"
	"time"

	"github.com/kruakemaths/tru-activity/backend/internal/models"
	"github.com/kruakemaths/tru-activity/backend/pkg/utils"
	"gorm.io/gorm"
)

type QRService struct {
	DB            *gorm.DB
	SecretManager *utils.QRSecretManager
	MaxQRAge      time.Duration
}

type QRScanRequest struct {
	QRData       string `json:"qr_data"`
	ActivityID   uint   `json:"activity_id"`
	AdminID      uint   `json:"admin_id"`
	ScanLocation string `json:"scan_location,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
}

type QRScanResult struct {
	Success        bool                  `json:"success"`
	Message        string                `json:"message"`
	Participation  *models.Participation `json:"participation,omitempty"`
	User           *models.User          `json:"user,omitempty"`
	ScanLog        *models.QRScanLog     `json:"scan_log,omitempty"`
}

func NewQRService(db *gorm.DB, masterKey string, maxAge time.Duration) *QRService {
	return &QRService{
		DB:            db,
		SecretManager: utils.NewQRSecretManager(masterKey),
		MaxQRAge:      maxAge,
	}
}

// ScanQRCode processes QR code scan and updates participation
func (qs *QRService) ScanQRCode(req *QRScanRequest) (*QRScanResult, error) {
	// Parse QR data
	qrData, err := utils.ParseQRData(req.QRData)
	if err != nil {
		return qs.createFailedResult("Invalid QR code format", req, err.Error()), nil
	}

	// Find user by student ID
	var user models.User
	if err := qs.DB.Where("student_id = ?", qrData.StudentID).First(&user).Error; err != nil {
		return qs.createFailedResult("Student not found", req, "Student ID not found in database"), nil
	}

	// Validate QR signature
	if err := qs.SecretManager.ValidateQRData(qrData, user.QRSecret, qs.MaxQRAge); err != nil {
		return qs.createFailedResult("Invalid QR code", req, err.Error()), nil
	}

	// Check if activity exists and admin has permission
	var activity models.Activity
	if err := qs.DB.Preload("Faculty").Preload("Department").First(&activity, req.ActivityID).Error; err != nil {
		return qs.createFailedResult("Activity not found", req, "Activity does not exist"), nil
	}

	// Verify admin permissions
	var admin models.User
	if err := qs.DB.First(&admin, req.AdminID).Error; err != nil {
		return qs.createFailedResult("Admin not found", req, "Admin user not found"), nil
	}

	if !qs.canAdminScanForActivity(&admin, &activity) {
		return qs.createFailedResult("Permission denied", req, "Admin does not have permission to scan for this activity"), nil
	}

	// Find or create participation
	var participation models.Participation
	err = qs.DB.Where("user_id = ? AND activity_id = ?", user.ID, req.ActivityID).First(&participation).Error
	
	if err == gorm.ErrRecordNotFound {
		// Auto-register user if activity allows it
		if activity.RequireApproval && !activity.AutoApprove {
			return qs.createFailedResult("Registration required", req, "User must register for this activity first"), nil
		}

		// Create new participation
		participation = models.Participation{
			UserID:       user.ID,
			ActivityID:   req.ActivityID,
			Status:       models.ParticipationStatusApproved,
			RegisteredAt: time.Now(),
			ApprovedAt:   timePtr(time.Now()),
		}

		if err := qs.DB.Create(&participation).Error; err != nil {
			return qs.createFailedResult("Failed to create participation", req, err.Error()), nil
		}
	} else if err != nil {
		return qs.createFailedResult("Database error", req, err.Error()), nil
	}

	// Update participation with scan details
	now := time.Now()
	updates := map[string]interface{}{
		"qr_scanned_at": &now,
		"scanned_by_id": req.AdminID,
		"scan_location": req.ScanLocation,
		"attended_at":   &now,
		"status":        models.ParticipationStatusAttended,
	}

	if err := qs.DB.Model(&participation).Updates(updates).Error; err != nil {
		return qs.createFailedResult("Failed to update participation", req, err.Error()), nil
	}

	// Create successful scan log
	scanLog := qs.createScanLog(req, qrData, &user, true, "")
	qs.DB.Create(&scanLog)

	// Reload participation with associations
	qs.DB.Preload("User").Preload("Activity").First(&participation, participation.ID)

	return &QRScanResult{
		Success:       true,
		Message:       "QR code scanned successfully",
		Participation: &participation,
		User:          &user,
		ScanLog:       &scanLog,
	}, nil
}

// GenerateUserQRData generates QR data for a user
func (qs *QRService) GenerateUserQRData(userID uint) (*utils.QRData, error) {
	var user models.User
	if err := qs.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	return qs.SecretManager.GenerateQRData(user.StudentID, user.QRSecret)
}

// RefreshUserQRSecret regenerates QR secret for a user
func (qs *QRService) RefreshUserQRSecret(userID uint) (*models.User, error) {
	var user models.User
	if err := qs.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	newSecret := utils.RegenerateUserSecret()
	if err := qs.DB.Model(&user).Update("qr_secret", newSecret).Error; err != nil {
		return nil, fmt.Errorf("failed to update QR secret: %v", err)
	}

	user.QRSecret = newSecret
	return &user, nil
}

// GetUserQRHistory gets QR scan history for a user
func (qs *QRService) GetUserQRHistory(userID uint, limit int) ([]models.QRScanLog, error) {
	var logs []models.QRScanLog
	query := qs.DB.Preload("Activity").Preload("ScannedBy").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch QR history: %v", err)
	}

	return logs, nil
}

// GetActivityQRScans gets QR scan logs for an activity
func (qs *QRService) GetActivityQRScans(activityID uint, limit int) ([]models.QRScanLog, error) {
	var logs []models.QRScanLog
	query := qs.DB.Preload("User").Preload("ScannedBy").
		Where("activity_id = ?", activityID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch activity QR scans: %v", err)
	}

	return logs, nil
}

// Helper methods

func (qs *QRService) createFailedResult(message, request string, errorDetails string) *QRScanResult {
	scanLog := qs.createScanLog(parseQRScanRequest(request), nil, nil, false, errorDetails)
	qs.DB.Create(&scanLog)

	return &QRScanResult{
		Success: false,
		Message: message,
		ScanLog: &scanLog,
	}
}

func (qs *QRService) createScanLog(req *QRScanRequest, qrData *utils.QRData, user *models.User, valid bool, errorMsg string) models.QRScanLog {
	log := models.QRScanLog{
		ActivityID:    req.ActivityID,
		ScannedByID:   req.AdminID,
		ScanTimestamp: time.Now(),
		Valid:         valid,
		ErrorMessage:  errorMsg,
		ScanLocation:  req.ScanLocation,
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
	}

	if qrData != nil {
		log.StudentID = qrData.StudentID
		log.QRTimestamp = time.Unix(qrData.Timestamp, 0)
	}

	if user != nil {
		log.UserID = &user.ID
	}

	return log
}

func (qs *QRService) canAdminScanForActivity(admin *models.User, activity *models.Activity) bool {
	// Super admin can scan for any activity
	if admin.Role == models.UserRoleSuperAdmin {
		return true
	}

	// Faculty admin can scan for activities in their faculty
	if admin.Role == models.UserRoleFacultyAdmin {
		if activity.FacultyID != nil && admin.FacultyID != nil && *activity.FacultyID == *admin.FacultyID {
			return true
		}
		// Also allow if admin created the activity
		if activity.CreatedByID == admin.ID {
			return true
		}
	}

	// Regular admin needs explicit assignment
	if admin.Role == models.UserRoleRegularAdmin {
		var assignment models.ActivityAssignment
		err := qs.DB.Where("activity_id = ? AND admin_id = ? AND can_scan_qr = true", activity.ID, admin.ID).First(&assignment).Error
		return err == nil
	}

	return false
}

func parseQRScanRequest(reqStr string) *QRScanRequest {
	// This is a helper for error cases - in practice, we'd parse the actual request
	return &QRScanRequest{}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
package services

import (
	"fmt"
	"log"
	"time"

	"github.com/kruakemaths/tru-activity/backend/internal/models"
	"gorm.io/gorm"
)

type EventPublisher struct {
	DB               *gorm.DB
	PubSubService    *PubSubService
	ConnectionManager *ConnectionManager
	instanceID       string
}

type EventContext struct {
	UserID        *uint  `json:"user_id,omitempty"`
	FacultyID     *uint  `json:"faculty_id,omitempty"`
	ActivityID    *uint  `json:"activity_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	Source        string `json:"source,omitempty"`
}

func NewEventPublisher(db *gorm.DB, pubsub *PubSubService, cm *ConnectionManager, instanceID string) *EventPublisher {
	return &EventPublisher{
		DB:               db,
		PubSubService:    pubsub,
		ConnectionManager: cm,
		instanceID:       instanceID,
	}
}

// Activity Events

func (ep *EventPublisher) PublishActivityCreated(activity *models.Activity, ctx *EventContext) error {
	metadata := ep.createMetadata(ctx)
	
	// Publish to faculty members
	if activity.FacultyID != nil {
		if err := ep.PubSubService.PublishNewActivity(*activity.FacultyID, activity, metadata); err != nil {
			log.Printf("Failed to publish new activity to faculty %d: %v", *activity.FacultyID, err)
		}
	}

	// Publish to all students (cross-faculty activities)
	if activity.FacultyID == nil {
		if err := ep.publishSystemWideActivity(activity, "activity_created", metadata); err != nil {
			log.Printf("Failed to publish system-wide activity: %v", err)
		}
	}

	// Publish activity update
	if err := ep.PubSubService.PublishActivityUpdate(activity.ID, activity, metadata); err != nil {
		log.Printf("Failed to publish activity update: %v", err)
	}

	log.Printf("Published activity created event for activity %d", activity.ID)
	return nil
}

func (ep *EventPublisher) PublishActivityUpdated(activity *models.Activity, updateType string, ctx *EventContext) error {
	metadata := ep.createMetadata(ctx)
	metadata.ActivityID = &activity.ID

	// Publish to activity subscribers
	if err := ep.PubSubService.PublishActivityUpdate(activity.ID, map[string]interface{}{
		"activity":    activity,
		"update_type": updateType,
	}, metadata); err != nil {
		return fmt.Errorf("failed to publish activity update: %v", err)
	}

	// Notify faculty members if status changed
	if updateType == "status_changed" && activity.FacultyID != nil {
		if err := ep.PubSubService.PublishFacultyUpdate(*activity.FacultyID, map[string]interface{}{
			"type":     "activity_status_changed",
			"activity": activity,
		}, metadata); err != nil {
			log.Printf("Failed to publish faculty update: %v", err)
		}
	}

	log.Printf("Published activity updated event for activity %d (type: %s)", activity.ID, updateType)
	return nil
}

func (ep *EventPublisher) PublishActivityAssigned(assignment *models.ActivityAssignment, ctx *EventContext) error {
	metadata := ep.createMetadata(ctx)
	metadata.ActivityID = &assignment.ActivityID
	metadata.UserID = &assignment.AdminID

	// Notify the assigned admin
	if err := ep.PubSubService.PublishActivityAssignment(assignment.AdminID, assignment, metadata); err != nil {
		return fmt.Errorf("failed to publish activity assignment: %v", err)
	}

	// Send personal notification
	if err := ep.PubSubService.PublishPersonalNotification(assignment.AdminID, map[string]interface{}{
		"type":       "activity_assigned",
		"message":    fmt.Sprintf("You have been assigned to activity: %s", assignment.Activity.Title),
		"assignment": assignment,
	}, metadata); err != nil {
		log.Printf("Failed to send personal notification: %v", err)
	}

	log.Printf("Published activity assignment for admin %d, activity %d", assignment.AdminID, assignment.ActivityID)
	return nil
}

// Participation Events

func (ep *EventPublisher) PublishParticipationUpdated(participation *models.Participation, updateType string, ctx *EventContext) error {
	metadata := ep.createMetadata(ctx)
	metadata.UserID = &participation.UserID
	metadata.ActivityID = &participation.ActivityID

	// Publish to participation event subscribers
	if err := ep.PubSubService.PublishParticipationEvent(
		participation.ActivityID, 
		participation.UserID, 
		map[string]interface{}{
			"participation": participation,
			"update_type":   updateType,
		}, 
		metadata,
	); err != nil {
		return fmt.Errorf("failed to publish participation event: %v", err)
	}

	// Send personal notification to participant
	message := ep.getParticipationMessage(updateType, participation)
	if err := ep.PubSubService.PublishPersonalNotification(participation.UserID, map[string]interface{}{
		"type":          "participation_update",
		"message":       message,
		"participation": participation,
		"update_type":   updateType,
	}, metadata); err != nil {
		log.Printf("Failed to send personal notification: %v", err)
	}

	// Publish activity update to admins
	if err := ep.PubSubService.PublishActivityUpdate(participation.ActivityID, map[string]interface{}{
		"type":          "participation_updated",
		"participation": participation,
		"update_type":   updateType,
	}, metadata); err != nil {
		log.Printf("Failed to publish activity update: %v", err)
	}

	log.Printf("Published participation updated event for user %d, activity %d (type: %s)", 
		participation.UserID, participation.ActivityID, updateType)
	return nil
}

// QR Scan Events

func (ep *EventPublisher) PublishQRScanResult(scanResult *QRScanResult, activity *models.Activity, ctx *EventContext) error {
	metadata := ep.createMetadata(ctx)
	metadata.ActivityID = &activity.ID

	if scanResult.User != nil {
		metadata.UserID = &scanResult.User.ID
	}

	// Publish to QR scan event subscribers
	if err := ep.PubSubService.PublishQRScanEvent(activity.ID, scanResult, metadata); err != nil {
		return fmt.Errorf("failed to publish QR scan event: %v", err)
	}

	// If scan was successful, also publish participation update
	if scanResult.Success && scanResult.Participation != nil {
		if err := ep.PublishParticipationUpdated(scanResult.Participation, "qr_scanned", ctx); err != nil {
			log.Printf("Failed to publish participation update from QR scan: %v", err)
		}
	}

	// Notify activity admins
	adminMessage := fmt.Sprintf("QR scan %s for activity: %s", 
		map[bool]string{true: "successful", false: "failed"}[scanResult.Success], 
		activity.Title)
	
	if err := ep.notifyActivityAdmins(activity, map[string]interface{}{
		"type":        "qr_scan_result",
		"message":     adminMessage,
		"scan_result": scanResult,
		"activity":    activity,
	}, metadata); err != nil {
		log.Printf("Failed to notify activity admins: %v", err)
	}

	log.Printf("Published QR scan result for activity %d (success: %v)", activity.ID, scanResult.Success)
	return nil
}

// System Events

func (ep *EventPublisher) PublishSystemAlert(alertType, message string, severity models.AlertSeverity, targetRoles []models.UserRole, facultyID *uint, ctx *EventContext) error {
	alert := &models.SystemAlert{
		Type:        alertType,
		Message:     message,
		Severity:    severity,
		TargetRoles: targetRoles,
		FacultyID:   facultyID,
		CreatedAt:   time.Now(),
	}

	// Save to database
	if err := ep.DB.Create(alert).Error; err != nil {
		log.Printf("Failed to save system alert: %v", err)
	}

	metadata := ep.createMetadata(ctx)
	if facultyID != nil {
		metadata.FacultyID = facultyID
	}

	// Publish system alert
	if err := ep.PubSubService.PublishSystemAlert(alert, metadata); err != nil {
		return fmt.Errorf("failed to publish system alert: %v", err)
	}

	log.Printf("Published system alert: %s (severity: %s)", message, severity)
	return nil
}

func (ep *EventPublisher) PublishSubscriptionWarning(subscription *models.Subscription, warningType string, ctx *EventContext) error {
	metadata := ep.createMetadata(ctx)
	metadata.FacultyID = &subscription.FacultyID

	warning := map[string]interface{}{
		"type":         warningType,
		"subscription": subscription,
		"message":      ep.getSubscriptionWarningMessage(warningType, subscription),
		"days_left":    subscription.DaysUntilExpiry(),
	}

	// Publish subscription warning
	if err := ep.PubSubService.PublishSubscriptionWarning(subscription.FacultyID, warning, metadata); err != nil {
		return fmt.Errorf("failed to publish subscription warning: %v", err)
	}

	// Notify faculty admins
	if err := ep.notifyFacultyAdmins(subscription.FacultyID, warning, metadata); err != nil {
		log.Printf("Failed to notify faculty admins: %v", err)
	}

	log.Printf("Published subscription warning for faculty %d (type: %s)", subscription.FacultyID, warningType)
	return nil
}

// Faculty Events

func (ep *EventPublisher) PublishFacultyUpdated(faculty *models.Faculty, updateType string, ctx *EventContext) error {
	metadata := ep.createMetadata(ctx)
	metadata.FacultyID = &faculty.ID

	update := map[string]interface{}{
		"type":        updateType,
		"faculty":     faculty,
		"message":     fmt.Sprintf("Faculty %s has been %s", faculty.Name, updateType),
		"update_type": updateType,
	}

	// Publish faculty update
	if err := ep.PubSubService.PublishFacultyUpdate(faculty.ID, update, metadata); err != nil {
		return fmt.Errorf("failed to publish faculty update: %v", err)
	}

	log.Printf("Published faculty updated event for faculty %d (type: %s)", faculty.ID, updateType)
	return nil
}

// Helper methods

func (ep *EventPublisher) createMetadata(ctx *EventContext) *SubscriptionMetadata {
	if ctx == nil {
		ctx = &EventContext{}
	}

	metadata := &SubscriptionMetadata{
		Source:        fmt.Sprintf("instance_%s", ep.instanceID),
		UserID:        ctx.UserID,
		FacultyID:     ctx.FacultyID,
		ActivityID:    ctx.ActivityID,
		CorrelationID: ctx.CorrelationID,
	}

	if ctx.Source != "" {
		metadata.Source = ctx.Source
	}

	return metadata
}

func (ep *EventPublisher) getParticipationMessage(updateType string, participation *models.Participation) string {
	messages := map[string]string{
		"registered": fmt.Sprintf("You have registered for: %s", participation.Activity.Title),
		"approved":   fmt.Sprintf("Your participation in '%s' has been approved", participation.Activity.Title),
		"rejected":   fmt.Sprintf("Your participation in '%s' has been rejected", participation.Activity.Title),
		"attended":   fmt.Sprintf("You have been marked as attended for: %s", participation.Activity.Title),
		"qr_scanned": fmt.Sprintf("QR code scanned successfully for: %s", participation.Activity.Title),
	}

	if message, exists := messages[updateType]; exists {
		return message
	}
	return fmt.Sprintf("Your participation status has been updated for: %s", participation.Activity.Title)
}

func (ep *EventPublisher) getSubscriptionWarningMessage(warningType string, subscription *models.Subscription) string {
	daysLeft := subscription.DaysUntilExpiry()
	
	messages := map[string]string{
		"expiring_soon": fmt.Sprintf("Your %s subscription will expire in %d days", subscription.Type, daysLeft),
		"expired":       fmt.Sprintf("Your %s subscription has expired", subscription.Type),
		"renewed":       fmt.Sprintf("Your %s subscription has been renewed", subscription.Type),
	}

	if message, exists := messages[warningType]; exists {
		return message
	}
	return fmt.Sprintf("Subscription status update: %s", warningType)
}

func (ep *EventPublisher) notifyActivityAdmins(activity *models.Activity, data interface{}, metadata *SubscriptionMetadata) error {
	// Find activity admins (super admins, faculty admins, and assigned regular admins)
	var adminUsers []models.User

	// Super admins
	if err := ep.DB.Where("role = ?", models.UserRoleSuperAdmin).Find(&adminUsers).Error; err != nil {
		log.Printf("Failed to fetch super admins: %v", err)
	}

	// Faculty admins
	if activity.FacultyID != nil {
		var facultyAdmins []models.User
		if err := ep.DB.Where("role = ? AND faculty_id = ?", models.UserRoleFacultyAdmin, *activity.FacultyID).Find(&facultyAdmins).Error; err != nil {
			log.Printf("Failed to fetch faculty admins: %v", err)
		}
		adminUsers = append(adminUsers, facultyAdmins...)
	}

	// Assigned regular admins
	var assignments []models.ActivityAssignment
	if err := ep.DB.Preload("Admin").Where("activity_id = ?", activity.ID).Find(&assignments).Error; err != nil {
		log.Printf("Failed to fetch activity assignments: %v", err)
	}
	for _, assignment := range assignments {
		adminUsers = append(adminUsers, assignment.Admin)
	}

	// Send notifications
	for _, admin := range adminUsers {
		if err := ep.PubSubService.PublishPersonalNotification(admin.ID, data, metadata); err != nil {
			log.Printf("Failed to notify admin %d: %v", admin.ID, err)
		}
	}

	return nil
}

func (ep *EventPublisher) notifyFacultyAdmins(facultyID uint, data interface{}, metadata *SubscriptionMetadata) error {
	var adminUsers []models.User

	// Super admins
	if err := ep.DB.Where("role = ?", models.UserRoleSuperAdmin).Find(&adminUsers).Error; err != nil {
		log.Printf("Failed to fetch super admins: %v", err)
	}

	// Faculty admins
	var facultyAdmins []models.User
	if err := ep.DB.Where("role = ? AND faculty_id = ?", models.UserRoleFacultyAdmin, facultyID).Find(&facultyAdmins).Error; err != nil {
		log.Printf("Failed to fetch faculty admins: %v", err)
	}
	adminUsers = append(adminUsers, facultyAdmins...)

	// Send notifications
	for _, admin := range adminUsers {
		if err := ep.PubSubService.PublishPersonalNotification(admin.ID, data, metadata); err != nil {
			log.Printf("Failed to notify faculty admin %d: %v", admin.ID, err)
		}
	}

	return nil
}

func (ep *EventPublisher) publishSystemWideActivity(activity *models.Activity, eventType string, metadata *SubscriptionMetadata) error {
	// Publish to all faculties for cross-faculty activities
	var faculties []models.Faculty
	if err := ep.DB.Where("is_active = true").Find(&faculties).Error; err != nil {
		return fmt.Errorf("failed to fetch faculties: %v", err)
	}

	for _, faculty := range faculties {
		if err := ep.PubSubService.PublishNewActivity(faculty.ID, map[string]interface{}{
			"type":     eventType,
			"activity": activity,
		}, metadata); err != nil {
			log.Printf("Failed to publish to faculty %d: %v", faculty.ID, err)
		}
	}

	return nil
}

// Batch event publishing for performance

func (ep *EventPublisher) PublishBulkParticipationUpdates(participations []models.Participation, updateType string, ctx *EventContext) error {
	for _, participation := range participations {
		if err := ep.PublishParticipationUpdated(&participation, updateType, ctx); err != nil {
			log.Printf("Failed to publish participation update for user %d: %v", participation.UserID, err)
		}
	}
	return nil
}

func (ep *EventPublisher) PublishBulkActivityUpdates(activities []models.Activity, updateType string, ctx *EventContext) error {
	for _, activity := range activities {
		if err := ep.PublishActivityUpdated(&activity, updateType, ctx); err != nil {
			log.Printf("Failed to publish activity update for activity %d: %v", activity.ID, err)
		}
	}
	return nil
}

// Connection status events

func (ep *EventPublisher) PublishConnectionStats() error {
	stats := ep.ConnectionManager.GetConnectionStats()
	
	metadata := &SubscriptionMetadata{
		Source: fmt.Sprintf("instance_%s", ep.instanceID),
	}

	// Publish to system alerts channel for monitoring
	if err := ep.PubSubService.Publish("connection_stats", &SubscriptionEvent{
		Type:     "connection_stats",
		Data:     stats,
		Metadata: metadata,
	}); err != nil {
		return fmt.Errorf("failed to publish connection stats: %v", err)
	}

	return nil
}
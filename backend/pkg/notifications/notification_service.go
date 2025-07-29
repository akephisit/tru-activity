package notifications

import (
	"fmt"
	"log"
	"net/smtp"
	"time"

	"github.com/kruakemaths/tru-activity/backend/internal/models"
	"gorm.io/gorm"
)

type NotificationService struct {
	DB         *gorm.DB
	SMTPConfig SMTPConfig
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type EmailTemplate struct {
	Subject string
	Body    string
}

func NewNotificationService(db *gorm.DB, smtpConfig SMTPConfig) *NotificationService {
	return &NotificationService{
		DB:         db,
		SMTPConfig: smtpConfig,
	}
}

// CheckExpiringSoon finds subscriptions that need notifications
func (ns *NotificationService) CheckExpiringSoon() error {
	var subscriptions []models.Subscription
	
	// Find subscriptions that need notifications
	if err := ns.DB.Preload("Faculty").Where("status = ?", models.SubscriptionStatusActive).Find(&subscriptions).Error; err != nil {
		return fmt.Errorf("failed to fetch subscriptions: %v", err)
	}

	for _, subscription := range subscriptions {
		if subscription.NeedsNotification() {
			if err := ns.SendExpiryNotification(&subscription); err != nil {
				log.Printf("Failed to send notification for subscription %d: %v", subscription.ID, err)
			}
		}
	}

	return nil
}

// SendExpiryNotification sends email notification for expiring subscription
func (ns *NotificationService) SendExpiryNotification(subscription *models.Subscription) error {
	notificationType := subscription.GetNotificationType()
	if notificationType == "" {
		return nil // No notification needed
	}

	// Get faculty admins to notify
	var facultyAdmins []models.User
	if err := ns.DB.Where("faculty_id = ? AND role = ?", subscription.FacultyID, models.UserRoleFacultyAdmin).Find(&facultyAdmins).Error; err != nil {
		return fmt.Errorf("failed to fetch faculty admins: %v", err)
	}

	template := ns.getEmailTemplate(notificationType, subscription)

	for _, admin := range facultyAdmins {
		// Create notification log
		notificationLog := models.NotificationLog{
			SubscriptionID: subscription.ID,
			Type:           models.NotificationType(fmt.Sprintf("expiry_%s", notificationType)),
			Status:         models.NotificationStatusPending,
			Email:          admin.Email,
			Subject:        template.Subject,
			Message:        template.Body,
		}

		if err := ns.DB.Create(&notificationLog).Error; err != nil {
			log.Printf("Failed to create notification log: %v", err)
			continue
		}

		// Send email
		if err := ns.sendEmail(admin.Email, template.Subject, template.Body); err != nil {
			// Mark as failed
			ns.DB.Model(&notificationLog).Updates(map[string]interface{}{
				"status":       models.NotificationStatusFailed,
				"error_message": err.Error(),
			})
			log.Printf("Failed to send email to %s: %v", admin.Email, err)
		} else {
			// Mark as sent
			now := time.Now()
			ns.DB.Model(&notificationLog).Updates(map[string]interface{}{
				"status":  models.NotificationStatusSent,
				"sent_at": &now,
			})

			// Update subscription notification flags
			updates := make(map[string]interface{})
			if notificationType == "7_days" {
				updates["notification_sent_7_days"] = true
			} else if notificationType == "1_day" {
				updates["notification_sent_1_day"] = true
			}
			updates["last_notification_at"] = &now

			ns.DB.Model(subscription).Updates(updates)
		}
	}

	return nil
}

func (ns *NotificationService) getEmailTemplate(notificationType string, subscription *models.Subscription) EmailTemplate {
	daysLeft := subscription.DaysUntilExpiry()
	facultyName := subscription.Faculty.Name

	switch notificationType {
	case "7_days":
		return EmailTemplate{
			Subject: fmt.Sprintf("Subscription Expiry Warning - %s", facultyName),
			Body: fmt.Sprintf(`
Dear Faculty Admin,

This is a reminder that your subscription for %s will expire in %d days.

Subscription Details:
- Faculty: %s
- Plan: %s
- Expiry Date: %s

Please contact your system administrator to renew your subscription.

Best regards,
TRU Activity System
`, facultyName, daysLeft, facultyName, subscription.Type, subscription.EndDate.Format("2006-01-02")),
		}
	case "1_day":
		return EmailTemplate{
			Subject: fmt.Sprintf("Urgent: Subscription Expires Tomorrow - %s", facultyName),
			Body: fmt.Sprintf(`
Dear Faculty Admin,

URGENT: Your subscription for %s will expire in %d day(s).

Subscription Details:
- Faculty: %s
- Plan: %s
- Expiry Date: %s

Please contact your system administrator immediately to renew your subscription.

Best regards,
TRU Activity System
`, facultyName, daysLeft, facultyName, subscription.Type, subscription.EndDate.Format("2006-01-02")),
		}
	default:
		return EmailTemplate{
			Subject: fmt.Sprintf("Subscription Notification - %s", facultyName),
			Body:    "Your subscription requires attention.",
		}
	}
}

func (ns *NotificationService) sendEmail(to, subject, body string) error {
	auth := smtp.PlainAuth("", ns.SMTPConfig.Username, ns.SMTPConfig.Password, ns.SMTPConfig.Host)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		ns.SMTPConfig.From, to, subject, body)

	addr := fmt.Sprintf("%s:%s", ns.SMTPConfig.Host, ns.SMTPConfig.Port)
	return smtp.SendMail(addr, auth, ns.SMTPConfig.From, []string{to}, []byte(msg))
}

// StartNotificationScheduler starts a background service to check for expiring subscriptions
func (ns *NotificationService) StartNotificationScheduler() {
	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()

	// Run once immediately
	if err := ns.CheckExpiringSoon(); err != nil {
		log.Printf("Error checking expiring subscriptions: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := ns.CheckExpiringSoon(); err != nil {
				log.Printf("Error checking expiring subscriptions: %v", err)
			}
		}
	}
}
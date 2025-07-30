package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/kruakemaths/tru-activity/backend/internal/models"
	"gorm.io/gorm"
)

type RealtimeService struct {
	DB          *gorm.DB
	subscribers map[string]map[string]*Subscriber
	mutex       sync.RWMutex
}

type Subscriber struct {
	ID      string
	UserID  uint
	Channel chan *NotificationMessage
	Filters map[string]interface{}
}

type NotificationMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	UserID    *uint       `json:"user_id,omitempty"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

type QRScanNotification struct {
	ScanResult    *QRScanResult         `json:"scan_result"`
	Activity      *models.Activity      `json:"activity"`
	Participation *models.Participation `json:"participation"`
}

type ParticipationUpdateNotification struct {
	Participation *models.Participation `json:"participation"`
	Activity      *models.Activity      `json:"activity"`
	UpdateType    string                `json:"update_type"` // "approved", "attended", "registered"
}

type ActivityUpdateNotification struct {
	Activity   *models.Activity `json:"activity"`
	UpdateType string           `json:"update_type"` // "created", "updated", "status_changed"
}

func NewRealtimeService(db *gorm.DB) *RealtimeService {
	return &RealtimeService{
		DB:          db,
		subscribers: make(map[string]map[string]*Subscriber),
		mutex:       sync.RWMutex{},
	}
}

// Subscribe adds a new subscriber to receive real-time notifications
func (rs *RealtimeService) Subscribe(ctx context.Context, userID uint, subscriptionType string, filters map[string]interface{}) (*Subscriber, error) {
	subscriber := &Subscriber{
		ID:      generateSubscriberID(),
		UserID:  userID,
		Channel: make(chan *NotificationMessage, 100), // Buffer for 100 messages
		Filters: filters,
	}

	rs.mutex.Lock()
	if rs.subscribers[subscriptionType] == nil {
		rs.subscribers[subscriptionType] = make(map[string]*Subscriber)
	}
	rs.subscribers[subscriptionType][subscriber.ID] = subscriber
	rs.mutex.Unlock()

	// Start cleanup goroutine for this subscriber
	go rs.cleanupSubscriber(ctx, subscriptionType, subscriber.ID)

	log.Printf("User %d subscribed to %s with ID %s", userID, subscriptionType, subscriber.ID)
	return subscriber, nil
}

// Unsubscribe removes a subscriber
func (rs *RealtimeService) Unsubscribe(subscriptionType, subscriberID string) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	if subscribers, exists := rs.subscribers[subscriptionType]; exists {
		if subscriber, exists := subscribers[subscriberID]; exists {
			close(subscriber.Channel)
			delete(subscribers, subscriberID)
			log.Printf("Subscriber %s unsubscribed from %s", subscriberID, subscriptionType)
		}
	}
}

// NotifyQRScan sends QR scan notifications to relevant subscribers
func (rs *RealtimeService) NotifyQRScan(scanResult *QRScanResult, activity *models.Activity, participation *models.Participation) {
	notification := &NotificationMessage{
		Type:      "qr_scanned",
		Timestamp: time.Now(),
		Data: &QRScanNotification{
			ScanResult:    scanResult,
			Activity:      activity,
			Participation: participation,
		},
	}

	rs.broadcastToSubscribers("qr_scanned", notification, func(subscriber *Subscriber) bool {
		// Send to admins who can manage this activity
		return rs.canUserReceiveActivityNotification(subscriber.UserID, activity)
	})
}

// NotifyParticipationUpdate sends participation update notifications
func (rs *RealtimeService) NotifyParticipationUpdate(participation *models.Participation, activity *models.Activity, updateType string) {
	notification := &NotificationMessage{
		Type:      "participation_updated",
		Timestamp: time.Now(),
		Data: &ParticipationUpdateNotification{
			Participation: participation,
			Activity:      activity,
			UpdateType:    updateType,
		},
	}

	rs.broadcastToSubscribers("participation_updated", notification, func(subscriber *Subscriber) bool {
		// Send to the participant and admins who can manage this activity
		if subscriber.UserID == participation.UserID {
			return true
		}
		return rs.canUserReceiveActivityNotification(subscriber.UserID, activity)
	})
}

// NotifyActivityUpdate sends activity update notifications
func (rs *RealtimeService) NotifyActivityUpdate(activity *models.Activity, updateType string) {
	notification := &NotificationMessage{
		Type:      "activity_updated",
		Timestamp: time.Now(),
		Data: &ActivityUpdateNotification{
			Activity:   activity,
			UpdateType: updateType,
		},
	}

	rs.broadcastToSubscribers("activity_updated", notification, func(subscriber *Subscriber) bool {
		return rs.canUserReceiveActivityNotification(subscriber.UserID, activity)
	})
}

// NotifySubscriptionExpiry sends subscription expiry alerts
func (rs *RealtimeService) NotifySubscriptionExpiry(subscription *models.Subscription) {
	notification := &NotificationMessage{
		Type:      "subscription_expiry",
		Timestamp: time.Now(),
		Data:      subscription,
	}

	rs.broadcastToSubscribers("subscription_expiry", notification, func(subscriber *Subscriber) bool {
		// Send to faculty admins of the affected faculty
		var user models.User
		if err := rs.DB.First(&user, subscriber.UserID).Error; err != nil {
			return false
		}

		return (user.Role == models.UserRoleSuperAdmin) ||
			(user.Role == models.UserRoleFacultyAdmin && 
				user.FacultyID != nil && 
				*user.FacultyID == subscription.FacultyID)
	})
}

// Helper methods

func (rs *RealtimeService) broadcastToSubscribers(subscriptionType string, notification *NotificationMessage, filter func(*Subscriber) bool) {
	rs.mutex.RLock()
	subscribers, exists := rs.subscribers[subscriptionType]
	if !exists {
		rs.mutex.RUnlock()
		return
	}

	// Create a copy of subscribers to avoid holding the lock during broadcast
	subscribersCopy := make([]*Subscriber, 0, len(subscribers))
	for _, subscriber := range subscribers {
		if filter == nil || filter(subscriber) {
			subscribersCopy = append(subscribersCopy, subscriber)
		}
	}
	rs.mutex.RUnlock()

	// Broadcast to filtered subscribers
	for _, subscriber := range subscribersCopy {
		select {
		case subscriber.Channel <- notification:
			// Message sent successfully
		default:
			// Channel is full, skip this subscriber
			log.Printf("Subscriber %s channel is full, skipping notification", subscriber.ID)
		}
	}

	log.Printf("Broadcasted %s notification to %d subscribers", subscriptionType, len(subscribersCopy))
}

func (rs *RealtimeService) canUserReceiveActivityNotification(userID uint, activity *models.Activity) bool {
	var user models.User
	if err := rs.DB.First(&user, userID).Error; err != nil {
		return false
	}

	// Super admin can receive all notifications
	if user.Role == models.UserRoleSuperAdmin {
		return true
	}

	// Faculty admin can receive notifications for their faculty
	if user.Role == models.UserRoleFacultyAdmin {
		if activity.FacultyID == nil {
			return true // Cross-faculty activity
		}
		return user.FacultyID != nil && *user.FacultyID == *activity.FacultyID
	}

	// Regular admin can receive notifications for assigned activities
	if user.Role == models.UserRoleRegularAdmin {
		var assignment models.ActivityAssignment
		err := rs.DB.Where("activity_id = ? AND admin_id = ?", activity.ID, userID).First(&assignment).Error
		return err == nil
	}

	return false
}

func (rs *RealtimeService) cleanupSubscriber(ctx context.Context, subscriptionType, subscriberID string) {
	<-ctx.Done()
	rs.Unsubscribe(subscriptionType, subscriberID)
}

func generateSubscriberID() string {
	return time.Now().Format("20060102150405") + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// GetActiveSubscribers returns the count of active subscribers by type
func (rs *RealtimeService) GetActiveSubscribers() map[string]int {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	counts := make(map[string]int)
	for subscriptionType, subscribers := range rs.subscribers {
		counts[subscriptionType] = len(subscribers)
	}

	return counts
}

// BroadcastSystemMessage sends a system-wide message to all subscribers
func (rs *RealtimeService) BroadcastSystemMessage(message string, messageType string) {
	notification := &NotificationMessage{
		Type:      "system_message",
		Timestamp: time.Now(),
		Data: map[string]string{
			"message": message,
			"type":    messageType,
		},
	}

	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	// Broadcast to all subscription types
	for subscriptionType, subscribers := range rs.subscribers {
		for _, subscriber := range subscribers {
			select {
			case subscriber.Channel <- notification:
				// Message sent successfully
			default:
				// Channel is full, skip this subscriber
				log.Printf("Subscriber %s channel is full during system broadcast", subscriber.ID)
			}
		}
		log.Printf("System message broadcasted to %d subscribers in %s", len(subscribers), subscriptionType)
	}
}

// StartHealthCheck starts a goroutine that periodically sends health check messages
func (rs *RealtimeService) StartHealthCheck(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			rs.BroadcastSystemMessage("Health check", "health_check")
		}
	}()
}
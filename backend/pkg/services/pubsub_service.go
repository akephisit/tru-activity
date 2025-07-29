package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kruakemaths/tru-activity/backend/internal/models"
)

type PubSubService struct {
	client     *redis.Client
	publishers map[string]*redis.PubSub
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

type SubscriptionEvent struct {
	Type        string                 `json:"type"`
	Channel     string                 `json:"channel"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        interface{}            `json:"data"`
	Metadata    *SubscriptionMetadata  `json:"metadata,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	InstanceID  string                 `json:"instance_id"`
}

type SubscriptionMetadata struct {
	Source        string `json:"source,omitempty"`
	UserID        *uint  `json:"user_id,omitempty"`
	FacultyID     *uint  `json:"faculty_id,omitempty"`
	ActivityID    *uint  `json:"activity_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

type EventHandler func(*SubscriptionEvent) error

const (
	// Channel patterns for different event types
	PersonalNotificationsChannel    = "personal_notifications:%d"     // user_id
	ActivityUpdatesChannel          = "activity_updates:%d"           // activity_id
	FacultyUpdatesChannel           = "faculty_updates:%d"            // faculty_id
	SystemAlertsChannel             = "system_alerts"
	QRScanEventsChannel             = "qr_scan_events:%d"             // activity_id
	ParticipationEventsChannel      = "participation_events:%d:%d"    // activity_id:user_id
	SubscriptionWarningsChannel     = "subscription_warnings:%d"      // faculty_id
	ActivityAssignmentsChannel      = "activity_assignments:%d"       // user_id
	NewActivitiesChannel            = "new_activities:%d"             // faculty_id
	HeartbeatChannel                = "heartbeat"
	
	// Global channels
	GlobalPersonalNotifications    = "personal_notifications:*"
	GlobalActivityUpdates          = "activity_updates:*"
	GlobalFacultyUpdates           = "faculty_updates:*"
	GlobalQRScanEvents             = "qr_scan_events:*"
	GlobalParticipationEvents      = "participation_events:*"
	GlobalSubscriptionWarnings     = "subscription_warnings:*"
	GlobalActivityAssignments      = "activity_assignments:*"
	GlobalNewActivities            = "new_activities:*"
)

func NewPubSubService(redisURL string, instanceID string) (*PubSubService, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opt)
	
	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	service := &PubSubService{
		client:     client,
		publishers: make(map[string]*redis.PubSub),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Start connection health checker
	go service.startHealthChecker()

	log.Printf("PubSub service initialized with instance ID: %s", instanceID)
	return service, nil
}

// Publish publishes an event to a specific channel
func (ps *PubSubService) Publish(channel string, event *SubscriptionEvent) error {
	event.Timestamp = time.Now()
	event.Channel = channel

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	if err := ps.client.Publish(ps.ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish to channel %s: %v", channel, err)
	}

	log.Printf("Published event %s to channel %s", event.Type, channel)
	return nil
}

// Subscribe subscribes to a channel pattern and handles messages
func (ps *PubSubService) Subscribe(pattern string, handler EventHandler) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if _, exists := ps.publishers[pattern]; exists {
		return fmt.Errorf("already subscribed to pattern: %s", pattern)
	}

	pubsub := ps.client.PSubscribe(ps.ctx, pattern)
	ps.publishers[pattern] = pubsub

	// Start message handling goroutine
	go ps.handleMessages(pattern, pubsub, handler)

	log.Printf("Subscribed to pattern: %s", pattern)
	return nil
}

// Unsubscribe unsubscribes from a channel pattern
func (ps *PubSubService) Unsubscribe(pattern string) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if pubsub, exists := ps.publishers[pattern]; exists {
		if err := pubsub.Close(); err != nil {
			log.Printf("Error closing pubsub for pattern %s: %v", pattern, err)
		}
		delete(ps.publishers, pattern)
		log.Printf("Unsubscribed from pattern: %s", pattern)
	}

	return nil
}

// Event publishing methods for different event types

func (ps *PubSubService) PublishPersonalNotification(userID uint, data interface{}, metadata *SubscriptionMetadata) error {
	channel := fmt.Sprintf(PersonalNotificationsChannel, userID)
	event := &SubscriptionEvent{
		Type:     "personal_notification",
		Data:     data,
		Metadata: metadata,
	}
	return ps.Publish(channel, event)
}

func (ps *PubSubService) PublishActivityUpdate(activityID uint, data interface{}, metadata *SubscriptionMetadata) error {
	channel := fmt.Sprintf(ActivityUpdatesChannel, activityID)
	event := &SubscriptionEvent{
		Type:     "activity_update",
		Data:     data,
		Metadata: metadata,
	}
	return ps.Publish(channel, event)
}

func (ps *PubSubService) PublishFacultyUpdate(facultyID uint, data interface{}, metadata *SubscriptionMetadata) error {
	channel := fmt.Sprintf(FacultyUpdatesChannel, facultyID)
	event := &SubscriptionEvent{
		Type:     "faculty_update",
		Data:     data,
		Metadata: metadata,
	}
	return ps.Publish(channel, event)
}

func (ps *PubSubService) PublishSystemAlert(alert *models.SystemAlert, metadata *SubscriptionMetadata) error {
	event := &SubscriptionEvent{
		Type:     "system_alert",
		Data:     alert,
		Metadata: metadata,
	}
	return ps.Publish(SystemAlertsChannel, event)
}

func (ps *PubSubService) PublishQRScanEvent(activityID uint, scanResult interface{}, metadata *SubscriptionMetadata) error {
	channel := fmt.Sprintf(QRScanEventsChannel, activityID)
	event := &SubscriptionEvent{
		Type:     "qr_scan_event",
		Data:     scanResult,
		Metadata: metadata,
	}
	return ps.Publish(channel, event)
}

func (ps *PubSubService) PublishParticipationEvent(activityID, userID uint, data interface{}, metadata *SubscriptionMetadata) error {
	channel := fmt.Sprintf(ParticipationEventsChannel, activityID, userID)
	event := &SubscriptionEvent{
		Type:     "participation_event",
		Data:     data,
		Metadata: metadata,
	}
	return ps.Publish(channel, event)
}

func (ps *PubSubService) PublishSubscriptionWarning(facultyID uint, warning interface{}, metadata *SubscriptionMetadata) error {
	channel := fmt.Sprintf(SubscriptionWarningsChannel, facultyID)
	event := &SubscriptionEvent{
		Type:     "subscription_warning",
		Data:     warning,
		Metadata: metadata,
	}
	return ps.Publish(channel, event)
}

func (ps *PubSubService) PublishActivityAssignment(userID uint, assignment interface{}, metadata *SubscriptionMetadata) error {
	channel := fmt.Sprintf(ActivityAssignmentsChannel, userID)
	event := &SubscriptionEvent{
		Type:     "activity_assignment",
		Data:     assignment,
		Metadata: metadata,
	}
	return ps.Publish(channel, event)
}

func (ps *PubSubService) PublishNewActivity(facultyID uint, activity interface{}, metadata *SubscriptionMetadata) error {
	channel := fmt.Sprintf(NewActivitiesChannel, facultyID)
	event := &SubscriptionEvent{
		Type:     "new_activity",
		Data:     activity,
		Metadata: metadata,
	}
	return ps.Publish(channel, event)
}

func (ps *PubSubService) PublishHeartbeat() error {
	event := &SubscriptionEvent{
		Type: "heartbeat",
		Data: map[string]interface{}{
			"status":    "alive",
			"timestamp": time.Now(),
		},
	}
	return ps.Publish(HeartbeatChannel, event)
}

// Private methods

func (ps *PubSubService) handleMessages(pattern string, pubsub *redis.PubSub, handler EventHandler) {
	ch := pubsub.Channel()
	
	for {
		select {
		case msg := <-ch:
			if msg == nil {
				log.Printf("Channel closed for pattern: %s", pattern)
				return
			}

			var event SubscriptionEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				log.Printf("Failed to unmarshal message from %s: %v", pattern, err)
				continue
			}

			if err := handler(&event); err != nil {
				log.Printf("Handler error for pattern %s: %v", pattern, err)
			}

		case <-ps.ctx.Done():
			log.Printf("Context cancelled, stopping message handler for pattern: %s", pattern)
			return
		}
	}
}

func (ps *PubSubService) startHealthChecker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ps.client.Ping(ps.ctx).Err(); err != nil {
				log.Printf("Redis health check failed: %v", err)
			}
		case <-ps.ctx.Done():
			return
		}
	}
}

// Close gracefully shuts down the PubSub service
func (ps *PubSubService) Close() error {
	ps.cancel()

	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	// Close all subscriptions
	for pattern, pubsub := range ps.publishers {
		if err := pubsub.Close(); err != nil {
			log.Printf("Error closing pubsub for pattern %s: %v", pattern, err)
		}
	}

	// Close Redis client
	if err := ps.client.Close(); err != nil {
		log.Printf("Error closing Redis client: %v", err)
		return err
	}

	log.Println("PubSub service closed")
	return nil
}

// GetConnectionStats returns statistics about active connections
func (ps *PubSubService) GetConnectionStats() map[string]interface{} {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	return map[string]interface{}{
		"active_subscriptions": len(ps.publishers),
		"redis_connected":      ps.client.Ping(ps.ctx).Err() == nil,
		"patterns":             getMapKeys(ps.publishers),
	}
}

func getMapKeys(m map[string]*redis.PubSub) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
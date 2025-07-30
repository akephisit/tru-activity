package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kruakemaths/tru-activity/backend/internal/database"
	"github.com/kruakemaths/tru-activity/backend/internal/models"
	"github.com/kruakemaths/tru-activity/backend/pkg/auth"
)

type SSEEvent struct {
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Data      interface{}            `json:"data"`
	Metadata  *SSEEventMetadata      `json:"metadata,omitempty"`
}

type SSEEventMetadata struct {
	Source        string `json:"source,omitempty"`
	UserID        string `json:"userId,omitempty"`
	FacultyID     string `json:"facultyId,omitempty"`
	ActivityID    string `json:"activityId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

type SSESubscription struct {
	EventType string                 `json:"eventType"`
	Filter    map[string]interface{} `json:"filter,omitempty"`
}

type SSEClient struct {
	ID            string
	UserID        uint
	FacultyID     *uint
	Role          string
	Channel       chan SSEEvent
	Subscriptions map[string]SSESubscription
	LastSeen      time.Time
	Context       context.Context
	Cancel        context.CancelFunc
	mu            sync.RWMutex
}

type SSEHandler struct {
	db          *database.DB
	jwtService  *auth.JWTService
	clients     map[string]*SSEClient
	broadcast   chan SSEEvent
	register    chan *SSEClient
	unregister  chan *SSEClient
	mu          sync.RWMutex
}

func NewSSEHandler(db *database.DB, jwtService *auth.JWTService) *SSEHandler {
	handler := &SSEHandler{
		db:         db,
		jwtService: jwtService,
		clients:    make(map[string]*SSEClient),
		broadcast:  make(chan SSEEvent, 256),
		register:   make(chan *SSEClient),
		unregister: make(chan *SSEClient),
	}

	// Start the hub goroutine
	go handler.run()

	return handler
}

func (h *SSEHandler) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
			log.Printf("SSE client connected: %s (user: %d)", client.ID, client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Channel)
			}
			h.mu.Unlock()
			log.Printf("SSE client disconnected: %s", client.ID)

		case event := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				if h.shouldReceiveEvent(client, event) {
					select {
					case client.Channel <- event:
					default:
						// Client channel is full, disconnect
						go func(c *SSEClient) {
							h.unregister <- c
						}(client)
					}
				}
			}
			h.mu.RUnlock()

		case <-ticker.C:
			// Clean up inactive clients
			h.cleanupInactiveClients()

			// Send heartbeat
			h.broadcast <- SSEEvent{
				Type:      "heartbeat",
				Timestamp: time.Now().Format(time.RFC3339),
				Data:      "ping",
			}
		}
	}
}

func (h *SSEHandler) shouldReceiveEvent(client *SSEClient, event SSEEvent) bool {
	client.mu.RLock()
	defer client.mu.RUnlock()

	// Check if client has subscription for this event type
	subscription, hasSubscription := client.Subscriptions[event.Type]
	if !hasSubscription && event.Type != "heartbeat" {
		return false
	}

	// Role-based filtering
	switch event.Type {
	case "system_alert":
		// Only admins receive system alerts
		return client.Role != "student"
		
	case "personal_notification":
		// Personal notifications only for the specific user
		return event.Metadata != nil && event.Metadata.UserID == fmt.Sprintf("%d", client.UserID)
		
	case "faculty_update":
		// Faculty updates only for members of that faculty
		if event.Metadata != nil && event.Metadata.FacultyID != "" && client.FacultyID != nil {
			return event.Metadata.FacultyID == fmt.Sprintf("%d", *client.FacultyID)
		}
		return false
		
	case "activity_update", "qr_scan_event", "participation_event":
		// Activity-related events: check permissions
		return h.hasActivityPermission(client, event)
	}

	// Apply subscription filters
	if hasSubscription && subscription.Filter != nil {
		return h.matchesFilter(event, subscription.Filter)
	}

	return true
}

func (h *SSEHandler) hasActivityPermission(client *SSEClient, event SSEEvent) bool {
	if event.Metadata == nil || event.Metadata.ActivityID == "" {
		return false
	}

	// Super admin can see everything
	if client.Role == "super_admin" {
		return true
	}

	// Faculty admin can see activities in their faculty
	if client.Role == "faculty_admin" && client.FacultyID != nil {
		// TODO: Check if activity belongs to this faculty
		return true
	}

	// Regular admin can see assigned activities
	if client.Role == "regular_admin" {
		// TODO: Check if admin is assigned to this activity
		return true
	}

	// Students can see activities they participate in
	if client.Role == "student" {
		// TODO: Check if student participates in this activity
		return true
	}

	return false
}

func (h *SSEHandler) matchesFilter(event SSEEvent, filter map[string]interface{}) bool {
	// Implement filter matching logic
	for key, value := range filter {
		switch key {
		case "activityID":
			if event.Metadata == nil || event.Metadata.ActivityID != fmt.Sprintf("%v", value) {
				return false
			}
		case "facultyID":
			if event.Metadata == nil || event.Metadata.FacultyID != fmt.Sprintf("%v", value) {
				return false
			}
		case "userID":
			if event.Metadata == nil || event.Metadata.UserID != fmt.Sprintf("%v", value) {
				return false
			}
		}
	}
	return true
}

func (h *SSEHandler) cleanupInactiveClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	for id, client := range h.clients {
		if now.Sub(client.LastSeen) > 5*time.Minute {
			delete(h.clients, id)
			close(client.Channel)
			log.Printf("Cleaned up inactive SSE client: %s", id)
		}
	}
}

// HTTP Handlers

func (h *SSEHandler) HandleSSEConnection(c *fiber.Ctx) error {
	// Get auth token
	token := c.Query("token")
	if token == "" {
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		return c.Status(401).JSON(fiber.Map{"error": "Authentication required"})
	}

	// Validate token
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Headers", "Cache-Control")

	// Create client
	ctx, cancel := context.WithCancel(c.Context())
	client := &SSEClient{
		ID:            fmt.Sprintf("%d_%d", claims.UserID, time.Now().UnixNano()),
		UserID:        claims.UserID,
		FacultyID:     claims.FacultyID,
		Role:          claims.Role,
		Channel:       make(chan SSEEvent, 64),
		Subscriptions: make(map[string]SSESubscription),
		LastSeen:      time.Now(),
		Context:       ctx,
		Cancel:        cancel,
	}

	// Register client
	h.register <- client

	// Handle client disconnection
	defer func() {
		cancel()
		h.unregister <- client
	}()

	// Send initial connection event
	initialEvent := SSEEvent{
		Type:      "connection",
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      map[string]interface{}{"status": "connected", "clientId": client.ID},
	}
	
	if err := h.writeSSEEvent(c, initialEvent); err != nil {
		return err
	}

	// Listen for events
	for {
		select {
		case event := <-client.Channel:
			if err := h.writeSSEEvent(c, event); err != nil {
				return err
			}
			
		case <-ctx.Done():
			return nil
		}
	}
}

func (h *SSEHandler) writeSSEEvent(c *fiber.Ctx, event SSEEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Write SSE format
	if event.Type != "heartbeat" {
		if _, err := fmt.Fprintf(c, "event: %s\n", event.Type); err != nil {
			return err
		}
	}
	
	if _, err := fmt.Fprintf(c, "data: %s\n\n", data); err != nil {
		return err
	}

	// Flush the response
	if flusher, ok := c.Response().BodyWriter().(interface{ Flush() }); ok {
		flusher.Flush()
	}

	return nil
}

func (h *SSEHandler) HandleSubscribe(c *fiber.Ctx) error {
	// Get auth token
	token := c.Get("Authorization")
	if !strings.HasPrefix(token, "Bearer ") {
		return c.Status(401).JSON(fiber.Map{"error": "Authentication required"})
	}
	token = strings.TrimPrefix(token, "Bearer ")

	// Validate token
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
	}
	_ = claims // TODO: Use claims for authorization logic

	// Parse subscription request
	var subscription SSESubscription
	if err := c.BodyParser(&subscription); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid subscription data"})
	}

	// Find client and add subscription
	clientID := c.Get("X-Client-ID") // Client should send this header
	if clientID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Client ID required"})
	}

	h.mu.RLock()
	client, exists := h.clients[clientID]
	h.mu.RUnlock()

	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "Client not found"})
	}

	// Add subscription
	client.mu.Lock()
	client.Subscriptions[subscription.EventType] = subscription
	client.LastSeen = time.Now()
	client.mu.Unlock()

	log.Printf("Client %s subscribed to %s", clientID, subscription.EventType)

	return c.JSON(fiber.Map{"status": "subscribed", "eventType": subscription.EventType})
}

func (h *SSEHandler) HandleUnsubscribe(c *fiber.Ctx) error {
	// Similar to subscribe but removes subscription
	token := c.Get("Authorization")
	if !strings.HasPrefix(token, "Bearer ") {
		return c.Status(401).JSON(fiber.Map{"error": "Authentication required"})
	}
	token = strings.TrimPrefix(token, "Bearer ")

	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
	}
	_ = claims // TODO: Use claims for authorization logic

	var subscription SSESubscription
	if err := c.BodyParser(&subscription); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid subscription data"})
	}

	clientID := c.Get("X-Client-ID")
	if clientID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Client ID required"})
	}

	h.mu.RLock()
	client, exists := h.clients[clientID]
	h.mu.RUnlock()

	if !exists {
		return c.Status(404).JSON(fiber.Map{"error": "Client not found"})
	}

	client.mu.Lock()
	delete(client.Subscriptions, subscription.EventType)
	client.LastSeen = time.Now()
	client.mu.Unlock()

	log.Printf("Client %s unsubscribed from %s", clientID, subscription.EventType)

	return c.JSON(fiber.Map{"status": "unsubscribed", "eventType": subscription.EventType})
}

func (h *SSEHandler) HandleHeartbeat(c *fiber.Ctx) error {
	// Simple heartbeat endpoint
	return c.JSON(fiber.Map{"status": "ok", "timestamp": time.Now().Format(time.RFC3339)})
}

// Event publishing methods

func (h *SSEHandler) PublishPersonalNotification(userID uint, data interface{}) {
	event := SSEEvent{
		Type:      "personal_notification",
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      data,
		Metadata: &SSEEventMetadata{
			UserID: fmt.Sprintf("%d", userID),
		},
	}
	h.broadcast <- event
}

func (h *SSEHandler) PublishSystemAlert(severity string, message string, targetRoles []string, facultyID *uint) {
	event := SSEEvent{
		Type:      "system_alert",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"severity":    severity,
			"message":     message,
			"targetRoles": targetRoles,
		},
		Metadata: &SSEEventMetadata{
			Source: "system",
		},
	}
	
	if facultyID != nil {
		event.Metadata.FacultyID = fmt.Sprintf("%d", *facultyID)
	}
	
	h.broadcast <- event
}

func (h *SSEHandler) PublishActivityUpdate(activity *models.Activity, updateType string) {
	event := SSEEvent{
		Type:      "activity_update",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"activity":   activity,
			"updateType": updateType,
		},
		Metadata: &SSEEventMetadata{
			ActivityID: fmt.Sprintf("%d", activity.ID),
			Source:     "activity_service",
		},
	}
	
	if activity.FacultyID != nil {
		event.Metadata.FacultyID = fmt.Sprintf("%d", *activity.FacultyID)
	}
	
	h.broadcast <- event
}

func (h *SSEHandler) PublishQRScanEvent(scanResult interface{}, activityID uint, userID uint) {
	event := SSEEvent{
		Type:      "qr_scan_event",
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      scanResult,
		Metadata: &SSEEventMetadata{
			ActivityID: fmt.Sprintf("%d", activityID),
			UserID:     fmt.Sprintf("%d", userID),
			Source:     "qr_scanner",
		},
	}
	h.broadcast <- event
}

func (h *SSEHandler) PublishParticipationEvent(participation *models.Participation, eventType string) {
	event := SSEEvent{
		Type:      "participation_event",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"participation": participation,
			"eventType":     eventType,
		},
		Metadata: &SSEEventMetadata{
			ActivityID: fmt.Sprintf("%d", participation.ActivityID),
			UserID:     fmt.Sprintf("%d", participation.UserID),
			Source:     "participation_service",
		},
	}
	h.broadcast <- event
}

func (h *SSEHandler) PublishFacultyUpdate(facultyID uint, updateType string, data interface{}) {
	event := SSEEvent{
		Type:      "faculty_update",
		Timestamp: time.Now().Format(time.RFC3339),
		Data: map[string]interface{}{
			"updateType": updateType,
			"data":       data,
		},
		Metadata: &SSEEventMetadata{
			FacultyID: fmt.Sprintf("%d", facultyID),
			Source:    "faculty_service",
		},
	}
	h.broadcast <- event
}

// GetConnectedClients returns the number of connected clients
func (h *SSEHandler) GetConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetClientsByFaculty returns clients for a specific faculty
func (h *SSEHandler) GetClientsByFaculty(facultyID uint) []*SSEClient {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	var clients []*SSEClient
	for _, client := range h.clients {
		if client.FacultyID != nil && *client.FacultyID == facultyID {
			clients = append(clients, client)
		}
	}
	return clients
}
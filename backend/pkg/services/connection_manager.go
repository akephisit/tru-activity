package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kruakemaths/tru-activity/backend/internal/models"
)

type ConnectionManager struct {
	connections     map[string]*Connection
	userConnections map[uint]map[string]*Connection // userID -> connectionID -> connection
	mutex           sync.RWMutex
	pubSub          *PubSubService
	cleanup         *time.Ticker
	ctx             context.Context
	cancel          context.CancelFunc
	maxConnections  int
	idleTimeout     time.Duration
	instanceID      string
}

type Connection struct {
	ID             string                 `json:"id"`
	UserID         uint                   `json:"user_id"`
	User           *models.User           `json:"user"`
	ConnectedAt    time.Time              `json:"connected_at"`
	LastActivity   time.Time              `json:"last_activity"`
	Subscriptions  map[string]*Subscription `json:"subscriptions"`
	Context        context.Context        `json:"-"`
	Cancel         context.CancelFunc     `json:"-"`
	Channel        chan *SubscriptionPayload `json:"-"`
	Metadata       map[string]interface{} `json:"metadata"`
	mutex          sync.RWMutex           `json:"-"`
}

type Subscription struct {
	Type       string                 `json:"type"`
	Filters    map[string]interface{} `json:"filters"`
	CreatedAt  time.Time              `json:"created_at"`
	LastEvent  time.Time              `json:"last_event"`
	EventCount int64                  `json:"event_count"`
}

type SubscriptionPayload struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type ConnectionStats struct {
	TotalConnections    int                    `json:"total_connections"`
	UserConnections     map[uint]int           `json:"user_connections"`
	ActiveSubscriptions map[string]int         `json:"active_subscriptions"`
	InstanceID          string                 `json:"instance_id"`
	Uptime             time.Duration          `json:"uptime"`
	MemoryUsage        int64                  `json:"memory_usage_bytes"`
}

func NewConnectionManager(pubSub *PubSubService, instanceID string, maxConnections int) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	cm := &ConnectionManager{
		connections:     make(map[string]*Connection),
		userConnections: make(map[uint]map[string]*Connection),
		pubSub:          pubSub,
		ctx:             ctx,
		cancel:          cancel,
		maxConnections:  maxConnections,
		idleTimeout:     10 * time.Minute, // Cloud Run friendly timeout
		instanceID:      instanceID,
	}

	// Start cleanup routine for idle connections
	cm.cleanup = time.NewTicker(2 * time.Minute)
	go cm.startCleanupRoutine()

	// Subscribe to global events
	cm.subscribeToGlobalEvents()

	log.Printf("Connection manager initialized for instance %s (max connections: %d)", instanceID, maxConnections)
	return cm
}

// CreateConnection creates a new WebSocket connection
func (cm *ConnectionManager) CreateConnection(userID uint, user *models.User, metadata map[string]interface{}) (*Connection, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Check connection limits
	if len(cm.connections) >= cm.maxConnections {
		return nil, fmt.Errorf("maximum connections reached (%d)", cm.maxConnections)
	}

	// Check per-user connection limit (max 3 per user for mobile + web + tablet)
	if userConns := cm.userConnections[userID]; len(userConns) >= 3 {
		// Close oldest connection
		cm.closeOldestUserConnection(userID)
	}

	ctx, cancel := context.WithCancel(cm.ctx)
	connID := generateConnectionID(userID)

	connection := &Connection{
		ID:            connID,
		UserID:        userID,
		User:          user,
		ConnectedAt:   time.Now(),
		LastActivity:  time.Now(),
		Subscriptions: make(map[string]*Subscription),
		Context:       ctx,
		Cancel:        cancel,
		Channel:       make(chan *SubscriptionPayload, 100), // Buffer for messages
		Metadata:      metadata,
	}

	// Store connection
	cm.connections[connID] = connection
	
	if cm.userConnections[userID] == nil {
		cm.userConnections[userID] = make(map[string]*Connection)
	}
	cm.userConnections[userID][connID] = connection

	// Start connection handler
	go cm.handleConnection(connection)

	log.Printf("Created connection %s for user %d", connID, userID)
	return connection, nil
}

// CloseConnection closes a specific connection
func (cm *ConnectionManager) CloseConnection(connectionID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	connection, exists := cm.connections[connectionID]
	if !exists {
		return
	}

	cm.closeConnection(connection)
}

// Subscribe adds a subscription to a connection
func (cm *ConnectionManager) Subscribe(connectionID, subscriptionType string, filters map[string]interface{}) error {
	cm.mutex.RLock()
	connection, exists := cm.connections[connectionID]
	cm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found: %s", connectionID)
	}

	connection.mutex.Lock()
	defer connection.mutex.Unlock()

	// Update last activity
	connection.LastActivity = time.Now()

	// Add subscription
	connection.Subscriptions[subscriptionType] = &Subscription{
		Type:      subscriptionType,
		Filters:   filters,
		CreatedAt: time.Now(),
	}

	log.Printf("Added subscription %s to connection %s", subscriptionType, connectionID)
	return nil
}

// Unsubscribe removes a subscription from a connection
func (cm *ConnectionManager) Unsubscribe(connectionID, subscriptionType string) error {
	cm.mutex.RLock()
	connection, exists := cm.connections[connectionID]
	cm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("connection not found: %s", connectionID)
	}

	connection.mutex.Lock()
	defer connection.mutex.Unlock()

	delete(connection.Subscriptions, subscriptionType)
	connection.LastActivity = time.Now()

	log.Printf("Removed subscription %s from connection %s", subscriptionType, connectionID)
	return nil
}

// BroadcastToUser sends a message to all connections of a specific user
func (cm *ConnectionManager) BroadcastToUser(userID uint, payload *SubscriptionPayload) {
	cm.mutex.RLock()
	userConns := cm.userConnections[userID]
	cm.mutex.RUnlock()

	if userConns == nil {
		return
	}

	for _, connection := range userConns {
		select {
		case connection.Channel <- payload:
			connection.mutex.Lock()
			connection.LastActivity = time.Now()
			connection.mutex.Unlock()
		default:
			log.Printf("Connection %s channel full, dropping message", connection.ID)
		}
	}
}

// BroadcastToConnections sends a message to connections matching criteria
func (cm *ConnectionManager) BroadcastToConnections(filter func(*Connection) bool, payload *SubscriptionPayload) {
	cm.mutex.RLock()
	connections := make([]*Connection, 0)
	for _, conn := range cm.connections {
		if filter(conn) {
			connections = append(connections, conn)
		}
	}
	cm.mutex.RUnlock()

	for _, connection := range connections {
		select {
		case connection.Channel <- payload:
			connection.mutex.Lock()
			connection.LastActivity = time.Now()
			connection.mutex.Unlock()
		default:
			log.Printf("Connection %s channel full, dropping message", connection.ID)
		}
	}
}

// GetConnectionStats returns statistics about active connections
func (cm *ConnectionManager) GetConnectionStats() *ConnectionStats {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	userCounts := make(map[uint]int)
	subscriptionCounts := make(map[string]int)

	for _, conn := range cm.connections {
		userCounts[conn.UserID]++
		
		conn.mutex.RLock()
		for subType := range conn.Subscriptions {
			subscriptionCounts[subType]++
		}
		conn.mutex.RUnlock()
	}

	return &ConnectionStats{
		TotalConnections:    len(cm.connections),
		UserConnections:     userCounts,
		ActiveSubscriptions: subscriptionCounts,
		InstanceID:          cm.instanceID,
		Uptime:             time.Since(time.Now()), // This should be instance start time
	}
}

// Private methods

func (cm *ConnectionManager) handleConnection(conn *Connection) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Connection handler panic for %s: %v", conn.ID, r)
		}
		cm.cleanupConnection(conn.ID)
	}()

	// Send welcome message
	welcome := &SubscriptionPayload{
		Type:      "connection_established",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"connection_id": conn.ID,
			"instance_id":   cm.instanceID,
		},
	}

	select {
	case conn.Channel <- welcome:
	default:
		log.Printf("Failed to send welcome message to connection %s", conn.ID)
	}

	// Handle connection lifecycle
	for {
		select {
		case <-conn.Context.Done():
			log.Printf("Connection %s context cancelled", conn.ID)
			return
		case <-cm.ctx.Done():
			log.Printf("Connection manager context cancelled, closing connection %s", conn.ID)
			return
		}
	}
}

func (cm *ConnectionManager) startCleanupRoutine() {
	defer cm.cleanup.Stop()

	for {
		select {
		case <-cm.cleanup.C:
			cm.cleanupIdleConnections()
		case <-cm.ctx.Done():
			return
		}
	}
}

func (cm *ConnectionManager) cleanupIdleConnections() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	var toClose []*Connection

	for _, conn := range cm.connections {
		conn.mutex.RLock()
		idle := now.Sub(conn.LastActivity) > cm.idleTimeout
		conn.mutex.RUnlock()

		if idle {
			toClose = append(toClose, conn)
		}
	}

	for _, conn := range toClose {
		log.Printf("Closing idle connection %s (idle for %v)", conn.ID, now.Sub(conn.LastActivity))
		cm.closeConnection(conn)
	}

	if len(toClose) > 0 {
		log.Printf("Cleaned up %d idle connections", len(toClose))
	}
}

func (cm *ConnectionManager) closeOldestUserConnection(userID uint) {
	userConns := cm.userConnections[userID]
	if len(userConns) == 0 {
		return
	}

	var oldest *Connection
	for _, conn := range userConns {
		if oldest == nil || conn.ConnectedAt.Before(oldest.ConnectedAt) {
			oldest = conn
		}
	}

	if oldest != nil {
		log.Printf("Closing oldest connection %s for user %d", oldest.ID, userID)
		cm.closeConnection(oldest)
	}
}

func (cm *ConnectionManager) closeConnection(conn *Connection) {
	// Remove from connections map
	delete(cm.connections, conn.ID)

	// Remove from user connections map
	if userConns := cm.userConnections[conn.UserID]; userConns != nil {
		delete(userConns, conn.ID)
		if len(userConns) == 0 {
			delete(cm.userConnections, conn.UserID)
		}
	}

	// Cancel connection context
	conn.Cancel()

	// Close channel
	close(conn.Channel)
}

func (cm *ConnectionManager) cleanupConnection(connectionID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if conn, exists := cm.connections[connectionID]; exists {
		cm.closeConnection(conn)
	}
}

func (cm *ConnectionManager) subscribeToGlobalEvents() {
	// Subscribe to all event types for this instance
	patterns := []string{
		GlobalPersonalNotifications,
		GlobalActivityUpdates,
		GlobalFacultyUpdates,
		SystemAlertsChannel,
		GlobalQRScanEvents,
		GlobalParticipationEvents,
		GlobalSubscriptionWarnings,
		GlobalActivityAssignments,
		GlobalNewActivities,
		HeartbeatChannel,
	}

	for _, pattern := range patterns {
		cm.pubSub.Subscribe(pattern, cm.handlePubSubEvent)
	}
}

func (cm *ConnectionManager) handlePubSubEvent(event *SubscriptionEvent) error {
	payload := &SubscriptionPayload{
		Type:      event.Type,
		Timestamp: event.Timestamp,
		Data:      event.Data,
		Metadata:  map[string]interface{}{
			"source":    "pubsub",
			"channel":   event.Channel,
			"instance":  event.InstanceID,
		},
	}

	// Apply event-specific routing logic
	switch event.Type {
	case "personal_notification":
		if event.Metadata != nil && event.Metadata.UserID != nil {
			cm.BroadcastToUser(*event.Metadata.UserID, payload)
		}
	case "system_alert":
		// Broadcast to all admin connections
		cm.BroadcastToConnections(func(conn *Connection) bool {
			return conn.User.IsAdmin()
		}, payload)
	default:
		// Generic filtering based on subscriptions
		cm.BroadcastToConnections(func(conn *Connection) bool {
			conn.mutex.RLock()
			defer conn.mutex.RUnlock()
			_, hasSubscription := conn.Subscriptions[event.Type]
			return hasSubscription
		}, payload)
	}

	return nil
}

// Close gracefully shuts down the connection manager
func (cm *ConnectionManager) Close() error {
	cm.cancel()
	cm.cleanup.Stop()

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Close all connections
	for _, conn := range cm.connections {
		cm.closeConnection(conn)
	}

	log.Printf("Connection manager closed for instance %s", cm.instanceID)
	return nil
}

func generateConnectionID(userID uint) string {
	return fmt.Sprintf("conn_%d_%d", userID, time.Now().UnixNano())
}
package resolvers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/kruakemaths/tru-activity/backend/graph/model"
	"github.com/kruakemaths/tru-activity/backend/internal/middleware"
	"github.com/kruakemaths/tru-activity/backend/internal/models"
	"github.com/kruakemaths/tru-activity/backend/pkg/services"
)

type SubscriptionResolver struct {
	ConnectionManager *services.ConnectionManager
	PubSubService     *services.PubSubService
}

func NewSubscriptionResolver(cm *services.ConnectionManager, pubsub *services.PubSubService) *SubscriptionResolver {
	return &SubscriptionResolver{
		ConnectionManager: cm,
		PubSubService:     pubsub,
	}
}

// PersonalNotifications resolves personal notifications for authenticated users
func (r *SubscriptionResolver) PersonalNotifications(ctx context.Context, filter *model.SubscriptionFilter) (<-chan *model.SubscriptionPayload, error) {
	authCtx, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Create connection if not exists
	connection, err := r.ConnectionManager.CreateConnection(authCtx.UserID, authCtx.User, map[string]interface{}{
		"subscription_type": "personal_notifications",
		"remote_addr":       getRemoteAddr(ctx),
		"user_agent":        getUserAgent(ctx),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}

	// Add subscription
	filters := make(map[string]interface{})
	if filter != nil {
		if filter.FacultyID != nil {
			filters["faculty_id"] = *filter.FacultyID
		}
		if filter.ActivityID != nil {
			filters["activity_id"] = *filter.ActivityID
		}
		if filter.Types != nil {
			filters["types"] = filter.Types
		}
	}

	if err := r.ConnectionManager.Subscribe(connection.ID, "personal_notifications", filters); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %v", err)
	}

	// Create output channel
	output := make(chan *model.SubscriptionPayload)

	// Start goroutine to handle messages
	go r.handlePersonalNotifications(ctx, connection, output)

	return output, nil
}

// ActivityUpdates resolves activity-specific updates
func (r *SubscriptionResolver) ActivityUpdates(ctx context.Context, activityID string) (<-chan *model.SubscriptionPayload, error) {
	authCtx, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Verify user can access this activity
	activityIDUint, err := strconv.ParseUint(activityID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid activity ID")
	}

	if !r.canUserAccessActivity(authCtx.User, uint(activityIDUint)) {
		return nil, fmt.Errorf("access denied to activity")
	}

	connection, err := r.ConnectionManager.CreateConnection(authCtx.UserID, authCtx.User, map[string]interface{}{
		"subscription_type": "activity_updates",
		"activity_id":       activityID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}

	filters := map[string]interface{}{
		"activity_id": activityID,
	}

	if err := r.ConnectionManager.Subscribe(connection.ID, "activity_updates", filters); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %v", err)
	}

	output := make(chan *model.SubscriptionPayload)
	go r.handleActivityUpdates(ctx, connection, output)

	return output, nil
}

// FacultyUpdates resolves faculty-wide updates
func (r *SubscriptionResolver) FacultyUpdates(ctx context.Context, facultyID string) (<-chan *model.SubscriptionPayload, error) {
	authCtx, err := middleware.RequireRole(ctx, models.UserRoleSuperAdmin, models.UserRoleFacultyAdmin, models.UserRoleRegularAdmin)
	if err != nil {
		return nil, err
	}

	// Verify user can access this faculty
	facultyIDUint, err := strconv.ParseUint(facultyID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid faculty ID")
	}

	if !r.canUserAccessFaculty(authCtx.User, uint(facultyIDUint)) {
		return nil, fmt.Errorf("access denied to faculty")
	}

	connection, err := r.ConnectionManager.CreateConnection(authCtx.UserID, authCtx.User, map[string]interface{}{
		"subscription_type": "faculty_updates",
		"faculty_id":        facultyID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}

	filters := map[string]interface{}{
		"faculty_id": facultyID,
	}

	if err := r.ConnectionManager.Subscribe(connection.ID, "faculty_updates", filters); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %v", err)
	}

	output := make(chan *model.SubscriptionPayload)
	go r.handleFacultyUpdates(ctx, connection, output)

	return output, nil
}

// SystemAlerts resolves system-wide alerts for admins
func (r *SubscriptionResolver) SystemAlerts(ctx context.Context, filter *model.SubscriptionFilter) (<-chan *model.SubscriptionPayload, error) {
	authCtx, err := middleware.RequireRole(ctx, models.UserRoleSuperAdmin, models.UserRoleFacultyAdmin, models.UserRoleRegularAdmin)
	if err != nil {
		return nil, err
	}

	connection, err := r.ConnectionManager.CreateConnection(authCtx.UserID, authCtx.User, map[string]interface{}{
		"subscription_type": "system_alerts",
		"user_role":         string(authCtx.Role),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}

	filters := make(map[string]interface{})
	if filter != nil {
		if filter.FacultyID != nil {
			filters["faculty_id"] = *filter.FacultyID
		}
		if filter.Types != nil {
			filters["types"] = filter.Types
		}
	}

	if err := r.ConnectionManager.Subscribe(connection.ID, "system_alerts", filters); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %v", err)
	}

	output := make(chan *model.SubscriptionPayload)
	go r.handleSystemAlerts(ctx, connection, output)

	return output, nil
}

// QrScanEvents resolves QR scan events for admins
func (r *SubscriptionResolver) QrScanEvents(ctx context.Context, activityID *string) (<-chan *model.SubscriptionPayload, error) {
	authCtx, err := middleware.RequireRole(ctx, models.UserRoleSuperAdmin, models.UserRoleFacultyAdmin, models.UserRoleRegularAdmin)
	if err != nil {
		return nil, err
	}

	connection, err := r.ConnectionManager.CreateConnection(authCtx.UserID, authCtx.User, map[string]interface{}{
		"subscription_type": "qr_scan_events",
		"activity_id":       activityID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}

	filters := make(map[string]interface{})
	if activityID != nil {
		// Verify user can access this activity
		activityIDUint, err := strconv.ParseUint(*activityID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid activity ID")
		}

		if !r.canUserScanForActivity(authCtx.User, uint(activityIDUint)) {
			return nil, fmt.Errorf("access denied to activity scanning")
		}

		filters["activity_id"] = *activityID
	}

	if err := r.ConnectionManager.Subscribe(connection.ID, "qr_scan_events", filters); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %v", err)
	}

	output := make(chan *model.SubscriptionPayload)
	go r.handleQRScanEvents(ctx, connection, output)

	return output, nil
}

// ParticipationEvents resolves participation events
func (r *SubscriptionResolver) ParticipationEvents(ctx context.Context, activityID *string, userID *string) (<-chan *model.SubscriptionPayload, error) {
	authCtx, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	connection, err := r.ConnectionManager.CreateConnection(authCtx.UserID, authCtx.User, map[string]interface{}{
		"subscription_type": "participation_events",
		"activity_id":       activityID,
		"target_user_id":    userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}

	filters := make(map[string]interface{})
	
	// Role-based filtering
	if authCtx.Role == models.UserRoleStudent {
		// Students can only see their own participation events
		filters["user_id"] = fmt.Sprintf("%d", authCtx.UserID)
	} else {
		// Admins can filter by user and activity
		if userID != nil {
			filters["user_id"] = *userID
		}
		if activityID != nil {
			filters["activity_id"] = *activityID
		}
	}

	if err := r.ConnectionManager.Subscribe(connection.ID, "participation_events", filters); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %v", err)
	}

	output := make(chan *model.SubscriptionPayload)
	go r.handleParticipationEvents(ctx, connection, output)

	return output, nil
}

// SubscriptionWarnings resolves subscription limit warnings
func (r *SubscriptionResolver) SubscriptionWarnings(ctx context.Context, facultyID *string) (<-chan *model.SubscriptionPayload, error) {
	authCtx, err := middleware.RequireRole(ctx, models.UserRoleSuperAdmin, models.UserRoleFacultyAdmin)
	if err != nil {
		return nil, err
	}

	connection, err := r.ConnectionManager.CreateConnection(authCtx.UserID, authCtx.User, map[string]interface{}{
		"subscription_type": "subscription_warnings",
		"faculty_id":        facultyID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}

	filters := make(map[string]interface{})
	if facultyID != nil {
		facultyIDUint, err := strconv.ParseUint(*facultyID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid faculty ID")
		}

		if !r.canUserAccessFaculty(authCtx.User, uint(facultyIDUint)) {
			return nil, fmt.Errorf("access denied to faculty")
		}

		filters["faculty_id"] = *facultyID
	} else if authCtx.Role == models.UserRoleFacultyAdmin && authCtx.FacultyID != nil {
		// Faculty admins can only see their own faculty warnings
		filters["faculty_id"] = fmt.Sprintf("%d", *authCtx.FacultyID)
	}

	if err := r.ConnectionManager.Subscribe(connection.ID, "subscription_warnings", filters); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %v", err)
	}

	output := make(chan *model.SubscriptionPayload)
	go r.handleSubscriptionWarnings(ctx, connection, output)

	return output, nil
}

// ActivityAssignments resolves activity assignments for regular admins
func (r *SubscriptionResolver) ActivityAssignments(ctx context.Context) (<-chan *model.SubscriptionPayload, error) {
	authCtx, err := middleware.RequireRole(ctx, models.UserRoleRegularAdmin)
	if err != nil {
		return nil, err
	}

	connection, err := r.ConnectionManager.CreateConnection(authCtx.UserID, authCtx.User, map[string]interface{}{
		"subscription_type": "activity_assignments",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}

	filters := map[string]interface{}{
		"admin_id": fmt.Sprintf("%d", authCtx.UserID),
	}

	if err := r.ConnectionManager.Subscribe(connection.ID, "activity_assignments", filters); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %v", err)
	}

	output := make(chan *model.SubscriptionPayload)
	go r.handleActivityAssignments(ctx, connection, output)

	return output, nil
}

// NewActivities resolves new activity notifications
func (r *SubscriptionResolver) NewActivities(ctx context.Context, facultyID *string) (<-chan *model.SubscriptionPayload, error) {
	authCtx, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	connection, err := r.ConnectionManager.CreateConnection(authCtx.UserID, authCtx.User, map[string]interface{}{
		"subscription_type": "new_activities",
		"faculty_id":        facultyID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}

	filters := make(map[string]interface{})
	if facultyID != nil {
		facultyIDUint, err := strconv.ParseUint(*facultyID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid faculty ID")
		}

		if !r.canUserAccessFaculty(authCtx.User, uint(facultyIDUint)) {
			return nil, fmt.Errorf("access denied to faculty")
		}

		filters["faculty_id"] = *facultyID
	} else if authCtx.FacultyID != nil {
		// Users automatically see activities from their faculty
		filters["faculty_id"] = fmt.Sprintf("%d", *authCtx.FacultyID)
	}

	if err := r.ConnectionManager.Subscribe(connection.ID, "new_activities", filters); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %v", err)
	}

	output := make(chan *model.SubscriptionPayload)
	go r.handleNewActivities(ctx, connection, output)

	return output, nil
}

// Heartbeat resolves connection heartbeat
func (r *SubscriptionResolver) Heartbeat(ctx context.Context) (<-chan string, error) {
	authCtx, err := middleware.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}

	connection, err := r.ConnectionManager.CreateConnection(authCtx.UserID, authCtx.User, map[string]interface{}{
		"subscription_type": "heartbeat",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %v", err)
	}

	if err := r.ConnectionManager.Subscribe(connection.ID, "heartbeat", nil); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %v", err)
	}

	output := make(chan string)
	go r.handleHeartbeat(ctx, connection, output)

	return output, nil
}

// Helper methods for message handling

func (r *SubscriptionResolver) handlePersonalNotifications(ctx context.Context, conn *services.Connection, output chan<- *model.SubscriptionPayload) {
	defer close(output)

	for {
		select {
		case msg := <-conn.Channel:
			if r.shouldReceivePersonalNotification(conn.User, msg) {
				output <- convertToGraphQLPayload(msg)
			}
		case <-ctx.Done():
			return
		case <-conn.Context.Done():
			return
		}
	}
}

func (r *SubscriptionResolver) handleActivityUpdates(ctx context.Context, conn *services.Connection, output chan<- *model.SubscriptionPayload) {
	defer close(output)

	for {
		select {
		case msg := <-conn.Channel:
			output <- convertToGraphQLPayload(msg)
		case <-ctx.Done():
			return
		case <-conn.Context.Done():
			return
		}
	}
}

func (r *SubscriptionResolver) handleFacultyUpdates(ctx context.Context, conn *services.Connection, output chan<- *model.SubscriptionPayload) {
	defer close(output)

	for {
		select {
		case msg := <-conn.Channel:
			output <- convertToGraphQLPayload(msg)
		case <-ctx.Done():
			return
		case <-conn.Context.Done():
			return
		}
	}
}

func (r *SubscriptionResolver) handleSystemAlerts(ctx context.Context, conn *services.Connection, output chan<- *model.SubscriptionPayload) {
	defer close(output)

	for {
		select {
		case msg := <-conn.Channel:
			if r.shouldReceiveSystemAlert(conn.User, msg) {
				output <- convertToGraphQLPayload(msg)
			}
		case <-ctx.Done():
			return
		case <-conn.Context.Done():
			return
		}
	}
}

func (r *SubscriptionResolver) handleQRScanEvents(ctx context.Context, conn *services.Connection, output chan<- *model.SubscriptionPayload) {
	defer close(output)

	for {
		select {
		case msg := <-conn.Channel:
			output <- convertToGraphQLPayload(msg)
		case <-ctx.Done():
			return
		case <-conn.Context.Done():
			return
		}
	}
}

func (r *SubscriptionResolver) handleParticipationEvents(ctx context.Context, conn *services.Connection, output chan<- *model.SubscriptionPayload) {
	defer close(output)

	for {
		select {
		case msg := <-conn.Channel:
			output <- convertToGraphQLPayload(msg)
		case <-ctx.Done():
			return
		case <-conn.Context.Done():
			return
		}
	}
}

func (r *SubscriptionResolver) handleSubscriptionWarnings(ctx context.Context, conn *services.Connection, output chan<- *model.SubscriptionPayload) {
	defer close(output)

	for {
		select {
		case msg := <-conn.Channel:
			output <- convertToGraphQLPayload(msg)
		case <-ctx.Done():
			return
		case <-conn.Context.Done():
			return
		}
	}
}

func (r *SubscriptionResolver) handleActivityAssignments(ctx context.Context, conn *services.Connection, output chan<- *model.SubscriptionPayload) {
	defer close(output)

	for {
		select {
		case msg := <-conn.Channel:
			output <- convertToGraphQLPayload(msg)
		case <-ctx.Done():
			return
		case <-conn.Context.Done():
			return
		}
	}
}

func (r *SubscriptionResolver) handleNewActivities(ctx context.Context, conn *services.Connection, output chan<- *model.SubscriptionPayload) {
	defer close(output)

	for {
		select {
		case msg := <-conn.Channel:
			output <- convertToGraphQLPayload(msg)
		case <-ctx.Done():
			return
		case <-conn.Context.Done():
			return
		}
	}
}

func (r *SubscriptionResolver) handleHeartbeat(ctx context.Context, conn *services.Connection, output chan<- string) {
	defer close(output)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			output <- fmt.Sprintf("heartbeat_%d", time.Now().Unix())
		case <-ctx.Done():
			return
		case <-conn.Context.Done():
			return
		}
	}
}

// Helper methods for permission checking

func (r *SubscriptionResolver) canUserAccessActivity(user *models.User, activityID uint) bool {
	// Implementation would check if user can access the activity
	// This is a simplified version
	return user.IsAdmin() || user.Role == models.UserRoleStudent
}

func (r *SubscriptionResolver) canUserAccessFaculty(user *models.User, facultyID uint) bool {
	if user.Role == models.UserRoleSuperAdmin {
		return true
	}
	return user.FacultyID != nil && *user.FacultyID == facultyID
}

func (r *SubscriptionResolver) canUserScanForActivity(user *models.User, activityID uint) bool {
	// Implementation would check if user can scan QR codes for this activity
	return user.IsAdmin()
}

func (r *SubscriptionResolver) shouldReceivePersonalNotification(user *models.User, msg *services.SubscriptionPayload) bool {
	// Implement filtering logic based on user role and message content
	return true
}

func (r *SubscriptionResolver) shouldReceiveSystemAlert(user *models.User, msg *services.SubscriptionPayload) bool {
	// Implement filtering logic for system alerts based on user role
	return user.IsAdmin()
}

// Utility functions

func convertToGraphQLPayload(payload *services.SubscriptionPayload) *model.SubscriptionPayload {
	return &model.SubscriptionPayload{
		Type:      payload.Type,
		Timestamp: payload.Timestamp,
		Data:      nil, // This would need proper type conversion based on payload type
		Metadata: &model.SubscriptionMetadata{
			Source: getStringFromMap(payload.Metadata, "source"),
		},
	}
}

func getStringFromMap(m map[string]interface{}, key string) *string {
	if m != nil {
		if val, ok := m[key].(string); ok {
			return &val
		}
	}
	return nil
}

func getRemoteAddr(ctx context.Context) string {
	// Extract remote address from context
	return "unknown"
}

func getUserAgent(ctx context.Context) string {
	// Extract user agent from context
	return "unknown"
}
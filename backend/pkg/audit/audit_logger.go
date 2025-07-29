package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// AuditLogger handles comprehensive audit logging
type AuditLogger struct {
	db          *gorm.DB
	redisClient *redis.Client
}

// AuditEvent represents an audit log entry
type AuditEvent struct {
	ID            string                 `json:"id" gorm:"primaryKey"`
	UserID        string                 `json:"user_id" gorm:"index"`
	UserRole      string                 `json:"user_role"`
	Action        string                 `json:"action" gorm:"index"`
	Resource      string                 `json:"resource" gorm:"index"`
	ResourceID    string                 `json:"resource_id" gorm:"index"`
	FacultyID     string                 `json:"faculty_id" gorm:"index"`
	Details       map[string]interface{} `json:"details" gorm:"type:jsonb"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	Timestamp     time.Time              `json:"timestamp" gorm:"index"`
	Success       bool                   `json:"success"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	SessionID     string                 `json:"session_id"`
	Severity      string                 `json:"severity" gorm:"index"` // INFO, WARN, ERROR, CRITICAL
	Category      string                 `json:"category" gorm:"index"` // ADMIN, SECURITY, DATA, SYSTEM
	CreatedAt     time.Time              `json:"created_at"`
}

// SecurityEvent represents security-specific events
type SecurityEvent struct {
	ID            string                 `json:"id" gorm:"primaryKey"`
	EventType     string                 `json:"event_type" gorm:"index"`
	UserID        string                 `json:"user_id" gorm:"index"`
	IPAddress     string                 `json:"ip_address" gorm:"index"`
	UserAgent     string                 `json:"user_agent"`
	Details       map[string]interface{} `json:"details" gorm:"type:jsonb"`
	RiskLevel     string                 `json:"risk_level" gorm:"index"` // LOW, MEDIUM, HIGH, CRITICAL
	Blocked       bool                   `json:"blocked"`
	Timestamp     time.Time              `json:"timestamp" gorm:"index"`
	CreatedAt     time.Time              `json:"created_at"`
}

// PerformanceMetric represents performance monitoring data
type PerformanceMetric struct {
	ID            string                 `json:"id" gorm:"primaryKey"`
	MetricType    string                 `json:"metric_type" gorm:"index"`
	Operation     string                 `json:"operation" gorm:"index"`
	Duration      int64                  `json:"duration"` // milliseconds
	QueryCount    int                    `json:"query_count"`
	CacheHits     int                    `json:"cache_hits"`
	CacheMisses   int                    `json:"cache_misses"`
	UserID        string                 `json:"user_id" gorm:"index"`
	Details       map[string]interface{} `json:"details" gorm:"type:jsonb"`
	Timestamp     time.Time              `json:"timestamp" gorm:"index"`
	CreatedAt     time.Time              `json:"created_at"`
}

// Constants for audit logging
const (
	// Actions
	ActionCreate = "CREATE"
	ActionRead   = "READ"
	ActionUpdate = "UPDATE"
	ActionDelete = "DELETE"
	ActionLogin  = "LOGIN"
	ActionLogout = "LOGOUT"
	ActionScan   = "SCAN_QR"
	ActionExport = "EXPORT"
	
	// Resources
	ResourceUser         = "USER"
	ResourceActivity     = "ACTIVITY"
	ResourceFaculty      = "FACULTY"
	ResourceDepartment   = "DEPARTMENT"
	ResourceParticipation = "PARTICIPATION"
	ResourceSubscription = "SUBSCRIPTION"
	ResourceQRCode       = "QR_CODE"
	ResourceReport       = "REPORT"
	
	// Severities
	SeverityInfo     = "INFO"
	SeverityWarn     = "WARN"
	SeverityError    = "ERROR"
	SeverityCritical = "CRITICAL"
	
	// Categories
	CategoryAdmin    = "ADMIN"
	CategorySecurity = "SECURITY"
	CategoryData     = "DATA"
	CategorySystem   = "SYSTEM"
	
	// Security Event Types
	SecurityEventLoginFailure        = "LOGIN_FAILURE"
	SecurityEventRateLimitExceeded   = "RATE_LIMIT_EXCEEDED"
	SecurityEventSuspiciousActivity  = "SUSPICIOUS_ACTIVITY"
	SecurityEventUnauthorizedAccess  = "UNAUTHORIZED_ACCESS"
	SecurityEventDataBreach          = "DATA_BREACH"
	SecurityEventQRTampering         = "QR_TAMPERING"
	SecurityEventBruteForce          = "BRUTE_FORCE"
	SecurityEventPrivilegeEscalation = "PRIVILEGE_ESCALATION"
	
	// Risk Levels
	RiskLevelLow      = "LOW"
	RiskLevelMedium   = "MEDIUM"
	RiskLevelHigh     = "HIGH"
	RiskLevelCritical = "CRITICAL"
)

// NewAuditLogger creates a new audit logger
func NewAuditLogger(db *gorm.DB, redisClient *redis.Client) *AuditLogger {
	return &AuditLogger{
		db:          db,
		redisClient: redisClient,
	}
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(ctx context.Context, event *AuditEvent) error {
	// Set default values
	if event.ID == "" {
		event.ID = generateID()
	}
	event.CreatedAt = time.Now()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	// Extract user info from context if not provided
	if event.UserID == "" {
		event.UserID = getUserIDFromContext(ctx)
	}
	if event.UserRole == "" {
		event.UserRole = getUserRoleFromContext(ctx)
	}
	if event.FacultyID == "" {
		event.FacultyID = getFacultyIDFromContext(ctx)
	}
	if event.IPAddress == "" {
		event.IPAddress = getIPFromContext(ctx)
	}
	if event.UserAgent == "" {
		event.UserAgent = getUserAgentFromContext(ctx)
	}
	if event.SessionID == "" {
		event.SessionID = getSessionIDFromContext(ctx)
	}
	
	// Store in database
	if err := al.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("failed to store audit event: %v", err)
	}
	
	// Also store in Redis for real-time monitoring
	go al.storeInRedis(ctx, event)
	
	// Check for security patterns
	go al.analyzeSecurityPatterns(ctx, event)
	
	return nil
}

// LogSecurityEvent logs a security-specific event
func (al *AuditLogger) LogSecurityEvent(ctx context.Context, event *SecurityEvent) error {
	if event.ID == "" {
		event.ID = generateID()
	}
	event.CreatedAt = time.Now()
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	// Extract context info if not provided
	if event.UserID == "" {
		event.UserID = getUserIDFromContext(ctx)
	}
	if event.IPAddress == "" {
		event.IPAddress = getIPFromContext(ctx)
	}
	if event.UserAgent == "" {
		event.UserAgent = getUserAgentFromContext(ctx)
	}
	
	// Store in database
	if err := al.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("failed to store security event: %v", err)
	}
	
	// Store in Redis for real-time alerts
	go al.storeSecurityEventInRedis(ctx, event)
	
	// Trigger alerts for high-risk events
	if event.RiskLevel == RiskLevelHigh || event.RiskLevel == RiskLevelCritical {
		go al.triggerSecurityAlert(ctx, event)
	}
	
	return nil
}

// LogPerformanceMetric logs performance metrics
func (al *AuditLogger) LogPerformanceMetric(ctx context.Context, metric *PerformanceMetric) error {
	if metric.ID == "" {
		metric.ID = generateID()
	}
	metric.CreatedAt = time.Now()
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}
	
	if metric.UserID == "" {
		metric.UserID = getUserIDFromContext(ctx)
	}
	
	// Store in database (async to avoid impacting performance)
	go func() {
		if err := al.db.Create(metric).Error; err != nil {
			fmt.Printf("Failed to store performance metric: %v\n", err)
		}
	}()
	
	// Update real-time performance metrics in Redis
	go al.updatePerformanceMetrics(ctx, metric)
	
	return nil
}

// Specific logging methods for common operations

// LogAdminAction logs administrative actions
func (al *AuditLogger) LogAdminAction(ctx context.Context, action, resource, resourceID string, details map[string]interface{}, success bool, errorMsg string) error {
	event := &AuditEvent{
		Action:       action,
		Resource:     resource,
		ResourceID:   resourceID,
		Details:      details,
		Success:      success,
		ErrorMessage: errorMsg,
		Severity:     SeverityInfo,
		Category:     CategoryAdmin,
	}
	
	if !success {
		event.Severity = SeverityError
	}
	
	return al.LogEvent(ctx, event)
}

// LogDataAccess logs data access events
func (al *AuditLogger) LogDataAccess(ctx context.Context, resource, resourceID string, details map[string]interface{}) error {
	event := &AuditEvent{
		Action:     ActionRead,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		Success:    true,
		Severity:   SeverityInfo,
		Category:   CategoryData,
	}
	
	return al.LogEvent(ctx, event)
}

// LogQRScan logs QR code scanning events
func (al *AuditLogger) LogQRScan(ctx context.Context, studentID, activityID, scannerID string, success bool, errorMsg string, details map[string]interface{}) error {
	event := &AuditEvent{
		Action:       ActionScan,
		Resource:     ResourceQRCode,
		ResourceID:   studentID,
		Details:      details,
		Success:      success,
		ErrorMessage: errorMsg,
		Severity:     SeverityInfo,
		Category:     CategorySecurity,
	}
	
	if details == nil {
		details = make(map[string]interface{})
	}
	details["activity_id"] = activityID
	details["scanner_id"] = scannerID
	
	if !success {
		event.Severity = SeverityWarn
	}
	
	return al.LogEvent(ctx, event)
}

// LogLogin logs user login events
func (al *AuditLogger) LogLogin(ctx context.Context, userID, email string, success bool, errorMsg string) error {
	event := &AuditEvent{
		Action:       ActionLogin,
		Resource:     ResourceUser,
		ResourceID:   userID,
		Details:      map[string]interface{}{"email": email},
		Success:      success,
		ErrorMessage: errorMsg,
		Severity:     SeverityInfo,
		Category:     CategorySecurity,
	}
	
	if !success {
		event.Severity = SeverityWarn
		// Also log as security event for failed logins
		securityEvent := &SecurityEvent{
			EventType: SecurityEventLoginFailure,
			Details:   map[string]interface{}{"email": email, "error": errorMsg},
			RiskLevel: RiskLevelMedium,
		}
		go al.LogSecurityEvent(ctx, securityEvent)
	}
	
	return al.LogEvent(ctx, event)
}

// Query methods for audit logs

// GetAuditLogs retrieves audit logs with filtering
func (al *AuditLogger) GetAuditLogs(ctx context.Context, filters AuditFilters, limit, offset int) ([]AuditEvent, int64, error) {
	var events []AuditEvent
	var total int64
	
	query := al.db.WithContext(ctx).Model(&AuditEvent{})
	
	// Apply filters
	if filters.UserID != "" {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.Resource != "" {
		query = query.Where("resource = ?", filters.Resource)
	}
	if filters.FacultyID != "" {
		query = query.Where("faculty_id = ?", filters.FacultyID)
	}
	if filters.Severity != "" {
		query = query.Where("severity = ?", filters.Severity)
	}
	if filters.Category != "" {
		query = query.Where("category = ?", filters.Category)
	}
	if !filters.StartDate.IsZero() {
		query = query.Where("timestamp >= ?", filters.StartDate)
	}
	if !filters.EndDate.IsZero() {
		query = query.Where("timestamp <= ?", filters.EndDate)
	}
	if filters.Success != nil {
		query = query.Where("success = ?", *filters.Success)
	}
	
	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// Get events with pagination
	if err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&events).Error; err != nil {
		return nil, 0, err
	}
	
	return events, total, nil
}

// GetSecurityEvents retrieves security events
func (al *AuditLogger) GetSecurityEvents(ctx context.Context, filters SecurityFilters, limit, offset int) ([]SecurityEvent, int64, error) {
	var events []SecurityEvent
	var total int64
	
	query := al.db.WithContext(ctx).Model(&SecurityEvent{})
	
	// Apply filters
	if filters.EventType != "" {
		query = query.Where("event_type = ?", filters.EventType)
	}
	if filters.UserID != "" {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.IPAddress != "" {
		query = query.Where("ip_address = ?", filters.IPAddress)
	}
	if filters.RiskLevel != "" {
		query = query.Where("risk_level = ?", filters.RiskLevel)
	}
	if !filters.StartDate.IsZero() {
		query = query.Where("timestamp >= ?", filters.StartDate)
	}
	if !filters.EndDate.IsZero() {
		query = query.Where("timestamp <= ?", filters.EndDate)
	}
	
	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// Get events with pagination
	if err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&events).Error; err != nil {
		return nil, 0, err
	}
	
	return events, total, nil
}

// Analytics methods

// GetAuditAnalytics provides audit analytics
func (al *AuditLogger) GetAuditAnalytics(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})
	
	// Total events
	var totalEvents int64
	al.db.WithContext(ctx).Model(&AuditEvent{}).
		Where("timestamp BETWEEN ? AND ?", startDate, endDate).
		Count(&totalEvents)
	analytics["total_events"] = totalEvents
	
	// Events by action
	var actionStats []struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}
	al.db.WithContext(ctx).Model(&AuditEvent{}).
		Select("action, COUNT(*) as count").
		Where("timestamp BETWEEN ? AND ?", startDate, endDate).
		Group("action").
		Find(&actionStats)
	analytics["events_by_action"] = actionStats
	
	// Events by resource
	var resourceStats []struct {
		Resource string `json:"resource"`
		Count    int64  `json:"count"`
	}
	al.db.WithContext(ctx).Model(&AuditEvent{}).
		Select("resource, COUNT(*) as count").
		Where("timestamp BETWEEN ? AND ?", startDate, endDate).
		Group("resource").
		Find(&resourceStats)
	analytics["events_by_resource"] = resourceStats
	
	// Events by severity
	var severityStats []struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	al.db.WithContext(ctx).Model(&AuditEvent{}).
		Select("severity, COUNT(*) as count").
		Where("timestamp BETWEEN ? AND ?", startDate, endDate).
		Group("severity").
		Find(&severityStats)
	analytics["events_by_severity"] = severityStats
	
	// Top users by activity
	var userStats []struct {
		UserID string `json:"user_id"`
		Count  int64  `json:"count"`
	}
	al.db.WithContext(ctx).Model(&AuditEvent{}).
		Select("user_id, COUNT(*) as count").
		Where("timestamp BETWEEN ? AND ? AND user_id != ''", startDate, endDate).
		Group("user_id").
		Order("count DESC").
		Limit(10).
		Find(&userStats)
	analytics["top_users"] = userStats
	
	// Failed operations
	var failedEvents int64
	al.db.WithContext(ctx).Model(&AuditEvent{}).
		Where("timestamp BETWEEN ? AND ? AND success = false", startDate, endDate).
		Count(&failedEvents)
	analytics["failed_events"] = failedEvents
	
	// Security events summary
	var securityEventCount int64
	al.db.WithContext(ctx).Model(&SecurityEvent{}).
		Where("timestamp BETWEEN ? AND ?", startDate, endDate).
		Count(&securityEventCount)
	analytics["security_events"] = securityEventCount
	
	return analytics, nil
}

// Real-time monitoring methods

// storeInRedis stores audit events in Redis for real-time monitoring
func (al *AuditLogger) storeInRedis(ctx context.Context, event *AuditEvent) {
	// Store recent events for real-time dashboard
	eventData, _ := json.Marshal(event)
	
	pipe := al.redisClient.Pipeline()
	
	// Add to recent events list
	pipe.LPush(ctx, "audit:recent_events", eventData)
	pipe.LTrim(ctx, "audit:recent_events", 0, 999) // Keep last 1000 events
	pipe.Expire(ctx, "audit:recent_events", 24*time.Hour)
	
	// Update counters
	today := time.Now().Format("2006-01-02")
	pipe.Incr(ctx, fmt.Sprintf("audit:daily_count:%s", today))
	pipe.Incr(ctx, fmt.Sprintf("audit:action_count:%s:%s", event.Action, today))
	pipe.Incr(ctx, fmt.Sprintf("audit:resource_count:%s:%s", event.Resource, today))
	
	if !event.Success {
		pipe.Incr(ctx, fmt.Sprintf("audit:failed_count:%s", today))
	}
	
	// Set expiry for counters
	pipe.Expire(ctx, fmt.Sprintf("audit:daily_count:%s", today), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("audit:action_count:%s:%s", event.Action, today), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("audit:resource_count:%s:%s", event.Resource, today), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("audit:failed_count:%s", today), 7*24*time.Hour)
	
	pipe.Exec(ctx)
}

// storeSecurityEventInRedis stores security events in Redis
func (al *AuditLogger) storeSecurityEventInRedis(ctx context.Context, event *SecurityEvent) {
	eventData, _ := json.Marshal(event)
	
	pipe := al.redisClient.Pipeline()
	
	// Add to security events list
	pipe.LPush(ctx, "security:recent_events", eventData)
	pipe.LTrim(ctx, "security:recent_events", 0, 499) // Keep last 500 events
	pipe.Expire(ctx, "security:recent_events", 24*time.Hour)
	
	// Update security counters
	today := time.Now().Format("2006-01-02")
	pipe.Incr(ctx, fmt.Sprintf("security:daily_count:%s", today))
	pipe.Incr(ctx, fmt.Sprintf("security:type_count:%s:%s", event.EventType, today))
	pipe.Incr(ctx, fmt.Sprintf("security:risk_count:%s:%s", event.RiskLevel, today))
	
	// Set expiry for counters
	pipe.Expire(ctx, fmt.Sprintf("security:daily_count:%s", today), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("security:type_count:%s:%s", event.EventType, today), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("security:risk_count:%s:%s", event.RiskLevel, today), 7*24*time.Hour)
	
	pipe.Exec(ctx)
}

// updatePerformanceMetrics updates real-time performance metrics
func (al *AuditLogger) updatePerformanceMetrics(ctx context.Context, metric *PerformanceMetric) {
	pipe := al.redisClient.Pipeline()
	
	// Rolling averages for performance metrics
	today := time.Now().Format("2006-01-02")
	hour := time.Now().Format("2006-01-02:15")
	
	// Update counters
	pipe.Incr(ctx, fmt.Sprintf("perf:count:%s:%s", metric.Operation, today))
	pipe.IncrBy(ctx, fmt.Sprintf("perf:duration:%s:%s", metric.Operation, today), metric.Duration)
	pipe.IncrBy(ctx, fmt.Sprintf("perf:queries:%s:%s", metric.Operation, today), int64(metric.QueryCount))
	
	// Hourly metrics for more granular analysis
	pipe.Incr(ctx, fmt.Sprintf("perf:count:%s:%s", metric.Operation, hour))
	pipe.IncrBy(ctx, fmt.Sprintf("perf:duration:%s:%s", metric.Operation, hour), metric.Duration)
	
	// Set expiry
	pipe.Expire(ctx, fmt.Sprintf("perf:count:%s:%s", metric.Operation, today), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("perf:duration:%s:%s", metric.Operation, today), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("perf:queries:%s:%s", metric.Operation, today), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("perf:count:%s:%s", metric.Operation, hour), 48*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("perf:duration:%s:%s", metric.Operation, hour), 48*time.Hour)
	
	pipe.Exec(ctx)
}

// Security analysis methods

// analyzeSecurityPatterns analyzes audit events for security patterns
func (al *AuditLogger) analyzeSecurityPatterns(ctx context.Context, event *AuditEvent) {
	// Check for brute force patterns
	if event.Action == ActionLogin && !event.Success {
		go al.checkBruteForce(ctx, event)
	}
	
	// Check for privilege escalation
	if event.Action == ActionUpdate && event.Resource == ResourceUser {
		go al.checkPrivilegeEscalation(ctx, event)
	}
	
	// Check for unusual access patterns
	if event.Action == ActionRead {
		go al.checkUnusualAccess(ctx, event)
	}
}

// checkBruteForce checks for brute force attack patterns
func (al *AuditLogger) checkBruteForce(ctx context.Context, event *AuditEvent) {
	// Count failed login attempts from the same IP in the last hour
	oneHourAgo := time.Now().Add(-time.Hour)
	
	var failedAttempts int64
	al.db.WithContext(ctx).Model(&AuditEvent{}).
		Where("action = ? AND success = false AND ip_address = ? AND timestamp > ?", 
			ActionLogin, event.IPAddress, oneHourAgo).
		Count(&failedAttempts)
	
	if failedAttempts >= 5 { // Threshold for brute force
		securityEvent := &SecurityEvent{
			EventType: SecurityEventBruteForce,
			Details: map[string]interface{}{
				"failed_attempts": failedAttempts,
				"time_window":     "1 hour",
			},
			RiskLevel: RiskLevelHigh,
		}
		al.LogSecurityEvent(ctx, securityEvent)
	}
}

// checkPrivilegeEscalation checks for privilege escalation attempts
func (al *AuditLogger) checkPrivilegeEscalation(ctx context.Context, event *AuditEvent) {
	// Check if a non-admin user is trying to modify admin privileges
	if details, ok := event.Details["role_changes"]; ok {
		if roleChanges, ok := details.(map[string]interface{}); ok {
			if newRole, exists := roleChanges["new_role"]; exists {
				if strings.Contains(fmt.Sprintf("%v", newRole), "ADMIN") && 
				   event.UserRole == "STUDENT" {
					securityEvent := &SecurityEvent{
						EventType: SecurityEventPrivilegeEscalation,
						Details:   event.Details,
						RiskLevel: RiskLevelCritical,
					}
					al.LogSecurityEvent(ctx, securityEvent)
				}
			}
		}
	}
}

// checkUnusualAccess checks for unusual access patterns
func (al *AuditLogger) checkUnusualAccess(ctx context.Context, event *AuditEvent) {
	// Check for access from unusual locations or times
	// This is a simplified implementation
	
	// Count access attempts from this IP in the last 24 hours
	yesterday := time.Now().Add(-24 * time.Hour)
	
	var accessCount int64
	al.db.WithContext(ctx).Model(&AuditEvent{}).
		Where("user_id = ? AND ip_address = ? AND timestamp > ?", 
			event.UserID, event.IPAddress, yesterday).
		Count(&accessCount)
	
	// If this is the first time accessing from this IP, flag as suspicious
	if accessCount == 1 && event.UserID != "" {
		securityEvent := &SecurityEvent{
			EventType: SecurityEventSuspiciousActivity,
			Details: map[string]interface{}{
				"reason": "Access from new IP address",
				"resource": event.Resource,
			},
			RiskLevel: RiskLevelMedium,
		}
		al.LogSecurityEvent(ctx, securityEvent)
	}
}

// triggerSecurityAlert triggers alerts for high-risk security events
func (al *AuditLogger) triggerSecurityAlert(ctx context.Context, event *SecurityEvent) {
	// Store alert in Redis for immediate processing
	alertData := map[string]interface{}{
		"event_id":   event.ID,
		"event_type": event.EventType,
		"risk_level": event.RiskLevel,
		"user_id":    event.UserID,
		"ip_address": event.IPAddress,
		"timestamp":  event.Timestamp.Unix(),
		"details":    event.Details,
	}
	
	alertJSON, _ := json.Marshal(alertData)
	
	// Add to alerts queue
	al.redisClient.LPush(ctx, "security:alerts", alertJSON)
	al.redisClient.Expire(ctx, "security:alerts", 24*time.Hour)
	
	// Publish to real-time notification system
	al.redisClient.Publish(ctx, "security_alerts", alertJSON)
	
	fmt.Printf("SECURITY ALERT: %s - Risk Level: %s - User: %s - IP: %s\n", 
		event.EventType, event.RiskLevel, event.UserID, event.IPAddress)
}

// Filter types for querying
type AuditFilters struct {
	UserID    string
	Action    string
	Resource  string
	FacultyID string
	Severity  string
	Category  string
	StartDate time.Time
	EndDate   time.Time
	Success   *bool
}

type SecurityFilters struct {
	EventType string
	UserID    string
	IPAddress string
	RiskLevel string
	StartDate time.Time
	EndDate   time.Time
}

// Helper functions
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func getUserIDFromContext(ctx context.Context) string {
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

func getUserRoleFromContext(ctx context.Context) string {
	if role := ctx.Value("user_role"); role != nil {
		if r, ok := role.(string); ok {
			return r
		}
	}
	return ""
}

func getFacultyIDFromContext(ctx context.Context) string {
	if facultyID := ctx.Value("faculty_id"); facultyID != nil {
		if id, ok := facultyID.(string); ok {
			return id
		}
	}
	return ""
}

func getIPFromContext(ctx context.Context) string {
	if ip := ctx.Value("client_ip"); ip != nil {
		if clientIP, ok := ip.(string); ok {
			return clientIP
		}
	}
	return ""
}

func getUserAgentFromContext(ctx context.Context) string {
	if ua := ctx.Value("user_agent"); ua != nil {
		if userAgent, ok := ua.(string); ok {
			return userAgent
		}
	}
	return ""
}

func getSessionIDFromContext(ctx context.Context) string {
	if sessionID := ctx.Value("session_id"); sessionID != nil {
		if id, ok := sessionID.(string); ok {
			return id
		}
	}
	return ""
}
package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/redis/go-redis/v9"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

const (
	// Security limits
	MaxQueryDepth        = 15
	MaxQueryComplexity   = 1000
	MaxQueryNodes        = 100
	DefaultRateLimit     = 100  // requests per minute
	AdminRateLimit       = 1000 // higher limit for admins
	QRScanRateLimit      = 30   // QR scans per minute
	
	// Redis keys
	RateLimitPrefix      = "rate_limit:"
	QRScanLimitPrefix    = "qr_scan_limit:"
	QueryCachePrefix     = "query_cache:"
	SecurityEventPrefix  = "security_event:"
)

type SecurityMiddleware struct {
	redisClient *redis.Client
}

func NewSecurityMiddleware(redisClient *redis.Client) *SecurityMiddleware {
	return &SecurityMiddleware{
		redisClient: redisClient,
	}
}

// GraphQL Security Extension
func (s *SecurityMiddleware) ExtensionName() string {
	return "Security"
}

func (s *SecurityMiddleware) Validate(schema *ast.Schema) error {
	return nil
}

// Request validation and security checks
func (s *SecurityMiddleware) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	return func(ctx context.Context) *graphql.Response {
		// Get operation context
		oc := graphql.GetOperationContext(ctx)
		
		// 1. Query Depth Limiting
		if err := s.checkQueryDepth(oc.Operation); err != nil {
			return graphql.ErrorResponse(ctx, err.Error())
		}
		
		// 2. Query Complexity Analysis
		if err := s.checkQueryComplexity(oc.Operation); err != nil {
			return graphql.ErrorResponse(ctx, err.Error())
		}
		
		// 3. Rate Limiting
		if err := s.checkRateLimit(ctx, oc); err != nil {
			return graphql.ErrorResponse(ctx, err.Error())
		}
		
		// 4. Input Validation
		if err := s.validateInputs(oc.Variables); err != nil {
			return graphql.ErrorResponse(ctx, err.Error())
		}
		
		// 5. Log security event
		s.logSecurityEvent(ctx, oc, "operation_started")
		
		return next(ctx)
	}
}

// Query depth checking
func (s *SecurityMiddleware) checkQueryDepth(operation *ast.OperationDefinition) error {
	depth := s.calculateDepth(operation.SelectionSet, 0)
	if depth > MaxQueryDepth {
		return fmt.Errorf("query depth %d exceeds maximum allowed depth %d", depth, MaxQueryDepth)
	}
	return nil
}

func (s *SecurityMiddleware) calculateDepth(selectionSet ast.SelectionSet, currentDepth int) int {
	if currentDepth > MaxQueryDepth {
		return currentDepth
	}
	
	maxDepth := currentDepth
	for _, selection := range selectionSet {
		switch sel := selection.(type) {
		case *ast.Field:
			if sel.SelectionSet != nil {
				depth := s.calculateDepth(sel.SelectionSet, currentDepth+1)
				if depth > maxDepth {
					maxDepth = depth
				}
			}
		case *ast.InlineFragment:
			depth := s.calculateDepth(sel.SelectionSet, currentDepth)
			if depth > maxDepth {
				maxDepth = depth
			}
		case *ast.FragmentSpread:
			// For fragment spreads, we would need to resolve the fragment
			// This is a simplified implementation
			maxDepth = currentDepth + 1
		}
	}
	return maxDepth
}

// Query complexity analysis
func (s *SecurityMiddleware) checkQueryComplexity(operation *ast.OperationDefinition) error {
	complexity := s.calculateComplexity(operation.SelectionSet)
	if complexity > MaxQueryComplexity {
		return fmt.Errorf("query complexity %d exceeds maximum allowed complexity %d", complexity, MaxQueryComplexity)
	}
	return nil
}

func (s *SecurityMiddleware) calculateComplexity(selectionSet ast.SelectionSet) int {
	complexity := 0
	nodeCount := 0
	
	for _, selection := range selectionSet {
		nodeCount++
		if nodeCount > MaxQueryNodes {
			return MaxQueryComplexity + 1 // Exceed limit
		}
		
		switch sel := selection.(type) {
		case *ast.Field:
			// Base complexity for field
			fieldComplexity := 1
			
			// Increase complexity for list fields
			if s.isListField(sel.Name) {
				fieldComplexity *= 10
			}
			
			// Increase complexity for expensive operations
			if s.isExpensiveField(sel.Name) {
				fieldComplexity *= 5
			}
			
			complexity += fieldComplexity
			
			// Add complexity for nested fields
			if sel.SelectionSet != nil {
				complexity += s.calculateComplexity(sel.SelectionSet)
			}
			
		case *ast.InlineFragment:
			complexity += s.calculateComplexity(sel.SelectionSet)
		case *ast.FragmentSpread:
			complexity += 5 // Base complexity for fragments
		}
	}
	
	return complexity
}

func (s *SecurityMiddleware) isListField(fieldName string) bool {
	listFields := []string{
		"activities", "users", "participations", "faculties", 
		"departments", "systemMetrics", "facultyMetrics",
	}
	
	for _, field := range listFields {
		if field == fieldName {
			return true
		}
	}
	return false
}

func (s *SecurityMiddleware) isExpensiveField(fieldName string) bool {
	expensiveFields := []string{
		"systemMetrics", "facultyMetrics", "activityAnalytics",
		"participationStats", "subscriptionAnalytics",
	}
	
	for _, field := range expensiveFields {
		if field == fieldName {
			return true
		}
	}
	return false
}

// Rate limiting per user/faculty
func (s *SecurityMiddleware) checkRateLimit(ctx context.Context, oc *graphql.OperationContext) error {
	userID := getUserID(ctx)
	if userID == "" {
		return fmt.Errorf("authentication required")
	}
	
	// Determine rate limit based on user role
	limit := DefaultRateLimit
	userRole := getUserRole(ctx)
	if userRole == "SUPER_ADMIN" || userRole == "FACULTY_ADMIN" {
		limit = AdminRateLimit
	}
	
	// Special rate limiting for QR scan operations
	if s.isQRScanOperation(oc.OperationName) {
		return s.checkQRScanRateLimit(ctx, userID)
	}
	
	// General rate limiting
	key := fmt.Sprintf("%s%s", RateLimitPrefix, userID)
	return s.checkRedisRateLimit(ctx, key, limit, time.Minute)
}

func (s *SecurityMiddleware) checkQRScanRateLimit(ctx context.Context, userID string) error {
	key := fmt.Sprintf("%s%s", QRScanLimitPrefix, userID)
	return s.checkRedisRateLimit(ctx, key, QRScanRateLimit, time.Minute)
}

func (s *SecurityMiddleware) checkRedisRateLimit(ctx context.Context, key string, limit int, window time.Duration) error {
	// Use Redis for distributed rate limiting
	pipe := s.redisClient.Pipeline()
	
	// Increment counter
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("rate limit check failed: %v", err)
	}
	
	count := incr.Val()
	if count > int64(limit) {
		s.logSecurityEvent(ctx, nil, fmt.Sprintf("rate_limit_exceeded:%d", count))
		return fmt.Errorf("rate limit exceeded: %d requests per %v", limit, window)
	}
	
	return nil
}

func (s *SecurityMiddleware) isQRScanOperation(operationName string) bool {
	qrOperations := []string{"scanQRCode", "validateQRCode", "generateQRCode"}
	for _, op := range qrOperations {
		if op == operationName {
			return true
		}
	}
	return false
}

// Input validation and sanitization
func (s *SecurityMiddleware) validateInputs(variables map[string]interface{}) error {
	for key, value := range variables {
		if err := s.validateInput(key, value); err != nil {
			return fmt.Errorf("invalid input for %s: %v", key, err)
		}
	}
	return nil
}

func (s *SecurityMiddleware) validateInput(key string, value interface{}) error {
	switch key {
	case "email":
		return s.validateEmail(value)
	case "studentID":
		return s.validateStudentID(value)
	case "qrData":
		return s.validateQRData(value)
	case "activityID", "userID", "facultyID":
		return s.validateID(value)
	default:
		return s.validateGenericInput(value)
	}
}

func (s *SecurityMiddleware) validateEmail(value interface{}) error {
	email, ok := value.(string)
	if !ok {
		return fmt.Errorf("email must be a string")
	}
	
	// Basic email validation
	if !strings.Contains(email, "@") || len(email) < 5 || len(email) > 100 {
		return fmt.Errorf("invalid email format")
	}
	
	// Check for malicious patterns
	if s.containsMaliciousPatterns(email) {
		return fmt.Errorf("email contains invalid characters")
	}
	
	return nil
}

func (s *SecurityMiddleware) validateStudentID(value interface{}) error {
	studentID, ok := value.(string)
	if !ok {
		return fmt.Errorf("studentID must be a string")
	}
	
	// Student ID should be alphanumeric and reasonable length
	if len(studentID) < 5 || len(studentID) > 20 {
		return fmt.Errorf("studentID length must be between 5 and 20 characters")
	}
	
	// Check format (adjust based on your institution's format)
	for _, char := range studentID {
		if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
			return fmt.Errorf("studentID contains invalid characters")
		}
	}
	
	return nil
}

func (s *SecurityMiddleware) validateQRData(value interface{}) error {
	qrData, ok := value.(string)
	if !ok {
		return fmt.Errorf("qrData must be a string")
	}
	
	// QR data should be base64 or JSON string
	if len(qrData) == 0 || len(qrData) > 2000 {
		return fmt.Errorf("qrData length must be between 1 and 2000 characters")
	}
	
	// Check for malicious patterns
	if s.containsMaliciousPatterns(qrData) {
		return fmt.Errorf("qrData contains invalid patterns")
	}
	
	return nil
}

func (s *SecurityMiddleware) validateID(value interface{}) error {
	id, ok := value.(string)
	if !ok {
		return fmt.Errorf("ID must be a string")
	}
	
	// UUID format validation (adjust based on your ID format)
	if len(id) < 10 || len(id) > 50 {
		return fmt.Errorf("ID length must be between 10 and 50 characters")
	}
	
	return nil
}

func (s *SecurityMiddleware) validateGenericInput(value interface{}) error {
	switch v := value.(type) {
	case string:
		if len(v) > 10000 { // Prevent extremely large strings
			return fmt.Errorf("string input too large")
		}
		if s.containsMaliciousPatterns(v) {
			return fmt.Errorf("input contains malicious patterns")
		}
	case []interface{}:
		if len(v) > 1000 { // Prevent extremely large arrays
			return fmt.Errorf("array input too large")
		}
		for _, item := range v {
			if err := s.validateGenericInput(item); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		if len(v) > 100 { // Prevent extremely large objects
			return fmt.Errorf("object input too large")
		}
		for key, val := range v {
			if err := s.validateGenericInput(val); err != nil {
				return err
			}
			if s.containsMaliciousPatterns(key) {
				return fmt.Errorf("object key contains malicious patterns")
			}
		}
	}
	return nil
}

func (s *SecurityMiddleware) containsMaliciousPatterns(input string) bool {
	maliciousPatterns := []string{
		"<script", "javascript:", "onload=", "onerror=",
		"SELECT ", "INSERT ", "UPDATE ", "DELETE ", "DROP ",
		"UNION ", "OR 1=1", "' OR '1'='1", "admin'--",
		"../", "..\\", "/etc/passwd", "cmd.exe",
	}
	
	lowerInput := strings.ToLower(input)
	for _, pattern := range maliciousPatterns {
		if strings.Contains(lowerInput, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// Security event logging
func (s *SecurityMiddleware) logSecurityEvent(ctx context.Context, oc *graphql.OperationContext, eventType string) {
	userID := getUserID(ctx)
	timestamp := time.Now().Unix()
	
	event := map[string]interface{}{
		"user_id":   userID,
		"timestamp": timestamp,
		"event":     eventType,
		"ip":        getClientIP(ctx),
		"user_agent": getUserAgent(ctx),
	}
	
	if oc != nil {
		event["operation"] = oc.OperationName
		event["query_hash"] = s.hashQuery(oc.RawQuery)
	}
	
	// Store in Redis for real-time monitoring
	key := fmt.Sprintf("%s%s:%d", SecurityEventPrefix, userID, timestamp)
	s.redisClient.HMSet(ctx, key, event)
	s.redisClient.Expire(ctx, key, 24*time.Hour) // Keep for 24 hours
}

func (s *SecurityMiddleware) hashQuery(query string) string {
	hash := sha256.Sum256([]byte(query))
	return hex.EncodeToString(hash[:])
}

// Helper functions to extract context information
func getUserID(ctx context.Context) string {
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

func getUserRole(ctx context.Context) string {
	if role := ctx.Value("user_role"); role != nil {
		if r, ok := role.(string); ok {
			return r
		}
	}
	return ""
}

func getClientIP(ctx context.Context) string {
	if ip := ctx.Value("client_ip"); ip != nil {
		if clientIP, ok := ip.(string); ok {
			return clientIP
		}
	}
	return ""
}

func getUserAgent(ctx context.Context) string {
	if ua := ctx.Value("user_agent"); ua != nil {
		if userAgent, ok := ua.(string); ok {
			return userAgent
		}
	}
	return ""
}

// Field-level security directive
func (s *SecurityMiddleware) InterceptField(ctx context.Context, next graphql.Resolver) (interface{}, error) {
	fc := graphql.GetFieldContext(ctx)
	
	// Check field-level permissions
	if err := s.checkFieldPermission(ctx, fc.Field.Name); err != nil {
		s.logSecurityEvent(ctx, nil, fmt.Sprintf("field_access_denied:%s", fc.Field.Name))
		return nil, err
	}
	
	return next(ctx)
}

func (s *SecurityMiddleware) checkFieldPermission(ctx context.Context, fieldName string) error {
	userRole := getUserRole(ctx)
	
	// Define field-level permissions
	restrictedFields := map[string][]string{
		"systemMetrics":      {"SUPER_ADMIN"},
		"allUsers":          {"SUPER_ADMIN"},
		"facultyMetrics":    {"SUPER_ADMIN", "FACULTY_ADMIN"},
		"subscriptions":     {"SUPER_ADMIN", "FACULTY_ADMIN"},
		"qrSecret":          {"STUDENT"}, // Students can only access their own QR secret
		"adminActions":      {"SUPER_ADMIN", "FACULTY_ADMIN", "REGULAR_ADMIN"},
	}
	
	if allowedRoles, exists := restrictedFields[fieldName]; exists {
		for _, role := range allowedRoles {
			if role == userRole {
				return nil
			}
		}
		return fmt.Errorf("insufficient permissions to access field: %s", fieldName)
	}
	
	return nil
}

// Query result filtering based on permissions
func (s *SecurityMiddleware) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	resp := next(ctx)
	
	// Filter sensitive data based on user permissions
	if resp.Data != nil {
		filteredData := s.filterResponseData(ctx, resp.Data)
		resp.Data = filteredData
	}
	
	return resp
}

func (s *SecurityMiddleware) filterResponseData(ctx context.Context, data interface{}) interface{} {
	userRole := getUserRole(ctx)
	userID := getUserID(ctx)
	
	switch d := data.(type) {
	case map[string]interface{}:
		filtered := make(map[string]interface{})
		for key, value := range d {
			if s.canAccessField(userRole, userID, key) {
				filtered[key] = s.filterResponseData(ctx, value)
			}
		}
		return filtered
	case []interface{}:
		filtered := make([]interface{}, 0, len(d))
		for _, item := range d {
			filteredItem := s.filterResponseData(ctx, item)
			if filteredItem != nil {
				filtered = append(filtered, filteredItem)
			}
		}
		return filtered
	default:
		return data
	}
}

func (s *SecurityMiddleware) canAccessField(userRole, userID, fieldName string) bool {
	// Define field access rules
	sensitiveFields := []string{"password", "secret", "token", "privateKey"}
	
	// Remove sensitive fields for all users
	for _, field := range sensitiveFields {
		if field == fieldName {
			return false
		}
	}
	
	// Role-based field access
	switch userRole {
	case "SUPER_ADMIN":
		return true // Super admin can access all fields
	case "FACULTY_ADMIN":
		// Faculty admin cannot access other faculties' sensitive data
		restrictedFields := []string{"otherFacultyData", "systemSecrets"}
		for _, field := range restrictedFields {
			if field == fieldName {
				return false
			}
		}
		return true
	case "REGULAR_ADMIN":
		// Regular admin has limited access
		allowedFields := []string{"assignedActivities", "scanResults", "basicUserData"}
		for _, field := range allowedFields {
			if field == fieldName {
				return true
			}
		}
		return false
	case "STUDENT":
		// Students can only access their own data
		personalFields := []string{"myActivities", "myQRCode", "myParticipations"}
		for _, field := range personalFields {
			if field == fieldName {
				return true
			}
		}
		return false
	}
	
	return false
}
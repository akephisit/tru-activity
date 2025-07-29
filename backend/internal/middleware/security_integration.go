package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"your-project/pkg/audit"
	"your-project/pkg/monitoring"
	"your-project/pkg/performance"
	"your-project/pkg/security"
)

// SecurityIntegration provides a unified security and performance layer
type SecurityIntegration struct {
	// Core components
	db                *gorm.DB
	redisClient       *redis.Client
	
	// Security components
	securityMiddleware *SecurityMiddleware
	qrSecurity        *security.QRSecurityManager
	
	// Performance components
	dataLoaders       *performance.DataLoaderContainer
	cacheManager      *performance.CacheManager
	queryOptimizer    *database.QueryOptimizer
	
	// Monitoring components
	auditLogger       *audit.AuditLogger
	perfMonitor       *monitoring.PerformanceMonitor
}

// NewSecurityIntegration creates a new integrated security system
func NewSecurityIntegration(db *gorm.DB, redisClient *redis.Client, masterSecret []byte) *SecurityIntegration {
	si := &SecurityIntegration{
		db:          db,
		redisClient: redisClient,
	}
	
	// Initialize security components
	si.securityMiddleware = NewSecurityMiddleware(redisClient)
	si.qrSecurity = security.NewQRSecurityManager(redisClient, masterSecret)
	
	// Initialize performance components
	si.dataLoaders = performance.NewDataLoaderContainer(db, redisClient)
	si.cacheManager = performance.NewCacheManager(redisClient, db)
	si.queryOptimizer = database.NewQueryOptimizer(db, redisClient)
	
	// Initialize monitoring components
	si.auditLogger = audit.NewAuditLogger(db, redisClient)
	si.perfMonitor = monitoring.NewPerformanceMonitor(db, redisClient)
	
	return si
}

// SetupGraphQLServer configures a GraphQL server with all security and performance features
func (si *SecurityIntegration) SetupGraphQLServer(schema graphql.ExecutableSchema) *handler.Server {
	srv := handler.New(schema)
	
	// Configure transports
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Implement proper origin checking
				origin := r.Header.Get("Origin")
				allowedOrigins := []string{
					"http://localhost:3000",
					"https://your-domain.com",
				}
				
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						return true
					}
				}
				return false
			},
		},
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	
	// Add security extensions
	srv.Use(si.securityMiddleware)
	
	// Add performance extensions
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(1000), // Cache for automatic persisted queries
	})
	
	// Add custom extensions
	srv.Use(&PerformanceExtension{
		perfMonitor: si.perfMonitor,
		auditLogger: si.auditLogger,
	})
	
	srv.Use(&CacheExtension{
		cacheManager: si.cacheManager,
	})
	
	return srv
}

// HTTP middleware chain for comprehensive security
func (si *SecurityIntegration) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		start := time.Now()
		
		// 1. Security headers
		si.setSecurityHeaders(w)
		
		// 2. Rate limiting check
		if err := si.checkRateLimit(ctx, r); err != nil {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			si.auditLogger.LogSecurityEvent(ctx, &audit.SecurityEvent{
				EventType: audit.SecurityEventRateLimitExceeded,
				Details:   map[string]interface{}{"path": r.URL.Path, "method": r.Method},
				RiskLevel: audit.RiskLevelMedium,
			})
			return
		}
		
		// 3. Add context information
		ctx = si.enrichContext(ctx, r)
		r = r.WithContext(ctx)
		
		// 4. Add DataLoaders to context
		ctx = performance.WithDataLoaders(ctx, si.dataLoaders)
		r = r.WithContext(ctx)
		
		// 5. Execute request
		next.ServeHTTP(w, r)
		
		// 6. Log request
		duration := time.Since(start)
		si.logHTTPRequest(ctx, r, duration)
	})
}

// setSecurityHeaders adds security headers to the response
func (si *SecurityIntegration) setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws: wss:")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
}

// checkRateLimit performs HTTP-level rate limiting
func (si *SecurityIntegration) checkRateLimit(ctx context.Context, r *http.Request) error {
	clientIP := getClientIP(r)
	key := fmt.Sprintf("http_rate_limit:%s", clientIP)
	
	// Different limits for different endpoints
	limit := 60 // Default: 60 requests per minute
	if strings.Contains(r.URL.Path, "/query") {
		limit = 30 // GraphQL queries: 30 per minute
	}
	
	pipe := si.redisClient.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Minute)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("rate limit check failed: %v", err)
	}
	
	if incr.Val() > int64(limit) {
		return fmt.Errorf("rate limit exceeded: %d requests per minute", limit)
	}
	
	return nil
}

// enrichContext adds security and performance context
func (si *SecurityIntegration) enrichContext(ctx context.Context, r *http.Request) context.Context {
	// Add request metadata
	ctx = context.WithValue(ctx, "client_ip", getClientIP(r))
	ctx = context.WithValue(ctx, "user_agent", r.UserAgent())
	ctx = context.WithValue(ctx, "request_id", generateRequestID())
	ctx = context.WithValue(ctx, "request_start", time.Now())
	
	// Add security context if authenticated
	if token := r.Header.Get("Authorization"); token != "" {
		if strings.HasPrefix(token, "Bearer ") {
			tokenStr := strings.TrimPrefix(token, "Bearer ")
			if claims, err := si.validateJWT(tokenStr); err == nil {
				ctx = context.WithValue(ctx, "user_id", claims["user_id"])
				ctx = context.WithValue(ctx, "user_role", claims["role"])
				ctx = context.WithValue(ctx, "faculty_id", claims["faculty_id"])
				ctx = context.WithValue(ctx, "session_id", claims["session_id"])
			}
		}
	}
	
	return ctx
}

// logHTTPRequest logs HTTP request details
func (si *SecurityIntegration) logHTTPRequest(ctx context.Context, r *http.Request, duration time.Duration) {
	// Record performance metrics
	si.perfMonitor.RecordMetric(ctx, monitoring.MetricPoint{
		Name:      "http_request_duration",
		Value:     float64(duration.Milliseconds()),
		Unit:      "milliseconds",
		Tags:      map[string]string{"method": r.Method, "path": r.URL.Path},
		Timestamp: time.Now(),
	})
	
	// Log audit event for sensitive endpoints
	if si.isSensitiveEndpoint(r.URL.Path) {
		si.auditLogger.LogDataAccess(ctx, "HTTP_ENDPOINT", r.URL.Path, map[string]interface{}{
			"method":   r.Method,
			"duration": duration.String(),
		})
	}
}

// isSensitiveEndpoint checks if an endpoint is sensitive
func (si *SecurityIntegration) isSensitiveEndpoint(path string) bool {
	sensitiveEndpoints := []string{
		"/admin", "/query", "/upload", "/export",
	}
	
	for _, sensitive := range sensitiveEndpoints {
		if strings.Contains(path, sensitive) {
			return true
		}
	}
	return false
}

// QR Code security integration
func (si *SecurityIntegration) ValidateQRCode(ctx context.Context, qrData string, scannerID string, activityID string) (*security.QRValidationResult, error) {
	clientIP := getClientIPFromContext(ctx)
	userAgent := getUserAgentFromContext(ctx)
	
	result, err := si.qrSecurity.ValidateQRData(ctx, qrData, scannerID, activityID, clientIP, userAgent)
	
	// Log QR scan attempt
	si.auditLogger.LogQRScan(ctx, result.StudentID, activityID, scannerID, result.Valid, result.Message, map[string]interface{}{
		"qr_security_level": result.SecurityLevel,
		"scan_timestamp":   result.Timestamp,
	})
	
	return result, err
}

// Performance Extensions

// PerformanceExtension monitors GraphQL performance
type PerformanceExtension struct {
	perfMonitor *monitoring.PerformanceMonitor
	auditLogger *audit.AuditLogger
}

func (pe *PerformanceExtension) ExtensionName() string {
	return "Performance"
}

func (pe *PerformanceExtension) Validate(schema *ast.Schema) error {
	return nil
}

func (pe *PerformanceExtension) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	return func(ctx context.Context) *graphql.Response {
		start := time.Now()
		oc := graphql.GetOperationContext(ctx)
		
		resp := next(ctx)
		
		duration := time.Since(start)
		
		// Record performance metrics
		pe.perfMonitor.RecordMetric(ctx, monitoring.MetricPoint{
			Name:      "graphql_operation_duration",
			Value:     float64(duration.Milliseconds()),
			Unit:      "milliseconds",
			Tags:      map[string]string{"operation": oc.OperationName, "type": string(oc.Operation.Operation)},
			Timestamp: time.Now(),
		})
		
		// Log slow operations
		if duration > 2*time.Second {
			pe.auditLogger.LogPerformanceMetric(ctx, &audit.PerformanceMetric{
				MetricType: "slow_graphql_operation",
				Operation:  oc.OperationName,
				Duration:   duration.Milliseconds(),
				Details: map[string]interface{}{
					"query_complexity": len(oc.Operation.SelectionSet),
					"variables_count":  len(oc.Variables),
				},
			})
		}
		
		return resp
	}
}

// CacheExtension provides GraphQL query caching
type CacheExtension struct {
	cacheManager *performance.CacheManager
}

func (ce *CacheExtension) ExtensionName() string {
	return "Cache"
}

func (ce *CacheExtension) Validate(schema *ast.Schema) error {
	return nil
}

func (ce *CacheExtension) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	return func(ctx context.Context) *graphql.Response {
		oc := graphql.GetOperationContext(ctx)
		
		// Try cache first for queries
		if oc.Operation.Operation == ast.Query {
			queryHash := hashQuery(oc.RawQuery)
			
			var cachedResult interface{}
			if err := ce.cacheManager.GetCachedQuery(ctx, queryHash, oc.Variables, &cachedResult); err == nil {
				// Cache hit
				return &graphql.Response{Data: cachedResult}
			}
		}
		
		// Execute query
		resp := next(ctx)
		
		// Cache successful query results
		if resp.Errors == nil && oc.Operation.Operation == ast.Query {
			queryHash := hashQuery(oc.RawQuery)
			ce.cacheManager.CacheQuery(ctx, queryHash, oc.Variables, resp.Data)
		}
		
		return resp
	}
}

// Utility functions

func getClientIP(r *http.Request) string {
	// Check various headers for real IP
	headers := []string{
		"CF-Connecting-IP",      // Cloudflare
		"X-Real-IP",             // Nginx
		"X-Forwarded-For",       // Standard
		"X-Client-IP",
		"X-Forwarded",
		"X-Cluster-Client-IP",
		"Forwarded-For",
		"Forwarded",
	}
	
	for _, header := range headers {
		if ip := r.Header.Get(header); ip != "" {
			// Handle comma-separated IPs (take the first one)
			if strings.Contains(ip, ",") {
				ip = strings.Split(ip, ",")[0]
			}
			ip = strings.TrimSpace(ip)
			if ip != "" {
				return ip
			}
		}
	}
	
	// Fallback to remote address
	return strings.Split(r.RemoteAddr, ":")[0]
}

func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (si *SecurityIntegration) validateJWT(token string) (map[string]interface{}, error) {
	// Implement JWT validation using your JWT service
	// This is a placeholder implementation
	return map[string]interface{}{
		"user_id":    "user123",
		"role":       "STUDENT",
		"faculty_id": "faculty123",
		"session_id": "session123",
	}, nil
}

func getClientIPFromContext(ctx context.Context) string {
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

func hashQuery(query string) string {
	// Simple hash implementation
	return fmt.Sprintf("%x", md5.Sum([]byte(query)))[:16]
}

// Health check endpoint with comprehensive system status
func (si *SecurityIntegration) HealthCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		health, err := si.perfMonitor.GetSystemHealth(ctx)
		if err != nil {
			http.Error(w, "Health check failed", http.StatusInternalServerError)
			return
		}
		
		// Add additional health checks
		health.Details["database_status"] = si.checkDatabaseHealth(ctx)
		health.Details["redis_status"] = si.checkRedisHealth(ctx)
		health.Details["cache_stats"] = si.getCacheStats(ctx)
		
		// Set appropriate HTTP status
		status := http.StatusOK
		if health.Status == "DEGRADED" {
			status = http.StatusServiceUnavailable
		} else if health.Status == "UNHEALTHY" {
			status = http.StatusServiceUnavailable
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		
		healthJSON, _ := json.Marshal(health)
		w.Write(healthJSON)
	}
}

func (si *SecurityIntegration) checkDatabaseHealth(ctx context.Context) string {
	if err := si.db.Raw("SELECT 1").Error; err != nil {
		return "UNHEALTHY"
	}
	return "HEALTHY"
}

func (si *SecurityIntegration) checkRedisHealth(ctx context.Context) string {
	if err := si.redisClient.Ping(ctx).Err(); err != nil {
		return "UNHEALTHY"
	}
	return "HEALTHY"
}

func (si *SecurityIntegration) getCacheStats(ctx context.Context) map[string]interface{} {
	stats, err := si.cacheManager.GetCacheStats(ctx)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return stats
}

// Metrics endpoint for monitoring integration
func (si *SecurityIntegration) MetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		metrics := map[string]interface{}{
			"query_stats":        si.queryOptimizer.GetQueryStatistics(),
			"slow_queries":       si.queryOptimizer.GetSlowQueries(10),
			"performance_alerts": si.getRecentAlerts(ctx),
			"security_events":    si.getRecentSecurityEvents(ctx),
		}
		
		w.Header().Set("Content-Type", "application/json")
		metricsJSON, _ := json.Marshal(metrics)
		w.Write(metricsJSON)
	}
}

func (si *SecurityIntegration) getRecentAlerts(ctx context.Context) interface{} {
	// Get recent performance alerts from monitoring system
	// Implementation would depend on your monitoring setup
	return []interface{}{}
}

func (si *SecurityIntegration) getRecentSecurityEvents(ctx context.Context) interface{} {
	filters := audit.SecurityFilters{
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now(),
	}
	
	events, _, err := si.auditLogger.GetSecurityEvents(ctx, filters, 10, 0)
	if err != nil {
		return []interface{}{}
	}
	
	return events
}
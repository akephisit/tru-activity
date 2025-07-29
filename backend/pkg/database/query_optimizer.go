package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// QueryOptimizer handles database query optimization and monitoring
type QueryOptimizer struct {
	db          *gorm.DB
	redisClient *redis.Client
	
	// Query statistics
	queryStats  sync.Map
	slowQueries sync.Map
	
	// Configuration
	slowQueryThreshold time.Duration
	enableQueryCache   bool
	cacheTimeout       time.Duration
}

// QueryStats represents statistics for a specific query
type QueryStats struct {
	Query          string        `json:"query"`
	QueryHash      string        `json:"query_hash"`
	TotalCalls     int64         `json:"total_calls"`
	TotalDuration  time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	MinDuration    time.Duration `json:"min_duration"`
	MaxDuration    time.Duration `json:"max_duration"`
	LastExecuted   time.Time     `json:"last_executed"`
	ErrorCount     int64         `json:"error_count"`
}

// SlowQuery represents a slow query log entry
type SlowQuery struct {
	ID            string        `json:"id"`
	Query         string        `json:"query"`
	QueryHash     string        `json:"query_hash"`
	Duration      time.Duration `json:"duration"`
	Tables        []string      `json:"tables"`
	RowsExamined  int64         `json:"rows_examined"`
	RowsReturned  int64         `json:"rows_returned"`
	UserID        string        `json:"user_id"`
	Timestamp     time.Time     `json:"timestamp"`
	ExecutionPlan string        `json:"execution_plan"`
}

// QueryCacheEntry represents a cached query result
type QueryCacheEntry struct {
	QueryHash string      `json:"query_hash"`
	Result    interface{} `json:"result"`
	CachedAt  time.Time   `json:"cached_at"`
	ExpiresAt time.Time   `json:"expires_at"`
	HitCount  int64       `json:"hit_count"`
}

// ConnectionPool manages database connections with enhanced monitoring
type ConnectionPool struct {
	db           *sql.DB
	maxOpen      int
	maxIdle      int
	maxLifetime  time.Duration
	connTimeout  time.Duration
	
	// Statistics
	totalConnections    int64
	activeConnections   int64
	idleConnections     int64
	connectionFailures  int64
	connectionWaitTime  time.Duration
	
	mu sync.RWMutex
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *gorm.DB, redisClient *redis.Client) *QueryOptimizer {
	qo := &QueryOptimizer{
		db:                 db,
		redisClient:        redisClient,
		slowQueryThreshold: 1 * time.Second,
		enableQueryCache:   true,
		cacheTimeout:       5 * time.Minute,
	}
	
	// Set up custom logger to capture query metrics
	qo.setupQueryLogger()
	
	// Start background optimization tasks
	go qo.startQueryAnalysis()
	go qo.startQueryCacheCleanup()
	
	return qo
}

// setupQueryLogger sets up custom GORM logger for query monitoring
func (qo *QueryOptimizer) setupQueryLogger() {
	customLogger := &QueryLogger{
		optimizer: qo,
		Logger:    logger.Default,
	}
	
	qo.db.Logger = customLogger
}

// QueryLogger is a custom GORM logger that captures query metrics
type QueryLogger struct {
	optimizer *QueryOptimizer
	logger.Interface
}

// Trace captures query execution details
func (ql *QueryLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	// Call original logger
	ql.Interface.Trace(ctx, begin, fc, err)
	
	// Capture metrics
	elapsed := time.Since(begin)
	sql, rowsAffected := fc()
	
	go ql.optimizer.recordQueryMetrics(ctx, sql, elapsed, rowsAffected, err)
}

// recordQueryMetrics records query execution metrics
func (qo *QueryOptimizer) recordQueryMetrics(ctx context.Context, query string, duration time.Duration, rowsAffected int64, err error) {
	queryHash := hashQuery(query)
	
	// Update query statistics
	if stats, exists := qo.queryStats.Load(queryHash); exists {
		s := stats.(*QueryStats)
		s.TotalCalls++
		s.TotalDuration += duration
		s.AverageDuration = s.TotalDuration / time.Duration(s.TotalCalls)
		s.LastExecuted = time.Now()
		
		if duration < s.MinDuration || s.MinDuration == 0 {
			s.MinDuration = duration
		}
		if duration > s.MaxDuration {
			s.MaxDuration = duration
		}
		
		if err != nil {
			s.ErrorCount++
		}
	} else {
		stats := &QueryStats{
			Query:          sanitizeQuery(query),
			QueryHash:      queryHash,
			TotalCalls:     1,
			TotalDuration:  duration,
			AverageDuration: duration,
			MinDuration:    duration,
			MaxDuration:    duration,
			LastExecuted:   time.Now(),
			ErrorCount:     0,
		}
		
		if err != nil {
			stats.ErrorCount = 1
		}
		
		qo.queryStats.Store(queryHash, stats)
	}
	
	// Log slow queries
	if duration > qo.slowQueryThreshold {
		slowQuery := &SlowQuery{
			ID:           fmt.Sprintf("%s_%d", queryHash, time.Now().UnixNano()),
			Query:        sanitizeQuery(query),
			QueryHash:    queryHash,
			Duration:     duration,
			Tables:       extractTables(query),
			RowsReturned: rowsAffected,
			UserID:       getUserIDFromContext(ctx),
			Timestamp:    time.Now(),
		}
		
		qo.logSlowQuery(ctx, slowQuery)
	}
	
	// Store metrics in Redis for real-time monitoring
	qo.storeMetricsInRedis(ctx, queryHash, duration, err != nil)
}

// logSlowQuery logs a slow query
func (qo *QueryOptimizer) logSlowQuery(ctx context.Context, slowQuery *SlowQuery) {
	qo.slowQueries.Store(slowQuery.ID, slowQuery)
	
	// Store in Redis for real-time monitoring
	slowQueryJSON, _ := json.Marshal(slowQuery)
	
	pipe := qo.redisClient.Pipeline()
	pipe.LPush(ctx, "slow_queries", slowQueryJSON)
	pipe.LTrim(ctx, "slow_queries", 0, 999) // Keep last 1000 slow queries
	pipe.Expire(ctx, "slow_queries", 24*time.Hour)
	
	// Update slow query counters
	today := time.Now().Format("2006-01-02")
	pipe.Incr(ctx, fmt.Sprintf("slow_queries_count:%s", today))
	pipe.Expire(ctx, fmt.Sprintf("slow_queries_count:%s", today), 7*24*time.Hour)
	
	pipe.Exec(ctx)
	
	fmt.Printf("SLOW QUERY DETECTED: %s (Duration: %v)\n", slowQuery.QueryHash, slowQuery.Duration)
}

// storeMetricsInRedis stores query metrics in Redis
func (qo *QueryOptimizer) storeMetricsInRedis(ctx context.Context, queryHash string, duration time.Duration, hasError bool) {
	pipe := qo.redisClient.Pipeline()
	
	// Update query counters
	today := time.Now().Format("2006-01-02")
	hour := time.Now().Format("2006-01-02:15")
	
	pipe.Incr(ctx, fmt.Sprintf("query_count:%s", today))
	pipe.Incr(ctx, fmt.Sprintf("query_count:%s", hour))
	pipe.IncrBy(ctx, fmt.Sprintf("query_duration:%s", today), duration.Milliseconds())
	pipe.IncrBy(ctx, fmt.Sprintf("query_duration:%s", hour), duration.Milliseconds())
	
	if hasError {
		pipe.Incr(ctx, fmt.Sprintf("query_errors:%s", today))
		pipe.Incr(ctx, fmt.Sprintf("query_errors:%s", hour))
	}
	
	// Per-query metrics
	pipe.Incr(ctx, fmt.Sprintf("query_stats:%s:count", queryHash))
	pipe.IncrBy(ctx, fmt.Sprintf("query_stats:%s:duration", queryHash), duration.Milliseconds())
	
	// Set expiry
	pipe.Expire(ctx, fmt.Sprintf("query_count:%s", today), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("query_count:%s", hour), 48*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("query_duration:%s", today), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("query_duration:%s", hour), 48*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("query_errors:%s", today), 7*24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("query_errors:%s", hour), 48*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("query_stats:%s:count", queryHash), 24*time.Hour)
	pipe.Expire(ctx, fmt.Sprintf("query_stats:%s:duration", queryHash), 24*time.Hour)
	
	pipe.Exec(ctx)
}

// OptimizedFind executes an optimized find query with caching
func (qo *QueryOptimizer) OptimizedFind(ctx context.Context, dest interface{}, query interface{}, args ...interface{}) error {
	// Generate cache key
	cacheKey := generateCacheKey("find", query, args...)
	
	// Try cache first if enabled
	if qo.enableQueryCache {
		if cached, err := qo.getCachedQuery(ctx, cacheKey); err == nil {
			// Cache hit
			qo.recordCacheHit(ctx, cacheKey)
			return mapToStruct(cached, dest)
		}
		qo.recordCacheMiss(ctx, cacheKey)
	}
	
	// Execute query with optimizations
	start := time.Now()
	
	db := qo.db.WithContext(ctx)
	
	// Apply query optimizations
	db = qo.applyQueryOptimizations(db, query)
	
	var err error
	switch q := query.(type) {
	case string:
		err = db.Raw(q, args...).Scan(dest).Error
	default:
		err = db.Find(dest, query, args...).Error
	}
	
	duration := time.Since(start)
	
	// Cache result if successful and query is cacheable
	if err == nil && qo.enableQueryCache && qo.isCacheable(query) {
		qo.cacheQuery(ctx, cacheKey, dest, duration)
	}
	
	return err
}

// OptimizedCreate executes an optimized create operation
func (qo *QueryOptimizer) OptimizedCreate(ctx context.Context, value interface{}) error {
	start := time.Now()
	
	db := qo.db.WithContext(ctx)
	
	// Use batch insert for slices
	if isSlice(value) {
		err := db.CreateInBatches(value, 100).Error
		
		// Invalidate related caches
		qo.invalidateRelatedCaches(ctx, value)
		
		return err
	}
	
	// Single create
	err := db.Create(value).Error
	duration := time.Since(start)
	
	// Invalidate related caches
	qo.invalidateRelatedCaches(ctx, value)
	
	// Log if slow
	if duration > qo.slowQueryThreshold {
		fmt.Printf("SLOW CREATE OPERATION: %v (Duration: %v)\n", getTypeName(value), duration)
	}
	
	return err
}

// OptimizedUpdate executes an optimized update operation
func (qo *QueryOptimizer) OptimizedUpdate(ctx context.Context, dest interface{}, updates interface{}) error {
	start := time.Now()
	
	db := qo.db.WithContext(ctx)
	err := db.Model(dest).Updates(updates).Error
	duration := time.Since(start)
	
	// Invalidate related caches
	qo.invalidateRelatedCaches(ctx, dest)
	
	// Log if slow
	if duration > qo.slowQueryThreshold {
		fmt.Printf("SLOW UPDATE OPERATION: %v (Duration: %v)\n", getTypeName(dest), duration)
	}
	
	return err
}

// applyQueryOptimizations applies various query optimizations
func (qo *QueryOptimizer) applyQueryOptimizations(db *gorm.DB, query interface{}) *gorm.DB {
	// Add appropriate indexes hints based on query type
	switch q := query.(type) {
	case string:
		if strings.Contains(strings.ToLower(q), "where") {
			// Suggest using indexes for WHERE clauses
			db = db.Set("gorm:query_hint", "USE INDEX")
		}
	}
	
	// Set reasonable query timeout
	ctx, cancel := context.WithTimeout(db.Statement.Context, 30*time.Second)
	defer cancel()
	db = db.WithContext(ctx)
	
	return db
}

// Query caching methods

// getCachedQuery retrieves a cached query result
func (qo *QueryOptimizer) getCachedQuery(ctx context.Context, cacheKey string) (interface{}, error) {
	cacheData, err := qo.redisClient.Get(ctx, fmt.Sprintf("query_cache:%s", cacheKey)).Result()
	if err != nil {
		return nil, err
	}
	
	var entry QueryCacheEntry
	if err := json.Unmarshal([]byte(cacheData), &entry); err != nil {
		return nil, err
	}
	
	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		qo.redisClient.Del(ctx, fmt.Sprintf("query_cache:%s", cacheKey))
		return nil, fmt.Errorf("cache expired")
	}
	
	// Update hit count
	entry.HitCount++
	if updatedData, err := json.Marshal(entry); err == nil {
		qo.redisClient.Set(ctx, fmt.Sprintf("query_cache:%s", cacheKey), updatedData, qo.cacheTimeout)
	}
	
	return entry.Result, nil
}

// cacheQuery caches a query result
func (qo *QueryOptimizer) cacheQuery(ctx context.Context, cacheKey string, result interface{}, queryDuration time.Duration) {
	entry := QueryCacheEntry{
		QueryHash: cacheKey,
		Result:    result,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(qo.cacheTimeout),
		HitCount:  0,
	}
	
	// Adjust cache timeout based on query performance
	if queryDuration > time.Second {
		entry.ExpiresAt = time.Now().Add(qo.cacheTimeout * 2) // Cache longer for slow queries
	}
	
	if entryData, err := json.Marshal(entry); err == nil {
		qo.redisClient.Set(ctx, fmt.Sprintf("query_cache:%s", cacheKey), entryData, qo.cacheTimeout*2)
	}
}

// isCacheable determines if a query should be cached
func (qo *QueryOptimizer) isCacheable(query interface{}) bool {
	queryStr := fmt.Sprintf("%v", query)
	queryLower := strings.ToLower(queryStr)
	
	// Don't cache INSERT, UPDATE, DELETE operations
	if strings.Contains(queryLower, "insert") ||
		strings.Contains(queryLower, "update") ||
		strings.Contains(queryLower, "delete") {
		return false
	}
	
	// Don't cache queries with time-sensitive functions
	if strings.Contains(queryLower, "now()") ||
		strings.Contains(queryLower, "current_timestamp") ||
		strings.Contains(queryLower, "rand()") {
		return false
	}
	
	return true
}

// invalidateRelatedCaches invalidates caches related to the given entity
func (qo *QueryOptimizer) invalidateRelatedCaches(ctx context.Context, entity interface{}) {
	typeName := getTypeName(entity)
	
	// Get all cache keys related to this entity type
	pattern := fmt.Sprintf("query_cache:*%s*", strings.ToLower(typeName))
	keys, err := qo.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return
	}
	
	if len(keys) > 0 {
		qo.redisClient.Del(ctx, keys...)
		fmt.Printf("Invalidated %d cache entries for %s\n", len(keys), typeName)
	}
}

// recordCacheHit records a cache hit
func (qo *QueryOptimizer) recordCacheHit(ctx context.Context, cacheKey string) {
	today := time.Now().Format("2006-01-02")
	qo.redisClient.Incr(ctx, fmt.Sprintf("cache_hits:%s", today))
	qo.redisClient.Expire(ctx, fmt.Sprintf("cache_hits:%s", today), 7*24*time.Hour)
}

// recordCacheMiss records a cache miss
func (qo *QueryOptimizer) recordCacheMiss(ctx context.Context, cacheKey string) {
	today := time.Now().Format("2006-01-02")
	qo.redisClient.Incr(ctx, fmt.Sprintf("cache_misses:%s", today))
	qo.redisClient.Expire(ctx, fmt.Sprintf("cache_misses:%s", today), 7*24*time.Hour)
}

// Analysis and optimization methods

// startQueryAnalysis starts background query analysis
func (qo *QueryOptimizer) startQueryAnalysis() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			qo.analyzeQueryPerformance()
			qo.generateOptimizationSuggestions()
		}
	}
}

// analyzeQueryPerformance analyzes query performance patterns
func (qo *QueryOptimizer) analyzeQueryPerformance() {
	var slowQueries []*QueryStats
	
	qo.queryStats.Range(func(key, value interface{}) bool {
		stats := value.(*QueryStats)
		if stats.AverageDuration > qo.slowQueryThreshold {
			slowQueries = append(slowQueries, stats)
		}
		return true
	})
	
	if len(slowQueries) > 0 {
		fmt.Printf("Found %d queries with average duration > %v\n", 
			len(slowQueries), qo.slowQueryThreshold)
		
		// Log top 10 slowest queries
		for i, query := range slowQueries {
			if i >= 10 {
				break
			}
			fmt.Printf("  %d. %s (Avg: %v, Calls: %d)\n", 
				i+1, query.QueryHash, query.AverageDuration, query.TotalCalls)
		}
	}
}

// generateOptimizationSuggestions generates query optimization suggestions
func (qo *QueryOptimizer) generateOptimizationSuggestions() {
	// This would analyze query patterns and suggest optimizations
	// For example:
	// - Missing indexes
	// - N+1 query problems
	// - Inefficient JOINs
	// - Large result sets without pagination
	
	suggestions := []string{}
	
	qo.queryStats.Range(func(key, value interface{}) bool {
		stats := value.(*QueryStats)
		
		// Check for N+1 problems (many small similar queries)
		if stats.TotalCalls > 100 && stats.AverageDuration < 10*time.Millisecond {
			suggestions = append(suggestions, 
				fmt.Sprintf("Possible N+1 problem: %s (Called %d times)", 
					stats.QueryHash, stats.TotalCalls))
		}
		
		// Check for queries without WHERE clauses on large tables
		if strings.Contains(stats.Query, "SELECT") && 
			!strings.Contains(strings.ToUpper(stats.Query), "WHERE") &&
			stats.AverageDuration > 100*time.Millisecond {
			suggestions = append(suggestions, 
				fmt.Sprintf("Query without WHERE clause: %s", stats.QueryHash))
		}
		
		return true
	})
	
	if len(suggestions) > 0 {
		fmt.Printf("Query Optimization Suggestions:\n")
		for _, suggestion := range suggestions {
			fmt.Printf("  - %s\n", suggestion)
		}
	}
}

// startQueryCacheCleanup starts background cache cleanup
func (qo *QueryOptimizer) startQueryCacheCleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			qo.cleanupExpiredCache()
		}
	}
}

// cleanupExpiredCache removes expired cache entries
func (qo *QueryOptimizer) cleanupExpiredCache() {
	ctx := context.Background()
	
	// Get all cache keys
	keys, err := qo.redisClient.Keys(ctx, "query_cache:*").Result()
	if err != nil {
		return
	}
	
	var expiredKeys []string
	
	for _, key := range keys {
		cacheData, err := qo.redisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		
		var entry QueryCacheEntry
		if err := json.Unmarshal([]byte(cacheData), &entry); err != nil {
			expiredKeys = append(expiredKeys, key)
			continue
		}
		
		if time.Now().After(entry.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	if len(expiredKeys) > 0 {
		qo.redisClient.Del(ctx, expiredKeys...)
		fmt.Printf("Cleaned up %d expired cache entries\n", len(expiredKeys))
	}
}

// GetQueryStatistics returns query performance statistics
func (qo *QueryOptimizer) GetQueryStatistics() map[string]*QueryStats {
	stats := make(map[string]*QueryStats)
	
	qo.queryStats.Range(func(key, value interface{}) bool {
		hash := key.(string)
		stat := value.(*QueryStats)
		stats[hash] = stat
		return true
	})
	
	return stats
}

// GetSlowQueries returns recent slow queries
func (qo *QueryOptimizer) GetSlowQueries(limit int) []*SlowQuery {
	var queries []*SlowQuery
	count := 0
	
	qo.slowQueries.Range(func(key, value interface{}) bool {
		if count >= limit {
			return false
		}
		
		query := value.(*SlowQuery)
		queries = append(queries, query)
		count++
		return true
	})
	
	return queries
}

// Helper functions

import (
	"crypto/md5"
	"encoding/json"
	"reflect"
	"regexp"
)

func hashQuery(query string) string {
	// Normalize query by removing values and whitespace
	normalized := regexp.MustCompile(`\s+`).ReplaceAllString(query, " ")
	normalized = regexp.MustCompile(`\$\d+`).ReplaceAllString(normalized, "?")
	normalized = strings.TrimSpace(normalized)
	
	hash := md5.Sum([]byte(normalized))
	return fmt.Sprintf("%x", hash)[:12]
}

func sanitizeQuery(query string) string {
	// Remove potential sensitive data from query for logging
	sanitized := regexp.MustCompile(`('[^']*'|"[^"]*")`).ReplaceAllString(query, "'***'")
	return sanitized
}

func extractTables(query string) []string {
	// Simple table extraction - in production, use a proper SQL parser
	var tables []string
	
	// Look for FROM and JOIN clauses
	fromRegex := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+([a-zA-Z_]\w*)`)
	matches := fromRegex.FindAllStringSubmatch(query, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			tables = append(tables, match[1])
		}
	}
	
	return tables
}

func generateCacheKey(operation string, query interface{}, args ...interface{}) string {
	data := struct {
		Operation string      `json:"operation"`
		Query     interface{} `json:"query"`
		Args      interface{} `json:"args"`
	}{
		Operation: operation,
		Query:     query,
		Args:      args,
	}
	
	jsonData, _ := json.Marshal(data)
	hash := md5.Sum(jsonData)
	return fmt.Sprintf("%x", hash)[:16]
}

func mapToStruct(src interface{}, dest interface{}) error {
	jsonData, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, dest)
}

func isSlice(value interface{}) bool {
	return reflect.TypeOf(value).Kind() == reflect.Slice
}

func getTypeName(value interface{}) string {
	return reflect.TypeOf(value).String()
}

func getUserIDFromContext(ctx context.Context) string {
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}
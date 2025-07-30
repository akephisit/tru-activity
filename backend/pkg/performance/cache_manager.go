package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// CacheManager handles all caching operations
type CacheManager struct {
	redisClient *redis.Client
	db         *gorm.DB
}

// CacheConfig defines caching configuration for different types
type CacheConfig struct {
	KeyPrefix   string
	TTL         time.Duration
	Tags        []string
	MaxSize     int
	EnableRedis bool
}

// Cache configurations for different entity types
var (
	UserCacheConfig = CacheConfig{
		KeyPrefix: "user:",
		TTL:       15 * time.Minute,
		Tags:      []string{"users"},
	}
	
	ActivityCacheConfig = CacheConfig{
		KeyPrefix: "activity:",
		TTL:       10 * time.Minute,
		Tags:      []string{"activities"},
	}
	
	FacultyCacheConfig = CacheConfig{
		KeyPrefix: "faculty:",
		TTL:       30 * time.Minute,
		Tags:      []string{"faculties"},
	}
	
	ParticipationCacheConfig = CacheConfig{
		KeyPrefix: "participation:",
		TTL:       5 * time.Minute,
		Tags:      []string{"participations"},
	}
	
	MetricsCacheConfig = CacheConfig{
		KeyPrefix: "metrics:",
		TTL:       2 * time.Minute,
		Tags:      []string{"metrics", "analytics"},
	}
	
	QueryCacheConfig = CacheConfig{
		KeyPrefix: "query:",
		TTL:       5 * time.Minute,
		Tags:      []string{"queries"},
	}
)

// NewCacheManager creates a new cache manager
func NewCacheManager(redisClient *redis.Client, db *gorm.DB) *CacheManager {
	return &CacheManager{
		redisClient: redisClient,
		db:         db,
	}
}

// Set stores a value in cache with the given configuration
func (cm *CacheManager) Set(ctx context.Context, key string, value interface{}, config CacheConfig) error {
	// Serialize value
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %v", err)
	}
	
	// Build full key
	fullKey := config.KeyPrefix + key
	
	// Store in Redis
	pipe := cm.redisClient.Pipeline()
	
	// Set the main key-value pair
	pipe.Set(ctx, fullKey, data, config.TTL)
	
	// Add to tag sets for cache invalidation
	for _, tag := range config.Tags {
		tagKey := "tag:" + tag
		pipe.SAdd(ctx, tagKey, fullKey)
		pipe.Expire(ctx, tagKey, config.TTL+time.Hour) // Keep tags longer
	}
	
	// Execute pipeline
	_, err = pipe.Exec(ctx)
	return err
}

// Get retrieves a value from cache
func (cm *CacheManager) Get(ctx context.Context, key string, config CacheConfig, dest interface{}) error {
	fullKey := config.KeyPrefix + key
	
	data, err := cm.redisClient.Get(ctx, fullKey).Result()
	if err != nil {
		return err
	}
	
	return json.Unmarshal([]byte(data), dest)
}

// GetOrSet retrieves from cache or sets if not found
func (cm *CacheManager) GetOrSet(ctx context.Context, key string, config CacheConfig, fetchFunc func() (interface{}, error), dest interface{}) error {
	// Try to get from cache first
	err := cm.Get(ctx, key, config, dest)
	if err == nil {
		return nil // Cache hit
	}
	
	// Cache miss - fetch data
	value, err := fetchFunc()
	if err != nil {
		return err
	}
	
	// Store in cache
	if err := cm.Set(ctx, key, value, config); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: failed to cache value for key %s: %v\n", key, err)
	}
	
	// Marshal and unmarshal to ensure type consistency
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, dest)
}

// Delete removes a key from cache
func (cm *CacheManager) Delete(ctx context.Context, key string, config CacheConfig) error {
	fullKey := config.KeyPrefix + key
	return cm.redisClient.Del(ctx, fullKey).Err()
}

// InvalidateByTag removes all cache entries with the given tag
func (cm *CacheManager) InvalidateByTag(ctx context.Context, tag string) error {
	tagKey := "tag:" + tag
	
	// Get all keys with this tag
	keys, err := cm.redisClient.SMembers(ctx, tagKey).Result()
	if err != nil {
		return err
	}
	
	if len(keys) == 0 {
		return nil
	}
	
	// Delete all keys
	pipe := cm.redisClient.Pipeline()
	pipe.Del(ctx, keys...)
	pipe.Del(ctx, tagKey) // Also delete the tag set
	
	_, err = pipe.Exec(ctx)
	return err
}

// Query cache for GraphQL queries
func (cm *CacheManager) CacheQuery(ctx context.Context, queryHash string, variables map[string]interface{}, result interface{}) error {
	key := fmt.Sprintf("%s:%s", queryHash, cm.hashVariables(variables))
	return cm.Set(ctx, key, result, QueryCacheConfig)
}

func (cm *CacheManager) GetCachedQuery(ctx context.Context, queryHash string, variables map[string]interface{}, dest interface{}) error {
	key := fmt.Sprintf("%s:%s", queryHash, cm.hashVariables(variables))
	return cm.Get(ctx, key, QueryCacheConfig, dest)
}

func (cm *CacheManager) hashVariables(variables map[string]interface{}) string {
	// Simple hash of variables for cache key
	data, _ := json.Marshal(variables)
	return fmt.Sprintf("%x", data)[:16] // Use first 16 chars of hex
}

// Specific caching methods for different entities

// User caching
func (cm *CacheManager) CacheUser(ctx context.Context, userID string, user interface{}) error {
	return cm.Set(ctx, userID, user, UserCacheConfig)
}

func (cm *CacheManager) GetCachedUser(ctx context.Context, userID string, dest interface{}) error {
	return cm.Get(ctx, userID, UserCacheConfig, dest)
}

func (cm *CacheManager) InvalidateUser(ctx context.Context, userID string) error {
	return cm.Delete(ctx, userID, UserCacheConfig)
}

// Activity caching
func (cm *CacheManager) CacheActivity(ctx context.Context, activityID string, activity interface{}) error {
	return cm.Set(ctx, activityID, activity, ActivityCacheConfig)
}

func (cm *CacheManager) GetCachedActivity(ctx context.Context, activityID string, dest interface{}) error {
	return cm.Get(ctx, activityID, ActivityCacheConfig, dest)
}

func (cm *CacheManager) InvalidateActivity(ctx context.Context, activityID string) error {
	// When an activity is invalidated, also invalidate related caches
	pipe := cm.redisClient.Pipeline()
	
	// Delete the activity itself
	fullKey := ActivityCacheConfig.KeyPrefix + activityID
	pipe.Del(ctx, fullKey)
	
	// Invalidate related queries
	cm.InvalidateByTag(ctx, "activities")
	
	_, err := pipe.Exec(ctx)
	return err
}

// Faculty caching
func (cm *CacheManager) CacheFaculty(ctx context.Context, facultyID string, faculty interface{}) error {
	return cm.Set(ctx, facultyID, faculty, FacultyCacheConfig)
}

func (cm *CacheManager) GetCachedFaculty(ctx context.Context, facultyID string, dest interface{}) error {
	return cm.Get(ctx, facultyID, FacultyCacheConfig, dest)
}

// Metrics caching with automatic refresh
func (cm *CacheManager) CacheMetrics(ctx context.Context, metricsKey string, metrics interface{}) error {
	return cm.Set(ctx, metricsKey, metrics, MetricsCacheConfig)
}

func (cm *CacheManager) GetCachedMetrics(ctx context.Context, metricsKey string, dest interface{}) error {
	return cm.Get(ctx, metricsKey, MetricsCacheConfig, dest)
}

func (cm *CacheManager) GetOrComputeMetrics(ctx context.Context, metricsKey string, computeFunc func() (interface{}, error), dest interface{}) error {
	return cm.GetOrSet(ctx, metricsKey, MetricsCacheConfig, computeFunc, dest)
}

// Complex caching patterns

// Cache with write-through pattern
func (cm *CacheManager) WriteThrough(ctx context.Context, key string, value interface{}, config CacheConfig, writeFunc func() error) error {
	// Write to database first
	if err := writeFunc(); err != nil {
		return err
	}
	
	// Then update cache
	return cm.Set(ctx, key, value, config)
}

// Cache with write-behind pattern (async)
func (cm *CacheManager) WriteBehind(ctx context.Context, key string, value interface{}, config CacheConfig, writeFunc func() error) error {
	// Update cache immediately
	if err := cm.Set(ctx, key, value, config); err != nil {
		return err
	}
	
	// Write to database asynchronously
	go func() {
		if err := writeFunc(); err != nil {
			// Log error and potentially implement retry logic
			fmt.Printf("Write-behind failed for key %s: %v\n", key, err)
			
			// Remove from cache if database write failed
			cm.Delete(context.Background(), key, config)
		}
	}()
	
	return nil
}

// Bulk operations
func (cm *CacheManager) SetMany(ctx context.Context, keyValues map[string]interface{}, config CacheConfig) error {
	pipe := cm.redisClient.Pipeline()
	
	for key, value := range keyValues {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value for key %s: %v", key, err)
		}
		
		fullKey := config.KeyPrefix + key
		pipe.Set(ctx, fullKey, data, config.TTL)
		
		// Add to tag sets
		for _, tag := range config.Tags {
			tagKey := "tag:" + tag
			pipe.SAdd(ctx, tagKey, fullKey)
			pipe.Expire(ctx, tagKey, config.TTL+time.Hour)
		}
	}
	
	_, err := pipe.Exec(ctx)
	return err
}

func (cm *CacheManager) GetMany(ctx context.Context, keys []string, config CacheConfig) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}
	
	// Build full keys
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = config.KeyPrefix + key
	}
	
	// Get all values
	values, err := cm.redisClient.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return nil, err
	}
	
	// Parse results
	result := make(map[string]interface{})
	for i, value := range values {
		if value != nil && value != redis.Nil {
			var parsed interface{}
			if err := json.Unmarshal([]byte(value.(string)), &parsed); err == nil {
				result[keys[i]] = parsed
			}
		}
	}
	
	return result, nil
}

// Cache warming - preload frequently accessed data
func (cm *CacheManager) WarmCache(ctx context.Context) error {
	// Warm up faculty cache
	go cm.warmFaculties(ctx)
	
	// Warm up active activities
	go cm.warmActiveActivities(ctx)
	
	// Warm up system metrics
	go cm.warmSystemMetrics(ctx)
	
	return nil
}

func (cm *CacheManager) warmFaculties(ctx context.Context) {
	var faculties []interface{}
	if err := cm.db.WithContext(ctx).Find(&faculties).Error; err != nil {
		fmt.Printf("Failed to warm faculty cache: %v\n", err)
		return
	}
	
	keyValues := make(map[string]interface{})
	for _, faculty := range faculties {
		if f, ok := faculty.(map[string]interface{}); ok {
			if id, exists := f["id"]; exists {
				keyValues[fmt.Sprintf("%v", id)] = faculty
			}
		}
	}
	
	cm.SetMany(ctx, keyValues, FacultyCacheConfig)
}

func (cm *CacheManager) warmActiveActivities(ctx context.Context) {
	var activities []interface{}
	if err := cm.db.WithContext(ctx).Where("status = ?", "ACTIVE").Find(&activities).Error; err != nil {
		fmt.Printf("Failed to warm activity cache: %v\n", err)
		return
	}
	
	keyValues := make(map[string]interface{})
	for _, activity := range activities {
		if a, ok := activity.(map[string]interface{}); ok {
			if id, exists := a["id"]; exists {
				keyValues[fmt.Sprintf("%v", id)] = activity
			}
		}
	}
	
	cm.SetMany(ctx, keyValues, ActivityCacheConfig)
}

func (cm *CacheManager) warmSystemMetrics(ctx context.Context) {
	// Pre-compute and cache common metrics
	metrics := map[string]func() (interface{}, error){
		"daily_stats": func() (interface{}, error) {
			// Compute daily statistics
			return map[string]interface{}{
				"total_users":       0, // Implement actual computation
				"active_activities": 0,
				"daily_scans":      0,
			}, nil
		},
		"faculty_counts": func() (interface{}, error) {
			// Compute faculty statistics
			return map[string]interface{}{
				"total_faculties": 0,
				"total_students":  0,
			}, nil
		},
	}
	
	for key, computeFunc := range metrics {
		if value, err := computeFunc(); err == nil {
			cm.Set(ctx, key, value, MetricsCacheConfig)
		}
	}
}

// Cache statistics and monitoring
func (cm *CacheManager) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := cm.redisClient.Info(ctx, "memory").Result()
	if err != nil {
		return nil, err
	}
	
	stats := map[string]interface{}{
		"redis_info": info,
		"timestamp":  time.Now().Unix(),
	}
	
	// Get hit/miss ratios for different cache types
	cacheTypes := []string{"user", "activity", "faculty", "metrics"}
	for _, cacheType := range cacheTypes {
		hitKey := fmt.Sprintf("cache_hits:%s", cacheType)
		missKey := fmt.Sprintf("cache_misses:%s", cacheType)
		
		hits, _ := cm.redisClient.Get(ctx, hitKey).Int64()
		misses, _ := cm.redisClient.Get(ctx, missKey).Int64()
		
		total := hits + misses
		hitRatio := 0.0
		if total > 0 {
			hitRatio = float64(hits) / float64(total)
		}
		
		stats[cacheType] = map[string]interface{}{
			"hits":      hits,
			"misses":    misses,
			"hit_ratio": hitRatio,
		}
	}
	
	return stats, nil
}

// Record cache hit/miss for monitoring
func (cm *CacheManager) recordCacheHit(ctx context.Context, cacheType string) {
	key := fmt.Sprintf("cache_hits:%s", cacheType)
	cm.redisClient.Incr(ctx, key)
	cm.redisClient.Expire(ctx, key, 24*time.Hour)
}

func (cm *CacheManager) recordCacheMiss(ctx context.Context, cacheType string) {
	key := fmt.Sprintf("cache_misses:%s", cacheType)
	cm.redisClient.Incr(ctx, key)
	cm.redisClient.Expire(ctx, key, 24*time.Hour)
}

// Enhanced Get method with hit/miss tracking
func (cm *CacheManager) GetWithStats(ctx context.Context, key string, config CacheConfig, dest interface{}) error {
	cacheType := strings.TrimSuffix(config.KeyPrefix, ":")
	
	err := cm.Get(ctx, key, config, dest)
	if err == nil {
		cm.recordCacheHit(ctx, cacheType)
	} else {
		cm.recordCacheMiss(ctx, cacheType)
	}
	
	return err
}

// Cache middleware for automatic caching
type CacheMiddleware struct {
	cacheManager *CacheManager
}

func NewCacheMiddleware(cacheManager *CacheManager) *CacheMiddleware {
	return &CacheMiddleware{
		cacheManager: cacheManager,
	}
}

// Auto-cache GraphQL queries based on operation name
func (cm *CacheMiddleware) CacheQuery(ctx context.Context, operationName string, variables map[string]interface{}, result interface{}) bool {
	// Define cacheable operations
	cacheableOperations := map[string]CacheConfig{
		"GetActivities":      ActivityCacheConfig,
		"GetFaculties":       FacultyCacheConfig,
		"GetSystemMetrics":   MetricsCacheConfig,
		"GetFacultyMetrics":  MetricsCacheConfig,
		"GetUser":            UserCacheConfig,
	}
	
	config, cacheable := cacheableOperations[operationName]
	if !cacheable {
		return false
	}
	
	// Try to get from cache
	key := fmt.Sprintf("%s:%s", operationName, cm.cacheManager.hashVariables(variables))
	if err := cm.cacheManager.Get(ctx, key, config, result); err == nil {
		return true // Cache hit
	}
	
	return false // Cache miss
}

func (cm *CacheMiddleware) StoreQueryResult(ctx context.Context, operationName string, variables map[string]interface{}, result interface{}) {
	// Define cacheable operations
	cacheableOperations := map[string]CacheConfig{
		"GetActivities":      ActivityCacheConfig,
		"GetFaculties":       FacultyCacheConfig,
		"GetSystemMetrics":   MetricsCacheConfig,
		"GetFacultyMetrics":  MetricsCacheConfig,
		"GetUser":            UserCacheConfig,
	}
	
	config, cacheable := cacheableOperations[operationName]
	if !cacheable {
		return
	}
	
	key := fmt.Sprintf("%s:%s", operationName, cm.cacheManager.hashVariables(variables))
	cm.cacheManager.Set(ctx, key, result, config)
}
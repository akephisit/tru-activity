package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// PerformanceMonitor handles performance monitoring and alerting
type PerformanceMonitor struct {
	db             *gorm.DB
	redisClient    *redis.Client
	metrics        sync.Map
	alertThresholds map[string]AlertThreshold
	mu             sync.RWMutex
}

// MetricPoint represents a single performance metric measurement
type MetricPoint struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Unit      string                 `json:"unit"`
	Tags      map[string]string      `json:"tags"`
	Fields    map[string]interface{} `json:"fields"`
	Timestamp time.Time              `json:"timestamp"`
}

// AlertThreshold defines when to trigger performance alerts
type AlertThreshold struct {
	MetricName   string        `json:"metric_name"`
	WarningLevel float64       `json:"warning_level"`
	CriticalLevel float64      `json:"critical_level"`
	Duration     time.Duration `json:"duration"`
	Enabled      bool          `json:"enabled"`
}

// PerformanceAlert represents a performance alert
type PerformanceAlert struct {
	ID          string                 `json:"id"`
	MetricName  string                 `json:"metric_name"`
	Level       string                 `json:"level"` // WARNING, CRITICAL
	Value       float64                `json:"value"`
	Threshold   float64                `json:"threshold"`
	Message     string                 `json:"message"`
	Tags        map[string]string      `json:"tags"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at"`
}

// SystemHealth represents overall system health status
type SystemHealth struct {
	Status        string                 `json:"status"` // HEALTHY, DEGRADED, UNHEALTHY
	Timestamp     time.Time              `json:"timestamp"`
	Metrics       map[string]float64     `json:"metrics"`
	Alerts        []PerformanceAlert     `json:"alerts"`
	Uptime        time.Duration          `json:"uptime"`
	Version       string                 `json:"version"`
	Details       map[string]interface{} `json:"details"`
}

// Database performance metrics
type DatabaseMetrics struct {
	ActiveConnections  int           `json:"active_connections"`
	IdleConnections    int           `json:"idle_connections"`
	MaxConnections     int           `json:"max_connections"`
	QueryDuration      time.Duration `json:"query_duration"`
	SlowQueries        int64         `json:"slow_queries"`
	Deadlocks          int64         `json:"deadlocks"`
	CacheHitRatio      float64       `json:"cache_hit_ratio"`
}

// Redis performance metrics
type RedisMetrics struct {
	UsedMemory        int64   `json:"used_memory"`
	MaxMemory         int64   `json:"max_memory"`
	MemoryUsageRatio  float64 `json:"memory_usage_ratio"`
	ConnectedClients  int     `json:"connected_clients"`
	KeyspaceHits      int64   `json:"keyspace_hits"`
	KeyspaceMisses    int64   `json:"keyspace_misses"`
	HitRate           float64 `json:"hit_rate"`
	CacheSize         int64   `json:"cache_size"`
}

// Application metrics
type ApplicationMetrics struct {
	RequestsPerSecond  float64       `json:"requests_per_second"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	ErrorRate          float64       `json:"error_rate"`
	ActiveSessions     int           `json:"active_sessions"`
	QRScansPerMinute   float64       `json:"qr_scans_per_minute"`
	ConcurrentUsers    int           `json:"concurrent_users"`
	MemoryUsage        int64         `json:"memory_usage"`
	CPUUsage           float64       `json:"cpu_usage"`
	GoroutineCount     int           `json:"goroutine_count"`
}

var startTime = time.Now()

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(db *gorm.DB, redisClient *redis.Client) *PerformanceMonitor {
	pm := &PerformanceMonitor{
		db:          db,
		redisClient: redisClient,
		alertThresholds: map[string]AlertThreshold{
			"response_time": {
				MetricName:    "response_time",
				WarningLevel:  1000,  // 1 second
				CriticalLevel: 5000,  // 5 seconds
				Duration:      5 * time.Minute,
				Enabled:       true,
			},
			"error_rate": {
				MetricName:    "error_rate",
				WarningLevel:  5.0,   // 5%
				CriticalLevel: 10.0,  // 10%
				Duration:      5 * time.Minute,
				Enabled:       true,
			},
			"cpu_usage": {
				MetricName:    "cpu_usage",
				WarningLevel:  80.0,  // 80%
				CriticalLevel: 95.0,  // 95%
				Duration:      2 * time.Minute,
				Enabled:       true,
			},
			"memory_usage": {
				MetricName:    "memory_usage",
				WarningLevel:  80.0,  // 80%
				CriticalLevel: 95.0,  // 95%
				Duration:      2 * time.Minute,
				Enabled:       true,
			},
			"database_connections": {
				MetricName:    "database_connections",
				WarningLevel:  80.0,  // 80% of max connections
				CriticalLevel: 95.0,  // 95% of max connections
				Duration:      1 * time.Minute,
				Enabled:       true,
			},
		},
	}
	
	// Start background monitoring
	go pm.startMetricsCollection()
	go pm.startAlertMonitoring()
	
	return pm
}

// RecordMetric records a performance metric
func (pm *PerformanceMonitor) RecordMetric(ctx context.Context, metric MetricPoint) error {
	// Store in Redis for real-time monitoring
	key := fmt.Sprintf("metrics:%s:%d", metric.Name, metric.Timestamp.Unix())
	
	metricData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %v", err)
	}
	
	pipe := pm.redisClient.Pipeline()
	
	// Store individual metric
	pipe.Set(ctx, key, metricData, 24*time.Hour)
	
	// Add to time series
	pipe.ZAdd(ctx, fmt.Sprintf("metrics:series:%s", metric.Name), redis.Z{
		Score:  float64(metric.Timestamp.Unix()),
		Member: metric.Value,
	})
	
	// Keep only last 24 hours of data
	dayAgo := time.Now().Add(-24 * time.Hour).Unix()
	pipe.ZRemRangeByScore(ctx, fmt.Sprintf("metrics:series:%s", metric.Name), "-inf", fmt.Sprintf("%d", dayAgo))
	
	// Update rolling averages
	pm.updateRollingAverages(ctx, pipe, metric)
	
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store metric: %v", err)
	}
	
	// Check against thresholds
	go pm.checkThresholds(ctx, metric)
	
	return nil
}

// updateRollingAverages updates rolling averages for metrics
func (pm *PerformanceMonitor) updateRollingAverages(ctx context.Context, pipe redis.Pipeliner, metric MetricPoint) {
	// 1-minute rolling average
	minuteKey := fmt.Sprintf("avg:1m:%s:%d", metric.Name, metric.Timestamp.Unix()/60)
	pipe.LPush(ctx, minuteKey, metric.Value)
	pipe.LTrim(ctx, minuteKey, 0, 59) // Keep last 60 values
	pipe.Expire(ctx, minuteKey, 2*time.Minute)
	
	// 5-minute rolling average
	fiveMinuteKey := fmt.Sprintf("avg:5m:%s:%d", metric.Name, metric.Timestamp.Unix()/300)
	pipe.LPush(ctx, fiveMinuteKey, metric.Value)
	pipe.LTrim(ctx, fiveMinuteKey, 0, 299) // Keep last 300 values
	pipe.Expire(ctx, fiveMinuteKey, 10*time.Minute)
	
	// 1-hour rolling average
	hourKey := fmt.Sprintf("avg:1h:%s:%d", metric.Name, metric.Timestamp.Unix()/3600)
	pipe.LPush(ctx, hourKey, metric.Value)
	pipe.LTrim(ctx, hourKey, 0, 3599) // Keep last 3600 values
	pipe.Expire(ctx, hourKey, 2*time.Hour)
}

// RecordDatabaseMetrics records database performance metrics
func (pm *PerformanceMonitor) RecordDatabaseMetrics(ctx context.Context) error {
	// Get database connection stats
	sqlDB, err := pm.db.DB()
	if err != nil {
		return err
	}
	
	stats := sqlDB.Stats()
	
	metrics := DatabaseMetrics{
		ActiveConnections: stats.InUse,
		IdleConnections:   stats.Idle,
		MaxConnections:    stats.MaxOpenConnections,
	}
	
	// Record individual metrics
	timestamp := time.Now()
	
	pm.RecordMetric(ctx, MetricPoint{
		Name:      "database_active_connections",
		Value:     float64(metrics.ActiveConnections),
		Unit:      "count",
		Timestamp: timestamp,
		Tags:      map[string]string{"component": "database"},
	})
	
	pm.RecordMetric(ctx, MetricPoint{
		Name:      "database_idle_connections",
		Value:     float64(metrics.IdleConnections),
		Unit:      "count",
		Timestamp: timestamp,
		Tags:      map[string]string{"component": "database"},
	})
	
	connectionUsage := float64(metrics.ActiveConnections) / float64(metrics.MaxConnections) * 100
	pm.RecordMetric(ctx, MetricPoint{
		Name:      "database_connection_usage",
		Value:     connectionUsage,
		Unit:      "percent",
		Timestamp: timestamp,
		Tags:      map[string]string{"component": "database"},
	})
	
	return nil
}

// RecordRedisMetrics records Redis performance metrics
func (pm *PerformanceMonitor) RecordRedisMetrics(ctx context.Context) error {
	// Get Redis info
	info, err := pm.redisClient.Info(ctx, "memory", "stats").Result()
	if err != nil {
		return err
	}
	
	// Parse Redis info (simplified)
	metrics := RedisMetrics{}
	
	// This would need proper parsing of Redis INFO command output
	// For now, using placeholder values
	
	timestamp := time.Now()
	
	pm.RecordMetric(ctx, MetricPoint{
		Name:      "redis_memory_usage",
		Value:     float64(metrics.UsedMemory),
		Unit:      "bytes",
		Timestamp: timestamp,
		Tags:      map[string]string{"component": "redis"},
	})
	
	pm.RecordMetric(ctx, MetricPoint{
		Name:      "redis_connected_clients",
		Value:     float64(metrics.ConnectedClients),
		Unit:      "count",
		Timestamp: timestamp,
		Tags:      map[string]string{"component": "redis"},
	})
	
	return nil
}

// RecordApplicationMetrics records application-specific metrics
func (pm *PerformanceMonitor) RecordApplicationMetrics(ctx context.Context) error {
	timestamp := time.Now()
	
	// Memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	pm.RecordMetric(ctx, MetricPoint{
		Name:      "memory_usage_bytes",
		Value:     float64(memStats.Alloc),
		Unit:      "bytes",
		Timestamp: timestamp,
		Tags:      map[string]string{"component": "application"},
	})
	
	pm.RecordMetric(ctx, MetricPoint{
		Name:      "memory_usage_percent",
		Value:     float64(memStats.Alloc) / float64(memStats.Sys) * 100,
		Unit:      "percent",
		Timestamp: timestamp,
		Tags:      map[string]string{"component": "application"},
	})
	
	// Goroutine count
	pm.RecordMetric(ctx, MetricPoint{
		Name:      "goroutine_count",
		Value:     float64(runtime.NumGoroutine()),
		Unit:      "count",
		Timestamp: timestamp,
		Tags:      map[string]string{"component": "application"},
	})
	
	// GC statistics
	pm.RecordMetric(ctx, MetricPoint{
		Name:      "gc_cycles",
		Value:     float64(memStats.NumGC),
		Unit:      "count",
		Timestamp: timestamp,
		Tags:      map[string]string{"component": "gc"},
	})
	
	return nil
}

// GetSystemHealth returns overall system health status
func (pm *PerformanceMonitor) GetSystemHealth(ctx context.Context) (*SystemHealth, error) {
	health := &SystemHealth{
		Status:    "HEALTHY",
		Timestamp: time.Now(),
		Uptime:    time.Since(startTime),
		Version:   "1.0.0",
		Metrics:   make(map[string]float64),
		Details:   make(map[string]interface{}),
	}
	
	// Get recent alerts
	alerts, err := pm.getActiveAlerts(ctx)
	if err != nil {
		return nil, err
	}
	health.Alerts = alerts
	
	// Determine status based on alerts
	for _, alert := range alerts {
		if alert.Level == "CRITICAL" {
			health.Status = "UNHEALTHY"
			break
		} else if alert.Level == "WARNING" && health.Status == "HEALTHY" {
			health.Status = "DEGRADED"
		}
	}
	
	// Get latest metrics
	metricNames := []string{
		"response_time", "error_rate", "cpu_usage", "memory_usage_percent",
		"database_connection_usage", "redis_memory_usage",
	}
	
	for _, metricName := range metricNames {
		if value, err := pm.getLatestMetricValue(ctx, metricName); err == nil {
			health.Metrics[metricName] = value
		}
	}
	
	// Add system details
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	health.Details["memory_alloc"] = memStats.Alloc
	health.Details["memory_sys"] = memStats.Sys
	health.Details["gc_cycles"] = memStats.NumGC
	health.Details["goroutines"] = runtime.NumGoroutine()
	
	return health, nil
}

// getLatestMetricValue gets the latest value for a metric
func (pm *PerformanceMonitor) getLatestMetricValue(ctx context.Context, metricName string) (float64, error) {
	// Get latest value from time series
	values, err := pm.redisClient.ZRevRange(ctx, fmt.Sprintf("metrics:series:%s", metricName), 0, 0).Result()
	if err != nil || len(values) == 0 {
		return 0, fmt.Errorf("no data available for metric %s", metricName)
	}
	
	// Parse the value
	var value float64
	if _, err := fmt.Sscanf(values[0], "%f", &value); err != nil {
		return 0, err
	}
	
	return value, nil
}

// getActiveAlerts gets currently active alerts
func (pm *PerformanceMonitor) getActiveAlerts(ctx context.Context) ([]PerformanceAlert, error) {
	alertsJSON, err := pm.redisClient.LRange(ctx, "alerts:active", 0, -1).Result()
	if err != nil {
		return nil, err
	}
	
	var alerts []PerformanceAlert
	for _, alertJSON := range alertsJSON {
		var alert PerformanceAlert
		if err := json.Unmarshal([]byte(alertJSON), &alert); err == nil {
			alerts = append(alerts, alert)
		}
	}
	
	return alerts, nil
}

// checkThresholds checks if metric exceeds thresholds
func (pm *PerformanceMonitor) checkThresholds(ctx context.Context, metric MetricPoint) {
	pm.mu.RLock()
	threshold, exists := pm.alertThresholds[metric.Name]
	pm.mu.RUnlock()
	
	if !exists || !threshold.Enabled {
		return
	}
	
	var level string
	var thresholdValue float64
	
	if metric.Value >= threshold.CriticalLevel {
		level = "CRITICAL"
		thresholdValue = threshold.CriticalLevel
	} else if metric.Value >= threshold.WarningLevel {
		level = "WARNING"
		thresholdValue = threshold.WarningLevel
	} else {
		// Check if we need to resolve any existing alerts
		pm.resolveAlert(ctx, metric.Name)
		return
	}
	
	// Create alert
	alert := PerformanceAlert{
		ID:         fmt.Sprintf("%s_%s_%d", metric.Name, level, metric.Timestamp.Unix()),
		MetricName: metric.Name,
		Level:      level,
		Value:      metric.Value,
		Threshold:  thresholdValue,
		Message:    fmt.Sprintf("%s %s threshold exceeded: %.2f > %.2f", metric.Name, level, metric.Value, thresholdValue),
		Tags:       metric.Tags,
		Timestamp:  metric.Timestamp,
		Resolved:   false,
	}
	
	// Store alert
	pm.storeAlert(ctx, alert)
}

// storeAlert stores an alert
func (pm *PerformanceMonitor) storeAlert(ctx context.Context, alert PerformanceAlert) {
	alertJSON, err := json.Marshal(alert)
	if err != nil {
		return
	}
	
	pipe := pm.redisClient.Pipeline()
	
	// Add to active alerts
	pipe.LPush(ctx, "alerts:active", alertJSON)
	pipe.LTrim(ctx, "alerts:active", 0, 99) // Keep last 100 alerts
	
	// Add to all alerts history
	pipe.LPush(ctx, "alerts:history", alertJSON)
	pipe.LTrim(ctx, "alerts:history", 0, 999) // Keep last 1000 alerts
	
	// Set expiry
	pipe.Expire(ctx, "alerts:active", 24*time.Hour)
	pipe.Expire(ctx, "alerts:history", 7*24*time.Hour)
	
	// Publish to real-time notification system
	pipe.Publish(ctx, "performance_alerts", alertJSON)
	
	pipe.Exec(ctx)
	
	fmt.Printf("PERFORMANCE ALERT: %s - %s: %.2f > %.2f\n", 
		alert.Level, alert.MetricName, alert.Value, alert.Threshold)
}

// resolveAlert resolves an active alert
func (pm *PerformanceMonitor) resolveAlert(ctx context.Context, metricName string) {
	// Get active alerts
	alertsJSON, err := pm.redisClient.LRange(ctx, "alerts:active", 0, -1).Result()
	if err != nil {
		return
	}
	
	var updatedAlerts []string
	now := time.Now()
	
	for _, alertJSON := range alertsJSON {
		var alert PerformanceAlert
		if err := json.Unmarshal([]byte(alertJSON), &alert); err != nil {
			continue
		}
		
		if alert.MetricName == metricName && !alert.Resolved {
			// Resolve this alert
			alert.Resolved = true
			alert.ResolvedAt = &now
			
			if updatedJSON, err := json.Marshal(alert); err == nil {
				// Publish resolution
				pm.redisClient.Publish(ctx, "performance_alerts_resolved", updatedJSON)
				fmt.Printf("ALERT RESOLVED: %s - %s\n", alert.Level, alert.MetricName)
			}
		} else {
			// Keep unresolved alerts
			updatedAlerts = append(updatedAlerts, alertJSON)
		}
	}
	
	// Update active alerts list
	pipe := pm.redisClient.Pipeline()
	pipe.Del(ctx, "alerts:active")
	if len(updatedAlerts) > 0 {
		pipe.LPush(ctx, "alerts:active", updatedAlerts...)
	}
	pipe.Exec(ctx)
}

// startMetricsCollection starts background metrics collection
func (pm *PerformanceMonitor) startMetricsCollection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			
			// Collect various metrics
			pm.RecordDatabaseMetrics(ctx)
			pm.RecordRedisMetrics(ctx)
			pm.RecordApplicationMetrics(ctx)
		}
	}
}

// startAlertMonitoring starts background alert monitoring
func (pm *PerformanceMonitor) startAlertMonitoring() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			
			// Clean up old resolved alerts
			pm.cleanupOldAlerts(ctx)
		}
	}
}

// cleanupOldAlerts removes old resolved alerts
func (pm *PerformanceMonitor) cleanupOldAlerts(ctx context.Context) {
	alertsJSON, err := pm.redisClient.LRange(ctx, "alerts:active", 0, -1).Result()
	if err != nil {
		return
	}
	
	var activeAlerts []string
	cutoff := time.Now().Add(-1 * time.Hour)
	
	for _, alertJSON := range alertsJSON {
		var alert PerformanceAlert
		if err := json.Unmarshal([]byte(alertJSON), &alert); err != nil {
			continue
		}
		
		// Keep unresolved alerts or recently resolved alerts
		if !alert.Resolved || (alert.ResolvedAt != nil && alert.ResolvedAt.After(cutoff)) {
			activeAlerts = append(activeAlerts, alertJSON)
		}
	}
	
	// Update active alerts list
	pipe := pm.redisClient.Pipeline()
	pipe.Del(ctx, "alerts:active")
	if len(activeAlerts) > 0 {
		pipe.LPush(ctx, "alerts:active", activeAlerts...)
	}
	pipe.Exec(ctx)
}

// GetMetricsData gets historical metrics data
func (pm *PerformanceMonitor) GetMetricsData(ctx context.Context, metricName string, startTime, endTime time.Time) ([]MetricPoint, error) {
	// Get data from time series
	values, err := pm.redisClient.ZRangeByScore(ctx, fmt.Sprintf("metrics:series:%s", metricName), &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", startTime.Unix()),
		Max: fmt.Sprintf("%d", endTime.Unix()),
	}).Result()
	
	if err != nil {
		return nil, err
	}
	
	var metrics []MetricPoint
	for _, value := range values {
		var metricValue float64
		if _, err := fmt.Sscanf(value, "%f", &metricValue); err == nil {
			metrics = append(metrics, MetricPoint{
				Name:  metricName,
				Value: metricValue,
				Unit:  "unknown", // Would need to store unit info separately
			})
		}
	}
	
	return metrics, nil
}

// UpdateAlertThreshold updates alert threshold configuration
func (pm *PerformanceMonitor) UpdateAlertThreshold(metricName string, threshold AlertThreshold) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.alertThresholds[metricName] = threshold
}

// GetAlertThresholds gets all alert threshold configurations
func (pm *PerformanceMonitor) GetAlertThresholds() map[string]AlertThreshold {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	thresholds := make(map[string]AlertThreshold)
	for k, v := range pm.alertThresholds {
		thresholds[k] = v
	}
	
	return thresholds
}
package performance

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// DataLoader interface for batch loading
type DataLoader interface {
	Load(ctx context.Context, key string) (interface{}, error)
	LoadMany(ctx context.Context, keys []string) ([]interface{}, error)
	Prime(key string, value interface{})
	Clear(key string)
	ClearAll()
}

// BatchFunc defines the function signature for batch loading
type BatchFunc func(ctx context.Context, keys []string) ([]interface{}, error)

// CacheConfig defines caching configuration
type CacheConfig struct {
	TTL         time.Duration
	MaxSize     int
	EnableRedis bool
}

// dataLoader implements DataLoader with batching and caching
type dataLoader struct {
	batchFunc   BatchFunc
	cache       *sync.Map
	redisClient *redis.Client
	config      CacheConfig
	
	// Batching fields
	batch       []string
	batchMutex  sync.Mutex
	waiting     map[string][]chan *loadResult
	batchTimer  *time.Timer
	batchSize   int
	batchDelay  time.Duration
}

type loadResult struct {
	value interface{}
	error error
}

// DataLoaderConfig defines configuration for DataLoader
type DataLoaderConfig struct {
	BatchSize  int
	BatchDelay time.Duration
	Cache      CacheConfig
}

// NewDataLoader creates a new DataLoader instance
func NewDataLoader(batchFunc BatchFunc, redisClient *redis.Client, config DataLoaderConfig) DataLoader {
	dl := &dataLoader{
		batchFunc:   batchFunc,
		cache:       &sync.Map{},
		redisClient: redisClient,
		config:      config.Cache,
		waiting:     make(map[string][]chan *loadResult),
		batchSize:   config.BatchSize,
		batchDelay:  config.BatchDelay,
	}
	
	if dl.batchSize == 0 {
		dl.batchSize = 100
	}
	if dl.batchDelay == 0 {
		dl.batchDelay = 16 * time.Millisecond
	}
	
	return dl
}

// Load loads a single item by key
func (dl *dataLoader) Load(ctx context.Context, key string) (interface{}, error) {
	// Check in-memory cache first
	if cached, ok := dl.cache.Load(key); ok {
		return cached, nil
	}
	
	// Check Redis cache if enabled
	if dl.config.EnableRedis && dl.redisClient != nil {
		if cached, err := dl.loadFromRedis(ctx, key); err == nil {
			// Cache in memory for faster subsequent access
			dl.cache.Store(key, cached)
			return cached, nil
		}
	}
	
	// Create result channel
	resultChan := make(chan *loadResult, 1)
	
	dl.batchMutex.Lock()
	
	// Add to waiting list
	if waiters, exists := dl.waiting[key]; exists {
		dl.waiting[key] = append(waiters, resultChan)
	} else {
		dl.waiting[key] = []chan *loadResult{resultChan}
		dl.batch = append(dl.batch, key)
	}
	
	// Check if we should trigger batch immediately
	shouldExecute := len(dl.batch) >= dl.batchSize
	
	// Set up timer if this is the first item in batch
	if len(dl.batch) == 1 && dl.batchTimer == nil {
		dl.batchTimer = time.AfterFunc(dl.batchDelay, func() {
			dl.executeBatch(ctx)
		})
	}
	
	dl.batchMutex.Unlock()
	
	// Execute immediately if batch is full
	if shouldExecute {
		go dl.executeBatch(ctx)
	}
	
	// Wait for result
	result := <-resultChan
	return result.value, result.error
}

// LoadMany loads multiple items by keys
func (dl *dataLoader) LoadMany(ctx context.Context, keys []string) ([]interface{}, error) {
	results := make([]interface{}, len(keys))
	errors := make([]error, len(keys))
	
	// Use goroutines for concurrent loading
	var wg sync.WaitGroup
	wg.Add(len(keys))
	
	for i, key := range keys {
		go func(index int, k string) {
			defer wg.Done()
			value, err := dl.Load(ctx, k)
			results[index] = value
			errors[index] = err
		}(i, key)
	}
	
	wg.Wait()
	
	// Check if any errors occurred
	for _, err := range errors {
		if err != nil {
			return results, err
		}
	}
	
	return results, nil
}

// Prime adds a value to the cache
func (dl *dataLoader) Prime(key string, value interface{}) {
	dl.cache.Store(key, value)
	
	// Also cache in Redis if enabled
	if dl.config.EnableRedis && dl.redisClient != nil {
		go dl.storeInRedis(context.Background(), key, value)
	}
}

// Clear removes a key from cache
func (dl *dataLoader) Clear(key string) {
	dl.cache.Delete(key)
	
	// Also clear from Redis if enabled
	if dl.config.EnableRedis && dl.redisClient != nil {
		go dl.redisClient.Del(context.Background(), dl.redisKey(key))
	}
}

// ClearAll clears all cached data
func (dl *dataLoader) ClearAll() {
	dl.cache = &sync.Map{}
	
	// Clear Redis cache if enabled
	if dl.config.EnableRedis && dl.redisClient != nil {
		// This would clear all keys with our prefix
		// Implementation depends on your Redis setup
	}
}

// executeBatch executes the batch loading
func (dl *dataLoader) executeBatch(ctx context.Context) {
	dl.batchMutex.Lock()
	
	if len(dl.batch) == 0 {
		dl.batchMutex.Unlock()
		return
	}
	
	// Get current batch and reset
	currentBatch := make([]string, len(dl.batch))
	copy(currentBatch, dl.batch)
	currentWaiting := dl.waiting
	
	dl.batch = nil
	dl.waiting = make(map[string][]chan *loadResult)
	
	if dl.batchTimer != nil {
		dl.batchTimer.Stop()
		dl.batchTimer = nil
	}
	
	dl.batchMutex.Unlock()
	
	// Execute batch function
	results, err := dl.batchFunc(ctx, currentBatch)
	
	// Distribute results to waiting channels
	for i, key := range currentBatch {
		var result *loadResult
		
		if err != nil {
			result = &loadResult{error: err}
		} else if i < len(results) {
			result = &loadResult{value: results[i]}
			
			// Cache successful results
			dl.cache.Store(key, results[i])
			if dl.config.EnableRedis && dl.redisClient != nil {
				go dl.storeInRedis(ctx, key, results[i])
			}
		} else {
			result = &loadResult{error: fmt.Errorf("no result for key: %s", key)}
		}
		
		// Send result to all waiting channels for this key
		if waiters, exists := currentWaiting[key]; exists {
			for _, waiter := range waiters {
				waiter <- result
				close(waiter)
			}
		}
	}
}

// Redis helper methods
func (dl *dataLoader) redisKey(key string) string {
	return fmt.Sprintf("dataloader:%s", key)
}

func (dl *dataLoader) loadFromRedis(ctx context.Context, key string) (interface{}, error) {
	return dl.redisClient.Get(ctx, dl.redisKey(key)).Result()
}

func (dl *dataLoader) storeInRedis(ctx context.Context, key string, value interface{}) {
	dl.redisClient.Set(ctx, dl.redisKey(key), value, dl.config.TTL)
}

// Specific DataLoaders for different entity types

// UserDataLoader for loading users
type UserDataLoader struct {
	DataLoader
	db *gorm.DB
}

func NewUserDataLoader(db *gorm.DB, redisClient *redis.Client) *UserDataLoader {
	batchFunc := func(ctx context.Context, ids []string) ([]interface{}, error) {
		var users []interface{}
		
		// Convert string IDs to the appropriate type (assuming UUID strings)
		if err := db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
			return nil, err
		}
		
		// Create a map for O(1) lookup
		userMap := make(map[string]interface{})
		for _, user := range users {
			// Assuming users have an ID field that can be converted to string
			if u, ok := user.(map[string]interface{}); ok {
				if id, exists := u["id"]; exists {
					userMap[fmt.Sprintf("%v", id)] = user
				}
			}
		}
		
		// Return results in the same order as requested IDs
		results := make([]interface{}, len(ids))
		for i, id := range ids {
			if user, exists := userMap[id]; exists {
				results[i] = user
			} else {
				results[i] = nil
			}
		}
		
		return results, nil
	}
	
	config := DataLoaderConfig{
		BatchSize:  50,
		BatchDelay: 16 * time.Millisecond,
		Cache: CacheConfig{
			TTL:         15 * time.Minute,
			MaxSize:     1000,
			EnableRedis: true,
		},
	}
	
	return &UserDataLoader{
		DataLoader: NewDataLoader(batchFunc, redisClient, config),
		db:         db,
	}
}

// ActivityDataLoader for loading activities
type ActivityDataLoader struct {
	DataLoader
	db *gorm.DB
}

func NewActivityDataLoader(db *gorm.DB, redisClient *redis.Client) *ActivityDataLoader {
	batchFunc := func(ctx context.Context, ids []string) ([]interface{}, error) {
		var activities []interface{}
		
		if err := db.WithContext(ctx).Where("id IN ?", ids).Find(&activities).Error; err != nil {
			return nil, err
		}
		
		// Create a map for O(1) lookup
		activityMap := make(map[string]interface{})
		for _, activity := range activities {
			if a, ok := activity.(map[string]interface{}); ok {
				if id, exists := a["id"]; exists {
					activityMap[fmt.Sprintf("%v", id)] = activity
				}
			}
		}
		
		// Return results in the same order as requested IDs
		results := make([]interface{}, len(ids))
		for i, id := range ids {
			if activity, exists := activityMap[id]; exists {
				results[i] = activity
			} else {
				results[i] = nil
			}
		}
		
		return results, nil
	}
	
	config := DataLoaderConfig{
		BatchSize:  50,
		BatchDelay: 16 * time.Millisecond,
		Cache: CacheConfig{
			TTL:         10 * time.Minute,
			MaxSize:     500,
			EnableRedis: true,
		},
	}
	
	return &ActivityDataLoader{
		DataLoader: NewDataLoader(batchFunc, redisClient, config),
		db:         db,
	}
}

// FacultyDataLoader for loading faculties
type FacultyDataLoader struct {
	DataLoader
	db *gorm.DB
}

func NewFacultyDataLoader(db *gorm.DB, redisClient *redis.Client) *FacultyDataLoader {
	batchFunc := func(ctx context.Context, ids []string) ([]interface{}, error) {
		var faculties []interface{}
		
		if err := db.WithContext(ctx).Where("id IN ?", ids).Find(&faculties).Error; err != nil {
			return nil, err
		}
		
		// Create a map for O(1) lookup
		facultyMap := make(map[string]interface{})
		for _, faculty := range faculties {
			if f, ok := faculty.(map[string]interface{}); ok {
				if id, exists := f["id"]; exists {
					facultyMap[fmt.Sprintf("%v", id)] = faculty
				}
			}
		}
		
		// Return results in the same order as requested IDs
		results := make([]interface{}, len(ids))
		for i, id := range ids {
			if faculty, exists := facultyMap[id]; exists {
				results[i] = faculty
			} else {
				results[i] = nil
			}
		}
		
		return results, nil
	}
	
	config := DataLoaderConfig{
		BatchSize:  30,
		BatchDelay: 16 * time.Millisecond,
		Cache: CacheConfig{
			TTL:         30 * time.Minute, // Faculties change less frequently
			MaxSize:     100,
			EnableRedis: true,
		},
	}
	
	return &FacultyDataLoader{
		DataLoader: NewDataLoader(batchFunc, redisClient, config),
		db:         db,
	}
}

// ParticipationDataLoader for loading participations by user or activity
type ParticipationDataLoader struct {
	DataLoader
	db *gorm.DB
}

func NewParticipationDataLoader(db *gorm.DB, redisClient *redis.Client) *ParticipationDataLoader {
	batchFunc := func(ctx context.Context, keys []string) ([]interface{}, error) {
		// Keys can be in format "user:userID" or "activity:activityID"
		userIDs := []string{}
		activityIDs := []string{}
		keyMap := make(map[string]string) // maps actual key to type:id
		
		for _, key := range keys {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 {
				keyMap[key] = key
				if parts[0] == "user" {
					userIDs = append(userIDs, parts[1])
				} else if parts[0] == "activity" {
					activityIDs = append(activityIDs, parts[1])
				}
			}
		}
		
		var participations []interface{}
		
		// Query for user participations
		if len(userIDs) > 0 {
			var userParticipations []interface{}
			if err := db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&userParticipations).Error; err != nil {
				return nil, err
			}
			participations = append(participations, userParticipations...)
		}
		
		// Query for activity participations
		if len(activityIDs) > 0 {
			var activityParticipations []interface{}
			if err := db.WithContext(ctx).Where("activity_id IN ?", activityIDs).Find(&activityParticipations).Error; err != nil {
				return nil, err
			}
			participations = append(participations, activityParticipations...)
		}
		
		// Group participations by key
		participationGroups := make(map[string][]interface{})
		for _, participation := range participations {
			if p, ok := participation.(map[string]interface{}); ok {
				if userID, exists := p["user_id"]; exists {
					key := fmt.Sprintf("user:%v", userID)
					participationGroups[key] = append(participationGroups[key], participation)
				}
				if activityID, exists := p["activity_id"]; exists {
					key := fmt.Sprintf("activity:%v", activityID)
					participationGroups[key] = append(participationGroups[key], participation)
				}
			}
		}
		
		// Return results in the same order as requested keys
		results := make([]interface{}, len(keys))
		for i, key := range keys {
			if group, exists := participationGroups[key]; exists {
				results[i] = group
			} else {
				results[i] = []interface{}{}
			}
		}
		
		return results, nil
	}
	
	config := DataLoaderConfig{
		BatchSize:  40,
		BatchDelay: 16 * time.Millisecond,
		Cache: CacheConfig{
			TTL:         5 * time.Minute, // Participations change frequently
			MaxSize:     800,
			EnableRedis: true,
		},
	}
	
	return &ParticipationDataLoader{
		DataLoader: NewDataLoader(batchFunc, redisClient, config),
		db:         db,
	}
}

// DataLoaderContainer holds all DataLoaders
type DataLoaderContainer struct {
	User          *UserDataLoader
	Activity      *ActivityDataLoader
	Faculty       *FacultyDataLoader
	Participation *ParticipationDataLoader
}

// NewDataLoaderContainer creates a new container with all DataLoaders
func NewDataLoaderContainer(db *gorm.DB, redisClient *redis.Client) *DataLoaderContainer {
	return &DataLoaderContainer{
		User:          NewUserDataLoader(db, redisClient),
		Activity:      NewActivityDataLoader(db, redisClient),
		Faculty:       NewFacultyDataLoader(db, redisClient),
		Participation: NewParticipationDataLoader(db, redisClient),
	}
}

// Context key for DataLoaders
type contextKey string

const DataLoaderKey contextKey = "dataloaders"

// WithDataLoaders adds DataLoaders to context
func WithDataLoaders(ctx context.Context, loaders *DataLoaderContainer) context.Context {
	return context.WithValue(ctx, DataLoaderKey, loaders)
}

// GetDataLoaders retrieves DataLoaders from context
func GetDataLoaders(ctx context.Context) *DataLoaderContainer {
	if loaders, ok := ctx.Value(DataLoaderKey).(*DataLoaderContainer); ok {
		return loaders
	}
	return nil
}
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"your-project/graph"
	"your-project/graph/generated"
	"your-project/internal/config"
	"your-project/internal/middleware"
	"your-project/pkg/audit"
	"your-project/pkg/monitoring"
)

func main() {
	// Load configuration
	cfg := config.Load()
	
	// Initialize database with optimized connection pool
	db, err := initDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	
	// Initialize Redis
	redisClient := initRedis(cfg)
	
	// Generate or load master secret for cryptographic operations
	masterSecret := initMasterSecret(cfg)
	
	// Initialize integrated security system
	securityIntegration := middleware.NewSecurityIntegration(db, redisClient, masterSecret)
	
	// Initialize GraphQL schema and resolvers
	resolver := &graph.Resolver{
		DB:                  db,
		RedisClient:         redisClient,
		SecurityIntegration: securityIntegration,
	}
	
	schema := generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	})
	
	// Create GraphQL server with all security and performance features
	graphqlServer := securityIntegration.SetupGraphQLServer(schema)
	
	// Setup HTTP router
	router := setupRouter(securityIntegration, graphqlServer, cfg)
	
	// Create HTTP server with security configurations
	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", cfg.Port),
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}
	
	// Start server
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		log.Printf("GraphQL endpoint: http://localhost:%s/query", cfg.Port)
		log.Printf("GraphQL playground: http://localhost:%s/playground", cfg.Port)
		log.Printf("Health check: http://localhost:%s/health", cfg.Port)
		log.Printf("Metrics: http://localhost:%s/metrics", cfg.Port)
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()
	
	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Server shutting down...")
	
	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	
	log.Println("Server exited")
}

// initDatabase initializes database with optimized settings
func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	// Database configuration with connection pooling
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Bangkok",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt:              true,  // Cache prepared statements
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	
	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %v", err)
	}
	
	// Connection pool settings for high performance
	sqlDB.SetMaxOpenConns(50)           // Maximum open connections
	sqlDB.SetMaxIdleConns(10)           // Maximum idle connections
	sqlDB.SetConnMaxLifetime(time.Hour) // Connection maximum lifetime
	sqlDB.SetConnMaxIdleTime(30 * time.Minute) // Connection maximum idle time
	
	// Auto-migrate database schema
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}
	
	return db, nil
}

// initRedis initializes Redis client with optimized settings
func initRedis(cfg *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		
		// Connection pool settings
		PoolSize:     50,                    // Maximum connections
		MinIdleConns: 10,                    // Minimum idle connections
		MaxIdleConns: 20,                    // Maximum idle connections
		PoolTimeout:  4 * time.Second,       // Pool timeout
		IdleTimeout:  5 * time.Minute,       // Idle connection timeout
		
		// Connection settings
		DialTimeout:  10 * time.Second,      // Connection timeout
		ReadTimeout:  3 * time.Second,       // Read timeout
		WriteTimeout: 3 * time.Second,       // Write timeout
		
		// Retry settings
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	})
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	
	return client
}

// initMasterSecret initializes or loads master secret for cryptographic operations
func initMasterSecret(cfg *config.Config) []byte {
	// In production, load from secure key management system
	if cfg.Security.MasterSecret != "" {
		return []byte(cfg.Security.MasterSecret)
	}
	
	// Generate random master secret for development
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		log.Fatalf("Failed to generate master secret: %v", err)
	}
	
	log.Println("WARNING: Using randomly generated master secret. This should be persistent in production!")
	return secret
}

// setupRouter configures HTTP router with middleware
func setupRouter(securityIntegration *middleware.SecurityIntegration, graphqlServer *handler.Server, cfg *config.Config) http.Handler {
	router := mux.NewRouter()
	
	// Security middleware chain
	router.Use(securityIntegration.HTTPMiddleware)
	
	// CORS configuration
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: cfg.CORS.AllowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept", "Authorization", "Content-Type", "X-CSRF-Token",
			"X-Requested-With", "Apollo-Require-Preflight",
		},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	})
	
	// Apply CORS
	router.Use(corsHandler.Handler)
	
	// Health check endpoint
	router.HandleFunc("/health", securityIntegration.HealthCheckHandler()).Methods("GET")
	
	// Metrics endpoint (protected)
	router.HandleFunc("/metrics", securityIntegration.MetricsHandler()).Methods("GET")
	
	// GraphQL endpoints
	router.Handle("/query", graphqlServer).Methods("POST", "GET")
	
	// GraphQL playground (only in development)
	if cfg.Environment == "development" {
		router.Handle("/playground", playground.Handler("GraphQL playground", "/query")).Methods("GET")
	}
	
	// Static file serving with security headers
	if cfg.Environment == "production" {
		fs := http.FileServer(http.Dir("./static/"))
		router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", 
			addSecurityHeaders(fs))).Methods("GET")
	}
	
	// API versioning
	v1 := router.PathPrefix("/api/v1").Subrouter()
	
	// REST API endpoints (if needed)
	v1.HandleFunc("/qr/validate", handleQRValidation(securityIntegration)).Methods("POST")
	v1.HandleFunc("/export/audit", handleAuditExport(securityIntegration)).Methods("GET")
	
	return router
}

// autoMigrate performs database schema migrations
func autoMigrate(db *gorm.DB) error {
	// Migrate audit and monitoring tables
	return db.AutoMigrate(
		&audit.AuditEvent{},
		&audit.SecurityEvent{},
		&audit.PerformanceMetric{},
		// Add your application models here
	)
}

// REST API handlers

// handleQRValidation provides REST endpoint for QR validation
func handleQRValidation(si *middleware.SecurityIntegration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse request
		var req struct {
			QRData     string `json:"qr_data"`
			ScannerID  string `json:"scanner_id"`
			ActivityID string `json:"activity_id"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// Validate QR code
		result, err := si.ValidateQRCode(r.Context(), req.QRData, req.ScannerID, req.ActivityID)
		if err != nil {
			http.Error(w, "QR validation failed", http.StatusInternalServerError)
			return
		}
		
		// Return result
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// handleAuditExport provides audit log export functionality
func handleAuditExport(si *middleware.SecurityIntegration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This would implement audit log export
		// Check permissions, generate export, etc.
		
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Audit export functionality"}`))
	}
}

// addSecurityHeaders adds security headers to static file responses
func addSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year for static assets
		next.ServeHTTP(w, r)
	})
}

// Configuration example
func exampleConfig() *config.Config {
	return &config.Config{
		Port:        "8080",
		Environment: "development",
		
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "password",
			Name:     "tru_activity",
			SSLMode:  "disable",
		},
		
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		
		Security: config.SecurityConfig{
			JWTSecret:     "your-jwt-secret-here",
			MasterSecret:  "your-master-secret-here",
			BCryptCost:    12,
		},
		
		CORS: config.CORSConfig{
			AllowedOrigins: []string{
				"http://localhost:3000",
				"https://your-domain.com",
			},
		},
		
		Monitoring: config.MonitoringConfig{
			EnableMetrics:     true,
			EnableTracing:     true,
			MetricsPort:      "9090",
		},
	}
}

// Performance and monitoring setup
func setupMonitoring(cfg *config.Config) {
	// Set up Prometheus metrics if enabled
	if cfg.Monitoring.EnableMetrics {
		// Initialize Prometheus metrics
		log.Println("Prometheus metrics enabled on port", cfg.Monitoring.MetricsPort)
	}
	
	// Set up distributed tracing if enabled
	if cfg.Monitoring.EnableTracing {
		// Initialize Jaeger or other tracing
		log.Println("Distributed tracing enabled")
	}
}

// Cleanup function for graceful shutdown
func cleanup(db *gorm.DB, redisClient *redis.Client) {
	log.Println("Cleaning up resources...")
	
	// Close database connections
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}
	
	// Close Redis connections
	redisClient.Close()
	
	log.Println("Cleanup completed")
}
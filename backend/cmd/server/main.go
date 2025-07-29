package main

import (
	"log"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/kruakemaths/tru-activity/backend/graph"
	"github.com/kruakemaths/tru-activity/backend/graph/generated"
	"github.com/kruakemaths/tru-activity/backend/internal/config"
	"github.com/kruakemaths/tru-activity/backend/internal/database"
	"github.com/kruakemaths/tru-activity/backend/internal/middleware"
	"github.com/kruakemaths/tru-activity/backend/internal/models"
	"github.com/kruakemaths/tru-activity/backend/pkg/auth"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.NewConnection(cfg.DatabaseURL, cfg.Environment)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate database models
	err = db.Migrate(
		&models.User{},
		&models.Faculty{},
		&models.Department{},
		&models.Activity{},
		&models.Participation{},
		&models.Subscription{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize JWT service
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpireHours)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtService)
	gqlAuthMiddleware := middleware.NewGraphQLAuthMiddleware(jwtService, db.DB)

	// Initialize GraphQL resolver
	resolverConfig := &graph.Resolver{
		DB:         db,
		JWTService: jwtService,
	}

	// Create GraphQL server
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolverConfig}))
	srv.Use(gqlAuthMiddleware.ExtractAuth())

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Printf("Error: %v", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "Internal server error",
			})
		},
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"message": "TRU Activity API is running",
		})
	})

	// GraphQL Playground (development only)
	if cfg.Environment == "development" {
		app.Get("/", func(c *fiber.Ctx) error {
			c.Set("Content-Type", "text/html")
			playground.Handler("GraphQL playground", "/query").ServeHTTP(c.Response(), c.Request())
			return nil
		})
	}

	// GraphQL endpoint
	app.All("/query", func(c *fiber.Ctx) error {
		srv.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	// Protected routes group
	protected := app.Group("/api")
	protected.Use(authMiddleware.RequireAuth())

	// Admin routes group
	admin := protected.Group("/admin")
	admin.Use(authMiddleware.RequireRole("super_admin", "faculty_admin", "regular_admin"))

	log.Printf("Server starting on port %s", cfg.Port)
	log.Printf("GraphQL playground available at http://localhost:%s/", cfg.Port)
	log.Printf("GraphQL endpoint at http://localhost:%s/query", cfg.Port)

	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
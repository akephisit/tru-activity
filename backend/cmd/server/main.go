package main

import (
	"log"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/kruakemaths/tru-activity/backend/graph"
	"github.com/kruakemaths/tru-activity/backend/graph/generated"
	"github.com/kruakemaths/tru-activity/backend/internal/config"
	"github.com/kruakemaths/tru-activity/backend/internal/database"
	"github.com/kruakemaths/tru-activity/backend/internal/handlers"
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

	// Initialize SSE handler
	sseHandler := handlers.NewSSEHandler(db, jwtService)

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
		AllowOriginsFunc: func(origin string) bool { return true },
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Readiness check endpoint (checks database connectivity)
	app.Get("/ready", func(c *fiber.Ctx) error {
		sqlDB, err := db.DB.DB()
		if err != nil {
			return c.Status(503).JSON(fiber.Map{
				"status": "not ready",
				"error":  "database connection failed",
			})
		}

		if err := sqlDB.Ping(); err != nil {
			return c.Status(503).JSON(fiber.Map{
				"status": "not ready",
				"error":  "database ping failed",
			})
		}

		return c.JSON(fiber.Map{
			"status":  "ready",
			"message": "TRU Activity API is ready",
		})
	})

	// GraphQL Playground (development only)
	if cfg.Environment == "development" {
		app.Get("/", func(c *fiber.Ctx) error {
			c.Set("Content-Type", "text/html")
			// Simple HTML playground instead of using net/http handler
			playgroundHTML := `<!DOCTYPE html>
<html>
<head>
    <meta charset=utf-8/>
    <meta name="viewport" content="user-scalable=no, initial-scale=1.0, minimum-scale=1.0, maximum-scale=1.0, minimal-ui">
    <link href="https://fonts.googleapis.com/css?family=Open+Sans:300,400,600,700|Source+Code+Pro:400,700" rel="stylesheet">
    <title>GraphQL playground</title>
    <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
    <link rel="shortcut icon" href="//cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
    <script src="//cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
    <div id="root">
        <style>
            body { background-color: rgb(23, 42, 58); font-family: Open Sans, sans-serif; height: 90vh; }
            #root { height: 100%; width: 100%; display: flex; align-items: center; justify-content: center; }
            .loading { font-size: 32px; font-weight: 200; color: rgba(255, 255, 255, .6); margin-left: 20px; }
            img { width: 78px; height: 78px; }
            .title { font-weight: 400; }
        </style>
        <img src='//cdn.jsdelivr.net/npm/graphql-playground-react/build/logo.png' alt=''>
        <div class="loading"> Loading
            <span class="title">GraphQL Playground</span>
        </div>
    </div>
    <script>window.addEventListener('load', function (event) {
        GraphQLPlayground.init(document.getElementById('root'), {
            endpoint: '/query'
        })
    })</script>
</body>
</html>`
			return c.SendString(playgroundHTML)
		})
	}

	// GraphQL endpoint - disable for now due to compatibility issues
	app.All("/query", func(c *fiber.Ctx) error {
		// TODO: Fix Fiber to net/http adapter
		return c.JSON(fiber.Map{
			"error": "GraphQL endpoint temporarily disabled due to HTTP adapter issues",
		})
	})

	// SSE endpoints
	app.Get("/events", sseHandler.HandleSSEConnection)
	app.Post("/events/subscribe", sseHandler.HandleSubscribe)
	app.Post("/events/unsubscribe", sseHandler.HandleUnsubscribe)
	app.Post("/events/heartbeat", sseHandler.HandleHeartbeat)

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

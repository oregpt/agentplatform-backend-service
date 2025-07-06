package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oregpt/agentplatform-backend-service/internal/config"
	"github.com/oregpt/agentplatform-backend-service/internal/db"
	"github.com/oregpt/agentplatform-backend-service/internal/handlers"
	"github.com/oregpt/agentplatform-backend-service/internal/middleware"
	"github.com/oregpt/agentplatform-backend-service/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	spannerClient, err := db.NewSpannerClient(context.Background(), cfg.GCPProjectID, cfg.SpannerInstance, cfg.SpannerDatabase)
	if err != nil {
		log.Fatalf("Failed to initialize Spanner client: %v", err)
	}
	defer spannerClient.Close()

	// Ensure tables exist
	if err := spannerClient.EnsureTablesExist(context.Background()); err != nil {
		log.Fatalf("Failed to ensure tables exist: %v", err)
	}

	// Initialize storage
	storageClient, err := storage.NewGCSClient(context.Background(), cfg.GCPProjectID, cfg.GCSBucket)
	if err != nil {
		log.Fatalf("Failed to initialize GCS client: %v", err)
	}
	defer storageClient.Close()

	// Initialize handlers
	orgHandler := handlers.NewOrganizationHandler(spannerClient)
	agentHandler := handlers.NewAgentHandler(spannerClient)
	fileHandler := handlers.NewFileHandler(spannerClient, storageClient)
	userHandler := handlers.NewUserHandler(spannerClient)

	// Set up Gin router
	router := gin.Default()
	
	// Add CORS middleware
	router.Use(middleware.CORS())

	// Set up routes
	v1 := router.Group("/api/v1")
	{
		// Public routes
		v1.GET("/health", handlers.HealthCheck)

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.AuthRequired(cfg.AuthServiceURL))
		{
			// Organization routes
			organizations := protected.Group("/organizations")
			{
				organizations.GET("", orgHandler.List)
				organizations.POST("", orgHandler.Create)
				organizations.GET("/:id", orgHandler.Get)
				organizations.PUT("/:id", orgHandler.Update)
				organizations.DELETE("/:id", orgHandler.Delete)
			}

			// Agent routes
			agents := protected.Group("/agents")
			{
				agents.GET("", agentHandler.List)
				agents.POST("", agentHandler.Create)
				agents.GET("/:id", agentHandler.Get)
				agents.PUT("/:id", agentHandler.Update)
				agents.DELETE("/:id", agentHandler.Delete)
			}

			// File routes
			files := protected.Group("/files")
			{
				files.GET("/agent/:agent_id", fileHandler.List)
				files.GET("/organization", fileHandler.ListByOrganization)
				files.POST("/agent/:agent_id", fileHandler.Upload)
				files.GET("/:id", fileHandler.Get)
				files.DELETE("/:id", fileHandler.Delete)
			}

			// User routes
			users := protected.Group("/users")
			{
				users.GET("", userHandler.List)
				users.POST("", userHandler.Create)
				users.GET("/:id", userHandler.Get)
				users.PUT("/:id", userHandler.Update)
				users.DELETE("/:id", userHandler.Delete)
				users.POST("/assign", userHandler.AssignToAgent)
				
				// Fix for route conflict - use a different route structure
				userAgents := users.Group("/by-id/:user_id/agents")
				{
					userAgents.DELETE("/:agent_id", userHandler.RemoveFromAgent)
					userAgents.GET("", userHandler.ListAgents)
				}
			}
		}
	}

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting Backend Service on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

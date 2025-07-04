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

	// Initialize database connection
	database, err := db.NewSpannerClient(context.Background(), cfg.GCPProjectID, cfg.SpannerInstance, cfg.SpannerDatabase)
	if err != nil {
		log.Fatalf("Failed to initialize Spanner client: %v", err)
	}
	defer database.Close()

	// Initialize storage client
	storageClient, err := storage.NewGCSClient(context.Background(), cfg.GCPProjectID, cfg.GCSBucket)
	if err != nil {
		log.Fatalf("Failed to initialize GCS client: %v", err)
	}
	defer storageClient.Close()

	// Initialize handlers
	orgHandler := handlers.NewOrganizationHandler(database)
	agentHandler := handlers.NewAgentHandler(database)
	fileHandler := handlers.NewFileHandler(database, storageClient)
	userHandler := handlers.NewUserHandler(database)

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
			orgs := protected.Group("/organizations")
			{
				orgs.GET("", orgHandler.List)
				orgs.POST("", orgHandler.Create)
				orgs.GET("/:id", orgHandler.Get)
				orgs.PUT("/:id", orgHandler.Update)
				orgs.DELETE("/:id", orgHandler.Delete)
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
				users.DELETE("/:user_id/agents/:agent_id", userHandler.RemoveFromAgent)
				users.GET("/:user_id/agents", userHandler.ListAgents)
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

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

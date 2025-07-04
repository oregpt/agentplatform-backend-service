package config

import (
	"os"
	"errors"
)

// Config holds the application configuration
type Config struct {
	Port            string
	AuthServiceURL  string
	GCPProjectID    string
	SpannerInstance string
	SpannerDatabase string
	GCSBucket       string
}

// Load loads the configuration from environment variables
func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // Default port for Backend Service
	}

	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		authServiceURL = "http://localhost:8080" // Default Auth Service URL
	}

	gcpProjectID := os.Getenv("GCP_PROJECT_ID")
	if gcpProjectID == "" {
		gcpProjectID = "oregpt-agent-platform" // Default project ID
	}

	spannerInstance := os.Getenv("SPANNER_INSTANCE")
	if spannerInstance == "" {
		return nil, errors.New("SPANNER_INSTANCE environment variable is required")
	}

	spannerDatabase := os.Getenv("SPANNER_DATABASE")
	if spannerDatabase == "" {
		return nil, errors.New("SPANNER_DATABASE environment variable is required")
	}

	gcsBucket := os.Getenv("GCS_BUCKET")
	if gcsBucket == "" {
		return nil, errors.New("GCS_BUCKET environment variable is required")
	}

	return &Config{
		Port:            port,
		AuthServiceURL:  authServiceURL,
		GCPProjectID:    gcpProjectID,
		SpannerInstance: spannerInstance,
		SpannerDatabase: spannerDatabase,
		GCSBucket:       gcsBucket,
	}, nil
}

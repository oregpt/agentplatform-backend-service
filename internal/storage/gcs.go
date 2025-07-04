package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/oregpt/agentplatform-backend-service/internal/models"
)

// GCSClient wraps the Google Cloud Storage client
type GCSClient struct {
	Client     *storage.Client
	BucketName string
}

// NewGCSClient creates a new Google Cloud Storage client
func NewGCSClient(ctx context.Context, projectID, bucketName string) (*GCSClient, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %v", err)
	}

	return &GCSClient{
		Client:     client,
		BucketName: bucketName,
	}, nil
}

// Close closes the GCS client
func (g *GCSClient) Close() error {
	return g.Client.Close()
}

// UploadFile uploads a file to Google Cloud Storage
func (g *GCSClient) UploadFile(ctx context.Context, orgID, agentID, fileName string, contentType string, fileSize int64, content io.Reader) (*models.File, error) {
	// Generate a unique file ID
	fileID := uuid.New().String()

	// Create a path for the file
	path := fmt.Sprintf("%s/%s/%s", orgID, agentID, fileID)

	// Get a handle to the bucket
	bucket := g.Client.Bucket(g.BucketName)

	// Get a handle to the object
	obj := bucket.Object(path)

	// Create a writer
	w := obj.NewWriter(ctx)
	w.ContentType = contentType

	// Copy the file data to the GCS object
	if _, err := io.Copy(w, content); err != nil {
		return nil, fmt.Errorf("failed to copy file to GCS: %v", err)
	}

	// Close the writer
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %v", err)
	}

	// Create a file record
	file := &models.File{
		ID:            fileID,
		AgentID:       agentID,
		OrganizationID: orgID,
		Name:          fileName,
		Path:          path,
		ContentType:   contentType,
		SizeBytes:     fileSize,
		CreatedBy:     "system", // This should be replaced with the actual user ID
		CreatedAt:     time.Now(),
	}

	return file, nil
}

// GetFile gets a file from Google Cloud Storage
func (g *GCSClient) GetFile(ctx context.Context, path string) (io.Reader, error) {
	// Get a handle to the bucket
	bucket := g.Client.Bucket(g.BucketName)

	// Get a handle to the object
	obj := bucket.Object(path)

	// Create a reader
	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %v", err)
	}

	return r, nil
}

// DeleteFile deletes a file from Google Cloud Storage
func (g *GCSClient) DeleteFile(ctx context.Context, path string) error {
	// Get a handle to the bucket
	bucket := g.Client.Bucket(g.BucketName)

	// Get a handle to the object
	obj := bucket.Object(path)

	// Delete the object
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}

	return nil
}

// ListFiles lists all files in a directory
func (g *GCSClient) ListFiles(ctx context.Context, orgID, agentID string) ([]*storage.ObjectAttrs, error) {
	// Create a path for the directory
	path := fmt.Sprintf("%s/%s/", orgID, agentID)

	// Get a handle to the bucket
	bucket := g.Client.Bucket(g.BucketName)

	// Create a query
	query := &storage.Query{
		Prefix: path,
	}

	// Get an iterator
	it := bucket.Objects(ctx, query)

	// Iterate over the objects
	var objects []*storage.ObjectAttrs
	for {
		attrs, err := it.Next()
		if err == storage.ErrObjectNotExist {
			break
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %v", err)
		}

		objects = append(objects, attrs)
	}

	return objects, nil
}

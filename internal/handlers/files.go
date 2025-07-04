package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/oregpt/agentplatform-backend-service/internal/db"
	"github.com/oregpt/agentplatform-backend-service/internal/storage"
)

// FileHandler handles file-related requests
type FileHandler struct {
	DB      *db.SpannerClient
	Storage *storage.GCSClient
}

// NewFileHandler creates a new file handler
func NewFileHandler(db *db.SpannerClient, storage *storage.GCSClient) *FileHandler {
	return &FileHandler{
		DB:      db,
		Storage: storage,
	}
}

// Upload uploads a file
func (h *FileHandler) Upload(c *gin.Context) {
	// Get user ID and org ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	// Get agent ID from path
	agentID := c.Param("agent_id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	// Get agent from database to verify it belongs to the organization
	agent, err := h.DB.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get agent: %v", err)})
		return
	}

	if agent.OrganizationID != orgID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Agent does not belong to your organization"})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to get file: %v", err)})
		return
	}
	defer file.Close()

	// Check if file is a markdown file
	if header.Header.Get("Content-Type") != "text/markdown" && header.Filename[len(header.Filename)-3:] != ".md" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only markdown files are allowed"})
		return
	}

	// Upload file to GCS
	uploadedFile, err := h.Storage.UploadFile(
		c.Request.Context(),
		orgID.(string),
		agentID,
		header.Filename,
		header.Header.Get("Content-Type"),
		header.Size,
		file,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload file: %v", err)})
		return
	}

	// Set the user ID as the creator
	uploadedFile.CreatedBy = userID.(string)

	// Save file record to database
	if err := h.DB.CreateFile(c.Request.Context(), uploadedFile); err != nil {
		// Try to delete the file from GCS if database operation fails
		_ = h.Storage.DeleteFile(c.Request.Context(), uploadedFile.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save file record: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, uploadedFile)
}

// Get gets a file
func (h *FileHandler) Get(c *gin.Context) {
	// Get file ID from path
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID is required"})
		return
	}

	// Get file from database
	file, err := h.DB.GetFile(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get file: %v", err)})
		return
	}

	// Check if file belongs to the user's organization
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	if file.OrganizationID != orgID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "File does not belong to your organization"})
		return
	}

	// Get file content from GCS
	reader, err := h.Storage.GetFile(c.Request.Context(), file.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get file content: %v", err)})
		return
	}

	// Read the content
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read file content: %v", err)})
		return
	}

	// Set content type and file name
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.Name))
	c.Data(http.StatusOK, file.ContentType, content)
}

// List lists all files for an agent
func (h *FileHandler) List(c *gin.Context) {
	// Get agent ID from path
	agentID := c.Param("agent_id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	// Get agent from database to verify it belongs to the organization
	agent, err := h.DB.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get agent: %v", err)})
		return
	}

	// Check if agent belongs to the user's organization
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	if agent.OrganizationID != orgID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Agent does not belong to your organization"})
		return
	}

	// Get files from database
	files, err := h.DB.ListFiles(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to list files: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

// Delete deletes a file
func (h *FileHandler) Delete(c *gin.Context) {
	// Get file ID from path
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File ID is required"})
		return
	}

	// Get file from database
	file, err := h.DB.GetFile(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get file: %v", err)})
		return
	}

	// Check if file belongs to the user's organization
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	if file.OrganizationID != orgID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "File does not belong to your organization"})
		return
	}

	// Delete file from GCS
	if err := h.Storage.DeleteFile(c.Request.Context(), file.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete file from storage: %v", err)})
		return
	}

	// Delete file from database
	if err := h.DB.DeleteFile(c.Request.Context(), fileID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete file record: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted"})
}

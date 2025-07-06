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
	fmt.Println("[File Upload] Starting file upload handler")
	
	// Get user ID and org ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		fmt.Println("[File Upload] ERROR: User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	fmt.Printf("[File Upload] User ID from context: %s\n", userID)

	orgID, exists := c.Get("org_id")
	if !exists {
		fmt.Println("[File Upload] ERROR: Organization ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}
	fmt.Printf("[File Upload] Organization ID from context: %s\n", orgID)

	// Get agent ID from path
	agentID := c.Param("agent_id")
	if agentID == "" {
		fmt.Println("[File Upload] ERROR: Agent ID is missing from request path")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}
	fmt.Printf("[File Upload] Agent ID from path: %s\n", agentID)

	// Get agent from database to verify it belongs to the organization
	fmt.Printf("[File Upload] Fetching agent %s from database to verify organization...\n", agentID)
	agent, err := h.DB.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		fmt.Printf("[File Upload] ERROR: Failed to get agent %s: %v\n", agentID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get agent: %v", err)})
		return
	}
	fmt.Printf("[File Upload] Agent found: ID=%s, Organization=%s\n", agent.ID, agent.OrganizationID)

	// Verify agent belongs to the organization
	if agent.OrganizationID != orgID.(string) {
		fmt.Printf("[File Upload] ERROR: Agent organization mismatch. Agent org: %s, Request org: %s\n", 
			agent.OrganizationID, orgID.(string))
		c.JSON(http.StatusForbidden, gin.H{"error": "Agent does not belong to your organization"})
		return
	}
	fmt.Println("[File Upload] Agent organization verification successful")

	// Get file from form
	fmt.Println("[File Upload] Getting file from form data...")
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		fmt.Printf("[File Upload] ERROR: Failed to get file from form: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to get file: %v", err)})
		return
	}
	defer file.Close()
	fmt.Printf("[File Upload] File received: %s, Size: %d bytes, Content-Type: %s\n", 
		header.Filename, header.Size, header.Header.Get("Content-Type"))

	// Check if file is a markdown file
	fmt.Println("[File Upload] Validating file type...")
	if header.Header.Get("Content-Type") != "text/markdown" && header.Filename[len(header.Filename)-3:] != ".md" {
		fmt.Printf("[File Upload] ERROR: Invalid file type: %s\n", header.Header.Get("Content-Type"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only markdown files are allowed"})
		return
	}
	fmt.Println("[File Upload] File type validation successful")

	// Upload file to GCS
	fmt.Println("[File Upload] Starting upload to Google Cloud Storage...")
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
		fmt.Printf("[File Upload] ERROR: Failed to upload file to GCS: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload file: %v", err)})
		return
	}
	fmt.Printf("[File Upload] File successfully uploaded to GCS: %s\n", uploadedFile.Path)

	// Set the user ID as the creator
	fmt.Printf("[File Upload] Setting creator user ID: %s\n", userID.(string))
	uploadedFile.CreatedBy = userID.(string)

	// Save file record to database
	fmt.Println("[File Upload] Saving file record to database...")
	if err := h.DB.CreateFile(c.Request.Context(), uploadedFile); err != nil {
		fmt.Printf("[File Upload] ERROR: Failed to save file record to database: %v\n", err)
		// Try to delete the file from GCS if database operation fails
		fmt.Printf("[File Upload] Attempting to delete file from GCS: %s\n", uploadedFile.Path)
		_ = h.Storage.DeleteFile(c.Request.Context(), uploadedFile.Path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save file record: %v", err)})
		return
	}
	fmt.Println("[File Upload] File record successfully saved to database")

	fmt.Println("[File Upload] File upload process completed successfully")
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

// ListByOrganization lists all files for an organization
func (h *FileHandler) ListByOrganization(c *gin.Context) {
	// Check for organization_id query parameter first
	queryOrgID := c.Query("organization_id")
	
	// If no query parameter, get org ID from context
	orgID := queryOrgID
	if orgID == "" {
		contextOrgID, exists := c.Get("org_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
			return
		}
		orgID = contextOrgID.(string)
	}

	// Get files from database
	files, err := h.DB.ListFilesByOrganizationID(c.Request.Context(), orgID)
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

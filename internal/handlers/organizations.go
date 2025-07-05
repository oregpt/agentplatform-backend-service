package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oregpt/agentplatform-backend-service/internal/db"
	"github.com/oregpt/agentplatform-backend-service/internal/models"
)

// OrganizationHandler handles organization-related requests
type OrganizationHandler struct {
	DB *db.SpannerClient
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(db *db.SpannerClient) *OrganizationHandler {
	return &OrganizationHandler{
		DB: db,
	}
}

// Create creates a new organization
func (h *OrganizationHandler) Create(c *gin.Context) {
	// Get user ID from context
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Parse request body
	var req models.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create organization
	org := &models.Organization{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userID.(string),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save organization
	if err := h.DB.CreateOrganization(c.Request.Context(), org); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, org)
}

// Get gets an organization by ID
func (h *OrganizationHandler) Get(c *gin.Context) {
	// Get organization ID from path
	orgID := c.Param("id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID is required"})
		return
	}

	// Get organization from database
	org, err := h.DB.GetOrganization(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, org)
}

// List lists all organizations
func (h *OrganizationHandler) List(c *gin.Context) {
	// Check if this is an 'all organizations' request
	allOrgsAccess, _ := c.Get("all_orgs_access")

	// Get user ID from context
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var orgs []*models.Organization
	var err error

	if allOrgsAccess == true {
		// Get all organizations for this user
		orgs, err = h.DB.ListOrganizations(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// Get organization ID from context
		orgID, exists := c.Get("org_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
			return
		}

		// Get organization from database
		org, err := h.DB.GetOrganization(c.Request.Context(), orgID.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return single organization as a slice
		orgs = []*models.Organization{org}
	}

	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

// Update updates an organization
func (h *OrganizationHandler) Update(c *gin.Context) {
	// Get organization ID from path
	orgID := c.Param("id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID is required"})
		return
	}

	// Parse request body
	var req models.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get organization from database
	org, err := h.DB.GetOrganization(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update organization
	org.Name = req.Name
	org.Description = req.Description
	org.UpdatedAt = time.Now()

	// Save organization
	if err := h.DB.UpdateOrganization(c.Request.Context(), org); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, org)
}

// Delete deletes an organization
func (h *OrganizationHandler) Delete(c *gin.Context) {
	// Get organization ID from path
	orgID := c.Param("id")
	if orgID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID is required"})
		return
	}

	// Delete organization from database
	if err := h.DB.DeleteOrganization(c.Request.Context(), orgID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Organization deleted"})
}

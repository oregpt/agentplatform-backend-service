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
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Get user email from context
	userEmail, exists := c.Get("user_email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User email not found in context"})
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

	// Create user organization membership for the creator (as admin)
	now := time.Now()
	userOrg := &models.UserOrg{
		OrganizationID: org.ID,
		UserID:         userID.(string),
		Email:          userEmail.(string),
		DisplayName:    "", // We don't have display name in the context, can be updated later
		Role:           "admin", // Creator is always an admin
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Save user organization membership
	if err := h.DB.CreateUserOrg(c.Request.Context(), userOrg); err != nil {
		// Log the error but don't fail the request since the organization was created successfully
		// In a production system, you might want to roll back the organization creation or retry
		c.JSON(http.StatusCreated, gin.H{
			"organization": org,
			"warning":      "Organization created but failed to assign creator as admin: " + err.Error(),
		})
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
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	var orgs []*models.Organization
	var err error

	// Check if this is an 'all organizations' request
	allOrgsAccess, _ := c.Get("all_orgs_access")

	if allOrgsAccess == true {
		// Get all organizations for this user
		orgs, err = h.DB.ListOrganizationsByUserID(c.Request.Context(), userID.(string))
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

package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oregpt/agentplatform-backend-service/internal/db"
	"github.com/oregpt/agentplatform-backend-service/internal/models"
)

// UserHandler handles core user operations
type UserHandler struct {
	DB *db.SpannerClient
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(db *db.SpannerClient) *UserHandler {
	return &UserHandler{DB: db}
}

// Create creates a new core user
func (h *UserHandler) Create(c *gin.Context) {
	// Parse request body
	var req models.User
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set timestamps
	now := time.Now()
	req.CreatedAt = now
	req.UpdatedAt = now

	// Create user
	if err := h.DB.CreateUser(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, req)
}

// Get gets a user by ID
func (h *UserHandler) Get(c *gin.Context) {
	// Get user ID from path
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Get user from database
	user, err := h.DB.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetByEmail gets a user by email
func (h *UserHandler) GetByEmail(c *gin.Context) {
	// Get email from query
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	// Get user from database
	user, err := h.DB.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// List lists all users
func (h *UserHandler) List(c *gin.Context) {
	// Check for organization_id query parameter first
	queryOrgID := c.Query("organization_id")

	// If organization_id is provided, we need to filter users by organization membership
	if queryOrgID != "" {
		// Get user organization memberships from database
		userOrgs, err := h.DB.ListUserOrgs(c.Request.Context(), queryOrgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Return user organization memberships in the expected format
		c.JSON(http.StatusOK, gin.H{"users": userOrgs})
		return
	}

	// If no organization_id is provided, return all users (core user records)
	users, err := h.DB.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return users in the expected format
	c.JSON(http.StatusOK, gin.H{"users": users})
}

// Update updates a user
func (h *UserHandler) Update(c *gin.Context) {
	// Get user ID from path
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Parse request body
	var req models.User
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure ID matches path parameter
	req.ID = userID

	// Get existing user to ensure it exists
	_, err := h.DB.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update timestamp
	req.UpdatedAt = time.Now()

	// Update user
	if err := h.DB.UpdateUser(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, req)
}

// Delete deletes a user
func (h *UserHandler) Delete(c *gin.Context) {
	// Get user ID from path
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Delete user
	if err := h.DB.DeleteUser(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

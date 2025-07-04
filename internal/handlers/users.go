package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oregpt/agentplatform-backend-service/internal/db"
	"github.com/oregpt/agentplatform-backend-service/internal/models"
)

// UserHandler handles user-related requests
type UserHandler struct {
	DB *db.SpannerClient
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *db.SpannerClient) *UserHandler {
	return &UserHandler{
		DB: db,
	}
}

// Create creates a new user
func (h *UserHandler) Create(c *gin.Context) {
	// Get org ID from context
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	// Parse request body
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create user
	user := &models.User{
		ID:            req.Email, // Using email as the user ID for now
		OrganizationID: orgID.(string),
		Email:         req.Email,
		DisplayName:   req.DisplayName,
		Role:          req.Role,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Save user
	if err := h.DB.CreateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// Get gets a user by ID
func (h *UserHandler) Get(c *gin.Context) {
	// Get user ID from path
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Get org ID from context
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	// Get user from database
	user, err := h.DB.GetUser(c.Request.Context(), userID, orgID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// List lists all users for an organization
func (h *UserHandler) List(c *gin.Context) {
	// Get org ID from context
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	// Get users from database
	users, err := h.DB.ListUsers(c.Request.Context(), orgID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	// Get org ID from context
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	// Parse request body
	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from database
	user, err := h.DB.GetUser(c.Request.Context(), userID, orgID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update user
	user.DisplayName = req.DisplayName
	user.Role = req.Role
	user.UpdatedAt = time.Now()

	// Save user
	if err := h.DB.UpdateUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// Delete deletes a user
func (h *UserHandler) Delete(c *gin.Context) {
	// Get user ID from path
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Get org ID from context
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	// Delete user from database
	if err := h.DB.DeleteUser(c.Request.Context(), userID, orgID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// AssignToAgent assigns a user to an agent
func (h *UserHandler) AssignToAgent(c *gin.Context) {
	// Get org ID from context
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	// Parse request body
	var req models.AssignUserAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user exists
	_, err := h.DB.GetUser(c.Request.Context(), req.UserID, orgID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	// Verify agent exists and belongs to the organization
	agent, err := h.DB.GetAgent(c.Request.Context(), req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent not found"})
		return
	}

	if agent.OrganizationID != orgID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Agent does not belong to your organization"})
		return
	}

	// Assign user to agent
	if err := h.DB.AssignUserToAgent(c.Request.Context(), req.UserID, orgID.(string), req.AgentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User assigned to agent"})
}

// RemoveFromAgent removes a user from an agent
func (h *UserHandler) RemoveFromAgent(c *gin.Context) {
	// Get user ID and agent ID from path
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	agentID := c.Param("agent_id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	// Get org ID from context
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	// Remove user from agent
	if err := h.DB.RemoveUserFromAgent(c.Request.Context(), userID, orgID.(string), agentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User removed from agent"})
}

// ListAgents lists all agents for a user
func (h *UserHandler) ListAgents(c *gin.Context) {
	// Get user ID from path
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Get org ID from context
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	// Get agents from database
	agentIDs, err := h.DB.ListUserAgents(c.Request.Context(), userID, orgID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get agent details
	agents := make([]*models.Agent, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		agent, err := h.DB.GetAgent(c.Request.Context(), agentID)
		if err != nil {
			continue
		}
		agents = append(agents, agent)
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

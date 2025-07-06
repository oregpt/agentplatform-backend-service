package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/oregpt/agentplatform-backend-service/internal/db"
	"github.com/oregpt/agentplatform-backend-service/internal/models"
)

// UserOrgHandler handles user organization membership-related requests
type UserOrgHandler struct {
	DB *db.SpannerClient
}

// NewUserOrgHandler creates a new user organization handler
func NewUserOrgHandler(db *db.SpannerClient) *UserOrgHandler {
	return &UserOrgHandler{
		DB: db,
	}
}

// Create creates a new user organization membership
func (h *UserOrgHandler) Create(c *gin.Context) {
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

	// Create user organization membership
	userOrg := &models.UserOrg{
		UserID:        req.Email, // Using email as the user ID for now
		OrganizationID: orgID.(string),
		Email:         req.Email,
		DisplayName:   req.DisplayName,
		Role:          req.Role,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Save user organization membership
	if err := h.DB.CreateUserOrg(c.Request.Context(), userOrg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, userOrg)
}

// Get gets a user organization membership by ID
func (h *UserOrgHandler) Get(c *gin.Context) {
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

	// Get user organization membership from database
	userOrg, err := h.DB.GetUserOrg(c.Request.Context(), userID, orgID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, userOrg)
}

// List lists all user organization memberships for an organization
func (h *UserOrgHandler) List(c *gin.Context) {
	// Get user ID from context for access control
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Check for organization_id query parameter first
	queryOrgID := c.Query("organization_id")
	
	// Check if this is an 'all organizations' request
	allOrgsAccess, _ := c.Get("all_orgs_access")
	
	// Handle 'All' organizations case
	if queryOrgID == "All" || (queryOrgID == "" && allOrgsAccess == true) {
		// Get all organizations this user has access to
		userOrgs, err := h.DB.ListUserOrganizations(c.Request.Context(), userID.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		// For each organization, get all users
		var allUserOrgs []*models.UserOrg
		for _, userOrg := range userOrgs {
			// Get all users for this organization
			orgUsers, err := h.DB.ListUserOrgs(c.Request.Context(), userOrg.OrganizationID)
			if err != nil {
				continue // Skip if error
			}
			
			// Add to combined result
			allUserOrgs = append(allUserOrgs, orgUsers...)
		}
		
		c.JSON(http.StatusOK, gin.H{"users": allUserOrgs})
		return
	}
	
	// Handle specific organization case
	orgID := queryOrgID
	if orgID == "" {
		// If no query parameter, get org ID from context
		contextOrgID, exists := c.Get("org_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
			return
		}
		orgID = contextOrgID.(string)
	}

	// Verify user has access to this organization
	userOrgs, err := h.DB.ListUserOrganizations(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	hasAccess := false
	for _, userOrg := range userOrgs {
		if userOrg.OrganizationID == orgID {
			hasAccess = true
			break
		}
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "User does not have access to this organization"})
		return
	}

	// Get user organization memberships from database
	userOrgs, err = h.DB.ListUserOrgs(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": userOrgs})
}

// Update updates a user organization membership
func (h *UserOrgHandler) Update(c *gin.Context) {
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

	// Get user organization membership from database
	userOrg, err := h.DB.GetUserOrg(c.Request.Context(), userID, orgID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update user organization membership
	userOrg.DisplayName = req.DisplayName
	userOrg.Role = req.Role
	userOrg.UpdatedAt = time.Now()

	// Save user organization membership
	if err := h.DB.UpdateUserOrg(c.Request.Context(), userOrg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, userOrg)
}

// Delete deletes a user organization membership
func (h *UserOrgHandler) Delete(c *gin.Context) {
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

	// Delete user organization membership
	if err := h.DB.DeleteUserOrg(c.Request.Context(), userID, orgID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// AssignToAgent assigns a user to an agent
func (h *UserOrgHandler) AssignToAgent(c *gin.Context) {
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

	// Verify user organization membership exists
	_, err := h.DB.GetUserOrg(c.Request.Context(), req.UserID, orgID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found in this organization"})
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
func (h *UserOrgHandler) RemoveFromAgent(c *gin.Context) {
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

// ListUserAgents lists all agents for a user
func (h *UserOrgHandler) ListUserAgents(c *gin.Context) {
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

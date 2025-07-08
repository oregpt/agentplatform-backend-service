package handlers

import (
	"log"
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
	// Parse request body to get the user and organization data
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Use organization ID from the request payload if provided, otherwise fall back to context
	orgID := req.OrganizationID
	if orgID == "" {
		// Fall back to org_id from context if not in payload
		orgIDValue, exists := c.Get("org_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found in request or context"})
			return
		}
		
		// Safely convert orgID to string
		var ok bool
		orgID, ok = orgIDValue.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid organization ID format"})
			return
		}
	}

	// Set up safe logging
	logger := func(format string, args ...interface{}) {
		loggerValue := c.Request.Context().Value("logger")
		if loggerFunc, ok := loggerValue.(func(string, ...interface{})); ok {
			loggerFunc(format, args...)
		} else {
			log.Printf(format, args...)
		}
	}

	// Log the request for debugging
	logger("Creating user-org association with payload: %+v", req)
	logger("Organization ID from context: %s", orgID)

	// Generate a UUID for the user if not provided
	userID := req.UserID
	if userID == "" {
		// If no UserID is provided, we'll use the email as a unique identifier
		// This should be consistent with how user IDs are generated in the core_users.go handler
		userID = req.Email
	}

	// Verify that the organization exists before creating the user-org association
	org, err := h.DB.GetOrganization(c.Request.Context(), orgID)
	if err != nil {
		logger("Organization lookup failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization does not exist: " + err.Error()})
		return
	}
	logger("Found organization: %+v", org)

	// Verify that the user exists or create them if they don't
	user, err := h.DB.GetUser(c.Request.Context(), userID)
	if err != nil {
		logger("User not found, creating new user: %s", userID)
		user = &models.User{
			ID:          userID,
			Email:       req.Email,
			DisplayName: req.DisplayName,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata:    "{}",
		}

		if err := h.DB.CreateUser(c.Request.Context(), user); err != nil {
			logger("Failed to create user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
			return
		}
		logger("Created new user: %+v", user)
	} else {
		logger("Found existing user: %+v", user)
	}

	// Create user organization membership
	userOrg := &models.UserOrg{
		OrganizationID: orgID, // Ensure organization_id is correctly set as the first field
		UserID:         userID, // Ensure user_id is correctly set as the second field
		Email:          req.Email,
		DisplayName:    req.DisplayName,
		Role:           req.Role,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Log the exact values being used for debugging
	logger("UserOrg struct - OrganizationID: %s, UserID: %s", userOrg.OrganizationID, userOrg.UserID)
	logger("Attempting to create user-org association: %+v", userOrg)

	// Save user organization membership
	if err := h.DB.CreateUserOrg(c.Request.Context(), userOrg); err != nil {
		logger("Failed to create user-org association: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user-org association: " + err.Error()})
		return
	}

	logger("Successfully created user-org association")
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

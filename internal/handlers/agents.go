package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oregpt/agentplatform-backend-service/internal/db"
	"github.com/oregpt/agentplatform-backend-service/internal/models"
)

// AgentHandler handles agent-related requests
type AgentHandler struct {
	DB *db.SpannerClient
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(db *db.SpannerClient) *AgentHandler {
	return &AgentHandler{
		DB: db,
	}
}

// Create creates a new agent
func (h *AgentHandler) Create(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	
	// Get org ID from context (used as default if not specified in request)
	contextOrgID, _ := c.Get("org_id")

	// Parse request body
	var req models.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use organization ID from request, or from context if not specified
	var orgIDStr string
	
	// First check if organization ID is provided in the request
	if req.OrganizationID != "" && req.OrganizationID != "all" {
		orgIDStr = req.OrganizationID
		log.Printf("Using organization ID from request: %s", orgIDStr)
	} else if contextOrgID != nil && contextOrgID.(string) != "" && contextOrgID.(string) != "all" {
		// Fall back to context organization ID
		orgIDStr = contextOrgID.(string)
		log.Printf("Using organization ID from context: %s", orgIDStr)
	} else {
		// If still no valid organization ID, return an error
		log.Printf("Error: No valid organization ID provided in request or context")
		c.JSON(http.StatusBadRequest, gin.H{"error": "A valid organization ID is required to create an agent"})
		return
	}
	
	// At this point, orgIDStr should always have a valid value
	
	// Create agent
	agent := &models.Agent{
		ID:            uuid.New().String(),
		OrganizationID: orgIDStr,
		Name:          req.Name,
		Description:   req.Description,
		Instructions:  req.Instructions,
		AIProvider:    req.AIProvider,
		CreatedBy:     userID.(string),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Log the agent creation attempt
	log.Printf("Creating agent with ID: %s, Name: %s, OrganizationID: %s", agent.ID, agent.Name, agent.OrganizationID)

	// Save agent
	if err := h.DB.CreateAgent(c.Request.Context(), agent); err != nil {
		log.Printf("Error creating agent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create a UserAgent record to link the agent to the user and organization
	// At this point, we should always have a valid organization ID
	log.Printf("Creating UserAgent record for agent %s, user %s, organization %s", agent.ID, userID.(string), orgIDStr)
	
	userAgent := &models.UserAgent{
		OrganizationID: orgIDStr,
		UserID:         userID.(string),
		AgentID:        agent.ID,
		CreatedAt:      time.Now(),
	}

	// Save UserAgent record
	if err := h.DB.CreateUserAgent(c.Request.Context(), userAgent); err != nil {
		// Log the error but don't fail the request since the agent was created successfully
		log.Printf("Warning: Agent created but failed to assign creator: %v", err)
		c.JSON(http.StatusCreated, gin.H{
			"agent":   agent,
			"warning": "Agent created but failed to assign creator: " + err.Error(),
		})
		return
	}
	
	log.Printf("Successfully created UserAgent record for agent %s", agent.ID)

	c.JSON(http.StatusCreated, agent)
}

// Get gets an agent by ID
func (h *AgentHandler) Get(c *gin.Context) {
	// Get agent ID from path
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	// Get agent from database
	agent, err := h.DB.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	c.JSON(http.StatusOK, agent)
}

// List lists all agents for an organization
func (h *AgentHandler) List(c *gin.Context) {
	// Get user ID from context for filtering by user access
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}
	userIDStr := userID.(string)

	// Check for organization_id query parameter first
	queryOrgID := c.Query("organization_id")
	
	// Special case: if queryOrgID is "All", list agents from all organizations the user has access to
	if queryOrgID == "All" {
		// Get all organizations the user has access to
		userOrgs, err := h.DB.ListUserOrganizations(c.Request.Context(), userIDStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list user organizations: " + err.Error()})
			return
		}
		
		// Collect agents from all organizations the user has access to
		allAgents := []*models.Agent{}
		for _, userOrg := range userOrgs {
			// Get agent IDs for this user and organization from UserAgents table
			agentIDs, err := h.DB.ListUserAgents(c.Request.Context(), userIDStr, userOrg.OrganizationID)
			if err != nil {
				// Log error but continue with other organizations
				log.Printf("Error listing agent IDs for user %s in organization %s: %v", userIDStr, userOrg.OrganizationID, err)
				continue
			}
			
			// Get full agent details for each agent ID
			for _, agentID := range agentIDs {
				agent, err := h.DB.GetAgent(c.Request.Context(), agentID)
				if err != nil {
					log.Printf("Error getting agent %s: %v", agentID, err)
					continue
				}
				allAgents = append(allAgents, agent)
			}
		}
		
		c.JSON(http.StatusOK, gin.H{"agents": allAgents})
		return
	}
	
	// If no query parameter or not "All", get org ID from context or use provided ID
	orgID := queryOrgID
	if orgID == "" {
		contextOrgID, exists := c.Get("org_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
			return
		}
		orgID = contextOrgID.(string)
	}

	// Get agent IDs for this user and organization from UserAgents table
	agentIDs, err := h.DB.ListUserAgents(c.Request.Context(), userIDStr, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list user agents: " + err.Error()})
		return
	}
	
	// Get full agent details for each agent ID
	agents := []*models.Agent{}
	for _, agentID := range agentIDs {
		agent, err := h.DB.GetAgent(c.Request.Context(), agentID)
		if err != nil {
			log.Printf("Error getting agent %s: %v", agentID, err)
			continue
		}
		agents = append(agents, agent)
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

// Update updates an agent
func (h *AgentHandler) Update(c *gin.Context) {
	// Get agent ID from path
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	// Parse request body
	var req models.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get agent from database
	agent, err := h.DB.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	// Update agent
	agent.Name = req.Name
	agent.Description = req.Description
	agent.Instructions = req.Instructions
	agent.AIProvider = req.AIProvider
	agent.UpdatedAt = time.Now()

	// Save agent
	if err := h.DB.UpdateAgent(c.Request.Context(), agent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// Delete deletes an agent
func (h *AgentHandler) Delete(c *gin.Context) {
	// Get agent ID from path
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	// Get agent from database
	agent, err := h.DB.GetAgent(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	// Delete agent from database
	if err := h.DB.DeleteAgent(c.Request.Context(), agentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent deleted"})
}

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

	// Parse request body
	var req models.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Override organization_id from the request with the one from the context
	// This ensures the agent is created in the correct organization
	orgIDStr := orgID.(string)
	
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

	// Save agent
	if err := h.DB.CreateAgent(c.Request.Context(), agent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create UserAgent record to automatically assign the creator to the agent
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
	// Check for organization_id query parameter first
	queryOrgID := c.Query("organization_id")
	
	// Special case: if queryOrgID is "All", list agents from all organizations
	if queryOrgID == "All" {
		// Get all organizations
		orgs, err := h.DB.ListOrganizations(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		// Collect agents from all organizations
		allAgents := []*models.Agent{}
		for _, org := range orgs {
			agents, err := h.DB.ListAgents(c.Request.Context(), org.ID)
			if err != nil {
				// Log error but continue with other organizations
				log.Printf("Error listing agents for organization %s: %v", org.ID, err)
				continue
			}
			allAgents = append(allAgents, agents...)
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

	// Get agents from database for specific organization
	agents, err := h.DB.ListAgents(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

package handlers

import (
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

	// Create agent
	agent := &models.Agent{
		ID:            uuid.New().String(),
		OrganizationID: orgID.(string),
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
	// Get org ID from context
	orgID, exists := c.Get("org_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization ID not found in context"})
		return
	}

	// Get agents from database
	agents, err := h.DB.ListAgents(c.Request.Context(), orgID.(string))
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

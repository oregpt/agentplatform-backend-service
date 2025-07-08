package models

import (
	"time"
)

// Organization represents an organization in the system
type Organization struct {
	ID          string    `json:"id" spanner:"OrganizationID"`
	Name        string    `json:"name" spanner:"Name"`
	Description string    `json:"description,omitempty" spanner:"Description"`
	CreatedBy   string    `json:"created_by" spanner:"CreatedBy"`
	CreatedAt   time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt   time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// Agent represents an AI agent in the system
type Agent struct {
	ID             string    `json:"id" spanner:"AgentID"`
	OrganizationID string    `json:"organization_id" spanner:"OrganizationID"`
	Name           string    `json:"name" spanner:"Name"`
	Description    string    `json:"description,omitempty" spanner:"Description"`
	Instructions   string    `json:"instructions,omitempty" spanner:"Instructions"`
	AIProvider     string    `json:"ai_provider" spanner:"AIProvider"`
	CreatedBy      string    `json:"created_by" spanner:"CreatedBy"`
	CreatedAt      time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt      time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// File represents a file uploaded to an agent
type File struct {
	ID          string    `json:"id" spanner:"FileID"`
	AgentID     string    `json:"agent_id" spanner:"AgentID"`
	Name        string    `json:"name" spanner:"Name"`
	Path        string    `json:"path" spanner:"Path"`
	ContentType string    `json:"content_type" spanner:"ContentType"`
	SizeBytes   int64     `json:"size_bytes" spanner:"SizeBytes"`
	CreatedBy   string    `json:"created_by" spanner:"CreatedBy"`
	CreatedAt   time.Time `json:"created_at" spanner:"CreatedAt"`
}

// UserOrg represents a user's membership in an organization
type UserOrg struct {
	OrganizationID string    `json:"organization_id" spanner:"OrganizationID"`
	UserID         string    `json:"user_id" spanner:"UserID"`
	Email          string    `json:"email" spanner:"Email"`
	DisplayName    string    `json:"display_name,omitempty" spanner:"DisplayName"`
	Role           string    `json:"role" spanner:"Role"`
	CreatedAt      time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt      time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// User represents a user in the system (core user data)
type User struct {
	ID          string    `json:"id" spanner:"UserID"`
	Email       string    `json:"email" spanner:"Email"`
	DisplayName string    `json:"display_name,omitempty" spanner:"DisplayName"`
	Address     string    `json:"address,omitempty" spanner:"Address"`
	Phone       string    `json:"phone,omitempty" spanner:"Phone"`
	// Metadata field is omitted from operations to avoid JSON decoding issues
	CreatedAt   time.Time `json:"created_at" spanner:"CreatedAt"`
	UpdatedAt   time.Time `json:"updated_at" spanner:"UpdatedAt"`
}

// UserAgent represents a relationship between a user and an agent
type UserAgent struct {
	OrganizationID string    `json:"organization_id" spanner:"OrganizationID"`
	UserID         string    `json:"user_id" spanner:"UserID"`
	AgentID        string    `json:"agent_id" spanner:"AgentID"`
	CreatedAt      time.Time `json:"created_at" spanner:"CreatedAt"`
}

// UserAgentMapping represents a simplified user-agent relationship for API responses
type UserAgentMapping struct {
	UserID         string `json:"userId"`
	OrganizationID string `json:"organizationId"`
	AgentID        string `json:"agentId"`
}

// CreateOrganizationRequest represents a request to create an organization
type CreateOrganizationRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateOrganizationRequest represents a request to update an organization
type UpdateOrganizationRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// CreateAgentRequest represents a request to create an agent
type CreateAgentRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	Instructions   string `json:"instructions"`
	AIProvider     string `json:"ai_provider" binding:"required"`
	OrganizationID string `json:"organization_id"`
}

// UpdateAgentRequest represents a request to update an agent
type UpdateAgentRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	Instructions string `json:"instructions"`
	AIProvider   string `json:"ai_provider" binding:"required"`
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email" binding:"required,email"`
	DisplayName    string `json:"display_name"`
	Role           string `json:"role" binding:"required"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
	Role        string `json:"role" binding:"required"`
}

// AssignUserAgentRequest represents a request to assign a user to an agent
type AssignUserAgentRequest struct {
	UserID  string `json:"user_id" binding:"required"`
	AgentID string `json:"agent_id" binding:"required"`
}

package db

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	databaseadmin "cloud.google.com/go/spanner/admin/database/apiv1"
	databasepb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/oregpt/agentplatform-backend-service/internal/models"
	"google.golang.org/api/iterator"
)

// SpannerClient wraps the Spanner client
type SpannerClient struct {
	Client       *spanner.Client
	AdminClient  *databaseadmin.DatabaseAdminClient
	DatabaseName string
}

// NewSpannerClient creates a new Spanner client
func NewSpannerClient(ctx context.Context, projectID, instance, database string) (*SpannerClient, error) {
	// Create the database path
	databaseName := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		projectID,
		instance,
		database)

	// Create the Spanner client
	client, err := spanner.NewClient(ctx, databaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Spanner client: %v", err)
	}

	// Create the Spanner admin client
	adminClient, err := databaseadmin.NewDatabaseAdminClient(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create Spanner admin client: %v", err)
	}

	return &SpannerClient{
		Client:       client,
		AdminClient:  adminClient,
		DatabaseName: databaseName,
	}, nil
}

// Close closes the Spanner client
func (s *SpannerClient) Close() {
	if s.Client != nil {
		s.Client.Close()
	}
	if s.AdminClient != nil {
		s.AdminClient.Close()
	}
}

// EnsureTablesExist creates the required tables if they don't exist
func (s *SpannerClient) EnsureTablesExist(ctx context.Context) error {
	// Check if the tables already exist by getting the database DDL
	ddlResp, err := s.AdminClient.GetDatabaseDdl(ctx, &databasepb.GetDatabaseDdlRequest{
		Database: s.DatabaseName,
	})
	if err != nil {
		return fmt.Errorf("failed to get database DDL: %v", err)
	}

	// Check if our tables exist in the DDL
	existingTables := make(map[string]bool)
	for _, stmt := range ddlResp.Statements {
		// Simple check for table names in CREATE TABLE statements
		if strings.Contains(stmt, "CREATE TABLE") {
			parts := strings.Fields(stmt)
			if len(parts) >= 3 {
				tableName := strings.Trim(parts[2], "`()")
				existingTables[tableName] = true
			}
		}
	}

	// Define the required tables
	requiredTables := []string{
		"Organizations",
		"Agents",
		"Files",
		"UserOrgs",
		"UserAgents",
		"Users",
	}

	// Check if all required tables exist
	missingTables := []string{}
	for _, table := range requiredTables {
		if !existingTables[table] {
			missingTables = append(missingTables, table)
		}
	}

	// If all tables exist, return
	if len(missingTables) == 0 {
		return nil
	}

	// Create the missing tables
	statements := []string{}

	// Organizations table
	if !existingTables["Organizations"] {
		statements = append(statements, `
			CREATE TABLE Organizations (
				OrganizationID STRING(36) NOT NULL,
				Name STRING(255) NOT NULL,
				Description STRING(MAX),
				CreatedBy STRING(128) NOT NULL,
				CreatedAt TIMESTAMP NOT NULL,
				UpdatedAt TIMESTAMP NOT NULL,
			) PRIMARY KEY (OrganizationID)
		`)
	}

	// Agents table
	if !existingTables["Agents"] {
		statements = append(statements, `
			CREATE TABLE Agents (
				AgentID STRING(36) NOT NULL,
				OrganizationID STRING(36),
				Name STRING(255) NOT NULL,
				Description STRING(MAX),
				Instructions STRING(MAX),
				AIProvider STRING(50) NOT NULL,
				CreatedBy STRING(128) NOT NULL,
				CreatedAt TIMESTAMP NOT NULL,
				UpdatedAt TIMESTAMP NOT NULL,
			) PRIMARY KEY (AgentID)
		`)
	}

	// Files table
	if !existingTables["Files"] {
		statements = append(statements, `
			CREATE TABLE Files (
				AgentID STRING(36) NOT NULL,
				FileID STRING(36) NOT NULL,
				Name STRING(255) NOT NULL,
				Path STRING(MAX) NOT NULL,
				ContentType STRING(100) NOT NULL,
				SizeBytes INT64 NOT NULL,
				CreatedBy STRING(128) NOT NULL,
				CreatedAt TIMESTAMP NOT NULL,
			) PRIMARY KEY (AgentID, FileID),
			INTERLEAVE IN PARENT Agents ON DELETE CASCADE
		`)
	}

	// UserOrgs table
	if !existingTables["UserOrgs"] {
		statements = append(statements, `
			CREATE TABLE UserOrgs (
				OrganizationID STRING(36) NOT NULL,
				UserID STRING(128) NOT NULL,
				Email STRING(255) NOT NULL,
				DisplayName STRING(255),
				Role STRING(50) NOT NULL,
				CreatedAt TIMESTAMP NOT NULL,
				UpdatedAt TIMESTAMP NOT NULL,
			) PRIMARY KEY (OrganizationID, UserID),
			INTERLEAVE IN PARENT Organizations ON DELETE CASCADE
		`)
	}

	// UserAgents table
	if !existingTables["UserAgents"] {
		statements = append(statements, `
			CREATE TABLE UserAgents (
				OrganizationID STRING(36) NOT NULL,
				UserID STRING(128) NOT NULL,
				AgentID STRING(36) NOT NULL,
				CreatedAt TIMESTAMP NOT NULL,
			) PRIMARY KEY (OrganizationID, UserID, AgentID),
			INTERLEAVE IN PARENT UserOrgs ON DELETE CASCADE
		`)
	}

	// Users table
	if !existingTables["Users"] {
		statements = append(statements, `
			CREATE TABLE Users (
				UserID STRING(128) NOT NULL,
				Email STRING(255) NOT NULL,
				DisplayName STRING(255),
				Address STRING(MAX),
				Phone STRING(50),
				Metadata JSON,
				CreatedAt TIMESTAMP NOT NULL,
				UpdatedAt TIMESTAMP NOT NULL,
			) PRIMARY KEY (UserID)
		`)
	}

	// Execute the statements
	op, err := s.AdminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   s.DatabaseName,
		Statements: statements,
	})
	if err != nil {
		return fmt.Errorf("failed to update database DDL: %v", err)
	}

	// Wait for the operation to complete
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for operation: %v", err)
	}

	return nil
}

// CreateOrganization creates a new organization
func (s *SpannerClient) CreateOrganization(ctx context.Context, org *models.Organization) error {
	mutation := spanner.InsertOrUpdateMap("Organizations", map[string]interface{}{
		"OrganizationID": org.ID,
		"Name":           org.Name,
		"Description":    org.Description,
		"CreatedBy":      org.CreatedBy,
		"CreatedAt":      org.CreatedAt,
		"UpdatedAt":      org.UpdatedAt,
	})

	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// GetOrganization gets an organization by ID
func (s *SpannerClient) GetOrganization(ctx context.Context, orgID string) (*models.Organization, error) {
	row, err := s.Client.Single().ReadRow(ctx, "Organizations", spanner.Key{orgID}, []string{
		"OrganizationID", "Name", "Description", "CreatedBy", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		return nil, err
	}

	var org models.Organization
	if err := row.ToStruct(&org); err != nil {
		return nil, err
	}

	return &org, nil
}

// ListOrganizations lists all organizations
func (s *SpannerClient) ListOrganizations(ctx context.Context) ([]*models.Organization, error) {
	stmt := spanner.Statement{SQL: `SELECT OrganizationID, Name, Description, CreatedBy, CreatedAt, UpdatedAt FROM Organizations`}
	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var orgs []*models.Organization
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var org models.Organization
		if err := row.ToStruct(&org); err != nil {
			return nil, err
		}

		orgs = append(orgs, &org)
	}

	return orgs, nil
}

// ListOrganizationsByUserID lists all organizations that a user has access to
func (s *SpannerClient) ListOrganizationsByUserID(ctx context.Context, userID string) ([]*models.Organization, error) {
	// First, find all organizations the user belongs to by querying the UserOrgs table
	stmt := spanner.Statement{
		SQL: `SELECT DISTINCT OrganizationID FROM UserOrgs WHERE UserID = @userID`,
		Params: map[string]interface{}{
			"userID": userID,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	// Collect organization IDs the user has access to
	var orgIDs []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error querying user organizations: %v", err)
		}

		var orgID string
		if err := row.Column(0, &orgID); err != nil {
			return nil, fmt.Errorf("error reading organization ID: %v", err)
		}

		orgIDs = append(orgIDs, orgID)
	}

	// If no organizations found, return empty list
	if len(orgIDs) == 0 {
		return []*models.Organization{}, nil
	}

	// Now fetch the organization details for each organization ID
	var orgs []*models.Organization
	for _, orgID := range orgIDs {
		org, err := s.GetOrganization(ctx, orgID)
		if err != nil {
			// Skip organizations that can't be found
			continue
		}
		orgs = append(orgs, org)
	}

	return orgs, nil
}

// UpdateOrganization updates an organization
func (s *SpannerClient) UpdateOrganization(ctx context.Context, org *models.Organization) error {
	mutation := spanner.UpdateMap("Organizations", map[string]interface{}{
		"OrganizationID": org.ID,
		"Name":           org.Name,
		"Description":    org.Description,
		"UpdatedAt":      org.UpdatedAt,
	})

	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// DeleteOrganization deletes an organization
func (s *SpannerClient) DeleteOrganization(ctx context.Context, orgID string) error {
	mutation := spanner.Delete("Organizations", spanner.Key{orgID})
	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// CreateAgent creates a new agent
func (s *SpannerClient) CreateAgent(ctx context.Context, agent *models.Agent) error {
	// Create a map for the agent data
	agentData := map[string]interface{}{
		"AgentID":      agent.ID,
		"Name":         agent.Name,
		"Description":  agent.Description,
		"Instructions": agent.Instructions,
		"AIProvider":   agent.AIProvider,
		"CreatedBy":    agent.CreatedBy,
		"CreatedAt":    agent.CreatedAt,
		"UpdatedAt":    agent.UpdatedAt,
	}

	// Only include OrganizationID if it's not empty
	if agent.OrganizationID != "" {
		agentData["OrganizationID"] = agent.OrganizationID
	} else {
		// Set OrganizationID to NULL explicitly
		agentData["OrganizationID"] = nil
	}

	mutation := spanner.InsertOrUpdateMap("Agents", agentData)

	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// GetAgent gets an agent by ID
func (s *SpannerClient) GetAgent(ctx context.Context, agentID string) (*models.Agent, error) {
	row, err := s.Client.Single().ReadRow(ctx, "Agents", spanner.Key{agentID}, []string{
		"AgentID", "OrganizationID", "Name", "Description", "Instructions", "AIProvider",
		"CreatedBy", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		return nil, err
	}

	var agent models.Agent
	if err := row.ToStruct(&agent); err != nil {
		return nil, err
	}

	return &agent, nil
}

// ListAgents lists all agents for an organization
func (s *SpannerClient) ListAgents(ctx context.Context, orgID string) ([]*models.Agent, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AgentID, OrganizationID, Name, Description, Instructions, AIProvider, 
			  CreatedBy, CreatedAt, UpdatedAt FROM Agents WHERE OrganizationID = @orgID`,
		Params: map[string]interface{}{
			"orgID": orgID,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var agents []*models.Agent
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var agent models.Agent
		if err := row.ToStruct(&agent); err != nil {
			return nil, err
		}

		agents = append(agents, &agent)
	}

	return agents, nil
}

// UpdateAgent updates an agent
func (s *SpannerClient) UpdateAgent(ctx context.Context, agent *models.Agent) error {
	mutation := spanner.UpdateMap("Agents", map[string]interface{}{
		"AgentID":        agent.ID,
		"OrganizationID": agent.OrganizationID,
		"Name":           agent.Name,
		"Description":    agent.Description,
		"Instructions":   agent.Instructions,
		"AIProvider":     agent.AIProvider,
		"UpdatedAt":      agent.UpdatedAt,
	})

	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// CheckUserAgentAccess checks if a user has access to an agent through UserAgent mappings
func (s *SpannerClient) CheckUserAgentAccess(ctx context.Context, userID string, agentID string) (bool, error) {
	// Query to check if a UserAgent mapping exists for this user and agent
	query := spanner.Statement{
		SQL: `SELECT COUNT(*) FROM UserAgents WHERE UserID = @userID AND AgentID = @agentID`,
		Params: map[string]interface{}{
			"userID": userID,
			"agentID": agentID,
		},
	}

	iter := s.Client.Single().Query(ctx, query)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error checking user agent access: %v", err)
	}

	var count int64
	if err := row.Columns(&count); err != nil {
		return false, fmt.Errorf("error scanning count: %v", err)
	}

	return count > 0, nil
}

// DeleteAgent deletes an agent and all related UserAgent records
func (s *SpannerClient) DeleteAgent(ctx context.Context, agentID string) error {
	// First, find all UserAgent records for this agent
	var userAgentKeys []spanner.Key
	query := spanner.Statement{
		SQL: `SELECT UserID, OrganizationID FROM UserAgents WHERE AgentID = @agentID`,
		Params: map[string]interface{}{"agentID": agentID},
	}

	iter := s.Client.Single().Query(ctx, query)
	defer iter.Stop()

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading UserAgent records: %v", err)
		}

		var userID, orgID string
		if err := row.Columns(&userID, &orgID); err != nil {
			return fmt.Errorf("error scanning UserAgent record: %v", err)
		}

		// Create a key for each UserAgent record to delete
		userAgentKeys = append(userAgentKeys, spanner.Key{userID, orgID, agentID})
	}

	// Create mutations to delete all UserAgent records and the Agent
	mutations := make([]*spanner.Mutation, 0, len(userAgentKeys)+1)

	// Add mutations to delete all UserAgent records
	for _, key := range userAgentKeys {
		mutations = append(mutations, spanner.Delete("UserAgents", key))
	}

	// Add mutation to delete the Agent
	mutations = append(mutations, spanner.Delete("Agents", spanner.Key{agentID}))

	// Apply all mutations in a single transaction
	_, err := s.Client.Apply(ctx, mutations)
	if err != nil {
		return fmt.Errorf("error deleting agent and related records: %v", err)
	}

	return nil
}

// CreateFile creates a new file record
func (s *SpannerClient) CreateFile(ctx context.Context, file *models.File) error {
	// Create a map with the required fields
	fileMap := map[string]interface{}{
		"FileID":         file.ID,
		"AgentID":        file.AgentID,
		"Name":           file.Name,
		"Path":           file.Path,
		"ContentType":    file.ContentType,
		"SizeBytes":      file.SizeBytes,
		"CreatedBy":      file.CreatedBy,
		"CreatedAt":      file.CreatedAt,
	}
	
	mutation := spanner.InsertOrUpdateMap("Files", fileMap)

	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	
	return err
}

// GetFile gets a file by ID
func (s *SpannerClient) GetFile(ctx context.Context, fileID string) (*models.File, error) {
	// First, find the AgentID for this file
	stmt := spanner.Statement{
		SQL: `SELECT AgentID FROM Files WHERE FileID = @fileID LIMIT 1`,
		Params: map[string]interface{}{
			"fileID": fileID,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var agentID string
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("file not found: %s", fileID)
	}
	if err != nil {
		return nil, fmt.Errorf("error querying for file: %v", err)
	}
	if err := row.Columns(&agentID); err != nil {
		return nil, fmt.Errorf("error reading agent ID: %v", err)
	}

	// Now read the full file record with the composite key
	row, err = s.Client.Single().ReadRow(ctx, "Files", spanner.Key{agentID, fileID}, []string{
		"FileID", "AgentID", "Name", "Path", "ContentType",
		"SizeBytes", "CreatedBy", "CreatedAt",
	})
	if err != nil {
		return nil, fmt.Errorf("error reading file with composite key: %v", err)
	}

	var file models.File
	if err := row.ToStruct(&file); err != nil {
		return nil, fmt.Errorf("error converting row to struct: %v", err)
	}

	return &file, nil
}

// ListFiles lists all files for an agent
func (s *SpannerClient) ListFiles(ctx context.Context, agentID string) ([]*models.File, error) {
	// Try first with the new schema that includes OrganizationID
	stmt := spanner.Statement{
		SQL: `SELECT FileID, AgentID, Name, Path, ContentType, 
			  SizeBytes, CreatedBy, CreatedAt FROM Files WHERE AgentID = @agentID`,
		Params: map[string]interface{}{
			"agentID": agentID,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var files []*models.File
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var file models.File
		if err := row.Columns(&file.ID, &file.AgentID, &file.Name, &file.Path, 
			&file.ContentType, &file.SizeBytes, &file.CreatedBy, &file.CreatedAt); err != nil {
			return nil, err
		}

		files = append(files, &file)
	}

	return files, nil
}

// ListAgentsByOrganizationID lists all agents for an organization
func (s *SpannerClient) ListAgentsByOrganizationID(ctx context.Context, organizationID string) ([]*models.Agent, error) {
	stmt := spanner.Statement{
		SQL: `SELECT AgentID, OrganizationID, Name, Description, CreatedBy, CreatedAt, UpdatedAt 
			  FROM Agents WHERE OrganizationID = @organizationID`,
		Params: map[string]interface{}{
			"organizationID": organizationID,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var agents []*models.Agent
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var agent models.Agent
		if err := row.ToStruct(&agent); err != nil {
			return nil, err
		}

		agents = append(agents, &agent)
	}

	return agents, nil
}

// ListFilesByOrganizationID lists all files for an organization
// Note: This function may not work if the OrganizationID column doesn't exist in the Files table
// It's kept for backward compatibility but may need to be updated in the future
func (s *SpannerClient) ListFilesByOrganizationID(ctx context.Context, organizationID string) ([]*models.File, error) {
	// First try to get all agents for this organization
	agents, err := s.ListAgentsByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents for organization: %v", err)
	}
	
	// Then get files for each agent
	var allFiles []*models.File
	for _, agent := range agents {
		agentFiles, err := s.ListFiles(ctx, agent.ID)
		if err != nil {
			// Log error but continue with other agents
			fmt.Printf("Error listing files for agent %s: %v\n", agent.ID, err)
			continue
		}
		
		// Files no longer have OrganizationID field
		
		allFiles = append(allFiles, agentFiles...)
	}
	
	return allFiles, nil
}

// DeleteFile deletes a file record
func (s *SpannerClient) DeleteFile(ctx context.Context, fileID string) error {
	// First, find the AgentID for this file
	stmt := spanner.Statement{
		SQL: `SELECT AgentID FROM Files WHERE FileID = @fileID LIMIT 1`,
		Params: map[string]interface{}{
			"fileID": fileID,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var agentID string
	row, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("file not found: %s", fileID)
	}
	if err != nil {
		return fmt.Errorf("error querying for file: %v", err)
	}
	if err := row.Columns(&agentID); err != nil {
		return fmt.Errorf("error reading agent ID: %v", err)
	}

	// Now delete the file record with the composite key
	mutation := spanner.Delete("Files", spanner.Key{agentID, fileID})
	_, err = s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	if err != nil {
		return fmt.Errorf("error deleting file: %v", err)
	}
	return nil
}

// CreateUserOrg creates a new user organization membership
func (s *SpannerClient) CreateUserOrg(ctx context.Context, userOrg *models.UserOrg) error {
	// First, verify that the organization exists
	_, err := s.GetOrganization(ctx, userOrg.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to create user-org association: organization %s does not exist: %w", 
			userOrg.OrganizationID, err)
	}

	// Then, verify that the user exists
	_, err = s.GetUser(ctx, userOrg.UserID)
	if err != nil {
		return fmt.Errorf("failed to create user-org association: user %s does not exist: %w", 
			userOrg.UserID, err)
	}

	// Create the mutation
	mutation := spanner.InsertOrUpdateMap("UserOrgs", map[string]interface{}{
		"OrganizationID": userOrg.OrganizationID,
		"UserID":         userOrg.UserID,
		"Email":          userOrg.Email,
		"DisplayName":    userOrg.DisplayName,
		"Role":           userOrg.Role,
		"CreatedAt":      userOrg.CreatedAt,
		"UpdatedAt":      userOrg.UpdatedAt,
	})

	// Apply the mutation
	_, err = s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	if err != nil {
		return fmt.Errorf("failed to apply user-org mutation: %w", err)
	}

	return nil
}

// GetUserOrg gets a user organization membership by user ID and organization ID
func (s *SpannerClient) GetUserOrg(ctx context.Context, userID, orgID string) (*models.UserOrg, error) {
	row, err := s.Client.Single().ReadRow(ctx, "UserOrgs", spanner.Key{orgID, userID}, []string{
		"OrganizationID", "UserID", "Email", "DisplayName", "Role", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		return nil, err
	}

	var userOrg models.UserOrg
	if err := row.ToStruct(&userOrg); err != nil {
		return nil, err
	}

	return &userOrg, nil
}

// ListUserOrgs lists all user organization memberships for an organization
func (s *SpannerClient) ListUserOrgs(ctx context.Context, orgID string) ([]*models.UserOrg, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrganizationID, UserID, Email, DisplayName, Role, CreatedAt, UpdatedAt 
			  FROM UserOrgs WHERE OrganizationID = @orgID`,
		Params: map[string]interface{}{
			"orgID": orgID,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var userOrgs []*models.UserOrg
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var userOrg models.UserOrg
		if err := row.ToStruct(&userOrg); err != nil {
			return nil, err
		}

		userOrgs = append(userOrgs, &userOrg)
	}

	return userOrgs, nil
}

// ListUserOrganizations lists all organizations a user belongs to
func (s *SpannerClient) ListUserOrganizations(ctx context.Context, userID string) ([]*models.UserOrg, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrganizationID, UserID, Email, DisplayName, Role, CreatedAt, UpdatedAt 
			  FROM UserOrgs WHERE UserID = @userID`,
		Params: map[string]interface{}{
			"userID": userID,
		},
	}

	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var userOrgs []*models.UserOrg
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var userOrg models.UserOrg
		if err := row.ToStruct(&userOrg); err != nil {
			return nil, err
		}

		userOrgs = append(userOrgs, &userOrg)
	}

	return userOrgs, nil
}

// UpdateUserOrg updates a user organization membership
func (s *SpannerClient) UpdateUserOrg(ctx context.Context, userOrg *models.UserOrg) error {
	mutation := spanner.UpdateMap("UserOrgs", map[string]interface{}{
		"OrganizationID": userOrg.OrganizationID,
		"UserID":         userOrg.UserID,
		"DisplayName":    userOrg.DisplayName,
		"Role":           userOrg.Role,
		"UpdatedAt":      userOrg.UpdatedAt,
	})

	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// DeleteUserOrg deletes a user organization membership
func (s *SpannerClient) DeleteUserOrg(ctx context.Context, userID, orgID string) error {
	mutation := spanner.Delete("UserOrgs", spanner.Key{orgID, userID})
	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// AssignUserToAgent assigns a user to an agent
func (s *SpannerClient) AssignUserToAgent(ctx context.Context, userID, orgID, agentID string) error {
	mutation := spanner.InsertOrUpdateMap("UserAgents", map[string]interface{}{
		"OrganizationID": orgID,
		"UserID":         userID,
		"AgentID":        agentID,
		"CreatedAt":      spanner.CommitTimestamp,
	})

	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// CreateUserAgent creates a new user-agent relationship
func (s *SpannerClient) CreateUserAgent(ctx context.Context, userAgent *models.UserAgent) error {
	mutation := spanner.InsertOrUpdateMap("UserAgents", map[string]interface{}{
		"OrganizationID": userAgent.OrganizationID,
		"UserID":         userAgent.UserID,
		"AgentID":        userAgent.AgentID,
		"CreatedAt":      userAgent.CreatedAt,
	})

	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// RemoveUserFromAgent removes a user from an agent
func (s *SpannerClient) RemoveUserFromAgent(ctx context.Context, userID, orgID, agentID string) error {
	mutation := spanner.Delete("UserAgents", spanner.Key{orgID, userID, agentID})
	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// ListUserAgents lists all agents for a user in an organization
func (s *SpannerClient) ListUserAgents(ctx context.Context, userID string, organizationID string) ([]string, error) {
	query := spanner.Statement{
		SQL: `SELECT AgentID FROM UserAgents WHERE UserID = @userID AND OrganizationID = @organizationID`,
		Params: map[string]interface{}{
			"userID": userID,
			"organizationID": organizationID,
		},
	}

	iter := s.Client.Single().Query(ctx, query)
	defer iter.Stop()

	var agentIDs []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var agentID string
		if err := row.Columns(&agentID); err != nil {
			return nil, err
		}

		agentIDs = append(agentIDs, agentID)
	}

	return agentIDs, nil
}

// GetUsersByAgent gets all users assigned to a specific agent
func (s *SpannerClient) GetUsersByAgent(ctx context.Context, agentID string) ([]models.UserAgentMapping, error) {
	// Query to get all UserAgent records for this agent with user email information
	query := spanner.Statement{
		SQL: `
			SELECT 
				ua.UserID, 
				ua.OrganizationID, 
				u.Email, 
				u.DisplayName 
			FROM UserAgents ua 
			LEFT JOIN Users u ON ua.UserID = u.ID 
			WHERE ua.AgentID = @agentID
		`,
		Params: map[string]interface{}{
			"agentID": agentID,
		},
	}

	iter := s.Client.Single().Query(ctx, query)
	defer iter.Stop()

	var userAgentMappings []models.UserAgentMapping
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var userID, organizationID string
		var email, displayName spanner.NullString
		if err := row.Columns(&userID, &organizationID, &email, &displayName); err != nil {
			return nil, err
		}

		// Create mapping with email information
		userAgentMappings = append(userAgentMappings, models.UserAgentMapping{
			UserID:         userID,
			OrganizationID: organizationID,
			AgentID:        agentID,
			Email:          email.StringVal,
			DisplayName:    displayName.StringVal,
		})
	}

	return userAgentMappings, nil
}

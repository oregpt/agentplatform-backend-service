package db

import (
	"context"
	"log"
	"time"
	"errors"

	"cloud.google.com/go/spanner"
	"github.com/oregpt/agentplatform-backend-service/internal/models"
	"google.golang.org/api/iterator"
)

// CreateUser creates a new core user record
func (s *SpannerClient) CreateUser(ctx context.Context, user *models.User) error {
	// Set timestamps if not already set
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	// Log the user data for debugging
	log.Printf("Creating user with ID: %s, Email: %s", user.ID, user.Email)

	// Create mutation without the Metadata field
	mutation := spanner.InsertOrUpdateMap("Users", map[string]interface{}{
		"UserID":      user.ID,
		"Email":       user.Email,
		"DisplayName": user.DisplayName,
		"Address":     user.Address,
		"Phone":       user.Phone,
		"CreatedAt":   user.CreatedAt,
		"UpdatedAt":   user.UpdatedAt,
	})
	
	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	if err != nil {
		log.Printf("Error creating user: %v", err)
	}
	return err
}

// GetUser gets a user by ID
func (s *SpannerClient) GetUser(ctx context.Context, userID string) (*models.User, error) {
	// Read the user row without the Metadata field
	row, err := s.Client.Single().ReadRow(ctx, "Users", spanner.Key{userID}, []string{
		"UserID", "Email", "DisplayName", "Address", "Phone", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		log.Printf("Error reading user %s: %v", userID, err)
		return nil, err
	}

	// Extract user fields
	var userID2, email, displayName, address, phone string
	var createdAt, updatedAt time.Time

	if err := row.Columns(&userID2, &email, &displayName, &address, &phone, &createdAt, &updatedAt); err != nil {
		log.Printf("Error extracting user columns for %s: %v", userID, err)
		return nil, err
	}

	// Construct the user object without setting Metadata
	user := &models.User{
		ID:          userID2,
		Email:       email,
		DisplayName: displayName,
		Address:     address,
		Phone:       phone,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	return user, nil
}

// ListUsers lists all users
func (s *SpannerClient) ListUsers(ctx context.Context) ([]*models.User, error) {
	stmt := spanner.Statement{
		SQL: `SELECT UserID, Email, DisplayName, Address, Phone, CreatedAt, UpdatedAt 
			  FROM Users`,
	}
	
	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()
	
	var users []*models.User
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		
		// Extract user fields manually to avoid Metadata issues
		var userID, email, displayName, address, phone string
		var createdAt, updatedAt time.Time
		
		if err := row.Columns(&userID, &email, &displayName, &address, &phone, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		
		user := &models.User{
			ID:          userID,
			Email:       email,
			DisplayName: displayName,
			Address:     address,
			Phone:       phone,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}
		
		users = append(users, user)
	}
	
	return users, nil
}

// UpdateUser updates a user's core information
func (s *SpannerClient) UpdateUser(ctx context.Context, user *models.User) error {
	mutation := spanner.UpdateMap("Users", map[string]interface{}{
		"UserID":      user.ID,
		"Email":       user.Email,
		"DisplayName": user.DisplayName,
		"Address":     user.Address,
		"Phone":       user.Phone,
		"UpdatedAt":   user.UpdatedAt,
	})
	
	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// DeleteUser deletes a user
func (s *SpannerClient) DeleteUser(ctx context.Context, userID string) error {
	mutation := spanner.Delete("Users", spanner.Key{userID})
	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// GetUserByEmail gets a user by email
func (s *SpannerClient) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	stmt := spanner.Statement{
		SQL: `SELECT UserID, Email, DisplayName, Address, Phone, CreatedAt, UpdatedAt 
			  FROM Users WHERE Email = @email LIMIT 1`,
		Params: map[string]interface{}{
			"email": email,
		},
	}
	
	iter := s.Client.Single().Query(ctx, stmt)
	defer iter.Stop()
	
	row, err := iter.Next()
	if err == iterator.Done {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}
	
	// Extract user fields manually to avoid Metadata issues
	var userID, userEmail, displayName, address, phone string
	var createdAt, updatedAt time.Time
	
	if err := row.Columns(&userID, &userEmail, &displayName, &address, &phone, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	
	user := &models.User{
		ID:          userID,
		Email:       userEmail,
		DisplayName: displayName,
		Address:     address,
		Phone:       phone,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	
	return user, nil
}

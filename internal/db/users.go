package db

import (
	"context"
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

	mutation := spanner.InsertOrUpdateMap("Users", map[string]interface{}{
		"UserID":      user.ID,
		"Email":       user.Email,
		"DisplayName": user.DisplayName,
		"Address":     user.Address,
		"Phone":       user.Phone,
		"Metadata":    user.Metadata,
		"CreatedAt":   user.CreatedAt,
		"UpdatedAt":   user.UpdatedAt,
	})
	
	_, err := s.Client.Apply(ctx, []*spanner.Mutation{mutation})
	return err
}

// GetUser gets a user by ID
func (s *SpannerClient) GetUser(ctx context.Context, userID string) (*models.User, error) {
	row, err := s.Client.Single().ReadRow(ctx, "Users", spanner.Key{userID}, []string{
		"UserID", "Email", "DisplayName", "Address", "Phone", "Metadata", "CreatedAt", "UpdatedAt",
	})
	if err != nil {
		return nil, err
	}
	
	var user models.User
	if err := row.ToStruct(&user); err != nil {
		return nil, err
	}
	
	return &user, nil
}

// ListUsers lists all users
func (s *SpannerClient) ListUsers(ctx context.Context) ([]*models.User, error) {
	stmt := spanner.Statement{
		SQL: `SELECT UserID, Email, DisplayName, Address, Phone, Metadata, CreatedAt, UpdatedAt 
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
		
		var user models.User
		if err := row.ToStruct(&user); err != nil {
			return nil, err
		}
		
		users = append(users, &user)
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
		"Metadata":    user.Metadata,
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
		SQL: `SELECT UserID, Email, DisplayName, Address, Phone, Metadata, CreatedAt, UpdatedAt 
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
	
	var user models.User
	if err := row.ToStruct(&user); err != nil {
		return nil, err
	}
	
	return &user, nil
}

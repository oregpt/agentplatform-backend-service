package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TokenVerificationResponse represents the response from the auth service
type TokenVerificationResponse struct {
	UID    string                 `json:"uid"`
	Email  string                 `json:"email"`
	Claims map[string]interface{} `json:"claims"`
}

// AuthRequired middleware to validate tokens via the Auth Service
func AuthRequired(authServiceURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Check if the header has the Bearer prefix
		idToken := strings.TrimPrefix(authHeader, "Bearer ")
		if idToken == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be in the format 'Bearer {token}'"})
			c.Abort()
			return
		}

		// Create a request to the auth service
		client := &http.Client{}
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/auth/verify", authServiceURL), strings.NewReader(fmt.Sprintf(`{"token": "%s"}`, idToken)))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create request: %v", err)})
			c.Abort()
			return
		}

		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to verify token: %v", err)})
			c.Abort()
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Parse the response
		var tokenResp TokenVerificationResponse
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to parse token verification response: %v", err)})
			c.Abort()
			return
		}

		// Extract organization ID from claims
		orgID, ok := tokenResp.Claims["org_id"].(string)
		if !ok {
			// If org_id is not in claims, set it to empty string (all orgs access)
			orgID = ""
			ok = true
		}

		// Check if this is an 'all organizations' request (empty org_id)
		// Empty org_id means access to all organizations the user belongs to
		allOrgsAccess := orgID == ""
		c.Set("all_orgs_access", allOrgsAccess)

		// Set user and organization info in context
		c.Set("user_id", tokenResp.UID)
		c.Set("org_id", orgID)
		c.Set("user_email", tokenResp.Email)
		c.Set("user_roles", tokenResp.Claims["roles"])
		c.Set("user_permissions", tokenResp.Claims["permissions"])

		c.Next()
	}
}

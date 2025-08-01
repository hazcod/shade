package auth

import (
	"github.com/hazcod/shade/pkg/model"
	"net/http"
)

// Provider defines the interface for authentication providers
type Provider interface {
	// Initialize sets up the provider with necessary configuration
	Initialize(logger interface{}, config map[string]interface{}) error

	// Authenticate verifies credentials and returns user info or error
	Authenticate(username, password string) (*model.User, error)

	// HandleLogin processes login requests
	HandleLogin() http.HandlerFunc

	// HandleLogout processes logout requests
	HandleLogout() http.HandlerFunc

	// RenderLoginPage renders the login page
	RenderLoginPage() http.HandlerFunc

	// Middleware provides authentication check for protected routes
	Middleware(next http.Handler) http.Handler

	// HandleCallback is optional for OIDC-like schemes
	HandleCallback() http.HandlerFunc
}

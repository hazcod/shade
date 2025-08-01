package oidc

import (
	"context"
	"errors"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/hazcod/shade/pkg/auth/session"
	"github.com/hazcod/shade/pkg/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"net/http"
	"time"
)

// Config represents OIDC provider configuration
type Config struct {
	ProviderURL     string
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	Scopes          []string
	SessionDuration time.Duration
}

// Provider implements the auth.Provider interface for OIDC authentication
type Provider struct {
	logger         *logrus.Logger
	config         *Config
	provider       *oidc.Provider
	oauth2Config   oauth2.Config
	verifier       *oidc.IDTokenVerifier
	authStateCache map[string]time.Time
}

// NewProvider creates a new OIDC authentication provider
func NewProvider(logger *logrus.Logger) *Provider {
	return &Provider{
		logger:         logger,
		authStateCache: make(map[string]time.Time),
	}
}

// Initialize sets up the OIDC authentication provider
func (p *Provider) Initialize(logger interface{}, config map[string]interface{}) error {
	// Convert the generic logger to a logrus logger
	logrusLogger, ok := logger.(*logrus.Logger)
	if !ok {
		return errors.New("logger must be a *logrus.Logger")
	}
	p.logger = logrusLogger

	// Extract configuration
	providerURL, ok := config["provider_url"].(string)
	if !ok || providerURL == "" {
		return errors.New("provider_url must be provided")
	}

	clientID, ok := config["client_id"].(string)
	if !ok || clientID == "" {
		return errors.New("client_id must be provided")
	}

	clientSecret, ok := config["client_secret"].(string)
	if !ok || clientSecret == "" {
		return errors.New("client_secret must be provided")
	}

	redirectURL, ok := config["redirect_url"].(string)
	if !ok || redirectURL == "" {
		return errors.New("redirect_url must be provided")
	}

	// Configure OIDC provider
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, providerURL)
	if err != nil {
		return err
	}

	// Set up the config structure
	p.config = &Config{
		ProviderURL:  providerURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	// Extract custom scopes if provided
	if scopes, ok := config["scopes"].([]interface{}); ok {
		customScopes := []string{oidc.ScopeOpenID} // OpenID scope is required
		for _, s := range scopes {
			if scope, ok := s.(string); ok {
				customScopes = append(customScopes, scope)
			}
		}
		p.config.Scopes = customScopes
	}

	// Configure OAuth2
	p.oauth2Config = oauth2.Config{
		ClientID:     p.config.ClientID,
		ClientSecret: p.config.ClientSecret,
		RedirectURL:  p.config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       p.config.Scopes,
	}

	// Configure token verifier
	p.provider = provider
	p.verifier = provider.Verifier(&oidc.Config{ClientID: p.config.ClientID})

	return nil
}

// Authenticate verifies the username and password (not used in OIDC)
func (p *Provider) Authenticate(username, password string) (*model.User, error) {
	return nil, errors.New("direct authentication not supported with OIDC")
}

// HandleLogin redirects to the OIDC provider
func (p *Provider) HandleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Generate a random state for CSRF protection
		state := generateRandomState()

		// Store the state with a timestamp (for expiration)
		p.authStateCache[state] = time.Now().Add(15 * time.Minute)

		// Clean up expired states
		p.cleanupExpiredStates()

		// Redirect to the OIDC provider
		url := p.oauth2Config.AuthCodeURL(state)
		http.Redirect(w, r, url, http.StatusFound)
	}
}

// HandleCallback processes the OIDC callback
func (p *Provider) HandleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the state and code from the callback
		state := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")

		// Verify the state
		expiry, ok := p.authStateCache[state]
		if !ok || time.Now().After(expiry) {
			p.logger.WithField("state", state).Error("Invalid or expired state")
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}

		// Delete the used state
		delete(p.authStateCache, state)

		// Exchange the code for a token
		ctx := context.Background()
		oauth2Token, err := p.oauth2Config.Exchange(ctx, code)
		if err != nil {
			p.logger.WithError(err).Error("Failed to exchange code for token")
			http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
			return
		}

		// Extract the ID token
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			p.logger.Error("No ID token found in OAuth2 token")
			http.Error(w, "No ID token found", http.StatusInternalServerError)
			return
		}

		// Verify the ID token
		idToken, err := p.verifier.Verify(ctx, rawIDToken)
		if err != nil {
			p.logger.WithError(err).Error("Failed to verify ID token")
			http.Error(w, "Failed to verify ID token", http.StatusInternalServerError)
			return
		}

		// Extract claims from the ID token
		var claims struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := idToken.Claims(&claims); err != nil {
			p.logger.WithError(err).Error("Failed to parse ID token claims")
			http.Error(w, "Failed to parse ID token claims", http.StatusInternalServerError)
			return
		}

		// Create a user object
		user := &model.User{
			Email: claims.Email,
			Roles: []string{"user"}, // Default role
		}

		// Store the user in the session
		err = session.SetUser(w, r, user)
		if err != nil {
			p.logger.WithError(err).Error("Failed to create session")
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		// Redirect to the dashboard
		http.Redirect(w, r, "/dashboard/", http.StatusSeeOther)
	}
}

// RenderLoginPage renders a login page with OIDC button
func (p *Provider) RenderLoginPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For OIDC, we simply show a button that redirects to the HandleLogin endpoint
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(oidcLoginPage))
	}
}

// HandleLogout processes logout requests
func (p *Provider) HandleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Clear the session
		err := session.ClearSession(w, r)
		if err != nil {
			p.logger.WithError(err).Error("Failed to clear session")
			http.Error(w, "Failed to logout", http.StatusInternalServerError)
			return
		}

		// Redirect to the login page
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
	}
}

// Middleware provides authentication check for protected routes
func (p *Provider) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the user is authenticated
		user, err := session.GetUser(r)
		if err != nil {
			p.logger.WithError(err).Error("Error retrieving session")
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		if user == nil {
			// User is not authenticated, redirect to login page
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		// User is authenticated, proceed to the next handler
		next.ServeHTTP(w, r)
	})
}

// Helper methods

// generateRandomState creates a random state string for CSRF protection
func generateRandomState() string {
	// Simplified implementation - in production use a proper random generator
	return "state-" + time.Now().Format("20060102150405")
}

// cleanupExpiredStates removes expired state entries
func (p *Provider) cleanupExpiredStates() {
	now := time.Now()
	for state, expiry := range p.authStateCache {
		if now.After(expiry) {
			delete(p.authStateCache, state)
		}
	}
}

// Login page template with OIDC button
const oidcLoginPage = `
<!DOCTYPE html>
<html lang="en" data-bs-theme="auto">
<head>
    <meta charset="utf-8">
    <title>Login - Shade</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.7/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-LN+7fdVzj6u52u30Kp6M/trliBMCMKTyK833zpbD+pXdCLuTusPj697FH4R/5mcr" crossorigin="anonymous">
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.7/dist/js/bootstrap.bundle.min.js" integrity="sha384-ndDqU0Gzau9qJ1lfW4pNLlhNTkCfHzAVBReH9diLvGRem5+R9g2FzA8ZGN954O5Q" crossorigin="anonymous"></script>
    <style>
        body {
            background-color: #f8f9fa;
            height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .login-container {
            max-width: 400px;
            width: 100%;
            padding: 15px;
        }
        .card {
            border-radius: 10px;
            box-shadow: 0 4px 10px rgba(0, 0, 0, 0.1);
        }
        .card-title {
            color: #212529;
            margin-bottom: 20px;
        }
        .btn-primary {
            background-color: #0d6efd;
            border-color: #0d6efd;
            padding: 10px 0;
            font-weight: 500;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="card">
            <div class="card-body p-4 p-md-5">
                <h3 class="card-title text-center">Shade Login</h3>
                <a href="/auth/login" class="btn btn-primary w-100 mt-3">Sign In with SSO</a>
            </div>
        </div>
    </div>
</body>
</html>
`

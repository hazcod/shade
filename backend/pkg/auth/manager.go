package auth

import (
	"errors"
	"fmt"
	"github.com/hazcod/shade/pkg/auth/local"
	"github.com/hazcod/shade/pkg/auth/oidc"
	"github.com/hazcod/shade/pkg/auth/session"
	"github.com/sirupsen/logrus"
)

// GetProvider returns an authentication provider based on the specified type
func GetProvider(logger *logrus.Logger, providerType string, devMode bool, properties map[string]interface{}) (Provider, error) {
	sessionSecret, ok := properties["secret"].(string)
	if !ok || sessionSecret == "" {
		return nil, errors.New("property 'secret' is required")
	}

	// Initialize the session store
	session.Initialize(sessionSecret, devMode)

	// Create the appropriate provider based on the type
	var provider Provider
	switch providerType {
	case "local":
		provider = local.NewProvider(logger)
	case "oidc":
		provider = oidc.NewProvider(logger)
	default:
		return nil, fmt.Errorf("unsupported auth provider type: %s", providerType)
	}

	// Initialize the provider with the configuration
	if err := provider.Initialize(logger, properties); err != nil {
		return nil, fmt.Errorf("failed to initialize %s provider: %w", providerType, err)
	}

	return provider, nil
}

// GeneratePasswordHash generates a bcrypt hash from a plaintext password
func GeneratePasswordHash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	// Import golang.org/x/crypto/bcrypt and use its functions to hash the password
	// For example: hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	// Return string(hash), err

	// This is a placeholder - you would implement the actual bcrypt hashing here
	return "hashed_password", nil
}

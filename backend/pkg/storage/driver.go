package storage

import (
	"github.com/hazcod/shade/pkg/events"
	"github.com/hazcod/shade/pkg/models"
	"github.com/sirupsen/logrus"
)

type Driver interface {
	Init(logger *logrus.Logger, settings map[string]string) error
	AddLoginEvent(data events.LoginEvent) error
	GetAllDomains() ([]string, error)
	GetDomainsForUser(username string) ([]string, error)
	GetDuplicatePasswordsForUser(username string) ([][]string, error)
	IsDuplicatePassword(username, passwordHash string) ([]string, error)
	GetDuplicatePasswords() (map[string]map[string]string, error)
	IsValidToken(token string) (bool, error)
	GetCompromisedPasswords() (map[string]string, error)
	GetEnrolledUsers() ([]models.EnrolledUser, error)
	GetDashboardStats() (models.DashboardStats, error)
	GetUsersWithoutMFA() ([]string, error)
	// HIBP-related methods
	StoreHIBPResult(passwordHash string, breachCount int) error
	GetHIBPResult(passwordHash string) (int, bool, error)
	GetAllPasswordHashes() ([]string, error)
}

package storage

import (
	"github.com/hazcod/shade/pkg/events"
	"github.com/sirupsen/logrus"
)

type DuplicatePasswordEntry struct {
	User    string
	Domains []string
}

type Driver interface {
	Init(logger *logrus.Logger, settings map[string]string) error
	AddLoginEvent(data events.LoginEvent) error
	GetAllDomains() ([]string, error)
	GetDomainsForUser(username string) ([]string, error)
	GetDuplicatePasswordsForUser(username string) ([][]string, error)
	IsDuplicatePassword(username, passwordHash string) ([]string, error)
	GetDuplicatePasswords() (map[string]map[string]string, error)
	IsValidToken(token string) (bool, error)
}

package storage

import (
	"fmt"
	"github.com/hazcod/shade/pkg/storage/memory"
	"github.com/sirupsen/logrus"
	"strings"
)

func GetDriver(logger *logrus.Logger, driverName string, properties map[string]string) (Driver, error) {
	switch strings.ToLower(driverName) {
	case "memory":
		driver := &memory.InMemoryStore{}
		if err := driver.Init(logger, properties); err != nil {
			return nil, fmt.Errorf("failed to create memory driver: %v", err)
		}
		return driver, nil
	default:
		return nil, fmt.Errorf("unknown driver: %s", driverName)
	}
}

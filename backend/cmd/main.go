package main

import (
	"flag"
	"fmt"
	"github.com/hazcod/shade/config"
	"github.com/hazcod/shade/pkg/service/health"
	"github.com/hazcod/shade/pkg/service/login"
	"github.com/hazcod/shade/pkg/service/password"
	"github.com/hazcod/shade/pkg/service/web"
	"github.com/hazcod/shade/pkg/storage"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
)

// LoginData represents the login information captured by the extension

// InMemoryStore is a simple in-memory storage for login data
// In a production environment, this would be replaced with a proper database

func main() {
	logger := logrus.New()

	cfgPath := flag.String("config", "", "path to config file")
	logLevel := flag.String("log", "", "log level")
	flag.Parse()

	cfg, err := config.LoadConfig(*cfgPath)
	if err != nil {
		logger.WithError(err).Fatal("error loading config")
	}

	levelToUse := cfg.Log.Level
	if *logLevel != "" {
		levelToUse = *logLevel
	}

	logrusLevel, err := logrus.ParseLevel(levelToUse)
	if err != nil {
		logger.WithError(err).Fatal("error parsing log level")
	}

	logger.WithField("level", logrusLevel.String()).Info("set log level")
	logger.SetLevel(logrusLevel)

	// ---

	// Create storage
	storageDriver, err := storage.GetDriver(logger, cfg.Storage.Type, cfg.Storage.Properties)
	if err != nil {
		logger.WithError(err).Fatal("error loading memory store")
	}
	logger.WithField("driver", cfg.Storage.Type).Info("registered storage driver")

	// Set up HTTP server
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/health", health.HandleHealthCheck(logger, storageDriver))
	mux.HandleFunc("/api/login/register", login.HandleLoginData(logger, storageDriver))
	mux.HandleFunc("/api/password/domaincheck", password.CheckDuplicatePassword(logger, storageDriver))

	// web endpoints
	mux.HandleFunc("/", web.GetDashboard(logger, storageDriver))

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.HTTP.Interface, cfg.HTTP.Port)
	logger.WithField("listener", addr).Info("started server")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

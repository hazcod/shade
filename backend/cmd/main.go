package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/csrf"
	gorillamux "github.com/gorilla/mux"
	"github.com/hazcod/shade/config"
	"github.com/hazcod/shade/pkg/auth"
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

	// --

	devMode := cfg.HTTP.Interface == "127.0.0.1" || cfg.HTTP.Interface == "localhost"

	// ---

	// Create storage
	storageDriver, err := storage.GetDriver(logger, cfg.Storage.Type, cfg.Storage.Properties)
	if err != nil {
		logger.WithError(err).Fatal("error loading storage driver")
	}
	logger.WithField("driver", cfg.Storage.Type).Info("registered storage driver")

	// Create auth provider
	authProperties := make(map[string]interface{})
	for k, v := range cfg.Auth.Properties {
		authProperties[k] = v
	}
	authProperties["secret"] = cfg.Auth.Secret
	authProvider, err := auth.GetProvider(logger, cfg.Auth.Type, devMode, authProperties)
	if err != nil {
		logger.WithError(err).Fatal("error initializing authentication provider")
	}
	logger.WithField("provider", cfg.Auth.Type).Info("registered authentication provider")

	// CSRF protections
	logger.WithField("origin", cfg.HTTP.Origin).Info("setting up CSRF protection")
	sameSiteMode := csrf.SameSiteStrictMode
	if devMode {
		sameSiteMode = csrf.SameSiteLaxMode
	}

	// Configure CSRF options based on environment
	csrfOptions := []csrf.Option{
		csrf.Secure(!devMode),
		csrf.CookieName("csrf"),
		csrf.RequestHeader("X-CSRF-Token"),
		csrf.Path("/"),
		csrf.FieldName("csrf"),
		csrf.SameSite(sameSiteMode),
		csrf.MaxAge(3600),
	}
	// Only add TrustedOrigins in production mode to avoid origin validation issues in development
	csrfOptions = append(csrfOptions, csrf.TrustedOrigins([]string{cfg.HTTP.Origin}))
	logger.Info("CSRF TrustedOrigins configured for production")
	// setup csrf http middleware
	csrfMiddleware := csrf.Protect([]byte(cfg.Auth.Secret), csrfOptions...)

	// Set up HTTP server
	mux := gorillamux.NewRouter()

	protected := mux.PathPrefix("/").Subrouter()
	if !devMode {
		protected.Use(csrfMiddleware)
	}
	// Root redirect to dashboard, will redirect to login if not authenticated
	protected.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard/", http.StatusSeeOther)
	})

	// Authentication endpoints
	protected.PathPrefix("/auth/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logger.WithError(err).Debug("failed to parse form")
		}

		switch r.URL.Path {
		case "/auth/login":
			if r.Method == http.MethodGet {
				// Call the handler directly instead of wrapping it
				authProvider.RenderLoginPage().ServeHTTP(w, r)
			} else if r.Method == http.MethodPost {
				// Call the handler directly instead of wrapping it
				authProvider.HandleLogin().ServeHTTP(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		case "/auth/logout":
			authProvider.HandleLogout().ServeHTTP(w, r)
		case "/auth/callback":
			authProvider.HandleCallback().ServeHTTP(w, r)
		default:
			logger.WithField("path", r.URL.Path).Warn("unknown auth endpoint")
			http.NotFound(w, r)
		}
	}))

	// Protected web endpoints
	protected.PathPrefix("/dashboard/").Handler(authProvider.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dashboard/":
			web.GetDashboard(logger, storageDriver).ServeHTTP(w, r)
		case "/dashboard/saas":
			web.GetSaasPage(logger, storageDriver).ServeHTTP(w, r)
		case "/dashboard/security":
			web.GetSecurityPage(logger, storageDriver).ServeHTTP(w, r)
		case "/dashboard/endpoints":
			web.GetUsersPage(logger, storageDriver).ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})))

	// Static file handler for embedded files
	protected.PathPrefix("/static/").Handler(authProvider.Middleware(web.GetStaticFile(logger)))

	// API endpoints to be used by the extension
	mux.PathPrefix("/api/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			health.HandleHealthCheck(logger, storageDriver).ServeHTTP(w, r)
		case "/api/creds/register":
			login.HandleLoginData(logger, storageDriver).ServeHTTP(w, r)
		case "/api/password/domaincheck":
			password.CheckDuplicatePassword(logger, storageDriver).ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	}))

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.HTTP.Interface, cfg.HTTP.Port)
	logger.WithField("listener", addr).WithField("dev_mode", devMode).
		Info("started server")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

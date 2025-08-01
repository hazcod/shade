package web

import (
	"embed"
	"github.com/hazcod/shade/pkg/auth/session"
	"github.com/hazcod/shade/pkg/models"
	"github.com/hazcod/shade/pkg/storage"
	"github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"strings"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

//go:embed static/js/*.js
var staticFS embed.FS

// Templates loaded from embedded files
var dashboardTmpl = template.Must(template.ParseFS(templateFS, "templates/base.tmpl", "templates/dashboard.tmpl"))
var saasTmpl = template.Must(template.ParseFS(templateFS, "templates/base.tmpl", "templates/saas.tmpl"))
var securityTmpl = template.Must(template.ParseFS(templateFS, "templates/base.tmpl", "templates/security.tmpl"))
var usersTmpl = template.Must(template.ParseFS(templateFS, "templates/base.tmpl", "templates/users.tmpl"))

// Static file handler for embedded files
func GetStaticFile(logger *logrus.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// Remove /static/ prefix to get the file path
		filePath := r.URL.Path[8:] // Remove "/static/"

		data, err := staticFS.ReadFile("static/" + filePath)
		if err != nil {
			logger.WithError(err).WithField("file", filePath).Error("error reading static file")
			http.NotFound(w, r)
			return
		}

		// Set appropriate content type
		if strings.HasSuffix(filePath, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(filePath, ".css") {
			w.Header().Set("Content-Type", "text/css")
		}

		w.Write(data)
	}
}

// Data structures for different pages
type baseData struct {
	Title       string
	Username    string
	CurrentPage string
}

type dashboardPageData struct {
	baseData
	Stats models.DashboardStats
}

type saasPageData struct {
	baseData
	Domains []string
}

type securityPageData struct {
	baseData
	DuplicatePasswords map[string]map[string]string
	UsersWithoutMFA    []string
}

type usersPageData struct {
	baseData
	Users []models.EnrolledUser
}

// Dashboard stats page handler
func GetDashboard(logger *logrus.Logger, store storage.Driver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		user, err := session.GetUser(r)
		if err != nil {
			logger.WithError(err).Error("error getting user from session")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		stats, err := store.GetDashboardStats()
		if err != nil {
			logger.WithError(err).Error("error getting dashboard stats")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data := dashboardPageData{
			baseData: baseData{
				Title:       "Dashboard",
				Username:    user.Email,
				CurrentPage: "dashboard",
			},
			Stats: stats,
		}

		w.Header().Set("Content-Type", "text/html")
		if err := dashboardTmpl.Execute(w, data); err != nil {
			logger.WithError(err).Error("error rendering template")
			http.Error(w, "Template Error", http.StatusInternalServerError)
		}
	}
}

// SaaS discovery page handler
func GetSaasPage(logger *logrus.Logger, store storage.Driver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		user, err := session.GetUser(r)
		if err != nil {
			logger.WithError(err).Error("error getting user from session")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		domains, err := store.GetAllDomains()
		if err != nil {
			logger.WithError(err).Error("error getting all domains")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data := saasPageData{
			baseData: baseData{
				Title:       "Discovered SaaS",
				Username:    user.Email,
				CurrentPage: "saas",
			},
			Domains: domains,
		}

		w.Header().Set("Content-Type", "text/html")
		if err := saasTmpl.Execute(w, data); err != nil {
			logger.WithError(err).Error("error rendering template")
			http.Error(w, "Template Error", http.StatusInternalServerError)
		}
	}
}

// Password security page handler
func GetSecurityPage(logger *logrus.Logger, store storage.Driver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		user, err := session.GetUser(r)
		if err != nil {
			logger.WithError(err).Error("error getting user from session")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		dupePasswords, err := store.GetDuplicatePasswords()
		if err != nil {
			logger.WithError(err).Error("error getting duplicate passwords")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		usersWithoutMFA, err := store.GetUsersWithoutMFA()
		if err != nil {
			logger.WithError(err).Error("error getting users without MFA")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data := securityPageData{
			baseData: baseData{
				Title:       "Password Security",
				Username:    user.Email,
				CurrentPage: "security",
			},
			DuplicatePasswords: dupePasswords,
			UsersWithoutMFA:    usersWithoutMFA,
		}

		w.Header().Set("Content-Type", "text/html")
		if err := securityTmpl.Execute(w, data); err != nil {
			logger.WithError(err).Error("error rendering template")
			http.Error(w, "Template Error", http.StatusInternalServerError)
		}
	}
}

// Enrolled users page handler
func GetUsersPage(logger *logrus.Logger, store storage.Driver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		user, err := session.GetUser(r)
		if err != nil {
			logger.WithError(err).Error("error getting user from session")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		users, err := store.GetEnrolledUsers()
		if err != nil {
			logger.WithError(err).Error("error getting enrolled users")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data := usersPageData{
			baseData: baseData{
				Title:       "Endpoints",
				Username:    user.Email,
				CurrentPage: "endpoints",
			},
			Users: users,
		}

		w.Header().Set("Content-Type", "text/html")
		if err := usersTmpl.Execute(w, data); err != nil {
			logger.WithError(err).Error("error rendering template")
			http.Error(w, "Template Error", http.StatusInternalServerError)
		}
	}
}

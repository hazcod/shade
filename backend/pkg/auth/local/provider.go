package local

import (
	"errors"
	"fmt"
	"github.com/gorilla/csrf"
	"github.com/hazcod/shade/pkg/auth/session"
	"github.com/hazcod/shade/pkg/model"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"net/http"
	"strings"
)

// UserCredential represents a local user's credentials
type UserCredential struct {
	PasswordHash string   `json:"password_hash"`
	Email        string   `json:"email"`
	Roles        []string `json:"roles"`
}

// Config represents the local authentication provider configuration
type Config struct {
	Users []UserCredential `json:"users"`
}

// Provider implements the auth.Provider interface for local authentication
type Provider struct {
	logger        *logrus.Logger
	config        *Config
	loginTemplate *template.Template
}

// NewProvider creates a new local authentication provider
func NewProvider(logger *logrus.Logger) *Provider {
	return &Provider{
		logger:        logger,
		loginTemplate: template.Must(template.New("login").Parse(loginTmpl)),
	}
}

// Initialize sets up the local authentication provider
func (p *Provider) Initialize(logger interface{}, config map[string]interface{}) error {
	// Convert the generic logger to a logrus logger
	logrusLogger, ok := logger.(*logrus.Logger)
	if !ok {
		return errors.New("logger must be a *logrus.Logger")
	}
	p.logger = logrusLogger

	// Extract users from the configuration
	usersConfig, ok := config["users"].([]interface{})
	if !ok || len(usersConfig) == 0 {
		return errors.New("users configuration must be provided")
	}

	// Convert the generic users configuration to UserCredential
	p.config = &Config{
		Users: make([]UserCredential, 0, len(usersConfig)),
	}

	for _, u := range usersConfig {
		userMap, ok := u.(map[string]interface{})
		if !ok {
			continue
		}

		user := UserCredential{}

		if username, ok := userMap["username"].(string); ok {
			user.Email = username
		}

		if passwordHash, ok := userMap["password_hash"].(string); ok {
			user.PasswordHash = passwordHash
		}

		if email, ok := userMap["email"].(string); ok {
			user.Email = email
		}

		if roles, ok := userMap["roles"].([]interface{}); ok {
			user.Roles = make([]string, 0, len(roles))
			for _, r := range roles {
				if role, ok := r.(string); ok {
					user.Roles = append(user.Roles, role)
				}
			}
		}

		p.config.Users = append(p.config.Users, user)
	}

	if len(p.config.Users) == 0 {
		return errors.New("no valid users found in configuration")
	}

	return nil
}

// Authenticate verifies the username and password
func (p *Provider) Authenticate(username, password string) (*model.User, error) {
	p.logger.WithFields(logrus.Fields{
		"username": username,
	}).Debug("auth request")

	for _, u := range p.config.Users {
		if !strings.EqualFold(u.Email, username) {
			continue
		}

		// Check the password hash
		err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
		if err != nil {
			return nil, fmt.Errorf("invalid password: %w", err)
		}

		// Authentication successful
		return &model.User{
			Email: u.Email,
			Roles: u.Roles,
		}, nil
	}

	return nil, fmt.Errorf("user %s not found", username)
}

// HandleLogin processes login requests
func (p *Provider) HandleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the login form
		err := r.ParseForm()
		if err != nil {
			p.logger.WithError(err).Error("Failed to parse login form")
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		// Get credentials from the form
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Authenticate the user
		user, err := p.Authenticate(username, password)
		if err != nil {
			p.logger.WithError(err).WithField("username", username).Info("Authentication failed")
			// Redirect back to login page with error message
			http.Redirect(w, r, "/auth/login?error=Invalid+credentials", http.StatusSeeOther)
			return
		}

		// Store the authenticated user in the session
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

func (p *Provider) HandleCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
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

// RenderLoginPage renders the login page
func (p *Provider) RenderLoginPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if the user is already authenticated
		if user, err := session.GetUser(r); user != nil || err != nil {
			// User is already logged in, redirect to dashboard
			http.Redirect(w, r, "/dashboard/", http.StatusSeeOther)
			return
		}

		// Get error message from query parameter
		errorMsg := r.URL.Query().Get("error")

		// Prepare template data
		templateData := map[string]interface{}{
			"Error":          errorMsg,
			csrf.TemplateTag: csrf.TemplateField(r), // Use gorilla/csrf's built-in template field
		}

		p.logger.WithFields(logrus.Fields{
			"csrf_token": csrf.Token(r),
			"method":     r.Method,
			"path":       r.URL.Path,
		}).Debug("rendering login form")

		w.Header().Set("Content-Type", "text/html")
		if err := p.loginTemplate.Execute(w, templateData); err != nil {
			p.logger.WithError(err).Error("Failed to render login template")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
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

// Login page template
const loginTmpl = `
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
        .form-control:focus {
            border-color: #0d6efd;
            box-shadow: 0 0 0 0.25rem rgba(13, 110, 253, 0.25);
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="card">
            <div class="card-body p-4 p-md-5">
                <h3 class="card-title text-center">Login to Shade</h3>

                {{if .Error}}
                <div class="alert alert-danger" role="alert">
                    {{.Error}}
                </div>
                {{end}}

                <form method="POST" action="/auth/login">
					{{ .csrfField }}

                    <div class="mb-3">
                        <label for="username" class="form-label">Username</label>
                        <input type="email" class="form-control" id="username" name="username" required autofocus>
                    </div>

                    <div class="mb-3">
                        <label for="password" class="form-label">Password</label>
                        <input type="password" class="form-control" id="password" name="password" required>
                    </div>

                    <button type="submit" class="btn btn-primary w-100 mt-3">Sign In</button>
                </form>
            </div>
        </div>
    </div>
</body>
</html>
`

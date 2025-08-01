package session

import (
	"encoding/gob"
	"github.com/gorilla/sessions"
	"github.com/hazcod/shade/pkg/model"
	"net/http"
)

const (
	// SessionName is the name of the session cookie
	SessionName = "shade-session"
	// UserKey is the key used to store the user in the session
	UserKey = "user"
)

var (
	// Store is the session store
	Store *sessions.CookieStore
)

// Initialize sets up the session store
func Initialize(sessionSecret string, devMode bool) {
	// Register custom types with gob for session storage
	gob.Register(&model.User{})

	sameSiteMode := http.SameSiteStrictMode
	if devMode {
		sameSiteMode = http.SameSiteLaxMode
	}

	// Create a new session store with the provided secret
	Store = sessions.NewCookieStore([]byte(sessionSecret))
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   !devMode,
		SameSite: sameSiteMode,
	}
}

// GetUser retrieves the currently authenticated user from the session
func GetUser(r *http.Request) (*model.User, error) {
	session, err := Store.Get(r, SessionName)
	if err != nil {
		return nil, err
	}

	userVal, ok := session.Values[UserKey]
	if !ok {
		return nil, nil
	}

	user, ok := userVal.(*model.User)
	if !ok {
		return nil, nil
	}

	return user, nil
}

// SetUser stores the user in the session
func SetUser(w http.ResponseWriter, r *http.Request, user *model.User) error {
	session, err := Store.Get(r, SessionName)
	if err != nil {
		return err
	}

	session.Values[UserKey] = user
	return session.Save(r, w)
}

// ClearSession removes the user from the session
func ClearSession(w http.ResponseWriter, r *http.Request) error {
	session, err := Store.Get(r, SessionName)
	if err != nil {
		return err
	}

	session.Values[UserKey] = nil
	return session.Save(r, w)
}

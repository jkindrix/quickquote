package handler

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
	"github.com/jkindrix/quickquote/internal/service"
)

// Context key for user
type contextKey string

const userContextKey contextKey = "user"

// GetUserFromContext retrieves the authenticated user from the context.
func GetUserFromContext(ctx context.Context) *domain.User {
	user, ok := ctx.Value(userContextKey).(*domain.User)
	if !ok {
		return nil
	}
	return user
}

// AuthMiddleware validates the session and adds the user to context.
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			h.logger.Debug("no session cookie found")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, err := h.authService.ValidateSession(r.Context(), cookie.Value)
		if err != nil {
			h.logger.Debug("invalid session", zap.Error(err))
			// Clear invalid cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "session_token",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HandleLoginPage renders the login page.
func (h *Handler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	// Check if already logged in
	cookie, err := r.Cookie("session_token")
	if err == nil {
		if _, err := h.authService.ValidateSession(r.Context(), cookie.Value); err == nil {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
	}

	h.renderTemplate(w, "login", map[string]interface{}{
		"Title": "Login",
	})
}

// HandleLogin processes login form submissions.
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", zap.Error(err))
		h.renderTemplate(w, "login", map[string]interface{}{
			"Title": "Login",
			"Error": "Invalid request",
		})
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		h.renderTemplate(w, "login", map[string]interface{}{
			"Title": "Login",
			"Error": "Email and password are required",
			"Email": email,
		})
		return
	}

	session, err := h.authService.Login(r.Context(), email, password)
	if err != nil {
		h.logger.Warn("login failed",
			zap.String("email", email),
			zap.Error(err),
		)

		errorMsg := "Invalid email or password"
		if _, ok := err.(*service.AuthError); !ok {
			errorMsg = "An error occurred. Please try again."
		}

		h.renderTemplate(w, "login", map[string]interface{}{
			"Title": "Login",
			"Error": errorMsg,
			"Email": email,
		})
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	h.logger.Info("user logged in successfully", zap.String("email", email))
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// HandleLogout logs the user out.
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		if err := h.authService.Logout(r.Context(), cookie.Value); err != nil {
			h.logger.Error("failed to logout", zap.Error(err))
		}
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// renderTemplate is a helper to render HTML templates.
func (h *Handler) renderTemplate(w http.ResponseWriter, name string, data map[string]interface{}) {
	// TODO: Implement with templ templates
	// For now, use a simple HTML response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch name {
	case "login":
		h.renderLoginHTML(w, data)
	case "dashboard":
		h.renderDashboardHTML(w, data)
	case "calls":
		h.renderCallsHTML(w, data)
	case "call_detail":
		h.renderCallDetailHTML(w, data)
	default:
		http.Error(w, "Template not found", http.StatusInternalServerError)
	}
}

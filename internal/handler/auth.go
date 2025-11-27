package handler

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
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

		result, err := h.authService.ValidateAndRefreshSession(r.Context(), cookie.Value)
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

		// If token was rotated, set the new cookie
		if result.NewToken != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "session_token",
				Value:    result.NewToken,
				Path:     "/",
				MaxAge:   86400 * 7, // 7 days
				HttpOnly: true,
				Secure:   r.TLS != nil,
				SameSite: http.SameSiteLaxMode,
			})
			h.logger.Debug("session token rotated for user")
		}

		ctx := context.WithValue(r.Context(), userContextKey, result.User)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getClientIP extracts the client IP from a request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return strings.TrimSpace(xff[:i])
			}
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr - use net.SplitHostPort for proper IPv4/IPv6 handling
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, return RemoteAddr as-is (may not have port)
		return r.RemoteAddr
	}
	return host
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

	// Get CSRF token
	csrfToken := h.getCSRFToken(r)

	h.renderTemplate(w, r, "login", map[string]interface{}{
		"Title":     "Login",
		"CSRFToken": csrfToken,
	})
}

// HandleLogin processes login form submissions.
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", zap.Error(err))
		h.renderTemplate(w, r, "login", map[string]interface{}{
			"Title":     "Login",
			"Error":     "Invalid request",
			"CSRFToken": h.getCSRFToken(r),
		})
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	ip := getClientIP(r)

	// Check login rate limit
	if h.loginRateLimiter != nil && !h.loginRateLimiter.Check(ip, email) {
		h.logger.Warn("login rate limited",
			zap.String("email", email),
			zap.String("ip", ip),
		)
		h.renderTemplate(w, r, "login", map[string]interface{}{
			"Title":     "Login",
			"Error":     "Too many login attempts. Please try again in 30 minutes.",
			"Email":     email,
			"CSRFToken": h.getCSRFToken(r),
		})
		return
	}

	if email == "" || password == "" {
		h.renderTemplate(w, r, "login", map[string]interface{}{
			"Title":     "Login",
			"Error":     "Email and password are required",
			"Email":     email,
			"CSRFToken": h.getCSRFToken(r),
		})
		return
	}

	// Create login context with IP and user agent
	loginCtx := &service.LoginContext{
		IPAddress: ip,
		UserAgent: r.UserAgent(),
	}

	session, err := h.authService.LoginWithContext(r.Context(), email, password, loginCtx)
	if err != nil {
		h.logger.Warn("login failed",
			zap.String("email", email),
			zap.Error(err),
		)

		errorMsg := "Invalid email or password"
		var authErr *service.AuthError
		if !errors.As(err, &authErr) {
			errorMsg = "An error occurred. Please try again."
		}

		// Add remaining attempts info
		remaining := 5
		if h.loginRateLimiter != nil {
			remaining = h.loginRateLimiter.RemainingAttempts(ip, email)
		}
		if remaining <= 2 && remaining > 0 {
			errorMsg = fmt.Sprintf("%s %d attempts remaining.", errorMsg, remaining)
		}

		h.renderTemplate(w, r, "login", map[string]interface{}{
			"Title":     "Login",
			"Error":     errorMsg,
			"Email":     email,
			"CSRFToken": h.getCSRFToken(r),
		})
		return
	}

	// Record successful login to reset rate limit
	if h.loginRateLimiter != nil {
		h.loginRateLimiter.RecordSuccess(ip, email)
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

// getCSRFToken returns the CSRF token for the current request.
func (h *Handler) getCSRFToken(r *http.Request) string {
	if h.csrfProtection != nil {
		return h.csrfProtection.GetToken(r)
	}
	return ""
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
func (h *Handler) renderTemplate(w http.ResponseWriter, r *http.Request, name string, data map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Add CSRF token to all templates if not already present
	if _, ok := data["CSRFToken"]; !ok {
		data["CSRFToken"] = h.getCSRFToken(r)
	}

	// Use template engine if available
	if h.templateEngine != nil && h.templateEngine.HasTemplate(name) {
		if err := h.templateEngine.Render(w, name, data); err != nil {
			h.logger.Error("failed to render template", zap.String("name", name), zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Fallback to inline templates
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

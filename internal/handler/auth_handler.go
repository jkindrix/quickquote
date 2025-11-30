package handler

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/metrics"
	"github.com/jkindrix/quickquote/internal/middleware"
	"github.com/jkindrix/quickquote/internal/service"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	*BaseHandler
	authService      *service.AuthService
	loginRateLimiter *middleware.LoginRateLimiter
	metrics          *metrics.Metrics
}

// AuthHandlerConfig holds configuration for AuthHandler.
type AuthHandlerConfig struct {
	Base             BaseHandlerConfig
	AuthService      *service.AuthService
	LoginRateLimiter *middleware.LoginRateLimiter
	Metrics          *metrics.Metrics
}

// NewAuthHandler creates a new AuthHandler with all required dependencies.
func NewAuthHandler(cfg AuthHandlerConfig) *AuthHandler {
	if cfg.AuthService == nil {
		panic("authService is required")
	}
	return &AuthHandler{
		BaseHandler:      NewBaseHandler(cfg.Base),
		authService:      cfg.AuthService,
		loginRateLimiter: cfg.LoginRateLimiter,
		metrics:          cfg.Metrics,
	}
}

// RegisterRoutes registers auth routes on the router.
func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.HandleIndex)
	r.Get("/login", h.HandleLoginPage)
	r.With(middleware.BodySizeLimiterForm()).Post("/login", h.HandleLogin)
	r.Get("/logout", h.HandleLogout)
}

// Middleware returns the authentication middleware.
func (h *AuthHandler) Middleware(next http.Handler) http.Handler {
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
			h.setSessionCookie(w, r, result.NewToken, 86400*7)
			h.logger.Debug("session token rotated for user")
		}

		ctx := context.WithValue(r.Context(), userContextKey, result.User)
		if result.User != nil {
			ctx = middleware.WithUserID(ctx, result.User.ID)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HandleIndex redirects to dashboard or login based on auth status.
func (h *AuthHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		if _, err := h.authService.ValidateSession(r.Context(), cookie.Value); err == nil {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// HandleLoginPage renders the login page.
func (h *AuthHandler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	// Check if already logged in
	cookie, err := r.Cookie("session_token")
	if err == nil {
		if _, err := h.authService.ValidateSession(r.Context(), cookie.Value); err == nil {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
	}

	h.Render(w, r, "login", &LoginPageData{
		Title: "Login",
	})
}

// HandleLogin processes login form submissions.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.logger.Error("failed to parse form", zap.Error(err))
		h.Render(w, r, "login", &LoginPageData{
			Title: "Login",
			Error: "Invalid request",
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
		if h.metrics != nil {
			h.metrics.RecordAuthRateLimited()
		}
		h.Render(w, r, "login", &LoginPageData{
			Title: "Login",
			Error: "Too many login attempts. Please try again in 30 minutes.",
			Email: email,
		})
		return
	}

	if email == "" || password == "" {
		h.Render(w, r, "login", &LoginPageData{
			Title: "Login",
			Error: "Email and password are required",
			Email: email,
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
		if h.metrics != nil {
			h.metrics.RecordAuthAttempt(false)
		}

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

		h.Render(w, r, "login", &LoginPageData{
			Title: "Login",
			Error: errorMsg,
			Email: email,
		})
		return
	}

	// Record successful login to reset rate limit
	if h.loginRateLimiter != nil {
		h.loginRateLimiter.RecordSuccess(ip, email)
	}

	if h.metrics != nil {
		h.metrics.RecordAuthAttempt(true)
		h.metrics.RecordSessionCreated()
	}

	// Set session cookie
	h.setSessionCookie(w, r, session.Token, int(time.Until(session.ExpiresAt).Seconds()))

	h.logger.Info("user logged in successfully", zap.String("email", email))
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// HandleLogout logs the user out.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		if logoutErr := h.authService.Logout(r.Context(), cookie.Value); logoutErr != nil {
			h.logger.Error("failed to logout", zap.Error(logoutErr))
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

// setSessionCookie sets the session cookie with proper security flags.
func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, maxAge int) {
	// Always use Secure in production
	secure := isProduction() || r.TLS != nil

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// isProduction checks if running in production environment.
func isProduction() bool {
	env := os.Getenv("APP_ENV")
	return env == "production" || env == "prod"
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

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

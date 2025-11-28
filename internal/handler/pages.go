package handler

import (
	"fmt"
	"html"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
)

// HandleIndex serves the landing page.
func (h *Handler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	// Check if logged in, redirect to dashboard
	cookie, err := r.Cookie("session_token")
	if err == nil {
		if _, err := h.authService.ValidateSession(r.Context(), cookie.Value); err == nil {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// HandleDashboard serves the main dashboard.
func (h *Handler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get recent calls
	calls, total, err := h.callService.ListCalls(r.Context(), 1, 10)
	if err != nil {
		h.logger.Error("failed to list calls", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.renderTemplate(w, r, "dashboard", map[string]interface{}{
		"Title":         "Dashboard",
		"ActiveNav":     "dashboard",
		"User":          user,
		"Calls":         calls,
		"TotalCalls":    total,
		"PendingQuotes": countPendingQuotes(calls),
	})
}

// HandleCallsList serves the calls list page.
func (h *Handler) HandleCallsList(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if _, err := fmt.Sscanf(p, "%d", &page); err != nil || page < 1 {
			page = 1 // Default to page 1 on invalid input
		}
	}

	calls, total, err := h.callService.ListCalls(r.Context(), page, 20)
	if err != nil {
		h.logger.Error("failed to list calls", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	pageSize := 20
	totalPages := (total + pageSize - 1) / pageSize

	h.renderTemplate(w, r, "calls", map[string]interface{}{
		"Title":      "Calls",
		"ActiveNav":  "calls",
		"User":       user,
		"Calls":      calls,
		"TotalCalls": total,
		"Page":       page,
		"PageSize":   pageSize,
		"TotalPages": totalPages,
	})
}

// HandleCallDetail serves a single call detail page.
func (h *Handler) HandleCallDetail(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid call ID", http.StatusBadRequest)
		return
	}

	call, err := h.callService.GetCall(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get call", zap.Error(err), zap.String("id", idStr))
		http.Error(w, "Call not found", http.StatusNotFound)
		return
	}

	h.renderTemplate(w, r, "call_detail", map[string]interface{}{
		"Title":     "Call Details",
		"ActiveNav": "calls",
		"User":      user,
		"Call":      call,
	})
}

// HandleRegenerateQuote regenerates the quote for a call.
func (h *Handler) HandleRegenerateQuote(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid call ID", http.StatusBadRequest)
		return
	}

	call, err := h.callService.GenerateQuote(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to regenerate quote", zap.Error(err), zap.String("id", idStr))
		http.Error(w, "Failed to regenerate quote", http.StatusInternalServerError)
		return
	}

	// For htmx requests, return just the quote section
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		h.renderQuoteSection(w, call)
		return
	}

	// Otherwise redirect to call detail
	http.Redirect(w, r, fmt.Sprintf("/calls/%s", id), http.StatusSeeOther)
}

// renderQuoteSection renders just the quote section for htmx updates.
func (h *Handler) renderQuoteSection(w http.ResponseWriter, call *domain.Call) {
	quote := "No quote generated yet"
	if call.QuoteSummary != nil {
		quote = *call.QuoteSummary
	}

	csrfToken := ""
	if h.csrfProtection != nil {
		var err error
		csrfToken, err = h.csrfProtection.GenerateToken()
		if err != nil {
			h.logger.Error("failed to generate CSRF token", zap.Error(err))
		}
	}

	fmt.Fprintf(w, `
		<div class="card" id="quote-section">
			<h2>Generated Quote</h2>
			<div class="quote-content">
				<pre>%s</pre>
			</div>
			<form hx-post="/calls/%s/regenerate-quote"
				  hx-target="#quote-section"
				  hx-swap="outerHTML"
				  hx-indicator="#quote-loading"
				  style="margin-top: 1rem;">
				<input type="hidden" name="csrf_token" value="%s">
				<button type="submit" class="btn btn-secondary">
					Regenerate Quote
				</button>
				<span id="quote-loading" class="htmx-indicator">Generating...</span>
			</form>
		</div>
	`, html.EscapeString(quote), call.ID, html.EscapeString(csrfToken))
}

// countPendingQuotes counts calls that are completed but don't have quotes.
func countPendingQuotes(calls []*domain.Call) int {
	count := 0
	for _, call := range calls {
		if call.Status == domain.CallStatusCompleted && (call.QuoteSummary == nil || *call.QuoteSummary == "") {
			count++
		}
	}
	return count
}

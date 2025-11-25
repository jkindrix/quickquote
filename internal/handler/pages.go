package handler

import (
	"fmt"
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

	h.renderTemplate(w, "dashboard", map[string]interface{}{
		"Title":      "Dashboard",
		"User":       user,
		"Calls":      calls,
		"TotalCalls": total,
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
		fmt.Sscanf(p, "%d", &page)
	}

	calls, total, err := h.callService.ListCalls(r.Context(), page, 20)
	if err != nil {
		h.logger.Error("failed to list calls", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.renderTemplate(w, "calls", map[string]interface{}{
		"Title":      "Calls",
		"User":       user,
		"Calls":      calls,
		"TotalCalls": total,
		"Page":       page,
		"PageSize":   20,
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

	h.renderTemplate(w, "call_detail", map[string]interface{}{
		"Title": "Call Details",
		"User":  user,
		"Call":  call,
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

	fmt.Fprintf(w, `
		<div id="quote-section" class="quote-section">
			<h3>Generated Quote</h3>
			<div class="quote-content">%s</div>
			<button hx-post="/calls/%s/regenerate-quote"
					hx-target="#quote-section"
					hx-swap="outerHTML"
					class="btn btn-secondary">
				Regenerate Quote
			</button>
		</div>
	`, quote, call.ID)
}

package handler

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
	"github.com/jkindrix/quickquote/internal/middleware"
	"github.com/jkindrix/quickquote/internal/service"
)

// CallsHandler handles call-related HTTP requests including dashboard.
type CallsHandler struct {
	*BaseHandler
	callService *service.CallService
}

// CallsHandlerConfig holds configuration for CallsHandler.
type CallsHandlerConfig struct {
	Base        BaseHandlerConfig
	CallService *service.CallService
}

// NewCallsHandler creates a new CallsHandler with all required dependencies.
func NewCallsHandler(cfg CallsHandlerConfig) *CallsHandler {
	if cfg.CallService == nil {
		panic("callService is required")
	}
	return &CallsHandler{
		BaseHandler: NewBaseHandler(cfg.Base),
		callService: cfg.CallService,
	}
}

// RegisterRoutes registers calls routes on the router.
// Note: These routes require authentication middleware to be applied by the caller.
func (h *CallsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/dashboard", h.HandleDashboard)
	r.Get("/calls", h.HandleCallsList)
	r.Get("/calls/{id}", h.HandleCallDetail)
	r.Post("/calls/{id}/regenerate-quote", h.HandleRegenerateQuote)
}

// HandleDashboard serves the main dashboard.
func (h *CallsHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	calls, total, err := h.callService.ListCalls(r.Context(), 1, 10, nil)
	if err != nil {
		h.logger.Error("failed to list calls", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.Render(w, r, "dashboard", &DashboardPageData{
		BasePageData: BasePageData{
			Title:     "Dashboard",
			ActiveNav: "dashboard",
			User:      user,
		},
		Calls:         calls,
		TotalCalls:    total,
		PendingQuotes: countPendingQuotes(calls),
	})
}

// HandleCallsList serves the calls list page.
func (h *CallsHandler) HandleCallsList(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if _, err := fmt.Sscanf(p, "%d", &page); err != nil || page < 1 {
			page = 1
		}
	}

	query := r.URL.Query()
	statusParam := strings.TrimSpace(query.Get("status"))
	searchParam := strings.TrimSpace(query.Get("q"))

	filter := buildCallListFilter(statusParam, searchParam)

	calls, total, err := h.callService.ListCalls(r.Context(), page, 20, filter)
	if err != nil {
		h.logger.Error("failed to list calls", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	pageSize := 20
	totalPages := (total + pageSize - 1) / pageSize

	h.Render(w, r, "calls", &CallsPageData{
		BasePageData: BasePageData{
			Title:     "Calls",
			ActiveNav: "calls",
			User:      user,
		},
		Calls:      calls,
		TotalCalls: total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		Filter: CallListFilterView{
			Status: statusParam,
			Query:  searchParam,
		},
	})
}

// HandleCallDetail serves a single call detail page.
func (h *CallsHandler) HandleCallDetail(w http.ResponseWriter, r *http.Request) {
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

	h.Render(w, r, "call_detail", &CallDetailPageData{
		BasePageData: BasePageData{
			Title:     "Call Details",
			ActiveNav: "calls",
			User:      user,
		},
		Call: call,
	})
}

// HandleRegenerateQuote regenerates the quote for a call.
func (h *CallsHandler) HandleRegenerateQuote(w http.ResponseWriter, r *http.Request) {
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
		h.renderQuoteSection(w, r, call)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/calls/%s", id), http.StatusSeeOther)
}

// renderQuoteSection renders just the quote section for htmx updates.
func (h *CallsHandler) renderQuoteSection(w http.ResponseWriter, r *http.Request, call *domain.Call) {
	quote := "No quote generated yet"
	if call.QuoteSummary != nil {
		quote = *call.QuoteSummary
	}

	csrfToken := h.GetCSRFToken(r)

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

// CallListFilterView holds the UI filter state.
type CallListFilterView struct {
	Status string
	Query  string
}

// buildCallListFilter creates a domain filter from UI inputs.
func buildCallListFilter(status, search string) *domain.CallListFilter {
	var filter domain.CallListFilter

	if status != "" {
		switch domain.CallStatus(status) {
		case domain.CallStatusPending,
			domain.CallStatusInProgress,
			domain.CallStatusCompleted,
			domain.CallStatusFailed,
			domain.CallStatusNoAnswer:
			statusValue := domain.CallStatus(status)
			filter.Status = &statusValue
		}
	}

	if strings.TrimSpace(search) != "" {
		filter.Search = search
	}

	if filter.Status == nil && strings.TrimSpace(filter.Search) == "" {
		return nil
	}
	return &filter
}

// CSRFMiddleware returns a CSRF validation middleware for the calls handler.
func CSRFMiddleware(csrf *middleware.CSRFProtection) func(http.Handler) http.Handler {
	if csrf == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}
	return csrf.Middleware
}

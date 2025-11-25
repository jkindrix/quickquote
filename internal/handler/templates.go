package handler

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/jkindrix/quickquote/internal/domain"
)

// renderLoginHTML renders the login page HTML.
func (h *Handler) renderLoginHTML(w http.ResponseWriter, data map[string]interface{}) {
	errorMsg := ""
	if e, ok := data["Error"].(string); ok {
		errorMsg = fmt.Sprintf(`<div class="alert alert-error">%s</div>`, html.EscapeString(e))
	}

	email := ""
	if e, ok := data["Email"].(string); ok {
		email = html.EscapeString(e)
	}

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - QuickQuote</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; min-height: 100vh; display: flex; align-items: center; justify-content: center; }
        .login-container { background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); width: 100%%; max-width: 400px; }
        h1 { margin-bottom: 1.5rem; color: #333; text-align: center; }
        .form-group { margin-bottom: 1rem; }
        label { display: block; margin-bottom: 0.5rem; color: #555; font-weight: 500; }
        input[type="email"], input[type="password"] { width: 100%%; padding: 0.75rem; border: 1px solid #ddd; border-radius: 4px; font-size: 1rem; }
        input:focus { outline: none; border-color: #007bff; box-shadow: 0 0 0 2px rgba(0,123,255,0.25); }
        .btn { width: 100%%; padding: 0.75rem; background: #007bff; color: white; border: none; border-radius: 4px; font-size: 1rem; cursor: pointer; }
        .btn:hover { background: #0056b3; }
        .alert { padding: 0.75rem; border-radius: 4px; margin-bottom: 1rem; }
        .alert-error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .logo { text-align: center; margin-bottom: 1rem; font-size: 2rem; }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="logo">📞 QuickQuote</div>
        <h1>Sign In</h1>
        %s
        <form method="POST" action="/login">
            <div class="form-group">
                <label for="email">Email</label>
                <input type="email" id="email" name="email" value="%s" required autofocus>
            </div>
            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit" class="btn">Sign In</button>
        </form>
    </div>
</body>
</html>`, errorMsg, email)
}

// renderDashboardHTML renders the dashboard page HTML.
func (h *Handler) renderDashboardHTML(w http.ResponseWriter, data map[string]interface{}) {
	user := data["User"].(*domain.User)
	calls := data["Calls"].([]*domain.Call)
	totalCalls := data["TotalCalls"].(int)

	var callRows strings.Builder
	for _, call := range calls {
		statusClass := getStatusClass(call.Status)
		callerName := "Unknown"
		if call.CallerName != nil {
			callerName = *call.CallerName
		}
		callRows.WriteString(fmt.Sprintf(`
			<tr>
				<td>%s</td>
				<td>%s</td>
				<td><span class="status %s">%s</span></td>
				<td>%s</td>
				<td><a href="/calls/%s" class="btn btn-sm">View</a></td>
			</tr>`,
			html.EscapeString(callerName),
			html.EscapeString(call.PhoneNumber),
			statusClass,
			call.Status,
			formatTime(call.CreatedAt),
			call.ID,
		))
	}

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard - QuickQuote</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    %s
</head>
<body>
    %s
    <main class="container">
        <div class="dashboard-header">
            <h1>Dashboard</h1>
            <p>Welcome back! You have <strong>%d</strong> total calls.</p>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <h3>Total Calls</h3>
                <p class="stat-number">%d</p>
            </div>
            <div class="stat-card">
                <h3>Pending Quotes</h3>
                <p class="stat-number">%d</p>
            </div>
        </div>

        <div class="card">
            <div class="card-header">
                <h2>Recent Calls</h2>
                <a href="/calls" class="btn btn-secondary">View All</a>
            </div>
            <table class="table">
                <thead>
                    <tr>
                        <th>Caller</th>
                        <th>Phone</th>
                        <th>Status</th>
                        <th>Date</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    %s
                </tbody>
            </table>
        </div>
    </main>
</body>
</html>`, getStyles(), getNavbar(user), totalCalls, totalCalls, countPendingQuotes(calls), callRows.String())
}

// renderCallsHTML renders the calls list page HTML.
func (h *Handler) renderCallsHTML(w http.ResponseWriter, data map[string]interface{}) {
	user := data["User"].(*domain.User)
	calls := data["Calls"].([]*domain.Call)
	totalCalls := data["TotalCalls"].(int)
	page := data["Page"].(int)
	pageSize := data["PageSize"].(int)

	var callRows strings.Builder
	for _, call := range calls {
		statusClass := getStatusClass(call.Status)
		callerName := "Unknown"
		if call.CallerName != nil {
			callerName = *call.CallerName
		}
		duration := "-"
		if call.DurationSeconds != nil {
			duration = fmt.Sprintf("%d sec", *call.DurationSeconds)
		}
		hasQuote := "No"
		if call.QuoteSummary != nil && *call.QuoteSummary != "" {
			hasQuote = "Yes"
		}
		callRows.WriteString(fmt.Sprintf(`
			<tr>
				<td>%s</td>
				<td>%s</td>
				<td><span class="status %s">%s</span></td>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td><a href="/calls/%s" class="btn btn-sm">View</a></td>
			</tr>`,
			html.EscapeString(callerName),
			html.EscapeString(call.PhoneNumber),
			statusClass,
			call.Status,
			duration,
			hasQuote,
			formatTime(call.CreatedAt),
			call.ID,
		))
	}

	totalPages := (totalCalls + pageSize - 1) / pageSize
	pagination := buildPagination(page, totalPages)

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Calls - QuickQuote</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    %s
</head>
<body>
    %s
    <main class="container">
        <div class="page-header">
            <h1>Call History</h1>
            <p>Showing %d of %d calls</p>
        </div>

        <div class="card">
            <table class="table">
                <thead>
                    <tr>
                        <th>Caller</th>
                        <th>Phone</th>
                        <th>Status</th>
                        <th>Duration</th>
                        <th>Quote</th>
                        <th>Date</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    %s
                </tbody>
            </table>
            %s
        </div>
    </main>
</body>
</html>`, getStyles(), getNavbar(user), len(calls), totalCalls, callRows.String(), pagination)
}

// renderCallDetailHTML renders a single call detail page HTML.
func (h *Handler) renderCallDetailHTML(w http.ResponseWriter, data map[string]interface{}) {
	user := data["User"].(*domain.User)
	call := data["Call"].(*domain.Call)

	callerName := "Unknown"
	if call.CallerName != nil {
		callerName = *call.CallerName
	}

	transcript := "No transcript available"
	if call.Transcript != nil {
		transcript = *call.Transcript
	}

	quote := "No quote generated yet"
	if call.QuoteSummary != nil {
		quote = *call.QuoteSummary
	}

	duration := "-"
	if call.DurationSeconds != nil {
		duration = fmt.Sprintf("%d seconds", *call.DurationSeconds)
	}

	var extractedData strings.Builder
	if call.ExtractedData != nil {
		if call.ExtractedData.ProjectType != "" {
			extractedData.WriteString(fmt.Sprintf("<p><strong>Project Type:</strong> %s</p>", html.EscapeString(call.ExtractedData.ProjectType)))
		}
		if call.ExtractedData.Requirements != "" {
			extractedData.WriteString(fmt.Sprintf("<p><strong>Requirements:</strong> %s</p>", html.EscapeString(call.ExtractedData.Requirements)))
		}
		if call.ExtractedData.Timeline != "" {
			extractedData.WriteString(fmt.Sprintf("<p><strong>Timeline:</strong> %s</p>", html.EscapeString(call.ExtractedData.Timeline)))
		}
		if call.ExtractedData.BudgetRange != "" {
			extractedData.WriteString(fmt.Sprintf("<p><strong>Budget Range:</strong> %s</p>", html.EscapeString(call.ExtractedData.BudgetRange)))
		}
		if call.ExtractedData.ContactPreference != "" {
			extractedData.WriteString(fmt.Sprintf("<p><strong>Contact Preference:</strong> %s</p>", html.EscapeString(call.ExtractedData.ContactPreference)))
		}
	}
	if extractedData.Len() == 0 {
		extractedData.WriteString("<p>No extracted data available</p>")
	}

	statusClass := getStatusClass(call.Status)

	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Call Details - QuickQuote</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    %s
</head>
<body>
    %s
    <main class="container">
        <div class="page-header">
            <a href="/calls" class="back-link">← Back to Calls</a>
            <h1>Call with %s</h1>
        </div>

        <div class="detail-grid">
            <div class="card">
                <h2>Call Information</h2>
                <div class="info-list">
                    <p><strong>Phone:</strong> %s</p>
                    <p><strong>From:</strong> %s</p>
                    <p><strong>Status:</strong> <span class="status %s">%s</span></p>
                    <p><strong>Duration:</strong> %s</p>
                    <p><strong>Date:</strong> %s</p>
                </div>
            </div>

            <div class="card">
                <h2>Extracted Information</h2>
                <div class="info-list">
                    %s
                </div>
            </div>
        </div>

        <div class="card">
            <h2>Transcript</h2>
            <div class="transcript-box">
                <pre>%s</pre>
            </div>
        </div>

        <div class="card" id="quote-section">
            <h2>Generated Quote</h2>
            <div class="quote-content">
                <pre>%s</pre>
            </div>
            <button hx-post="/calls/%s/regenerate-quote"
                    hx-target="#quote-section"
                    hx-swap="outerHTML"
                    hx-indicator="#quote-loading"
                    class="btn btn-secondary">
                Regenerate Quote
            </button>
            <span id="quote-loading" class="htmx-indicator">Generating...</span>
        </div>
    </main>
</body>
</html>`,
		getStyles(),
		getNavbar(user),
		html.EscapeString(callerName),
		html.EscapeString(call.PhoneNumber),
		html.EscapeString(call.FromNumber),
		statusClass,
		call.Status,
		duration,
		formatTime(call.CreatedAt),
		extractedData.String(),
		html.EscapeString(transcript),
		html.EscapeString(quote),
		call.ID,
	)
}

// Helper functions

func getStatusClass(status domain.CallStatus) string {
	switch status {
	case domain.CallStatusCompleted:
		return "status-completed"
	case domain.CallStatusFailed:
		return "status-failed"
	case domain.CallStatusNoAnswer:
		return "status-no-answer"
	case domain.CallStatusInProgress:
		return "status-in-progress"
	default:
		return "status-pending"
	}
}

func formatTime(t time.Time) string {
	return t.Format("Jan 2, 2006 3:04 PM")
}

func countPendingQuotes(calls []*domain.Call) int {
	count := 0
	for _, call := range calls {
		if call.Status == domain.CallStatusCompleted && (call.QuoteSummary == nil || *call.QuoteSummary == "") {
			count++
		}
	}
	return count
}

func buildPagination(page, totalPages int) string {
	if totalPages <= 1 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`<div class="pagination">`)

	if page > 1 {
		sb.WriteString(fmt.Sprintf(`<a href="/calls?page=%d" class="btn btn-sm">Previous</a>`, page-1))
	}

	sb.WriteString(fmt.Sprintf(`<span class="page-info">Page %d of %d</span>`, page, totalPages))

	if page < totalPages {
		sb.WriteString(fmt.Sprintf(`<a href="/calls?page=%d" class="btn btn-sm">Next</a>`, page+1))
	}

	sb.WriteString(`</div>`)
	return sb.String()
}

func getNavbar(user *domain.User) string {
	return fmt.Sprintf(`
    <nav class="navbar">
        <div class="nav-brand">
            <a href="/dashboard">📞 QuickQuote</a>
        </div>
        <div class="nav-links">
            <a href="/dashboard">Dashboard</a>
            <a href="/calls">Calls</a>
        </div>
        <div class="nav-user">
            <span>%s</span>
            <a href="/logout" class="btn btn-sm btn-outline">Logout</a>
        </div>
    </nav>`, html.EscapeString(user.Email))
}

func getStyles() string {
	return `<style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; color: #333; }

        .navbar { background: #fff; padding: 1rem 2rem; display: flex; align-items: center; justify-content: space-between; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        .nav-brand a { font-size: 1.25rem; font-weight: 600; text-decoration: none; color: #333; }
        .nav-links { display: flex; gap: 1.5rem; }
        .nav-links a { text-decoration: none; color: #666; }
        .nav-links a:hover { color: #007bff; }
        .nav-user { display: flex; align-items: center; gap: 1rem; }

        .container { max-width: 1200px; margin: 0 auto; padding: 2rem; }

        .dashboard-header, .page-header { margin-bottom: 2rem; }
        .dashboard-header h1, .page-header h1 { margin-bottom: 0.5rem; }
        .back-link { display: inline-block; margin-bottom: 1rem; color: #007bff; text-decoration: none; }

        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
        .stat-card { background: #fff; padding: 1.5rem; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        .stat-card h3 { font-size: 0.875rem; color: #666; margin-bottom: 0.5rem; }
        .stat-number { font-size: 2rem; font-weight: 600; color: #007bff; }

        .detail-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1rem; margin-bottom: 1rem; }

        .card { background: #fff; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); padding: 1.5rem; margin-bottom: 1rem; }
        .card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
        .card h2 { font-size: 1.25rem; margin-bottom: 1rem; }

        .info-list p { margin-bottom: 0.5rem; }

        .table { width: 100%; border-collapse: collapse; }
        .table th, .table td { padding: 0.75rem; text-align: left; border-bottom: 1px solid #eee; }
        .table th { font-weight: 600; color: #666; font-size: 0.875rem; }
        .table tbody tr:hover { background: #f9f9f9; }

        .status { display: inline-block; padding: 0.25rem 0.75rem; border-radius: 9999px; font-size: 0.75rem; font-weight: 500; }
        .status-completed { background: #d4edda; color: #155724; }
        .status-failed { background: #f8d7da; color: #721c24; }
        .status-no-answer { background: #fff3cd; color: #856404; }
        .status-in-progress { background: #cce5ff; color: #004085; }
        .status-pending { background: #e2e3e5; color: #383d41; }

        .btn { display: inline-block; padding: 0.5rem 1rem; background: #007bff; color: white; border: none; border-radius: 4px; text-decoration: none; cursor: pointer; font-size: 0.875rem; }
        .btn:hover { background: #0056b3; }
        .btn-secondary { background: #6c757d; }
        .btn-secondary:hover { background: #545b62; }
        .btn-outline { background: transparent; border: 1px solid #007bff; color: #007bff; }
        .btn-outline:hover { background: #007bff; color: white; }
        .btn-sm { padding: 0.25rem 0.5rem; font-size: 0.75rem; }

        .transcript-box, .quote-content { background: #f8f9fa; padding: 1rem; border-radius: 4px; overflow-x: auto; }
        .transcript-box pre, .quote-content pre { white-space: pre-wrap; word-wrap: break-word; font-family: inherit; margin: 0; }

        .pagination { display: flex; justify-content: center; align-items: center; gap: 1rem; margin-top: 1rem; padding-top: 1rem; border-top: 1px solid #eee; }

        .htmx-indicator { display: none; }
        .htmx-request .htmx-indicator { display: inline; }
        .htmx-request.htmx-indicator { display: inline; }

        .alert { padding: 0.75rem; border-radius: 4px; margin-bottom: 1rem; }
        .alert-error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .alert-success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
    </style>`
}

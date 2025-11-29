// Package handler provides HTTP handlers for the application.
// This file contains template utility functions.
package handler

import (
	"time"

	"github.com/jkindrix/quickquote/internal/domain"
)

// getStatusClass returns the CSS class for a call status.
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

// formatTime formats a time for display.
func formatTime(t time.Time) string {
	return t.Format("Jan 2, 2006 3:04 PM")
}

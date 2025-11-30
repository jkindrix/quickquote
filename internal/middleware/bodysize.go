package middleware

import (
	"net/http"
)

// Default body size limits.
const (
	// DefaultMaxBodySize is the default maximum request body size (1MB).
	DefaultMaxBodySize = 1 << 20 // 1MB

	// MaxWebhookBodySize is the maximum size for webhook payloads (10MB).
	MaxWebhookBodySize = 10 << 20 // 10MB

	// MaxFormBodySize is the maximum size for form submissions (100KB).
	MaxFormBodySize = 100 << 10 // 100KB

	// MaxJSONBodySize is the maximum size for JSON API requests (1MB).
	MaxJSONBodySize = 1 << 20 // 1MB
)

// BodySizeLimiter limits the size of request bodies.
// This protects against denial-of-service attacks via large payloads.
func BodySizeLimiter(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip if no body
			if r.Body == nil || r.ContentLength == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Check Content-Length header if present
			if r.ContentLength > maxBytes {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}

			// Wrap body with a size limiter (handles chunked encoding)
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}

// BodySizeLimiterJSON returns a middleware limiting JSON API request bodies.
func BodySizeLimiterJSON() func(http.Handler) http.Handler {
	return BodySizeLimiter(MaxJSONBodySize)
}

// BodySizeLimiterForm returns a middleware limiting form submission bodies.
func BodySizeLimiterForm() func(http.Handler) http.Handler {
	return BodySizeLimiter(MaxFormBodySize)
}

// BodySizeLimiterWebhook returns a middleware limiting webhook payload bodies.
func BodySizeLimiterWebhook() func(http.Handler) http.Handler {
	return BodySizeLimiter(MaxWebhookBodySize)
}

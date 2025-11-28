// Package bland provides a comprehensive client for the Bland AI API.
// It enables full control over voice AI calls, including call initiation,
// voice management, personas, knowledge bases, pathways, memory, and more.
package bland

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/circuitbreaker"
)

const (
	// DefaultBaseURL is the default Bland AI API endpoint.
	DefaultBaseURL = "https://api.bland.ai/v1"

	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second

	// APIVersion is the current API version header value.
	APIVersion = "2024-01-01"
)

// Client is the Bland AI API client.
type Client struct {
	apiKey         string
	baseURL        string
	httpClient     *http.Client
	circuitBreaker *circuitbreaker.CircuitBreaker
	logger         *zap.Logger
}

// Config holds configuration for the Bland API client.
type Config struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
}

// New creates a new Bland AI API client.
func New(cfg *Config, logger *zap.Logger) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	// Configure circuit breaker for Bland API
	cbConfig := &circuitbreaker.Config{
		FailureThreshold:    5,
		SuccessThreshold:    3,
		OpenTimeout:         30 * time.Second,
		HalfOpenMaxRequests: 3,
	}

	return &Client{
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		circuitBreaker: circuitbreaker.New("bland-api", cbConfig, logger),
		logger:         logger,
	}
}

// APIError represents an error response from the Bland API.
type APIError struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("bland API error: %s - %v", e.Message, e.Errors)
	}
	return fmt.Sprintf("bland API error: %s", e.Message)
}

// request performs an HTTP request to the Bland API with circuit breaker protection.
func (c *Client) request(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	return c.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		return c.doRequest(ctx, method, path, body, result)
	})
}

// doRequest performs the actual HTTP request.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c.logger.Debug("bland API request",
		zap.String("method", method),
		zap.String("path", path),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	c.logger.Debug("bland API response",
		zap.String("path", path),
		zap.Int("status", resp.StatusCode),
		zap.Int("body_length", len(respBody)),
	)

	// Check for error responses
	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return &apiErr
	}

	// Parse successful response
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// requestMultipart performs a multipart form request (for file uploads).
func (c *Client) requestMultipart(ctx context.Context, path string, body io.Reader, contentType string, result interface{}) error {
	return c.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		return c.doRequestMultipart(ctx, path, body, contentType, result)
	})
}

// doRequestMultipart performs the actual multipart HTTP request.
func (c *Client) doRequestMultipart(ctx context.Context, path string, body io.Reader, contentType string, result interface{}) error {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return &apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// CircuitBreakerStats returns the current circuit breaker statistics.
func (c *Client) CircuitBreakerStats() circuitbreaker.Stats {
	return c.circuitBreaker.Stats()
}

// IsCircuitOpen returns true if the circuit breaker is open.
func (c *Client) IsCircuitOpen() bool {
	return c.circuitBreaker.IsOpen()
}

// ResetCircuitBreaker resets the circuit breaker to closed state.
func (c *Client) ResetCircuitBreaker() {
	c.circuitBreaker.Reset()
}

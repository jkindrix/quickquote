// Package ai provides AI-powered functionality using Claude.
package ai

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
	"github.com/jkindrix/quickquote/internal/config"
	"github.com/jkindrix/quickquote/internal/domain"
)

// ClaudeClient handles communication with the Anthropic API.
type ClaudeClient struct {
	apiKey         string
	model          string
	httpClient     *http.Client
	circuitBreaker *circuitbreaker.CircuitBreaker
	logger         *zap.Logger
}

// NewClaudeClient creates a new Claude client.
func NewClaudeClient(cfg *config.AnthropicConfig, logger *zap.Logger) *ClaudeClient {
	// Configure circuit breaker for Claude API
	cbConfig := &circuitbreaker.Config{
		FailureThreshold:    5,                // Open after 5 consecutive failures
		SuccessThreshold:    3,                // Close after 3 successes in half-open
		OpenTimeout:         30 * time.Second, // Wait 30s before trying again
		HalfOpenMaxRequests: 3,                // Allow 3 test requests in half-open
	}

	return &ClaudeClient{
		apiKey: cfg.APIKey,
		model:  cfg.Model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		circuitBreaker: circuitbreaker.New("claude-api", cbConfig, logger),
		logger:         logger,
	}
}

// ClaudeRequest represents a request to the Claude API.
type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
}

// ClaudeMessage represents a message in a Claude conversation.
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents a response from the Claude API.
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence,omitempty"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// ClaudeError represents an error response from the Claude API.
type ClaudeError struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateQuote generates a quote summary from a call transcript.
func (c *ClaudeClient) GenerateQuote(ctx context.Context, transcript string, extractedData *domain.ExtractedData) (string, error) {
	prompt := buildQuotePrompt(transcript, extractedData)

	c.logger.Debug("generating quote with Claude",
		zap.Int("transcript_length", len(transcript)),
	)

	response, err := c.sendMessage(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate quote: %w", err)
	}

	return response, nil
}

// CircuitBreakerStats returns the current circuit breaker statistics.
func (c *ClaudeClient) CircuitBreakerStats() circuitbreaker.Stats {
	return c.circuitBreaker.Stats()
}

// IsCircuitOpen returns true if the circuit breaker is open.
func (c *ClaudeClient) IsCircuitOpen() bool {
	return c.circuitBreaker.IsOpen()
}

// ResetCircuitBreaker resets the circuit breaker to closed state.
// Use with caution - typically for administrative purposes.
func (c *ClaudeClient) ResetCircuitBreaker() {
	c.circuitBreaker.Reset()
}

// sendMessage sends a message to Claude and returns the response text.
func (c *ClaudeClient) sendMessage(ctx context.Context, message string) (string, error) {
	var result string

	err := c.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		var execErr error
		result, execErr = c.doSendMessage(ctx, message)
		return execErr
	})

	if err != nil {
		return "", err
	}

	return result, nil
}

// doSendMessage performs the actual HTTP request to Claude API.
func (c *ClaudeClient) doSendMessage(ctx context.Context, message string) (string, error) {
	reqBody := ClaudeRequest{
		Model:     c.model,
		MaxTokens: 2048,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: message,
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ClaudeError
		if err := json.Unmarshal(body, &errResp); err == nil {
			return "", fmt.Errorf("Claude API error: %s - %s", errResp.Error.Type, errResp.Error.Message)
		}
		return "", fmt.Errorf("Claude API error: status %d", resp.StatusCode)
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude")
	}

	c.logger.Debug("quote generated",
		zap.Int("input_tokens", claudeResp.Usage.InputTokens),
		zap.Int("output_tokens", claudeResp.Usage.OutputTokens),
	)

	return claudeResp.Content[0].Text, nil
}

// buildQuotePrompt constructs the prompt for generating a quote.
func buildQuotePrompt(transcript string, extractedData *domain.ExtractedData) string {
	var context string
	if extractedData != nil {
		if extractedData.ProjectType != "" {
			context += fmt.Sprintf("- Project Type: %s\n", extractedData.ProjectType)
		}
		if extractedData.Requirements != "" {
			context += fmt.Sprintf("- Requirements: %s\n", extractedData.Requirements)
		}
		if extractedData.Timeline != "" {
			context += fmt.Sprintf("- Timeline: %s\n", extractedData.Timeline)
		}
		if extractedData.BudgetRange != "" {
			context += fmt.Sprintf("- Budget Range: %s\n", extractedData.BudgetRange)
		}
		if extractedData.ContactPreference != "" {
			context += fmt.Sprintf("- Contact Preference: %s\n", extractedData.ContactPreference)
		}
		if extractedData.CallerName != "" {
			context += fmt.Sprintf("- Caller Name: %s\n", extractedData.CallerName)
		}
	}

	prompt := `You are a professional quote generator for a services business. Based on the following phone call transcript, generate a clear and professional quote summary.

The quote summary should include:
1. **Project Overview** - A brief summary of what the caller is looking for
2. **Key Requirements** - Bullet points of the main requirements discussed
3. **Timeline** - The discussed or recommended timeline
4. **Budget Considerations** - Any budget information mentioned, or a general range if not specified
5. **Next Steps** - Recommended actions for both parties
6. **Notes** - Any important details or concerns from the conversation

Keep the tone professional but friendly. Be specific where possible, but if information is missing, note that it needs to be clarified.
`

	if context != "" {
		prompt += fmt.Sprintf("\n**Extracted Information:**\n%s\n", context)
	}

	prompt += fmt.Sprintf("\n**Call Transcript:**\n%s\n\nPlease generate a professional quote summary:", transcript)

	return prompt
}

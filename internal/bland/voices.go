package bland

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"go.uber.org/zap"
)

// Voice represents a voice available for use in calls.
type Voice struct {
	ID          string   `json:"id"`
	VoiceID     int      `json:"voice_id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Gender      string   `json:"gender,omitempty"`
	Language    string   `json:"language,omitempty"`
	Accent      string   `json:"accent,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	IsPublic    bool     `json:"is_public,omitempty"`
	IsCustom    bool     `json:"is_custom,omitempty"`
	AvgRating   float64  `json:"average_rating,omitempty"`
	TotalRatings int     `json:"total_ratings,omitempty"`
	PreviewURL  string   `json:"preview_url,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
}

// VoiceSettings contains customizable voice parameters.
type VoiceSettings struct {
	Stability       float64 `json:"stability,omitempty"`
	SimilarityBoost float64 `json:"similarity_boost,omitempty"`
	Style           float64 `json:"style,omitempty"`
	SpeakerBoost    bool    `json:"use_speaker_boost,omitempty"`
}

// ListVoicesResponse contains the response from listing voices.
type ListVoicesResponse struct {
	Voices []Voice `json:"voices"`
}

// CloneVoiceRequest contains parameters for cloning a voice.
type CloneVoiceRequest struct {
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	AudioSamples []io.Reader   `json:"-"` // Audio files for cloning
}

// CloneVoiceResponse contains the response from cloning a voice.
type CloneVoiceResponse struct {
	VoiceID int    `json:"voice_id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
}

// GenerateSampleRequest contains parameters for generating a voice sample.
type GenerateSampleRequest struct {
	Text          string         `json:"text"`
	VoiceSettings *VoiceSettings `json:"voice_settings,omitempty"`
	Language      string         `json:"language,omitempty"`
}

// GenerateSampleResponse contains the audio sample data.
type GenerateSampleResponse struct {
	AudioURL string `json:"audio_url,omitempty"`
	Audio    []byte `json:"-"` // Raw audio bytes if returned directly
}

// ListVoices retrieves all available voices.
func (c *Client) ListVoices(ctx context.Context) ([]Voice, error) {
	var resp ListVoicesResponse
	if err := c.request(ctx, "GET", "/voices", nil, &resp); err != nil {
		return nil, err
	}

	return resp.Voices, nil
}

// GetVoice retrieves details for a specific voice.
func (c *Client) GetVoice(ctx context.Context, voiceID string) (*Voice, error) {
	if voiceID == "" {
		return nil, fmt.Errorf("voice_id is required")
	}

	var voice Voice
	if err := c.request(ctx, "GET", "/voices/"+voiceID, nil, &voice); err != nil {
		return nil, err
	}

	return &voice, nil
}

// CloneVoice creates a new voice clone from audio samples.
func (c *Client) CloneVoice(ctx context.Context, req *CloneVoiceRequest) (*CloneVoiceResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(req.AudioSamples) == 0 {
		return nil, fmt.Errorf("at least one audio sample is required")
	}

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add name field
	if err := writer.WriteField("name", req.Name); err != nil {
		return nil, fmt.Errorf("failed to write name field: %w", err)
	}

	// Add description if provided
	if req.Description != "" {
		if err := writer.WriteField("description", req.Description); err != nil {
			return nil, fmt.Errorf("failed to write description field: %w", err)
		}
	}

	// Add audio samples
	for i, sample := range req.AudioSamples {
		part, err := writer.CreateFormFile("audio_samples", fmt.Sprintf("sample_%d.mp3", i))
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}
		if _, err := io.Copy(part, sample); err != nil {
			return nil, fmt.Errorf("failed to copy audio sample: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	var resp CloneVoiceResponse
	if err := c.requestMultipart(ctx, "/voices", &buf, writer.FormDataContentType(), &resp); err != nil {
		return nil, err
	}

	c.logger.Info("voice cloned",
		zap.String("name", req.Name),
		zap.Int("voice_id", resp.VoiceID),
	)

	return &resp, nil
}

// GenerateVoiceSample generates an audio sample using a voice.
func (c *Client) GenerateVoiceSample(ctx context.Context, voiceID string, req *GenerateSampleRequest) (*GenerateSampleResponse, error) {
	if voiceID == "" {
		return nil, fmt.Errorf("voice_id is required")
	}
	if req.Text == "" {
		return nil, fmt.Errorf("text is required")
	}
	if len(req.Text) > 200 {
		return nil, fmt.Errorf("text must be 200 characters or less")
	}

	var resp GenerateSampleResponse
	if err := c.request(ctx, "POST", "/voices/"+voiceID+"/sample", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// DeleteVoice deletes a custom voice clone.
func (c *Client) DeleteVoice(ctx context.Context, voiceID string) error {
	if voiceID == "" {
		return fmt.Errorf("voice_id is required")
	}

	if err := c.request(ctx, "DELETE", "/voices/"+voiceID, nil, nil); err != nil {
		return err
	}

	c.logger.Info("voice deleted", zap.String("voice_id", voiceID))
	return nil
}

// RenameVoice renames a custom voice.
func (c *Client) RenameVoice(ctx context.Context, voiceID, newName string) error {
	if voiceID == "" {
		return fmt.Errorf("voice_id is required")
	}
	if newName == "" {
		return fmt.Errorf("new_name is required")
	}

	req := map[string]string{"name": newName}
	if err := c.request(ctx, "PATCH", "/voices/"+voiceID, req, nil); err != nil {
		return err
	}

	return nil
}

// GetPublicVoices retrieves the curated list of public voices.
func (c *Client) GetPublicVoices(ctx context.Context) ([]Voice, error) {
	voices, err := c.ListVoices(ctx)
	if err != nil {
		return nil, err
	}

	// Filter to only public/curated voices
	var publicVoices []Voice
	for _, v := range voices {
		if v.IsPublic || containsTag(v.Tags, "Bland Curated") {
			publicVoices = append(publicVoices, v)
		}
	}

	return publicVoices, nil
}

// GetCustomVoices retrieves only custom/cloned voices.
func (c *Client) GetCustomVoices(ctx context.Context) ([]Voice, error) {
	voices, err := c.ListVoices(ctx)
	if err != nil {
		return nil, err
	}

	var customVoices []Voice
	for _, v := range voices {
		if v.IsCustom {
			customVoices = append(customVoices, v)
		}
	}

	return customVoices, nil
}

// containsTag checks if a tag exists in a slice.
func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}

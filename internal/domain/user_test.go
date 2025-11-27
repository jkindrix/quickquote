package domain

import (
	"testing"
	"time"
)

func TestNewUser(t *testing.T) {
	email := "test@example.com"
	password := "securepassword123"

	user, err := NewUser(email, password)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	if user.ID.String() == "" {
		t.Error("expected user ID to be generated")
	}
	if user.Email != email {
		t.Errorf("expected Email %s, got %s", email, user.Email)
	}
	if user.PasswordHash == "" {
		t.Error("expected PasswordHash to be set")
	}
	if user.PasswordHash == password {
		t.Error("password should be hashed, not stored in plain text")
	}
	if user.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if user.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestUser_CheckPassword(t *testing.T) {
	email := "test@example.com"
	password := "securepassword123"

	user, err := NewUser(email, password)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{"correct password", "securepassword123", true},
		{"wrong password", "wrongpassword", false},
		{"empty password", "", false},
		{"similar password", "securepassword124", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := user.CheckPassword(tt.password); got != tt.expected {
				t.Errorf("CheckPassword(%q) = %v, expected %v", tt.password, got, tt.expected)
			}
		})
	}
}

func TestNewSession(t *testing.T) {
	user, _ := NewUser("test@example.com", "password")
	token := "test-token-12345"
	duration := 24 * time.Hour

	session := NewSession(user.ID, token, duration)

	if session.ID.String() == "" {
		t.Error("expected session ID to be generated")
	}
	if session.UserID != user.ID {
		t.Errorf("expected UserID %s, got %s", user.ID, session.UserID)
	}
	if session.Token != token {
		t.Errorf("expected Token %s, got %s", token, session.Token)
	}
	if session.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if session.ExpiresAt.IsZero() {
		t.Error("expected ExpiresAt to be set")
	}
	if session.LastActiveAt.IsZero() {
		t.Error("expected LastActiveAt to be set")
	}

	// Check that expiration is approximately correct (within 1 second)
	expectedExpiry := session.CreatedAt.Add(duration)
	if session.ExpiresAt.Sub(expectedExpiry) > time.Second {
		t.Errorf("ExpiresAt not set correctly: expected ~%v, got %v", expectedExpiry, session.ExpiresAt)
	}
}

func TestNewSessionWithContext(t *testing.T) {
	user, _ := NewUser("test@example.com", "password")
	token := "test-token-12345"
	duration := 24 * time.Hour
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0"

	session := NewSessionWithContext(user.ID, token, duration, ipAddress, userAgent)

	if session.IPAddress != ipAddress {
		t.Errorf("expected IPAddress %s, got %s", ipAddress, session.IPAddress)
	}
	if session.UserAgent != userAgent {
		t.Errorf("expected UserAgent %s, got %s", userAgent, session.UserAgent)
	}
}

func TestSession_IsExpired(t *testing.T) {
	user, _ := NewUser("test@example.com", "password")

	tests := []struct {
		name     string
		duration time.Duration
		wait     time.Duration
		expected bool
	}{
		{"not expired", 1 * time.Hour, 0, false},
		{"expired", -1 * time.Hour, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewSession(user.ID, "token", tt.duration)
			if got := session.IsExpired(); got != tt.expected {
				t.Errorf("IsExpired() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestSession_Touch(t *testing.T) {
	user, _ := NewUser("test@example.com", "password")
	session := NewSession(user.ID, "token", 1*time.Hour)

	originalLastActive := session.LastActiveAt
	time.Sleep(10 * time.Millisecond)
	session.Touch()

	if !session.LastActiveAt.After(originalLastActive) {
		t.Error("Touch() should update LastActiveAt to a later time")
	}
}

func TestSession_ShouldRotate(t *testing.T) {
	user, _ := NewUser("test@example.com", "password")

	// New session should not need rotation
	session := NewSession(user.ID, "token", 1*time.Hour)
	if session.ShouldRotate() {
		t.Error("new session should not need rotation")
	}

	// Session with old LastActiveAt should need rotation
	session.LastActiveAt = time.Now().UTC().Add(-20 * time.Minute)
	if !session.ShouldRotate() {
		t.Error("session with 20 minute old LastActiveAt should need rotation")
	}
}

func TestSession_Refresh(t *testing.T) {
	user, _ := NewUser("test@example.com", "password")
	session := NewSession(user.ID, "token", 1*time.Hour)

	originalExpiry := session.ExpiresAt
	time.Sleep(10 * time.Millisecond)
	session.Refresh(2 * time.Hour)

	if !session.ExpiresAt.After(originalExpiry) {
		t.Error("Refresh() should extend ExpiresAt")
	}

	// Check expiry is now ~2 hours from now
	expectedExpiry := time.Now().UTC().Add(2 * time.Hour)
	diff := session.ExpiresAt.Sub(expectedExpiry)
	if diff > time.Second || diff < -time.Second {
		t.Errorf("Refresh() ExpiresAt should be ~2 hours from now, diff: %v", diff)
	}
}

func TestSession_RotateToken(t *testing.T) {
	user, _ := NewUser("test@example.com", "password")
	session := NewSession(user.ID, "original-token", 1*time.Hour)

	originalToken := session.Token
	newToken := "new-rotated-token"

	session.RotateToken(newToken)

	// New token should be set
	if session.Token != newToken {
		t.Errorf("expected Token %q, got %q", newToken, session.Token)
	}

	// Previous token should be stored
	if session.PreviousToken == nil {
		t.Fatal("expected PreviousToken to be set")
	}
	if *session.PreviousToken != originalToken {
		t.Errorf("expected PreviousToken %q, got %q", originalToken, *session.PreviousToken)
	}

	// RotatedAt should be set
	if session.RotatedAt == nil {
		t.Fatal("expected RotatedAt to be set")
	}
	if time.Since(*session.RotatedAt) > time.Second {
		t.Error("RotatedAt should be very recent")
	}
}

func TestSession_MatchesToken(t *testing.T) {
	user, _ := NewUser("test@example.com", "password")
	session := NewSession(user.ID, "original-token", 1*time.Hour)

	// Should match current token
	if !session.MatchesToken("original-token") {
		t.Error("should match current token")
	}

	// Should not match random token
	if session.MatchesToken("wrong-token") {
		t.Error("should not match wrong token")
	}

	// Rotate token
	session.RotateToken("new-token")

	// Should match new token
	if !session.MatchesToken("new-token") {
		t.Error("should match new token after rotation")
	}

	// Should match old token within grace period
	if !session.MatchesToken("original-token") {
		t.Error("should match old token within grace period")
	}

	// Should not match random token
	if session.MatchesToken("completely-wrong") {
		t.Error("should not match random token")
	}
}

func TestSession_IsWithinGracePeriod(t *testing.T) {
	user, _ := NewUser("test@example.com", "password")
	session := NewSession(user.ID, "token", 1*time.Hour)

	// No rotation - not in grace period
	if session.IsWithinGracePeriod() {
		t.Error("should not be in grace period before rotation")
	}

	// After rotation - should be in grace period
	session.RotateToken("new-token")
	if !session.IsWithinGracePeriod() {
		t.Error("should be in grace period after rotation")
	}
}

func TestSession_InvalidatePreviousToken(t *testing.T) {
	user, _ := NewUser("test@example.com", "password")
	session := NewSession(user.ID, "token", 1*time.Hour)

	session.RotateToken("new-token")
	if session.PreviousToken == nil {
		t.Fatal("expected PreviousToken to be set after rotation")
	}

	session.InvalidatePreviousToken()

	if session.PreviousToken != nil {
		t.Error("expected PreviousToken to be nil after invalidation")
	}
	if session.RotatedAt != nil {
		t.Error("expected RotatedAt to be nil after invalidation")
	}

	// Old token should no longer match
	if session.MatchesToken("token") {
		t.Error("old token should not match after invalidation")
	}
}

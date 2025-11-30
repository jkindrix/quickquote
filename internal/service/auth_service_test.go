package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/domain"
)

func newTestAuthService() (*AuthService, *MockUserRepository, *MockSessionRepository) {
	logger := zap.NewNop()
	mockUserRepo := NewMockUserRepository()
	mockSessionRepo := NewMockSessionRepository()
	service := NewAuthService(mockUserRepo, mockSessionRepo, 24*time.Hour, logger, nil)
	return service, mockUserRepo, mockSessionRepo
}

func TestAuthService_Login_Success(t *testing.T) {
	service, mockUserRepo, mockSessionRepo := newTestAuthService()
	ctx := context.Background()

	// Create a user
	password := "securepassword123"
	user, _ := domain.NewUser("test@example.com", password)
	mockUserRepo.Create(ctx, user)

	session, err := service.Login(ctx, "test@example.com", password)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if session == nil {
		t.Fatal("Login() returned nil session")
	}
	if session.UserID != user.ID {
		t.Errorf("expected UserID %s, got %s", user.ID, session.UserID)
	}
	if session.Token == "" {
		t.Error("expected Token to be set")
	}
	if mockSessionRepo.CreateCalls != 1 {
		t.Errorf("expected 1 session Create call, got %d", mockSessionRepo.CreateCalls)
	}
}

func TestAuthService_Login_InvalidEmail(t *testing.T) {
	service, _, _ := newTestAuthService()
	ctx := context.Background()

	_, err := service.Login(ctx, "nonexistent@example.com", "password")
	if err == nil {
		t.Error("expected error for non-existent user, got nil")
	}
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	service, mockUserRepo, _ := newTestAuthService()
	ctx := context.Background()

	// Create a user
	user, _ := domain.NewUser("test@example.com", "correctpassword")
	mockUserRepo.Create(ctx, user)

	_, err := service.Login(ctx, "test@example.com", "wrongpassword")
	if err == nil {
		t.Error("expected error for wrong password, got nil")
	}
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_EmptyPassword(t *testing.T) {
	service, mockUserRepo, _ := newTestAuthService()
	ctx := context.Background()

	user, _ := domain.NewUser("test@example.com", "password123")
	mockUserRepo.Create(ctx, user)

	_, err := service.Login(ctx, "test@example.com", "")
	if err == nil {
		t.Error("expected error for empty password, got nil")
	}
}

func TestAuthService_Logout_Success(t *testing.T) {
	service, mockUserRepo, mockSessionRepo := newTestAuthService()
	ctx := context.Background()

	// Create user and session
	user, _ := domain.NewUser("test@example.com", "password")
	mockUserRepo.Create(ctx, user)
	session, _ := service.Login(ctx, "test@example.com", "password")

	err := service.Logout(ctx, session.Token)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if mockSessionRepo.DeleteCalls != 1 {
		t.Errorf("expected 1 Delete call, got %d", mockSessionRepo.DeleteCalls)
	}
}

func TestAuthService_ValidateSession_Success(t *testing.T) {
	service, mockUserRepo, _ := newTestAuthService()
	ctx := context.Background()

	// Create user and session
	user, _ := domain.NewUser("test@example.com", "password")
	mockUserRepo.Create(ctx, user)
	session, _ := service.Login(ctx, "test@example.com", "password")

	validatedUser, err := service.ValidateSession(ctx, session.Token)
	if err != nil {
		t.Fatalf("ValidateSession() error = %v", err)
	}

	if validatedUser.ID != user.ID {
		t.Errorf("expected user ID %s, got %s", user.ID, validatedUser.ID)
	}
}

func TestAuthService_ValidateSession_InvalidToken(t *testing.T) {
	service, _, _ := newTestAuthService()
	ctx := context.Background()

	_, err := service.ValidateSession(ctx, "invalid-token")
	if err == nil {
		t.Error("expected error for invalid token, got nil")
	}
	if !errors.Is(err, ErrSessionExpired) {
		t.Errorf("expected ErrSessionExpired, got %v", err)
	}
}

func TestAuthService_ValidateSession_ExpiredSession(t *testing.T) {
	// Create service with very short session duration for testing
	logger := zap.NewNop()
	mockUserRepo := NewMockUserRepository()
	mockSessionRepo := NewMockSessionRepository()
	service := NewAuthService(mockUserRepo, mockSessionRepo, -1*time.Hour, logger, nil) // Already expired

	ctx := context.Background()

	user, _ := domain.NewUser("test@example.com", "password")
	mockUserRepo.Create(ctx, user)

	session, _ := service.Login(ctx, "test@example.com", "password")

	_, err := service.ValidateSession(ctx, session.Token)
	if err == nil {
		t.Error("expected error for expired session, got nil")
	}
}

func TestAuthService_CreateUser_Success(t *testing.T) {
	service, mockUserRepo, _ := newTestAuthService()
	ctx := context.Background()

	user, err := service.CreateUser(ctx, "newuser@example.com", "password123")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if user == nil {
		t.Fatal("CreateUser() returned nil user")
	}
	if user.Email != "newuser@example.com" {
		t.Errorf("expected email newuser@example.com, got %s", user.Email)
	}
	if mockUserRepo.CreateCalls != 1 {
		t.Errorf("expected 1 Create call, got %d", mockUserRepo.CreateCalls)
	}
}

func TestAuthService_CreateUser_DuplicateEmail(t *testing.T) {
	service, mockUserRepo, _ := newTestAuthService()
	ctx := context.Background()

	// Create first user
	existingUser, _ := domain.NewUser("existing@example.com", "password")
	mockUserRepo.Create(ctx, existingUser)

	// Try to create user with same email
	_, err := service.CreateUser(ctx, "existing@example.com", "password123")
	if err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
}

func TestAuthService_CleanupExpiredSessions(t *testing.T) {
	service, _, mockSessionRepo := newTestAuthService()
	ctx := context.Background()

	err := service.CleanupExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredSessions() error = %v", err)
	}

	if mockSessionRepo.DeleteExpiredCalls != 1 {
		t.Errorf("expected 1 DeleteExpired call, got %d", mockSessionRepo.DeleteExpiredCalls)
	}
}

func TestAuthError_Error(t *testing.T) {
	err := &AuthError{Message: "test error message"}
	if err.Error() != "test error message" {
		t.Errorf("expected 'test error message', got %q", err.Error())
	}
}

func TestGenerateToken(t *testing.T) {
	token1, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}

	if len(token1) != 64 { // 32 bytes = 64 hex characters
		t.Errorf("expected 64 character token, got %d", len(token1))
	}

	token2, _ := generateToken()
	if token1 == token2 {
		t.Error("expected different tokens, got same")
	}
}

package repository

import (
	"errors"
	"testing"
)

func TestErrNotFound(t *testing.T) {
	// Verify ErrNotFound is properly defined
	if ErrNotFound == nil {
		t.Fatal("expected ErrNotFound to be defined")
	}

	if ErrNotFound.Error() != "record not found" {
		t.Errorf("expected 'record not found', got %q", ErrNotFound.Error())
	}
}

func TestErrNotFound_ErrorsIs(t *testing.T) {
	// Verify errors.Is works with ErrNotFound
	wrappedErr := errors.New("wrapper: " + ErrNotFound.Error())

	// Direct comparison should work
	if !errors.Is(ErrNotFound, ErrNotFound) {
		t.Error("errors.Is should return true for same error")
	}

	// Wrapped error should not match (unless using %w)
	if errors.Is(wrappedErr, ErrNotFound) {
		t.Error("wrapped error without %w should not match")
	}
}

func TestNewCallRepository(t *testing.T) {
	// Test that NewCallRepository creates a repository with nil pool
	// (just testing the constructor, not database operations)
	repo := NewCallRepository(nil)

	if repo == nil {
		t.Fatal("expected non-nil repository")
	}

	if repo.pool != nil {
		t.Error("expected nil pool")
	}
}

func TestNewUserRepository(t *testing.T) {
	// Test that NewUserRepository creates a repository
	repo := NewUserRepository(nil)

	if repo == nil {
		t.Fatal("expected non-nil repository")
	}

	if repo.pool != nil {
		t.Error("expected nil pool")
	}
}

func TestNewQuoteJobRepository(t *testing.T) {
	// Test that NewQuoteJobRepository creates a repository
	repo := NewQuoteJobRepository(nil)

	if repo == nil {
		t.Fatal("expected non-nil repository")
	}

	if repo.pool != nil {
		t.Error("expected nil pool")
	}
}

func TestNewCSRFRepository(t *testing.T) {
	// Test that NewCSRFRepository creates a repository
	repo := NewCSRFRepository(nil)

	if repo == nil {
		t.Fatal("expected non-nil repository")
	}

	if repo.pool != nil {
		t.Error("expected nil pool")
	}
}

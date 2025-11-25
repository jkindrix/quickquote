// Package main is the entry point for the QuickQuote server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/jkindrix/quickquote/internal/ai"
	"github.com/jkindrix/quickquote/internal/config"
	"github.com/jkindrix/quickquote/internal/database"
	"github.com/jkindrix/quickquote/internal/handler"
	"github.com/jkindrix/quickquote/internal/middleware"
	"github.com/jkindrix/quickquote/internal/repository"
	"github.com/jkindrix/quickquote/internal/service"
)

func main() {
	// Initialize logger
	logger, err := initLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load configuration", zap.Error(err))
	}

	logger.Info("starting QuickQuote server",
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
		zap.String("env", cfg.Server.Environment),
	)

	// Initialize database
	ctx := context.Background()
	db, err := database.New(ctx, &cfg.Database, logger)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize repositories
	callRepo := repository.NewCallRepository(db.Pool)
	userRepo := repository.NewUserRepository(db.Pool)
	sessionRepo := repository.NewSessionRepository(db.Pool)

	// Initialize AI client
	claudeClient := ai.NewClaudeClient(&cfg.Anthropic, logger)

	// Initialize services
	callService := service.NewCallService(callRepo, claudeClient, logger)
	authService := service.NewAuthService(
		userRepo,
		sessionRepo,
		cfg.Auth.SessionDuration,
		logger,
	)

	// Initialize handlers
	h := handler.New(callService, authService, logger)
	h.SetHealthChecker(db)

	// Initialize router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.Recovery(logger))
	r.Use(chimiddleware.Compress(5))

	// Serve static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Register routes
	h.RegisterRoutes(r)

	// Create server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("server listening", zap.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Start session cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			if err := authService.CleanupExpiredSessions(ctx); err != nil {
				logger.Error("failed to cleanup expired sessions", zap.Error(err))
			} else {
				logger.Debug("cleaned up expired sessions")
			}
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server stopped")
}

// initLogger initializes the zap logger based on environment.
func initLogger() (*zap.Logger, error) {
	env := os.Getenv("APP_ENV")
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

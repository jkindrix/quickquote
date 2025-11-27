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
	"github.com/jkindrix/quickquote/internal/ratelimit"
	"github.com/jkindrix/quickquote/internal/repository"
	"github.com/jkindrix/quickquote/internal/service"
	"github.com/jkindrix/quickquote/internal/shutdown"
	"github.com/jkindrix/quickquote/internal/voiceprovider"
	"github.com/jkindrix/quickquote/internal/voiceprovider/bland"
	"github.com/jkindrix/quickquote/internal/voiceprovider/retell"
	"github.com/jkindrix/quickquote/internal/voiceprovider/vapi"
)

func main() {
	// Initialize logger
	logger, err := initLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

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
	// Note: db.Close() is handled by shutdown coordinator

	// Initialize repositories
	callRepo := repository.NewCallRepository(db.Pool)
	userRepo := repository.NewUserRepository(db.Pool)
	sessionRepo := repository.NewSessionRepository(db.Pool)
	quoteJobRepo := repository.NewQuoteJobRepository(db.Pool)
	csrfRepo := repository.NewCSRFRepository(db.Pool)

	// Initialize AI client
	claudeClient := ai.NewClaudeClient(&cfg.Anthropic, logger)

	// Initialize voice provider registry
	providerRegistry := initVoiceProviders(cfg, logger)

	// Initialize quote rate limiter for cost control
	quoteLimiterConfig := ratelimit.DefaultQuoteLimiterConfig()
	quoteLimiter := ratelimit.NewQuoteLimiter(quoteLimiterConfig, logger)
	logger.Info("initialized quote rate limiter",
		zap.Int("max_per_minute", quoteLimiterConfig.MaxRequestsPerMinute),
		zap.Int("max_per_hour", quoteLimiterConfig.MaxRequestsPerHour),
		zap.Int("max_per_day", quoteLimiterConfig.MaxRequestsPerDay),
		zap.Int("max_concurrent", quoteLimiterConfig.MaxConcurrent),
	)

	// Initialize quote job processor
	jobProcessorConfig := service.DefaultQuoteJobProcessorConfig()
	jobProcessor := service.NewQuoteJobProcessor(
		quoteJobRepo,
		callRepo,
		claudeClient,
		quoteLimiter,
		logger,
		jobProcessorConfig,
	)

	// Initialize services
	callService := service.NewCallService(callRepo, claudeClient, jobProcessor, logger)
	authService := service.NewAuthService(
		userRepo,
		sessionRepo,
		cfg.Auth.SessionDuration,
		logger,
	)

	// Initialize rate limiters
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit.Requests, cfg.RateLimit.Window, logger)
	loginRateLimiter := middleware.NewLoginRateLimiter(logger)

	// Initialize CSRF protection with database persistence
	csrfProtection := middleware.NewCSRFProtectionWithRepo(csrfRepo, logger)
	logger.Info("initialized CSRF protection with database persistence")

	// Initialize template engine
	templateEngine, err := handler.NewTemplateEngine("web/templates", logger)
	if err != nil {
		logger.Warn("failed to initialize template engine, using inline templates", zap.Error(err))
	}

	// Initialize handlers
	h := handler.New(callService, authService, logger)
	h.SetHealthChecker(db)
	h.SetAIHealthChecker(claudeClient)
	h.SetLoginRateLimiter(loginRateLimiter)
	h.SetCSRFProtection(csrfProtection)
	h.SetProviderRegistry(providerRegistry)
	if templateEngine != nil {
		h.SetTemplateEngine(templateEngine)
	}

	// Initialize request correlation
	correlation := middleware.NewRequestCorrelation(logger)

	// Initialize router
	r := chi.NewRouter()

	// Global middleware (order matters)
	r.Use(correlation.Middleware) // First: add correlation IDs
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.Recovery(logger))
	r.Use(chimiddleware.Compress(5))
	r.Use(middleware.RateLimit(rateLimiter))

	// CSRF protection (skip webhook endpoints)
	r.Use(csrfProtection.SkipPath("/webhook/bland", "/health", "/ready", "/live"))

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

	// Start quote job processor
	if err := jobProcessor.Start(ctx); err != nil {
		logger.Fatal("failed to start job processor", zap.Error(err))
	}

	// Start server in goroutine
	go func() {
		logger.Info("server listening", zap.String("addr", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Initialize shutdown coordinator
	shutdownCoord := shutdown.NewCoordinator(&shutdown.Config{
		Timeout: 30 * time.Second,
	}, logger)

	// Start session cleanup goroutine (respects shutdown signal)
	cleanupDone := make(chan struct{})
	go func() {
		defer close(cleanupDone)
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := authService.CleanupExpiredSessions(ctx); err != nil {
					logger.Error("failed to cleanup expired sessions", zap.Error(err))
				} else {
					logger.Debug("cleaned up expired sessions")
				}
			case <-shutdownCoord.ShutdownCh():
				logger.Debug("session cleanup goroutine stopping")
				return
			}
		}
	}()

	// Register services for graceful shutdown (in order of shutdown phases)
	// Phase 1 (PreDrain): Stop accepting new work - handled by signal receipt
	// Phase 2 (Drain): Let in-flight requests complete
	shutdownCoord.RegisterFunc(shutdown.PhaseDrain, "http-server", func(ctx context.Context) error {
		return server.Shutdown(ctx)
	})

	// Phase 3 (Shutdown): Stop background workers
	shutdownCoord.RegisterFunc(shutdown.PhaseShutdown, "job-processor", func(ctx context.Context) error {
		return jobProcessor.Stop(ctx)
	})

	// Phase 4 (Cleanup): Close connections and flush buffers
	shutdownCoord.RegisterFunc(shutdown.PhaseCleanup, "session-cleanup", func(ctx context.Context) error {
		// Wait for session cleanup goroutine to finish
		select {
		case <-cleanupDone:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	shutdownCoord.RegisterFunc(shutdown.PhaseCleanup, "database", func(ctx context.Context) error {
		db.Close()
		return nil
	})

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("received shutdown signal")

	// Execute graceful shutdown
	if err := shutdownCoord.Shutdown(ctx); err != nil {
		logger.Error("shutdown completed with errors", zap.Error(err))
	}
}

// initLogger initializes the zap logger based on environment.
func initLogger() (*zap.Logger, error) {
	env := os.Getenv("APP_ENV")
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

// initVoiceProviders initializes and registers all configured voice providers.
func initVoiceProviders(cfg *config.Config, logger *zap.Logger) *voiceprovider.Registry {
	registry := voiceprovider.NewRegistry(logger)

	// Register Bland provider if enabled
	if cfg.VoiceProvider.Bland.Enabled || cfg.Bland.APIKey != "" {
		blandCfg := &bland.Config{
			APIKey:        cfg.VoiceProvider.Bland.APIKey,
			WebhookSecret: cfg.VoiceProvider.Bland.WebhookSecret,
			APIURL:        cfg.VoiceProvider.Bland.APIURL,
		}
		// Fallback to legacy config
		if blandCfg.APIKey == "" {
			blandCfg.APIKey = cfg.Bland.APIKey
			blandCfg.WebhookSecret = cfg.Bland.WebhookSecret
			blandCfg.APIURL = cfg.Bland.APIURL
		}
		registry.Register(bland.New(blandCfg, logger))
		logger.Info("registered Bland voice provider")
	}

	// Register Vapi provider if enabled
	if cfg.VoiceProvider.Vapi.Enabled && cfg.VoiceProvider.Vapi.APIKey != "" {
		vapiCfg := &vapi.Config{
			APIKey:        cfg.VoiceProvider.Vapi.APIKey,
			WebhookSecret: cfg.VoiceProvider.Vapi.WebhookSecret,
			APIURL:        cfg.VoiceProvider.Vapi.APIURL,
		}
		registry.Register(vapi.New(vapiCfg, logger))
		logger.Info("registered Vapi voice provider")
	}

	// Register Retell provider if enabled
	if cfg.VoiceProvider.Retell.Enabled && cfg.VoiceProvider.Retell.APIKey != "" {
		retellCfg := &retell.Config{
			APIKey:        cfg.VoiceProvider.Retell.APIKey,
			WebhookSecret: cfg.VoiceProvider.Retell.WebhookSecret,
			APIURL:        cfg.VoiceProvider.Retell.APIURL,
		}
		registry.Register(retell.New(retellCfg, logger))
		logger.Info("registered Retell voice provider")
	}

	// Set primary provider
	primary := cfg.VoiceProvider.Primary
	if primary == "" {
		primary = "bland" // Default to bland for backward compatibility
	}
	if err := registry.SetPrimary(voiceprovider.ProviderType(primary)); err != nil {
		logger.Warn("could not set primary provider, using first registered", zap.Error(err))
	}

	return registry
}

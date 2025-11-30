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
	"github.com/jkindrix/quickquote/internal/audit"
	"github.com/jkindrix/quickquote/internal/bland"
	"github.com/jkindrix/quickquote/internal/config"
	"github.com/jkindrix/quickquote/internal/database"
	"github.com/jkindrix/quickquote/internal/handler"
	"github.com/jkindrix/quickquote/internal/metrics"
	"github.com/jkindrix/quickquote/internal/middleware"
	"github.com/jkindrix/quickquote/internal/ratelimit"
	"github.com/jkindrix/quickquote/internal/repository"
	"github.com/jkindrix/quickquote/internal/service"
	"github.com/jkindrix/quickquote/internal/shutdown"
	"github.com/jkindrix/quickquote/internal/voiceprovider"
	blandprovider "github.com/jkindrix/quickquote/internal/voiceprovider/bland"
	"github.com/jkindrix/quickquote/internal/voiceprovider/retell"
	"github.com/jkindrix/quickquote/internal/voiceprovider/vapi"
)

func main() {
	// Initialize logger with atomic level for runtime adjustment
	logger, logLevel, err := initLogger()
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

	appMetrics := metrics.NewMetrics()

	logger.Info("starting QuickQuote server",
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
		zap.String("env", cfg.Server.Environment),
	)

	// Initialize database with query logging
	ctx := context.Background()
	var queryLoggerCfg *database.QueryLoggerConfig
	if cfg.Database.SlowQueryThreshold > 0 {
		queryLoggerCfg = &database.QueryLoggerConfig{
			SlowQueryThreshold:     cfg.Database.SlowQueryThreshold,
			VerySlowQueryThreshold: cfg.Database.VerySlowQueryThreshold,
			LogAllQueries:          cfg.Database.LogAllQueries,
			SampleRate:             0.1, // Sample 10% of queries when logging all
		}
	}
	db, err := database.NewWithQueryLogger(ctx, &cfg.Database, queryLoggerCfg, logger)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	// Note: db.Close() is handled by shutdown coordinator

	// Run database migrations automatically on startup
	migrator := database.NewMigrator(db.Pool, logger)
	if err := migrator.MigrateFromDir(ctx, "migrations"); err != nil {
		logger.Fatal("failed to run database migrations", zap.Error(err))
	}
	logger.Info("database migrations completed successfully")

	// Initialize repositories (needed for user seeding)
	userRepo := repository.NewUserRepository(db.Pool)
	sessionRepo := repository.NewSessionRepository(db.Pool)

	// Initialize auth service early for admin user seeding
	authService := service.NewAuthService(
		userRepo,
		sessionRepo,
		cfg.Auth.SessionDuration,
		logger,
		appMetrics,
	)

	// Seed initial admin user if no users exist (enables zero-config deployment)
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminEmail != "" && adminPassword != "" {
		created, err := authService.EnsureAdminUser(ctx, adminEmail, adminPassword)
		if err != nil {
			logger.Warn("failed to ensure admin user", zap.Error(err))
		} else if created {
			logger.Info("initial admin user created from environment variables")
		}
	} else {
		logger.Debug("ADMIN_EMAIL/ADMIN_PASSWORD not set, skipping admin user seed")
	}

	// Initialize remaining repositories
	callRepo := repository.NewCallRepository(db.Pool)
	quoteJobRepo := repository.NewQuoteJobRepository(db.Pool)
	csrfRepo := repository.NewCSRFRepository(db.Pool)
	promptRepo := repository.NewPromptRepository(db.Pool)
	settingsRepo := repository.NewSettingsRepository(db.Pool)
	idempotencyRepo := repository.NewIdempotencyRepository(db.Pool, logger)

	// Initialize Bland entity repositories (for local caching)
	knowledgeBaseRepo := repository.NewKnowledgeBaseRepository(db.Pool)
	pathwayRepo := repository.NewPathwayRepository(db.Pool)
	personaRepo := repository.NewPersonaRepository(db.Pool)
	_ = knowledgeBaseRepo // Available for future use
	_ = pathwayRepo       // Available for future use
	_ = personaRepo       // Available for future use

	// Initialize AI client
	claudeClient := ai.NewClaudeClient(&cfg.Anthropic, logger)

	// Initialize Bland API client (for full API capabilities)
	blandAPIKey := cfg.VoiceProvider.Bland.APIKey
	if blandAPIKey == "" {
		blandAPIKey = cfg.Bland.APIKey
	}
	blandClient := bland.New(&bland.Config{
		APIKey: blandAPIKey,
	}, logger)
	logger.Info("initialized Bland API client")

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
	callService := service.NewCallService(callRepo, claudeClient, jobProcessor, quoteLimiter, logger, appMetrics)

	// Initialize settings service (needed by BlandService)
	settingsService := service.NewSettingsService(settingsRepo, logger)
	logger.Info("initialized settings service")

	// Build webhook URL for Bland callbacks
	// In production, this should be configured to your public URL
	webhookURL := fmt.Sprintf("http://%s:%d/webhook/bland", cfg.Server.Host, cfg.Server.Port)
	if os.Getenv("WEBHOOK_BASE_URL") != "" {
		webhookURL = os.Getenv("WEBHOOK_BASE_URL") + "/webhook/bland"
	}

	// Initialize Bland service (for full API access)
	blandService := service.NewBlandService(
		blandClient,
		callRepo,
		promptRepo,
		settingsService,
		webhookURL,
		idempotencyRepo,
		logger,
	)
	logger.Info("initialized Bland service", zap.String("webhook_url", webhookURL))

	// Initialize prompt service
	promptService := service.NewPromptService(promptRepo, logger)

	// Initialize audit logger
	auditLogger := audit.NewLogger(logger)
	logger.Info("initialized audit logger")

	// Initialize rate limiters
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit.Requests, cfg.RateLimit.Window, logger)
	loginRateLimiter := middleware.NewLoginRateLimiter(logger)
	userRateLimitRepo := repository.NewUserRateLimitRepository(db.Pool, logger)
	userRateLimiter := ratelimit.NewUserRateLimiter(ratelimit.DefaultUserRateLimitConfig(), userRateLimitRepo, logger)

	// Initialize CSRF protection with database persistence
	csrfProtection := middleware.NewCSRFProtectionWithRepo(csrfRepo, logger)
	logger.Info("initialized CSRF protection with database persistence")

	// Initialize template engine
	templateEngine, err := handler.NewTemplateEngine("web/templates", logger)
	if err != nil {
		logger.Warn("failed to initialize template engine, using inline templates", zap.Error(err))
	}

	assetVersion := os.Getenv("ASSET_VERSION")
	if assetVersion == "" {
		assetVersion = Version
	}
	if assetVersion == "" || assetVersion == "dev" {
		assetVersion = fmt.Sprintf("%s-%d", Version, time.Now().Unix())
	}

	// Initialize focused handlers with constructor injection
	baseHandlerCfg := handler.BaseHandlerConfig{
		TemplateEngine: templateEngine,
		CSRFProtection: csrfProtection,
		Logger:         logger,
		AssetVersion:   assetVersion,
	}

	// Auth handler for login/logout/session management
	authHandler := handler.NewAuthHandler(handler.AuthHandlerConfig{
		Base:             baseHandlerCfg,
		AuthService:      authService,
		LoginRateLimiter: loginRateLimiter,
		Metrics:          appMetrics,
	})

	// Health handler for health check endpoints
	healthHandler := handler.NewHealthHandler(handler.HealthHandlerConfig{
		HealthChecker:    db,
		AIHealthChecker:  claudeClient,
		ProviderRegistry: providerRegistry,
		Logger:           logger,
	})

	// Webhook handler for voice provider callbacks
	webhookHandler := handler.NewWebhookHandler(handler.WebhookHandlerConfig{
		CallService:      callService,
		ProviderRegistry: providerRegistry,
		Logger:           logger,
		Metrics:          appMetrics,
	})

	// Calls handler for dashboard and call management
	callsHandler := handler.NewCallsHandler(handler.CallsHandlerConfig{
		Base:        baseHandlerCfg,
		CallService: callService,
	})

	// Admin handler for settings, voices, usage, etc.
	adminHandler := handler.NewAdminHandler(handler.AdminHandlerConfig{
		Base:            baseHandlerCfg,
		BlandService:    blandService,
		PromptService:   promptService,
		SettingsService: settingsService,
		QuoteJobRepo:    quoteJobRepo,
	})

	// Initialize API handlers
	callAPIHandler := handler.NewCallAPIHandler(blandService, auditLogger, logger)
	promptAPIHandler := handler.NewPromptAPIHandler(promptService, auditLogger, logger)
	promptAPIHandler.SetBlandService(blandService) // Enable apply-to-inbound functionality
	blandAPIHandler := handler.NewBlandAPIHandler(blandService, logger)

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
	r.Use(middleware.RateLimit(rateLimiter, appMetrics))
	r.Use(appMetrics.Middleware)

	// CSRF protection (skip webhook endpoints and API routes)
	r.Use(csrfProtection.SkipPath("/webhook/bland", "/health", "/ready", "/live", "/metrics"))

	// Serve static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	r.Handle("/metrics", appMetrics.Handler())

	// Register public routes (auth handlers)
	authHandler.RegisterRoutes(r)

	// Register webhook routes (no auth required)
	webhookHandler.RegisterRoutes(r)

	// Register health check routes
	healthHandler.RegisterRoutes(r)

	// Initialize log level handler for runtime adjustment
	logLevelHandler := handler.NewLogLevelHandler(logLevel, logger)

	// Register protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(authHandler.Middleware)
		r.Use(middleware.UserRateLimit(userRateLimiter, logger, appMetrics))

		// Dashboard and calls
		callsHandler.RegisterRoutes(r)

		// Admin pages (settings, phone numbers, voices, usage, knowledge bases, presets)
		adminHandler.RegisterRoutes(r)

		// Admin API for runtime log level adjustment
		r.Handle("/admin/log-level", logLevelHandler)
	})

	// Authenticated API routes (JSON responses, no redirects)
	r.Group(func(r chi.Router) {
		r.Use(authHandler.APIAuthMiddleware)
		r.Use(middleware.UserRateLimit(userRateLimiter, logger, appMetrics))

		apiRouter := chi.NewRouter()
		apiRouter.Use(middleware.BodySizeLimiterJSON())
		callAPIHandler.RegisterRoutes(apiRouter)
		promptAPIHandler.RegisterRoutes(apiRouter)
		blandAPIHandler.RegisterRoutes(apiRouter)
		r.Mount("/api/v1", apiRouter)
	})

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

	var metricsStop chan struct{}
	if appMetrics != nil {
		metricsStop = make(chan struct{})
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					stats := db.Stats()
					if stats != nil {
						appMetrics.UpdateDBConnections(int(stats.TotalConns()), int(stats.AcquiredConns()))
					}
				case <-metricsStop:
					return
				}
			}
		}()
		shutdownCoord.RegisterFunc(shutdown.PhaseCleanup, "metrics-updater", func(ctx context.Context) error {
			close(metricsStop)
			return nil
		})
	}

	rateLimitCleanupStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = userRateLimitRepo.ResetExpiredWindows(cleanupCtx)
				cancel()
			case <-rateLimitCleanupStop:
				return
			}
		}
	}()
	shutdownCoord.RegisterFunc(shutdown.PhaseCleanup, "user-rate-limit-cleanup", func(ctx context.Context) error {
		close(rateLimitCleanupStop)
		return nil
	})

	idempotencyCleanupStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if err := idempotencyRepo.CleanupExpired(cleanupCtx); err != nil {
					logger.Warn("failed to cleanup idempotency keys", zap.Error(err))
				}
				cancel()
			case <-idempotencyCleanupStop:
				return
			}
		}
	}()
	shutdownCoord.RegisterFunc(shutdown.PhaseCleanup, "idempotency-cleanup", func(ctx context.Context) error {
		close(idempotencyCleanupStop)
		return nil
	})

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
	shutdownCoord.RegisterFunc(shutdown.PhaseShutdown, "csrf-protection", func(ctx context.Context) error {
		return csrfProtection.Shutdown(ctx)
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
// Returns both the logger and the atomic level for runtime adjustment.
func initLogger() (*zap.Logger, zap.AtomicLevel, error) {
	env := os.Getenv("APP_ENV")

	// Create atomic level for runtime adjustment
	var level zap.AtomicLevel
	if env == "production" {
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	} else {
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	// Build config with atomic level
	var config zap.Config
	if env == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}
	config.Level = level

	logger, err := config.Build()
	if err != nil {
		return nil, level, err
	}

	return logger, level, nil
}

// initVoiceProviders initializes and registers all configured voice providers.
func initVoiceProviders(cfg *config.Config, logger *zap.Logger) *voiceprovider.Registry {
	registry := voiceprovider.NewRegistry(logger)

	// Register Bland provider if enabled
	if cfg.VoiceProvider.Bland.Enabled || cfg.Bland.APIKey != "" {
		blandCfg := &blandprovider.Config{
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
		registry.Register(blandprovider.New(blandCfg, logger))
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

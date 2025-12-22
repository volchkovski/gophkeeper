// Package app provides the application initialization and startup logic.
package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/volchkovski/gophkeeper/internal/server/config"
	"github.com/volchkovski/gophkeeper/internal/server/handlers"
	"github.com/volchkovski/gophkeeper/internal/server/repository/postgres"
	"github.com/volchkovski/gophkeeper/internal/server/service"
	"github.com/volchkovski/gophkeeper/pkg/logger"
)

// App represents the server application.
type App struct {
	cfg    *config.Config
	log    *logger.Logger
	db     *postgres.DB
	server *http.Server
}

// BuildInfo contains build information.
type BuildInfo struct {
	Version   string
	BuildDate string
}

// New creates and initializes a new App instance.
func New(cfg *config.Config, buildInfo BuildInfo) (*App, error) {
	// Initialize logger
	log, err := logger.New(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.Info("starting GophKeeper server",
		logger.String("version", buildInfo.Version),
		logger.String("build_date", buildInfo.BuildDate),
	)

	// Connect to database
	db, err := postgres.NewDB(postgres.Config{
		DSN:             cfg.Database.DSN,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	})
	if err != nil {
		log.Error("failed to connect to database", logger.Error(err))
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Ping database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		log.Error("failed to ping database", logger.Error(err))
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	log.Info("connected to database")

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	secretRepo := postgres.NewSecretRepository(db)

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		service.AuthServiceConfig{
			JWTSecret:       cfg.JWT.Secret,
			AccessTokenTTL:  cfg.JWT.AccessTokenTTL,
			RefreshTokenTTL: cfg.JWT.RefreshTokenTTL,
			BcryptCost:      cfg.Security.BcryptCost,
		},
		log,
	)

	secretService := service.NewSecretService(secretRepo, log)

	// Initialize handlers
	handler := handlers.NewHandler(authService, secretService, log)

	// Set version info in handlers
	handlers.Version = buildInfo.Version
	handlers.BuildDate = buildInfo.BuildDate

	// Setup router
	router := handlers.SetupRouter(handler, authService, log, handlers.RouterConfig{
		RateLimit: cfg.Security.RateLimit,
	})

	// Create HTTP server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &App{
		cfg:    cfg,
		log:    log,
		db:     db,
		server: server,
	}, nil
}

// Run starts the server and blocks until shutdown signal is received.
func (a *App) Run() error {
	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		a.log.Info("server listening",
			logger.String("address", a.server.Addr),
			logger.String("tls", fmt.Sprintf("%v", a.cfg.TLS.Enabled)),
		)

		var err error
		if a.cfg.TLS.Enabled {
			err = a.server.ListenAndServeTLS(a.cfg.TLS.CertFile, a.cfg.TLS.KeyFile)
		} else {
			err = a.server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for interrupt signal or error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-quit:
		a.log.Info("shutting down server...")
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	a.log.Info("server stopped")
	return nil
}

// Close releases all resources.
func (a *App) Close() error {
	var errs []error

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.log.Error("failed to close database", logger.Error(err))
			errs = append(errs, err)
		}
	}

	if a.log != nil {
		_ = a.log.Sync()
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	return nil
}

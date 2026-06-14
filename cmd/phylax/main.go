package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pycify/phylax/internal/app/controllers"
	attestationservice "github.com/pycify/phylax/internal/app/services/attestation"
	identityservice "github.com/pycify/phylax/internal/app/services/identity"
	projectservice "github.com/pycify/phylax/internal/app/services/projects"
	"github.com/pycify/phylax/internal/app/services/store"
	"github.com/pycify/phylax/internal/config"
	"github.com/pycify/phylax/internal/env"
	"github.com/pycify/phylax/internal/httpapi"
)

// @title			Phylax API
// @version		0.1.0
// @description	Server-side mobile app attestation for Android, Apple, and Firebase-backed clients.
// @BasePath		/
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description					Type "Bearer " followed by a Phylax access token.
func main() {
	envPath := flag.String("env-file", ".env", "Path to optional .env file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := env.LoadFile(*envPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.Error("failed to load env file", "error", err)
		os.Exit(1)
	}

	if err := env.ValidateRequired(); err != nil {
		logger.Error("invalid environment", "error", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	st, err := store.New(cfg.Storage)
	if err != nil {
		logger.Error("failed to create store", "error", err)
		os.Exit(1)
	}

	projectRepository, err := createProjectRepository(cfg)
	if err != nil {
		logger.Error("failed to create project repository", "error", err)
		os.Exit(1)
	}
	projectService := projectservice.NewService(projectRepository)
	projectController := controllers.NewProjectController(projectService)

	service, err := attestationservice.NewService(cfg, st, logger, projectService)
	if err != nil {
		logger.Error("failed to create service", "error", err)
		os.Exit(1)
	}
	attestationController := controllers.NewAttestationController(service)

	authRepository, err := createAuthRepository(cfg)
	if err != nil {
		logger.Error("failed to create auth repository", "error", err)
		os.Exit(1)
	}
	otpStore, err := createOTPStore(cfg)
	if err != nil {
		logger.Error("failed to create otp store", "error", err)
		os.Exit(1)
	}
	authService := identityservice.NewAuthService(authRepository, otpStore, createMailer(cfg, logger))
	authController := controllers.NewAuthController(authService)

	handler := httpapi.NewRouter(attestationController, authService, authController, projectController)
	server := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("phylax listening", "addr", cfg.Server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("phylax stopped")
}

func createAuthRepository(cfg config.Config) (identityservice.AuthRepository, error) {
	switch cfg.Storage.Driver {
	case "", "memory":
		return identityservice.NewMemoryAuthRepository(), nil
	case "postgres":
		return identityservice.NewGormAuthRepository(cfg.Storage.DatabaseURL)
	default:
		return nil, store.ErrUnsupportedDriver(cfg.Storage.Driver)
	}
}

func createProjectRepository(cfg config.Config) (projectservice.Repository, error) {
	switch cfg.Storage.Driver {
	case "", "memory":
		return projectservice.NewMemoryRepository(), nil
	case "postgres":
		return projectservice.NewGormRepository(cfg.Storage.DatabaseURL)
	default:
		return nil, store.ErrUnsupportedDriver(cfg.Storage.Driver)
	}
}

func createOTPStore(cfg config.Config) (identityservice.OTPStore, error) {
	if cfg.Redis.URL == "" {
		return identityservice.NewMemoryOTPStore(), nil
	}
	return identityservice.NewRedisOTPStore(cfg.Redis.URL)
}

func createMailer(cfg config.Config, logger *slog.Logger) identityservice.Mailer {
	if cfg.Mail.Driver == "smtp" {
		return identityservice.NewSMTPMailer(cfg.Mail)
	}
	return identityservice.NewLogMailer(logger)
}

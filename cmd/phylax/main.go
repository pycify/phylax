package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pycify/phylax/internal/config"
	"github.com/pycify/phylax/internal/httpapi"
	"github.com/pycify/phylax/internal/phylax"
	"github.com/pycify/phylax/internal/store"
)

func main() {
	configPath := flag.String("config", "phylax.json", "Path to Phylax JSON config")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	service, err := phylax.NewService(cfg, store.NewMemoryStore(), logger)
	if err != nil {
		logger.Error("failed to create service", "error", err)
		os.Exit(1)
	}

	handler := httpapi.NewRouter(service)
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

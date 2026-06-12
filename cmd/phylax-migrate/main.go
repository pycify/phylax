package main

import (
	"errors"
	"flag"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	direction := flag.String("direction", "up", "Migration direction: up or down")
	source := flag.String("source", "file://database/migrations", "Migration source URL")
	envPath := flag.String("env-file", ".env", "Path to optional .env file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	_ = godotenv.Load(*envPath)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	migrator, err := migrate.New(*source, databaseURL)
	if err != nil {
		logger.Error("failed to create migrator", "error", err)
		os.Exit(1)
	}
	defer migrator.Close()

	switch *direction {
	case "up":
		err = migrator.Up()
	case "down":
		err = migrator.Down()
	default:
		logger.Error("unsupported migration direction", "direction", *direction)
		os.Exit(1)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		logger.Info("migrations already up to date")
		return
	}
	if err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	logger.Info("migration completed", "direction", *direction)
}

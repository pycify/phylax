package main

import (
	"database/sql"
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	seedDir := flag.String("dir", "database/seeders", "Seeder SQL directory")
	envPath := flag.String("env-file", ".env", "Path to optional .env file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	_ = godotenv.Load(*envPath)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	files, err := filepath.Glob(filepath.Join(*seedDir, "*.sql"))
	if err != nil {
		logger.Error("failed to list seeders", "error", err)
		os.Exit(1)
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			logger.Error("failed to read seeder", "file", file, "error", err)
			os.Exit(1)
		}
		if _, err := db.Exec(string(content)); err != nil {
			logger.Error("failed to run seeder", "file", file, "error", err)
			os.Exit(1)
		}
		logger.Info("seeder completed", "file", file)
	}
}

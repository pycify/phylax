package store

import (
	"fmt"

	"github.com/pycify/phylax/internal/config"
)

func ErrUnsupportedDriver(driver string) error {
	return fmt.Errorf("unsupported storage driver %q", driver)
}

func New(cfg config.StorageConfig) (Store, error) {
	switch cfg.Driver {
	case "", "memory":
		return NewMemoryStore(), nil
	case "postgres":
		return NewPostgresStore(cfg.DatabaseURL)
	default:
		return nil, ErrUnsupportedDriver(cfg.Driver)
	}
}

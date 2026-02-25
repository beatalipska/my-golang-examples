package config

import (
	"errors"
	"os"
)

type Config struct {
	DBURL         string
	WebhookSecret string
}

func Load() (Config, error) {
	cfg := Config{
		DBURL:         os.Getenv("DB_URL"),
		WebhookSecret: os.Getenv("WEBHOOK_SECRET"),
	}
	if cfg.DBURL == "" {
		return Config{}, errors.New("DB_URL is required")
	}
	if cfg.WebhookSecret == "" {
		// na start możesz pozwolić na empty i dodać później,
		// ale w take-home lepiej wymusić
		return Config{}, errors.New("WEBHOOK_SECRET is required")
	}
	return cfg, nil
}

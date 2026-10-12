// Package config loads runtime settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// Config holds every setting the server needs.
type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	TokenTTL    time.Duration
	HISTimeout  time.Duration
}

// Load reads the configuration from the environment and validates required values.
func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}

	var err error
	if cfg.TokenTTL, err = getDuration("TOKEN_TTL", 24*time.Hour); err != nil {
		return nil, err
	}
	if cfg.HISTimeout, err = getDuration("HIS_TIMEOUT", 5*time.Second); err != nil {
		return nil, err
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 16 {
		return nil, errors.New("JWT_SECRET is required and must be at least 16 characters")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}

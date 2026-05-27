package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL         string
	MigratorDatabaseURL string
	BaseURL             string
	Port                string
	SessionSecret       string
	OwnerTwitchID       string
	TwitchClientID      string
	TwitchClientSecret  string
	TMDBAPIKey          string
	LogLevel            string
	SecureCookies       bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:         coalesce("DATABASE_URL", "MARQUEE_APP_DATABASE_URL"),
		MigratorDatabaseURL: coalesce("MIGRATOR_DATABASE_URL", "MARQUEE_MIGRATOR_DATABASE_URL"),
		BaseURL:             coalesce("BASE_URL", "MARQUEE_BASE_URL"),
		Port:                coalesce("PORT", "MARQUEE_PORT"),
		SessionSecret:       coalesce("SESSION_SECRET", "MARQUEE_SESSION_SECRET"),
		OwnerTwitchID:       coalesce("OWNER_TWITCH_ID", "MARQUEE_OWNER_TWITCH_ID"),
		TwitchClientID:      coalesce("TWITCH_CLIENT_ID", "MARQUEE_TWITCH_CLIENT_ID"),
		TwitchClientSecret:  coalesce("TWITCH_CLIENT_SECRET", "MARQUEE_TWITCH_CLIENT_SECRET"),
		TMDBAPIKey:          coalesce("TMDB_API_KEY", "MARQUEE_TMDB_API_KEY"),
		LogLevel:            coalesce("LOG_LEVEL", "MARQUEE_LOG_LEVEL"),
	}

	if cfg.Port == "" {
		cfg.Port = "3051"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	cfg.SecureCookies = strings.HasPrefix(cfg.BaseURL, "https://")

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	slog.Info("config loaded", "port", cfg.Port, "base_url", cfg.BaseURL, "secure_cookies", cfg.SecureCookies)
	return cfg, nil
}

func (c *Config) validate() error {
	required := map[string]string{
		"DATABASE_URL":         c.DatabaseURL,
		"MIGRATOR_DATABASE_URL": c.MigratorDatabaseURL,
		"BASE_URL":             c.BaseURL,
		"SESSION_SECRET":       c.SessionSecret,
		"OWNER_TWITCH_ID":      c.OwnerTwitchID,
	}
	for name, val := range required {
		if val == "" {
			return fmt.Errorf("required env var %s is not set", name)
		}
	}
	if len(c.SessionSecret) < 32 {
		return fmt.Errorf("SESSION_SECRET must be at least 32 bytes (got %d)", len(c.SessionSecret))
	}
	return nil
}

func coalesce(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

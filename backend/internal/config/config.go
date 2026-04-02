package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration values.
type Config struct {
	TursoDatabaseURL string
	TursoAuthToken   string
	ServerPort       string
}

// Load reads environment variables and returns a validated Config.
// It loads from a .env file if present, then overrides with actual env vars.
func Load() (*Config, error) {
	// Attempt to load .env file (best-effort, not fatal if missing)
	loadDotEnv(".env")

	cfg := &Config{
		TursoDatabaseURL: getEnv("TURSO_DATABASE_URL", ""),
		TursoAuthToken:   getEnv("TURSO_AUTH_TOKEN", ""),
		ServerPort:       getEnv("SERVER_PORT", "8080"),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// DSN returns the full connection string for the libsql driver.
// Format: libsql://host?authToken=token
func (c *Config) DSN() string {
	if c.TursoAuthToken == "" {
		return c.TursoDatabaseURL
	}
	return fmt.Sprintf("%s?authToken=%s", c.TursoDatabaseURL, c.TursoAuthToken)
}

// validate checks that all required configuration values are present.
func (c *Config) validate() error {
	if c.TursoDatabaseURL == "" {
		return fmt.Errorf("TURSO_DATABASE_URL is required")
	}
	return nil
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

// loadDotEnv is a minimal .env file parser.
// It reads KEY=VALUE pairs (one per line) and sets them as environment variables
// only if they are not already set (env vars take precedence).
func loadDotEnv(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return // .env file is optional
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Only set if not already defined in the environment
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

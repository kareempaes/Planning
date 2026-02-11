package main

import "os"

// Config holds application configuration loaded from environment variables.
type Config struct {
	DBDriver       string
	DBDSN          string
	Port           string
	JWTSecret      string
	MigrationsPath string
}

// LoadConfig reads configuration from environment variables with sensible defaults.
func LoadConfig() Config {
	return Config{
		DBDriver:       getEnv("DB_DRIVER", "sqlite"),
		DBDSN:          getEnv("DB_DSN", ":memory:"),
		Port:           getEnv("PORT", "8080"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-secret-do-not-use-in-production"),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "db/migrations"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

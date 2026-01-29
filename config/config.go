package config

import (
	"fmt"
	"os"
)

type Config struct {
	TelegramToken string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	PythonAPIURL string

	CheckInterval int
}

func Load() (*Config, error) {
	cfg := &Config{
		DBHost:        getEnv("DB_Host", "localhost"),
		DBPort:        getEnv("DB_Posrt", "5432"),
		DBUser:        getEnv("DB_USER", "botuser"),
		DBPassword:    getEnv("DB_PASSWORD", "botpass"),
		DBName:        getEnv("DB_NAME", "shopbot"),
		PythonAPIURL:  getEnv("PYTHON_API_URL", "http://localhost:5000"),
		CheckInterval: getEnvAsInt("CHECK_INTERVAL", 5),
	}

	cfg.TelegramToken = os.Getenv("TELEGRAM_TOKEN")
	if cfg.TelegramToken == "" {
		return nil, fmt.Errorf("TELEGRAM_TOKEN is required")
	}

	return cfg, nil
}

func (c *Config) PostgreSQLConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "d%", &result); err == nil {
			return result
		}
	}

	return defaultValue
}

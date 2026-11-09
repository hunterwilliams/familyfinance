package config

import (
	"os"
	"strings"
)

type Config struct {
	Port                   string
	DatabaseURL            string
	APIKey                 string
	ReceiptsDir            string
	MigrationsPath         string
	JWTSecret              string
	SkipAuth               bool
	WebBaseURL             string
	PublicBaseURL          string
	OpenAIAPIKey           string
	OpenAIModel            string
	OpenAIBaseURL          string
	LineLoginChannelID     string
	LineLoginChannelSecret string
	LineLoginCallbackURL   string
	LineChannelSecret      string
	LineChannelAccessToken string
}

func Load() Config {
	port := getenv("PORT", "8080")
	dbURL := getenv("DATABASE_URL", "postgres://familyfinance:familyfinance@localhost:5432/familyfinance?sslmode=disable")
	apiKey := getenv("API_KEY", "dev-api-key")
	receiptsDir := getenv("RECEIPTS_DIR", "./data/receipts")
	migrationsPath := getenv("MIGRATIONS_PATH", "migrations")
	jwtSecret := getenv("JWT_SECRET", "dev-jwt-secret-change-me")
	skip := strings.EqualFold(os.Getenv("SKIP_AUTH"), "true") || strings.EqualFold(os.Getenv("AUTH_MODE"), "skip")

	return Config{
		Port:                   port,
		DatabaseURL:            dbURL,
		APIKey:                 apiKey,
		ReceiptsDir:            receiptsDir,
		MigrationsPath:         migrationsPath,
		JWTSecret:              jwtSecret,
		SkipAuth:               skip,
		WebBaseURL:             getenv("WEB_BASE_URL", "http://localhost:5173"),
		PublicBaseURL:          getenv("PUBLIC_BASE_URL", "http://localhost:8080"),
		OpenAIAPIKey:           os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:            getenv("OPENAI_MODEL", "gpt-4o"),
		OpenAIBaseURL:          os.Getenv("OPENAI_BASE_URL"),
		LineLoginChannelID:     os.Getenv("LINE_LOGIN_CHANNEL_ID"),
		LineLoginChannelSecret: os.Getenv("LINE_LOGIN_CHANNEL_SECRET"),
		LineLoginCallbackURL:   getenv("LINE_LOGIN_CALLBACK_URL", "http://localhost:5173/callback"),
		LineChannelSecret:      os.Getenv("LINE_CHANNEL_SECRET"),
		LineChannelAccessToken: os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

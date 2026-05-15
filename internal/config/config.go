package config

import (
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr string
	LogLevel slog.Level
	DBDSN    string
	SaveBody bool

	OpenAIBaseURL string
	OpenAIAPIKey  string
}

func Load() Config {
	return Config{
		HTTPAddr:     getenv("AIOS_HTTP_ADDR", ":8080"),
		LogLevel:     slog.LevelInfo,
		DBDSN:        getenv("AIOS_DB_DSN", "itw.sqlite"),
		SaveBody:     getenvBool("AIOS_SAVE_BODY", false),
		OpenAIBaseURL: getenv("AIOS_OPENAI_BASE_URL", ""),
		OpenAIAPIKey:  getenv("AIOS_OPENAI_API_KEY", ""),
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

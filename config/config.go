package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               string
	FrontendOrigin     string
	JWTSecret          string
	JWTExpiration      time.Duration
	GoogleClientID     string
	DatabaseURL        string
	DBMaxConns         int32
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	ShutdownTimeoutSec int
}

func Load() Config {
	return Config{
		Port:               getEnv("PORT", "8080"),
		FrontendOrigin:     getEnv("FRONTEND_ORIGIN", "http://localhost:5500"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpiration:      time.Duration(getEnvAsInt("JWT_EXP_HOURS", 24)) * time.Hour,
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/quizarena?sslmode=disable"),
		DBMaxConns:         int32(getEnvAsInt("DB_MAX_CONNS", 30)),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvAsInt("REDIS_DB", 0),
		ShutdownTimeoutSec: getEnvAsInt("SHUTDOWN_TIMEOUT_SEC", 10),
	}
}

func CORSOrigins(origin string) []string {
	parts := strings.Split(origin, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if valueStr == "" {
		return defaultVal
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultVal
	}
	return value
}

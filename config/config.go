package config

import (
	"math"
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
		DBMaxConns:         getEnvAsInt32("DB_MAX_CONNS", 30),
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

func getEnvAsInt32(name string, defaultVal int32) int32 {
	valueStr := getEnv(name, "")
	if valueStr == "" {
		return defaultVal
	}
	value64, err := strconv.ParseInt(valueStr, 10, 32)
	if err != nil {
		return defaultVal
	}
	if value64 < 0 || value64 > math.MaxInt32 {
		return defaultVal
	}
	return int32(value64)
}

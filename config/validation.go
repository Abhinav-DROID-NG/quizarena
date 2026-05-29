package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Abhinav-DROID-NG/quizarena/utils"
)

var ErrInvalidConfig = errors.New("invalid config")

func Validate(cfg Config) ([]string, error) {
	var warnings []string
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return nil, fmt.Errorf("%w: JWT_SECRET is required", ErrInvalidConfig)
	}
	if cfg.JWTSecret == "change-me-in-production" {
		warnings = append(warnings, "JWT_SECRET is using insecure default")
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return nil, fmt.Errorf("%w: DATABASE_URL is required", ErrInvalidConfig)
	}
	if strings.TrimSpace(cfg.GoogleClientID) == "" {
		warnings = append(warnings, "GOOGLE_CLIENT_ID is empty; Google OAuth will not work")
	}

	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < utils.MinPort || port > utils.MaxPort {
		return nil, fmt.Errorf("%w: invalid PORT", ErrInvalidConfig)
	}

	expHours := int(cfg.JWTExpiration / time.Hour)
	if expHours < utils.MinJWTExpirationHours || expHours > utils.MaxJWTExpirationHours {
		return nil, fmt.Errorf("%w: JWT_EXP_HOURS out of range", ErrInvalidConfig)
	}

	if cfg.DBMaxConns < utils.MinDBMaxConns || cfg.DBMaxConns > utils.MaxDBMaxConns {
		return nil, fmt.Errorf("%w: DB_MAX_CONNS out of range", ErrInvalidConfig)
	}
	if cfg.ShutdownTimeoutSec < utils.MinShutdownTimeoutSec || cfg.ShutdownTimeoutSec > utils.MaxShutdownTimeoutSec {
		return nil, fmt.Errorf("%w: SHUTDOWN_TIMEOUT_SEC out of range", ErrInvalidConfig)
	}

	origins := CORSOrigins(cfg.FrontendOrigin)
	if len(origins) == 0 {
		warnings = append(warnings, "FRONTEND_ORIGIN is empty; browser clients from other origins will be rejected")
	}
	if len(cfg.AdminEmails) == 0 {
		warnings = append(warnings, "ADMIN_EMAILS is empty; admin routes will return forbidden")
	}

	return warnings, nil
}

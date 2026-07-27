package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	AppEnv                  string
	HTTPAddr                string
	MySQLDSN                string
	JWTIssuer               string
	JWTAccessSecret         string
	JWTAccessTTL            time.Duration
	JWTRefreshTTL           time.Duration
	RedisAddr               string
	RedisPassword           string
	RedisDB                 int
	PublicCacheTTL          time.Duration
	CORSAllowedOrigins      []string
	DefaultOrganizationSlug string
	BootstrapAdminEmail     string
	BootstrapAdminPassword  string
	BootstrapAdminName      string
}

func Load() Config {
	return Config{
		AppEnv:                  value("APP_ENV", "development"),
		HTTPAddr:                value("HTTP_ADDR", ":8080"),
		MySQLDSN:                value("MYSQL_DSN", "qutcraft:qutcraft@tcp(localhost:3306)/qutcraft?charset=utf8mb4&parseTime=True&loc=UTC"),
		JWTIssuer:               value("JWT_ISSUER", "qutcraft-platform"),
		JWTAccessSecret:         value("JWT_ACCESS_SECRET", "development-only-change-me-before-production"),
		JWTAccessTTL:            duration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:           duration("JWT_REFRESH_TTL", 7*24*time.Hour),
		RedisAddr:               value("REDIS_ADDR", "localhost:6379"),
		RedisPassword:           os.Getenv("REDIS_PASSWORD"),
		RedisDB:                 integer("REDIS_DB", 0),
		PublicCacheTTL:          duration("PUBLIC_CACHE_TTL", 30*time.Second),
		CORSAllowedOrigins:      strings.Split(value("CORS_ALLOWED_ORIGINS", "http://localhost:8082,http://127.0.0.1:8082,http://localhost,http://127.0.0.1"), ","),
		DefaultOrganizationSlug: value("DEFAULT_ORGANIZATION_SLUG", "qutcraft"),
		BootstrapAdminEmail:     strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL"))),
		BootstrapAdminPassword:  os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		BootstrapAdminName:      value("BOOTSTRAP_ADMIN_NAME", "QUTCraft Admin"),
	}
}

func integer(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscanf(raw, "%d", &parsed); err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func value(key, fallback string) string {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		return raw
	}
	return fallback
}

func duration(key string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(value(key, fallback.String()))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

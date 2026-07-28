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
	ServerAdapterTimeout    time.Duration
	CORSAllowedOrigins      []string
	AuthRateLimitPerMinute  int
	PublicWriteLimitPerHour int
	SensitiveLimitPerMinute int
	DefaultOrganizationSlug string
	BootstrapAdminEmail     string
	BootstrapAdminPassword  string
	BootstrapAdminName      string
	DemoSeedEnabled         bool
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
		ServerAdapterTimeout:    duration("SERVER_ADAPTER_TIMEOUT", 5*time.Second),
		CORSAllowedOrigins:      csv(value("CORS_ALLOWED_ORIGINS", "http://localhost:8082,http://127.0.0.1:8082,http://localhost,http://127.0.0.1")),
		AuthRateLimitPerMinute:  positiveInteger("AUTH_RATE_LIMIT_PER_MINUTE", 20),
		PublicWriteLimitPerHour: positiveInteger("PUBLIC_WRITE_LIMIT_PER_HOUR", 10),
		SensitiveLimitPerMinute: positiveInteger("SENSITIVE_RATE_LIMIT_PER_MINUTE", 30),
		DefaultOrganizationSlug: value("DEFAULT_ORGANIZATION_SLUG", "qutcraft"),
		BootstrapAdminEmail:     strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL"))),
		BootstrapAdminPassword:  os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		BootstrapAdminName:      value("BOOTSTRAP_ADMIN_NAME", "QUTCraft Admin"),
		DemoSeedEnabled:         boolean("DEMO_SEED_ENABLED", false),
	}
}

func (c Config) Validate() error {
	if len(c.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must contain at least one origin")
	}
	for _, origin := range c.CORSAllowedOrigins {
		if origin == "*" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS cannot contain wildcard when credentials are enabled")
		}
		if !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
			return fmt.Errorf("invalid CORS origin %q", origin)
		}
	}
	if c.BootstrapAdminPassword != "" && len(c.BootstrapAdminPassword) < 12 {
		return fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD must be at least 12 characters")
	}
	if strings.EqualFold(c.AppEnv, "production") {
		if len(c.JWTAccessSecret) < 32 || c.JWTAccessSecret == "development-only-change-me-before-production" || strings.Contains(strings.ToLower(c.JWTAccessSecret), "replace-with") {
			return fmt.Errorf("JWT_ACCESS_SECRET must be a non-placeholder secret of at least 32 characters in production")
		}
		if c.DemoSeedEnabled {
			return fmt.Errorf("DEMO_SEED_ENABLED must be false in production")
		}
	}
	return nil
}

func boolean(key string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
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

func positiveInteger(key string, fallback int) int {
	value := integer(key, fallback)
	if value <= 0 {
		return fallback
	}
	return value
}

func csv(raw string) []string {
	values := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
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

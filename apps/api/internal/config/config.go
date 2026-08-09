package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	AppEnv                    string
	HTTPAddr                  string
	MySQLDSN                  string
	JWTIssuer                 string
	JWTAccessSecret           string
	JWTAccessTTL              time.Duration
	JWTRefreshTTL             time.Duration
	RedisAddr                 string
	RedisPassword             string
	RedisDB                   int
	PublicCacheTTL            time.Duration
	ServerAdapterTimeout      time.Duration
	StorageDriver             string
	StorageLocalRoot          string
	S3Endpoint                string
	S3AccessKey               string
	S3SecretKey               string
	S3Bucket                  string
	S3Region                  string
	S3UseSSL                  bool
	PublicWebBaseURL          string
	EmailDriver               string
	SMTPHost                  string
	SMTPPort                  int
	SMTPUsername              string
	SMTPPassword              string
	SMTPFromAddress           string
	SMTPFromName              string
	SMTPSecurity              string
	SMTPTimeout               time.Duration
	AIProvider                string
	AIBaseURL                 string
	AIAPIKey                  string
	AIModel                   string
	AIRequestTimeout          time.Duration
	AIRunLimitPerHour         int
	CORSAllowedOrigins        []string
	AuthRateLimitPerMinute    int
	PublicWriteLimitPerHour   int
	SensitiveLimitPerMinute   int
	DefaultOrganizationSlug   string
	BootstrapAdminEmail       string
	BootstrapAdminPassword    string
	BootstrapAdminName        string
	DemoSeedEnabled           bool
	DemoSeedProfile           string
	DemoSeedMultiOrganization bool
}

func Load() Config {
	return Config{
		AppEnv:                    value("APP_ENV", "development"),
		HTTPAddr:                  value("HTTP_ADDR", ":8080"),
		MySQLDSN:                  value("MYSQL_DSN", "qutcraft:qutcraft@tcp(localhost:3306)/qutcraft?charset=utf8mb4&parseTime=True&loc=UTC"),
		JWTIssuer:                 value("JWT_ISSUER", "qutcraft-platform"),
		JWTAccessSecret:           value("JWT_ACCESS_SECRET", "development-only-change-me-before-production"),
		JWTAccessTTL:              duration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:             duration("JWT_REFRESH_TTL", 7*24*time.Hour),
		RedisAddr:                 value("REDIS_ADDR", "localhost:6379"),
		RedisPassword:             os.Getenv("REDIS_PASSWORD"),
		RedisDB:                   integer("REDIS_DB", 0),
		PublicCacheTTL:            duration("PUBLIC_CACHE_TTL", 30*time.Second),
		ServerAdapterTimeout:      duration("SERVER_ADAPTER_TIMEOUT", 5*time.Second),
		StorageDriver:             strings.ToLower(value("STORAGE_DRIVER", "local")),
		StorageLocalRoot:          value("STORAGE_LOCAL_ROOT", "/tmp/qutcraft-uploads"),
		S3Endpoint:                value("S3_ENDPOINT", "minio:9000"),
		S3AccessKey:               strings.TrimSpace(os.Getenv("S3_ACCESS_KEY")),
		S3SecretKey:               os.Getenv("S3_SECRET_KEY"),
		S3Bucket:                  value("S3_BUCKET", "qutcraft-media"),
		S3Region:                  value("S3_REGION", "us-east-1"),
		S3UseSSL:                  boolean("S3_USE_SSL", false),
		PublicWebBaseURL:          strings.TrimRight(value("PUBLIC_WEB_BASE_URL", "http://localhost:8082"), "/"),
		EmailDriver:               strings.ToLower(value("EMAIL_DRIVER", "disabled")),
		SMTPHost:                  strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:                  integer("SMTP_PORT", 587),
		SMTPUsername:              strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:              os.Getenv("SMTP_PASSWORD"),
		SMTPFromAddress:           strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_FROM_ADDRESS"))),
		SMTPFromName:              value("SMTP_FROM_NAME", "QUTCraft Commons"),
		SMTPSecurity:              strings.ToLower(value("SMTP_SECURITY", "starttls")),
		SMTPTimeout:               duration("SMTP_TIMEOUT", 8*time.Second),
		AIProvider:                strings.ToLower(value("AI_PROVIDER", "disabled")),
		AIBaseURL:                 strings.TrimRight(strings.TrimSpace(os.Getenv("AI_BASE_URL")), "/"),
		AIAPIKey:                  os.Getenv("AI_API_KEY"),
		AIModel:                   strings.TrimSpace(os.Getenv("AI_MODEL")),
		AIRequestTimeout:          duration("AI_REQUEST_TIMEOUT", 30*time.Second),
		AIRunLimitPerHour:         positiveInteger("AI_RUN_LIMIT_PER_HOUR", 20),
		CORSAllowedOrigins:        csv(value("CORS_ALLOWED_ORIGINS", "http://localhost:8082,http://127.0.0.1:8082,http://localhost,http://127.0.0.1")),
		AuthRateLimitPerMinute:    positiveInteger("AUTH_RATE_LIMIT_PER_MINUTE", 20),
		PublicWriteLimitPerHour:   positiveInteger("PUBLIC_WRITE_LIMIT_PER_HOUR", 10),
		SensitiveLimitPerMinute:   positiveInteger("SENSITIVE_RATE_LIMIT_PER_MINUTE", 30),
		DefaultOrganizationSlug:   value("DEFAULT_ORGANIZATION_SLUG", "qutcraft"),
		BootstrapAdminEmail:       strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL"))),
		BootstrapAdminPassword:    os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		BootstrapAdminName:        value("BOOTSTRAP_ADMIN_NAME", "QUTCraft Admin"),
		DemoSeedEnabled:           boolean("DEMO_SEED_ENABLED", false),
		DemoSeedProfile:           strings.ToLower(value("DEMO_SEED_PROFILE", "qutcraft")),
		DemoSeedMultiOrganization: boolean("DEMO_SEED_MULTI_ORGANIZATION", false),
	}
}

func (c Config) Validate() error {
	if c.DemoSeedProfile != "" && c.DemoSeedProfile != "qutcraft" && c.DemoSeedProfile != "generic" {
		return fmt.Errorf("DEMO_SEED_PROFILE must be qutcraft or generic")
	}
	switch c.StorageDriver {
	case "local":
		if strings.TrimSpace(c.StorageLocalRoot) == "" {
			return fmt.Errorf("STORAGE_LOCAL_ROOT is required when STORAGE_DRIVER=local")
		}
	case "s3":
		if strings.TrimSpace(c.S3Endpoint) == "" || strings.Contains(c.S3Endpoint, "://") {
			return fmt.Errorf("S3_ENDPOINT must be a non-empty host:port without URL scheme")
		}
		if strings.TrimSpace(c.S3AccessKey) == "" || strings.TrimSpace(c.S3SecretKey) == "" {
			return fmt.Errorf("S3_ACCESS_KEY and S3_SECRET_KEY are required when STORAGE_DRIVER=s3")
		}
		if strings.TrimSpace(c.S3Bucket) == "" {
			return fmt.Errorf("S3_BUCKET is required when STORAGE_DRIVER=s3")
		}
	default:
		return fmt.Errorf("STORAGE_DRIVER must be local or s3")
	}
	if !strings.HasPrefix(c.PublicWebBaseURL, "http://") && !strings.HasPrefix(c.PublicWebBaseURL, "https://") {
		return fmt.Errorf("PUBLIC_WEB_BASE_URL must be an absolute http or https URL")
	}
	switch c.EmailDriver {
	case "disabled":
	case "smtp":
		if strings.TrimSpace(c.SMTPHost) == "" {
			return fmt.Errorf("SMTP_HOST is required when EMAIL_DRIVER=smtp")
		}
		if c.SMTPPort < 1 || c.SMTPPort > 65535 {
			return fmt.Errorf("SMTP_PORT must be between 1 and 65535")
		}
		if strings.TrimSpace(c.SMTPFromAddress) == "" {
			return fmt.Errorf("SMTP_FROM_ADDRESS is required when EMAIL_DRIVER=smtp")
		}
		if c.SMTPUsername != "" && c.SMTPPassword == "" {
			return fmt.Errorf("SMTP_PASSWORD is required when SMTP_USERNAME is set")
		}
		if c.SMTPSecurity != "starttls" && c.SMTPSecurity != "tls" && c.SMTPSecurity != "none" {
			return fmt.Errorf("SMTP_SECURITY must be starttls, tls or none")
		}
	default:
		return fmt.Errorf("EMAIL_DRIVER must be disabled or smtp")
	}
	switch c.AIProvider {
	case "", "disabled", "mock":
	case "openai_compatible":
		if !strings.HasPrefix(c.AIBaseURL, "http://") && !strings.HasPrefix(c.AIBaseURL, "https://") {
			return fmt.Errorf("AI_BASE_URL must be an absolute http or https URL when AI_PROVIDER=openai_compatible")
		}
		if strings.TrimSpace(c.AIAPIKey) == "" {
			return fmt.Errorf("AI_API_KEY is required when AI_PROVIDER=openai_compatible")
		}
		if strings.TrimSpace(c.AIModel) == "" {
			return fmt.Errorf("AI_MODEL is required when AI_PROVIDER=openai_compatible")
		}
	default:
		return fmt.Errorf("AI_PROVIDER must be disabled, mock or openai_compatible")
	}
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
		if c.DemoSeedMultiOrganization {
			return fmt.Errorf("DEMO_SEED_MULTI_ORGANIZATION must be false in production")
		}
		if c.StorageDriver == "s3" && (strings.EqualFold(c.S3AccessKey, "minioadmin") || strings.Contains(strings.ToLower(c.S3SecretKey), "change-me")) {
			return fmt.Errorf("S3 credentials must not use development placeholders in production")
		}
		if c.EmailDriver == "smtp" && c.SMTPSecurity == "none" {
			return fmt.Errorf("SMTP_SECURITY=none is not allowed in production")
		}
		if c.AIProvider == "mock" {
			return fmt.Errorf("AI_PROVIDER=mock is not allowed in production")
		}
		if c.AIProvider == "openai_compatible" && !strings.HasPrefix(c.AIBaseURL, "https://") {
			return fmt.Errorf("AI_BASE_URL must use https in production")
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

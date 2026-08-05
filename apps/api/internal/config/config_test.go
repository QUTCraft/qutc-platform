package config

import (
	"strings"
	"testing"
)

func TestLoadDemoSeedEnabled(t *testing.T) {
	t.Setenv("DEMO_SEED_ENABLED", "true")
	if !Load().DemoSeedEnabled {
		t.Fatal("DEMO_SEED_ENABLED=true was not enabled")
	}

	t.Setenv("DEMO_SEED_ENABLED", "false")
	if Load().DemoSeedEnabled {
		t.Fatal("DEMO_SEED_ENABLED=false was not disabled")
	}

	t.Setenv("DEMO_SEED_ENABLED", "invalid")
	if Load().DemoSeedEnabled {
		t.Fatal("invalid DEMO_SEED_ENABLED should use the safe false fallback")
	}

	t.Setenv("DEMO_SEED_MULTI_ORGANIZATION", "true")
	if !Load().DemoSeedMultiOrganization {
		t.Fatal("DEMO_SEED_MULTI_ORGANIZATION=true was not enabled")
	}
}

func TestDemoSeedProfiles(t *testing.T) {
	t.Setenv("DEMO_SEED_PROFILE", "generic")
	if cfg := Load(); cfg.DemoSeedProfile != "generic" || cfg.Validate() != nil {
		t.Fatalf("generic demo profile was not accepted: %+v", cfg)
	}
	t.Setenv("DEMO_SEED_PROFILE", "unknown")
	if err := Load().Validate(); err == nil {
		t.Fatal("unknown demo profile was accepted")
	}
}

func TestLoadNormalizesSecuritySettings(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://portal.example.test, ,https://admin.example.test ")
	t.Setenv("AUTH_RATE_LIMIT_PER_MINUTE", "0")
	t.Setenv("PUBLIC_WRITE_LIMIT_PER_HOUR", "25")
	t.Setenv("SENSITIVE_RATE_LIMIT_PER_MINUTE", "invalid")

	cfg := Load()
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[0] != "https://portal.example.test" || cfg.CORSAllowedOrigins[1] != "https://admin.example.test" {
		t.Fatalf("CORSAllowedOrigins = %#v", cfg.CORSAllowedOrigins)
	}
	if cfg.AuthRateLimitPerMinute != 20 || cfg.PublicWriteLimitPerHour != 25 || cfg.SensitiveLimitPerMinute != 30 {
		t.Fatalf(
			"rate limits = %d/%d/%d",
			cfg.AuthRateLimitPerMinute,
			cfg.PublicWriteLimitPerHour,
			cfg.SensitiveLimitPerMinute,
		)
	}
}

func TestValidateStorageConfiguration(t *testing.T) {
	base := Config{
		AppEnv:             "development",
		StorageDriver:      "local",
		StorageLocalRoot:   "/tmp/qutcraft-uploads",
		PublicWebBaseURL:   "https://portal.example.test",
		EmailDriver:        "disabled",
		CORSAllowedOrigins: []string{"https://portal.example.test"},
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("local storage config rejected: %v", err)
	}

	s3 := base
	s3.StorageDriver = "s3"
	s3.S3Endpoint = "minio:9000"
	s3.S3AccessKey = "integration-user"
	s3.S3SecretKey = "integration-secret"
	s3.S3Bucket = "qutcraft-media"
	if err := s3.Validate(); err != nil {
		t.Fatalf("S3 storage config rejected: %v", err)
	}

	for name, mutate := range map[string]func(*Config){
		"unknown driver":     func(cfg *Config) { cfg.StorageDriver = "ftp" },
		"endpoint scheme":    func(cfg *Config) { cfg.S3Endpoint = "http://minio:9000" },
		"missing access key": func(cfg *Config) { cfg.S3AccessKey = "" },
		"missing secret key": func(cfg *Config) { cfg.S3SecretKey = "" },
		"missing bucket":     func(cfg *Config) { cfg.S3Bucket = "" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := s3
			mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("invalid storage configuration was accepted")
			}
		})
	}
}

func TestValidateRejectsUnsafeProductionConfiguration(t *testing.T) {
	base := Config{
		AppEnv:                  "production",
		JWTAccessSecret:         strings.Repeat("a", 48),
		StorageDriver:           "local",
		StorageLocalRoot:        "/tmp/qutcraft-uploads",
		PublicWebBaseURL:        "https://portal.example.test",
		EmailDriver:             "disabled",
		CORSAllowedOrigins:      []string{"https://portal.example.test"},
		AuthRateLimitPerMinute:  20,
		PublicWriteLimitPerHour: 10,
		SensitiveLimitPerMinute: 30,
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("safe production config rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "wildcard CORS", mutate: func(cfg *Config) { cfg.CORSAllowedOrigins = []string{"*"} }},
		{name: "placeholder JWT", mutate: func(cfg *Config) { cfg.JWTAccessSecret = "replace-with-a-long-random-production-secret" }},
		{name: "short JWT", mutate: func(cfg *Config) { cfg.JWTAccessSecret = "too-short" }},
		{name: "demo seed", mutate: func(cfg *Config) { cfg.DemoSeedEnabled = true }},
		{name: "multi-organization demo seed", mutate: func(cfg *Config) { cfg.DemoSeedMultiOrganization = true }},
		{name: "short bootstrap password", mutate: func(cfg *Config) { cfg.BootstrapAdminPassword = "short" }},
		{name: "placeholder S3 credentials", mutate: func(cfg *Config) {
			cfg.StorageDriver = "s3"
			cfg.S3Endpoint = "minio:9000"
			cfg.S3AccessKey = "minioadmin"
			cfg.S3SecretKey = "minioadmin-change-me"
			cfg.S3Bucket = "qutcraft-media"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := base
			test.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("unsafe production config was accepted")
			}
		})
	}
}

func TestValidateEmailConfiguration(t *testing.T) {
	base := Config{
		AppEnv:             "development",
		StorageDriver:      "local",
		StorageLocalRoot:   "/tmp/qutcraft-uploads",
		PublicWebBaseURL:   "https://portal.example.test",
		EmailDriver:        "smtp",
		SMTPHost:           "smtp.example.test",
		SMTPPort:           587,
		SMTPUsername:       "mailer",
		SMTPPassword:       "secret",
		SMTPFromAddress:    "noreply@example.test",
		SMTPSecurity:       "starttls",
		CORSAllowedOrigins: []string{"https://portal.example.test"},
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid SMTP config rejected: %v", err)
	}
	for name, mutate := range map[string]func(*Config){
		"driver":        func(cfg *Config) { cfg.EmailDriver = "sendmail" },
		"host":          func(cfg *Config) { cfg.SMTPHost = "" },
		"port":          func(cfg *Config) { cfg.SMTPPort = 70000 },
		"sender":        func(cfg *Config) { cfg.SMTPFromAddress = "" },
		"password":      func(cfg *Config) { cfg.SMTPPassword = "" },
		"security":      func(cfg *Config) { cfg.SMTPSecurity = "invalid" },
		"public origin": func(cfg *Config) { cfg.PublicWebBaseURL = "/relative" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := base
			mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("invalid email configuration was accepted")
			}
		})
	}

	production := base
	production.AppEnv = "production"
	production.JWTAccessSecret = strings.Repeat("a", 48)
	production.SMTPSecurity = "none"
	if err := production.Validate(); err == nil {
		t.Fatal("unencrypted production SMTP was accepted")
	}
}

func TestLoadAndValidateAIConfiguration(t *testing.T) {
	t.Setenv("AI_PROVIDER", "openai_compatible")
	t.Setenv("AI_BASE_URL", "https://models.example.test/v1/")
	t.Setenv("AI_API_KEY", "test-only-provider-key")
	t.Setenv("AI_MODEL", "example-model")
	t.Setenv("AI_REQUEST_TIMEOUT", "45s")
	t.Setenv("AI_RUN_LIMIT_PER_HOUR", "12")

	cfg := Load()
	if cfg.AIBaseURL != "https://models.example.test/v1" || cfg.AIModel != "example-model" {
		t.Fatalf("AI configuration was not normalized: %+v", cfg)
	}
	if cfg.AIRequestTimeout.String() != "45s" || cfg.AIRunLimitPerHour != 12 {
		t.Fatalf("AI limits = %s/%d, want 45s/12", cfg.AIRequestTimeout, cfg.AIRunLimitPerHour)
	}

	base := Config{
		AppEnv:             "development",
		StorageDriver:      "local",
		StorageLocalRoot:   "/tmp/qutcraft-uploads",
		PublicWebBaseURL:   "https://portal.example.test",
		EmailDriver:        "disabled",
		AIProvider:         "openai_compatible",
		AIBaseURL:          "https://models.example.test/v1",
		AIAPIKey:           "test-only-provider-key",
		AIModel:            "example-model",
		CORSAllowedOrigins: []string{"https://portal.example.test"},
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid AI configuration rejected: %v", err)
	}
	for name, mutate := range map[string]func(*Config){
		"driver":   func(cfg *Config) { cfg.AIProvider = "unknown" },
		"base URL": func(cfg *Config) { cfg.AIBaseURL = "/v1" },
		"API key":  func(cfg *Config) { cfg.AIAPIKey = "" },
		"model":    func(cfg *Config) { cfg.AIModel = "" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := base
			mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("invalid AI configuration was accepted")
			}
		})
	}

	production := base
	production.AppEnv = "production"
	production.JWTAccessSecret = strings.Repeat("a", 48)
	production.AIProvider = "mock"
	if err := production.Validate(); err == nil {
		t.Fatal("production mock provider was accepted")
	}
	production.AIProvider = "openai_compatible"
	production.AIBaseURL = "http://models.example.test/v1"
	if err := production.Validate(); err == nil {
		t.Fatal("unencrypted production model endpoint was accepted")
	}
}

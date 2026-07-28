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

func TestValidateRejectsUnsafeProductionConfiguration(t *testing.T) {
	base := Config{
		AppEnv:                  "production",
		JWTAccessSecret:         strings.Repeat("a", 48),
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
		{name: "short bootstrap password", mutate: func(cfg *Config) { cfg.BootstrapAdminPassword = "short" }},
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

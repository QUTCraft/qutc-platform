package config

import (
	"strings"
	"testing"
)

// TestLoadDemoSeedEnabled 验证演示数据相关布尔环境变量的解析规则。
// 该测试同时覆盖 true、false、非法值回退和多组织开关，确保错误输入不会意外开启危险功能。
func TestLoadDemoSeedEnabled(t *testing.T) {
	// 标准真值应打开演示数据初始化。
	t.Setenv("DEMO_SEED_ENABLED", "true")
	if !Load().DemoSeedEnabled {
		t.Fatal("DEMO_SEED_ENABLED=true was not enabled")
	}

	// 标准假值应明确关闭该功能。
	t.Setenv("DEMO_SEED_ENABLED", "false")
	if Load().DemoSeedEnabled {
		t.Fatal("DEMO_SEED_ENABLED=false was not disabled")
	}

	// 无法识别的输入必须使用 Load 中定义的安全默认值 false，而不能按“非空即真”处理。
	t.Setenv("DEMO_SEED_ENABLED", "invalid")
	if Load().DemoSeedEnabled {
		t.Fatal("invalid DEMO_SEED_ENABLED should use the safe false fallback")
	}

	// 多组织演示数据使用相同的布尔解析器，但有独立的环境变量和配置字段。
	t.Setenv("DEMO_SEED_MULTI_ORGANIZATION", "true")
	if !Load().DemoSeedMultiOrganization {
		t.Fatal("DEMO_SEED_MULTI_ORGANIZATION=true was not enabled")
	}
}

// TestDemoSeedProfiles 验证已实现的演示数据模板会被接受，而未知模板会被拒绝。
func TestDemoSeedProfiles(t *testing.T) {
	// generic 是除默认 qutcraft 外的另一个合法 profile；这里同时经过 Load 规范化和 Validate 校验。
	t.Setenv("DEMO_SEED_PROFILE", "generic")
	if cfg := Load(); cfg.DemoSeedProfile != "generic" || cfg.Validate() != nil {
		t.Fatalf("generic demo profile was not accepted: %+v", cfg)
	}
	// 未知 profile 没有对应的数据生成逻辑，必须在启动前报错。
	t.Setenv("DEMO_SEED_PROFILE", "unknown")
	if err := Load().Validate(); err == nil {
		t.Fatal("unknown demo profile was accepted")
	}
}

// TestLoadNormalizesSecuritySettings 验证列表清洗和正整数限流参数的安全回退行为。
func TestLoadNormalizesSecuritySettings(t *testing.T) {
	// CORS 输入故意包含外围空白和一个空项，以确认 csv 会清洗而不会产生无效来源。
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://portal.example.test, ,https://admin.example.test ")
	// 0 对速率限制没有业务意义，应回退到默认 20；合法的 25 则应保留。
	t.Setenv("AUTH_RATE_LIMIT_PER_MINUTE", "0")
	t.Setenv("PUBLIC_WRITE_LIMIT_PER_HOUR", "25")
	// 非数字输入应回退到敏感接口的默认每分钟 30 次。
	t.Setenv("SENSITIVE_RATE_LIMIT_PER_MINUTE", "invalid")

	cfg := Load()
	// 不仅检查数量，也检查顺序和值，确认清洗没有重排有效来源。
	if len(cfg.CORSAllowedOrigins) != 2 || cfg.CORSAllowedOrigins[0] != "https://portal.example.test" || cfg.CORSAllowedOrigins[1] != "https://admin.example.test" {
		t.Fatalf("CORSAllowedOrigins = %#v", cfg.CORSAllowedOrigins)
	}
	// 一次断言比较三个相关限流字段，失败消息按同样顺序打印实际值，便于定位。
	if cfg.AuthRateLimitPerMinute != 20 || cfg.PublicWriteLimitPerHour != 25 || cfg.SensitiveLimitPerMinute != 30 {
		t.Fatalf(
			"rate limits = %d/%d/%d",
			cfg.AuthRateLimitPerMinute,
			cfg.PublicWriteLimitPerHour,
			cfg.SensitiveLimitPerMinute,
		)
	}
}

// TestValidateStorageConfiguration 覆盖 local 与 s3 两种存储后端的有效基线及主要错误分支。
func TestValidateStorageConfiguration(t *testing.T) {
	// base 是满足通用校验条件的最小开发配置，后续所有变体都从它复制，避免无关字段干扰结果。
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

	// 从有效 local 配置复制出一份完整 S3 配置，先证明正向路径可用。
	s3 := base
	s3.StorageDriver = "s3"
	s3.S3Endpoint = "minio:9000"
	s3.S3AccessKey = "integration-user"
	s3.S3SecretKey = "integration-secret"
	s3.S3Bucket = "qutcraft-media"
	if err := s3.Validate(); err != nil {
		t.Fatalf("S3 storage config rejected: %v", err)
	}

	// 每个表项只破坏一个约束：驱动枚举、endpoint 格式，以及三项 S3 必填信息。
	for name, mutate := range map[string]func(*Config){
		"unknown driver":     func(cfg *Config) { cfg.StorageDriver = "ftp" },
		"endpoint scheme":    func(cfg *Config) { cfg.S3Endpoint = "http://minio:9000" },
		"missing access key": func(cfg *Config) { cfg.S3AccessKey = "" },
		"missing secret key": func(cfg *Config) { cfg.S3SecretKey = "" },
		"missing bucket":     func(cfg *Config) { cfg.S3Bucket = "" },
	} {
		t.Run(name, func(t *testing.T) {
			// Config 仅包含值类型和本测试不修改的切片，因此浅复制足以隔离这些变体。
			cfg := s3
			mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("invalid storage configuration was accepted")
			}
		})
	}
}

// TestValidateRejectsUnsafeProductionConfiguration 验证生产环境会拒绝开发占位值和高风险功能。
func TestValidateRejectsUnsafeProductionConfiguration(t *testing.T) {
	// 先建立一份能通过全部通用及生产检查的安全基线，确保后续失败确实由单项 mutate 引起。
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

	// 表驱动用例逐一注入不安全设置，覆盖 CORS、JWT、演示数据、管理员密码和 S3 凭据。
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
		// 使用子测试保留具体场景名，某个约束回归时可直接看出是哪条生产规则失效。
		t.Run(test.name, func(t *testing.T) {
			cfg := base
			test.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("unsafe production config was accepted")
			}
		})
	}
}

// TestValidateEmailConfiguration 验证 SMTP 的完整配置、字段级错误以及生产环境的加密要求。
func TestValidateEmailConfiguration(t *testing.T) {
	// base 包含启用 SMTP 所需的全部字段，作为正向用例及各负向变体的共同起点。
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
	// 每次只修改一个字段，以分别触发驱动、主机、端口、发件人、认证、安全模式和公开 URL 校验。
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

	// 开发环境允许显式使用 none，生产环境则必须拒绝明文 SMTP 传输。
	production := base
	production.AppEnv = "production"
	production.JWTAccessSecret = strings.Repeat("a", 48)
	production.SMTPSecurity = "none"
	if err := production.Validate(); err == nil {
		t.Fatal("unencrypted production SMTP was accepted")
	}
}

// TestLoadAndValidateAIConfiguration 同时覆盖 AI 环境变量的规范化、有效配置和非法组合。
func TestLoadAndValidateAIConfiguration(t *testing.T) {
	// 通过环境变量构造一组完整的 OpenAI 兼容服务配置；BaseURL 故意带尾随斜杠以测试清洗。
	t.Setenv("AI_PROVIDER", "openai_compatible")
	t.Setenv("AI_BASE_URL", "https://models.example.test/v1/")
	t.Setenv("AI_API_KEY", "test-only-provider-key")
	t.Setenv("AI_MODEL", "example-model")
	t.Setenv("AI_REQUEST_TIMEOUT", "45s")
	t.Setenv("AI_RUN_LIMIT_PER_HOUR", "12")

	// Load 应去除 BaseURL 尾随斜杠，同时原样保留已经规范的模型名称。
	cfg := Load()
	if cfg.AIBaseURL != "https://models.example.test/v1" || cfg.AIModel != "example-model" {
		t.Fatalf("AI configuration was not normalized: %+v", cfg)
	}
	// 时长和每小时次数应从字符串正确转换为强类型字段。
	if cfg.AIRequestTimeout.String() != "45s" || cfg.AIRunLimitPerHour != 12 {
		t.Fatalf("AI limits = %s/%d, want 45s/12", cfg.AIRequestTimeout, cfg.AIRunLimitPerHour)
	}

	// 手工构造的 base 用来单独测试 Validate，不依赖前半段的进程环境装载结果。
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
	// 外部 AI 提供方必须是受支持的驱动，并同时具备绝对 URL、API 密钥和模型名。
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

	// mock 仅供非生产环境使用；真实生产端点还必须从 HTTP 升级为 HTTPS。
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

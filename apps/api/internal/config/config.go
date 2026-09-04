// Package config 负责从环境变量装载 API 服务的运行配置，并在服务启动前校验配置的完整性与安全性。
//
// 本包刻意将“读取并规范化环境变量”和“检查配置是否合法”拆成 Load 与 Config.Validate 两步：
// 调用方既可以直接装载真实环境，也可以在测试中手工构造 Config，然后复用同一套校验规则。
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config 汇总 API 服务启动和运行期间使用的全部可配置参数。
// 字段值通常由 Load 从同名环境变量读取；敏感字段不会在此处做掩码，因此不要直接将整个结构体写入日志。
type Config struct {
	// AppEnv 标识运行环境，例如 development 或 production；生产环境会启用额外的安全校验。
	AppEnv string
	// HTTPAddr 是 HTTP 服务监听地址，采用 Go net/http 可识别的 host:port 形式。
	HTTPAddr string
	// MySQLDSN 是连接 MySQL 的数据源名称，其中可能包含数据库凭据，不应输出到日志。
	MySQLDSN string

	// JWTIssuer 写入 JWT 的 iss 声明，用于标识令牌签发方。
	JWTIssuer string
	// JWTAccessSecret 用于签名访问令牌；生产环境要求至少 32 个字符且不能使用已知占位值。
	JWTAccessSecret string
	// JWTAccessTTL 是短期访问令牌的有效期。
	JWTAccessTTL time.Duration
	// JWTRefreshTTL 是刷新令牌的有效期，通常显著长于访问令牌。
	JWTRefreshTTL time.Duration

	// RedisAddr 是 Redis 服务地址，格式通常为 host:port。
	RedisAddr string
	// RedisPassword 是 Redis 认证密码；允许为空，但属于敏感信息。
	RedisPassword string
	// RedisDB 是 Redis 逻辑数据库编号，只接受非负整数。
	RedisDB int
	// PublicCacheTTL 是公开数据写入缓存后的有效时长。
	PublicCacheTTL time.Duration

	// SuperbedEnabled 控制是否启用聚合图床上传能力。
	SuperbedEnabled bool
	// SuperbedToken 是调用聚合图床接口时使用的认证令牌。
	SuperbedToken string
	// SuperbedUploadURL 是聚合图床的上传接口地址。
	SuperbedUploadURL string
	// SuperbedTimeout 是单次聚合图床请求允许占用的最长时间。
	SuperbedTimeout time.Duration

	// StorageDriver 选择媒体存储后端，目前只支持 local 和 s3。
	StorageDriver string
	// StorageLocalRoot 是 local 后端保存上传文件的根目录。
	StorageLocalRoot string
	// S3Endpoint 是 S3 兼容服务的 host:port；协议由 S3UseSSL 单独控制，因此此字段不能带 URL scheme。
	S3Endpoint string
	// S3AccessKey 是访问 S3 兼容服务的访问密钥标识。
	S3AccessKey string
	// S3SecretKey 是访问 S3 兼容服务的秘密密钥，属于敏感信息。
	S3SecretKey string
	// S3Bucket 是存放媒体对象的桶名称。
	S3Bucket string
	// S3Region 是 S3 请求所使用的区域名称。
	S3Region string
	// S3UseSSL 决定连接 S3Endpoint 时是否启用 TLS。
	S3UseSSL bool

	// PublicWebBaseURL 是面向用户的 Web 站点绝对基址；Load 会移除末尾的斜杠，便于后续拼接路径。
	PublicWebBaseURL string

	// EmailDriver 选择邮件发送后端，目前支持 disabled 和 smtp。
	EmailDriver string
	// SMTPHost 是 SMTP 服务器主机名或 IP 地址。
	SMTPHost string
	// SMTPPort 是 SMTP 服务端口，启用 SMTP 时必须位于 1～65535。
	SMTPPort int
	// SMTPUsername 是可选的 SMTP 登录用户名。
	SMTPUsername string
	// SMTPPassword 是 SMTP 登录密码；设置用户名时必须同时提供该字段。
	SMTPPassword string
	// SMTPFromAddress 是邮件 From 头中的发件地址；Load 会去除空白并转为小写。
	SMTPFromAddress string
	// SMTPFromName 是邮件 From 头中展示给收件人的发件人名称。
	SMTPFromName string
	// SMTPSecurity 指定 SMTP 传输安全模式，可选 starttls、tls 或 none。
	SMTPSecurity string
	// SMTPTimeout 是一次 SMTP 操作允许占用的最长时间。
	SMTPTimeout time.Duration

	// AIProvider 选择 AI 实现，可为 disabled、mock 或 openai_compatible；空字符串等同于未启用。
	AIProvider string
	// AIBaseURL 是 OpenAI 兼容接口的绝对基址；Load 会去掉末尾斜杠。
	AIBaseURL string
	// AIAPIKey 是调用 AI 服务的认证密钥，属于敏感信息。
	AIAPIKey string
	// AIModel 是提交给 AI 服务的模型标识。
	AIModel string
	// AIRequestTimeout 是单次 AI 请求的超时时间。
	AIRequestTimeout time.Duration
	// AIRunLimitPerHour 是每小时允许发起的 AI 运行次数，必须为正数。
	AIRunLimitPerHour int

	// CORSAllowedOrigins 是允许携带凭据访问 API 的 Web 源列表，因此明确禁止通配符 "*"。
	CORSAllowedOrigins []string
	// AuthRateLimitPerMinute 是认证相关接口每分钟的请求上限。
	AuthRateLimitPerMinute int
	// PublicWriteLimitPerHour 是公开写接口每小时的请求上限。
	PublicWriteLimitPerHour int
	// SensitiveLimitPerMinute 是其他敏感接口每分钟的请求上限。
	SensitiveLimitPerMinute int

	// DefaultOrganizationSlug 是未明确指定组织时使用的默认组织短标识。
	DefaultOrganizationSlug string
	// BootstrapAdminEmail 是首次引导创建管理员时使用的邮箱；Load 会规范化为空白剔除后的小写形式。
	BootstrapAdminEmail string
	// BootstrapAdminPassword 是首次引导创建管理员时使用的密码，非空时至少需要 12 个字符。
	BootstrapAdminPassword string
	// BootstrapAdminName 是首次引导创建的管理员显示名称。
	BootstrapAdminName string

	// DemoSeedEnabled 控制是否写入演示数据；为避免污染真实数据，生产环境强制禁用。
	DemoSeedEnabled bool
	// DemoSeedProfile 选择演示数据模板，目前只接受 qutcraft 或 generic。
	DemoSeedProfile string
	// DemoSeedMultiOrganization 控制演示数据是否覆盖多个组织；生产环境强制禁用。
	DemoSeedMultiOrganization bool
}

// Load 从当前进程环境变量构造 Config。
//
// 每个配置项都提供适合本地开发的默认值。辅助函数负责去除首尾空白、解析类型以及在非法输入时
// 回退到默认值；枚举类字符串在此处统一转成小写，从而让后续 Validate 的比较保持简单且一致。
// Load 只负责读取和规范化，不保证组合后的配置一定可用，调用方仍应在启动服务前调用 Validate。
func Load() Config {
	return Config{
		// 基础服务与数据依赖。
		AppEnv:   value("APP_ENV", "development"),
		HTTPAddr: value("HTTP_ADDR", ":8080"),
		MySQLDSN: value("MYSQL_DSN", "qutcraft:qutcraft@tcp(localhost:3306)/qutcraft?charset=utf8mb4&parseTime=True&loc=UTC"),

		// 认证令牌：访问令牌默认 15 分钟，刷新令牌默认 7 天。
		JWTIssuer:       value("JWT_ISSUER", "qutcraft-platform"),
		JWTAccessSecret: value("JWT_ACCESS_SECRET", "development-only-change-me-before-production"),
		JWTAccessTTL:    duration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:   duration("JWT_REFRESH_TTL", 7*24*time.Hour),

		// Redis 和公开数据缓存。
		RedisAddr:      value("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  os.Getenv("REDIS_PASSWORD"),
		RedisDB:        integer("REDIS_DB", 0),
		PublicCacheTTL: duration("PUBLIC_CACHE_TTL", 30*time.Second),

		// 聚合图床配置；令牌去除意外空白，但不会提供不安全的默认凭据。
		SuperbedEnabled:   boolean("SUPERBED_ENABLED", false),
		SuperbedToken:     strings.TrimSpace(os.Getenv("SUPERBED_TOKEN")),
		SuperbedUploadURL: value("SUPERBED_UPLOAD_URL", "https://api.superbed.cn/upload"),
		SuperbedTimeout:   duration("SUPERBED_TIMEOUT", 30*time.Second),

		// 媒体存储配置；驱动名转小写以支持 LOCAL、S3 等大小写变体。
		StorageDriver:    strings.ToLower(value("STORAGE_DRIVER", "local")),
		StorageLocalRoot: value("STORAGE_LOCAL_ROOT", "/tmp/qutcraft-uploads"),
		S3Endpoint:       value("S3_ENDPOINT", "minio:9000"),
		S3AccessKey:      strings.TrimSpace(os.Getenv("S3_ACCESS_KEY")),
		S3SecretKey:      os.Getenv("S3_SECRET_KEY"),
		S3Bucket:         value("S3_BUCKET", "qutcraft-media"),
		S3Region:         value("S3_REGION", "us-east-1"),
		S3UseSSL:         boolean("S3_USE_SSL", false),

		// 对外 Web 基址去掉所有尾随斜杠，避免拼接路径时产生双斜杠。
		PublicWebBaseURL: strings.TrimRight(value("PUBLIC_WEB_BASE_URL", "http://localhost:8082"), "/"),

		// 邮件配置；驱动与安全模式均转成小写，地址字段按各自语义规范化。
		EmailDriver:     strings.ToLower(value("EMAIL_DRIVER", "disabled")),
		SMTPHost:        strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:        integer("SMTP_PORT", 587),
		SMTPUsername:    strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:    os.Getenv("SMTP_PASSWORD"),
		SMTPFromAddress: strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_FROM_ADDRESS"))),
		SMTPFromName:    value("SMTP_FROM_NAME", "QUTCraft Commons"),
		SMTPSecurity:    strings.ToLower(value("SMTP_SECURITY", "starttls")),
		SMTPTimeout:     duration("SMTP_TIMEOUT", 8*time.Second),

		// AI 配置；BaseURL 既去除外围空白，也移除尾随斜杠。
		AIProvider:        strings.ToLower(value("AI_PROVIDER", "disabled")),
		AIBaseURL:         strings.TrimRight(strings.TrimSpace(os.Getenv("AI_BASE_URL")), "/"),
		AIAPIKey:          os.Getenv("AI_API_KEY"),
		AIModel:           strings.TrimSpace(os.Getenv("AI_MODEL")),
		AIRequestTimeout:  duration("AI_REQUEST_TIMEOUT", 30*time.Second),
		AIRunLimitPerHour: positiveInteger("AI_RUN_LIMIT_PER_HOUR", 20),

		// Web 安全与限流。正整数解析器会拒绝 0，避免因误配置而关闭限流。
		CORSAllowedOrigins:      csv(value("CORS_ALLOWED_ORIGINS", "http://localhost:8082,http://127.0.0.1:8082,http://localhost,http://127.0.0.1")),
		AuthRateLimitPerMinute:  positiveInteger("AUTH_RATE_LIMIT_PER_MINUTE", 20),
		PublicWriteLimitPerHour: positiveInteger("PUBLIC_WRITE_LIMIT_PER_HOUR", 10),
		SensitiveLimitPerMinute: positiveInteger("SENSITIVE_RATE_LIMIT_PER_MINUTE", 30),

		// 默认组织、首次管理员和演示数据。
		DefaultOrganizationSlug:   value("DEFAULT_ORGANIZATION_SLUG", "qutcraft"),
		BootstrapAdminEmail:       strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL"))),
		BootstrapAdminPassword:    os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		BootstrapAdminName:        value("BOOTSTRAP_ADMIN_NAME", "QUTCraft Admin"),
		DemoSeedEnabled:           boolean("DEMO_SEED_ENABLED", false),
		DemoSeedProfile:           strings.ToLower(value("DEMO_SEED_PROFILE", "qutcraft")),
		DemoSeedMultiOrganization: boolean("DEMO_SEED_MULTI_ORGANIZATION", false),
	}
}

// Validate 检查字段间约束以及生产环境特有的安全要求。
// 返回的第一个错误使用对应环境变量名描述问题，便于部署人员直接定位配置来源。
// 此方法不修改 Config，因此同一个配置值可以被安全地重复校验。
func (c Config) Validate() error {
	// 空 profile 可用于手工构造的最小配置；非空时只允许两个已实现的演示模板。
	if c.DemoSeedProfile != "" && c.DemoSeedProfile != "qutcraft" && c.DemoSeedProfile != "generic" {
		return fmt.Errorf("DEMO_SEED_PROFILE must be qutcraft or generic")
	}

	// 每种存储后端拥有不同的必填字段，先按驱动分支检查，避免要求无关配置。
	switch c.StorageDriver {
	case "local":
		// 本地存储必须有明确根目录，否则无法确定上传文件的落盘位置。
		if strings.TrimSpace(c.StorageLocalRoot) == "" {
			return fmt.Errorf("STORAGE_LOCAL_ROOT is required when STORAGE_DRIVER=local")
		}
	case "s3":
		// S3 客户端分别接收 endpoint 与 TLS 开关，所以 endpoint 必须是无协议头的 host:port。
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

	// 对外链接可能出现在邮件或 API 响应中，因此必须使用可独立解析的 HTTP(S) 绝对地址。
	if !strings.HasPrefix(c.PublicWebBaseURL, "http://") && !strings.HasPrefix(c.PublicWebBaseURL, "https://") {
		return fmt.Errorf("PUBLIC_WEB_BASE_URL must be an absolute http or https URL")
	}

	// 禁用邮件时无需 SMTP 参数；只有 smtp 分支才检查服务器、认证和传输安全配置。
	switch c.EmailDriver {
	case "disabled":
		// 显式保留空分支，表示 disabled 是合法且无需额外字段的驱动。
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

	// disabled、空值和 mock 都不访问外部模型；兼容 OpenAI 的实现则需要完整连接信息。
	switch c.AIProvider {
	case "", "disabled", "mock":
		// 这些模式没有外部端点或凭据依赖。
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

	// 至少显式声明一个来源，防止出现“未配置即意外放行”或服务无法被任何前端访问的歧义。
	if len(c.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must contain at least one origin")
	}
	for _, origin := range c.CORSAllowedOrigins {
		// 服务允许凭据跨域；按 CORS 规范，此场景不能把通配符与凭据组合使用。
		if origin == "*" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS cannot contain wildcard when credentials are enabled")
		}
		if !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
			return fmt.Errorf("invalid CORS origin %q", origin)
		}
	}

	// 密码为空表示不通过该配置引导管理员；一旦提供，就执行最低长度要求。
	if c.BootstrapAdminPassword != "" && len(c.BootstrapAdminPassword) < 12 {
		return fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD must be at least 12 characters")
	}

	// 生产环境额外排除开发占位凭据、不加密连接和演示功能，避免可运行但不安全的部署。
	if strings.EqualFold(c.AppEnv, "production") {
		// JWT 密钥不仅要足够长，还必须排除仓库默认值和常见 replace-with 占位写法。
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
			// MinIO 默认账号和带 change-me 的秘密密钥只能用于开发环境。
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

// boolean 读取布尔环境变量，并兼容部署配置中常见的多种真假写法。
// 空值或无法识别的值不会猜测其含义，而是返回 fallback。
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

// integer 读取非负十进制整数。空值、解析失败或负数都会回退到 fallback。
// fmt.Sscanf 在这里与既有配置格式保持兼容；业务上必须大于零的字段由 positiveInteger 再约束。
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

// positiveInteger 在 integer 的非负约束上进一步拒绝 0，适用于速率上限等必须为正的配置。
func positiveInteger(key string, fallback int) int {
	value := integer(key, fallback)
	if value <= 0 {
		return fallback
	}
	return value
}

// csv 将逗号分隔字符串转换为切片，同时去除每项空白并丢弃空项。
// 它保留有效项的原始顺序，也不会自动去重，以便配置行为与输入顺序保持一致。
func csv(raw string) []string {
	values := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

// value 读取并去除字符串环境变量两端空白；变量缺失或仅含空白时返回 fallback。
func value(key, fallback string) string {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		return raw
	}
	return fallback
}

// duration 使用 time.ParseDuration 解析 Go 风格时长（例如 500ms、30s、2h）。
// 解析失败、零值或负值均使用 fallback，避免产生立即过期或永不正常执行的超时配置。
func duration(key string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(value(key, fallback.String()))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

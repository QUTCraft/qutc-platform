package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type contextKey string

const requestIDKey contextKey = "qutc.request_id"

// Init 根据环境初始化全局 slog logger。生产环境输出 JSON，开发环境输出纯文本。
func Init(env string) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(newSanitizingHandler(handler)))
}

// Ctx 返回从 context 中提取了 request_id 的 logger。
func Ctx(ctx context.Context) *slog.Logger {
	requestID, _ := ctx.Value(requestIDKey).(string)
	logger := slog.Default()
	if requestID != "" {
		logger = logger.With("request_id", requestID)
	}
	return logger
}

// WithRequestID 把 request_id 注入 context，供后续 Ctx(ctx) 提取。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// ---------------------------------------------------------------------------
// 敏感字段脱敏 handler
// ---------------------------------------------------------------------------

// 匹配方式为 key 名包含任一下列片段（不区分大小写）。
var sensitiveKeyFragments = []string{
	"password", "passwd", "secret", "token", "authorization",
	"auth_code", "smtp_password", "rcon_password", "minio_secret",
	"jwt_secret", "api_key", "private_key",
}

type sanitizingHandler struct {
	next slog.Handler
}

func newSanitizingHandler(next slog.Handler) *sanitizingHandler {
	return &sanitizingHandler{next: next}
}

func (h *sanitizingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *sanitizingHandler) Handle(ctx context.Context, record slog.Record) error {
	redacted := slog.Record{Time: record.Time, Level: record.Level, Message: record.Message, PC: record.PC}
	record.Attrs(func(attr slog.Attr) bool {
		redacted.AddAttrs(sanitizeAttr(attr))
		return true
	})
	return h.next.Handle(ctx, redacted)
}

func (h *sanitizingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	sanitized := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		sanitized[i] = sanitizeAttr(attr)
	}
	return newSanitizingHandler(h.next.WithAttrs(sanitized))
}

func (h *sanitizingHandler) WithGroup(name string) slog.Handler {
	return newSanitizingHandler(h.next.WithGroup(name))
}

func sanitizeAttr(attr slog.Attr) slog.Attr {
	if attr.Value.Kind() == slog.KindString && isSensitiveKey(attr.Key) {
		return slog.String(attr.Key, "[REDACTED]")
	}
	return attr
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, fragment := range sensitiveKeyFragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

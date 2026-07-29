package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/logging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID 在每个请求入口注入或生成 request_id，写入 gin.Context 响应头和
// context.Context，确保后续所有 handler、service 和日志调用都能获取。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Request = c.Request.WithContext(logging.WithRequestID(c.Request.Context(), requestID))
		c.Next()
	}
}

// SlogLogger 是替换 gin.Logger() 的结构化日志中间件。
// 每条请求日志包含 method、path、status、latency、client_ip、request_id 和 error（如有）。
func SlogLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		logger := logging.Ctx(c.Request.Context())
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.String("latency", latency.String()),
			slog.String("client_ip", c.ClientIP()),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("error", c.Errors.String()))
		}
		level := slog.LevelInfo
		if c.Writer.Status() >= http.StatusInternalServerError {
			level = slog.LevelError
		} else if c.Writer.Status() >= http.StatusBadRequest {
			level = slog.LevelWarn
		}
		logger.LogAttrs(c.Request.Context(), level, "http", attrs...)
	}
}

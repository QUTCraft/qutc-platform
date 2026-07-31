package middleware

import (
	"log/slog"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDKey = "qutc.request_id"

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)

// EnsureRequestID returns the request correlation ID shared by responses,
// audit events, and structured access logs. Unsafe caller-provided values are
// replaced so they cannot forge or corrupt log records.
func EnsureRequestID(c *gin.Context) string {
	if value, ok := c.Get(requestIDKey); ok {
		if requestID, valid := value.(string); valid && requestIDPattern.MatchString(requestID) {
			return requestID
		}
	}

	requestID := c.GetHeader("X-Request-ID")
	if !requestIDPattern.MatchString(requestID) {
		requestID = uuid.NewString()
	}
	c.Set(requestIDKey, requestID)
	c.Request.Header.Set("X-Request-ID", requestID)
	c.Header("X-Request-ID", requestID)
	return requestID
}

func StructuredLogger(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		attributes := []any{
			"event", "http_request",
			"request_id", EnsureRequestID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"route", route,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(startedAt).Milliseconds(),
			"client_ip", c.ClientIP(),
		}
		if principal, ok := PrincipalFromContext(c); ok {
			attributes = append(attributes,
				"user_id", principal.UserID,
				"organization_id", principal.OrganizationID,
			)
		}

		switch status := c.Writer.Status(); {
		case status >= 500:
			logger.Error("http request completed", attributes...)
		case status >= 400:
			logger.Warn("http request completed", attributes...)
		default:
			logger.Info("http request completed", attributes...)
		}
	}
}

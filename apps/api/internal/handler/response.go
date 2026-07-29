package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func respond(c *gin.Context, status int, data interface{}) {
	respondWithMeta(c, status, data, gin.H{})
}

func respondWithMeta(c *gin.Context, status int, data interface{}, meta gin.H) {
	requestID := ensureRequestID(c)
	meta["request_id"] = requestID
	c.JSON(status, gin.H{"data": data, "meta": meta})
}

func fail(c *gin.Context, status int, code, message string) {
	requestID := ensureRequestID(c)
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": requestID}})
}

func failWithDetails(c *gin.Context, status int, code, message string, details interface{}) {
	requestID := ensureRequestID(c)
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "details": details, "request_id": requestID}})
}

func ensureRequestID(c *gin.Context) string {
	if id, ok := c.Get("request_id"); ok {
		if s, ok2 := id.(string); ok2 {
			c.Header("X-Request-ID", s)
			return s
		}
	}
	id := c.GetHeader("X-Request-ID")
	if id == "" {
		id = uuid.NewString()
		c.Set("request_id", id)
	}
	c.Header("X-Request-ID", id)
	return id
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// HealthHandler 提供健康检查端点，包含 DB 和 Redis 依赖探活。
type HealthHandler struct {
	db    interface{ Ping() error }
	cache interface{ Ping() error }
}

func NewHealthHandler(db interface{ Ping() error }, cache interface{ Ping() error }) *HealthHandler {
	return &HealthHandler{db: db, cache: cache}
}

// Healthz 检查所有关键依赖是否可达。任意依赖不可用时返回 503。
func (h *HealthHandler) Healthz(c *gin.Context) {
	checks := make(map[string]string, 2)
	healthy := true

	if err := h.db.Ping(); err != nil {
		checks["database"] = err.Error()
		healthy = false
	} else {
		checks["database"] = "ok"
	}

	if err := h.cache.Ping(); err != nil {
		checks["cache"] = err.Error()
		healthy = false
	} else {
		checks["cache"] = "ok"
	}

	if healthy {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "checks": checks})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "checks": checks})
	}
}

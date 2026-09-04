// health.go 提供容器编排和负载均衡使用的存活与就绪探针。
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/cache"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler 持有 MySQL 与 Redis 客户端，用于执行就绪检查。
type HealthHandler struct {
	db    *gorm.DB
	cache *cache.Cache
}

// NewHealthHandler 创建健康检查处理器。
func NewHealthHandler(db *gorm.DB, publicCache *cache.Cache) *HealthHandler {
	return &HealthHandler{db: db, cache: publicCache}
}

// Liveness 只表示进程仍能响应请求，不检查外部依赖，因此适合作为存活探针。
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readiness 在两秒超时上下文内检查 MySQL 和 Redis；任一依赖不可用都会返回 503。
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{"mysql": "unavailable", "redis": "unavailable"}
	ready := true

	if h.db == nil {
		ready = false
	} else if sqlDB, err := h.db.DB(); err != nil {
		ready = false
	} else if err := sqlDB.PingContext(ctx); err != nil {
		ready = false
	} else {
		checks["mysql"] = "ok"
	}

	if h.cache == nil || h.cache.Ping(ctx) != nil {
		ready = false
	} else {
		checks["redis"] = "ok"
	}

	if !ready {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "checks": checks})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready", "checks": checks})
}

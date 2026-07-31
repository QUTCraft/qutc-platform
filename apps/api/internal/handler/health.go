package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/cache"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db    *gorm.DB
	cache *cache.Cache
}

func NewHealthHandler(db *gorm.DB, publicCache *cache.Cache) *HealthHandler {
	return &HealthHandler{db: db, cache: publicCache}
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

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

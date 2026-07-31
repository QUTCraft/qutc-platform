package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuditHandler struct {
	db *gorm.DB
}

type auditEventRow struct {
	ID             string    `json:"id"`
	ActorUserID    string    `json:"actor_user_id"`
	ActorName      string    `json:"actor_name"`
	Action         string    `json:"action"`
	TargetType     string    `json:"target_type"`
	TargetID       string    `json:"target_id"`
	Result         string    `json:"result"`
	RequestID      string    `json:"request_id"`
	CreatedAt      time.Time `json:"created_at"`
	OrganizationID string    `json:"-"`
}

func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

func (h *AuditHandler) List(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	page, pageSize, ok := listMeta(c, 0)
	if !ok {
		return
	}

	action, ok := queryMax(c, "action", 96)
	if !ok {
		return
	}
	targetType, ok := queryMax(c, "target_type", 64)
	if !ok {
		return
	}
	result, ok := queryMax(c, "result", 24)
	if !ok {
		return
	}
	if result != "" && result != "success" && result != "accepted" && result != "failed" {
		fail(c, http.StatusBadRequest, "audit.invalid_result_filter", "result 仅支持 success、accepted 或 failed。")
		return
	}
	actorUserID, ok := queryMax(c, "actor_user_id", 36)
	if !ok {
		return
	}
	requestID, ok := queryMax(c, "request_id", 64)
	if !ok {
		return
	}
	dateFrom, hasDateFrom, ok := auditDate(c, "date_from")
	if !ok {
		return
	}
	dateTo, hasDateTo, ok := auditDate(c, "date_to")
	if !ok {
		return
	}
	if hasDateFrom && hasDateTo && dateTo.Before(dateFrom) {
		fail(c, http.StatusBadRequest, "audit.invalid_date_range", "date_to 不能早于 date_from。")
		return
	}

	query := h.db.Table("audit_events AS audit").
		Where("audit.organization_id = ?", principal.OrganizationID)
	if action != "" {
		query = query.Where("audit.action = ?", action)
	}
	if targetType != "" {
		query = query.Where("audit.target_type = ?", targetType)
	}
	if result != "" {
		query = query.Where("audit.result = ?", result)
	}
	if actorUserID != "" {
		query = query.Where("audit.actor_user_id = ?", actorUserID)
	}
	if requestID != "" {
		query = query.Where("audit.request_id = ?", requestID)
	}
	if hasDateFrom {
		query = query.Where("audit.created_at >= ?", dateFrom)
	}
	if hasDateTo {
		query = query.Where("audit.created_at < ?", dateTo.AddDate(0, 0, 1))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "audit.list_failed", "审计记录暂时无法加载。")
		return
	}

	var events []auditEventRow
	err := query.
		Select("audit.id, audit.organization_id, audit.actor_user_id, users.display_name AS actor_name, audit.action, audit.target_type, audit.target_id, audit.result, audit.request_id, audit.created_at").
		Joins("LEFT JOIN users ON users.id = audit.actor_user_id").
		Order("audit.created_at DESC, audit.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&events).Error
	if err != nil {
		fail(c, http.StatusInternalServerError, "audit.list_failed", "审计记录暂时无法加载。")
		return
	}
	respondWithMeta(c, http.StatusOK, events, gin.H{"page": page, "page_size": pageSize, "total": total})
}

func auditDate(c *gin.Context, key string) (time.Time, bool, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return time.Time{}, false, true
	}
	value, err := time.Parse("2006-01-02", raw)
	if err != nil {
		fail(c, http.StatusBadRequest, "audit.invalid_date", key+" 必须使用 YYYY-MM-DD 格式。")
		return time.Time{}, false, false
	}
	return value.UTC(), true, true
}

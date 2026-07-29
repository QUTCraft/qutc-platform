package handler

import (
	"net/http"
	"strings"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AuditHandler 暴露组织范围内的审计事件查询能力，仅支持读取。
type AuditHandler struct {
	db *gorm.DB
}

func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

type auditEventItem struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	Result      string `json:"result"`
	RequestID   string `json:"request_id"`
	ActorUserID string `json:"actor_user_id"`
	ActorName   string `json:"actor_name"`
	CreatedAt   string `json:"created_at"`
}

// AdminAuditEvents 返回当前组织范围内的审计事件，支持按操作、目标、结果、
// request_id 和操作者筛选。它只读取已落库的审计事件，不暴露其他组织的记录。
func (h *AuditHandler) AdminAuditEvents(c *gin.Context) {
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
	result := strings.TrimSpace(c.Query("result"))
	if result != "" && result != "success" && result != "accepted" && result != "failed" {
		fail(c, http.StatusBadRequest, "audit.invalid_result_filter", "result 仅支持 success、accepted 或 failed。")
		return
	}
	requestID, ok := queryMax(c, "request_id", 64)
	if !ok {
		return
	}
	actorUserID, ok := queryMax(c, "actor_user_id", 64)
	if !ok {
		return
	}

	query := h.db.Model(&model.AuditEvent{}).Where("organization_id = ?", principal.OrganizationID)
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	if result != "" {
		query = query.Where("result = ?", result)
	}
	if requestID != "" {
		query = query.Where("request_id = ?", requestID)
	}
	if actorUserID != "" {
		query = query.Where("actor_user_id = ?", actorUserID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "audit.list_failed", "审计记录暂时无法加载。")
		return
	}
	var events []model.AuditEvent
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&events).Error; err != nil {
		fail(c, http.StatusInternalServerError, "audit.list_failed", "审计记录暂时无法加载。")
		return
	}

	actorIDs := make(map[string]struct{}, len(events))
	for _, event := range events {
		if event.ActorUserID != "" {
			actorIDs[event.ActorUserID] = struct{}{}
		}
	}
	names := make(map[string]string, len(actorIDs))
	if len(actorIDs) > 0 {
		ids := make([]string, 0, len(actorIDs))
		for id := range actorIDs {
			ids = append(ids, id)
		}
		var users []model.User
		if err := h.db.Select("id, display_name").Where("id IN ?", ids).Find(&users).Error; err == nil {
			for _, user := range users {
				names[user.ID] = user.DisplayName
			}
		}
	}

	items := make([]auditEventItem, 0, len(events))
	for _, event := range events {
		items = append(items, auditEventItem{
			ID:          event.ID,
			Action:      event.Action,
			TargetType:  event.TargetType,
			TargetID:    event.TargetID,
			Result:      event.Result,
			RequestID:   event.RequestID,
			ActorUserID: event.ActorUserID,
			ActorName:   names[event.ActorUserID],
			CreatedAt:   event.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	respondWithMeta(c, http.StatusOK, items, gin.H{"page": page, "page_size": pageSize, "total": total})
}

// notifications.go 提供邀请邮件模板、通知发件箱查询和重试接口。
package handler

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationHandler 组合数据库和通知服务，处理组织级邮件通知管理。
type NotificationHandler struct {
	db            *gorm.DB
	notifications *service.NotificationService
}

// invitationTemplateRequest 保存邀请邮件主题和正文模板。
type invitationTemplateRequest struct {
	SubjectTemplate string `json:"subject_template"`
	BodyTemplate    string `json:"body_template"`
}

// invitationTemplateVariablePattern 用于识别模板中的双大括号变量。
var invitationTemplateVariablePattern = regexp.MustCompile(`\{\{([a-z_]+)\}\}`)

// NewNotificationHandler 创建通知管理处理器。
func NewNotificationHandler(db *gorm.DB, notifications *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{db: db, notifications: notifications}
}

// GetInvitationTemplate 返回当前组织保存的邀请邮件模板及可用变量信息。
func (h *NotificationHandler) GetInvitationTemplate(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var organization model.Organization
	if err := h.db.Where("id = ?", principal.OrganizationID).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "organization.not_found", "组织不存在。")
		return
	}
	respond(c, http.StatusOK, invitationTemplateResponse(organization))
}

// UpdateInvitationTemplate 校验模板长度和变量白名单后，在事务中保存并写入审计事件。
func (h *NotificationHandler) UpdateInvitationTemplate(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var body invitationTemplateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "notification.template_invalid", "邮件模板格式不正确。")
		return
	}
	body.SubjectTemplate = strings.TrimSpace(body.SubjectTemplate)
	body.BodyTemplate = strings.TrimSpace(body.BodyTemplate)
	if len([]rune(body.SubjectTemplate)) > 255 || len([]rune(body.BodyTemplate)) > 4000 || !validInvitationTemplate(body.SubjectTemplate) || !validInvitationTemplate(body.BodyTemplate) {
		fail(c, http.StatusBadRequest, "notification.template_invalid", "模板长度或变量不符合规范。可用变量：organization、role、invite_url、expires_at。")
		return
	}
	var organization model.Organization
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", principal.OrganizationID).First(&organization).Error; err != nil {
			return err
		}
		if err := tx.Model(&organization).Updates(map[string]any{
			"invitation_subject_template": body.SubjectTemplate,
			"invitation_body_template":    body.BodyTemplate,
		}).Error; err != nil {
			return err
		}
		organization.InvitationSubjectTemplate = body.SubjectTemplate
		organization.InvitationBodyTemplate = body.BodyTemplate
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "notification.invitation_template_update", "organization", organization.ID)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "organization.not_found", "组织不存在。")
			return
		}
		fail(c, http.StatusInternalServerError, "notification.template_save_failed", "邮件模板暂时无法保存。")
		return
	}
	respond(c, http.StatusOK, invitationTemplateResponse(organization))
}

// ListOutbox 分页列出当前组织的通知发件箱记录。
func (h *NotificationHandler) ListOutbox(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	page, pageSize, ok := listMeta(c, 0)
	if !ok {
		return
	}
	items, total, err := h.notifications.List(principal.OrganizationID, strings.TrimSpace(c.Query("status")), page, pageSize)
	if err != nil {
		if errors.Is(err, service.ErrNotificationInvalidStatus) {
			fail(c, http.StatusBadRequest, "notification.invalid_status", "通知状态筛选值不符合要求。")
			return
		}
		fail(c, http.StatusInternalServerError, "notification.list_failed", "通知队列暂时无法加载。")
		return
	}
	respondWithMeta(c, http.StatusOK, items, gin.H{"page": page, "page_size": pageSize, "total": total})
}

// RetryOutbox 触发指定失败通知的再次投递。
func (h *NotificationHandler) RetryOutbox(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	item, err := h.notifications.Retry(principal.OrganizationID, c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotificationNotFound):
			fail(c, http.StatusNotFound, "notification.not_found", "通知记录不存在。")
		case errors.Is(err, service.ErrNotificationNotRetryable):
			fail(c, http.StatusConflict, "notification.not_retryable", "当前通知状态不可重试。")
		default:
			fail(c, http.StatusInternalServerError, "notification.retry_failed", "通知重试暂时无法提交。")
		}
		return
	}
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "notification.retry", TargetType: "notification_outbox", TargetID: item.ID, Result: "accepted", RequestID: ensureRequestID(c)}).Error
	respond(c, http.StatusOK, item)
}

// invitationTemplateResponse 生成模板 API 的稳定响应结构。
func invitationTemplateResponse(organization model.Organization) gin.H {
	return gin.H{
		"subject_template": organization.InvitationSubjectTemplate,
		"body_template":    organization.InvitationBodyTemplate,
		"variables":        []string{"organization", "role", "invite_url", "expires_at"},
	}
}

// validInvitationTemplate 确保模板变量只使用系统支持的四类邀请上下文变量。
func validInvitationTemplate(value string) bool {
	for _, match := range invitationTemplateVariablePattern.FindAllStringSubmatch(value, -1) {
		if len(match) != 2 {
			return false
		}
		switch match[1] {
		case "organization", "role", "invite_url", "expires_at":
		default:
			return false
		}
	}
	remaining := invitationTemplateVariablePattern.ReplaceAllString(value, "")
	return !strings.Contains(remaining, "{{") && !strings.Contains(remaining, "}}")
}

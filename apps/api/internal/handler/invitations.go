// invitations.go 实现组织邀请的创建、批量创建、列表、撤销、邮件投递重试和接受注册流程。
// 邮件适配器通过接口注入，便于按组织动态解析接入配置并在测试中替换为静态实现。
package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/mailadapter"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InvitationHandler 组合邀请所需的数据库、认证服务和组织级外部接入解析器。
type InvitationHandler struct {
	db           *gorm.DB
	auth         *service.AuthService
	integrations InvitationIntegrationResolver
}

// InvitationIntegrationResolver 按组织解析邮件发送器和公开 Web 基址。
type InvitationIntegrationResolver interface {
	MailSender(context.Context, string) (mailadapter.Sender, error)
	PublicWebBaseURL(context.Context, string) string
}

// staticInvitationIntegrations 是兼容旧构造函数的固定接入实现。
type staticInvitationIntegrations struct {
	mail             mailadapter.Sender
	publicWebBaseURL string
}

// MailSender 返回预先注入的固定邮件发送器。
func (s staticInvitationIntegrations) MailSender(context.Context, string) (mailadapter.Sender, error) {
	return s.mail, nil
}

// PublicWebBaseURL 返回预先注入的公开站点基址。
func (s staticInvitationIntegrations) PublicWebBaseURL(context.Context, string) string {
	return s.publicWebBaseURL
}

// NewInvitationHandler 使用固定邮件发送器创建邀请处理器，保留向后兼容的调用方式。
func NewInvitationHandler(db *gorm.DB, auth *service.AuthService, mail mailadapter.Sender, publicWebBaseURL string) *InvitationHandler {
	return NewInvitationHandlerWithIntegrations(db, auth, staticInvitationIntegrations{mail: mail, publicWebBaseURL: strings.TrimRight(publicWebBaseURL, "/")})
}

// NewInvitationHandlerWithIntegrations 创建支持按组织动态解析接入配置的邀请处理器。
func NewInvitationHandlerWithIntegrations(db *gorm.DB, auth *service.AuthService, integrations InvitationIntegrationResolver) *InvitationHandler {
	return &InvitationHandler{db: db, auth: auth, integrations: integrations}
}

// createInvitationRequest 描述单个邀请的邮箱、角色和可选有效期。
type createInvitationRequest struct {
	Email          string `json:"email" binding:"required,email,max=254"`
	Role           string `json:"role" binding:"required"`
	ExpiresInHours int    `json:"expires_in_hours"`
}

// batchInvitationRequest 包含批量创建时提交的邀请列表。
type batchInvitationRequest struct {
	Invitations []createInvitationRequest `json:"invitations" binding:"required"`
}

// batchInvitationResult 保留批量输入索引，并允许同一批次部分成功。
type batchInvitationResult struct {
	Index      int                 `json:"index"`
	Email      string              `json:"email"`
	Succeeded  bool                `json:"succeeded"`
	Invitation *invitationResponse `json:"invitation,omitempty"`
	Error      *batchItemError     `json:"error,omitempty"`
}

// batchItemError 是批量项目失败时返回的错误码和人类可读消息。
type batchItemError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// invitationFailure 是内部创建流程到 HTTP 响应之间的错误映射载体。
type invitationFailure struct {
	HTTPStatus int
	Code       string
	Message    string
}

// invitationResponse 将邀请视图与可选邀请链接、邮件投递状态组合返回。
type invitationResponse struct {
	service.InvitationView
	InviteURL string                `json:"invite_url,omitempty"`
	Delivery  emailDeliveryResponse `json:"delivery"`
}

// emailDeliveryResponse 描述邮件适配器、投递状态、尝试次数和错误信息。
type emailDeliveryResponse struct {
	Status        string     `json:"status"`
	Adapter       string     `json:"adapter"`
	Attempts      int        `json:"attempts"`
	LastError     string     `json:"last_error,omitempty"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
}

// Create 校验并创建单个组织邀请，随后尝试发送邀请邮件。
func (h *InvitationHandler) Create(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request createInvitationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "invitation.validation_failed", "邮箱、角色或邀请有效期不符合要求。")
		return
	}
	response, failure := h.createInvitation(c, principal, request)
	if failure != nil {
		fail(c, failure.HTTPStatus, failure.Code, failure.Message)
		return
	}
	respond(c, http.StatusCreated, response)
}

// CreateBatch 逐项处理邀请列表，返回成功项和失败项而不因单项错误中止整批请求。
func (h *InvitationHandler) CreateBatch(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request batchInvitationRequest
	if err := c.ShouldBindJSON(&request); err != nil || len(request.Invitations) < 1 || len(request.Invitations) > 20 {
		fail(c, http.StatusBadRequest, "invitation.batch_validation_failed", "批量邀请必须包含 1 到 20 条记录。")
		return
	}
	results := make([]batchInvitationResult, 0, len(request.Invitations))
	succeeded := 0
	for index, item := range request.Invitations {
		item.Email = strings.ToLower(strings.TrimSpace(item.Email))
		response, failure := h.createInvitation(c, principal, item)
		result := batchInvitationResult{Index: index, Email: item.Email, Succeeded: failure == nil}
		if failure == nil {
			result.Invitation = &response
			succeeded++
		} else {
			result.Error = &batchItemError{Code: failure.Code, Message: failure.Message}
		}
		results = append(results, result)
	}
	respond(c, http.StatusOK, gin.H{
		"total":     len(results),
		"succeeded": succeeded,
		"failed":    len(results) - succeeded,
		"results":   results,
	})
}

// createInvitation 执行单项邀请的服务调用、邀请链接生成和邮件投递记录组装。
func (h *InvitationHandler) createInvitation(c *gin.Context, principal service.Principal, request createInvitationRequest) (invitationResponse, *invitationFailure) {
	if request.ExpiresInHours < 0 || request.ExpiresInHours > int(service.MaxInvitationExpiry/time.Hour) {
		return invitationResponse{}, invitationCreateFailure(service.ErrInvitationInvalidExpiry)
	}
	expiresIn := service.DefaultInvitationExpiry
	if request.ExpiresInHours != 0 {
		expiresIn = time.Duration(request.ExpiresInHours) * time.Hour
	}
	result, err := h.auth.CreateInvitation(principal.OrganizationID, principal.UserID, request.Email, strings.TrimSpace(request.Role), expiresIn)
	if err != nil {
		return invitationResponse{}, invitationCreateFailure(err)
	}
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "membership.invite", TargetType: "invitation", TargetID: result.ID, Result: "success", RequestID: ensureRequestID(c)}).Error
	delivery := h.deliverInvitation(c.Request.Context(), result)
	h.auditDelivery(c, principal, result.ID, delivery)
	return invitationResponse{
		InvitationView: result.InvitationView,
		InviteURL:      "/invite/" + result.Token,
		Delivery:       delivery,
	}, nil
}

// invitationCreateFailure 将认证服务的领域错误转换为邀请 API 的错误信息。
func invitationCreateFailure(err error) *invitationFailure {
	switch {
	case errors.Is(err, service.ErrInvitationInvalidEmail), errors.Is(err, service.ErrInvitationInvalidRole), errors.Is(err, service.ErrInvitationInvalidExpiry):
		return &invitationFailure{HTTPStatus: http.StatusBadRequest, Code: "invitation.validation_failed", Message: "邮箱、角色或邀请有效期不符合要求。"}
	case errors.Is(err, service.ErrInvitationPending):
		return &invitationFailure{HTTPStatus: http.StatusConflict, Code: "invitation.already_pending", Message: "该邮箱已有尚未处理的邀请。"}
	case errors.Is(err, service.ErrInvitationAlreadyMember):
		return &invitationFailure{HTTPStatus: http.StatusConflict, Code: "membership.already_active", Message: "该邮箱已经是当前组织的有效成员。"}
	case errors.Is(err, service.ErrInvitationMemberExists):
		return &invitationFailure{HTTPStatus: http.StatusConflict, Code: "membership.already_managed", Message: "该邮箱已有停用或离开记录，请直接在成员管理中处理。"}
	default:
		return &invitationFailure{HTTPStatus: http.StatusInternalServerError, Code: "invitation.create_failed", Message: "邀请暂时无法创建。"}
	}
}

// List 分页列出当前组织的邀请，并附加每封邀请的最新投递状态。
func (h *InvitationHandler) List(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	page, pageSize, ok := listMeta(c, 0)
	if !ok {
		return
	}
	items, total, err := h.auth.ListInvitations(principal.OrganizationID, c.Query("status"), page, pageSize)
	if err != nil {
		if errors.Is(err, service.ErrInvitationInvalidStatus) {
			fail(c, http.StatusBadRequest, "invitation.invalid_status", "邀请状态筛选值不符合要求。")
			return
		}
		fail(c, http.StatusInternalServerError, "invitation.list_failed", "邀请列表暂时无法读取。")
		return
	}
	deliveries, err := h.deliveriesForInvitations(items)
	if err != nil {
		fail(c, http.StatusInternalServerError, "invitation.list_failed", "邀请列表暂时无法读取。")
		return
	}
	responses := make([]invitationResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, invitationResponse{
			InvitationView: item,
			Delivery:       deliveries[item.ID],
		})
	}
	respondWithMeta(c, http.StatusOK, responses, gin.H{"page": page, "page_size": pageSize, "total": total})
}

// Revoke 撤销尚未使用的邀请令牌，并写入对应审计事件。
func (h *InvitationHandler) Revoke(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	result, err := h.auth.RevokeInvitation(principal.OrganizationID, c.Param("id"))
	if err != nil {
		handleInvitationMutationError(c, err)
		return
	}
	_ = h.db.Create(&model.AuditEvent{
		ID:             uuid.NewString(),
		OrganizationID: principal.OrganizationID,
		ActorUserID:    principal.UserID,
		Action:         "membership.invitation_revoke",
		TargetType:     "invitation",
		TargetID:       result.ID,
		Result:         "success",
		RequestID:      ensureRequestID(c),
	}).Error
	respond(c, http.StatusOK, result)
}

// RetryEmail 对指定邀请重新执行邮件投递，并更新投递尝试记录。
func (h *InvitationHandler) RetryEmail(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	sender, err := h.integrations.MailSender(c.Request.Context(), principal.OrganizationID)
	if err != nil {
		fail(c, http.StatusServiceUnavailable, "notification.email_config_unavailable", "邮件配置暂时无法读取。")
		return
	}
	if sender == nil || !sender.Status().Enabled {
		fail(c, http.StatusConflict, "notification.email_disabled", "邮件投递未启用，请复制邀请链接发送给成员。")
		return
	}
	result, err := h.auth.RotateInvitationToken(principal.OrganizationID, c.Param("id"))
	if err != nil {
		handleInvitationLookupError(c, err)
		return
	}
	delivery := h.deliverInvitation(c.Request.Context(), result)
	h.auditDelivery(c, principal, result.ID, delivery)
	respond(c, http.StatusOK, invitationResponse{
		InvitationView: result.InvitationView,
		InviteURL:      "/invite/" + result.Token,
		Delivery:       delivery,
	})
}

// EmailStatus 返回指定邀请的邮件投递历史摘要。
func (h *InvitationHandler) EmailStatus(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	sender, err := h.integrations.MailSender(c.Request.Context(), principal.OrganizationID)
	if err != nil {
		fail(c, http.StatusServiceUnavailable, "notification.email_config_unavailable", "邮件配置暂时无法读取。")
		return
	}
	respond(c, http.StatusOK, sender.Status())
}

// Preview 生成邀请邮件预览，不创建邀请也不触发实际投递。
func (h *InvitationHandler) Preview(c *gin.Context) {
	view, err := h.auth.LookupInvitation(c.Param("token"))
	if err != nil {
		handleInvitationLookupError(c, err)
		return
	}
	respond(c, http.StatusOK, view)
}

// Accept 消费邀请令牌并完成注册或加入组织流程。
func (h *InvitationHandler) Accept(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	result, err := h.auth.AcceptInvitation(principal, c.Param("token"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvitationEmailMismatch):
			fail(c, http.StatusForbidden, "invitation.email_mismatch", "当前登录邮箱与邀请邮箱不一致。")
		case errors.Is(err, service.ErrInvitationAlreadyMember):
			fail(c, http.StatusConflict, "membership.already_active", "当前账户已经是该组织的有效成员。")
		default:
			handleInvitationLookupError(c, err)
		}
		return
	}
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: result.OrganizationID, ActorUserID: principal.UserID, Action: "membership.invitation_accept", TargetType: "invitation", TargetID: result.ID, Result: "success", RequestID: ensureRequestID(c)}).Error
	respond(c, http.StatusOK, result)
}

// handleInvitationLookupError 统一处理邀请查询阶段的未找到和数据库错误。
func handleInvitationLookupError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvitationExpired):
		fail(c, http.StatusGone, "invitation.expired", "邀请链接已过期。")
	case errors.Is(err, service.ErrInvitationRevoked):
		fail(c, http.StatusGone, "invitation.revoked", "邀请链接已撤销。")
	case errors.Is(err, service.ErrInvitationAccepted):
		fail(c, http.StatusGone, "invitation.already_accepted", "邀请链接已经被使用。")
	case errors.Is(err, service.ErrInvitationNotFound):
		fail(c, http.StatusNotFound, "invitation.not_found", "邀请链接不存在。")
	default:
		fail(c, http.StatusInternalServerError, "invitation.lookup_failed", "邀请状态暂时无法读取。")
	}
}

// handleInvitationMutationError 统一处理撤销、重试等邀请变更操作的领域错误。
func handleInvitationMutationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvitationExpired):
		fail(c, http.StatusConflict, "invitation.expired", "已过期的邀请不能撤销。")
	case errors.Is(err, service.ErrInvitationRevoked):
		fail(c, http.StatusConflict, "invitation.already_revoked", "邀请已经撤销。")
	case errors.Is(err, service.ErrInvitationAccepted):
		fail(c, http.StatusConflict, "invitation.already_accepted", "已接受的邀请不能撤销。")
	case errors.Is(err, service.ErrInvitationNotFound):
		fail(c, http.StatusNotFound, "invitation.not_found", "邀请不存在。")
	default:
		fail(c, http.StatusInternalServerError, "invitation.update_failed", "邀请状态暂时无法更新。")
	}
}

// deliverInvitation 渲染邀请邮件并调用按组织解析出的发送器，返回可持久化的投递结果。
func (h *InvitationHandler) deliverInvitation(ctx context.Context, invitation service.InvitationCreateResult) emailDeliveryResponse {
	sender, resolveErr := h.integrations.MailSender(ctx, invitation.OrganizationID)
	if resolveErr != nil || sender == nil {
		delivery := model.InvitationDelivery{ID: uuid.NewString(), InvitationID: invitation.ID, OrganizationID: invitation.OrganizationID, Channel: "email", Adapter: "disabled", Status: "failed", LastError: "邮件配置暂时无法读取"}
		delivery = h.persistDelivery(delivery, false)
		return deliveryResponse(delivery)
	}
	status := sender.Status()
	if !status.Enabled {
		delivery := model.InvitationDelivery{
			ID:             uuid.NewString(),
			InvitationID:   invitation.ID,
			OrganizationID: invitation.OrganizationID,
			Channel:        "email",
			Adapter:        status.Driver,
			Status:         "disabled",
		}
		delivery = h.persistDelivery(delivery, false)
		return deliveryResponse(delivery)
	}

	now := time.Now().UTC()
	delivery := model.InvitationDelivery{
		ID:             uuid.NewString(),
		InvitationID:   invitation.ID,
		OrganizationID: invitation.OrganizationID,
		Channel:        "email",
		Adapter:        status.Driver,
		Status:         "pending",
		Attempts:       1,
		LastAttemptAt:  &now,
	}
	delivery = h.persistDelivery(delivery, true)
	var organization model.Organization
	_ = h.db.Select("name, invitation_subject_template, invitation_body_template").Where("id = ?", invitation.OrganizationID).First(&organization).Error
	err := sender.SendInvitation(ctx, mailadapter.InvitationMessage{
		RecipientEmail:  invitation.Email,
		Organization:    invitation.Organization,
		Role:            invitation.Role,
		InvitationURL:   h.integrations.PublicWebBaseURL(ctx, invitation.OrganizationID) + "/invite/" + invitation.Token,
		ExpiresAt:       invitation.ExpiresAt,
		SubjectTemplate: organization.InvitationSubjectTemplate,
		BodyTemplate:    organization.InvitationBodyTemplate,
	})
	if err != nil {
		delivery.Status = "failed"
		delivery.LastError = safeDeliveryError(err)
	} else {
		delivery.Status = "sent"
		delivery.SentAt = &now
		delivery.LastError = ""
	}
	_ = h.db.Model(&model.InvitationDelivery{}).
		Where("invitation_id = ?", invitation.ID).
		Updates(map[string]interface{}{
			"status":          delivery.Status,
			"last_error":      delivery.LastError,
			"last_attempt_at": delivery.LastAttemptAt,
			"sent_at":         delivery.SentAt,
			"updated_at":      time.Now().UTC(),
		}).Error
	return deliveryResponse(delivery)
}

// persistDelivery 保存投递状态；incrementAttempt 为真时先递增尝试次数。
func (h *InvitationHandler) persistDelivery(delivery model.InvitationDelivery, incrementAttempt bool) model.InvitationDelivery {
	updates := map[string]interface{}{
		"adapter":    delivery.Adapter,
		"status":     delivery.Status,
		"last_error": "",
		"updated_at": time.Now().UTC(),
	}
	if incrementAttempt {
		updates["attempts"] = gorm.Expr("attempts + 1")
		updates["last_attempt_at"] = delivery.LastAttemptAt
		updates["sent_at"] = nil
	}
	_ = h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "invitation_id"}},
		DoUpdates: clause.Assignments(updates),
	}).Create(&delivery).Error
	var stored model.InvitationDelivery
	if err := h.db.Where("invitation_id = ?", delivery.InvitationID).First(&stored).Error; err == nil {
		return stored
	}
	return delivery
}

// auditDelivery 为邮件投递结果写入成功或失败审计事件，但不让审计失败覆盖主流程结果。
func (h *InvitationHandler) auditDelivery(c *gin.Context, principal service.Principal, invitationID string, delivery emailDeliveryResponse) {
	result := delivery.Status
	if result == "disabled" {
		result = "skipped"
	}
	_ = h.db.Create(&model.AuditEvent{
		ID:             uuid.NewString(),
		OrganizationID: principal.OrganizationID,
		ActorUserID:    principal.UserID,
		Action:         "membership.invite_email",
		TargetType:     "invitation",
		TargetID:       invitationID,
		Result:         result,
		RequestID:      ensureRequestID(c),
	}).Error
}

// deliveryResponse 将数据库投递记录映射为隐藏内部实现细节的 API 响应。
func deliveryResponse(delivery model.InvitationDelivery) emailDeliveryResponse {
	return emailDeliveryResponse{
		Status:        delivery.Status,
		Adapter:       delivery.Adapter,
		Attempts:      delivery.Attempts,
		LastError:     delivery.LastError,
		LastAttemptAt: delivery.LastAttemptAt,
		SentAt:        delivery.SentAt,
	}
}

// deliveriesForInvitations 批量读取邀请投递记录，按邀请 ID 建立快速查找表。
func (h *InvitationHandler) deliveriesForInvitations(invitations []service.InvitationView) (map[string]emailDeliveryResponse, error) {
	responses := make(map[string]emailDeliveryResponse, len(invitations))
	ids := make([]string, 0, len(invitations))
	for _, invitation := range invitations {
		ids = append(ids, invitation.ID)
		responses[invitation.ID] = emailDeliveryResponse{Status: "disabled", Adapter: "disabled", Attempts: 0}
	}
	if len(ids) == 0 {
		return responses, nil
	}
	var deliveries []model.InvitationDelivery
	if err := h.db.Where("invitation_id IN ?", ids).Find(&deliveries).Error; err != nil {
		return nil, err
	}
	for _, delivery := range deliveries {
		responses[delivery.InvitationID] = deliveryResponse(delivery)
	}
	return responses, nil
}

// safeDeliveryError 将底层邮件错误压缩为可向客户端展示且不泄露凭据的文本。
func safeDeliveryError(err error) string {
	message := strings.Join(strings.Fields(err.Error()), " ")
	if len(message) > 500 {
		message = message[:500]
	}
	return message
}

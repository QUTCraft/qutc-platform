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

type InvitationHandler struct {
	db               *gorm.DB
	auth             *service.AuthService
	mail             mailadapter.Sender
	publicWebBaseURL string
}

func NewInvitationHandler(db *gorm.DB, auth *service.AuthService, mail mailadapter.Sender, publicWebBaseURL string) *InvitationHandler {
	return &InvitationHandler{db: db, auth: auth, mail: mail, publicWebBaseURL: strings.TrimRight(publicWebBaseURL, "/")}
}

type createInvitationRequest struct {
	Email          string `json:"email" binding:"required,email,max=254"`
	Role           string `json:"role" binding:"required"`
	ExpiresInHours int    `json:"expires_in_hours"`
}

type invitationResponse struct {
	service.InvitationView
	InviteURL string                `json:"invite_url,omitempty"`
	Delivery  emailDeliveryResponse `json:"delivery"`
}

type emailDeliveryResponse struct {
	Status        string     `json:"status"`
	Adapter       string     `json:"adapter"`
	Attempts      int        `json:"attempts"`
	LastError     string     `json:"last_error,omitempty"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
}

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
	expiresIn := service.DefaultInvitationExpiry
	if request.ExpiresInHours != 0 {
		expiresIn = time.Duration(request.ExpiresInHours) * time.Hour
	}
	result, err := h.auth.CreateInvitation(principal.OrganizationID, principal.UserID, request.Email, strings.TrimSpace(request.Role), expiresIn)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvitationInvalidRole), errors.Is(err, service.ErrInvitationInvalidExpiry):
			fail(c, http.StatusBadRequest, "invitation.validation_failed", "邀请角色或有效期不符合要求。")
		case errors.Is(err, service.ErrInvitationPending):
			fail(c, http.StatusConflict, "invitation.already_pending", "该邮箱已有尚未处理的邀请。")
		case errors.Is(err, service.ErrInvitationAlreadyMember):
			fail(c, http.StatusConflict, "membership.already_active", "该邮箱已经是当前组织的有效成员。")
		default:
			fail(c, http.StatusInternalServerError, "invitation.create_failed", "邀请暂时无法创建。")
		}
		return
	}
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "membership.invite", TargetType: "invitation", TargetID: result.ID, Result: "success", RequestID: ensureRequestID(c)}).Error
	delivery := h.deliverInvitation(c.Request.Context(), result)
	h.auditDelivery(c, principal, result.ID, delivery)
	respond(c, http.StatusCreated, invitationResponse{
		InvitationView: result.InvitationView,
		InviteURL:      "/invite/" + result.Token,
		Delivery:       delivery,
	})
}

func (h *InvitationHandler) RetryEmail(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	if !h.mail.Status().Enabled {
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

func (h *InvitationHandler) EmailStatus(c *gin.Context) {
	respond(c, http.StatusOK, h.mail.Status())
}

func (h *InvitationHandler) Preview(c *gin.Context) {
	view, err := h.auth.LookupInvitation(c.Param("token"))
	if err != nil {
		handleInvitationLookupError(c, err)
		return
	}
	respond(c, http.StatusOK, view)
}

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

func (h *InvitationHandler) deliverInvitation(ctx context.Context, invitation service.InvitationCreateResult) emailDeliveryResponse {
	status := h.mail.Status()
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
	err := h.mail.SendInvitation(ctx, mailadapter.InvitationMessage{
		RecipientEmail: invitation.Email,
		Organization:   invitation.Organization,
		Role:           invitation.Role,
		InvitationURL:  h.publicWebBaseURL + "/invite/" + invitation.Token,
		ExpiresAt:      invitation.ExpiresAt,
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

func safeDeliveryError(err error) string {
	message := strings.Join(strings.Fields(err.Error()), " ")
	if len(message) > 500 {
		message = message[:500]
	}
	return message
}

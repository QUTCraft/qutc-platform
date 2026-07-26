package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type InvitationHandler struct {
	db   *gorm.DB
	auth *service.AuthService
}

func NewInvitationHandler(db *gorm.DB, auth *service.AuthService) *InvitationHandler {
	return &InvitationHandler{db: db, auth: auth}
}

type createInvitationRequest struct {
	Email          string `json:"email" binding:"required,email,max=254"`
	Role           string `json:"role" binding:"required"`
	ExpiresInHours int    `json:"expires_in_hours"`
}

type invitationResponse struct {
	service.InvitationView
	InviteURL string `json:"invite_url,omitempty"`
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
	_ = h.db.Create(&model.AuditEvent{OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "membership.invite", TargetType: "invitation", TargetID: result.ID, Result: "success", RequestID: ensureRequestID(c)}).Error
	respond(c, http.StatusCreated, invitationResponse{InvitationView: result.InvitationView, InviteURL: "/invite/" + result.Token})
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
	_ = h.db.Create(&model.AuditEvent{OrganizationID: result.OrganizationID, ActorUserID: principal.UserID, Action: "membership.invitation_accept", TargetType: "invitation", TargetID: result.ID, Result: "success", RequestID: ensureRequestID(c)}).Error
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

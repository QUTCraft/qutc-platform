package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Email           string `json:"email" binding:"required,email"`
	DisplayName     string `json:"display_name" binding:"required,max=80"`
	Password        string `json:"password" binding:"required,min=12,max=128"`
	InvitationToken string `json:"invitation_token"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,max=128"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request registerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "auth.validation_failed", "注册信息不符合要求。")
		return
	}
	var pair service.TokenPair
	var err error
	if strings.TrimSpace(request.InvitationToken) != "" {
		pair, err = h.auth.RegisterWithInvitation(request.Email, request.DisplayName, request.Password, request.InvitationToken)
	} else {
		pair, err = h.auth.Register(request.Email, request.DisplayName, request.Password)
	}
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailInUse):
			fail(c, http.StatusConflict, "auth.email_in_use", "该邮箱已注册。")
		case errors.Is(err, service.ErrInvalidPassword):
			fail(c, http.StatusBadRequest, "auth.password_too_short", "密码至少需要 12 个字符。")
		case errors.Is(err, service.ErrInvitationEmailMismatch):
			fail(c, http.StatusBadRequest, "invitation.email_mismatch", "注册邮箱必须与邀请邮箱一致。")
		case errors.Is(err, service.ErrInvitationExpired):
			fail(c, http.StatusGone, "invitation.expired", "邀请链接已过期。")
		case errors.Is(err, service.ErrInvitationRevoked):
			fail(c, http.StatusGone, "invitation.revoked", "邀请链接已撤销。")
		case errors.Is(err, service.ErrInvitationAccepted):
			fail(c, http.StatusConflict, "invitation.already_accepted", "邀请链接已经被使用。")
		case errors.Is(err, service.ErrInvitationNotFound):
			fail(c, http.StatusNotFound, "invitation.not_found", "邀请链接不存在。")
		default:
			fail(c, http.StatusInternalServerError, "auth.registration_failed", "注册暂时无法完成。")
		}
		return
	}
	respond(c, http.StatusCreated, pair)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "auth.validation_failed", "登录信息不符合要求。")
		return
	}
	pair, err := h.auth.Login(request.Email, request.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			fail(c, http.StatusUnauthorized, "auth.invalid_credentials", "邮箱或密码错误。")
			return
		}
		fail(c, http.StatusInternalServerError, "auth.login_failed", "登录暂时无法完成。")
		return
	}
	respond(c, http.StatusOK, pair)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var request refreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "auth.validation_failed", "刷新令牌不能为空。")
		return
	}
	pair, err := h.auth.Refresh(request.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefresh) || errors.Is(err, service.ErrInvalidCredentials) {
			fail(c, http.StatusUnauthorized, "auth.refresh_invalid", "刷新令牌无效或已过期。")
			return
		}
		fail(c, http.StatusInternalServerError, "auth.refresh_failed", "刷新会话暂时无法完成。")
		return
	}
	respond(c, http.StatusOK, pair)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "auth.validation_failed", "退出请求格式错误。")
		return
	}
	if err := h.auth.Logout(request.RefreshToken); err != nil {
		fail(c, http.StatusInternalServerError, "auth.logout_failed", "退出会话暂时无法完成。")
		return
	}
	respond(c, http.StatusOK, gin.H{"revoked": strings.TrimSpace(request.RefreshToken) != ""})
}

func (h *AuthHandler) Me(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	profile, err := h.auth.ProfileFor(principal)
	if err != nil {
		fail(c, http.StatusUnauthorized, "auth.session_invalid", "当前会话已失效。")
		return
	}
	respond(c, http.StatusOK, profile)
}

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request struct {
		DisplayName string `json:"display_name"`
		Bio         string `json:"bio"`
		AvatarURL   string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "profile.validation_failed", "资料格式不正确。")
		return
	}
	profile, err := h.auth.UpdateProfile(principal, request.DisplayName, request.Bio, request.AvatarURL)
	if err != nil {
		if strings.Contains(err.Error(), "profile fields") {
			fail(c, http.StatusBadRequest, "profile.validation_failed", "显示名、简介或头像地址长度不符合规范。")
			return
		}
		fail(c, http.StatusInternalServerError, "profile.update_failed", "个人资料保存失败。")
		return
	}
	respond(c, http.StatusOK, profile)
}

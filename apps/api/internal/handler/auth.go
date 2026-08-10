package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db           *gorm.DB
	auth         *service.AuthService
	refreshTTL   time.Duration
	secureCookie bool
}

func NewAuthHandler(db *gorm.DB, auth *service.AuthService, refreshTTL time.Duration, secureCookie bool) *AuthHandler {
	return &AuthHandler{db: db, auth: auth, refreshTTL: refreshTTL, secureCookie: secureCookie}
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

type switchOrganizationRequest struct {
	OrganizationID string `json:"organization_id" binding:"required,max=64"`
}

const (
	refreshCookieName       = "qutc_refresh"
	sessionExpiryCookieName = "qutc_session_expires"
)

func (h *AuthHandler) cookieSecure(c *gin.Context) bool {
	if !h.secureCookie {
		return false
	}
	if c.Request.TLS != nil {
		return true
	}
	forwardedProto := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0])
	return strings.EqualFold(forwardedProto, "https")
}

func (h *AuthHandler) setSessionCookies(c *gin.Context, pair service.TokenPair) time.Time {
	now := time.Now().UTC()
	sessionExpiresAt := pair.SessionExpiresAt.UTC()
	if sessionExpiresAt.IsZero() {
		sessionExpiresAt = now.Add(h.refreshTTL)
	}
	secure := h.cookieSecure(c)
	accessMaxAge := int(pair.ExpiresIn)
	if accessMaxAge < 1 {
		accessMaxAge = 1
	}
	refreshMaxAge := int(time.Until(sessionExpiresAt).Seconds())
	if refreshMaxAge < 1 {
		refreshMaxAge = 1
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: middleware.AccessCookieName, Value: pair.AccessToken, Path: "/api/v1", MaxAge: accessMaxAge, Expires: now.Add(time.Duration(accessMaxAge) * time.Second), HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: refreshCookieName, Value: pair.RefreshToken, Path: "/api/v1/auth", MaxAge: refreshMaxAge, Expires: sessionExpiresAt, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: sessionExpiryCookieName, Value: strconv.FormatInt(sessionExpiresAt.Unix(), 10), Path: "/", MaxAge: refreshMaxAge, Expires: sessionExpiresAt, Secure: secure, SameSite: http.SameSiteStrictMode})
	return sessionExpiresAt
}

func (h *AuthHandler) clearSessionCookies(c *gin.Context) {
	secure := h.cookieSecure(c)
	expired := time.Unix(1, 0).UTC()
	http.SetCookie(c.Writer, &http.Cookie{Name: middleware.AccessCookieName, Value: "", Path: "/api/v1", MaxAge: -1, Expires: expired, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: refreshCookieName, Value: "", Path: "/api/v1/auth", MaxAge: -1, Expires: expired, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: sessionExpiryCookieName, Value: "", Path: "/", MaxAge: -1, Expires: expired, Secure: secure, SameSite: http.SameSiteStrictMode})
}

func (h *AuthHandler) tokenPairResponse(c *gin.Context, pair service.TokenPair) gin.H {
	sessionExpiresAt := h.setSessionCookies(c, pair)
	return gin.H{"access_token": pair.AccessToken, "token_type": pair.TokenType, "expires_in": pair.ExpiresIn, "session_expires_at": sessionExpiresAt.Format(time.RFC3339), "user": pair.User}
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
	respond(c, http.StatusCreated, h.tokenPairResponse(c, pair))
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
	respond(c, http.StatusOK, h.tokenPairResponse(c, pair))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, cookieErr := c.Cookie(refreshCookieName)
	if cookieErr != nil || strings.TrimSpace(refreshToken) == "" {
		fail(c, http.StatusBadRequest, "auth.validation_failed", "刷新令牌不能为空。")
		return
	}
	pair, err := h.auth.Refresh(refreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRefresh) || errors.Is(err, service.ErrInvalidCredentials) {
			h.clearSessionCookies(c)
			fail(c, http.StatusUnauthorized, "auth.refresh_invalid", "刷新令牌无效或已过期。")
			return
		}
		fail(c, http.StatusInternalServerError, "auth.refresh_failed", "刷新会话暂时无法完成。")
		return
	}
	respond(c, http.StatusOK, h.tokenPairResponse(c, pair))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, _ := c.Cookie(refreshCookieName)
	if err := h.auth.Logout(refreshToken); err != nil {
		fail(c, http.StatusInternalServerError, "auth.logout_failed", "退出会话暂时无法完成。")
		return
	}
	h.clearSessionCookies(c)
	respond(c, http.StatusOK, gin.H{"revoked": strings.TrimSpace(refreshToken) != ""})
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

func (h *AuthHandler) Organizations(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	organizations, err := h.auth.ListOrganizations(principal)
	if err != nil {
		fail(c, http.StatusInternalServerError, "organization.list_failed", "可用组织暂时无法读取。")
		return
	}
	respond(c, http.StatusOK, organizations)
}

func (h *AuthHandler) SwitchOrganization(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request switchOrganizationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "organization.validation_failed", "目标组织不符合要求。")
		return
	}
	refreshToken, cookieErr := c.Cookie(refreshCookieName)
	if cookieErr != nil || strings.TrimSpace(refreshToken) == "" {
		fail(c, http.StatusUnauthorized, "auth.refresh_invalid", "当前会话无法切换组织，请重新登录。")
		return
	}
	pair, err := h.auth.SwitchOrganization(principal, request.OrganizationID, refreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrganizationUnavailable):
			fail(c, http.StatusForbidden, "organization.membership_unavailable", "当前账户不是该组织的有效成员。")
		case errors.Is(err, service.ErrInvalidRefresh):
			h.clearSessionCookies(c)
			fail(c, http.StatusUnauthorized, "auth.refresh_invalid", "当前会话无法切换组织，请重新登录。")
		default:
			fail(c, http.StatusInternalServerError, "organization.switch_failed", "组织暂时无法切换。")
		}
		return
	}
	_ = writeAudit(h.db, c, pair.User.OrganizationID, principal.UserID, "auth.organization_switch", "organization", pair.User.OrganizationID)
	respond(c, http.StatusOK, h.tokenPairResponse(c, pair))
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

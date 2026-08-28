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

// NewAuthHandler 创建认证 HTTP 处理器；refreshTTL 控制刷新 Cookie 生命周期，
// secureCookie 表示在生产 HTTPS 部署中强制 Cookie 仅经安全连接发送。
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

// cookieSecure 根据全局部署策略及代理转发协议判断当前响应 Cookie 是否应带 Secure 属性。
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

// setSessionCookies 写入访问与刷新 Cookie，并返回刷新会话的绝对到期时间供前端安排本地退出。
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

// clearSessionCookies 以过期 Cookie 覆盖浏览器中的认证凭据，配合服务端会话撤销完成注销。
func (h *AuthHandler) clearSessionCookies(c *gin.Context) {
	secure := h.cookieSecure(c)
	expired := time.Unix(1, 0).UTC()
	http.SetCookie(c.Writer, &http.Cookie{Name: middleware.AccessCookieName, Value: "", Path: "/api/v1", MaxAge: -1, Expires: expired, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: refreshCookieName, Value: "", Path: "/api/v1/auth", MaxAge: -1, Expires: expired, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: sessionExpiryCookieName, Value: "", Path: "/", MaxAge: -1, Expires: expired, Secure: secure, SameSite: http.SameSiteStrictMode})
}

// tokenPairResponse 设置浏览器 Cookie，并返回前端可安全读取的用户资料与会话截止时间。
func (h *AuthHandler) tokenPairResponse(c *gin.Context, pair service.TokenPair) gin.H {
	sessionExpiresAt := h.setSessionCookies(c, pair)
	return gin.H{"access_token": pair.AccessToken, "token_type": pair.TokenType, "expires_in": pair.ExpiresIn, "session_expires_at": sessionExpiresAt.Format(time.RFC3339), "user": pair.User}
}

// Register 接收账户信息；带邀请令牌时调用受邀注册流程，否则创建普通平台账户。
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

// Login 校验凭据并建立浏览器 Cookie 会话。
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

// Refresh 使用刷新 Cookie 轮换会话凭据；客户端不直接传递刷新令牌原文。
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

// Logout 尝试撤销刷新会话并始终清除浏览器 Cookie，使重复注销保持安全且幂等。
func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, _ := c.Cookie(refreshCookieName)
	if err := h.auth.Logout(refreshToken); err != nil {
		fail(c, http.StatusInternalServerError, "auth.logout_failed", "退出会话暂时无法完成。")
		return
	}
	h.clearSessionCookies(c)
	respond(c, http.StatusOK, gin.H{"revoked": strings.TrimSpace(refreshToken) != ""})
}

// Me 返回中间件已验证的当前用户资料。
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

// Organizations 返回当前用户可切换的有效组织成员关系。
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

// SwitchOrganization 验证目标组织成员关系并签发绑定该组织的新令牌对。
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

// UpdateMe 更新当前用户的公开资料；认证主体始终来自请求上下文而非客户端请求体。
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

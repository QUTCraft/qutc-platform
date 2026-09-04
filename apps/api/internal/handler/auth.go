// auth.go 实现注册、登录、刷新、退出、当前用户信息和组织切换等认证接口。
// 处理器同时维护 HTTP-only 会话 Cookie 与 JSON token 响应，确保浏览器和 API 客户端都能使用认证服务。
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

// AuthHandler 聚合认证服务、数据库和会话 Cookie 策略。
type AuthHandler struct {
	db           *gorm.DB
	auth         *service.AuthService
	refreshTTL   time.Duration
	secureCookie bool
}

// NewAuthHandler 创建认证处理器；refreshTTL 用于补全服务未返回会话过期时间的情况。
func NewAuthHandler(db *gorm.DB, auth *service.AuthService, refreshTTL time.Duration, secureCookie bool) *AuthHandler {
	return &AuthHandler{db: db, auth: auth, refreshTTL: refreshTTL, secureCookie: secureCookie}
}

// registerRequest 定义注册接口的输入，并通过 Gin binding 约束邮箱、名称和密码长度。
type registerRequest struct {
	Email           string `json:"email" binding:"required,email"`
	DisplayName     string `json:"display_name" binding:"required,max=80"`
	Password        string `json:"password" binding:"required,min=12,max=128"`
	InvitationToken string `json:"invitation_token"`
}

// loginRequest 定义登录接口所需的邮箱和密码。
type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,max=128"`
}

// switchOrganizationRequest 指定要切换到的组织 ID。
type switchOrganizationRequest struct {
	OrganizationID string `json:"organization_id" binding:"required,max=64"`
}

const (
	// refreshCookieName 保存长期刷新令牌，仅在认证 API 路径下发送。
	refreshCookieName = "qutc_refresh"
	// sessionExpiryCookieName 保存会话过期时间戳，供前端展示会话状态；它不包含令牌秘密。
	sessionExpiryCookieName = "qutc_session_expires"
)

// cookieSecure 根据部署开关、直接 TLS 或反向代理转发协议决定 Cookie 的 Secure 属性。
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

// setSessionCookies 写入访问、刷新和会话到期三个 Cookie，并返回最终采用的过期时间。
// 对过短或缺失的 TTL 使用至少 1 秒，避免浏览器立即丢弃新 Cookie。
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

// clearSessionCookies 以过去时间和 MaxAge=-1 覆盖三个会话 Cookie，使浏览器立即删除它们。
func (h *AuthHandler) clearSessionCookies(c *gin.Context) {
	secure := h.cookieSecure(c)
	expired := time.Unix(1, 0).UTC()
	http.SetCookie(c.Writer, &http.Cookie{Name: middleware.AccessCookieName, Value: "", Path: "/api/v1", MaxAge: -1, Expires: expired, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: refreshCookieName, Value: "", Path: "/api/v1/auth", MaxAge: -1, Expires: expired, HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(c.Writer, &http.Cookie{Name: sessionExpiryCookieName, Value: "", Path: "/", MaxAge: -1, Expires: expired, Secure: secure, SameSite: http.SameSiteStrictMode})
}

// tokenPairResponse 统一设置会话 Cookie，并返回访问令牌及用户信息 JSON。
func (h *AuthHandler) tokenPairResponse(c *gin.Context, pair service.TokenPair) gin.H {
	sessionExpiresAt := h.setSessionCookies(c, pair)
	return gin.H{"access_token": pair.AccessToken, "token_type": pair.TokenType, "expires_in": pair.ExpiresIn, "session_expires_at": sessionExpiresAt.Format(time.RFC3339), "user": pair.User}
}

// Register 创建用户账户；若携带邀请令牌，则由认证服务完成邀请关联。
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

// Login 校验凭据并建立新的访问/刷新会话。
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

// Refresh 使用刷新令牌轮换访问令牌，并同步更新会话 Cookie。
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

// Logout 撤销当前刷新会话并清理浏览器中的所有认证 Cookie。
func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, _ := c.Cookie(refreshCookieName)
	if err := h.auth.Logout(refreshToken); err != nil {
		fail(c, http.StatusInternalServerError, "auth.logout_failed", "退出会话暂时无法完成。")
		return
	}
	h.clearSessionCookies(c)
	respond(c, http.StatusOK, gin.H{"revoked": strings.TrimSpace(refreshToken) != ""})
}

// Me 返回当前访问令牌对应的用户和组织上下文。
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

// Organizations 列出当前用户有权加入的组织。
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

// SwitchOrganization 验证成员关系后签发目标组织上下文下的新令牌对。
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

// UpdateMe 更新当前用户的展示名称等个人资料，并返回更新后的用户信息。
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

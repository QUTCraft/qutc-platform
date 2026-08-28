package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

// principalKey 是鉴权结果在单次 Gin 请求上下文中的私有键，不直接使用字符串字面量，
// 以避免其他中间件覆写或读取到错误类型的值。
const principalKey = "qutc.principal"

// AccessCookieName is the browser session cookie used by the first-party web
// application. API clients may continue to send the same JWT as a Bearer token.
const AccessCookieName = "qutc_access"

// RequireAuth 解析请求凭据、验证访问令牌并将 Principal 写入 Gin 上下文。
// 验证失败时会中止请求，因此后续处理器可假定主体已存在。
func RequireAuth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 兼容浏览器 Cookie 与第三方客户端 Bearer Token；Bearer 优先，方便 API 调试与服务间调用。
		rawToken := accessTokenFromRequest(c.Request)
		if rawToken == "" {
			abort(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
			return
		}
		principal, err := auth.AuthenticateAccessToken(rawToken)
		if err != nil {
			if errors.Is(err, service.ErrSessionInactive) {
				abort(c, http.StatusUnauthorized, "auth.session_inactive", "账户或当前组织成员关系已停用。")
				return
			}
			abort(c, http.StatusUnauthorized, "auth.token_invalid", "访问令牌无效或已过期。")
			return
		}
		// 后续权限中间件和处理器复用已验证的主体，不重复解析 JWT 或访问数据库。
		c.Set(principalKey, principal)
		c.Next()
	}
}

// accessTokenFromRequest 优先读取 Authorization Bearer 令牌，缺失时回退读取同源访问 Cookie。
func accessTokenFromRequest(request *http.Request) string {
	// 仅接受完整的“Bearer <token>”格式，避免把畸形 Authorization 头当作凭据处理。
	header := strings.TrimSpace(request.Header.Get("Authorization"))
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[1])
	}
	cookie, err := request.Cookie(AccessCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

// RequirePermission 要求当前已认证主体拥有 permission 指定的最小业务权限。
// 它应当置于 RequireAuth 之后；缺少主体时返回 401，权限不足时返回 403。
func RequirePermission(auth *service.AuthService, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := PrincipalFromContext(c)
		if !ok {
			abort(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
			return
		}
		// 权限检查基于当前组织成员关系，而非仅依据 JWT 声明，角色变更可立即生效。
		allowed, err := auth.HasPermission(principal, permission)
		if err != nil || !allowed {
			abort(c, http.StatusForbidden, "admin.permission_denied", "当前角色没有执行此操作的权限。")
			return
		}
		c.Next()
	}
}

// PrincipalFromContext 读取 RequireAuth 缓存到当前请求的认证主体。
// 第二个返回值表示该上下文是否含有类型正确的主体。
func PrincipalFromContext(c *gin.Context) (service.Principal, bool) {
	value, ok := c.Get(principalKey)
	if !ok {
		return service.Principal{}, false
	}
	principal, ok := value.(service.Principal)
	return principal, ok
}

// abort 以统一错误信封结束当前请求，并附带请求 ID 以便客户端与日志关联排障。
func abort(c *gin.Context, status int, code, message string) {
	// 错误格式与请求 ID 始终一致，客户端可据此展示可读提示，运维可用 ID 关联结构化日志。
	requestID := EnsureRequestID(c)
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": requestID}})
}

package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

const principalKey = "qutc.principal"

func RequireAuth(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			abort(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
			return
		}
		principal, err := auth.AuthenticateAccessToken(parts[1])
		if err != nil {
			if errors.Is(err, service.ErrSessionInactive) {
				abort(c, http.StatusUnauthorized, "auth.session_inactive", "账户或当前组织成员关系已停用。")
				return
			}
			abort(c, http.StatusUnauthorized, "auth.token_invalid", "访问令牌无效或已过期。")
			return
		}
		c.Set(principalKey, principal)
		c.Next()
	}
}

func RequirePermission(auth *service.AuthService, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := PrincipalFromContext(c)
		if !ok {
			abort(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
			return
		}
		allowed, err := auth.HasPermission(principal, permission)
		if err != nil || !allowed {
			abort(c, http.StatusForbidden, "admin.permission_denied", "当前角色没有执行此操作的权限。")
			return
		}
		c.Next()
	}
}

func PrincipalFromContext(c *gin.Context) (service.Principal, bool) {
	value, ok := c.Get(principalKey)
	if !ok {
		return service.Principal{}, false
	}
	principal, ok := value.(service.Principal)
	return principal, ok
}

func abort(c *gin.Context, status int, code, message string) {
	requestID := EnsureRequestID(c)
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": requestID}})
}

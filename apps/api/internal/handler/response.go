// response.go 集中定义 API 成功与失败响应格式，并确保每个响应都带有 request_id。
package handler

import (
	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/gin-gonic/gin"
)

// respond 返回仅包含 data 和默认 meta 的成功响应。
func respond(c *gin.Context, status int, data interface{}) {
	respondWithMeta(c, status, data, gin.H{})
}

// respondWithMeta 在成功响应的 meta 中注入请求 ID，并保留调用方提供的分页等元数据。
func respondWithMeta(c *gin.Context, status int, data interface{}, meta gin.H) {
	requestID := ensureRequestID(c)
	meta["request_id"] = requestID
	c.JSON(status, gin.H{"data": data, "meta": meta})
}

// fail 返回不带额外详情的统一错误对象。
func fail(c *gin.Context, status int, code, message string) {
	requestID := ensureRequestID(c)
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": requestID}})
}

// failWithDetails 返回包含结构化 details 的统一错误对象，适合字段级校验失败。
func failWithDetails(c *gin.Context, status int, code, message string, details interface{}) {
	requestID := ensureRequestID(c)
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "details": details, "request_id": requestID}})
}

// ensureRequestID 获取中间件生成或复用的请求 ID，保证日志和客户端响应可关联。
func ensureRequestID(c *gin.Context) string {
	return middleware.EnsureRequestID(c)
}

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func respond(c *gin.Context, status int, data interface{}) {
	respondWithMeta(c, status, data, gin.H{})
}

func respondWithMeta(c *gin.Context, status int, data interface{}, meta gin.H) {
	requestID := ensureRequestID(c)
	meta["request_id"] = requestID
	c.JSON(status, gin.H{"data": data, "meta": meta})
}

func fail(c *gin.Context, status int, code, message string) {
	requestID := ensureRequestID(c)
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": requestID}})
}

func failWithDetails(c *gin.Context, status int, code, message string, details interface{}) {
	requestID := ensureRequestID(c)
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "details": details, "request_id": requestID}})
}

func ensureRequestID(c *gin.Context) string {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}
	c.Header("X-Request-ID", requestID)
	return requestID
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

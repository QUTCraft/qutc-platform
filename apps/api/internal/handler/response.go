package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func respond(c *gin.Context, status int, data interface{}) {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}
	c.Header("X-Request-ID", requestID)
	c.JSON(status, gin.H{"data": data, "meta": gin.H{"request_id": requestID}})
}

func fail(c *gin.Context, status int, code, message string) {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = uuid.NewString()
	}
	c.Header("X-Request-ID", requestID)
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": requestID}})
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

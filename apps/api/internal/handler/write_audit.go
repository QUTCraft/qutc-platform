package handler

import (
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func writeAudit(tx *gorm.DB, c *gin.Context, organizationID, actorUserID, action, targetType, targetID string) error {
	return tx.Create(&model.AuditEvent{
		ID: uuid.NewString(), OrganizationID: organizationID, ActorUserID: actorUserID,
		Action: action, TargetType: targetType, TargetID: targetID,
		Result: "success", RequestID: ensureRequestID(c),
	}).Error
}

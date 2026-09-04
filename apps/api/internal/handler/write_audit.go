// write_audit.go 提供事务内写入审计事件的共享辅助函数，确保业务变更和审计记录同事务提交。
package handler

import (
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// writeAudit 使用当前请求 ID 创建成功审计事件；调用方应在业务事务中调用它。
func writeAudit(tx *gorm.DB, c *gin.Context, organizationID, actorUserID, action, targetType, targetID string) error {
	return tx.Create(&model.AuditEvent{
		ID: uuid.NewString(), OrganizationID: organizationID, ActorUserID: actorUserID,
		Action: action, TargetType: targetType, TargetID: targetID,
		Result: "success", RequestID: ensureRequestID(c),
	}).Error
}

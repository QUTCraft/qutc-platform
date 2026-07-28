package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/serveradapter"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrApplicationNotFound         = errors.New("application not found")
	ErrApplicationAlreadyDecided   = errors.New("application already decided")
	ErrApplicationInvalidDecision  = errors.New("application decision is invalid")
	ErrApplicationReasonRequired   = errors.New("application rejection reason is required")
	ErrApplicationReasonTooLong    = errors.New("application decision reason is too long")
	ErrApplicationSyncNotFound     = errors.New("application server sync not found")
	ErrApplicationSyncNotRetryable = errors.New("application server sync is not retryable")
)

type ApplicationDecisionService struct {
	db      *gorm.DB
	adapter serveradapter.Adapter
}

func NewApplicationDecisionService(db *gorm.DB, adapter serveradapter.Adapter) *ApplicationDecisionService {
	return &ApplicationDecisionService{db: db, adapter: adapter}
}

func (s *ApplicationDecisionService) Decide(ctx context.Context, organizationID, actorUserID, applicationID, decision, reason, requestID string) (model.Application, *model.ApplicationServerSync, error) {
	if decision != "approved" && decision != "rejected" {
		return model.Application{}, nil, ErrApplicationInvalidDecision
	}
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) > 500 {
		return model.Application{}, nil, ErrApplicationReasonTooLong
	}
	if decision == "rejected" && reason == "" {
		return model.Application{}, nil, ErrApplicationReasonRequired
	}

	now := time.Now().UTC()
	var application model.Application
	var syncRecord *model.ApplicationServerSync
	err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.Application{}).
			Where("id = ? AND organization_id = ? AND status = ?", applicationID, organizationID, "pending").
			Updates(map[string]any{"status": decision, "decided_at": now, "decided_by": actorUserID, "decision_reason": reason})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			var existing model.Application
			if err := tx.Where("id = ? AND organization_id = ?", applicationID, organizationID).First(&existing).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrApplicationNotFound
				}
				return err
			}
			return ErrApplicationAlreadyDecided
		}
		if err := tx.Where("id = ? AND organization_id = ?", applicationID, organizationID).First(&application).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.AuditEvent{
			ID:             uuid.NewString(),
			OrganizationID: organizationID,
			ActorUserID:    actorUserID,
			Action:         "application." + decision,
			TargetType:     "application",
			TargetID:       applicationID,
			Result:         "success",
			RequestID:      requestID,
			CreatedAt:      now,
		}).Error; err != nil {
			return err
		}
		if decision == "approved" && application.Type == "whitelist" {
			record := model.ApplicationServerSync{
				ID:             uuid.NewString(),
				OrganizationID: organizationID,
				ApplicationID:  applicationID,
				Operation:      "whitelist.add",
				Adapter:        s.adapter.Name(),
				Mode:           s.adapter.Mode(),
				Status:         "pending",
				RequestedAt:    now,
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
			syncRecord = &record
		}
		return nil
	})
	if err != nil {
		return model.Application{}, nil, err
	}

	if syncRecord != nil {
		_ = s.completeWhitelistSync(ctx, syncRecord, application.GameID)
	}
	return application, syncRecord, nil
}

func (s *ApplicationDecisionService) RetryServerSync(ctx context.Context, organizationID, actorUserID, applicationID, requestID string) (model.ApplicationServerSync, error) {
	now := time.Now().UTC()
	var application model.Application
	var syncRecord model.ApplicationServerSync
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND organization_id = ?", applicationID, organizationID).First(&application).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrApplicationNotFound
			}
			return err
		}
		if application.Status != "approved" || application.Type != "whitelist" {
			return ErrApplicationSyncNotRetryable
		}
		if err := tx.Where("application_id = ? AND organization_id = ?", applicationID, organizationID).Order("created_at DESC").First(&syncRecord).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrApplicationSyncNotFound
			}
			return err
		}
		result := tx.Model(&model.ApplicationServerSync{}).
			Where("id = ? AND status = ?", syncRecord.ID, "failed").
			Updates(map[string]any{"status": "pending", "requested_at": now, "completed_at": nil, "message": "", "last_error": ""})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrApplicationSyncNotRetryable
		}
		syncRecord.Status = "pending"
		syncRecord.RequestedAt = now
		syncRecord.CompletedAt = nil
		syncRecord.Message = ""
		syncRecord.LastError = ""
		return tx.Create(&model.AuditEvent{
			ID:             uuid.NewString(),
			OrganizationID: organizationID,
			ActorUserID:    actorUserID,
			Action:         "application.server_sync_retry",
			TargetType:     "application",
			TargetID:       applicationID,
			Result:         "accepted",
			RequestID:      requestID,
			CreatedAt:      now,
		}).Error
	})
	if err != nil {
		return model.ApplicationServerSync{}, err
	}

	if err := s.completeWhitelistSync(ctx, &syncRecord, application.GameID); err != nil {
		return model.ApplicationServerSync{}, err
	}
	_ = s.db.Create(&model.AuditEvent{
		ID:             uuid.NewString(),
		OrganizationID: organizationID,
		ActorUserID:    actorUserID,
		Action:         "application.server_sync_retry_result",
		TargetType:     "application",
		TargetID:       applicationID,
		Result:         syncRecord.Status,
		RequestID:      requestID,
		CreatedAt:      time.Now().UTC(),
	}).Error
	return syncRecord, nil
}

func (s *ApplicationDecisionService) completeWhitelistSync(ctx context.Context, record *model.ApplicationServerSync, gameID string) error {
	result, err := s.adapter.AddWhitelist(ctx, gameID)
	now := time.Now().UTC()
	updates := map[string]any{
		"attempts":     gorm.Expr("attempts + 1"),
		"completed_at": now,
	}
	if err != nil {
		updates["status"] = "failed"
		updates["last_error"] = safeAdapterError(err)
		updates["message"] = ""
		record.Status = "failed"
		record.LastError = safeAdapterError(err)
	} else {
		updates["status"] = "succeeded"
		updates["last_error"] = ""
		updates["message"] = strings.TrimSpace(result.Message)
		record.Status = "succeeded"
		record.Message = strings.TrimSpace(result.Message)
	}
	record.Attempts++
	record.CompletedAt = &now
	updateResult := s.db.Model(&model.ApplicationServerSync{}).Where("id = ? AND status = ?", record.ID, "pending").Updates(updates)
	if updateResult.Error != nil {
		return updateResult.Error
	}
	if updateResult.RowsAffected == 0 {
		return ErrApplicationSyncNotRetryable
	}
	return nil
}

func safeAdapterError(err error) string {
	if err == nil {
		return ""
	}
	return "服务器适配器执行失败；详细原因仅记录于受控服务日志。"
}

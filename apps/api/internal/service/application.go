package service

import (
	"errors"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrApplicationNotFound        = errors.New("application not found")
	ErrApplicationAlreadyDecided  = errors.New("application already decided")
	ErrApplicationInvalidDecision = errors.New("application decision is invalid")
	ErrApplicationReasonRequired  = errors.New("application rejection reason is required")
	ErrApplicationReasonTooLong   = errors.New("application decision reason is too long")
)

// ApplicationDecisionService owns the platform approval transaction. It does
// not execute game-server commands or any other external side effects.
type ApplicationDecisionService struct {
	db            *gorm.DB
	notifications *NotificationService
}

func NewApplicationDecisionService(db *gorm.DB) *ApplicationDecisionService {
	return &ApplicationDecisionService{db: db}
}

func NewApplicationDecisionServiceWithNotifications(db *gorm.DB, notifications *NotificationService) *ApplicationDecisionService {
	return &ApplicationDecisionService{db: db, notifications: notifications}
}

func (s *ApplicationDecisionService) Decide(organizationID, actorUserID, applicationID, decision, reason, requestID string) (model.Application, error) {
	if decision != "approved" && decision != "rejected" {
		return model.Application{}, ErrApplicationInvalidDecision
	}
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) > 500 {
		return model.Application{}, ErrApplicationReasonTooLong
	}
	if decision == "rejected" && reason == "" {
		return model.Application{}, ErrApplicationReasonRequired
	}

	now := time.Now().UTC()
	var application model.Application
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
			ID: uuid.NewString(), OrganizationID: organizationID, ActorUserID: actorUserID,
			Action: "application." + decision, TargetType: "application", TargetID: applicationID,
			Result: "success", RequestID: requestID, CreatedAt: now,
		}).Error; err != nil {
			return err
		}
		if s.notifications != nil {
			return s.notifications.EnqueueApplicationDecision(tx, application)
		}
		return nil
	})
	return application, err
}

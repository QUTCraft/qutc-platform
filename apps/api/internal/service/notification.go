package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/mailadapter"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	NotificationStatusPending  = "pending"
	NotificationStatusSending  = "sending"
	NotificationStatusSent     = "sent"
	NotificationStatusFailed   = "failed"
	NotificationStatusDisabled = "disabled"
)

var (
	ErrNotificationNotFound      = errors.New("notification not found")
	ErrNotificationNotRetryable  = errors.New("notification is not retryable")
	ErrNotificationInvalidStatus = errors.New("notification status is invalid")
)

type NotificationService struct {
	db           *gorm.DB
	mailResolver MailSenderResolver
}

type MailSenderResolver interface {
	MailSender(context.Context, string) (mailadapter.Sender, error)
}

type staticMailSenderResolver struct{ sender mailadapter.Sender }

func (r staticMailSenderResolver) MailSender(context.Context, string) (mailadapter.Sender, error) {
	return r.sender, nil
}

type NotificationView struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	EventType      string     `json:"event_type"`
	TargetType     string     `json:"target_type"`
	TargetID       string     `json:"target_id"`
	RecipientEmail string     `json:"recipient_email"`
	Status         string     `json:"status"`
	Attempts       int        `json:"attempts"`
	LastError      string     `json:"last_error,omitempty"`
	AvailableAt    time.Time  `json:"available_at"`
	LastAttemptAt  *time.Time `json:"last_attempt_at,omitempty"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func NewNotificationService(db *gorm.DB, mail mailadapter.Sender) *NotificationService {
	return NewNotificationServiceWithResolver(db, staticMailSenderResolver{sender: mail})
}

func NewNotificationServiceWithResolver(db *gorm.DB, resolver MailSenderResolver) *NotificationService {
	return &NotificationService{db: db, mailResolver: resolver}
}

func (s *NotificationService) EnqueueApplicationDecision(tx *gorm.DB, application model.Application) error {
	if strings.TrimSpace(application.Email) == "" || (application.Status != "approved" && application.Status != "rejected") {
		return nil
	}
	now := time.Now().UTC()
	item := model.NotificationOutbox{
		ID: uuid.NewString(), OrganizationID: application.OrganizationID,
		EventType: "application." + application.Status, TargetType: "application", TargetID: application.ID,
		RecipientEmail: application.Email, Status: NotificationStatusPending, AvailableAt: now,
		CreatedAt: now, UpdatedAt: now,
	}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "event_type"}, {Name: "target_type"}, {Name: "target_id"}, {Name: "recipient_email"}}, DoNothing: true}).Create(&item).Error
}

func (s *NotificationService) EnqueueContentReview(tx *gorm.DB, request model.ContentReviewRequest, eventType string, recipientEmails []string) error {
	if s == nil || tx == nil {
		return nil
	}
	validEvents := map[string]bool{
		"content.review_submitted":  true,
		"content.review_rejected":   true,
		"content.published":         true,
		"content.archive_requested": true,
		"content.archive_rejected":  true,
		"content.archived":          true,
	}
	if !validEvents[eventType] {
		return errors.New("unsupported content review notification event")
	}
	now := time.Now().UTC()
	seen := make(map[string]bool, len(recipientEmails))
	for _, rawEmail := range recipientEmails {
		parsed, err := mail.ParseAddress(strings.TrimSpace(rawEmail))
		if err != nil || parsed.Address == "" {
			continue
		}
		email := strings.ToLower(parsed.Address)
		if seen[email] {
			continue
		}
		seen[email] = true
		item := model.NotificationOutbox{
			ID: uuid.NewString(), OrganizationID: request.OrganizationID,
			EventType: eventType, TargetType: "content_review", TargetID: request.ID,
			RecipientEmail: email, Status: NotificationStatusPending, AvailableAt: now,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "event_type"}, {Name: "target_type"}, {Name: "target_id"}, {Name: "recipient_email"}}, DoNothing: true}).Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *NotificationService) StartWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = s.ProcessPending(ctx)
			}
		}
	}()
}

func (s *NotificationService) ProcessPending(ctx context.Context) error {
	for i := 0; i < 10; i++ {
		item, ok, err := s.claimNext()
		if err != nil || !ok {
			return err
		}
		if err := s.deliver(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func (s *NotificationService) claimNext() (model.NotificationOutbox, bool, error) {
	var item model.NotificationOutbox
	now := time.Now().UTC()
	err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("status IN ? AND available_at <= ? AND attempts < ?", []string{NotificationStatusPending, NotificationStatusFailed}, now, 5).
			Order("available_at ASC, created_at ASC, id ASC").Limit(1).Find(&item)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		claim := tx.Model(&model.NotificationOutbox{}).
			Where("id = ? AND status IN ? AND available_at <= ? AND attempts < ?", item.ID, []string{NotificationStatusPending, NotificationStatusFailed}, now, 5).
			Updates(map[string]any{"status": NotificationStatusSending, "attempts": gorm.Expr("attempts + 1"), "last_attempt_at": now, "updated_at": now})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected == 0 {
			item = model.NotificationOutbox{}
			return nil
		}
		item.Status = NotificationStatusSending
		item.Attempts++
		item.LastAttemptAt = &now
		return nil
	})
	return item, item.ID != "", err
}

func (s *NotificationService) deliver(ctx context.Context, item model.NotificationOutbox) error {
	if s.mailResolver == nil {
		return s.finish(item, NotificationStatusDisabled, "邮件适配器未启用", nil, time.Time{})
	}
	sender, resolveErr := s.mailResolver.MailSender(ctx, item.OrganizationID)
	if resolveErr != nil {
		return s.finish(item, NotificationStatusFailed, "邮件配置暂时无法读取", resolveErr, time.Now().UTC().Add(time.Minute))
	}
	if sender == nil || !sender.Status().Enabled {
		return s.finish(item, NotificationStatusDisabled, "邮件适配器未启用", nil, time.Time{})
	}
	var organization model.Organization
	if err := s.db.Where("id = ?", item.OrganizationID).First(&organization).Error; err != nil {
		return s.finish(item, NotificationStatusFailed, "组织记录不存在", err, time.Now().UTC().Add(24*time.Hour))
	}
	var err error
	switch item.TargetType {
	case "application":
		var application model.Application
		if loadErr := s.db.Where("id = ? AND organization_id = ?", item.TargetID, item.OrganizationID).First(&application).Error; loadErr != nil {
			return s.finish(item, NotificationStatusFailed, "申请记录不存在", loadErr, time.Now().UTC().Add(24*time.Hour))
		}
		err = sender.SendApplicationDecision(ctx, mailadapter.ApplicationDecisionMessage{
			RecipientEmail: application.Email, Organization: organization.Name, ApplicantName: application.ApplicantName,
			ApplicationType: application.Type, Decision: application.Status, Reason: application.DecisionReason,
		})
	case "content_review":
		message, loadErr := s.contentReviewMessage(item, organization)
		if loadErr != nil {
			return s.finish(item, NotificationStatusFailed, "内容审核记录不存在", loadErr, time.Now().UTC().Add(24*time.Hour))
		}
		err = sender.SendContentReview(ctx, message)
	default:
		return s.finish(item, NotificationStatusFailed, "未知通知类型", errors.New("unsupported notification target"), time.Now().UTC().Add(24*time.Hour))
	}
	if err != nil {
		delay := time.Duration(item.Attempts*item.Attempts) * time.Minute
		return s.finish(item, NotificationStatusFailed, "邮件发送失败，请稍后重试", err, time.Now().UTC().Add(delay))
	}
	now := time.Now().UTC()
	return s.finish(item, NotificationStatusSent, "", nil, now)
}

func (s *NotificationService) contentReviewMessage(item model.NotificationOutbox, organization model.Organization) (mailadapter.ContentReviewMessage, error) {
	var request model.ContentReviewRequest
	if err := s.db.Where("id = ? AND organization_id = ?", item.TargetID, item.OrganizationID).First(&request).Error; err != nil {
		return mailadapter.ContentReviewMessage{}, err
	}
	var content model.Content
	if err := s.db.Where("id = ? AND organization_id = ?", request.ContentID, item.OrganizationID).First(&content).Error; err != nil {
		return mailadapter.ContentReviewMessage{}, err
	}
	var requester, reviewer, recipient model.User
	_ = s.db.First(&requester, "id = ?", request.RequesterUserID).Error
	if request.ReviewerUserID != "" {
		_ = s.db.First(&reviewer, "id = ?", request.ReviewerUserID).Error
	}
	_ = s.db.Where("LOWER(email) = ?", strings.ToLower(item.RecipientEmail)).First(&recipient).Error
	return mailadapter.ContentReviewMessage{
		RecipientEmail: item.RecipientEmail, RecipientName: recipient.DisplayName,
		Organization: organization.Name, EventType: item.EventType, ContentTitle: content.Title,
		RequesterName: requester.DisplayName, ReviewerName: reviewer.DisplayName,
		Note: request.Note, Feedback: request.Feedback,
	}, nil
}

func (s *NotificationService) finish(item model.NotificationOutbox, status, safeError string, err error, when time.Time) error {
	updates := map[string]any{"status": status, "last_error": safeError, "updated_at": time.Now().UTC()}
	if status == NotificationStatusSent {
		updates["sent_at"] = when
		updates["available_at"] = when
	} else if status == NotificationStatusFailed {
		updates["available_at"] = when
	}
	if updateErr := s.db.Model(&model.NotificationOutbox{}).Where("id = ? AND status = ?", item.ID, NotificationStatusSending).Updates(updates).Error; updateErr != nil {
		return updateErr
	}
	return nil
}

func (s *NotificationService) List(organizationID, status string, page, pageSize int) ([]NotificationView, int64, error) {
	if status != "" {
		valid := map[string]bool{NotificationStatusPending: true, NotificationStatusSending: true, NotificationStatusSent: true, NotificationStatusFailed: true, NotificationStatusDisabled: true}
		if !valid[status] {
			return nil, 0, ErrNotificationInvalidStatus
		}
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := s.db.Model(&model.NotificationOutbox{}).Where("organization_id = ?", organizationID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.NotificationOutbox
	if err := query.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	views := make([]NotificationView, 0, len(items))
	for _, item := range items {
		views = append(views, notificationView(item))
	}
	return views, total, nil
}

func (s *NotificationService) Retry(organizationID, notificationID string) (NotificationView, error) {
	now := time.Now().UTC()
	result := s.db.Model(&model.NotificationOutbox{}).Where("id = ? AND organization_id = ? AND status IN ?", notificationID, organizationID, []string{NotificationStatusFailed, NotificationStatusDisabled}).Updates(map[string]any{
		"status": NotificationStatusPending, "available_at": now, "last_error": "", "sent_at": nil, "attempts": 0, "updated_at": now,
	})
	if result.Error != nil {
		return NotificationView{}, result.Error
	}
	if result.RowsAffected == 0 {
		var item model.NotificationOutbox
		if err := s.db.Where("id = ? AND organization_id = ?", notificationID, organizationID).First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return NotificationView{}, ErrNotificationNotFound
			}
			return NotificationView{}, err
		}
		return NotificationView{}, ErrNotificationNotRetryable
	}
	var item model.NotificationOutbox
	if err := s.db.Where("id = ? AND organization_id = ?", notificationID, organizationID).First(&item).Error; err != nil {
		return NotificationView{}, err
	}
	return notificationView(item), nil
}

func notificationView(item model.NotificationOutbox) NotificationView {
	return NotificationView{ID: item.ID, OrganizationID: item.OrganizationID, EventType: item.EventType, TargetType: item.TargetType, TargetID: item.TargetID, RecipientEmail: item.RecipientEmail, Status: item.Status, Attempts: item.Attempts, LastError: item.LastError, AvailableAt: item.AvailableAt, LastAttemptAt: item.LastAttemptAt, SentAt: item.SentAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

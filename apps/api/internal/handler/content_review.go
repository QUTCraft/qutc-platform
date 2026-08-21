package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	contentReviewTypePublish = "publish"
	contentReviewTypeArchive = "archive"
	contentReviewPending     = "pending"
	contentReviewApproved    = "approved"
	contentReviewRejected    = "rejected"
)

var (
	errContentEditForbidden      = errors.New("content edit is restricted to its author")
	errContentReviewPending      = errors.New("content already has a pending review")
	errContentReviewNotFound     = errors.New("pending content review not found")
	errContentReviewStateInvalid = errors.New("content state cannot enter this review workflow")
	errContentReviewPermission   = errors.New("content review permission denied")
)

type contentReviewRequestBody struct {
	Note string `json:"note"`
}

type contentReviewDecisionBody struct {
	Feedback string `json:"feedback"`
}

func normalizeReviewText(value string) (string, bool) {
	value = strings.TrimSpace(value)
	return value, len([]rune(value)) <= 1000
}

func principalHasPermission(db *gorm.DB, principal service.Principal, permission string) (bool, error) {
	var count int64
	err := db.Table("permissions").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN membership_roles ON membership_roles.role_id = role_permissions.role_id").
		Joins("JOIN memberships ON memberships.id = membership_roles.membership_id").
		Where("memberships.user_id = ? AND memberships.organization_id = ? AND memberships.state = ? AND permissions.`key` = ?", principal.UserID, principal.OrganizationID, "active", permission).
		Count(&count).Error
	return count > 0, err
}

func principalCanModerateContent(db *gorm.DB, principal service.Principal) (bool, error) {
	canPublish, err := principalHasPermission(db, principal, "content:publish")
	if err != nil || canPublish {
		return canPublish, err
	}
	return principalHasPermission(db, principal, "content:archive")
}

func principalCanEditContent(db *gorm.DB, principal service.Principal, content model.Content) (bool, error) {
	if content.AuthorUserID == principal.UserID {
		return true, nil
	}
	return principalCanModerateContent(db, principal)
}

func requireContentEdit(db *gorm.DB, principal service.Principal, content model.Content) error {
	allowed, err := principalCanEditContent(db, principal, content)
	if err != nil {
		return err
	}
	if !allowed {
		return errContentEditForbidden
	}
	return nil
}

func latestContentRevision(tx *gorm.DB, organizationID, contentID string) (model.ContentRevision, error) {
	var revision model.ContentRevision
	err := tx.Where("organization_id = ? AND content_id = ?", organizationID, contentID).Order("version DESC").First(&revision).Error
	return revision, err
}

func pendingContentReview(tx *gorm.DB, organizationID, contentID string) (model.ContentReviewRequest, error) {
	var request model.ContentReviewRequest
	err := tx.Where("organization_id = ? AND content_id = ? AND status = ?", organizationID, contentID, contentReviewPending).Order("created_at DESC").First(&request).Error
	return request, err
}

func contentReviewerEmails(tx *gorm.DB, organizationID, permission, excludeEmail string) ([]string, error) {
	var emails []string
	err := tx.Table("users").Distinct("users.email").
		Joins("JOIN memberships ON memberships.user_id = users.id").
		Joins("JOIN membership_roles ON membership_roles.membership_id = memberships.id").
		Joins("JOIN role_permissions ON role_permissions.role_id = membership_roles.role_id").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("memberships.organization_id = ? AND memberships.state = ? AND users.state = ? AND permissions.`key` = ?", organizationID, "active", "active", permission).
		Pluck("users.email", &emails).Error
	if err != nil {
		return nil, err
	}
	filtered := make([]string, 0, len(emails))
	for _, email := range emails {
		if excludeEmail != "" && strings.EqualFold(strings.TrimSpace(email), strings.TrimSpace(excludeEmail)) {
			continue
		}
		filtered = append(filtered, email)
	}
	return filtered, nil
}

func userEmail(tx *gorm.DB, userID string) string {
	var user model.User
	if tx.Select("email").First(&user, "id = ?", userID).Error != nil {
		return ""
	}
	return user.Email
}

func (h *WorkspaceHandler) SubmitContentReview(c *gin.Context) {
	principal, _ := middleware.PrincipalFromContext(c)
	var body contentReviewRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "content.review_validation_failed", "提交说明格式不正确。")
		return
	}
	note, valid := normalizeReviewText(body.Note)
	if !valid {
		fail(c, http.StatusBadRequest, "content.review_validation_failed", "提交说明不能超过 1000 个字符。")
		return
	}

	var content model.Content
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&content).Error; err != nil {
			return err
		}
		if err := requireContentEdit(tx, principal, content); err != nil {
			return err
		}
		if content.Status != service.ContentStatusDraft && content.Status != service.ContentStatusArchived {
			return errContentReviewStateInvalid
		}
		if _, err := pendingContentReview(tx, principal.OrganizationID, content.ID); err == nil {
			return errContentReviewPending
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		content.Status = service.ContentStatusReview
		content.PublishedAt = nil
		if err := tx.Save(&content).Error; err != nil {
			return err
		}
		if err := createContentRevision(tx, content, principal.UserID, "submitted"); err != nil {
			return err
		}
		revision, err := latestContentRevision(tx, principal.OrganizationID, content.ID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		request := model.ContentReviewRequest{
			ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ContentID: content.ID, RevisionID: revision.ID,
			RequesterUserID: principal.UserID, Type: contentReviewTypePublish, Status: contentReviewPending, Note: note,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&request).Error; err != nil {
			return err
		}
		recipients, err := contentReviewerEmails(tx, principal.OrganizationID, "content:publish", principal.Email)
		if err != nil {
			return err
		}
		if err := h.notifications.EnqueueContentReview(tx, request, "content.review_submitted", recipients); err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "content.review_submit", "content", content.ID)
	})
	if err != nil {
		h.respondContentReviewError(c, err)
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, h.contentAdminItem(content, principal))
}

func (h *WorkspaceHandler) RequestContentArchive(c *gin.Context) {
	principal, _ := middleware.PrincipalFromContext(c)
	var body contentReviewRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "content.review_validation_failed", "下线说明格式不正确。")
		return
	}
	note, valid := normalizeReviewText(body.Note)
	if !valid {
		fail(c, http.StatusBadRequest, "content.review_validation_failed", "下线说明不能超过 1000 个字符。")
		return
	}

	var content model.Content
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&content).Error; err != nil {
			return err
		}
		if content.AuthorUserID != principal.UserID {
			return errContentEditForbidden
		}
		if content.Status != service.ContentStatusPublished {
			return errContentReviewStateInvalid
		}
		if _, err := pendingContentReview(tx, principal.OrganizationID, content.ID); err == nil {
			return errContentReviewPending
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		revision, err := latestContentRevision(tx, principal.OrganizationID, content.ID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		request := model.ContentReviewRequest{
			ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ContentID: content.ID, RevisionID: revision.ID,
			RequesterUserID: principal.UserID, Type: contentReviewTypeArchive, Status: contentReviewPending, Note: note,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&request).Error; err != nil {
			return err
		}
		recipients, err := contentReviewerEmails(tx, principal.OrganizationID, "content:archive", principal.Email)
		if err != nil {
			return err
		}
		if err := h.notifications.EnqueueContentReview(tx, request, "content.archive_requested", recipients); err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "content.archive_request", "content", content.ID)
	})
	if err != nil {
		h.respondContentReviewError(c, err)
		return
	}
	respond(c, http.StatusOK, h.contentAdminItem(content, principal))
}

func (h *WorkspaceHandler) RejectContentReview(c *gin.Context) {
	principal, _ := middleware.PrincipalFromContext(c)
	var body contentReviewDecisionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "content.review_validation_failed", "审核反馈格式不正确。")
		return
	}
	feedback, valid := normalizeReviewText(body.Feedback)
	if !valid || feedback == "" {
		fail(c, http.StatusBadRequest, "content.review_validation_failed", "退回审核时必须填写 1 至 1000 个字符的反馈。")
		return
	}

	var content model.Content
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&content).Error; err != nil {
			return err
		}
		request, err := pendingContentReview(tx, principal.OrganizationID, content.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errContentReviewNotFound
			}
			return err
		}
		permission := "content:publish"
		if request.Type == contentReviewTypeArchive {
			permission = "content:archive"
		}
		allowed, err := principalHasPermission(tx, principal, permission)
		if err != nil {
			return err
		}
		if !allowed {
			return errContentReviewPermission
		}
		now := time.Now().UTC()
		request.Status, request.Feedback, request.ReviewerUserID, request.ReviewedAt, request.UpdatedAt = contentReviewRejected, feedback, principal.UserID, &now, now
		if err := tx.Save(&request).Error; err != nil {
			return err
		}
		eventType := "content.archive_rejected"
		auditAction := "content.archive_rejected"
		if request.Type == contentReviewTypePublish {
			if content.Status != service.ContentStatusReview {
				return errContentReviewStateInvalid
			}
			content.Status = service.ContentStatusDraft
			content.PublishedAt = nil
			if err := tx.Save(&content).Error; err != nil {
				return err
			}
			if err := createContentRevision(tx, content, principal.UserID, "rejected"); err != nil {
				return err
			}
			eventType = "content.review_rejected"
			auditAction = "content.review_rejected"
		}
		if err := h.notifications.EnqueueContentReview(tx, request, eventType, []string{userEmail(tx, request.RequesterUserID)}); err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, auditAction, "content", content.ID)
	})
	if err != nil {
		h.respondContentReviewError(c, err)
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, h.contentAdminItem(content, principal))
}

func (h *WorkspaceHandler) resolveContentReviewDecision(tx *gorm.DB, content model.Content, revision model.ContentRevision, principal service.Principal, targetStatus string) error {
	reviewType, eventType := contentReviewTypePublish, "content.published"
	if targetStatus == service.ContentStatusArchived {
		reviewType, eventType = contentReviewTypeArchive, "content.archived"
	}
	request, err := pendingContentReview(tx, content.OrganizationID, content.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	now := time.Now().UTC()
	if errors.Is(err, gorm.ErrRecordNotFound) || request.Type != reviewType {
		request = model.ContentReviewRequest{
			ID: uuid.NewString(), OrganizationID: content.OrganizationID, ContentID: content.ID, RevisionID: revision.ID,
			RequesterUserID: content.AuthorUserID, Type: reviewType, Status: contentReviewApproved,
			ReviewerUserID: principal.UserID, ReviewedAt: &now, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&request).Error; err != nil {
			return err
		}
	} else {
		request.Status, request.ReviewerUserID, request.ReviewedAt, request.UpdatedAt = contentReviewApproved, principal.UserID, &now, now
		if err := tx.Save(&request).Error; err != nil {
			return err
		}
	}
	return h.notifications.EnqueueContentReview(tx, request, eventType, []string{userEmail(tx, request.RequesterUserID)})
}

func (h *WorkspaceHandler) respondContentReviewError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		fail(c, http.StatusNotFound, "content.not_found", "内容不存在或不属于当前组织。")
	case errors.Is(err, errContentEditForbidden):
		fail(c, http.StatusForbidden, "content.author_required", "普通编辑只能修改或提交自己创建的内容。")
	case errors.Is(err, errContentReviewPermission):
		fail(c, http.StatusForbidden, "content.review_permission_denied", "当前角色没有处理此审核的权限。")
	case errors.Is(err, errContentReviewPending):
		fail(c, http.StatusConflict, "content.review_pending", "该内容已有待处理审核，请勿重复提交。")
	case errors.Is(err, errContentReviewNotFound):
		fail(c, http.StatusConflict, "content.review_not_found", "该内容没有待处理审核。")
	case errors.Is(err, errContentReviewStateInvalid):
		fail(c, http.StatusConflict, "content.review_state_invalid", "内容当前状态不支持此审核操作。")
	default:
		fail(c, http.StatusInternalServerError, "content.review_failed", "内容审核操作失败。")
	}
}

func contentReviewItem(db *gorm.DB, request model.ContentReviewRequest) gin.H {
	var requester, reviewer model.User
	_ = db.First(&requester, "id = ?", request.RequesterUserID).Error
	if request.ReviewerUserID != "" {
		_ = db.First(&reviewer, "id = ?", request.ReviewerUserID).Error
	}
	var reviewerID any
	var reviewerName any
	if request.ReviewerUserID != "" {
		reviewerID, reviewerName = request.ReviewerUserID, reviewer.DisplayName
	}
	return gin.H{
		"id": request.ID, "type": request.Type, "status": request.Status,
		"revision_id": request.RevisionID, "requester_user_id": request.RequesterUserID,
		"requester": requester.DisplayName, "note": request.Note, "feedback": request.Feedback,
		"reviewer_user_id": reviewerID, "reviewer": reviewerName,
		"created_at": request.CreatedAt, "reviewed_at": request.ReviewedAt,
	}
}

func deleteContentReviewRecords(tx *gorm.DB, organizationID, contentID string) error {
	var reviewIDs []string
	if err := tx.Model(&model.ContentReviewRequest{}).Where("organization_id = ? AND content_id = ?", organizationID, contentID).Pluck("id", &reviewIDs).Error; err != nil {
		return err
	}
	if len(reviewIDs) > 0 {
		if err := tx.Where("organization_id = ? AND target_type = ? AND target_id IN ?", organizationID, "content_review", reviewIDs).Delete(&model.NotificationOutbox{}).Error; err != nil {
			return err
		}
	}
	return tx.Where("organization_id = ? AND content_id = ?", organizationID, contentID).Delete(&model.ContentReviewRequest{}).Error
}

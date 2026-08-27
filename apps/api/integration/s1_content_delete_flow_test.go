//go:build integration

package integration_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestS1ContentDeletionRemovesRecordAndReviewTrail(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	client := &http.Client{Timeout: 10 * time.Second}
	db := openIntegrationDB(t, cfg.mysqlDSN)

	var organization model.Organization
	if err := db.Where("slug = ?", cfg.organizationSlug).First(&organization).Error; err != nil {
		t.Fatalf("load organization: %v", err)
	}
	var editorRole model.Role
	if err := db.Where("`key` = ?", "editor").First(&editorRole).Error; err != nil {
		t.Fatalf("load editor role: %v", err)
	}
	password := "S1-Editor-Password-2026!"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash editor password: %v", err)
	}
	editor := model.User{ID: uuid.NewString(), Email: "s1-delete-editor-" + uuid.NewString() + "@example.test", DisplayName: "S1 Delete Editor", PasswordHash: string(hash), State: "active", DefaultOrganizationID: organization.ID}
	if err := db.Create(&editor).Error; err != nil {
		t.Fatalf("create delete editor: %v", err)
	}
	membership := model.Membership{ID: uuid.NewString(), OrganizationID: organization.ID, UserID: editor.ID, State: "active"}
	if err := db.Create(&membership).Error; err != nil {
		t.Fatalf("create delete editor membership: %v", err)
	}
	if err := db.Create(&model.MembershipRole{MembershipID: membership.ID, RoleID: editorRole.ID}).Error; err != nil {
		t.Fatalf("assign delete editor role: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Where("user_id = ?", editor.ID).Delete(&model.RefreshToken{}).Error
		var memberships []model.Membership
		_ = db.Where("user_id = ?", editor.ID).Find(&memberships).Error
		for _, item := range memberships {
			_ = db.Where("membership_id = ?", item.ID).Delete(&model.MembershipRole{}).Error
		}
		_ = db.Where("user_id = ?", editor.ID).Delete(&model.Membership{}).Error
		_ = db.Where("id = ?", editor.ID).Delete(&model.User{}).Error
	})

	ownerToken := loginAsOwner(t, client, cfg)
	editorToken := loginWithCredentials(t, client, cfg, editor.Email, password)
	content := createDraft(t, client, cfg, ownerToken, 101)
	t.Cleanup(func() { cleanupContentFixture(t, db, content.ID) })

	submitted := submitContentReview(t, client, cfg, ownerToken, content.ID)
	if submitted.Status != "review" {
		t.Fatalf("submit status = %q, want review", submitted.Status)
	}
	if !submitted.CanDelete {
		t.Fatalf("review content can_delete = false, want true")
	}

	deleteAdminContent(t, client, cfg, editorToken, content.ID, http.StatusForbidden)
	deleteAdminContent(t, client, cfg, ownerToken, content.ID, http.StatusOK)

	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/content/"+content.ID, ownerToken, nil, http.StatusNotFound)
	requireStatus(t, client, http.MethodGet, portalContentURL(cfg, content.ID), "", nil, http.StatusNotFound)

	var revisions int64
	if err := db.Model(&model.ContentRevision{}).Where("content_id = ?", content.ID).Count(&revisions).Error; err != nil {
		t.Fatalf("count content revisions: %v", err)
	}
	if revisions != 0 {
		t.Fatalf("content %s left %d revisions behind", content.ID, revisions)
	}
	var reviews int64
	if err := db.Model(&model.ContentReviewRequest{}).Where("content_id = ?", content.ID).Count(&reviews).Error; err != nil {
		t.Fatalf("count content review requests: %v", err)
	}
	if reviews != 0 {
		t.Fatalf("content %s left %d review requests behind", content.ID, reviews)
	}
}

func TestS1ContentDeleteRequiresArchiveFirst(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	client := &http.Client{Timeout: 10 * time.Second}
	db := openIntegrationDB(t, cfg.mysqlDSN)
	ownerToken := loginAsOwner(t, client, cfg)
	content := createDraft(t, client, cfg, ownerToken, 102)
	t.Cleanup(func() { cleanupContentFixture(t, db, content.ID) })

	submitted := submitContentReview(t, client, cfg, ownerToken, content.ID)
	if submitted.Status != "review" {
		t.Fatalf("submit status = %q, want review", submitted.Status)
	}
	published := changeContentStatus(t, client, cfg, ownerToken, content.ID, "publish")
	if published.Status != "published" {
		t.Fatalf("publish status = %q, want published", published.Status)
	}
	if published.CanDelete {
		t.Fatalf("published content can_delete = true, want false")
	}
	deleteAdminContent(t, client, cfg, ownerToken, content.ID, http.StatusConflict)

	archived := changeContentStatus(t, client, cfg, ownerToken, content.ID, "archive")
	if archived.Status != "archived" {
		t.Fatalf("archive status = %q, want archived", archived.Status)
	}
	if !archived.CanDelete {
		t.Fatalf("archived content can_delete = false, want true")
	}
	deleteAdminContent(t, client, cfg, ownerToken, content.ID, http.StatusOK)
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/content/"+content.ID, ownerToken, nil, http.StatusNotFound)
}

func deleteAdminContent(t *testing.T, client *http.Client, cfg integrationConfig, token, contentID string, expectedStatus int) {
	t.Helper()
	request(t, client, http.MethodDelete, cfg.apiURL+"/api/v1/admin/content/"+contentID, token, nil, expectedStatus)
}

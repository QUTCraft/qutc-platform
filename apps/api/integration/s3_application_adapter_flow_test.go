//go:build integration

package integration_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/config"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type adminApplicationDTO struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	Type           string `json:"type"`
	GameID         string `json:"game_id"`
	DecisionReason string `json:"decision_reason"`
}

func TestS3ApplicationApprovalWorkflow(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	client := &http.Client{Timeout: 10 * time.Second}
	db := openIntegrationDB(t, cfg.mysqlDSN)
	ownerToken := loginAsOwner(t, client, cfg)

	gameID := "S3" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	applicationID := submitApplicationFixture(t, client, cfg, gameID, strings.ToLower(gameID)+"@integration.invalid")
	t.Cleanup(func() { cleanupApplicationFixture(t, db, applicationID) })

	approveURL := cfg.apiURL + "/api/v1/admin/applications/" + applicationID + "/approve"
	body := request(t, client, http.MethodPost, approveURL, ownerToken, map[string]string{"reason": "集成测试审批通过。"}, http.StatusOK)
	if strings.Contains(string(body), "server_sync") {
		t.Fatal("approval response still exposed retired server synchronization state")
	}
	var approvedEnvelope apiEnvelope[adminApplicationDTO]
	decodeJSON(t, body, &approvedEnvelope)
	if approvedEnvelope.Data.Status != "approved" || approvedEnvelope.Data.DecisionReason != "集成测试审批通过。" {
		t.Fatalf("approved application = %+v", approvedEnvelope.Data)
	}
	requireStatus(t, client, http.MethodPost, approveURL, ownerToken, nil, http.StatusConflict)

	var filteredEnvelope apiEnvelope[[]adminApplicationDTO]
	filteredURL := cfg.apiURL + "/api/v1/admin/applications?status=approved&type=whitelist&page=1&page_size=1&query=" + gameID
	decodeJSON(t, request(t, client, http.MethodGet, filteredURL, ownerToken, nil, http.StatusOK), &filteredEnvelope)
	if len(filteredEnvelope.Data) != 1 || filteredEnvelope.Data[0].ID != applicationID {
		t.Fatalf("filtered applications = %+v, want only %s", filteredEnvelope.Data, applicationID)
	}
	if filteredEnvelope.Meta.Page != 1 || filteredEnvelope.Meta.PageSize != 1 || filteredEnvelope.Meta.Total != 1 {
		t.Fatalf("filtered pagination = %+v", filteredEnvelope.Meta)
	}

	rejectedID := submitApplicationFixture(t, client, cfg, "Reject"+strings.ReplaceAll(uuid.NewString(), "-", "")[:10], "reject-"+uuid.NewString()+"@integration.invalid")
	t.Cleanup(func() { cleanupApplicationFixture(t, db, rejectedID) })
	rejectURL := cfg.apiURL + "/api/v1/admin/applications/" + rejectedID + "/reject"
	requireStatus(t, client, http.MethodPost, rejectURL, ownerToken, nil, http.StatusBadRequest)
	var rejectedEnvelope apiEnvelope[adminApplicationDTO]
	decodeJSON(t, request(t, client, http.MethodPost, rejectURL, ownerToken, map[string]string{"reason": "资料需要补充。"}, http.StatusOK), &rejectedEnvelope)
	if rejectedEnvelope.Data.Status != "rejected" || rejectedEnvelope.Data.DecisionReason != "资料需要补充。" {
		t.Fatalf("rejected application = %+v", rejectedEnvelope.Data)
	}

	var auditCount int64
	if err := db.Model(&model.AuditEvent{}).
		Where("target_type = ? AND target_id = ? AND action = ?", "application", applicationID, "application.approved").
		Count(&auditCount).Error; err != nil {
		t.Fatalf("count approval audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("approval audit count = %d, want 1", auditCount)
	}

	// Removed game-server endpoints must remain unreachable.
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/server/status", ownerToken, nil, http.StatusNotFound)
	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/server/commands", ownerToken, map[string]string{"command": "list"}, http.StatusNotFound)
	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/applications/"+applicationID+"/server-sync/retry", ownerToken, nil, http.StatusNotFound)
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/portal/organizations/"+cfg.organizationSlug+"/server-status", "", nil, http.StatusNotFound)
}

func TestS3DemoSeedIsIdempotentAndNonDestructive(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	db := openIntegrationDB(t, cfg.mysqlDSN)

	var organization model.Organization
	if err := db.Where("slug = ?", cfg.organizationSlug).First(&organization).Error; err != nil {
		t.Fatalf("load organization: %v", err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin seed transaction: %v", tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })

	applicationIDs := []string{"application_demo", "application_demo_approved", "application_demo_rejected"}
	projectIDs := []string{"project_cms", "project_spawn", "project_wiki"}
	contentIDs := []string{"content_cms", "content_build", "content_resource", "content_knowledge"}
	directoryIDs := []string{"knowledge_directory_collaboration", "knowledge_directory_technology", "knowledge_directory_community"}
	if err := tx.Where("id IN ?", applicationIDs).Delete(&model.Application{}).Error; err != nil {
		t.Fatalf("delete seed application fixtures: %v", err)
	}
	if err := tx.Where("project_id IN ?", projectIDs).Delete(&model.ProjectMilestone{}).Error; err != nil {
		t.Fatalf("delete seed milestone fixtures: %v", err)
	}
	if err := tx.Where("project_id IN ?", projectIDs).Delete(&model.ProjectMember{}).Error; err != nil {
		t.Fatalf("delete seed project member fixtures: %v", err)
	}
	if err := tx.Where("id IN ?", projectIDs).Delete(&model.Project{}).Error; err != nil {
		t.Fatalf("delete seed project fixtures: %v", err)
	}
	if err := tx.Where("id IN ?", contentIDs).Delete(&model.Content{}).Error; err != nil {
		t.Fatalf("delete seed content fixtures: %v", err)
	}
	if err := tx.Where("id IN ?", directoryIDs).Delete(&model.KnowledgeDirectory{}).Error; err != nil {
		t.Fatalf("delete seed directory fixtures: %v", err)
	}

	seedConfig := config.Config{BootstrapAdminEmail: cfg.adminEmail}
	if err := database.SeedDemoData(tx, seedConfig, organization); err != nil {
		t.Fatalf("first SeedDemoData() error = %v", err)
	}
	if err := tx.Model(&model.Content{}).Where("id = ?", "content_cms").Update("title", "保留人工修改的标题").Error; err != nil {
		t.Fatalf("customize seed-owned record: %v", err)
	}
	if err := database.SeedDemoData(tx, seedConfig, organization); err != nil {
		t.Fatalf("second SeedDemoData() error = %v", err)
	}

	assertSeedCount(t, tx, &model.Content{}, "id IN ?", contentIDs, 4)
	assertSeedCount(t, tx, &model.KnowledgeDirectory{}, "id IN ?", directoryIDs, 3)
	assertSeedCount(t, tx, &model.Project{}, "id IN ?", projectIDs, 3)
	assertSeedCount(t, tx, &model.ProjectMilestone{}, "project_id = ?", "project_cms", 2)
	assertSeedCount(t, tx, &model.Application{}, "id IN ?", applicationIDs, 3)

	var content model.Content
	if err := tx.First(&content, "id = ?", "content_cms").Error; err != nil {
		t.Fatalf("reload customized seed content: %v", err)
	}
	if content.Title != "保留人工修改的标题" {
		t.Fatalf("second seed overwrote existing content title: %q", content.Title)
	}
}

func TestS3MultiOrganizationDemoSeedIsScopedAndIdempotent(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	db := openIntegrationDB(t, cfg.mysqlDSN)

	var primary model.Organization
	if err := db.Where("slug = ?", cfg.organizationSlug).First(&primary).Error; err != nil {
		t.Fatalf("load primary organization: %v", err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin multi-organization seed transaction: %v", tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })

	seedConfig := config.Config{BootstrapAdminEmail: cfg.adminEmail, BootstrapAdminPassword: cfg.adminPassword, BootstrapAdminName: "S3 Demo Owner", DemoSeedProfile: "qutcraft"}
	if err := database.SeedMultiOrganizationDemo(tx, seedConfig, primary); err != nil {
		t.Fatalf("first SeedMultiOrganizationDemo() error = %v", err)
	}
	var secondary model.Organization
	if err := tx.Where("slug = ?", "campus-commons").First(&secondary).Error; err != nil {
		t.Fatalf("load secondary demo organization: %v", err)
	}
	if secondary.Name != "Campus Commons" || secondary.ID == primary.ID {
		t.Fatalf("secondary demo organization = %+v", secondary)
	}
	applicationIDs := []string{"application_pending_generic", "application_approved_generic", "application_rejected_generic"}
	if err := tx.Where("organization_id = ? AND id IN ?", secondary.ID, applicationIDs).Delete(&model.Application{}).Error; err != nil {
		t.Fatalf("reset secondary demo application fixtures: %v", err)
	}
	if err := database.SeedMultiOrganizationDemo(tx, seedConfig, primary); err != nil {
		t.Fatalf("reseed secondary demo applications: %v", err)
	}

	var owner model.User
	if err := tx.Where("email = ?", cfg.adminEmail).First(&owner).Error; err != nil {
		t.Fatalf("load bootstrap owner: %v", err)
	}
	var secondaryMembership model.Membership
	if err := tx.Where("organization_id = ? AND user_id = ? AND state = ?", secondary.ID, owner.ID, "active").First(&secondaryMembership).Error; err != nil {
		t.Fatalf("load secondary owner membership: %v", err)
	}
	var ownerRole model.Role
	if err := tx.Where("`key` = ?", "owner").First(&ownerRole).Error; err != nil {
		t.Fatalf("load owner role: %v", err)
	}
	var ownerRoleLinkCount int64
	if err := tx.Model(&model.MembershipRole{}).Where("membership_id = ? AND role_id = ?", secondaryMembership.ID, ownerRole.ID).Count(&ownerRoleLinkCount).Error; err != nil {
		t.Fatalf("count secondary owner role link: %v", err)
	}
	if ownerRoleLinkCount != 1 {
		t.Fatalf("secondary owner role link count = %d, want 1", ownerRoleLinkCount)
	}

	contentIDs := []string{"content_main_generic", "content_event_generic", "content_resource_generic", "content_knowledge_generic", "content_safety_generic", "content_archive_generic"}
	projectIDs := []string{"project_main_generic", "project_event_generic", "project_knowledge_generic"}
	directoryIDs := []string{"knowledge_dir_collab_generic", "knowledge_dir_tech_generic", "knowledge_dir_community_generic"}
	assertScopedSeedCount(t, tx, &model.Content{}, secondary.ID, contentIDs, 6)
	assertScopedSeedCount(t, tx, &model.Project{}, secondary.ID, projectIDs, 3)
	assertScopedSeedCount(t, tx, &model.KnowledgeDirectory{}, secondary.ID, directoryIDs, 3)
	assertSeedCount(t, tx, &model.AgentDefinition{}, "organization_id = ?", secondary.ID, 2)
	var membershipApplicationCount int64
	if err := tx.Model(&model.Application{}).Where("organization_id = ? AND type = ?", secondary.ID, "membership").Count(&membershipApplicationCount).Error; err != nil {
		t.Fatalf("count secondary membership applications: %v", err)
	}
	if membershipApplicationCount != 3 {
		t.Fatalf("secondary membership application count = %d, want 3", membershipApplicationCount)
	}

	if err := tx.Model(&model.Content{}).Where("id = ?", "content_main_generic").Update("title", "保留通用组织人工修改").Error; err != nil {
		t.Fatalf("customize secondary demo content: %v", err)
	}
	if err := database.SeedMultiOrganizationDemo(tx, seedConfig, primary); err != nil {
		t.Fatalf("second SeedMultiOrganizationDemo() error = %v", err)
	}
	var customized model.Content
	if err := tx.First(&customized, "id = ?", "content_main_generic").Error; err != nil {
		t.Fatalf("reload customized secondary content: %v", err)
	}
	if customized.Title != "保留通用组织人工修改" {
		t.Fatalf("second multi-organization seed overwrote title: %q", customized.Title)
	}
	assertScopedSeedCount(t, tx, &model.Content{}, secondary.ID, contentIDs, 6)
}

func assertScopedSeedCount(t *testing.T, db *gorm.DB, value any, organizationID string, ids []string, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(value).Where("organization_id = ? AND id IN ?", organizationID, ids).Count(&count).Error; err != nil {
		t.Fatalf("count scoped seed records: %v", err)
	}
	if count != want {
		t.Fatalf("scoped seed record count = %d, want %d", count, want)
	}
}

func assertSeedCount(t *testing.T, db *gorm.DB, value any, query string, args any, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(value).Where(query, args).Count(&count).Error; err != nil {
		t.Fatalf("count seed records: %v", err)
	}
	if count != want {
		t.Fatalf("seed record count = %d, want %d", count, want)
	}
}

func submitApplicationFixture(t *testing.T, client *http.Client, cfg integrationConfig, gameID, email string) string {
	t.Helper()
	body := request(t, client, http.MethodPost, cfg.apiURL+"/api/v1/portal/organizations/"+cfg.organizationSlug+"/apply", "", map[string]string{
		"type": "whitelist", "class_name": "S3 集成测试", "name": "S3 Applicant",
		"game_id": gameID, "qq_number": "123456789", "email": email,
		"note": "自动化申请审批测试，结束后自动清理。",
	}, http.StatusCreated)
	var envelope apiEnvelope[struct {
		ID string `json:"id"`
	}]
	decodeJSON(t, body, &envelope)
	if envelope.Data.ID == "" {
		t.Fatal("submitted application did not return id")
	}
	return envelope.Data.ID
}

func cleanupApplicationFixture(t *testing.T, db *gorm.DB, applicationID string) {
	t.Helper()
	if applicationID == "" {
		return
	}
	for description, result := range map[string]*gorm.DB{
		"notification outbox": db.Where("target_type = ? AND target_id = ?", "application", applicationID).Delete(&model.NotificationOutbox{}),
		"audit events":        db.Where("target_type = ? AND target_id = ?", "application", applicationID).Delete(&model.AuditEvent{}),
		"application":         db.Where("id = ?", applicationID).Delete(&model.Application{}),
	} {
		if result.Error != nil {
			t.Errorf("cleanup %s: %v", description, result.Error)
		}
	}
}

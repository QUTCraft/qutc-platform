//go:build integration

package integration_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/config"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/database"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/serveradapter"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type applicationSyncDTO struct {
	ID        string     `json:"id"`
	Adapter   string     `json:"adapter"`
	Mode      string     `json:"mode"`
	Status    string     `json:"status"`
	Attempts  int        `json:"attempts"`
	Message   string     `json:"message"`
	LastError string     `json:"last_error"`
	Completed *time.Time `json:"completed_at"`
}

type adminApplicationDTO struct {
	ID             string              `json:"id"`
	Status         string              `json:"status"`
	Type           string              `json:"type"`
	GameID         string              `json:"game_id"`
	DecisionReason string              `json:"decision_reason"`
	ServerSync     *applicationSyncDTO `json:"server_sync"`
}

type adapterStatusDTO struct {
	Enabled bool   `json:"enabled"`
	Adapter string `json:"adapter"`
	Mode    string `json:"mode"`
	State   string `json:"state"`
}

type adapterCommandDTO struct {
	Accepted bool   `json:"accepted"`
	Executed bool   `json:"executed"`
	Mode     string `json:"mode"`
	Message  string `json:"message"`
}

func TestS3ApplicationApprovalAndMockAdapter(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	client := &http.Client{Timeout: 10 * time.Second}
	db := openIntegrationDB(t, cfg.mysqlDSN)
	ownerToken := loginAsOwner(t, client, cfg)

	gameID := "S3" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	email := strings.ToLower(gameID) + "@integration.invalid"
	applicationID := submitApplicationFixture(t, client, cfg, gameID, email)
	t.Cleanup(func() { cleanupApplicationFixture(t, db, applicationID) })

	var statusEnvelope apiEnvelope[adapterStatusDTO]
	decodeJSON(t, request(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/server/status", ownerToken, nil, http.StatusOK), &statusEnvelope)
	if statusEnvelope.Data.Mode != serveradapter.ModeMock || statusEnvelope.Data.State == "online" {
		t.Fatalf("adapter status = %+v, want explicit non-online mock mode", statusEnvelope.Data)
	}

	var applicationEnvelope apiEnvelope[adminApplicationDTO]
	approveURL := cfg.apiURL + "/api/v1/admin/applications/" + applicationID + "/approve"
	decodeJSON(t, request(t, client, http.MethodPost, approveURL, ownerToken, map[string]string{"reason": "集成测试审批通过。"}, http.StatusOK), &applicationEnvelope)
	application := applicationEnvelope.Data
	if application.Status != "approved" || application.ServerSync == nil {
		t.Fatalf("approved application = %+v, want separate server sync", application)
	}
	if application.DecisionReason != "集成测试审批通过。" {
		t.Fatalf("decision reason = %q, want persisted approval note", application.DecisionReason)
	}
	if application.ServerSync.Status != "succeeded" || application.ServerSync.Mode != serveradapter.ModeMock || application.ServerSync.Attempts != 1 {
		t.Fatalf("mock server sync = %+v, want one successful simulated attempt", application.ServerSync)
	}
	requireStatus(t, client, http.MethodPost, approveURL, ownerToken, nil, http.StatusConflict)

	var filteredEnvelope apiEnvelope[[]adminApplicationDTO]
	filteredURL := cfg.apiURL + "/api/v1/admin/applications?status=approved&type=whitelist&server_sync_status=succeeded&page=1&page_size=1&query=" + gameID
	decodeJSON(t, request(t, client, http.MethodGet, filteredURL, ownerToken, nil, http.StatusOK), &filteredEnvelope)
	if len(filteredEnvelope.Data) != 1 || filteredEnvelope.Data[0].ID != applicationID {
		t.Fatalf("filtered applications = %+v, want only %s", filteredEnvelope.Data, applicationID)
	}
	if filteredEnvelope.Meta.Page != 1 || filteredEnvelope.Meta.PageSize != 1 || filteredEnvelope.Meta.Total != 1 {
		t.Fatalf("filtered pagination = %+v, want page=1 page_size=1 total=1", filteredEnvelope.Meta)
	}
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/applications?server_sync_status=unknown", ownerToken, nil, http.StatusBadRequest)

	rejectedID := submitApplicationFixture(t, client, cfg, "Reject"+strings.ReplaceAll(uuid.NewString(), "-", "")[:10], "reject-"+uuid.NewString()+"@integration.invalid")
	t.Cleanup(func() { cleanupApplicationFixture(t, db, rejectedID) })
	rejectURL := cfg.apiURL + "/api/v1/admin/applications/" + rejectedID + "/reject"
	requireStatus(t, client, http.MethodPost, rejectURL, ownerToken, nil, http.StatusBadRequest)
	var rejectedEnvelope apiEnvelope[adminApplicationDTO]
	decodeJSON(t, request(t, client, http.MethodPost, rejectURL, ownerToken, map[string]string{"reason": "资料需要补充。"}, http.StatusOK), &rejectedEnvelope)
	if rejectedEnvelope.Data.Status != "rejected" || rejectedEnvelope.Data.DecisionReason != "资料需要补充。" {
		t.Fatalf("rejected application = %+v, want persisted rejection reason", rejectedEnvelope.Data)
	}

	var storedSync model.ApplicationServerSync
	if err := db.Where("application_id = ?", applicationID).First(&storedSync).Error; err != nil {
		t.Fatalf("load application sync: %v", err)
	}
	if storedSync.Status != "succeeded" || storedSync.Mode != serveradapter.ModeMock || storedSync.CompletedAt == nil {
		t.Fatalf("stored server sync = %+v", storedSync)
	}
	if err := db.Model(&model.ApplicationServerSync{}).Where("id = ?", storedSync.ID).Updates(map[string]any{
		"status":     "failed",
		"last_error": "模拟可重试故障。",
		"message":    "",
	}).Error; err != nil {
		t.Fatalf("prepare failed sync for HTTP retry: %v", err)
	}
	retryURL := cfg.apiURL + "/api/v1/admin/applications/" + applicationID + "/server-sync/retry"
	var retryEnvelope apiEnvelope[applicationSyncDTO]
	decodeJSON(t, request(t, client, http.MethodPost, retryURL, ownerToken, nil, http.StatusOK), &retryEnvelope)
	if retryEnvelope.Data.Status != "succeeded" || retryEnvelope.Data.Attempts != 2 {
		t.Fatalf("retried sync = %+v, want succeeded second attempt", retryEnvelope.Data)
	}
	requireStatus(t, client, http.MethodPost, retryURL, ownerToken, nil, http.StatusConflict)

	var commandEnvelope apiEnvelope[adapterCommandDTO]
	decodeJSON(t, request(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/server/commands", ownerToken, map[string]string{"command": "list"}, http.StatusOK), &commandEnvelope)
	if !commandEnvelope.Data.Accepted || commandEnvelope.Data.Executed || commandEnvelope.Data.Mode != serveradapter.ModeMock {
		t.Fatalf("mock command = %+v, want accepted simulation without real execution", commandEnvelope.Data)
	}
	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/server/commands", ownerToken, map[string]string{"command": "list\nop player"}, http.StatusForbidden)
}

func TestS3AdapterFailureDoesNotRollbackApproval(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	db := openIntegrationDB(t, cfg.mysqlDSN)

	var organization model.Organization
	if err := db.Where("slug = ?", cfg.organizationSlug).First(&organization).Error; err != nil {
		t.Fatalf("load organization: %v", err)
	}
	var owner model.User
	if err := db.Where("email = ?", cfg.adminEmail).First(&owner).Error; err != nil {
		t.Fatalf("load owner: %v", err)
	}

	application := model.Application{
		ID:             uuid.NewString(),
		OrganizationID: organization.ID,
		Type:           "whitelist",
		ClassName:      "S3 集成测试",
		ApplicantName:  "Adapter Failure",
		GameID:         "Failure" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		QQNumber:       "123456789",
		Email:          "s3-failure-" + uuid.NewString() + "@integration.invalid",
		Status:         "pending",
	}
	if err := db.Create(&application).Error; err != nil {
		t.Fatalf("create failure fixture: %v", err)
	}
	t.Cleanup(func() { cleanupApplicationFixture(t, db, application.ID) })

	decisionService := service.NewApplicationDecisionService(db, failingAdapter{})
	decided, syncRecord, err := decisionService.Decide(context.Background(), organization.ID, owner.ID, application.ID, "approved", "故障适配器测试。", "s3-failure-test")
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decided.Status != "approved" || syncRecord == nil || syncRecord.Status != "failed" {
		t.Fatalf("decision = %+v, sync = %+v; want approved decision with failed sync", decided, syncRecord)
	}
	if strings.Contains(syncRecord.LastError, "super-secret") {
		t.Fatalf("sync error leaked adapter details: %q", syncRecord.LastError)
	}

	var storedApplication model.Application
	if err := db.First(&storedApplication, "id = ?", application.ID).Error; err != nil {
		t.Fatalf("reload approved application: %v", err)
	}
	var storedSync model.ApplicationServerSync
	if err := db.Where("application_id = ?", application.ID).First(&storedSync).Error; err != nil {
		t.Fatalf("reload failed sync: %v", err)
	}
	if storedApplication.Status != "approved" || storedSync.Status != "failed" || storedSync.Attempts != 1 || storedSync.CompletedAt == nil {
		t.Fatalf("persisted decision = %q, sync = %+v", storedApplication.Status, storedSync)
	}

	retriedSync, err := decisionService.RetryServerSync(context.Background(), organization.ID, owner.ID, application.ID, "s3-failure-retry")
	if err != nil {
		t.Fatalf("RetryServerSync() error = %v", err)
	}
	if retriedSync.Status != "failed" || retriedSync.Attempts != 2 {
		t.Fatalf("retried failing sync = %+v, want failed second attempt", retriedSync)
	}
	var retryAuditCount int64
	if err := db.Model(&model.AuditEvent{}).
		Where("target_type = ? AND target_id = ? AND action IN ?", "application", application.ID, []string{"application.server_sync_retry", "application.server_sync_retry_result"}).
		Count(&retryAuditCount).Error; err != nil {
		t.Fatalf("count retry audits: %v", err)
	}
	if retryAuditCount != 2 {
		t.Fatalf("retry audit events = %d, want 2", retryAuditCount)
	}
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
	if err := tx.Where("application_id IN ?", applicationIDs).Delete(&model.ApplicationServerSync{}).Error; err != nil {
		t.Fatalf("delete seed sync fixtures: %v", err)
	}
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
	assertSeedCount(t, tx, &model.ApplicationServerSync{}, "application_id = ?", "application_demo_approved", 1)

	var content model.Content
	if err := tx.First(&content, "id = ?", "content_cms").Error; err != nil {
		t.Fatalf("reload customized seed content: %v", err)
	}
	if content.Title != "保留人工修改的标题" {
		t.Fatalf("second seed overwrote existing content title: %q", content.Title)
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
		"type":       "whitelist",
		"class_name": "S3 集成测试",
		"name":       "S3 Applicant",
		"game_id":    gameID,
		"qq_number":  "123456789",
		"email":      email,
		"note":       "自动化适配器闭环测试，结束后自动清理。",
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
		"application sync": db.Where("application_id = ?", applicationID).Delete(&model.ApplicationServerSync{}),
		"audit events":     db.Where("target_type = ? AND target_id = ?", "application", applicationID).Delete(&model.AuditEvent{}),
		"application":      db.Where("id = ?", applicationID).Delete(&model.Application{}),
	} {
		if result.Error != nil {
			t.Errorf("cleanup %s: %v", description, result.Error)
		}
	}
}

type failingAdapter struct{}

func (failingAdapter) Name() string { return "failing-test-adapter" }
func (failingAdapter) Mode() string { return "test" }
func (failingAdapter) Status(context.Context) (serveradapter.Status, error) {
	return serveradapter.Status{}, errors.New("super-secret status failure")
}
func (failingAdapter) Execute(context.Context, string) (serveradapter.Result, error) {
	return serveradapter.Result{}, errors.New("super-secret command failure")
}
func (failingAdapter) AddWhitelist(context.Context, string) (serveradapter.Result, error) {
	return serveradapter.Result{}, errors.New("super-secret whitelist failure")
}

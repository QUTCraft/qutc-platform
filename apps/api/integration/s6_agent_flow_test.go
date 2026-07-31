//go:build integration

package integration_test

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type aiProviderStatusDTO struct {
	Provider string `json:"provider"`
	Mode     string `json:"mode"`
	Model    string `json:"model"`
	Enabled  bool   `json:"enabled"`
}

type aiAgentCatalogDTO struct {
	Agents []struct {
		Key             string   `json:"key"`
		AllowedToolKeys []string `json:"allowed_tool_keys"`
	} `json:"agents"`
	Provider aiProviderStatusDTO `json:"provider"`
}

type aiConfigurationDTO struct {
	ID                    string              `json:"id"`
	Enabled               bool                `json:"enabled"`
	RunLimitPerHour       int                 `json:"run_limit_per_hour"`
	RequestTimeoutSeconds int                 `json:"request_timeout_seconds"`
	MaxSources            int                 `json:"max_sources"`
	MaxContextCharacters  int                 `json:"max_context_characters"`
	Provider              aiProviderStatusDTO `json:"provider"`
}

type aiKnowledgeResultDTO struct {
	ID         string `json:"id"`
	SourceType string `json:"source_type"`
	Title      string `json:"title"`
	Status     string `json:"status"`
}

type aiRunDTO struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	Mode           string `json:"mode"`
	Provider       string `json:"provider"`
	OutputTitle    string `json:"output_title"`
	OutputMarkdown string `json:"output_markdown"`
	FailureCode    string `json:"failure_code"`
	Citations      []struct {
		SourceID   string `json:"source_id"`
		SourceType string `json:"source_type"`
		Title      string `json:"title"`
	} `json:"citations"`
}

func TestS6AgentKnowledgeGenerationBoundary(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	client := &http.Client{Timeout: 10 * time.Second}
	db := openIntegrationDB(t, cfg.mysqlDSN)
	ownerToken := loginAsOwner(t, client, cfg)

	var organization model.Organization
	if err := db.Where("slug = ?", cfg.organizationSlug).First(&organization).Error; err != nil {
		t.Fatalf("load organization: %v", err)
	}
	var owner model.User
	if err := db.Where("email = ?", cfg.adminEmail).First(&owner).Error; err != nil {
		t.Fatalf("load owner: %v", err)
	}
	startedAt := time.Now().UTC().Add(-time.Second)
	var originalConfiguration model.AgentConfiguration
	configurationError := db.Where("organization_id = ?", organization.ID).First(&originalConfiguration).Error
	hadConfiguration := configurationError == nil
	if configurationError != nil && !errors.Is(configurationError, gorm.ErrRecordNotFound) {
		t.Fatalf("load original agent configuration: %v", configurationError)
	}

	uniqueTerm := "S6知识资料" + uuid.NewString()
	sourceID := uuid.NewString()
	otherOrganizationID := uuid.NewString()
	otherSourceID := uuid.NewString()
	if err := db.Create(&model.Content{
		ID: sourceID, OrganizationID: organization.ID, AuthorUserID: owner.ID,
		Title: uniqueTerm, Type: "knowledge", Category: "integration", Status: "draft",
		Excerpt: "八月社团技术分享会记录。", Body: "分享会包含 API 规范、内容协作和人工发布流程。",
	}).Error; err != nil {
		t.Fatalf("create knowledge fixture: %v", err)
	}
	if err := db.Create(&model.Organization{
		ID: otherOrganizationID, Slug: "s6-" + uuid.NewString(), Name: "S6 isolated organization",
	}).Error; err != nil {
		t.Fatalf("create isolated organization: %v", err)
	}
	if err := db.Create(&model.Content{
		ID: otherSourceID, OrganizationID: otherOrganizationID, AuthorUserID: owner.ID,
		Title: "S6 isolated knowledge", Type: "knowledge", Category: "integration", Status: "draft",
		Excerpt: "不应被其他组织的智能体读取。", Body: "isolated",
	}).Error; err != nil {
		t.Fatalf("create isolated knowledge fixture: %v", err)
	}
	createdRunIDs := make([]string, 0, 3)
	createdDraftIDs := make([]string, 0, 3)
	t.Cleanup(func() {
		for _, draftID := range createdDraftIDs {
			cleanupContentFixture(t, db, draftID)
		}
		if len(createdRunIDs) > 0 {
			_ = db.Where("run_id IN ?", createdRunIDs).Delete(&model.AgentCitation{}).Error
			_ = db.Where("target_type = ? AND target_id IN ?", "agent_run", createdRunIDs).Delete(&model.AuditEvent{}).Error
			_ = db.Where("id IN ?", createdRunIDs).Delete(&model.AgentRun{}).Error
		}
		_ = db.Where(
			"organization_id = ? AND action = ? AND target_type = ? AND created_at >= ?",
			organization.ID, "ai.config_update", "agent_configuration", startedAt,
		).Delete(&model.AuditEvent{}).Error
		if hadConfiguration {
			_ = db.Save(&originalConfiguration).Error
		} else {
			_ = db.Where("organization_id = ?", organization.ID).Delete(&model.AgentConfiguration{}).Error
		}
		_ = db.Where("id IN ?", []string{sourceID, otherSourceID}).Delete(&model.Content{}).Error
		_ = db.Where("id = ?", otherOrganizationID).Delete(&model.Organization{}).Error
	})

	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/ai/config", "", nil, http.StatusUnauthorized)
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/ai/agents", "", nil, http.StatusUnauthorized)

	configurationBody := request(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/ai/config", ownerToken, nil, http.StatusOK)
	var configurationEnvelope apiEnvelope[aiConfigurationDTO]
	decodeJSON(t, configurationBody, &configurationEnvelope)
	if configurationEnvelope.Data.RunLimitPerHour < 1 || configurationEnvelope.Data.MaxSources < 1 ||
		configurationEnvelope.Data.Provider.Mode != "mock" {
		t.Fatalf("initial agent configuration = %+v", configurationEnvelope.Data)
	}

	disabledConfiguration := map[string]any{
		"enabled": false, "run_limit_per_hour": 17, "request_timeout_seconds": 25,
		"max_sources": 3, "max_context_characters": 24000,
	}
	disabledBody := request(t, client, http.MethodPatch, cfg.apiURL+"/api/v1/admin/ai/config", ownerToken, disabledConfiguration, http.StatusOK)
	decodeJSON(t, disabledBody, &configurationEnvelope)
	if configurationEnvelope.Data.Enabled || configurationEnvelope.Data.RunLimitPerHour != 17 ||
		configurationEnvelope.Data.MaxSources != 3 || configurationEnvelope.Data.MaxContextCharacters != 24000 {
		t.Fatalf("disabled agent configuration = %+v", configurationEnvelope.Data)
	}
	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/ai/runs", ownerToken, map[string]any{
		"agent_key":    "content-copilot",
		"task":         "停用状态下不应创建运行",
		"context_refs": []map[string]string{{"type": "content", "id": sourceID}},
		"output_mode":  "proposal",
	}, http.StatusConflict)

	enabledConfiguration := map[string]any{
		"enabled": true, "run_limit_per_hour": 17, "request_timeout_seconds": 25,
		"max_sources": 3, "max_context_characters": 24000,
	}
	enabledBody := request(t, client, http.MethodPatch, cfg.apiURL+"/api/v1/admin/ai/config", ownerToken, enabledConfiguration, http.StatusOK)
	decodeJSON(t, enabledBody, &configurationEnvelope)
	if !configurationEnvelope.Data.Enabled || configurationEnvelope.Data.ID == "" {
		t.Fatalf("enabled agent configuration = %+v", configurationEnvelope.Data)
	}

	catalogBody := request(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/ai/agents", ownerToken, nil, http.StatusOK)
	var catalogEnvelope apiEnvelope[aiAgentCatalogDTO]
	decodeJSON(t, catalogBody, &catalogEnvelope)
	if len(catalogEnvelope.Data.Agents) != 1 || catalogEnvelope.Data.Agents[0].Key != "content-copilot" {
		t.Fatalf("agent catalog = %+v", catalogEnvelope.Data)
	}
	if !catalogEnvelope.Data.Provider.Enabled || catalogEnvelope.Data.Provider.Mode != "mock" || catalogEnvelope.Data.Provider.Provider != "mock" {
		t.Fatalf("provider status = %+v, want explicit development mock", catalogEnvelope.Data.Provider)
	}

	searchBody := request(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/ai/knowledge/search", ownerToken, map[string]any{
		"query": uniqueTerm,
		"limit": 5,
	}, http.StatusOK)
	var searchEnvelope apiEnvelope[[]aiKnowledgeResultDTO]
	decodeJSON(t, searchBody, &searchEnvelope)
	if len(searchEnvelope.Data) != 1 || searchEnvelope.Data[0].ID != sourceID || searchEnvelope.Data[0].SourceType != "content" {
		t.Fatalf("knowledge search = %+v, want current organization source", searchEnvelope.Data)
	}

	beforeContentCount := countOrganizationContent(t, db, organization.ID)
	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/ai/runs", ownerToken, map[string]any{
		"agent_key":    "content-copilot",
		"task":         "根据资料生成一篇技术分享会门户动态提案",
		"context_refs": []map[string]string{{"type": "content", "id": otherSourceID}},
		"output_mode":  "proposal",
	}, http.StatusNotFound)

	for iteration := 1; iteration <= 3; iteration++ {
		t.Run(fmt.Sprintf("confirmed_content_round_%d", iteration), func(t *testing.T) {
			createBody := request(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/ai/runs", ownerToken, map[string]any{
				"agent_key":    "content-copilot",
				"task":         fmt.Sprintf("根据资料生成第 %d 轮技术分享会门户动态提案", iteration),
				"context_refs": []map[string]string{{"type": "content", "id": sourceID}},
				"output_mode":  "proposal",
			}, http.StatusAccepted)
			var createEnvelope apiEnvelope[aiRunDTO]
			decodeJSON(t, createBody, &createEnvelope)
			createdRunID := createEnvelope.Data.ID
			createdRunIDs = append(createdRunIDs, createdRunID)
			if createdRunID == "" || (createEnvelope.Data.Status != "queued" && createEnvelope.Data.Status != "running" && createEnvelope.Data.Status != "succeeded") {
				t.Fatalf("created run = %+v", createEnvelope.Data)
			}

			run := waitForAgentRun(t, client, cfg, ownerToken, createdRunID)
			if run.Status != "succeeded" || run.Mode != "mock" || run.Provider != "mock" || run.FailureCode != "" {
				t.Fatalf("terminal run = %+v", run)
			}
			if len(run.Citations) != 1 || run.Citations[0].SourceID != sourceID || run.Citations[0].SourceType != "content" {
				t.Fatalf("citations = %+v", run.Citations)
			}
			if !strings.Contains(run.OutputMarkdown, "开发 Mock") || !strings.Contains(run.OutputMarkdown, sourceID) {
				t.Fatalf("mock output did not identify mode and citation: %s", run.OutputMarkdown)
			}
			expectedCountBeforeConfirmation := beforeContentCount + int64(iteration-1)
			if countOrganizationContent(t, db, organization.ID) != expectedCountBeforeConfirmation {
				t.Fatal("AI proposal unexpectedly created or modified CMS content")
			}

			draftBody := request(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/content", ownerToken, map[string]any{
				"title":                  run.OutputTitle,
				"type":                   "news",
				"category":               "AI integration",
				"knowledge_directory_id": "",
				"excerpt":                "人工确认后创建的 AI 内容提案草稿。",
				"body":                   run.OutputMarkdown,
			}, http.StatusCreated)
			var draftEnvelope apiEnvelope[contentDTO]
			decodeJSON(t, draftBody, &draftEnvelope)
			createdDraftID := draftEnvelope.Data.ID
			createdDraftIDs = append(createdDraftIDs, createdDraftID)
			if createdDraftID == "" || draftEnvelope.Data.Status != "draft" || !strings.Contains(draftEnvelope.Data.Body, sourceID) {
				t.Fatalf("AI confirmed draft = %+v", draftEnvelope.Data)
			}
			requireStatus(t, client, http.MethodGet, portalContentURL(cfg, createdDraftID), "", nil, http.StatusNotFound)

			publishedDraft := changeContentStatus(t, client, cfg, ownerToken, createdDraftID, "publish")
			if publishedDraft.Status != "published" {
				t.Fatalf("AI confirmed draft publish status = %q", publishedDraft.Status)
			}
			publicDraft := getPublicContent(t, client, cfg, createdDraftID)
			if body, _ := publicDraft["body"].(string); !strings.Contains(body, sourceID) {
				t.Fatalf("published AI draft lost source identifier: %#v", publicDraft["body"])
			}
			archivedDraft := changeContentStatus(t, client, cfg, ownerToken, createdDraftID, "archive")
			if archivedDraft.Status != "archived" {
				t.Fatalf("AI confirmed draft archive status = %q", archivedDraft.Status)
			}
			requireStatus(t, client, http.MethodGet, portalContentURL(cfg, createdDraftID), "", nil, http.StatusNotFound)
		})
	}

	lastRunID := createdRunIDs[len(createdRunIDs)-1]
	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/ai/runs/"+lastRunID+"/cancel", ownerToken, nil, http.StatusConflict)
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/ai/runs/"+uuid.NewString(), ownerToken, nil, http.StatusNotFound)

	var auditCount int64
	if err := db.Model(&model.AuditEvent{}).
		Where("organization_id = ? AND target_type = ? AND target_id IN ? AND action IN ?", organization.ID, "agent_run", createdRunIDs, []string{"ai.run_create", "ai.run_result"}).
		Count(&auditCount).Error; err != nil {
		t.Fatalf("count agent audit events: %v", err)
	}
	if auditCount < 6 {
		t.Fatalf("agent audit count = %d, want create and result events for three runs", auditCount)
	}
	var configurationAuditCount int64
	if err := db.Model(&model.AuditEvent{}).
		Where(
			"organization_id = ? AND action = ? AND target_type = ? AND created_at >= ?",
			organization.ID, "ai.config_update", "agent_configuration", startedAt,
		).Count(&configurationAuditCount).Error; err != nil {
		t.Fatalf("count agent configuration audit events: %v", err)
	}
	if configurationAuditCount < 2 {
		t.Fatalf("agent configuration audit count = %d, want disable and enable events", configurationAuditCount)
	}
}

func waitForAgentRun(t *testing.T, client *http.Client, cfg integrationConfig, token, runID string) aiRunDTO {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		body := request(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/ai/runs/"+runID, token, nil, http.StatusOK)
		var envelope apiEnvelope[aiRunDTO]
		decodeJSON(t, body, &envelope)
		if envelope.Data.Status == "succeeded" || envelope.Data.Status == "failed" || envelope.Data.Status == "canceled" {
			return envelope.Data
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("agent run %s did not reach a terminal state", runID)
	return aiRunDTO{}
}

func countOrganizationContent(t *testing.T, db *gorm.DB, organizationID string) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&model.Content{}).Where("organization_id = ?", organizationID).Count(&count).Error; err != nil {
		t.Fatalf("count organization content: %v", err)
	}
	return count
}

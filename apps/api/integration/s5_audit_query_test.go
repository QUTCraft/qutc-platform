//go:build integration

package integration_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/google/uuid"
)

type auditEventDTO struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	Result      string `json:"result"`
	RequestID   string `json:"request_id"`
	ActorUserID string `json:"actor_user_id"`
	ActorName   string `json:"actor_name"`
	CreatedAt   string `json:"created_at"`
}

func TestS5AuditEventQuery(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	client := &http.Client{Timeout: 10 * time.Second}
	db := openIntegrationDB(t, cfg.mysqlDSN)
	ownerToken := loginAsOwner(t, client, cfg)

	var organization model.Organization
	if err := db.Where("slug = ?", cfg.organizationSlug).First(&organization).Error; err != nil {
		t.Fatalf("find organization: %v", err)
	}
	var owner model.User
	if err := db.Where("email = ?", cfg.adminEmail).First(&owner).Error; err != nil {
		t.Fatalf("find owner user: %v", err)
	}

	now := time.Now().UTC()
	sharedRequestID := "req_s5_" + uuid.NewString()
	events := []model.AuditEvent{
		{ID: uuid.NewString(), OrganizationID: organization.ID, ActorUserID: owner.ID, Action: "content.publish", TargetType: "content", TargetID: "content_s5_a", Result: "success", RequestID: sharedRequestID, CreatedAt: now},
		{ID: uuid.NewString(), OrganizationID: organization.ID, ActorUserID: owner.ID, Action: "application.approve", TargetType: "application", TargetID: "application_s5", Result: "success", RequestID: sharedRequestID, CreatedAt: now.Add(time.Second)},
		{ID: uuid.NewString(), OrganizationID: organization.ID, ActorUserID: owner.ID, Action: "server.command", TargetType: "server", TargetID: "", Result: "failed", RequestID: "req_s5_isolated_" + uuid.NewString(), CreatedAt: now.Add(2 * time.Second)},
		{ID: uuid.NewString(), OrganizationID: uuid.NewString(), ActorUserID: owner.ID, Action: "content.publish", TargetType: "content", TargetID: "content_other_org", Result: "success", RequestID: "req_s5_other_org", CreatedAt: now.Add(3 * time.Second)},
	}
	for _, event := range events {
		if err := db.Create(&event).Error; err != nil {
			t.Fatalf("create audit event: %v", err)
		}
	}
	t.Cleanup(func() {
		db.Where("id IN ?", []string{events[0].ID, events[1].ID, events[2].ID, events[3].ID}).Delete(&model.AuditEvent{})
	})

	currentOrgIDs := []string{events[0].ID, events[1].ID, events[2].ID}

	// 无筛选：包含当前组织事件，不泄露其他组织。
	list := fetchAudit(t, client, cfg, ownerToken, "")
	if !auditContainsAll(list, currentOrgIDs) || auditContainsID(list, events[3].ID) {
		t.Fatalf("audit list missing current org events or leaked other org; got %d items", len(list))
	}
	for _, item := range list {
		if item.ID == events[0].ID && item.ActorName != owner.DisplayName {
			t.Fatalf("actor_name = %q, want %q", item.ActorName, owner.DisplayName)
		}
	}

	// 按 action 筛选：只返回当前组织的 content.publish，不返回其他组织同名事件。
	list = fetchAudit(t, client, cfg, ownerToken, "action=content.publish")
	if !auditContainsID(list, events[0].ID) || auditContainsAny(list, []string{events[1].ID, events[2].ID, events[3].ID}) {
		t.Fatalf("action filter failed; got IDs: %v", auditIDs(list))
	}

	// 按 result 筛选。
	list = fetchAudit(t, client, cfg, ownerToken, "result=failed")
	if !auditContainsID(list, events[2].ID) || auditContainsAny(list, []string{events[0].ID, events[1].ID}) {
		t.Fatalf("result filter failed; got IDs: %v", auditIDs(list))
	}

	// 按 request_id 筛选：串联定位共享 request_id 的两条事件。
	list = fetchAudit(t, client, cfg, ownerToken, "request_id="+sharedRequestID)
	if !auditContainsAll(list, []string{events[0].ID, events[1].ID}) || auditContainsID(list, events[2].ID) {
		t.Fatalf("request_id filter failed; got IDs: %v", auditIDs(list))
	}

	// 按 target_type 筛选。
	list = fetchAudit(t, client, cfg, ownerToken, "target_type=server")
	if !auditContainsID(list, events[2].ID) || auditContainsAny(list, []string{events[0].ID, events[1].ID}) {
		t.Fatalf("target_type filter failed; got IDs: %v", auditIDs(list))
	}

	// 分页：page_size=1 返回单条且 total 覆盖当前组织事件。
	var pagedEnvelope apiEnvelope[[]auditEventDTO]
	decodeJSON(t, request(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/audit?page=1&page_size=1", ownerToken, nil, http.StatusOK), &pagedEnvelope)
	if len(pagedEnvelope.Data) != 1 || pagedEnvelope.Meta.PageSize != 1 || pagedEnvelope.Meta.Total < int64(len(currentOrgIDs)) {
		t.Fatalf("pagination = %d items, page_size=%d, total=%d; want 1 item and total>=%d", len(pagedEnvelope.Data), pagedEnvelope.Meta.PageSize, pagedEnvelope.Meta.Total, len(currentOrgIDs))
	}

	// 非法 result 返回 400。
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/audit?result=bogus", ownerToken, nil, http.StatusBadRequest)

	// 未鉴权访问返回 401。
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/audit", "", nil, http.StatusUnauthorized)
}

func fetchAudit(t *testing.T, client *http.Client, cfg integrationConfig, token, query string) []auditEventDTO {
	t.Helper()
	url := cfg.apiURL + "/api/v1/admin/audit"
	if query != "" {
		url += "?" + query
	}
	var envelope apiEnvelope[[]auditEventDTO]
	decodeJSON(t, request(t, client, http.MethodGet, url, token, nil, http.StatusOK), &envelope)
	return envelope.Data
}

func auditContainsID(items []auditEventDTO, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func auditContainsAll(items []auditEventDTO, ids []string) bool {
	for _, id := range ids {
		if !auditContainsID(items, id) {
			return false
		}
	}
	return true
}

func auditContainsAny(items []auditEventDTO, ids []string) bool {
	for _, id := range ids {
		if auditContainsID(items, id) {
			return true
		}
	}
	return false
}

func auditIDs(items []auditEventDTO) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.ID)
	}
	return result
}

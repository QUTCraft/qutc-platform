//go:build integration

package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/google/uuid"
)

type observabilityAuditEventDTO struct {
	ID          string `json:"id"`
	ActorUserID string `json:"actor_user_id"`
	ActorName   string `json:"actor_name"`
	Action      string `json:"action"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	Result      string `json:"result"`
	RequestID   string `json:"request_id"`
	CreatedAt   string `json:"created_at"`
}

func TestS5ReadinessRequestIDAndOrganizationAuditQuery(t *testing.T) {
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

	requestID := "s5-audit-" + uuid.NewString()
	eventID := uuid.NewString()
	targetID := uuid.NewString()
	otherOrganizationID := uuid.NewString()
	otherEventID := uuid.NewString()
	if err := db.Create(&model.Organization{ID: otherOrganizationID, Slug: "s5-" + uuid.NewString(), Name: "S5 isolated organization"}).Error; err != nil {
		t.Fatalf("create isolated organization: %v", err)
	}
	if err := db.Create(&model.AuditEvent{
		ID: eventID, OrganizationID: organization.ID, ActorUserID: owner.ID,
		Action: "integration.audit_query", TargetType: "observability", TargetID: targetID,
		Result: "success", RequestID: requestID, CreatedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("create audit fixture: %v", err)
	}
	if err := db.Create(&model.AuditEvent{
		ID: otherEventID, OrganizationID: otherOrganizationID, ActorUserID: owner.ID,
		Action: "integration.audit_query", TargetType: "observability", TargetID: uuid.NewString(),
		Result: "success", RequestID: requestID, CreatedAt: time.Now().UTC(),
	}).Error; err != nil {
		t.Fatalf("create isolated audit fixture: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Where("id IN ?", []string{eventID, otherEventID}).Delete(&model.AuditEvent{}).Error; err != nil {
			t.Errorf("cleanup audit fixtures: %v", err)
		}
		if err := db.Where("id = ?", otherOrganizationID).Delete(&model.Organization{}).Error; err != nil {
			t.Errorf("cleanup isolated organization: %v", err)
		}
	})

	readyResponse := doRequestWithID(t, client, http.MethodGet, cfg.apiURL+"/readyz", "", "s5-ready-request", http.StatusOK)
	defer readyResponse.Body.Close()
	var readiness struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	decodeResponseJSON(t, readyResponse, &readiness)
	if readiness.Status != "ready" || readiness.Checks["mysql"] != "ok" || readiness.Checks["redis"] != "ok" {
		t.Fatalf("readiness = %+v, want MySQL and Redis ready", readiness)
	}
	if got := readyResponse.Header.Get("X-Request-ID"); got != "s5-ready-request" {
		t.Fatalf("readiness request ID = %q, want propagated value", got)
	}

	unsafeResponse := doRequestWithID(t, client, http.MethodGet, cfg.apiURL+"/healthz", "", "unsafe request id", http.StatusOK)
	defer unsafeResponse.Body.Close()
	if got := unsafeResponse.Header.Get("X-Request-ID"); got == "" || got == "unsafe request id" {
		t.Fatalf("unsafe request ID was not replaced: %q", got)
	}

	auditURL := cfg.apiURL + "/api/v1/admin/audit?request_id=" + url.QueryEscape(requestID)
	body := request(t, client, http.MethodGet, auditURL, ownerToken, nil, http.StatusOK)
	var envelope apiEnvelope[[]observabilityAuditEventDTO]
	decodeJSON(t, body, &envelope)
	if envelope.Meta.Total != 1 || len(envelope.Data) != 1 {
		t.Fatalf("audit result total=%d items=%d, want only current organization event", envelope.Meta.Total, len(envelope.Data))
	}
	event := envelope.Data[0]
	if event.ID != eventID || event.TargetID != targetID || event.ActorUserID != owner.ID || event.ActorName == "" {
		t.Fatalf("audit event = %+v, want current organization fixture", event)
	}

	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/audit?date_from=2026-99-99", ownerToken, nil, http.StatusBadRequest)
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/audit?date_from=2026-07-30&date_to=2026-07-29", ownerToken, nil, http.StatusBadRequest)
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/audit", "", nil, http.StatusUnauthorized)
}

func doRequestWithID(t *testing.T, client *http.Client, method, target, token, requestID string, expectedStatus int) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Request-ID", requestID)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, target, err)
	}
	if response.StatusCode != expectedStatus {
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		t.Fatalf("%s %s status = %d, want %d; body=%s", method, target, response.StatusCode, expectedStatus, body)
	}
	return response
}

func decodeResponseJSON(t *testing.T, response *http.Response, destination any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(destination); err != nil {
		t.Fatalf("decode response JSON: %v", err)
	}
}

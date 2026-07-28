//go:build integration

package integration_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/portalmanifest"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type portalConfigurationDTO struct {
	ID             string                   `json:"id"`
	DraftManifest  *portalmanifest.Manifest `json:"draft_manifest"`
	ActiveManifest *portalmanifest.Manifest `json:"active_manifest"`
	Active         bool                     `json:"active"`
	UpdatedAt      *time.Time               `json:"updated_at"`
	ActivatedAt    *time.Time               `json:"activated_at"`
}

type portalRuntimeConfigurationDTO struct {
	Manifest    portalmanifest.Manifest `json:"manifest"`
	Source      string                  `json:"source"`
	ActivatedAt *time.Time              `json:"activated_at"`
}

func TestS4PortalConfigurationDraftAndEnable(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	client := &http.Client{Timeout: 10 * time.Second}
	db := openIntegrationDB(t, cfg.mysqlDSN)
	ownerToken := loginAsOwner(t, client, cfg)

	var organization model.Organization
	if err := db.Where("slug = ?", cfg.organizationSlug).First(&organization).Error; err != nil {
		t.Fatalf("load organization: %v", err)
	}
	var original model.PortalConfiguration
	originalErr := db.Where("organization_id = ?", organization.ID).First(&original).Error
	if originalErr != nil && !errors.Is(originalErr, gorm.ErrRecordNotFound) {
		t.Fatalf("load original portal configuration: %v", originalErr)
	}
	hadOriginal := originalErr == nil
	startedAt := time.Now().UTC().Add(-time.Second)
	t.Cleanup(func() {
		var current model.PortalConfiguration
		if err := db.Where("organization_id = ?", organization.ID).First(&current).Error; err == nil {
			_ = db.Where("target_id = ? AND action IN ? AND created_at >= ?", current.ID, []string{"portal.config_update", "portal.config_enable", "portal.config_restore_default"}, startedAt).Delete(&model.AuditEvent{}).Error
			if hadOriginal {
				_ = db.Save(&original).Error
			} else {
				_ = db.Delete(&current).Error
			}
		}
	})

	configURL := cfg.apiURL + "/api/v1/admin/portal/config"
	enableURL := configURL + "/enable"
	runtimeURL := cfg.apiURL + "/api/v1/portal/organizations/" + cfg.organizationSlug + "/configuration"
	requireStatus(t, client, http.MethodGet, configURL, "", nil, http.StatusUnauthorized)

	invalid := validPortalManifest()
	invalid.Entry = "https://example.invalid/portal.html"
	requireStatus(t, client, http.MethodPatch, configURL, ownerToken, map[string]any{"manifest": invalid}, http.StatusBadRequest)

	var beforeEnvelope apiEnvelope[portalConfigurationDTO]
	decodeJSON(t, request(t, client, http.MethodGet, configURL, ownerToken, nil, http.StatusOK), &beforeEnvelope)
	previousActiveID := ""
	if beforeEnvelope.Data.ActiveManifest != nil {
		previousActiveID = beforeEnvelope.Data.ActiveManifest.ID
	}

	manifest := validPortalManifest()
	manifest.ID = "s4-" + uuid.NewString()[:8]
	var savedEnvelope apiEnvelope[portalConfigurationDTO]
	decodeJSON(t, request(t, client, http.MethodPatch, configURL, ownerToken, map[string]any{"manifest": manifest}, http.StatusOK), &savedEnvelope)
	if savedEnvelope.Data.DraftManifest == nil || savedEnvelope.Data.DraftManifest.ID != manifest.ID {
		t.Fatalf("saved draft = %+v, want manifest %s", savedEnvelope.Data.DraftManifest, manifest.ID)
	}
	if savedEnvelope.Data.ActiveManifest != nil && savedEnvelope.Data.ActiveManifest.ID == manifest.ID {
		t.Fatal("saving a draft changed the active manifest")
	}
	if savedEnvelope.Data.ActiveManifest == nil && previousActiveID != "" {
		t.Fatal("saving a draft removed the previous active manifest")
	}
	var runtimeBeforeEnable apiEnvelope[portalRuntimeConfigurationDTO]
	decodeJSON(t, request(t, client, http.MethodGet, runtimeURL, "", nil, http.StatusOK), &runtimeBeforeEnable)
	if runtimeBeforeEnable.Data.Manifest.ID == manifest.ID {
		t.Fatal("public runtime used a draft before it was enabled")
	}

	var reloadedEnvelope apiEnvelope[portalConfigurationDTO]
	decodeJSON(t, request(t, client, http.MethodGet, configURL, ownerToken, nil, http.StatusOK), &reloadedEnvelope)
	if reloadedEnvelope.Data.DraftManifest == nil || reloadedEnvelope.Data.DraftManifest.ID != manifest.ID || reloadedEnvelope.Data.UpdatedAt == nil {
		t.Fatalf("reloaded configuration = %+v, want persisted draft", reloadedEnvelope.Data)
	}

	var enabledEnvelope apiEnvelope[portalConfigurationDTO]
	decodeJSON(t, request(t, client, http.MethodPost, enableURL, ownerToken, nil, http.StatusOK), &enabledEnvelope)
	if !enabledEnvelope.Data.Active || enabledEnvelope.Data.ActiveManifest == nil || enabledEnvelope.Data.ActiveManifest.ID != manifest.ID || enabledEnvelope.Data.ActivatedAt == nil {
		t.Fatalf("enabled configuration = %+v, want active manifest %s", enabledEnvelope.Data, manifest.ID)
	}
	var runtimeEnvelope apiEnvelope[portalRuntimeConfigurationDTO]
	decodeJSON(t, request(t, client, http.MethodGet, runtimeURL, "", nil, http.StatusOK), &runtimeEnvelope)
	if runtimeEnvelope.Data.Source != "active" || runtimeEnvelope.Data.Manifest.ID != manifest.ID || runtimeEnvelope.Data.ActivatedAt == nil {
		t.Fatalf("runtime configuration = %+v, want active manifest %s", runtimeEnvelope.Data, manifest.ID)
	}

	if err := db.Model(&model.PortalConfiguration{}).Where("id = ?", enabledEnvelope.Data.ID).Update("active_manifest_json", `{"schema":"broken"}`).Error; err != nil {
		t.Fatalf("corrupt active manifest fixture: %v", err)
	}
	var fallbackEnvelope apiEnvelope[portalRuntimeConfigurationDTO]
	decodeJSON(t, request(t, client, http.MethodGet, runtimeURL, "", nil, http.StatusOK), &fallbackEnvelope)
	if fallbackEnvelope.Data.Source != "default" || fallbackEnvelope.Data.Manifest.ID != "qutc-md3" || fallbackEnvelope.Data.Manifest.Fallback != "md3" {
		t.Fatalf("corrupted runtime fallback = %+v, want built-in MD3", fallbackEnvelope.Data)
	}
	var restoredEnvelope apiEnvelope[portalConfigurationDTO]
	decodeJSON(t, request(t, client, http.MethodPost, configURL+"/restore-default", ownerToken, nil, http.StatusOK), &restoredEnvelope)
	if restoredEnvelope.Data.ActiveManifest == nil || restoredEnvelope.Data.ActiveManifest.ID != "qutc-md3" ||
		restoredEnvelope.Data.DraftManifest == nil || restoredEnvelope.Data.DraftManifest.ID != "qutc-md3" {
		t.Fatalf("restored configuration = %+v, want default MD3 as draft and active", restoredEnvelope.Data)
	}
	var restoredRuntimeEnvelope apiEnvelope[portalRuntimeConfigurationDTO]
	decodeJSON(t, request(t, client, http.MethodGet, runtimeURL, "", nil, http.StatusOK), &restoredRuntimeEnvelope)
	if restoredRuntimeEnvelope.Data.Manifest.ID != "qutc-md3" || restoredRuntimeEnvelope.Data.Manifest.Entry != "/index.html" {
		t.Fatalf("restored runtime = %+v, want default MD3", restoredRuntimeEnvelope.Data)
	}

	var auditCount int64
	if err := db.Model(&model.AuditEvent{}).
		Where("target_id = ? AND action IN ? AND created_at >= ?", enabledEnvelope.Data.ID, []string{"portal.config_update", "portal.config_enable", "portal.config_restore_default"}, startedAt).
		Count(&auditCount).Error; err != nil {
		t.Fatalf("count portal configuration audits: %v", err)
	}
	if auditCount != 3 {
		t.Fatalf("portal configuration audit events = %d, want 3", auditCount)
	}
}

func validPortalManifest() portalmanifest.Manifest {
	return portalmanifest.Manifest{
		Schema:      portalmanifest.SchemaV1,
		ID:          "qutcraft-md3",
		Version:     "0.1.0",
		DisplayName: "QUTCraft MD3 Portal",
		Entry:       "/index.html",
		Theme:       portalmanifest.ThemeRef{Mode: "md3"},
		Capabilities: []string{
			"organization.read",
			"public_content.read",
			"projects.read",
			"assets.read",
			"knowledge.read",
			"server.status.read",
		},
		Fallback: "md3",
	}
}

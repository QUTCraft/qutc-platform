//go:build integration

package integration_test

import (
	"bytes"
	"encoding/base64"
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

type organizationProfileDTO struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	ShortName    string `json:"short_name"`
	Tagline      string `json:"tagline"`
	Introduction string `json:"introduction"`
	ContactEmail string `json:"contact_email"`
	FilingNumber string `json:"filing_number"`
	LogoAssetID  string `json:"logo_asset_id"`
	LogoURL      string `json:"logo_url"`
	SocialLinks  []struct {
		Label string `json:"label"`
		Href  string `json:"href"`
	} `json:"social_links"`
	IsPublic bool `json:"is_public"`
}

func TestS4OrganizationProfileAndPublicBoundary(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	client := &http.Client{Timeout: 10 * time.Second}
	db := openIntegrationDB(t, cfg.mysqlDSN)
	ownerToken := loginAsOwner(t, client, cfg)

	var original model.Organization
	if err := db.Where("slug = ?", cfg.organizationSlug).First(&original).Error; err != nil {
		t.Fatalf("load organization: %v", err)
	}
	startedAt := time.Now().UTC().Add(-time.Second)
	logoAssetID := ""
	t.Cleanup(func() {
		_ = db.Save(&original).Error
		if logoAssetID != "" {
			var assetCount int64
			if err := db.Model(&model.MediaAsset{}).Where("id = ?", logoAssetID).Count(&assetCount).Error; err == nil && assetCount > 0 {
				requireStatus(t, client, http.MethodDelete, cfg.apiURL+"/api/v1/admin/assets/"+logoAssetID, ownerToken, nil, http.StatusOK)
			}
			_ = db.Where("target_id = ? AND created_at >= ?", logoAssetID, startedAt).Delete(&model.AuditEvent{}).Error
		}
		_ = db.Where("target_id = ? AND action = ? AND created_at >= ?", original.ID, "organization.profile_update", startedAt).Delete(&model.AuditEvent{}).Error
	})

	adminURL := cfg.apiURL + "/api/v1/admin/organization"
	portalURL := cfg.apiURL + "/api/v1/portal/organizations/" + cfg.organizationSlug
	requireStatus(t, client, http.MethodGet, adminURL, "", nil, http.StatusUnauthorized)
	var current apiEnvelope[organizationProfileDTO]
	decodeJSON(t, request(t, client, http.MethodGet, adminURL, ownerToken, nil, http.StatusOK), &current)
	pngBytes, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode organization logo fixture: %v", err)
	}
	logoAssetID = uploadAssetFixture(t, client, cfg.apiURL+"/api/v1/admin/assets", ownerToken, "", pngBytes)

	payload := map[string]any{
		"name": current.Data.Name, "short_name": current.Data.ShortName,
		"tagline": "S4 organization profile " + uuid.NewString()[:8], "introduction": current.Data.Introduction,
		"contact_email": current.Data.ContactEmail, "filing_number": "鲁ICP备2026000000号-1",
		"logo_asset_id": logoAssetID,
		"social_links":  current.Data.SocialLinks, "is_public": true,
	}
	var updated apiEnvelope[organizationProfileDTO]
	decodeJSON(t, request(t, client, http.MethodPatch, adminURL, ownerToken, payload, http.StatusOK), &updated)
	if updated.Data.Tagline != payload["tagline"] || updated.Data.FilingNumber != payload["filing_number"] || updated.Data.LogoAssetID != logoAssetID || updated.Data.LogoURL == "" || !updated.Data.IsPublic {
		t.Fatalf("updated organization = %+v", updated.Data)
	}
	var public apiEnvelope[organizationProfileDTO]
	decodeJSON(t, request(t, client, http.MethodGet, portalURL, "", nil, http.StatusOK), &public)
	if public.Data.Tagline != updated.Data.Tagline {
		t.Fatalf("portal tagline = %q, want %q", public.Data.Tagline, updated.Data.Tagline)
	}
	logoResponse := request(t, client, http.MethodGet, cfg.apiURL+public.Data.LogoURL, "", nil, http.StatusOK)
	if !bytes.Equal(logoResponse, pngBytes) {
		t.Fatal("public organization logo differs from uploaded image")
	}
	var deleted apiEnvelope[struct {
		ClearedLogo bool `json:"cleared_logo"`
	}]
	decodeJSON(t, request(t, client, http.MethodDelete, cfg.apiURL+"/api/v1/admin/assets/"+logoAssetID, ownerToken, nil, http.StatusOK), &deleted)
	if !deleted.Data.ClearedLogo {
		t.Fatal("deleting the active organization logo did not clear its reference")
	}
	var afterLogoDelete apiEnvelope[organizationProfileDTO]
	decodeJSON(t, request(t, client, http.MethodGet, adminURL, ownerToken, nil, http.StatusOK), &afterLogoDelete)
	if afterLogoDelete.Data.LogoAssetID != "" || afterLogoDelete.Data.LogoURL != "" {
		t.Fatalf("organization logo survived asset deletion: %+v", afterLogoDelete.Data)
	}
	payload["logo_asset_id"] = ""

	payload["is_public"] = false
	requireStatus(t, client, http.MethodPatch, adminURL, ownerToken, payload, http.StatusOK)
	for _, suffix := range []string{"", "/posts", "/server-status"} {
		requireStatus(t, client, http.MethodGet, portalURL+suffix, "", nil, http.StatusNotFound)
	}
	requireStatus(t, client, http.MethodPost, portalURL+"/apply", "", map[string]any{
		"type": "membership", "class_name": "integration", "name": "S4 Hidden",
		"game_id": "S4Hidden", "qq_number": "123456789", "email": "s4-hidden@integration.invalid",
	}, http.StatusNotFound)

	var auditCount int64
	if err := db.Model(&model.AuditEvent{}).Where("target_id = ? AND action = ? AND created_at >= ?", original.ID, "organization.profile_update", startedAt).Count(&auditCount).Error; err != nil {
		t.Fatalf("count organization profile audits: %v", err)
	}
	if auditCount != 2 {
		t.Fatalf("organization profile audit events = %d, want 2", auditCount)
	}
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
		},
		Fallback: "md3",
	}
}

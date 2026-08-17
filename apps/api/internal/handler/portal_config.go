package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/portalmanifest"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PortalConfigHandler struct {
	db *gorm.DB
}

type portalConfigRequest struct {
	Manifest portalmanifest.Manifest `json:"manifest" binding:"required"`
}

type portalConfigResponse struct {
	ID             string                   `json:"id,omitempty"`
	DraftManifest  *portalmanifest.Manifest `json:"draft_manifest"`
	ActiveManifest *portalmanifest.Manifest `json:"active_manifest"`
	Active         bool                     `json:"active"`
	UpdatedBy      string                   `json:"updated_by,omitempty"`
	UpdatedAt      *time.Time               `json:"updated_at,omitempty"`
	ActivatedBy    *string                  `json:"activated_by,omitempty"`
	ActivatedAt    *time.Time               `json:"activated_at,omitempty"`
}

type portalRuntimeResponse struct {
	Manifest    portalmanifest.Manifest `json:"manifest"`
	Source      string                  `json:"source"`
	ActivatedAt *time.Time              `json:"activated_at,omitempty"`
}

func NewPortalConfigHandler(db *gorm.DB) *PortalConfigHandler {
	return &PortalConfigHandler{db: db}
}

func (h *PortalConfigHandler) Get(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.unauthorized", "请先登录。")
		return
	}
	var configuration model.PortalConfiguration
	err := h.db.Where("organization_id = ?", principal.OrganizationID).First(&configuration).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respond(c, http.StatusOK, portalConfigResponse{})
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "portal_config.read_failed", "门户配置暂时无法读取。")
		return
	}
	response, err := portalConfigurationDTO(configuration)
	if err != nil {
		fail(c, http.StatusInternalServerError, "portal_config.corrupted", "已保存的门户配置无法解析，请联系管理员。")
		return
	}
	respond(c, http.StatusOK, response)
}

func (h *PortalConfigHandler) Public(c *gin.Context) {
	var organization model.Organization
	if err := h.db.Where("slug = ? AND is_public = ?", c.Param("slug"), true).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	response := portalRuntimeResponse{Manifest: defaultPortalManifest(), Source: "default"}
	c.Header("Cache-Control", "no-store")
	var configuration model.PortalConfiguration
	err := h.db.Where("organization_id = ?", organization.ID).First(&configuration).Error
	if err == nil && configuration.ActiveManifestJSON != "" {
		manifest, violations := portalmanifest.Parse([]byte(configuration.ActiveManifestJSON))
		if len(violations) == 0 {
			response.Manifest = manifest
			response.Source = "active"
			response.ActivatedAt = configuration.ActivatedAt
		} else {
			slog.Warn("portal runtime fallback",
				"event", "portal_runtime_fallback",
				"organization_id", organization.ID,
				"configuration_id", configuration.ID,
				"reason", "active_manifest_invalid",
			)
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusInternalServerError, "portal.configuration_failed", "门户配置暂时无法加载。")
		return
	}
	respond(c, http.StatusOK, response)
}

func (h *PortalConfigHandler) SaveDraft(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.unauthorized", "请先登录。")
		return
	}
	var body portalConfigRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "portal_config.invalid_request", "请求体必须包含 manifest。")
		return
	}
	if violations := portalmanifest.Validate(body.Manifest); len(violations) > 0 {
		failWithDetails(c, http.StatusBadRequest, "portal_config.manifest_invalid", "Manifest 校验失败。", violations)
		return
	}
	manifestJSON, err := json.Marshal(body.Manifest)
	if err != nil {
		fail(c, http.StatusInternalServerError, "portal_config.serialize_failed", "Manifest 暂时无法保存。")
		return
	}
	requestID := ensureRequestID(c)
	var configuration model.PortalConfiguration
	err = h.db.Transaction(func(tx *gorm.DB) error {
		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ?", principal.OrganizationID).First(&configuration).Error
		switch {
		case errors.Is(findErr, gorm.ErrRecordNotFound):
			configuration = model.PortalConfiguration{
				ID:                 uuid.NewString(),
				OrganizationID:     principal.OrganizationID,
				DraftManifestJSON:  string(manifestJSON),
				ActiveManifestJSON: "",
				UpdatedBy:          principal.UserID,
			}
			if err := tx.Create(&configuration).Error; err != nil {
				return err
			}
		case findErr != nil:
			return findErr
		default:
			if err := tx.Model(&configuration).Updates(map[string]interface{}{
				"draft_manifest_json": string(manifestJSON),
				"updated_by":          principal.UserID,
			}).Error; err != nil {
				return err
			}
			if err := tx.First(&configuration, "id = ?", configuration.ID).Error; err != nil {
				return err
			}
		}
		return tx.Create(&model.AuditEvent{
			ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID,
			Action: "portal.config_update", TargetType: "portal_configuration", TargetID: configuration.ID,
			Result: "success", RequestID: requestID,
		}).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "portal_config.save_failed", "门户草稿暂时无法保存。")
		return
	}
	response, err := portalConfigurationDTO(configuration)
	if err != nil {
		fail(c, http.StatusInternalServerError, "portal_config.serialize_failed", "门户配置暂时无法返回。")
		return
	}
	respond(c, http.StatusOK, response)
}

func (h *PortalConfigHandler) Enable(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.unauthorized", "请先登录。")
		return
	}
	requestID := ensureRequestID(c)
	now := time.Now().UTC()
	var configuration model.PortalConfiguration
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ?", principal.OrganizationID).First(&configuration).Error; err != nil {
			return err
		}
		_, violations := portalmanifest.Parse([]byte(configuration.DraftManifestJSON))
		if len(violations) > 0 {
			return errStoredManifestInvalid
		}
		if err := tx.Model(&configuration).Updates(map[string]interface{}{
			"active_manifest_json": configuration.DraftManifestJSON,
			"activated_by":         principal.UserID,
			"activated_at":         now,
		}).Error; err != nil {
			return err
		}
		if err := tx.First(&configuration, "id = ?", configuration.ID).Error; err != nil {
			return err
		}
		return tx.Create(&model.AuditEvent{
			ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID,
			Action: "portal.config_enable", TargetType: "portal_configuration", TargetID: configuration.ID,
			Result: "success", RequestID: requestID,
		}).Error
	})
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		fail(c, http.StatusConflict, "portal_config.draft_missing", "请先保存有效的门户草稿。")
		return
	case errors.Is(err, errStoredManifestInvalid):
		fail(c, http.StatusConflict, "portal_config.draft_invalid", "已保存的门户草稿不再符合当前 Manifest 规范，请重新保存。")
		return
	case err != nil:
		fail(c, http.StatusInternalServerError, "portal_config.enable_failed", "门户配置暂时无法启用。")
		return
	}
	response, err := portalConfigurationDTO(configuration)
	if err != nil {
		fail(c, http.StatusInternalServerError, "portal_config.serialize_failed", "门户配置暂时无法返回。")
		return
	}
	respond(c, http.StatusOK, response)
}

func (h *PortalConfigHandler) RestoreDefault(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.unauthorized", "请先登录。")
		return
	}
	manifestJSON, err := json.Marshal(defaultPortalManifest())
	if err != nil {
		fail(c, http.StatusInternalServerError, "portal_config.serialize_failed", "默认门户配置暂时无法生成。")
		return
	}
	requestID := ensureRequestID(c)
	now := time.Now().UTC()
	actorID := principal.UserID
	var configuration model.PortalConfiguration
	err = h.db.Transaction(func(tx *gorm.DB) error {
		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("organization_id = ?", principal.OrganizationID).First(&configuration).Error
		switch {
		case errors.Is(findErr, gorm.ErrRecordNotFound):
			configuration = model.PortalConfiguration{
				ID: uuid.NewString(), OrganizationID: principal.OrganizationID,
				DraftManifestJSON: string(manifestJSON), ActiveManifestJSON: string(manifestJSON),
				UpdatedBy: principal.UserID, ActivatedBy: &actorID, ActivatedAt: &now,
			}
			if err := tx.Create(&configuration).Error; err != nil {
				return err
			}
		case findErr != nil:
			return findErr
		default:
			if err := tx.Model(&configuration).Updates(map[string]interface{}{
				"draft_manifest_json":  string(manifestJSON),
				"active_manifest_json": string(manifestJSON),
				"updated_by":           principal.UserID,
				"activated_by":         principal.UserID,
				"activated_at":         now,
			}).Error; err != nil {
				return err
			}
			if err := tx.First(&configuration, "id = ?", configuration.ID).Error; err != nil {
				return err
			}
		}
		return tx.Create(&model.AuditEvent{
			ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID,
			Action: "portal.config_restore_default", TargetType: "portal_configuration", TargetID: configuration.ID,
			Result: "success", RequestID: requestID,
		}).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "portal_config.restore_failed", "默认 MD3 门户暂时无法恢复。")
		return
	}
	response, err := portalConfigurationDTO(configuration)
	if err != nil {
		fail(c, http.StatusInternalServerError, "portal_config.serialize_failed", "门户配置暂时无法返回。")
		return
	}
	respond(c, http.StatusOK, response)
}

var errStoredManifestInvalid = errors.New("stored portal manifest is invalid")

func defaultPortalManifest() portalmanifest.Manifest {
	return portalmanifest.Manifest{
		Schema:       portalmanifest.SchemaV1,
		ID:           "qutc-md3",
		Version:      "0.1.0",
		DisplayName:  "QUTCraft MD3 Portal",
		Entry:        "/index.html",
		Theme:        portalmanifest.ThemeRef{Mode: "md3"},
		Capabilities: []string{"organization.read", "public_content.read", "projects.read", "assets.read", "knowledge.read"},
		Fallback:     "md3",
	}
}

func portalConfigurationDTO(configuration model.PortalConfiguration) (portalConfigResponse, error) {
	response := portalConfigResponse{
		ID: configuration.ID, Active: configuration.ActiveManifestJSON != "", UpdatedBy: configuration.UpdatedBy,
		UpdatedAt: &configuration.UpdatedAt, ActivatedBy: configuration.ActivatedBy, ActivatedAt: configuration.ActivatedAt,
	}
	if configuration.DraftManifestJSON != "" {
		var manifest portalmanifest.Manifest
		if err := json.Unmarshal([]byte(configuration.DraftManifestJSON), &manifest); err != nil {
			return portalConfigResponse{}, err
		}
		response.DraftManifest = &manifest
	}
	if configuration.ActiveManifestJSON != "" {
		var manifest portalmanifest.Manifest
		if err := json.Unmarshal([]byte(configuration.ActiveManifestJSON), &manifest); err != nil {
			return portalConfigResponse{}, err
		}
		response.ActiveManifest = &manifest
	}
	return response, nil
}

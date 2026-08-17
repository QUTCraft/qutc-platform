package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type organizationSocialLink struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type organizationProfileRequest struct {
	Name         string                   `json:"name"`
	ShortName    string                   `json:"short_name"`
	Tagline      string                   `json:"tagline"`
	Introduction string                   `json:"introduction"`
	ContactEmail string                   `json:"contact_email"`
	FilingNumber string                   `json:"filing_number"`
	LogoAssetID  string                   `json:"logo_asset_id"`
	SocialLinks  []organizationSocialLink `json:"social_links"`
	IsPublic     bool                     `json:"is_public"`
}

var (
	errOrganizationLogoNotFound = errors.New("organization logo asset not found")
	errOrganizationLogoInvalid  = errors.New("organization logo asset is not an image")
)

func normalizeOrganizationProfile(body organizationProfileRequest) (organizationProfileRequest, bool) {
	body.Name = strings.TrimSpace(body.Name)
	body.ShortName = strings.TrimSpace(body.ShortName)
	body.Tagline = strings.TrimSpace(body.Tagline)
	body.Introduction = strings.TrimSpace(body.Introduction)
	body.ContactEmail = strings.ToLower(strings.TrimSpace(body.ContactEmail))
	body.FilingNumber = strings.TrimSpace(body.FilingNumber)
	body.LogoAssetID = strings.TrimSpace(body.LogoAssetID)
	if body.Name == "" || len([]rune(body.Name)) > 160 || body.ShortName == "" || len([]rune(body.ShortName)) > 40 || len([]rune(body.Tagline)) > 160 || len([]rune(body.Introduction)) > 2000 {
		return body, false
	}
	if len([]rune(body.FilingNumber)) > 80 {
		return body, false
	}
	if body.LogoAssetID != "" {
		if _, err := uuid.Parse(body.LogoAssetID); err != nil {
			return body, false
		}
	}
	if body.ContactEmail != "" {
		parsed, err := mail.ParseAddress(body.ContactEmail)
		if err != nil || parsed.Address != body.ContactEmail {
			return body, false
		}
	}
	if len(body.SocialLinks) > 12 {
		return body, false
	}
	for index := range body.SocialLinks {
		body.SocialLinks[index].Label = strings.TrimSpace(body.SocialLinks[index].Label)
		body.SocialLinks[index].Href = strings.TrimSpace(body.SocialLinks[index].Href)
		parsed, err := url.ParseRequestURI(body.SocialLinks[index].Href)
		if body.SocialLinks[index].Label == "" || len([]rune(body.SocialLinks[index].Label)) > 40 || err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return body, false
		}
	}
	return body, true
}

func organizationProfileItem(organization model.Organization) gin.H {
	links := make([]organizationSocialLink, 0)
	if strings.TrimSpace(organization.SocialLinksJSON) != "" {
		_ = json.Unmarshal([]byte(organization.SocialLinksJSON), &links)
	}
	logoURL := ""
	if organization.LogoAssetID != "" {
		logoURL = "/api/v1/portal/organizations/" + url.PathEscape(organization.Slug) + "/assets/" + url.PathEscape(organization.LogoAssetID) + "/download"
	}
	return gin.H{
		"id": organization.ID, "slug": organization.Slug, "name": organization.Name,
		"short_name": organization.ShortName, "tagline": organization.Tagline,
		"introduction": organization.Introduction, "contact_email": organization.ContactEmail,
		"filing_number": organization.FilingNumber,
		"logo_asset_id": organization.LogoAssetID, "logo_url": logoURL,
		"social_links": links, "is_public": organization.IsPublic, "updated_at": organization.UpdatedAt,
	}
}

func (h *WorkspaceHandler) AdminOrganization(c *gin.Context) {
	principal, _ := middleware.PrincipalFromContext(c)
	var organization model.Organization
	if err := h.db.Where("id = ?", principal.OrganizationID).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "organization.not_found", "组织不存在。")
		return
	}
	respond(c, http.StatusOK, organizationProfileItem(organization))
}

func (h *WorkspaceHandler) AdminUpdateOrganization(c *gin.Context) {
	principal, _ := middleware.PrincipalFromContext(c)
	var body organizationProfileRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "organization.validation_failed", "组织资料格式不正确。")
		return
	}
	normalized, valid := normalizeOrganizationProfile(body)
	if !valid {
		fail(c, http.StatusBadRequest, "organization.validation_failed", "组织资料字段不符合规范。")
		return
	}
	linksJSON, err := json.Marshal(normalized.SocialLinks)
	if err != nil {
		fail(c, http.StatusBadRequest, "organization.validation_failed", "社交链接格式不正确。")
		return
	}
	var organization model.Organization
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", principal.OrganizationID).First(&organization).Error; err != nil {
			return err
		}
		if normalized.LogoAssetID != "" {
			var logo model.MediaAsset
			if err := tx.Select("id", "mime_type").Where("id = ? AND organization_id = ?", normalized.LogoAssetID, principal.OrganizationID).First(&logo).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errOrganizationLogoNotFound
				}
				return err
			}
			if !isImageAsset(logo.MimeType) {
				return errOrganizationLogoInvalid
			}
		}
		organization.Name, organization.ShortName = normalized.Name, normalized.ShortName
		organization.Tagline, organization.Introduction = normalized.Tagline, normalized.Introduction
		organization.ContactEmail, organization.FilingNumber = normalized.ContactEmail, normalized.FilingNumber
		organization.LogoAssetID, organization.SocialLinksJSON, organization.IsPublic = normalized.LogoAssetID, string(linksJSON), normalized.IsPublic
		if err := tx.Save(&organization).Error; err != nil {
			return err
		}
		return tx.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "organization.profile_update", TargetType: "organization", TargetID: organization.ID, Result: "success", RequestID: ensureRequestID(c)}).Error
	})
	if err != nil {
		if errors.Is(err, errOrganizationLogoNotFound) {
			fail(c, http.StatusNotFound, "asset.not_found", "Logo 图片不存在或不属于当前组织。")
		} else if errors.Is(err, errOrganizationLogoInvalid) {
			fail(c, http.StatusBadRequest, "organization.logo_invalid", "门户 Logo 必须使用 PNG、JPEG 或 WebP 图片。")
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "organization.not_found", "组织不存在。")
		} else {
			fail(c, http.StatusInternalServerError, "organization.update_failed", "组织资料保存失败。")
		}
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, organizationProfileItem(organization))
}

package handler

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/storage"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	maxAssetSize        = 10 << 20
	maxAssetRequestSize = maxAssetSize + (1 << 20)
)

var errAssetAlreadyLinked = errors.New("asset is already linked")

type publishAssetResourceRequest struct {
	Title       string `json:"title"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

func (h *WorkspaceHandler) UploadAsset(c *gin.Context) {
	principal, _ := middleware.PrincipalFromContext(c)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAssetRequestSize)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "request body too large") {
			fail(c, http.StatusRequestEntityTooLarge, "asset.request_too_large", "上传请求不能超过 11 MB。")
			return
		}
		fail(c, http.StatusBadRequest, "asset.file_required", "请上传名为 file 的文件。")
		return
	}
	defer file.Close()
	if header.Size > maxAssetSize {
		fail(c, http.StatusBadRequest, "asset.file_too_large", "开发环境单文件不能超过 10 MB。")
		return
	}
	originalName := strings.TrimSpace(filepath.Base(header.Filename))
	if originalName == "" || originalName == "." || len([]rune(originalName)) > 255 {
		fail(c, http.StatusBadRequest, "asset.filename_invalid", "文件名不能为空且不能超过 255 个字符。")
		return
	}
	contentID := strings.TrimSpace(c.PostForm("content_id"))
	if contentID != "" {
		var content model.Content
		if err := h.db.Where("id = ? AND organization_id = ?", contentID, principal.OrganizationID).First(&content).Error; err != nil {
			fail(c, http.StatusNotFound, "content.not_found", "引用的内容不存在。")
			return
		}
	}
	mimeType, sniffErr := detectAssetType(file)
	if sniffErr != nil {
		fail(c, http.StatusBadRequest, "asset.file_invalid", "无法识别上传文件类型。")
		return
	}
	if !allowedAssetType(mimeType) {
		fail(c, http.StatusBadRequest, "asset.type_not_allowed", "该文件类型不在允许列表中。")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		fail(c, http.StatusBadRequest, "asset.file_invalid", "无法读取上传文件。")
		return
	}
	id := uuid.NewString()
	mediaStorage, storageErr := h.storageFor(c.Request.Context(), principal.OrganizationID, "")
	if storageErr != nil {
		fail(c, http.StatusServiceUnavailable, "asset.storage_unavailable", "媒体存储暂不可用。")
		return
	}
	storedName := id + filepath.Ext(originalName)
	storageKey := principal.OrganizationID + "/" + storedName
	written, storeErr := mediaStorage.Put(c.Request.Context(), storageKey, io.LimitReader(file, maxAssetSize+1), mimeType)
	if storeErr != nil || written > maxAssetSize {
		_ = mediaStorage.Delete(c.Request.Context(), storageKey)
		if storeErr != nil {
			fail(c, http.StatusServiceUnavailable, "asset.storage_unavailable", "媒体存储暂不可用。")
			return
		}
		fail(c, http.StatusBadRequest, "asset.file_invalid", "媒体文件保存失败或超过大小限制。")
		return
	}
	asset := model.MediaAsset{ID: id, OrganizationID: principal.OrganizationID, ContentID: contentID, UploadedBy: principal.UserID, OriginalName: originalName, StoredName: storedName, MimeType: mimeType, SizeBytes: written, StorageDriver: mediaStorage.Driver(), StoragePath: storageKey}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&asset).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "asset.upload", "asset", asset.ID)
	}); err != nil {
		_ = mediaStorage.Delete(c.Request.Context(), storageKey)
		fail(c, http.StatusInternalServerError, "asset.metadata_failed", "媒体元数据保存失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusCreated, assetResponse(asset))
}

// AdminAssets lists the current organization's media assets without exposing
// storage keys or S3/MinIO credentials. The browser can use the returned
// admin download URL, but it never talks to object storage directly.
func (h *WorkspaceHandler) AdminAssets(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	page, pageSize, ok := listMeta(c, 0)
	if !ok {
		return
	}
	search, ok := queryMax(c, "query", 120)
	if !ok {
		return
	}

	query := h.db.Model(&model.MediaAsset{}).Where("organization_id = ?", principal.OrganizationID)
	if search != "" {
		query = query.Where("original_name LIKE ?", "%"+search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "asset.list_failed", "媒体资源列表暂时无法加载。")
		return
	}
	var assets []model.MediaAsset
	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&assets).Error; err != nil {
		fail(c, http.StatusInternalServerError, "asset.list_failed", "媒体资源列表暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(assets))
	for _, asset := range assets {
		items = append(items, assetResponse(asset))
	}
	respondWithMeta(c, http.StatusOK, items, gin.H{"page": page, "page_size": pageSize, "total": total})
}

// PublishAssetAsResource turns an intentionally selected, unlinked asset into
// a public CMS resource. Uploads stay private by default; this explicit action
// atomically creates the published content record and binds the file so a
// failed request can never leave a half-published portal entry behind.
func (h *WorkspaceHandler) PublishAssetAsResource(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}

	var asset model.MediaAsset
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "asset.not_found", "媒体资源不存在。")
			return
		}
		fail(c, http.StatusInternalServerError, "asset.load_failed", "媒体资源暂时无法读取。")
		return
	}
	if asset.ContentID != "" {
		fail(c, http.StatusConflict, "asset.already_linked", "该文件已经关联内容，不能重复归档。")
		return
	}

	var body publishAssetResourceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "asset.resource_validation_failed", "公开资源信息格式不正确。")
		return
	}
	input, err := normalizeAssetResourceInput(asset, body)
	if err != nil {
		fail(c, http.StatusBadRequest, "asset.resource_validation_failed", "资源标题、类型或说明不符合规范。")
		return
	}

	now := time.Now().UTC()
	content := model.Content{
		ID:             uuid.NewString(),
		OrganizationID: principal.OrganizationID,
		AuthorUserID:   principal.UserID,
		Title:          input.Title,
		Type:           service.ContentTypeResource,
		Category:       input.Category,
		Status:         service.ContentStatusDraft,
		Excerpt:        input.Excerpt,
		Body:           input.Body,
	}
	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&content).Error; err != nil {
			return err
		}
		if err := createContentRevision(tx, content, principal.UserID, "create"); err != nil {
			return err
		}

		content.Status = service.ContentStatusPublished
		content.PublishedAt = &now
		if err := tx.Save(&content).Error; err != nil {
			return err
		}
		if err := createContentRevision(tx, content, principal.UserID, service.ContentStatusPublished); err != nil {
			return err
		}

		result := tx.Model(&model.MediaAsset{}).
			Where("id = ? AND organization_id = ? AND content_id = ''", asset.ID, principal.OrganizationID).
			Update("content_id", content.ID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errAssetAlreadyLinked
		}
		if err := writeAudit(tx, c, principal.OrganizationID, principal.UserID, "content.create_from_asset", "content", content.ID); err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "content.published", "content", content.ID)
	})
	if errors.Is(err, errAssetAlreadyLinked) {
		fail(c, http.StatusConflict, "asset.already_linked", "该文件已经被其他内容关联，请刷新后重试。")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, "asset.resource_publish_failed", "资源归档发布失败。")
		return
	}

	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusCreated, contentAdminItem(content, h.db))
}

func normalizeAssetResourceInput(asset model.MediaAsset, body publishAssetResourceRequest) (service.ContentInput, error) {
	kind := strings.TrimSpace(body.Kind)
	if kind == "" {
		kind = inferResourceKind(asset.MimeType)
	}
	if kind != "document" && kind != "template" && kind != "package" && kind != "video" {
		return service.ContentInput{}, fmt.Errorf("invalid resource kind")
	}
	description := strings.TrimSpace(body.Description)
	if description == "" {
		description = "公开文件：" + asset.OriginalName
	}
	return service.NormalizeContentInput(service.ContentInput{
		Title:    body.Title,
		Type:     service.ContentTypeResource,
		Category: kind,
		Excerpt:  description,
		Body:     description,
	})
}

func inferResourceKind(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case mimeType == "application/zip" || mimeType == "application/x-zip-compressed":
		return "package"
	default:
		return "document"
	}
}

func (h *WorkspaceHandler) DownloadAsset(c *gin.Context) {
	var asset model.MediaAsset
	if err := h.db.Where("id = ?", c.Param("id")).First(&asset).Error; err != nil {
		fail(c, http.StatusNotFound, "asset.not_found", "媒体资源不存在。")
		return
	}
	if slug := c.Param("slug"); slug != "" {
		var organization model.Organization
		var content model.Content
		if h.db.Where("slug = ? AND id = ? AND is_public = ?", slug, asset.OrganizationID, true).First(&organization).Error != nil || asset.ContentID == "" || h.db.Where("id = ? AND organization_id = ? AND status = ?", asset.ContentID, asset.OrganizationID, "published").First(&content).Error != nil {
			fail(c, http.StatusNotFound, "asset.not_public", "媒体资源尚未公开。")
			return
		}
	} else {
		principal, ok := middleware.PrincipalFromContext(c)
		if !ok || principal.OrganizationID != asset.OrganizationID {
			fail(c, http.StatusNotFound, "asset.not_found", "媒体资源不存在。")
			return
		}
	}
	assetStorageDriver := strings.TrimSpace(asset.StorageDriver)
	if assetStorageDriver == "" {
		assetStorageDriver = "local"
	}
	mediaStorage, storageErr := h.storageFor(c.Request.Context(), asset.OrganizationID, assetStorageDriver)
	if storageErr != nil {
		fail(c, http.StatusServiceUnavailable, "asset.storage_driver_unavailable", "该媒体资源所属的存储后端当前不可用，请检查服务接入配置。")
		return
	}
	source, err := mediaStorage.Open(c.Request.Context(), asset.StoragePath)
	if errors.Is(err, storage.ErrNotFound) {
		fail(c, http.StatusNotFound, "asset.file_missing", "媒体文件不存在。")
		return
	}
	if err != nil {
		fail(c, http.StatusServiceUnavailable, "asset.storage_unavailable", "媒体存储暂不可用。")
		return
	}
	defer source.Close()
	now := time.Now().UTC()
	_ = h.db.Model(&model.MediaAsset{}).Where("id = ? AND organization_id = ?", asset.ID, asset.OrganizationID).Updates(map[string]any{"download_count": gorm.Expr("download_count + 1"), "last_downloaded_at": now}).Error
	c.Header("X-Content-Type-Options", "nosniff")
	disposition := "attachment"
	if c.Param("slug") != "" && strings.HasPrefix(asset.MimeType, "image/") {
		disposition = "inline"
	}
	contentDisposition := mime.FormatMediaType(disposition, map[string]string{"filename": asset.OriginalName})
	c.DataFromReader(http.StatusOK, asset.SizeBytes, asset.MimeType, source, map[string]string{
		"Content-Disposition": contentDisposition,
	})
}

func assetResponse(asset model.MediaAsset) gin.H {
	var contentID interface{}
	if asset.ContentID != "" {
		contentID = asset.ContentID
	}
	return gin.H{"id": asset.ID, "content_id": contentID, "original_name": asset.OriginalName, "mime_type": asset.MimeType, "size_bytes": asset.SizeBytes, "download_count": asset.DownloadCount, "last_downloaded_at": asset.LastDownloadedAt, "created_at": asset.CreatedAt, "download_url": "/api/v1/admin/assets/" + asset.ID + "/download"}
}

func (h *WorkspaceHandler) AssetDownloadStats(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var asset model.MediaAsset
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "asset.not_found", "媒体资源不存在。")
			return
		}
		fail(c, http.StatusInternalServerError, "asset.stats_failed", "媒体下载统计暂时无法读取。")
		return
	}
	respond(c, http.StatusOK, gin.H{"id": asset.ID, "content_id": asset.ContentID, "download_count": asset.DownloadCount, "last_downloaded_at": asset.LastDownloadedAt})
}

// DeleteAsset removes the stored object and its metadata. A linked asset must
// be taken off the public portal first. Resource content exists to represent
// its files, so its record is also removed when the last linked file is
// deleted; news and knowledge content keep their editorial record.
func (h *WorkspaceHandler) DeleteAsset(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var asset model.MediaAsset
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "asset.not_found", "媒体资源不存在。")
			return
		}
		fail(c, http.StatusInternalServerError, "asset.delete_failed", "媒体资源暂时无法删除。")
		return
	}
	var linkedContent model.Content
	linkedContentFound := false
	if asset.ContentID != "" {
		err := h.db.Select("id", "type", "status").Where("id = ? AND organization_id = ?", asset.ContentID, principal.OrganizationID).First(&linkedContent).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusInternalServerError, "asset.content_status_failed", "关联内容状态暂时无法确认。")
			return
		}
		if err == nil {
			linkedContentFound = true
			if linkedContent.Status == service.ContentStatusPublished {
				fail(c, http.StatusConflict, "asset.still_public", "该文件仍在门户公开，请先下架后再删除。")
				return
			}
		}
	}
	assetStorageDriver := strings.TrimSpace(asset.StorageDriver)
	if assetStorageDriver == "" {
		assetStorageDriver = "local"
	}
	mediaStorage, storageErr := h.storageFor(c.Request.Context(), asset.OrganizationID, assetStorageDriver)
	if storageErr != nil {
		fail(c, http.StatusServiceUnavailable, "asset.storage_driver_unavailable", "该媒体资源所属的存储后端当前不可用，请检查服务接入配置。")
		return
	}
	if err := mediaStorage.Delete(c.Request.Context(), asset.StoragePath); err != nil && !errors.Is(err, storage.ErrNotFound) {
		fail(c, http.StatusServiceUnavailable, "asset.storage_unavailable", "媒体文件暂时无法删除。")
		return
	}
	removedContentID := ""
	detachedContentID := asset.ContentID
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND organization_id = ?", asset.ID, principal.OrganizationID).Delete(&model.MediaAsset{}).Error; err != nil {
			return err
		}
		if linkedContentFound && linkedContent.Type == service.ContentTypeResource {
			var remainingAssets int64
			if err := tx.Model(&model.MediaAsset{}).
				Where("content_id = ? AND organization_id = ?", linkedContent.ID, principal.OrganizationID).
				Count(&remainingAssets).Error; err != nil {
				return err
			}
			if remainingAssets == 0 {
				if err := tx.Where("content_id = ? AND organization_id = ?", linkedContent.ID, principal.OrganizationID).Delete(&model.ContentRevision{}).Error; err != nil {
					return err
				}
				result := tx.Where("id = ? AND organization_id = ?", linkedContent.ID, principal.OrganizationID).Delete(&model.Content{})
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected != 1 {
					return gorm.ErrRecordNotFound
				}
				if err := writeAudit(tx, c, principal.OrganizationID, principal.UserID, "content.delete_with_asset", "content", linkedContent.ID); err != nil {
					return err
				}
				removedContentID = linkedContent.ID
				detachedContentID = ""
			}
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "asset.delete", "asset", asset.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "asset.metadata_delete_failed", "媒体资源元数据暂时无法删除。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	var detachedContentValue interface{}
	if detachedContentID != "" {
		detachedContentValue = detachedContentID
	}
	var removedContentValue interface{}
	if removedContentID != "" {
		removedContentValue = removedContentID
	}
	respond(c, http.StatusOK, gin.H{
		"removed":             true,
		"id":                  asset.ID,
		"detached_content_id": detachedContentValue,
		"removed_content_id":  removedContentValue,
	})
}

func detectAssetType(reader io.Reader) (string, error) {
	buffer := make([]byte, 512)
	read, err := reader.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}
	if read == 0 {
		return "", fmt.Errorf("empty asset")
	}
	return http.DetectContentType(buffer[:read]), nil
}

func allowedAssetType(mime string) bool {
	for _, allowed := range []string{"image/png", "image/jpeg", "image/webp", "application/pdf", "application/zip", "video/mp4"} {
		if mime == allowed {
			return true
		}
	}
	return false
}

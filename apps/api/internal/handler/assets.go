package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	maxAssetSize        = 10 << 20
	maxAssetRequestSize = maxAssetSize + (1 << 20)
)

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
	root := "/tmp/qutcraft-uploads"
	if err := os.MkdirAll(root, 0o750); err != nil {
		fail(c, http.StatusInternalServerError, "asset.storage_unavailable", "媒体存储暂不可用。")
		return
	}
	storedName := id + filepath.Ext(originalName)
	storagePath := filepath.Join(root, storedName)
	target, err := os.OpenFile(storagePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		fail(c, http.StatusInternalServerError, "asset.storage_failed", "媒体文件保存失败。")
		return
	}
	written, copyErr := io.Copy(target, io.LimitReader(file, maxAssetSize+1))
	closeErr := target.Close()
	if copyErr != nil || closeErr != nil || written > maxAssetSize {
		_ = os.Remove(storagePath)
		fail(c, http.StatusBadRequest, "asset.file_invalid", "媒体文件保存失败或超过大小限制。")
		return
	}
	asset := model.MediaAsset{ID: id, OrganizationID: principal.OrganizationID, ContentID: contentID, UploadedBy: principal.UserID, OriginalName: originalName, StoredName: storedName, MimeType: mimeType, SizeBytes: written, StoragePath: storagePath}
	if err := h.db.Create(&asset).Error; err != nil {
		_ = os.Remove(storagePath)
		fail(c, http.StatusInternalServerError, "asset.metadata_failed", "媒体元数据保存失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusCreated, assetResponse(asset))
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
		if h.db.Where("slug = ? AND id = ?", slug, asset.OrganizationID).First(&organization).Error != nil || asset.ContentID == "" || h.db.Where("id = ? AND organization_id = ? AND status = ?", asset.ContentID, asset.OrganizationID, "published").First(&content).Error != nil {
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
	if _, err := os.Stat(asset.StoragePath); err != nil {
		fail(c, http.StatusNotFound, "asset.file_missing", "媒体文件不存在。")
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	if c.Param("slug") != "" && strings.HasPrefix(asset.MimeType, "image/") {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", asset.OriginalName))
		c.Header("Content-Type", asset.MimeType)
		c.File(asset.StoragePath)
		return
	}
	c.FileAttachment(asset.StoragePath, asset.OriginalName)
}

func assetResponse(asset model.MediaAsset) gin.H {
	var contentID interface{}
	if asset.ContentID != "" {
		contentID = asset.ContentID
	}
	return gin.H{"id": asset.ID, "content_id": contentID, "original_name": asset.OriginalName, "mime_type": asset.MimeType, "size_bytes": asset.SizeBytes, "download_url": "/api/v1/admin/assets/" + asset.ID + "/download"}
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

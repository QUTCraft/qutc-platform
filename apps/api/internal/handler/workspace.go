package handler

import (
	"context"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/cache"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkspaceHandler owns the first content read/write model shared by the
// public portal and the protected CMS workspace.
type WorkspaceHandler struct {
	db             *gorm.DB
	cache          *cache.Cache
	cacheNamespace string
}

func NewWorkspaceHandler(db *gorm.DB, publicCache *cache.Cache, environment string) *WorkspaceHandler {
	if strings.TrimSpace(environment) == "" {
		environment = "development"
	}
	return &WorkspaceHandler{db: db, cache: publicCache, cacheNamespace: environment}
}

func (h *WorkspaceHandler) cachedPortalPage(c *gin.Context, slug, resource string, loader func() ([]gin.H, error)) {
	key := "qutc:" + h.cacheNamespace + ":portal:" + slug + ":" + resource + ":" + cache.NormalizeQuery(c.Request.URL.RawQuery)
	var items []gin.H
	if h.cache != nil && h.cache.Get(context.Background(), key, &items) {
		pageOf(c, items)
		return
	}
	items, err := loader()
	if err != nil {
		fail(c, http.StatusInternalServerError, "portal."+resource+"_failed", "公开数据暂时无法加载。")
		return
	}
	if h.cache != nil {
		h.cache.Set(context.Background(), key, items)
	}
	pageOf(c, items)
}

func (h *WorkspaceHandler) cachedPortalItem(c *gin.Context, slug, resource string, loader func() (gin.H, error)) {
	key := "qutc:" + h.cacheNamespace + ":portal:" + slug + ":" + resource
	var item gin.H
	if h.cache != nil && h.cache.Get(context.Background(), key, &item) {
		respond(c, http.StatusOK, item)
		return
	}
	item, err := loader()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			fail(c, http.StatusNotFound, "portal.content_not_found", "公开内容不存在或尚未发布。")
			return
		}
		fail(c, http.StatusInternalServerError, "portal."+resource+"_failed", "公开数据暂时无法加载。")
		return
	}
	if h.cache != nil {
		h.cache.Set(context.Background(), key, item)
	}
	respond(c, http.StatusOK, item)
}

func (h *WorkspaceHandler) invalidatePortalCache(organizationID string) {
	if h.cache == nil {
		return
	}
	var organization model.Organization
	if h.db.First(&organization, "id = ?", organizationID).Error == nil {
		h.cache.DeletePrefix(context.Background(), "qutc:"+h.cacheNamespace+":portal:"+organization.Slug+":")
	}
}

func listMeta(c *gin.Context, total int) (int, int, bool) {
	page, pageSize := 1, 20
	var err error
	if raw := c.Query("page"); raw != "" {
		page, err = strconv.Atoi(raw)
	}
	if err != nil || page < 1 {
		fail(c, http.StatusBadRequest, "pagination.invalid_page", "page 必须是大于等于 1 的整数。")
		return 0, 0, false
	}
	if raw := c.Query("page_size"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
	}
	if err != nil || pageSize < 1 || pageSize > 100 {
		fail(c, http.StatusBadRequest, "pagination.invalid_page_size", "page_size 必须在 1 到 100 之间。")
		return 0, 0, false
	}
	return page, pageSize, true
}

func pageOf[T any](c *gin.Context, values []T) {
	page, pageSize, ok := listMeta(c, len(values))
	if !ok {
		return
	}
	start := (page - 1) * pageSize
	if start > len(values) {
		start = len(values)
	}
	end := start + pageSize
	if end > len(values) {
		end = len(values)
	}
	respondWithMeta(c, http.StatusOK, values[start:end], gin.H{"page": page, "page_size": pageSize, "total": len(values)})
}

func queryMax(c *gin.Context, key string, max int) (string, bool) {
	value := strings.TrimSpace(c.Query(key))
	if len([]rune(value)) > max {
		fail(c, http.StatusBadRequest, "query.value_too_long", key+" 超出长度限制。")
		return "", false
	}
	return value, true
}

func (h *WorkspaceHandler) Organization(c *gin.Context) {
	var org model.Organization
	if err := h.db.Where("slug = ?", c.Param("slug")).First(&org).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	respond(c, http.StatusOK, gin.H{"id": org.ID, "slug": org.Slug, "name": org.Name, "short_name": "QUTCraft", "tagline": "把社团正在发生的事，认真地呈现出来。", "introduction": "QUTCraft 是青岛理工大学的 Minecraft 社团，持续建设内容、项目与公共知识资产。", "contact_email": "contact@qutcraft.example", "social_links": []gin.H{{"label": "GitHub", "href": "https://github.com/QUTCraft/qutc-platform"}}})
}

func (h *WorkspaceHandler) PortalContentDetail(c *gin.Context) {
	var organization model.Organization
	if err := h.db.Where("slug = ?", c.Param("slug")).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	contentID := c.Param("id")
	h.cachedPortalItem(c, c.Param("slug"), "content:"+contentID, func() (gin.H, error) {
		var content model.Content
		if err := h.db.Where("id = ? AND organization_id = ? AND status = ?", contentID, organization.ID, "published").First(&content).Error; err != nil {
			return nil, gorm.ErrRecordNotFound
		}
		return h.contentPublicDetailItem(c.Param("slug"), content), nil
	})
}

func (h *WorkspaceHandler) PortalPosts(c *gin.Context) {
	category, ok := queryMax(c, "category", 64)
	if !ok {
		return
	}
	var organization model.Organization
	if err := h.db.Where("slug = ?", c.Param("slug")).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	query := h.db.Where("organization_id = ? AND type = ? AND status = ?", organization.ID, "news", "published").Order("published_at DESC")
	if category != "" {
		query = query.Where("title LIKE ? OR excerpt LIKE ?", "%"+category+"%", "%"+category+"%")
	}
	h.cachedPortalPage(c, c.Param("slug"), "posts", func() ([]gin.H, error) {
		var contents []model.Content
		if err := query.Find(&contents).Error; err != nil {
			return nil, err
		}
		items := make([]gin.H, 0, len(contents))
		for _, content := range contents {
			items = append(items, contentPublicItem(content))
		}
		return items, nil
	})
}
func (h *WorkspaceHandler) PortalProjects(c *gin.Context) {
	status, ok := queryMax(c, "status", 16)
	if !ok {
		return
	}
	if status != "" && status != "active" && status != "research" && status != "completed" {
		fail(c, http.StatusBadRequest, "query.invalid_status", "status 不是受支持的项目状态。")
		return
	}
	var organization model.Organization
	if err := h.db.Where("slug = ?", c.Param("slug")).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	h.cachedPortalPage(c, c.Param("slug"), "projects", func() ([]gin.H, error) {
		query := h.db.Where("organization_id = ? AND is_public = ?", organization.ID, true).Order("updated_at DESC")
		if status != "" {
			query = query.Where("status = ?", status)
		}
		var projects []model.Project
		if err := query.Find(&projects).Error; err != nil {
			return nil, err
		}
		items := make([]gin.H, 0, len(projects))
		for _, project := range projects {
			items = append(items, projectPublicItem(project))
		}
		return items, nil
	})
}
func (h *WorkspaceHandler) PortalResources(c *gin.Context) {
	kind, ok := queryMax(c, "kind", 16)
	if !ok {
		return
	}
	q, ok := queryMax(c, "q", 128)
	if !ok {
		return
	}
	if kind != "" && kind != "document" && kind != "template" && kind != "package" && kind != "video" {
		fail(c, http.StatusBadRequest, "query.invalid_kind", "kind 不是受支持的资源类型。")
		return
	}
	var organization model.Organization
	if err := h.db.Where("slug = ?", c.Param("slug")).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	query := h.db.Where("organization_id = ? AND type = ? AND status = ?", organization.ID, "resource", "published").Order("updated_at DESC")
	if q != "" {
		query = query.Where("title LIKE ? OR excerpt LIKE ? OR body LIKE ?", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	h.cachedPortalPage(c, c.Param("slug"), "resources", func() ([]gin.H, error) {
		var contents []model.Content
		if err := query.Find(&contents).Error; err != nil {
			return nil, err
		}
		items := make([]gin.H, 0, len(contents))
		for _, content := range contents {
			item := h.resourcePublicItem(c.Param("slug"), content)
			if kind == "" || kind == item["kind"] {
				items = append(items, item)
			}
		}
		return items, nil
	})
}
func (h *WorkspaceHandler) PortalKnowledge(c *gin.Context) {
	category, ok := queryMax(c, "category", 64)
	if !ok {
		return
	}
	q, ok := queryMax(c, "q", 128)
	if !ok {
		return
	}
	var organization model.Organization
	if err := h.db.Where("slug = ?", c.Param("slug")).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	query := h.db.Where("organization_id = ? AND type = ? AND status = ?", organization.ID, "knowledge", "published").Order("updated_at DESC")
	if category != "" {
		query = query.Where("category = ? OR title LIKE ? OR excerpt LIKE ?", category, "%"+category+"%", "%"+category+"%")
	}
	if q != "" {
		query = query.Where("title LIKE ? OR excerpt LIKE ? OR body LIKE ?", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	h.cachedPortalPage(c, c.Param("slug"), "knowledge", func() ([]gin.H, error) {
		var contents []model.Content
		if err := query.Find(&contents).Error; err != nil {
			return nil, err
		}
		items := make([]gin.H, 0, len(contents))
		for _, content := range contents {
			categoryName := content.Category
			if categoryName == "" {
				categoryName = "知识库"
			}
			items = append(items, gin.H{"id": content.ID, "title": content.Title, "summary": content.Excerpt, "category": categoryName, "updated_at": content.UpdatedAt, "reading_minutes": maxInt(1, len([]rune(content.Body))/900+1)})
		}
		return items, nil
	})
}

func (h *WorkspaceHandler) PortalKnowledgeDirectories(c *gin.Context) {
	var organization model.Organization
	if err := h.db.Where("slug = ?", c.Param("slug")).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	h.cachedPortalPage(c, c.Param("slug"), "knowledge-directories", func() ([]gin.H, error) {
		var directories []model.KnowledgeDirectory
		if err := h.db.Where("organization_id = ? AND is_public = ?", organization.ID, true).Order("sort_order ASC, name ASC").Find(&directories).Error; err != nil {
			return nil, err
		}
		items := make([]gin.H, 0, len(directories))
		for _, directory := range directories {
			var articleCount int64
			h.db.Model(&model.Content{}).Where("organization_id = ? AND type = ? AND status = ? AND category = ?", organization.ID, "knowledge", "published", directory.Name).Count(&articleCount)
			items = append(items, gin.H{"id": directory.ID, "name": directory.Name, "slug": directory.Slug, "description": directory.Description, "article_count": articleCount, "updated_at": directory.UpdatedAt})
		}
		return items, nil
	})
}
func (h *WorkspaceHandler) PortalServer(c *gin.Context) {
	respond(c, http.StatusOK, gin.H{"enabled": true, "label": "QUTCraft Java 生存服", "state": "online", "version": "Java 1.21.x", "online_players": 18, "max_players": 60, "updated_at": time.Now().UTC(), "apply_url": "#join"})
}

func (h *WorkspaceHandler) AdminDashboard(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var organization model.Organization
	if err := h.db.First(&organization, "id = ?", principal.OrganizationID).Error; err != nil {
		fail(c, http.StatusNotFound, "organization.not_found", "组织不存在。")
		return
	}
	var published, total, activeMembers int64
	h.db.Model(&model.Content{}).Where("organization_id = ? AND status = ?", principal.OrganizationID, "published").Count(&published)
	h.db.Model(&model.Content{}).Where("organization_id = ?", principal.OrganizationID).Count(&total)
	h.db.Model(&model.Membership{}).Where("organization_id = ? AND state = ?", principal.OrganizationID, "active").Count(&activeMembers)
	var recent []model.Content
	h.db.Where("organization_id = ?", principal.OrganizationID).Order("updated_at DESC").Limit(12).Find(&recent)
	recentItems := make([]gin.H, 0, len(recent))
	for _, item := range recent {
		recentItems = append(recentItems, contentAdminItem(item, h.db))
	}
	var pendingApplications []model.Application
	h.db.Where("organization_id = ? AND status = ?", principal.OrganizationID, "pending").Order("created_at DESC").Limit(12).Find(&pendingApplications)
	pendingItems := make([]gin.H, 0, len(pendingApplications))
	for _, item := range pendingApplications {
		pendingItems = append(pendingItems, applicationAdminItem(item))
	}
	respond(c, http.StatusOK, gin.H{"organization_name": organization.Name, "updated_at": time.Now().UTC(), "metrics": []gin.H{{"label": "活跃成员", "value": activeMembers, "change": "当前组织成员", "tone": "primary"}, {"label": "已发布内容", "value": published, "change": "当前公开内容", "tone": "secondary"}, {"label": "内容总数", "value": total, "change": "含草稿", "tone": "neutral"}, {"label": "在线玩家", "value": 18, "change": "服务器状态正常", "tone": "neutral"}}, "pending_applications": pendingItems, "recent_content": recentItems, "server": serverStatus()})
}
func contentItems() []gin.H {
	return []gin.H{{"id": "content_001", "title": "QUTCraft CMS 项目正式启动", "type": "news", "status": "published", "author": "QUTCraft Admin", "updated_at": "2026-07-17T03:00:00Z"}}
}
func serverStatus() gin.H {
	return gin.H{"enabled": true, "label": "QUTCraft Java 生存服", "state": "online", "online_players": 18, "max_players": 60}
}
func (h *WorkspaceHandler) AdminContent(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var contents []model.Content
	if err := h.db.Where("organization_id = ?", principal.OrganizationID).Order("updated_at DESC").Find(&contents).Error; err != nil {
		fail(c, http.StatusInternalServerError, "content.list_failed", "内容列表暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(contents))
	for _, item := range contents {
		items = append(items, contentAdminItem(item, h.db))
	}
	pageOf(c, items)
}

func (h *WorkspaceHandler) AdminKnowledgeDirectories(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var directories []model.KnowledgeDirectory
	if err := h.db.Where("organization_id = ?", principal.OrganizationID).Order("sort_order ASC, name ASC").Find(&directories).Error; err != nil {
		fail(c, http.StatusInternalServerError, "knowledge_directory.list_failed", "知识库目录暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(directories))
	for _, directory := range directories {
		items = append(items, knowledgeDirectoryItem(directory))
	}
	pageOf(c, items)
}

type knowledgeDirectoryRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	ParentID    string `json:"parent_id"`
	SortOrder   int    `json:"sort_order"`
	IsPublic    bool   `json:"is_public"`
}

func validKnowledgeDirectoryRequest(body knowledgeDirectoryRequest) bool {
	return strings.TrimSpace(body.Name) != "" && strings.TrimSpace(body.Slug) != "" && len([]rune(body.Name)) <= 120 && len([]rune(body.Slug)) <= 120 && len([]rune(body.Description)) <= 500 && body.SortOrder >= 0
}

func (h *WorkspaceHandler) AdminCreateKnowledgeDirectory(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var body knowledgeDirectoryRequest
	if err := c.ShouldBindJSON(&body); err != nil || !validKnowledgeDirectoryRequest(body) {
		fail(c, http.StatusBadRequest, "knowledge_directory.validation_failed", "知识库目录字段不符合规范。")
		return
	}
	directory := model.KnowledgeDirectory{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ParentID: strings.TrimSpace(body.ParentID), Name: strings.TrimSpace(body.Name), Slug: strings.TrimSpace(body.Slug), Description: strings.TrimSpace(body.Description), SortOrder: body.SortOrder, IsPublic: body.IsPublic}
	if err := h.db.Create(&directory).Error; err != nil {
		fail(c, http.StatusConflict, "knowledge_directory.slug_in_use", "知识库目录标识已存在。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusCreated, knowledgeDirectoryItem(directory))
}

func (h *WorkspaceHandler) AdminUpdateKnowledgeDirectory(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var body knowledgeDirectoryRequest
	if err := c.ShouldBindJSON(&body); err != nil || !validKnowledgeDirectoryRequest(body) {
		fail(c, http.StatusBadRequest, "knowledge_directory.validation_failed", "知识库目录字段不符合规范。")
		return
	}
	var directory model.KnowledgeDirectory
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&directory).Error; err != nil {
		fail(c, http.StatusNotFound, "knowledge_directory.not_found", "知识库目录不存在。")
		return
	}
	directory.ParentID, directory.Name, directory.Slug, directory.Description, directory.SortOrder, directory.IsPublic = strings.TrimSpace(body.ParentID), strings.TrimSpace(body.Name), strings.TrimSpace(body.Slug), strings.TrimSpace(body.Description), body.SortOrder, body.IsPublic
	if err := h.db.Save(&directory).Error; err != nil {
		fail(c, http.StatusConflict, "knowledge_directory.slug_in_use", "知识库目录标识已存在。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, knowledgeDirectoryItem(directory))
}

func knowledgeDirectoryItem(directory model.KnowledgeDirectory) gin.H {
	return gin.H{"id": directory.ID, "parent_id": directory.ParentID, "name": directory.Name, "slug": directory.Slug, "description": directory.Description, "sort_order": directory.SortOrder, "is_public": directory.IsPublic, "updated_at": directory.UpdatedAt}
}

func (h *WorkspaceHandler) AdminCreateContent(c *gin.Context) {
	var body struct {
		Title    string `json:"title"`
		Type     string `json:"type"`
		Category string `json:"category"`
		Excerpt  string `json:"excerpt"`
		Body     string `json:"body"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Title == "" {
		fail(c, http.StatusBadRequest, "content.validation_failed", "内容标题不能为空。")
		return
	}
	if len([]rune(body.Title)) > 160 || len([]rune(body.Category)) > 64 || len([]rune(body.Excerpt)) > 500 || (body.Type != "news" && body.Type != "resource" && body.Type != "knowledge") {
		fail(c, http.StatusBadRequest, "content.validation_failed", "title 最长 160 字符，type 必须为 news、resource 或 knowledge。")
		return
	}
	principal, _ := middleware.PrincipalFromContext(c)
	content := model.Content{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, AuthorUserID: principal.UserID, Title: body.Title, Type: body.Type, Category: strings.TrimSpace(body.Category), Status: "draft", Excerpt: body.Excerpt, Body: body.Body}
	if err := h.db.Create(&content).Error; err != nil {
		fail(c, http.StatusInternalServerError, "content.create_failed", "内容草稿创建失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusCreated, contentAdminItem(content, h.db))
}

func (h *WorkspaceHandler) AdminUpdateContent(c *gin.Context) {
	var body struct {
		Title    string `json:"title"`
		Type     string `json:"type"`
		Category string `json:"category"`
		Excerpt  string `json:"excerpt"`
		Body     string `json:"body"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "content.validation_failed", "内容格式不正确。")
		return
	}
	if strings.TrimSpace(body.Title) == "" || len([]rune(body.Title)) > 160 || len([]rune(body.Category)) > 64 || len([]rune(body.Excerpt)) > 500 || (body.Type != "news" && body.Type != "resource" && body.Type != "knowledge") {
		fail(c, http.StatusBadRequest, "content.validation_failed", "内容字段不符合规范。")
		return
	}
	principal, _ := middleware.PrincipalFromContext(c)
	var content model.Content
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&content).Error; err != nil {
		fail(c, http.StatusNotFound, "content.not_found", "内容不存在。")
		return
	}
	if content.Status == "published" {
		fail(c, http.StatusConflict, "content.published_immutable", "已发布内容不能直接编辑，请先下线。")
		return
	}
	content.Title, content.Type, content.Category, content.Excerpt, content.Body = strings.TrimSpace(body.Title), body.Type, strings.TrimSpace(body.Category), body.Excerpt, body.Body
	if err := h.db.Save(&content).Error; err != nil {
		fail(c, http.StatusInternalServerError, "content.update_failed", "内容保存失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, contentAdminItem(content, h.db))
}

func (h *WorkspaceHandler) PublishContent(c *gin.Context) { h.changeContentStatus(c, "published") }
func (h *WorkspaceHandler) ArchiveContent(c *gin.Context) { h.changeContentStatus(c, "archived") }
func (h *WorkspaceHandler) changeContentStatus(c *gin.Context, status string) {
	principal, _ := middleware.PrincipalFromContext(c)
	var content model.Content
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&content).Error; err != nil {
		fail(c, http.StatusNotFound, "content.not_found", "内容不存在。")
		return
	}
	if status == "published" && strings.TrimSpace(content.Title) == "" {
		fail(c, http.StatusBadRequest, "content.not_publishable", "内容标题不能为空。")
		return
	}
	if content.Status == status {
		fail(c, http.StatusConflict, "content.already_in_state", "内容已经处于目标状态。")
		return
	}
	if !canTransitionContentStatus(content.Status, status) {
		fail(c, http.StatusConflict, "content.invalid_transition", "内容不能从当前状态转换到目标状态。")
		return
	}
	content.Status = status
	if status == "published" {
		now := time.Now().UTC()
		content.PublishedAt = &now
	} else {
		content.PublishedAt = nil
	}
	if err := h.db.Save(&content).Error; err != nil {
		fail(c, http.StatusInternalServerError, "content.status_update_failed", "内容状态更新失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "content." + status, TargetType: "content", TargetID: content.ID, Result: "success", RequestID: ensureRequestID(c)}).Error
	respond(c, http.StatusOK, contentAdminItem(content, h.db))
}

func canTransitionContentStatus(current, target string) bool {
	switch target {
	case "published":
		return current == "draft" || current == "review" || current == "archived"
	case "archived":
		return current == "published"
	default:
		return false
	}
}

func contentPublicItem(content model.Content) gin.H {
	publishedAt := content.PublishedAt
	if publishedAt == nil {
		publishedAt = &content.UpdatedAt
	}
	category := content.Category
	if category == "" {
		category = content.Type
	}
	return gin.H{"id": content.ID, "title": content.Title, "excerpt": content.Excerpt, "category": category, "published_at": publishedAt, "reading_minutes": maxInt(1, len([]rune(content.Body))/900+1)}
}

func (h *WorkspaceHandler) contentPublicDetailItem(slug string, content model.Content) gin.H {
	item := gin.H{
		"id":              content.ID,
		"title":           content.Title,
		"type":            content.Type,
		"category":        content.Category,
		"excerpt":         content.Excerpt,
		"body":            content.Body,
		"published_at":    content.PublishedAt,
		"updated_at":      content.UpdatedAt,
		"reading_minutes": maxInt(1, len([]rune(content.Body))/900+1),
	}
	if content.Type == "resource" {
		var asset model.MediaAsset
		if h.db.Where("content_id = ? AND organization_id = ?", content.ID, content.OrganizationID).Order("created_at ASC").First(&asset).Error == nil {
			item["asset"] = gin.H{"id": asset.ID, "original_name": asset.OriginalName, "mime_type": asset.MimeType, "size_bytes": asset.SizeBytes}
			item["download_url"] = "/api/v1/portal/organizations/" + slug + "/assets/" + asset.ID + "/download"
		} else {
			item["asset"] = nil
			item["download_url"] = nil
		}
	}
	return item
}

func (h *WorkspaceHandler) resourcePublicItem(slug string, content model.Content) gin.H {
	kind := content.Category
	if kind != "document" && kind != "template" && kind != "package" && kind != "video" {
		kind = "document"
	}
	item := gin.H{"id": content.ID, "title": content.Title, "description": content.Excerpt, "kind": kind, "size_bytes": int64(0), "updated_at": content.UpdatedAt, "download_url": nil}
	var asset model.MediaAsset
	if h.db.Where("content_id = ? AND organization_id = ?", content.ID, content.OrganizationID).Order("created_at ASC").First(&asset).Error == nil {
		item["size_bytes"] = asset.SizeBytes
		item["download_url"] = "/api/v1/portal/organizations/" + slug + "/assets/" + asset.ID + "/download"
	}
	return item
}

func contentAdminItem(content model.Content, db *gorm.DB) gin.H {
	var author model.User
	_ = db.First(&author, "id = ?", content.AuthorUserID).Error
	return gin.H{"id": content.ID, "title": content.Title, "type": content.Type, "category": content.Category, "status": content.Status, "author": author.DisplayName, "excerpt": content.Excerpt, "body": content.Body, "published_at": content.PublishedAt, "updated_at": content.UpdatedAt}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (h *WorkspaceHandler) AdminUsers(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	type row struct {
		MembershipID    string    `gorm:"column:membership_id"`
		UserID          string    `gorm:"column:user_id"`
		Name            string    `gorm:"column:name"`
		Email           string    `gorm:"column:email"`
		UserState       string    `gorm:"column:user_state"`
		MembershipState string    `gorm:"column:membership_state"`
		Role            string    `gorm:"column:role"`
		JoinedAt        time.Time `gorm:"column:joined_at"`
	}
	var rows []row
	err := h.db.Table("memberships AS m").Select("m.id AS membership_id, u.id AS user_id, u.display_name AS name, u.email AS email, u.state AS user_state, m.state AS membership_state, COALESCE(MAX(r.key), 'member') AS role, m.created_at AS joined_at").Joins("JOIN users AS u ON u.id = m.user_id").Joins("LEFT JOIN membership_roles AS mr ON mr.membership_id = m.id").Joins("LEFT JOIN roles AS r ON r.id = mr.role_id").Where("m.organization_id = ?", principal.OrganizationID).Group("m.id, u.id, u.display_name, u.email, u.state, m.state, m.created_at").Order("m.created_at ASC").Scan(&rows).Error
	if err != nil {
		fail(c, http.StatusInternalServerError, "membership.list_failed", "成员列表暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, item := range rows {
		state := item.UserState
		if item.MembershipState != "active" {
			state = item.MembershipState
		}
		items = append(items, gin.H{"id": item.UserID, "name": item.Name, "email": item.Email, "role": item.Role, "state": state, "joined_at": item.JoinedAt})
	}
	pageOf(c, items)
}

func (h *WorkspaceHandler) MembershipHistory(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var membership model.Membership
	if err := h.db.Where("organization_id = ? AND user_id = ?", principal.OrganizationID, principal.UserID).First(&membership).Error; err != nil {
		fail(c, http.StatusNotFound, "membership.not_found", "当前组织成员关系不存在。")
		return
	}
	var events []model.MembershipEvent
	if err := h.db.Where("membership_id = ?", membership.ID).Order("created_at DESC").Find(&events).Error; err != nil {
		fail(c, http.StatusInternalServerError, "membership.history_failed", "成员变更记录暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(events))
	for _, event := range events {
		items = append(items, gin.H{"id": event.ID, "state": event.State, "reason": event.Reason, "created_at": event.CreatedAt})
	}
	pageOf(c, items)
}

func (h *WorkspaceHandler) LeaveMembership(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var membership model.Membership
	if err := h.db.Where("organization_id = ? AND user_id = ?", principal.OrganizationID, principal.UserID).First(&membership).Error; err != nil {
		fail(c, http.StatusNotFound, "membership.not_found", "当前组织成员关系不存在。")
		return
	}
	if membership.State != "active" {
		fail(c, http.StatusConflict, "membership.already_left", "当前成员关系已经不是 active 状态。")
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&membership).Update("state", "left").Error; err != nil {
			return err
		}
		if err := tx.Where("membership_id = ?", membership.ID).Delete(&model.MembershipRole{}).Error; err != nil {
			return err
		}
		return tx.Create(&model.MembershipEvent{ID: uuid.NewString(), MembershipID: membership.ID, State: "left", Reason: "self_leave"}).Error
	}); err != nil {
		fail(c, http.StatusInternalServerError, "membership.leave_failed", "退出组织暂时无法完成。")
		return
	}
	respond(c, http.StatusOK, gin.H{"state": "left", "left_at": time.Now().UTC()})
}

func (h *WorkspaceHandler) AdminUpdateUser(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var body struct {
		State string `json:"state"`
		Role  string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || !validMemberState(body.State) || !validRole(body.Role) {
		fail(c, http.StatusBadRequest, "membership.validation_failed", "成员状态或角色不符合规范。")
		return
	}
	var membership model.Membership
	if err := h.db.Where("organization_id = ? AND user_id = ?", principal.OrganizationID, c.Param("id")).First(&membership).Error; err != nil {
		fail(c, http.StatusNotFound, "membership.not_found", "成员不存在。")
		return
	}
	var user model.User
	if err := h.db.First(&user, "id = ?", c.Param("id")).Error; err != nil {
		fail(c, http.StatusNotFound, "user.not_found", "用户不存在。")
		return
	}
	targetRole := membershipRole(h.db, membership.ID)
	actorRole := membershipRoleByUser(h.db, principal.OrganizationID, principal.UserID)
	if code := membershipChangeError(actorRole, c.Param("id") == principal.UserID, targetRole, body.Role, body.State); code != "" {
		status := http.StatusConflict
		message := "成员权限变更不符合保护规则。"
		if code == "membership.owner_only" {
			status = http.StatusForbidden
			message = "只有所有者可以授予所有者角色。"
		} else if code == "membership.owner_protected" {
			message = "所有者不能被停用或降级。"
		} else if code == "membership.self_change_forbidden" {
			message = "不能通过成员管理解除自己的所有者或管理权限。"
		}
		fail(c, status, code, message)
		return
	}
	var role model.Role
	if err := h.db.Where("`key` = ?", body.Role).First(&role).Error; err != nil {
		fail(c, http.StatusBadRequest, "membership.role_not_found", "角色不存在。")
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&membership).Update("state", body.State).Error; err != nil {
			return err
		}
		if err := tx.Model(&user).Update("state", body.State).Error; err != nil {
			return err
		}
		if err := tx.Where("membership_id = ?", membership.ID).Delete(&model.MembershipRole{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.MembershipRole{MembershipID: membership.ID, RoleID: role.ID}).Error; err != nil {
			return err
		}
		return tx.Create(&model.MembershipEvent{ID: uuid.NewString(), MembershipID: membership.ID, State: body.State, Reason: "admin_update"}).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "membership.update_failed", "成员信息保存失败。")
		return
	}
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "membership.update", TargetType: "membership", TargetID: membership.ID, Result: "success", RequestID: ensureRequestID(c)}).Error
	respond(c, http.StatusOK, gin.H{"id": user.ID, "name": user.DisplayName, "email": user.Email, "role": body.Role, "state": body.State, "joined_at": membership.CreatedAt})
}

func validMemberState(value string) bool {
	return value == "active" || value == "invited" || value == "disabled"
}

func validRole(value string) bool {
	return value == "member" || value == "editor" || value == "administrator" || value == "owner"
}

func membershipRole(db *gorm.DB, membershipID string) string {
	var role string
	db.Table("membership_roles AS mr").Select("COALESCE(MAX(r.`key`), 'member')").Joins("JOIN roles AS r ON r.id = mr.role_id").Where("mr.membership_id = ?", membershipID).Scan(&role)
	return role
}

func membershipRoleByUser(db *gorm.DB, organizationID, userID string) string {
	var role string
	db.Table("memberships AS m").Select("COALESCE(MAX(r.`key`), 'member')").Joins("JOIN membership_roles AS mr ON mr.membership_id = m.id").Joins("JOIN roles AS r ON r.id = mr.role_id").Where("m.organization_id = ? AND m.user_id = ? AND m.state = ?", organizationID, userID, "active").Scan(&role)
	return role
}

func membershipChangeError(actorRole string, actorIsSelf bool, currentRole, nextRole, nextState string) string {
	if currentRole == "owner" && (nextRole != "owner" || nextState != "active") {
		return "membership.owner_protected"
	}
	if nextRole == "owner" && actorRole != "owner" {
		return "membership.owner_only"
	}
	if actorIsSelf && (nextRole != currentRole || nextState != "active") {
		return "membership.self_change_forbidden"
	}
	return ""
}

func (h *WorkspaceHandler) AdminProjects(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var projects []model.Project
	if err := h.db.Where("organization_id = ?", principal.OrganizationID).Order("updated_at DESC").Find(&projects).Error; err != nil {
		fail(c, http.StatusInternalServerError, "project.list_failed", "项目列表暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectAdminItem(project, h.db))
	}
	pageOf(c, items)
}

func (h *WorkspaceHandler) AdminCreateProject(c *gin.Context) {
	principal, _ := middleware.PrincipalFromContext(c)
	var body projectRequest
	if err := c.ShouldBindJSON(&body); err != nil || !validProjectRequest(body) {
		fail(c, http.StatusBadRequest, "project.validation_failed", "项目字段不符合规范。")
		return
	}
	project := model.Project{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, OwnerUserID: principal.UserID, Title: strings.TrimSpace(body.Title), Summary: strings.TrimSpace(body.Summary), Status: body.Status, Tags: strings.Join(body.Tags, ","), IsPublic: body.IsPublic}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&project).Error; err != nil {
			return err
		}
		return tx.Create(&model.ProjectMember{ProjectID: project.ID, UserID: principal.UserID, Role: "owner"}).Error
	}); err != nil {
		fail(c, http.StatusInternalServerError, "project.create_failed", "项目创建失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusCreated, projectAdminItem(project, h.db))
}

func (h *WorkspaceHandler) AdminUpdateProject(c *gin.Context) {
	principal, _ := middleware.PrincipalFromContext(c)
	var body projectRequest
	if err := c.ShouldBindJSON(&body); err != nil || !validProjectRequest(body) {
		fail(c, http.StatusBadRequest, "project.validation_failed", "项目字段不符合规范。")
		return
	}
	var project model.Project
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&project).Error; err != nil {
		fail(c, http.StatusNotFound, "project.not_found", "项目不存在。")
		return
	}
	project.Title, project.Summary, project.Status, project.Tags, project.IsPublic = strings.TrimSpace(body.Title), strings.TrimSpace(body.Summary), body.Status, strings.Join(body.Tags, ","), body.IsPublic
	if err := h.db.Save(&project).Error; err != nil {
		fail(c, http.StatusInternalServerError, "project.update_failed", "项目保存失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, projectAdminItem(project, h.db))
}

func (h *WorkspaceHandler) projectForPrincipal(c *gin.Context) (model.Project, service.Principal, bool) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return model.Project{}, service.Principal{}, false
	}
	var project model.Project
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&project).Error; err != nil {
		fail(c, http.StatusNotFound, "project.not_found", "项目不存在。")
		return model.Project{}, service.Principal{}, false
	}
	return project, principal, true
}

func (h *WorkspaceHandler) AdminProjectMembers(c *gin.Context) {
	project, _, ok := h.projectForPrincipal(c)
	if !ok {
		return
	}
	var rows []struct {
		UserID    string    `gorm:"column:user_id"`
		Name      string    `gorm:"column:name"`
		Email     string    `gorm:"column:email"`
		UserState string    `gorm:"column:user_state"`
		Role      string    `gorm:"column:role"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}
	err := h.db.Table("project_members AS pm").
		Select("pm.user_id, u.display_name AS name, u.email, u.state AS user_state, pm.role, pm.created_at").
		Joins("JOIN users AS u ON BINARY u.id = BINARY pm.user_id").
		Where("pm.project_id = ?", project.ID).
		Order("pm.role ASC, pm.created_at ASC").Scan(&rows).Error
	if err != nil {
		fail(c, http.StatusInternalServerError, "project_member.list_failed", "项目成员列表暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, gin.H{"user_id": row.UserID, "name": row.Name, "email": row.Email, "state": row.UserState, "role": row.Role, "assigned_at": row.CreatedAt})
	}
	pageOf(c, items)
}

type projectMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func validProjectMemberRole(value string) bool {
	return value == "member" || value == "contributor" || value == "lead"
}

func (h *WorkspaceHandler) AdminAddProjectMember(c *gin.Context) {
	project, principal, ok := h.projectForPrincipal(c)
	if !ok {
		return
	}
	var body projectMemberRequest
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.UserID) == "" || !validProjectMemberRole(body.Role) {
		fail(c, http.StatusBadRequest, "project_member.validation_failed", "user_id 不能为空，role 必须为 member、contributor 或 lead。")
		return
	}
	var membership model.Membership
	if err := h.db.Where("organization_id = ? AND user_id = ? AND state = ?", principal.OrganizationID, body.UserID, "active").First(&membership).Error; err != nil {
		fail(c, http.StatusBadRequest, "project_member.user_not_member", "只能添加当前组织中的活跃成员。")
		return
	}
	var member model.ProjectMember
	result := h.db.Where("project_id = ? AND user_id = ?", project.ID, body.UserID).First(&member)
	if result.Error == nil {
		if member.Role == "owner" {
			fail(c, http.StatusConflict, "project_member.owner_immutable", "项目负责人不能通过成员角色接口修改。")
			return
		}
		member.Role = body.Role
		if err := h.db.Save(&member).Error; err != nil {
			fail(c, http.StatusInternalServerError, "project_member.update_failed", "项目成员角色保存失败。")
			return
		}
		respond(c, http.StatusOK, projectMemberItem(h.db, member))
		return
	}
	if result.Error != gorm.ErrRecordNotFound {
		fail(c, http.StatusInternalServerError, "project_member.create_failed", "项目成员暂时无法保存。")
		return
	}
	member = model.ProjectMember{ProjectID: project.ID, UserID: body.UserID, Role: body.Role}
	if err := h.db.Create(&member).Error; err != nil {
		fail(c, http.StatusInternalServerError, "project_member.create_failed", "项目成员添加失败。")
		return
	}
	respond(c, http.StatusCreated, projectMemberItem(h.db, member))
}

func (h *WorkspaceHandler) AdminUpdateProjectMember(c *gin.Context) {
	project, _, ok := h.projectForPrincipal(c)
	if !ok {
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || !validProjectMemberRole(body.Role) {
		fail(c, http.StatusBadRequest, "project_member.validation_failed", "role 必须为 member、contributor 或 lead。")
		return
	}
	var member model.ProjectMember
	if err := h.db.Where("project_id = ? AND user_id = ?", project.ID, c.Param("user_id")).First(&member).Error; err != nil {
		fail(c, http.StatusNotFound, "project_member.not_found", "项目成员不存在。")
		return
	}
	if member.Role == "owner" {
		fail(c, http.StatusConflict, "project_member.owner_immutable", "项目负责人不能通过成员角色接口修改。")
		return
	}
	member.Role = body.Role
	if err := h.db.Save(&member).Error; err != nil {
		fail(c, http.StatusInternalServerError, "project_member.update_failed", "项目成员角色保存失败。")
		return
	}
	respond(c, http.StatusOK, projectMemberItem(h.db, member))
}

func (h *WorkspaceHandler) AdminRemoveProjectMember(c *gin.Context) {
	project, _, ok := h.projectForPrincipal(c)
	if !ok {
		return
	}
	var member model.ProjectMember
	if err := h.db.Where("project_id = ? AND user_id = ?", project.ID, c.Param("user_id")).First(&member).Error; err != nil {
		fail(c, http.StatusNotFound, "project_member.not_found", "项目成员不存在。")
		return
	}
	if member.Role == "owner" || member.UserID == project.OwnerUserID {
		fail(c, http.StatusConflict, "project_member.owner_immutable", "项目负责人不能移出项目。")
		return
	}
	if err := h.db.Delete(&member).Error; err != nil {
		fail(c, http.StatusInternalServerError, "project_member.delete_failed", "项目成员移除失败。")
		return
	}
	respond(c, http.StatusOK, gin.H{"removed": true, "user_id": member.UserID, "project_id": project.ID})
}

func projectMemberItem(db *gorm.DB, member model.ProjectMember) gin.H {
	var user model.User
	_ = db.First(&user, "id = ?", member.UserID).Error
	return gin.H{"user_id": member.UserID, "name": user.DisplayName, "email": user.Email, "state": user.State, "role": member.Role, "assigned_at": member.CreatedAt}
}

func (h *WorkspaceHandler) AdminProjectMilestones(c *gin.Context) {
	project, _, ok := h.projectForPrincipal(c)
	if !ok {
		return
	}
	var milestones []model.ProjectMilestone
	if err := h.db.Where("project_id = ?", project.ID).Order("due_at IS NULL, due_at ASC, created_at ASC").Find(&milestones).Error; err != nil {
		fail(c, http.StatusInternalServerError, "project_milestone.list_failed", "项目里程碑列表暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(milestones))
	for _, milestone := range milestones {
		items = append(items, projectMilestoneItem(milestone))
	}
	pageOf(c, items)
}

type projectMilestoneRequest struct {
	Title  string `json:"title"`
	Status string `json:"status"`
	DueAt  string `json:"due_at"`
}

func validProjectMilestoneStatus(value string) bool {
	return value == "planned" || value == "active" || value == "completed"
}

func parseOptionalTime(value string) (*time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, false
	}
	parsed = parsed.UTC()
	return &parsed, true
}

func validProjectMilestoneRequest(body projectMilestoneRequest) bool {
	return strings.TrimSpace(body.Title) != "" && len([]rune(body.Title)) <= 160 && validProjectMilestoneStatus(body.Status)
}

func (h *WorkspaceHandler) AdminCreateProjectMilestone(c *gin.Context) {
	project, _, ok := h.projectForPrincipal(c)
	if !ok {
		return
	}
	var body projectMilestoneRequest
	if err := c.ShouldBindJSON(&body); err != nil || !validProjectMilestoneRequest(body) {
		fail(c, http.StatusBadRequest, "project_milestone.validation_failed", "里程碑标题、状态或日期不符合规范。")
		return
	}
	dueAt, valid := parseOptionalTime(body.DueAt)
	if !valid {
		fail(c, http.StatusBadRequest, "project_milestone.invalid_due_at", "due_at 必须是 RFC3339 日期时间。")
		return
	}
	var completedAt *time.Time
	if body.Status == "completed" {
		now := time.Now().UTC()
		completedAt = &now
	}
	milestone := model.ProjectMilestone{ID: uuid.NewString(), ProjectID: project.ID, Title: strings.TrimSpace(body.Title), Status: body.Status, DueAt: dueAt, CompletedAt: completedAt}
	if err := h.db.Create(&milestone).Error; err != nil {
		fail(c, http.StatusInternalServerError, "project_milestone.create_failed", "里程碑创建失败。")
		return
	}
	respond(c, http.StatusCreated, projectMilestoneItem(milestone))
}

func (h *WorkspaceHandler) AdminUpdateProjectMilestone(c *gin.Context) {
	project, _, ok := h.projectForPrincipal(c)
	if !ok {
		return
	}
	var body projectMilestoneRequest
	if err := c.ShouldBindJSON(&body); err != nil || !validProjectMilestoneRequest(body) {
		fail(c, http.StatusBadRequest, "project_milestone.validation_failed", "里程碑标题、状态或日期不符合规范。")
		return
	}
	dueAt, valid := parseOptionalTime(body.DueAt)
	if !valid {
		fail(c, http.StatusBadRequest, "project_milestone.invalid_due_at", "due_at 必须是 RFC3339 日期时间。")
		return
	}
	var milestone model.ProjectMilestone
	if err := h.db.Where("id = ? AND project_id = ?", c.Param("milestone_id"), project.ID).First(&milestone).Error; err != nil {
		fail(c, http.StatusNotFound, "project_milestone.not_found", "里程碑不存在。")
		return
	}
	milestone.Title, milestone.Status, milestone.DueAt = strings.TrimSpace(body.Title), body.Status, dueAt
	if body.Status == "completed" {
		if milestone.CompletedAt == nil {
			now := time.Now().UTC()
			milestone.CompletedAt = &now
		}
	} else {
		milestone.CompletedAt = nil
	}
	if err := h.db.Save(&milestone).Error; err != nil {
		fail(c, http.StatusInternalServerError, "project_milestone.update_failed", "里程碑保存失败。")
		return
	}
	respond(c, http.StatusOK, projectMilestoneItem(milestone))
}

func (h *WorkspaceHandler) AdminDeleteProjectMilestone(c *gin.Context) {
	project, _, ok := h.projectForPrincipal(c)
	if !ok {
		return
	}
	var milestone model.ProjectMilestone
	if err := h.db.Where("id = ? AND project_id = ?", c.Param("milestone_id"), project.ID).First(&milestone).Error; err != nil {
		fail(c, http.StatusNotFound, "project_milestone.not_found", "里程碑不存在。")
		return
	}
	if err := h.db.Delete(&milestone).Error; err != nil {
		fail(c, http.StatusInternalServerError, "project_milestone.delete_failed", "里程碑删除失败。")
		return
	}
	respond(c, http.StatusOK, gin.H{"removed": true, "id": milestone.ID, "project_id": project.ID})
}

func projectMilestoneItem(milestone model.ProjectMilestone) gin.H {
	return gin.H{"id": milestone.ID, "project_id": milestone.ProjectID, "title": milestone.Title, "status": milestone.Status, "due_at": milestone.DueAt, "completed_at": milestone.CompletedAt, "updated_at": milestone.UpdatedAt}
}

type projectRequest struct {
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	Status   string   `json:"status"`
	Tags     []string `json:"tags"`
	IsPublic bool     `json:"is_public"`
}

func validProjectRequest(body projectRequest) bool {
	if strings.TrimSpace(body.Title) == "" || len([]rune(body.Title)) > 160 || len([]rune(body.Summary)) > 500 {
		return false
	}
	if body.Status != "active" && body.Status != "research" && body.Status != "completed" {
		return false
	}
	return len(body.Tags) <= 12
}

func projectPublicItem(project model.Project) gin.H {
	return gin.H{"id": project.ID, "title": project.Title, "summary": project.Summary, "status": project.Status, "tags": splitTags(project.Tags), "updated_at": project.UpdatedAt}
}

func projectAdminItem(project model.Project, db *gorm.DB) gin.H {
	var owner model.User
	_ = db.First(&owner, "id = ?", project.OwnerUserID).Error
	var memberCount, milestoneCount int64
	db.Model(&model.ProjectMember{}).Where("project_id = ?", project.ID).Count(&memberCount)
	db.Model(&model.ProjectMilestone{}).Where("project_id = ?", project.ID).Count(&milestoneCount)
	return gin.H{"id": project.ID, "title": project.Title, "summary": project.Summary, "status": project.Status, "tags": splitTags(project.Tags), "is_public": project.IsPublic, "owner": owner.DisplayName, "member_count": memberCount, "milestone_count": milestoneCount, "updated_at": project.UpdatedAt}
}

func splitTags(value string) []string {
	parts := strings.Split(value, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}

type applicationRequest struct {
	Type      string `json:"type"`
	ClassName string `json:"class_name"`
	Name      string `json:"name"`
	GameID    string `json:"game_id"`
	QQNumber  string `json:"qq_number"`
	Email     string `json:"email"`
	Note      string `json:"note"`
}

var qqNumberPattern = regexp.MustCompile(`^[0-9]{5,15}$`)

func validApplicationRequest(body applicationRequest) bool {
	if body.Type == "" {
		body.Type = "whitelist"
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address != email {
		return false
	}
	return (body.Type == "whitelist" || body.Type == "membership") &&
		strings.TrimSpace(body.ClassName) != "" && len([]rune(body.ClassName)) <= 120 &&
		strings.TrimSpace(body.Name) != "" && len([]rune(body.Name)) <= 80 &&
		strings.TrimSpace(body.GameID) != "" && len([]rune(body.GameID)) <= 80 &&
		qqNumberPattern.MatchString(strings.TrimSpace(body.QQNumber)) &&
		len([]rune(body.Note)) <= 500
}

func (h *WorkspaceHandler) SubmitApplication(c *gin.Context) {
	var body applicationRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "application.validation_failed", "申请数据格式不正确。")
		return
	}
	if strings.TrimSpace(body.Type) == "" {
		body.Type = "whitelist"
	}
	body.Type = strings.TrimSpace(body.Type)
	body.ClassName = strings.TrimSpace(body.ClassName)
	body.Name = strings.TrimSpace(body.Name)
	body.GameID = strings.TrimSpace(body.GameID)
	body.QQNumber = strings.TrimSpace(body.QQNumber)
	body.Email = strings.ToLower(strings.TrimSpace(body.Email))
	body.Note = strings.TrimSpace(body.Note)
	if !validApplicationRequest(body) {
		fail(c, http.StatusBadRequest, "application.validation_failed", "班级、姓名、游戏 ID、QQ 号码或邮箱不符合规范。")
		return
	}

	var organization model.Organization
	if err := h.db.Where("slug = ?", c.Param("slug")).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}

	var existing model.Application
	duplicateQuery := h.db.Where("organization_id = ? AND status = ? AND (LOWER(email) = ? OR game_id = ?)", organization.ID, "pending", body.Email, body.GameID)
	if err := duplicateQuery.First(&existing).Error; err == nil {
		fail(c, http.StatusConflict, "application.duplicate_pending", "相同邮箱或游戏 ID 已有待处理申请。")
		return
	} else if err != gorm.ErrRecordNotFound {
		fail(c, http.StatusInternalServerError, "application.lookup_failed", "申请暂时无法提交。")
		return
	}

	note := body.Note
	if note == "" {
		note = strings.Join([]string{"班级/专业：" + body.ClassName, "游戏 ID：" + body.GameID}, "；")
	}
	application := model.Application{
		ID:             uuid.NewString(),
		OrganizationID: organization.ID,
		Type:           body.Type,
		ClassName:      body.ClassName,
		ApplicantName:  body.Name,
		GameID:         body.GameID,
		QQNumber:       body.QQNumber,
		Email:          body.Email,
		Note:           note,
		Status:         "pending",
	}
	if err := h.db.Create(&application).Error; err != nil {
		fail(c, http.StatusInternalServerError, "application.create_failed", "申请暂时无法提交，请稍后重试。")
		return
	}
	respond(c, http.StatusCreated, gin.H{"id": application.ID, "status": application.Status, "submitted_at": application.CreatedAt})
}

func (h *WorkspaceHandler) AdminApplications(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var applications []model.Application
	if err := h.db.Where("organization_id = ?", principal.OrganizationID).Order("created_at DESC").Find(&applications).Error; err != nil {
		fail(c, http.StatusInternalServerError, "application.list_failed", "申请列表暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(applications))
	for _, application := range applications {
		items = append(items, applicationAdminItem(application))
	}
	pageOf(c, items)
}

func (h *WorkspaceHandler) AdminApplicationDecision(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	decision := c.Param("id")
	next := "approved"
	if strings.HasSuffix(c.FullPath(), "/reject") {
		next = "rejected"
	}
	now := time.Now().UTC()
	tx := h.db.Begin()
	if tx.Error != nil {
		fail(c, http.StatusInternalServerError, "application.decision_failed", "申请状态暂时无法更新。")
		return
	}
	result := tx.Model(&model.Application{}).Where("id = ? AND organization_id = ? AND status = ?", decision, principal.OrganizationID, "pending").Updates(map[string]interface{}{"status": next, "decided_at": now, "decided_by": principal.UserID})
	if result.Error != nil {
		tx.Rollback()
		fail(c, http.StatusInternalServerError, "application.decision_failed", "申请状态暂时无法更新。")
		return
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		var existing model.Application
		err := h.db.Where("id = ? AND organization_id = ?", decision, principal.OrganizationID).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			fail(c, http.StatusNotFound, "application.not_found", "申请不存在。")
		} else {
			fail(c, http.StatusConflict, "application.already_decided", "申请已经处理，不能重复审批。")
		}
		return
	}
	if err := tx.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "application." + next, TargetType: "application", TargetID: decision, Result: "success", RequestID: ensureRequestID(c), CreatedAt: now}).Error; err != nil {
		tx.Rollback()
		fail(c, http.StatusInternalServerError, "application.audit_failed", "申请状态已被阻止提交，请稍后重试。")
		return
	}
	if err := tx.Commit().Error; err != nil {
		fail(c, http.StatusInternalServerError, "application.decision_failed", "申请状态暂时无法更新。")
		return
	}
	var application model.Application
	if err := h.db.Where("id = ?", decision).First(&application).Error; err != nil {
		fail(c, http.StatusInternalServerError, "application.read_failed", "申请状态已更新，但结果暂时无法读取。")
		return
	}
	respond(c, http.StatusOK, applicationAdminItem(application))
}

func applicationAdminItem(application model.Application) gin.H {
	return gin.H{"id": application.ID, "applicant": application.ApplicantName, "type": application.Type, "submitted_at": application.CreatedAt, "note": application.Note, "status": application.Status, "class_name": application.ClassName, "game_id": application.GameID, "qq_number": application.QQNumber, "email": application.Email, "decided_at": application.DecidedAt, "decided_by": application.DecidedBy}
}

func (h *WorkspaceHandler) AdminServerStatus(c *gin.Context) {
	respond(c, http.StatusOK, serverStatus())
}
func (h *WorkspaceHandler) AdminServerCommand(c *gin.Context) {
	var body struct {
		Command string `json:"command"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Command == "" {
		fail(c, http.StatusBadRequest, "server.command_invalid", "命令不能为空。")
		return
	}
	body.Command = strings.TrimSpace(body.Command)
	if len([]rune(body.Command)) > 256 || strings.ContainsAny(body.Command, "\r\n") || !allowedCommand(body.Command) {
		fail(c, http.StatusForbidden, "server.command_not_allowed", "命令不在服务端白名单中。")
		return
	}
	principal, _ := middleware.PrincipalFromContext(c)
	now := time.Now().UTC()
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "server.command", TargetType: "server", Result: "accepted", RequestID: ensureRequestID(c), CreatedAt: now}).Error
	respond(c, http.StatusOK, gin.H{"accepted": true, "message": "开发环境已记录命令，未连接真实 RCON。", "executed_at": now})
}

func allowedCommand(command string) bool {
	for _, allowed := range []string{"list", "save-all", "time set day", "weather clear"} {
		if command == allowed {
			return true
		}
	}
	return strings.HasPrefix(command, "say ") && len(command) > 4
}

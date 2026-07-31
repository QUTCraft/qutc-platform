package handler

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/cache"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/serveradapter"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/storage"
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
	serverAdapter  serveradapter.Adapter
	mediaStorage   storage.Store
	applications   *service.ApplicationDecisionService
}

var (
	errContentDirectoryTypeInvalid = errors.New("knowledge directory can only be used by knowledge content")
	errContentDirectoryRequired    = errors.New("knowledge content requires a directory")
	errContentDirectoryNotFound    = errors.New("knowledge directory does not belong to the organization")
	errKnowledgeParentInvalid      = errors.New("knowledge directory parent is invalid")
	errKnowledgeParentCycle        = errors.New("knowledge directory parent creates a cycle")
)

var markdownAdminAssetPattern = regexp.MustCompile(`/api/v1/admin/assets/([a-zA-Z0-9-]+)/download`)

func NewWorkspaceHandler(db *gorm.DB, publicCache *cache.Cache, environment string) *WorkspaceHandler {
	return NewWorkspaceHandlerWithServerAdapterTimeout(db, publicCache, environment, serveradapter.NewMock(), 5*time.Second)
}

func NewWorkspaceHandlerWithServerAdapter(db *gorm.DB, publicCache *cache.Cache, environment string, adapter serveradapter.Adapter) *WorkspaceHandler {
	return NewWorkspaceHandlerWithServerAdapterTimeout(db, publicCache, environment, adapter, 5*time.Second)
}

func NewWorkspaceHandlerWithServerAdapterTimeout(db *gorm.DB, publicCache *cache.Cache, environment string, adapter serveradapter.Adapter, timeout time.Duration) *WorkspaceHandler {
	mediaStorage, err := storage.NewLocal("/tmp/qutcraft-uploads")
	if err != nil {
		panic(err)
	}
	return NewWorkspaceHandlerWithDependencies(db, publicCache, environment, adapter, timeout, mediaStorage)
}

func NewWorkspaceHandlerWithDependencies(db *gorm.DB, publicCache *cache.Cache, environment string, adapter serveradapter.Adapter, timeout time.Duration, mediaStorage storage.Store) *WorkspaceHandler {
	if strings.TrimSpace(environment) == "" {
		environment = "development"
	}
	if adapter == nil {
		adapter = serveradapter.NewMock()
	}
	if mediaStorage == nil {
		var err error
		mediaStorage, err = storage.NewLocal("/tmp/qutcraft-uploads")
		if err != nil {
			panic(err)
		}
	}
	adapter = serveradapter.WithTimeout(adapter, timeout)
	return &WorkspaceHandler{
		db:             db,
		cache:          publicCache,
		cacheNamespace: environment,
		serverAdapter:  adapter,
		mediaStorage:   mediaStorage,
		applications:   service.NewApplicationDecisionService(db, adapter),
	}
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
		if err := h.db.Where("id = ? AND organization_id = ? AND status = ?", contentID, organization.ID, service.ContentStatusPublished).Where("(type <> ? OR knowledge_directory_id IS NULL OR knowledge_directory_id = '' OR EXISTS (SELECT 1 FROM knowledge_directories AS directory WHERE directory.id = contents.knowledge_directory_id AND directory.organization_id = contents.organization_id AND directory.is_public = ?))", service.ContentTypeKnowledge, true).First(&content).Error; err != nil {
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
		query = query.Where("category = ?", category)
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
	query := h.db.Table("contents AS content").Select("content.*").Joins("LEFT JOIN knowledge_directories AS directory ON directory.id = content.knowledge_directory_id AND directory.organization_id = content.organization_id").Where("content.organization_id = ? AND content.type = ? AND content.status = ?", organization.ID, service.ContentTypeKnowledge, service.ContentStatusPublished).Where("(content.knowledge_directory_id IS NULL OR content.knowledge_directory_id = '' OR directory.is_public = ?)", true).Order("content.updated_at DESC")
	if category != "" {
		query = query.Where("(content.category = ? OR directory.name = ? OR directory.slug = ? OR content.title LIKE ? OR content.excerpt LIKE ?)", category, category, category, "%"+category+"%", "%"+category+"%")
	}
	if q != "" {
		query = query.Where("(content.title LIKE ? OR content.excerpt LIKE ? OR content.body LIKE ?)", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	h.cachedPortalPage(c, c.Param("slug"), "knowledge", func() ([]gin.H, error) {
		var contents []model.Content
		if err := query.Find(&contents).Error; err != nil {
			return nil, err
		}
		directoryNames := h.knowledgeDirectoryNames(organization.ID, contents)
		items := make([]gin.H, 0, len(contents))
		for _, content := range contents {
			categoryName := content.Category
			if content.KnowledgeDirectoryID != nil {
				if name := directoryNames[*content.KnowledgeDirectoryID]; name != "" {
					categoryName = name
				}
			}
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
			h.db.Model(&model.Content{}).Where("organization_id = ? AND type = ? AND status = ? AND (knowledge_directory_id = ? OR ((knowledge_directory_id IS NULL OR knowledge_directory_id = '') AND category = ?))", organization.ID, service.ContentTypeKnowledge, service.ContentStatusPublished, directory.ID, directory.Name).Count(&articleCount)
			items = append(items, gin.H{"id": directory.ID, "name": directory.Name, "slug": directory.Slug, "description": directory.Description, "article_count": articleCount, "updated_at": directory.UpdatedAt})
		}
		return items, nil
	})
}

func (h *WorkspaceHandler) knowledgeDirectoryNames(organizationID string, contents []model.Content) map[string]string {
	ids := make([]string, 0, len(contents))
	seen := make(map[string]struct{})
	for _, content := range contents {
		if content.KnowledgeDirectoryID == nil || *content.KnowledgeDirectoryID == "" {
			continue
		}
		if _, exists := seen[*content.KnowledgeDirectoryID]; exists {
			continue
		}
		seen[*content.KnowledgeDirectoryID] = struct{}{}
		ids = append(ids, *content.KnowledgeDirectoryID)
	}
	if len(ids) == 0 {
		return map[string]string{}
	}
	var directories []model.KnowledgeDirectory
	if h.db.Where("organization_id = ? AND is_public = ? AND id IN ?", organizationID, true, ids).Find(&directories).Error != nil {
		return map[string]string{}
	}
	names := make(map[string]string, len(directories))
	for _, directory := range directories {
		names[directory.ID] = directory.Name
	}
	return names
}
func (h *WorkspaceHandler) PortalServer(c *gin.Context) {
	status, err := h.serverAdapter.Status(c.Request.Context())
	if err != nil {
		respond(c, http.StatusOK, gin.H{"enabled": false, "label": "Minecraft 服务", "state": "offline", "version": nil, "online_players": nil, "max_players": nil, "updated_at": time.Now().UTC(), "apply_url": "#join"})
		return
	}
	respond(c, http.StatusOK, gin.H{"enabled": status.Enabled, "label": status.Label, "state": status.State, "version": status.Version, "online_players": status.OnlinePlayers, "max_players": status.MaxPlayers, "updated_at": status.UpdatedAt, "apply_url": "#join"})
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
		pendingItems = append(pendingItems, h.applicationAdminItem(item))
	}
	server := h.serverStatus(c.Request.Context())
	respond(c, http.StatusOK, gin.H{"organization_name": organization.Name, "updated_at": time.Now().UTC(), "metrics": []gin.H{{"label": "活跃成员", "value": activeMembers, "change": "当前组织成员", "tone": "primary"}, {"label": "已发布内容", "value": published, "change": "当前公开内容", "tone": "secondary"}, {"label": "内容总数", "value": total, "change": "含草稿", "tone": "neutral"}, {"label": "在线玩家", "value": server["online_players"], "change": "服务器适配器：" + h.serverAdapter.Mode(), "tone": "neutral"}}, "pending_applications": pendingItems, "recent_content": recentItems, "server": server})
}
func contentItems() []gin.H {
	return []gin.H{{"id": "content_001", "title": "QUTCraft CMS 项目正式启动", "type": "news", "status": "published", "author": "QUTCraft Admin", "updated_at": "2026-07-17T03:00:00Z"}}
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

func (h *WorkspaceHandler) AdminContentDetail(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var content model.Content
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&content).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			fail(c, http.StatusNotFound, "content.not_found", "内容不存在或不属于当前组织。")
			return
		}
		fail(c, http.StatusInternalServerError, "content.detail_failed", "内容暂时无法加载。")
		return
	}
	respond(c, http.StatusOK, contentAdminItem(content, h.db))
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

var knowledgeDirectorySlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func validKnowledgeDirectoryRequest(body knowledgeDirectoryRequest) bool {
	return strings.TrimSpace(body.Name) != "" && knowledgeDirectorySlugPattern.MatchString(strings.TrimSpace(body.Slug)) && len([]rune(body.Name)) <= 120 && len([]rune(body.Slug)) <= 120 && len([]rune(body.Description)) <= 500 && len([]rune(body.ParentID)) <= 64 && body.SortOrder >= 0
}

func (h *WorkspaceHandler) validateKnowledgeDirectoryParent(organizationID, directoryID, parentID string) error {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil
	}
	if directoryID != "" && directoryID == parentID {
		return errKnowledgeParentCycle
	}
	var parent model.KnowledgeDirectory
	if err := h.db.Where("id = ? AND organization_id = ?", parentID, organizationID).First(&parent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errKnowledgeParentInvalid
		}
		return err
	}
	visited := map[string]bool{parent.ID: true}
	for parent.ParentID != "" {
		if directoryID != "" && parent.ParentID == directoryID {
			return errKnowledgeParentCycle
		}
		if visited[parent.ParentID] {
			return errKnowledgeParentCycle
		}
		visited[parent.ParentID] = true
		var ancestor model.KnowledgeDirectory
		if err := h.db.Where("id = ? AND organization_id = ?", parent.ParentID, organizationID).First(&ancestor).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return errKnowledgeParentInvalid
			}
			return err
		}
		parent = ancestor
	}
	return nil
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
	directoryID := uuid.NewString()
	if err := h.validateKnowledgeDirectoryParent(principal.OrganizationID, directoryID, body.ParentID); err != nil {
		if errors.Is(err, errKnowledgeParentInvalid) || errors.Is(err, errKnowledgeParentCycle) {
			fail(c, http.StatusBadRequest, "knowledge_directory.parent_invalid", "父目录不存在或会造成目录循环。")
		} else {
			fail(c, http.StatusInternalServerError, "knowledge_directory.validation_failed", "知识库目录关系校验失败。")
		}
		return
	}
	directory := model.KnowledgeDirectory{ID: directoryID, OrganizationID: principal.OrganizationID, ParentID: strings.TrimSpace(body.ParentID), Name: strings.TrimSpace(body.Name), Slug: strings.TrimSpace(body.Slug), Description: strings.TrimSpace(body.Description), SortOrder: body.SortOrder, IsPublic: body.IsPublic}
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
	if err := h.validateKnowledgeDirectoryParent(principal.OrganizationID, directory.ID, body.ParentID); err != nil {
		if errors.Is(err, errKnowledgeParentInvalid) || errors.Is(err, errKnowledgeParentCycle) {
			fail(c, http.StatusBadRequest, "knowledge_directory.parent_invalid", "父目录不存在或会造成目录循环。")
		} else {
			fail(c, http.StatusInternalServerError, "knowledge_directory.validation_failed", "知识库目录关系校验失败。")
		}
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

func (h *WorkspaceHandler) resolveContentDirectory(organizationID, contentType, directoryID string) (*string, error) {
	directoryID = strings.TrimSpace(directoryID)
	if directoryID == "" {
		if contentType == service.ContentTypeKnowledge {
			return nil, errContentDirectoryRequired
		}
		return nil, nil
	}
	if contentType != service.ContentTypeKnowledge {
		return nil, errContentDirectoryTypeInvalid
	}
	var directory model.KnowledgeDirectory
	if err := h.db.Where("id = ? AND organization_id = ?", directoryID, organizationID).First(&directory).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errContentDirectoryNotFound
		}
		return nil, err
	}
	return &directory.ID, nil
}

func (h *WorkspaceHandler) respondContentDirectoryError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errContentDirectoryTypeInvalid) || errors.Is(err, errContentDirectoryRequired) || errors.Is(err, errContentDirectoryNotFound) {
		message := "知识库文章必须关联当前组织中存在的知识库目录。"
		if errors.Is(err, errContentDirectoryTypeInvalid) {
			message = "只有 knowledge 类型内容可以关联知识库目录。"
		}
		fail(c, http.StatusBadRequest, "content.knowledge_directory_invalid", message)
		return true
	}
	fail(c, http.StatusInternalServerError, "content.knowledge_directory_check_failed", "知识库目录关联校验失败。")
	return true
}

func (h *WorkspaceHandler) AdminCreateContent(c *gin.Context) {
	var body service.ContentInput
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "content.validation_failed", "内容标题不能为空。")
		return
	}
	normalized, err := service.NormalizeContentInput(body)
	if err != nil {
		fail(c, http.StatusBadRequest, "content.validation_failed", "内容字段不符合规范。")
		return
	}
	principal, _ := middleware.PrincipalFromContext(c)
	directoryID, directoryErr := h.resolveContentDirectory(principal.OrganizationID, normalized.Type, normalized.KnowledgeDirectoryID)
	if h.respondContentDirectoryError(c, directoryErr) {
		return
	}
	content := model.Content{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, AuthorUserID: principal.UserID, Title: normalized.Title, Type: normalized.Type, Category: normalized.Category, KnowledgeDirectoryID: directoryID, Status: service.ContentStatusDraft, Excerpt: normalized.Excerpt, Body: normalized.Body}
	if err := h.db.Create(&content).Error; err != nil {
		fail(c, http.StatusInternalServerError, "content.create_failed", "内容草稿创建失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusCreated, contentAdminItem(content, h.db))
}

func (h *WorkspaceHandler) AdminUpdateContent(c *gin.Context) {
	var body service.ContentInput
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "content.validation_failed", "内容格式不正确。")
		return
	}
	normalized, err := service.NormalizeContentInput(body)
	if err != nil {
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
	directoryID, directoryErr := h.resolveContentDirectory(principal.OrganizationID, normalized.Type, normalized.KnowledgeDirectoryID)
	if h.respondContentDirectoryError(c, directoryErr) {
		return
	}
	content.Title, content.Type, content.Category, content.KnowledgeDirectoryID, content.Excerpt, content.Body = normalized.Title, normalized.Type, normalized.Category, directoryID, normalized.Excerpt, normalized.Body
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&content).Error; err != nil {
			return err
		}
		return bindMarkdownAssets(tx, principal.OrganizationID, content.ID, content.Body)
	}); err != nil {
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
	if status == service.ContentStatusPublished && strings.TrimSpace(content.Title) == "" {
		fail(c, http.StatusBadRequest, "content.not_publishable", "内容标题不能为空。")
		return
	}
	if content.Status == status {
		fail(c, http.StatusConflict, "content.already_in_state", "内容已经处于目标状态。")
		return
	}
	if !service.CanTransitionContentStatus(content.Status, status) {
		fail(c, http.StatusConflict, "content.invalid_transition", "内容不能从当前状态转换到目标状态。")
		return
	}
	content.Status = status
	if status == service.ContentStatusPublished {
		now := time.Now().UTC()
		content.PublishedAt = &now
	} else {
		content.PublishedAt = nil
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if status == service.ContentStatusPublished {
			if err := bindMarkdownAssets(tx, principal.OrganizationID, content.ID, content.Body); err != nil {
				return err
			}
		}
		return tx.Save(&content).Error
	}); err != nil {
		fail(c, http.StatusInternalServerError, "content.status_update_failed", "内容状态更新失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "content." + status, TargetType: "content", TargetID: content.ID, Result: "success", RequestID: ensureRequestID(c)}).Error
	respond(c, http.StatusOK, contentAdminItem(content, h.db))
}

func bindMarkdownAssets(db *gorm.DB, organizationID, contentID, body string) error {
	matches := markdownAdminAssetPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	assetIDs := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		if _, exists := seen[match[1]]; exists {
			continue
		}
		seen[match[1]] = struct{}{}
		assetIDs = append(assetIDs, match[1])
	}
	if len(assetIDs) == 0 {
		return nil
	}

	var assets []model.MediaAsset
	if err := db.Where("organization_id = ? AND id IN ?", organizationID, assetIDs).Find(&assets).Error; err != nil {
		return err
	}
	for _, asset := range assets {
		if asset.ContentID != "" && asset.ContentID != contentID {
			continue
		}
		if asset.ContentID == contentID {
			continue
		}
		if err := db.Model(&model.MediaAsset{}).Where("id = ? AND organization_id = ?", asset.ID, organizationID).Update("content_id", contentID).Error; err != nil {
			return err
		}
	}
	return nil
}

func canTransitionContentStatus(current, target string) bool {
	return service.CanTransitionContentStatus(current, target)
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
	category := content.Category
	if content.Type == service.ContentTypeKnowledge && content.KnowledgeDirectoryID != nil {
		var directory model.KnowledgeDirectory
		if h.db != nil && h.db.Where("id = ? AND organization_id = ? AND is_public = ?", *content.KnowledgeDirectoryID, content.OrganizationID, true).First(&directory).Error == nil {
			category = directory.Name
		}
	}
	item := gin.H{
		"id":              content.ID,
		"title":           content.Title,
		"type":            content.Type,
		"category":        category,
		"excerpt":         content.Excerpt,
		"body":            publicContentBody(slug, content.Body),
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

func publicContentBody(slug, body string) string {
	publicPrefix := "/api/v1/portal/organizations/" + slug + "/assets/"
	return markdownAdminAssetPattern.ReplaceAllString(body, publicPrefix+"$1/download")
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
	var directoryID interface{} = nil
	if content.KnowledgeDirectoryID != nil && *content.KnowledgeDirectoryID != "" {
		directoryID = *content.KnowledgeDirectoryID
	}
	return gin.H{"id": content.ID, "title": content.Title, "type": content.Type, "category": content.Category, "knowledge_directory_id": directoryID, "status": content.Status, "author": author.DisplayName, "excerpt": content.Excerpt, "body": content.Body, "published_at": content.PublishedAt, "updated_at": content.UpdatedAt}
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
	if err := c.ShouldBindJSON(&body); err != nil || !validMemberWriteState(body.State) || !validRole(body.Role) {
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
	if user.State != "active" {
		fail(c, http.StatusConflict, "user.account_inactive", "该账户已被系统级停用，组织管理员不能在此恢复。")
		return
	}
	targetRole := membershipRole(h.db, membership.ID)
	currentState := membership.State
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
		if err := tx.Where("membership_id = ?", membership.ID).Delete(&model.MembershipRole{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.MembershipRole{MembershipID: membership.ID, RoleID: role.ID}).Error; err != nil {
			return err
		}
		if body.State != "active" {
			now := time.Now().UTC()
			if err := tx.Model(&model.RefreshToken{}).
				Where("user_id = ? AND revoked_at IS NULL", user.ID).
				Update("revoked_at", now).Error; err != nil {
				return err
			}
		}
		return tx.Create(&model.MembershipEvent{
			ID:           uuid.NewString(),
			MembershipID: membership.ID,
			State:        body.State,
			Reason:       membershipUpdateReason(currentState, targetRole, body.State, body.Role),
		}).Error
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "membership.update_failed", "成员信息保存失败。")
		return
	}
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "membership.update", TargetType: "membership", TargetID: membership.ID, Result: "success", RequestID: ensureRequestID(c)}).Error
	respond(c, http.StatusOK, gin.H{"id": user.ID, "name": user.DisplayName, "email": user.Email, "role": body.Role, "state": body.State, "joined_at": membership.CreatedAt})
}

func validMemberWriteState(value string) bool {
	return value == "active" || value == "disabled"
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

func membershipUpdateReason(currentState, currentRole, nextState, nextRole string) string {
	switch {
	case currentState == "active" && nextState == "disabled":
		return "admin_disabled"
	case currentState != "active" && nextState == "active":
		return "admin_reactivated"
	case currentRole != nextRole:
		return "admin_role_changed"
	default:
		return "admin_update"
	}
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

type applicationDecisionRequest struct {
	Reason string `json:"reason"`
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
	page, pageSize, ok := listMeta(c, 0)
	if !ok {
		return
	}

	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != "pending" && status != "approved" && status != "rejected" {
		fail(c, http.StatusBadRequest, "application.invalid_status_filter", "status 仅支持 pending、approved 或 rejected。")
		return
	}
	applicationType := strings.TrimSpace(c.Query("type"))
	if applicationType != "" && applicationType != "whitelist" && applicationType != "membership" {
		fail(c, http.StatusBadRequest, "application.invalid_type_filter", "type 仅支持 whitelist 或 membership。")
		return
	}
	syncStatus := strings.TrimSpace(c.Query("server_sync_status"))
	if syncStatus != "" && syncStatus != "none" && syncStatus != "pending" && syncStatus != "succeeded" && syncStatus != "failed" {
		fail(c, http.StatusBadRequest, "application.invalid_server_sync_status_filter", "server_sync_status 仅支持 none、pending、succeeded 或 failed。")
		return
	}
	search, ok := queryMax(c, "query", 80)
	if !ok {
		return
	}

	query := h.db.Model(&model.Application{}).Where("applications.organization_id = ?", principal.OrganizationID)
	if status != "" {
		query = query.Where("applications.status = ?", status)
	}
	if applicationType != "" {
		query = query.Where("applications.type = ?", applicationType)
	}
	if search != "" {
		term := "%" + search + "%"
		query = query.Where(
			"(applications.applicant_name LIKE ? OR applications.game_id LIKE ? OR applications.email LIKE ? OR applications.qq_number LIKE ?)",
			term, term, term, term,
		)
	}
	if syncStatus == "none" {
		query = query.Where("NOT EXISTS (SELECT 1 FROM application_server_syncs syncs WHERE syncs.application_id = applications.id AND syncs.organization_id = applications.organization_id)")
	} else if syncStatus != "" {
		query = query.Where(`EXISTS (
			SELECT 1 FROM application_server_syncs syncs
			WHERE syncs.application_id = applications.id
			  AND syncs.organization_id = applications.organization_id
			  AND syncs.status = ?
			  AND syncs.created_at = (
				SELECT MAX(latest_sync.created_at)
				FROM application_server_syncs latest_sync
				WHERE latest_sync.application_id = applications.id
				  AND latest_sync.organization_id = applications.organization_id
			  )
		)`, syncStatus)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "application.list_failed", "申请列表暂时无法加载。")
		return
	}
	var applications []model.Application
	if err := query.Order("applications.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&applications).Error; err != nil {
		fail(c, http.StatusInternalServerError, "application.list_failed", "申请列表暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(applications))
	for _, application := range applications {
		items = append(items, h.applicationAdminItem(application))
	}
	respondWithMeta(c, http.StatusOK, items, gin.H{"page": page, "page_size": pageSize, "total": total})
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
	var body applicationDecisionRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			fail(c, http.StatusBadRequest, "application.decision_validation_failed", "审核原因数据格式不正确。")
			return
		}
	}
	application, _, err := h.applications.Decide(c.Request.Context(), principal.OrganizationID, principal.UserID, decision, next, body.Reason, ensureRequestID(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrApplicationNotFound):
			fail(c, http.StatusNotFound, "application.not_found", "申请不存在。")
		case errors.Is(err, service.ErrApplicationAlreadyDecided):
			fail(c, http.StatusConflict, "application.already_decided", "申请已经处理，不能重复审批。")
		case errors.Is(err, service.ErrApplicationReasonRequired):
			fail(c, http.StatusBadRequest, "application.decision_reason_required", "拒绝申请时必须填写审核原因。")
		case errors.Is(err, service.ErrApplicationReasonTooLong):
			fail(c, http.StatusBadRequest, "application.decision_reason_too_long", "审核原因不能超过 500 个字符。")
		default:
			fail(c, http.StatusInternalServerError, "application.decision_failed", "申请状态暂时无法更新。")
		}
		return
	}
	respond(c, http.StatusOK, h.applicationAdminItem(application))
}

func (h *WorkspaceHandler) AdminRetryApplicationServerSync(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	record, err := h.applications.RetryServerSync(c.Request.Context(), principal.OrganizationID, principal.UserID, c.Param("id"), ensureRequestID(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrApplicationNotFound), errors.Is(err, service.ErrApplicationSyncNotFound):
			fail(c, http.StatusNotFound, "application.server_sync_not_found", "申请或服务器同步记录不存在。")
		case errors.Is(err, service.ErrApplicationSyncNotRetryable):
			fail(c, http.StatusConflict, "application.server_sync_not_retryable", "当前服务器同步状态不能重试。")
		default:
			fail(c, http.StatusInternalServerError, "application.server_sync_retry_failed", "服务器同步暂时无法重试。")
		}
		return
	}
	respond(c, http.StatusOK, applicationServerSyncItem(record))
}

func (h *WorkspaceHandler) applicationAdminItem(application model.Application) gin.H {
	item := gin.H{"id": application.ID, "applicant": application.ApplicantName, "type": application.Type, "submitted_at": application.CreatedAt, "note": application.Note, "status": application.Status, "class_name": application.ClassName, "game_id": application.GameID, "qq_number": application.QQNumber, "email": application.Email, "decided_at": application.DecidedAt, "decided_by": application.DecidedBy, "decision_reason": application.DecisionReason}
	var syncRecord model.ApplicationServerSync
	if err := h.db.Where("application_id = ?", application.ID).Order("created_at DESC").First(&syncRecord).Error; err == nil {
		item["server_sync"] = applicationServerSyncItem(syncRecord)
	} else {
		item["server_sync"] = nil
	}
	return item
}

func applicationServerSyncItem(record model.ApplicationServerSync) gin.H {
	return gin.H{"id": record.ID, "operation": record.Operation, "adapter": record.Adapter, "mode": record.Mode, "status": record.Status, "attempts": record.Attempts, "message": record.Message, "last_error": record.LastError, "requested_at": record.RequestedAt, "completed_at": record.CompletedAt}
}

func (h *WorkspaceHandler) AdminServerStatus(c *gin.Context) {
	respond(c, http.StatusOK, h.serverStatus(c.Request.Context()))
}

func (h *WorkspaceHandler) serverStatus(ctx context.Context) gin.H {
	status, err := h.serverAdapter.Status(ctx)
	if err != nil {
		return gin.H{"enabled": false, "adapter": h.serverAdapter.Name(), "mode": h.serverAdapter.Mode(), "label": "Minecraft 服务", "state": "offline", "online_players": 0, "max_players": 1, "updated_at": time.Now().UTC(), "last_error": "服务器适配器暂时不可用。"}
	}
	return gin.H{"enabled": status.Enabled, "adapter": status.Adapter, "mode": status.Mode, "label": status.Label, "state": status.State, "online_players": status.OnlinePlayers, "max_players": status.MaxPlayers, "updated_at": status.UpdatedAt, "last_command_at": status.LastCommandAt}
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
	if len([]rune(body.Command)) > 256 || strings.ContainsAny(body.Command, "\r\n") || !serveradapter.AllowedCommand(body.Command) {
		fail(c, http.StatusForbidden, "server.command_not_allowed", "命令不在服务端白名单中。")
		return
	}
	principal, _ := middleware.PrincipalFromContext(c)
	result, err := h.serverAdapter.Execute(c.Request.Context(), body.Command)
	auditResult := "accepted"
	if err != nil {
		auditResult = "failed"
	}
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "server.command", TargetType: "server", Result: auditResult, RequestID: ensureRequestID(c), CreatedAt: time.Now().UTC()}).Error
	if err != nil {
		fail(c, http.StatusBadGateway, "server.adapter_failed", "服务器适配器执行失败。")
		return
	}
	respond(c, http.StatusOK, result)
}

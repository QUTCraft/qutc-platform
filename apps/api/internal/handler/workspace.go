// workspace.go 是工作区核心 HTTP 处理器，覆盖公开门户、内容管理、知识目录、项目、成员、申请和资产关联。
// 该文件中的处理器负责组织隔离、分页参数、权限前置检查、事务编排和响应 DTO 转换，领域规则通过 service/model 协作完成。
package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/cache"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/storage"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/superbed"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WorkspaceHandler owns the first content read/write model shared by the
// public portal and the protected CMS workspace.
// WorkspaceHandler 保存工作区 API 使用的数据库、缓存、环境和可选外部依赖。
type WorkspaceHandler struct {
	db              *gorm.DB
	cache           *cache.Cache
	cacheNamespace  string
	mediaStorage    storage.Store
	storageResolver MediaStorageResolver
	applications    *service.ApplicationDecisionService
	notifications   *service.NotificationService
	superbed        *superbed.Uploader
}

// MediaStorageResolver 按组织和存储驱动解析媒体存储实例，用于支持运行时切换 local/S3。
type MediaStorageResolver interface {
	Storage(context.Context, string, string) (storage.Store, error)
}

var (
	errContentDirectoryTypeInvalid = errors.New("knowledge directory can only be used by knowledge content")
	errContentDirectoryRequired    = errors.New("knowledge content requires a directory")
	errContentDirectoryNotFound    = errors.New("knowledge directory does not belong to the organization")
	errKnowledgeParentInvalid      = errors.New("knowledge directory parent is invalid")
	errKnowledgeParentCycle        = errors.New("knowledge directory parent creates a cycle")
)

var markdownAdminAssetPattern = regexp.MustCompile(`/api/v1/admin/assets/([a-zA-Z0-9-]+)/download`)

// NewWorkspaceHandler 创建使用默认存储实现的工作区处理器。
func NewWorkspaceHandler(db *gorm.DB, publicCache *cache.Cache, environment string) *WorkspaceHandler {
	mediaStorage, err := storage.NewLocal("/tmp/qutcraft-uploads")
	if err != nil {
		panic(err)
	}
	return NewWorkspaceHandlerWithDependencies(db, publicCache, environment, mediaStorage)
}

// NewWorkspaceHandlerWithDependencies 创建并注入自定义媒体存储，主要用于应用组装和测试。
func NewWorkspaceHandlerWithDependencies(db *gorm.DB, publicCache *cache.Cache, environment string, mediaStorage storage.Store) *WorkspaceHandler {
	return NewWorkspaceHandlerWithDependenciesAndNotifications(db, publicCache, environment, mediaStorage, nil)
}

// NewWorkspaceHandlerWithDependenciesAndNotifications 在工作区处理器中同时注入媒体存储和通知服务。
func NewWorkspaceHandlerWithDependenciesAndNotifications(db *gorm.DB, publicCache *cache.Cache, environment string, mediaStorage storage.Store, notifications *service.NotificationService) *WorkspaceHandler {
	if strings.TrimSpace(environment) == "" {
		environment = "development"
	}
	if mediaStorage == nil {
		var err error
		mediaStorage, err = storage.NewLocal("/tmp/qutcraft-uploads")
		if err != nil {
			panic(err)
		}
	}
	return &WorkspaceHandler{
		db:             db,
		cache:          publicCache,
		cacheNamespace: environment,
		mediaStorage:   mediaStorage,
		applications:   service.NewApplicationDecisionServiceWithNotifications(db, notifications),
		notifications:  notifications,
	}
}

// UseStorageResolver enables organization-scoped runtime storage selection
// while preserving the existing static constructor for tests and embedders.
// UseStorageResolver 设置按组织解析存储的策略；传入 nil 时回退到处理器默认存储。
func (h *WorkspaceHandler) UseStorageResolver(resolver MediaStorageResolver) {
	h.storageResolver = resolver
}

// storageFor 根据组织和驱动名获取存储实例，并统一处理 resolver 未配置的回退路径。
func (h *WorkspaceHandler) storageFor(ctx context.Context, organizationID, driver string) (storage.Store, error) {
	if h.storageResolver != nil {
		return h.storageResolver.Storage(ctx, organizationID, driver)
	}
	if h.mediaStorage == nil {
		return nil, errors.New("media storage is unavailable")
	}
	if driver != "" && driver != h.mediaStorage.Driver() {
		return nil, fmt.Errorf("storage driver %q is unavailable", driver)
	}
	return h.mediaStorage, nil
}

// portalSeesMembersOnly 判断当前请求是否属于该组织的已登录成员，从而可阅读仅成员可见的已发布内容。
func (h *WorkspaceHandler) portalSeesMembersOnly(c *gin.Context, organizationID string) bool {
	principal, ok := middleware.PrincipalFromContext(c)
	return ok && principal.OrganizationID == organizationID
}

func (h *WorkspaceHandler) portalAudience(c *gin.Context, organizationID string) string {
	if h.portalSeesMembersOnly(c, organizationID) {
		return "member"
	}
	return "anon"
}

func (h *WorkspaceHandler) restrictPublishedContent(query *gorm.DB, tablePrefix string, includeMembersOnly bool) *gorm.DB {
	statusColumn, publicColumn := "status", "is_public"
	if tablePrefix != "" {
		statusColumn = tablePrefix + ".status"
		publicColumn = tablePrefix + ".is_public"
	}
	query = query.Where(statusColumn+" = ?", service.ContentStatusPublished)
	if !includeMembersOnly {
		query = query.Where(publicColumn+" = ?", true)
	}
	return query
}

// cachedPortalPage 读取或写入公开门户列表缓存，缓存失败不影响数据库查询结果返回。
func (h *WorkspaceHandler) cachedPortalPage(c *gin.Context, slug, resource, audience string, loader func() ([]gin.H, error)) {
	if audience == "" {
		audience = "anon"
	}
	key := "qutc:" + h.cacheNamespace + ":portal:" + slug + ":" + resource + ":" + audience + ":" + cache.NormalizeQuery(c.Request.URL.RawQuery)
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

// cachedPortalItem 读取或写入公开门户单项缓存，并保持与列表缓存相同的容错策略。
func (h *WorkspaceHandler) cachedPortalItem(c *gin.Context, slug, resource, audience string, loader func() (gin.H, error)) {
	if audience == "" {
		audience = "anon"
	}
	key := "qutc:" + h.cacheNamespace + ":portal:" + slug + ":" + resource + ":" + audience
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

// invalidatePortalCache 清除组织门户相关缓存，供所有会改变公开数据的写操作调用。
func (h *WorkspaceHandler) invalidatePortalCache(organizationID string) {
	if h.cache == nil {
		return
	}
	var organization model.Organization
	if h.db.First(&organization, "id = ?", organizationID).Error == nil {
		h.cache.DeletePrefix(context.Background(), "qutc:"+h.cacheNamespace+":portal:"+organization.Slug+":")
	}
}

// listMeta 解析分页参数并计算总页数；参数非法时直接写入 400 响应。
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

// pageOf 按请求页码裁剪内存切片，并以统一 meta 响应输出分页结果。
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

// queryMax 读取并限制查询参数长度，避免超长过滤条件进入数据库查询。
func queryMax(c *gin.Context, key string, max int) (string, bool) {
	value := strings.TrimSpace(c.Query(key))
	if len([]rune(value)) > max {
		fail(c, http.StatusBadRequest, "query.value_too_long", key+" 超出长度限制。")
		return "", false
	}
	return value, true
}

// Organization 返回公开门户使用的组织基本资料。
func (h *WorkspaceHandler) Organization(c *gin.Context) {
	var org model.Organization
	if err := h.db.Where("slug = ? AND is_public = ?", c.Param("slug"), true).First(&org).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	respond(c, http.StatusOK, organizationProfileItem(org))
}

// PortalContentDetail 返回公开内容详情，并重写正文中的资产地址。
func (h *WorkspaceHandler) PortalContentDetail(c *gin.Context) {
	var organization model.Organization
	if err := h.db.Where("slug = ? AND is_public = ?", c.Param("slug"), true).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	contentID := c.Param("id")
	h.cachedPortalItem(c, c.Param("slug"), "content:"+contentID, h.portalAudience(c, organization.ID), func() (gin.H, error) {
		var content model.Content
		query := h.db.Where("id = ? AND organization_id = ?", contentID, organization.ID)
		query = h.restrictPublishedContent(query, "", h.portalSeesMembersOnly(c, organization.ID))
		if err := query.Where("(type <> ? OR knowledge_directory_id IS NULL OR knowledge_directory_id = '' OR EXISTS (SELECT 1 FROM knowledge_directories AS directory WHERE directory.id = contents.knowledge_directory_id AND directory.organization_id = contents.organization_id AND directory.is_public = ?))", service.ContentTypeKnowledge, true).First(&content).Error; err != nil {
			return nil, gorm.ErrRecordNotFound
		}
		return h.contentPublicDetailItem(c.Param("slug"), content), nil
	})
}

// PortalPosts 分页返回组织已发布的文章内容。
func (h *WorkspaceHandler) PortalPosts(c *gin.Context) {
	category, ok := queryMax(c, "category", 64)
	if !ok {
		return
	}
	var organization model.Organization
	if err := h.db.Where("slug = ? AND is_public = ?", c.Param("slug"), true).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	query := h.db.Where("organization_id = ? AND type = ?", organization.ID, "news").Order("published_at DESC")
	query = h.restrictPublishedContent(query, "", h.portalSeesMembersOnly(c, organization.ID))
	if category != "" {
		query = query.Where("category = ?", category)
	}
	h.cachedPortalPage(c, c.Param("slug"), "posts", h.portalAudience(c, organization.ID), func() ([]gin.H, error) {
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

// PortalProjects 分页返回组织公开项目及其里程碑摘要。
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
	if err := h.db.Where("slug = ? AND is_public = ?", c.Param("slug"), true).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	h.cachedPortalPage(c, c.Param("slug"), "projects", "anon", func() ([]gin.H, error) {
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

// PortalResources 分页返回组织公开资源内容。
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
	if err := h.db.Where("slug = ? AND is_public = ?", c.Param("slug"), true).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	query := h.db.Where("organization_id = ? AND type = ?", organization.ID, "resource").Order("updated_at DESC")
	query = h.restrictPublishedContent(query, "", h.portalSeesMembersOnly(c, organization.ID))
	if q != "" {
		query = query.Where("title LIKE ? OR excerpt LIKE ? OR body LIKE ?", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	h.cachedPortalPage(c, c.Param("slug"), "resources", h.portalAudience(c, organization.ID), func() ([]gin.H, error) {
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

// PortalKnowledge 返回公开知识条目，并按目录信息补充展示字段。
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
	if err := h.db.Where("slug = ? AND is_public = ?", c.Param("slug"), true).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	query := h.db.Table("contents AS content").Select("content.*").Joins("LEFT JOIN knowledge_directories AS directory ON directory.id = content.knowledge_directory_id AND directory.organization_id = content.organization_id").Where("content.organization_id = ? AND content.type = ?", organization.ID, service.ContentTypeKnowledge).Where("(content.knowledge_directory_id IS NULL OR content.knowledge_directory_id = '' OR directory.is_public = ?)", true).Order("content.updated_at DESC")
	query = h.restrictPublishedContent(query, "content", h.portalSeesMembersOnly(c, organization.ID))
	if category != "" {
		query = query.Where("(content.category = ? OR directory.name = ? OR directory.slug = ? OR content.title LIKE ? OR content.excerpt LIKE ?)", category, category, category, "%"+category+"%", "%"+category+"%")
	}
	if q != "" {
		query = query.Where("(content.title LIKE ? OR content.excerpt LIKE ? OR content.body LIKE ?)", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	h.cachedPortalPage(c, c.Param("slug"), "knowledge", h.portalAudience(c, organization.ID), func() ([]gin.H, error) {
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
			items = append(items, gin.H{"id": content.ID, "title": content.Title, "summary": content.Excerpt, "category": categoryName, "updated_at": content.UpdatedAt, "reading_minutes": maxInt(1, len([]rune(content.Body))/900+1), "members_only": !content.IsPublic})
		}
		return items, nil
	})
}

// PortalKnowledgeDirectories 返回公开知识目录树及目录下的内容计数。
func (h *WorkspaceHandler) PortalKnowledgeDirectories(c *gin.Context) {
	var organization model.Organization
	if err := h.db.Where("slug = ? AND is_public = ?", c.Param("slug"), true).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	h.cachedPortalPage(c, c.Param("slug"), "knowledge-directories", h.portalAudience(c, organization.ID), func() ([]gin.H, error) {
		var directories []model.KnowledgeDirectory
		if err := h.db.Where("organization_id = ? AND is_public = ?", organization.ID, true).Order("sort_order ASC, name ASC").Find(&directories).Error; err != nil {
			return nil, err
		}
		items := make([]gin.H, 0, len(directories))
		for _, directory := range directories {
			var articleCount int64
			countQuery := h.db.Model(&model.Content{}).Where("organization_id = ? AND type = ? AND (knowledge_directory_id = ? OR ((knowledge_directory_id IS NULL OR knowledge_directory_id = '') AND category = ?))", organization.ID, service.ContentTypeKnowledge, directory.ID, directory.Name)
			countQuery = h.restrictPublishedContent(countQuery, "", h.portalSeesMembersOnly(c, organization.ID))
			countQuery.Count(&articleCount)
			items = append(items, gin.H{"id": directory.ID, "name": directory.Name, "slug": directory.Slug, "description": directory.Description, "article_count": articleCount, "updated_at": directory.UpdatedAt})
		}
		return items, nil
	})
}

// knowledgeDirectoryNames 批量读取内容引用的目录名称，减少公开列表中的逐条查询。
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

// AdminDashboard 汇总当前组织的内容、项目、成员和申请数量，供管理端首页展示。
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
	canReadApplications, _ := principalHasPermission(h.db, principal, "application:read")
	canReadContent, _ := principalHasPermission(h.db, principal, "content:read")
	canReadProjects, _ := principalHasPermission(h.db, principal, "project:read")
	canReadMembers, _ := principalHasPermission(h.db, principal, "membership:read")

	var published, total, activeMembers, activeProjects int64
	h.db.Model(&model.Content{}).Where("organization_id = ? AND status = ? AND is_public = ?", principal.OrganizationID, "published", true).Count(&published)
	if canReadContent {
		contentTotalQuery := h.db.Model(&model.Content{}).Where("organization_id = ?", principal.OrganizationID)
		contentTotalQuery, _ = scopeAdminContentQuery(h.db, contentTotalQuery, principal)
		contentTotalQuery.Count(&total)
	} else {
		total = published
	}
	if canReadMembers {
		h.db.Model(&model.Membership{}).Where("organization_id = ? AND state = ?", principal.OrganizationID, "active").Count(&activeMembers)
	}
	if canReadProjects {
		h.db.Model(&model.Project{}).Where("organization_id = ? AND status = ?", principal.OrganizationID, "active").Count(&activeProjects)
	}
	recentItems := make([]gin.H, 0)
	if canReadContent {
		recentQuery := h.db.Where("organization_id = ?", principal.OrganizationID)
		recentQuery, _ = scopeAdminContentQuery(h.db, recentQuery, principal)
		var recent []model.Content
		recentQuery.Order("updated_at DESC").Limit(12).Find(&recent)
		for _, item := range recent {
			recentItems = append(recentItems, h.contentAdminItem(item, principal))
		}
	}
	pendingItems := make([]gin.H, 0)
	if canReadApplications {
		var pendingApplications []model.Application
		h.db.Where("organization_id = ? AND status = ?", principal.OrganizationID, "pending").Order("created_at DESC").Limit(12).Find(&pendingApplications)
		for _, item := range pendingApplications {
			pendingItems = append(pendingItems, h.applicationAdminItem(item))
		}
	}
	lastMetric := gin.H{"label": "进行中项目", "value": activeProjects, "change": "当前组织项目", "tone": "neutral"}
	respond(c, http.StatusOK, gin.H{"organization_name": organization.Name, "updated_at": time.Now().UTC(), "metrics": []gin.H{{"label": "活跃成员", "value": activeMembers, "change": "当前组织成员", "tone": "primary"}, {"label": "已发布内容", "value": published, "change": "当前公开内容", "tone": "secondary"}, {"label": "内容总数", "value": total, "change": "含草稿", "tone": "neutral"}, lastMetric}, "pending_applications": pendingItems, "recent_content": recentItems})
}

// contentItems 返回管理端内容类型筛选器使用的固定选项。
func contentItems() []gin.H {
	return []gin.H{{"id": "content_001", "title": "QUTCraft CMS 项目正式启动", "type": "news", "status": "published", "author": "QUTCraft Admin", "updated_at": "2026-07-17T03:00:00Z"}}
}

// AdminContent 按类型、状态和关键词分页列出组织内容。
func (h *WorkspaceHandler) AdminContent(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	status, ok := queryMax(c, "status", 24)
	if !ok {
		return
	}
	if status != "" && !service.IsContentStatus(status) {
		fail(c, http.StatusBadRequest, "query.invalid_status", "status 不是受支持的内容状态。")
		return
	}
	query := h.db.Where("organization_id = ?", principal.OrganizationID)
	query, err := scopeAdminContentQuery(h.db, query, principal)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content.list_failed", "内容列表暂时无法加载。")
		return
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var contents []model.Content
	if err := query.Order("updated_at DESC").Find(&contents).Error; err != nil {
		fail(c, http.StatusInternalServerError, "content.list_failed", "内容列表暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(contents))
	for _, item := range contents {
		items = append(items, h.contentAdminItem(item, principal))
	}
	pageOf(c, items)
}

// AdminContentDetail 返回管理端内容详情及当前用户可见的内部字段。
func (h *WorkspaceHandler) AdminContentDetail(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	content, ok := h.loadVisibleAdminContent(c, principal, c.Param("id"))
	if !ok {
		return
	}
	respond(c, http.StatusOK, h.contentAdminItem(content, principal))
}

// loadVisibleAdminContent 读取当前组织中的内容，并对无权查看的未发布/待审核条目返回 404。
func (h *WorkspaceHandler) loadVisibleAdminContent(c *gin.Context, principal service.Principal, contentID string) (model.Content, bool) {
	var content model.Content
	if err := h.db.Where("id = ? AND organization_id = ?", contentID, principal.OrganizationID).First(&content).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "content.not_found", "内容不存在或不属于当前组织。")
			return content, false
		}
		fail(c, http.StatusInternalServerError, "content.detail_failed", "内容暂时无法加载。")
		return content, false
	}
	allowed, err := principalCanViewAdminContent(h.db, principal, content)
	if err != nil {
		fail(c, http.StatusInternalServerError, "content.permission_check_failed", "内容权限校验失败。")
		return content, false
	}
	if !allowed {
		fail(c, http.StatusNotFound, "content.not_found", "内容不存在或不属于当前组织。")
		return content, false
	}
	return content, true
}

// AdminKnowledgeDirectories 列出管理端可维护的完整知识目录。
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

// knowledgeDirectoryRequest 描述知识目录名称、slug 和父目录关系。
type knowledgeDirectoryRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	ParentID    string `json:"parent_id"`
	SortOrder   int    `json:"sort_order"`
	IsPublic    bool   `json:"is_public"`
}

var knowledgeDirectorySlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// validKnowledgeDirectoryRequest 验证目录字段长度和 slug 格式。
func validKnowledgeDirectoryRequest(body knowledgeDirectoryRequest) bool {
	return strings.TrimSpace(body.Name) != "" && knowledgeDirectorySlugPattern.MatchString(strings.TrimSpace(body.Slug)) && len([]rune(body.Name)) <= 120 && len([]rune(body.Slug)) <= 120 && len([]rune(body.Description)) <= 500 && len([]rune(body.ParentID)) <= 64 && body.SortOrder >= 0
}

// validateKnowledgeDirectoryParent 验证父目录存在、同组织且不会形成循环引用。
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

// AdminCreateKnowledgeDirectory 创建知识目录并写入审计记录。
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
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&directory).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "knowledge_directory.create", "knowledge_directory", directory.ID)
	}); err != nil {
		fail(c, http.StatusConflict, "knowledge_directory.slug_in_use", "知识库目录标识已存在。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusCreated, knowledgeDirectoryItem(directory))
}

// AdminUpdateKnowledgeDirectory 更新知识目录并重新校验父目录关系。
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
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&directory).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "knowledge_directory.update", "knowledge_directory", directory.ID)
	}); err != nil {
		fail(c, http.StatusConflict, "knowledge_directory.slug_in_use", "知识库目录标识已存在。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, knowledgeDirectoryItem(directory))
}

func (h *WorkspaceHandler) AdminDeleteKnowledgeDirectory(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var directory model.KnowledgeDirectory
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&directory).Error; err != nil {
		fail(c, http.StatusNotFound, "knowledge_directory.not_found", "知识库目录不存在。")
		return
	}
	var childCount int64
	if err := h.db.Model(&model.KnowledgeDirectory{}).
		Where("organization_id = ? AND parent_id = ?", principal.OrganizationID, directory.ID).
		Count(&childCount).Error; err != nil {
		fail(c, http.StatusInternalServerError, "knowledge_directory.delete_failed", "知识库目录暂时无法删除。")
		return
	}
	if childCount > 0 {
		fail(c, http.StatusConflict, "knowledge_directory.has_children", "该目录下仍有子目录，请先删除或移动子目录。")
		return
	}
	var articleCount int64
	if err := h.db.Model(&model.Content{}).
		Where("organization_id = ? AND knowledge_directory_id = ?", principal.OrganizationID, directory.ID).
		Count(&articleCount).Error; err != nil {
		fail(c, http.StatusInternalServerError, "knowledge_directory.delete_failed", "知识库目录暂时无法删除。")
		return
	}
	if articleCount > 0 {
		fail(c, http.StatusConflict, "knowledge_directory.has_content", "该目录下仍有知识文章，请先删除或移动文章。")
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&directory).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "knowledge_directory.delete", "knowledge_directory", directory.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "knowledge_directory.delete_failed", "知识库目录删除失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, gin.H{"removed": true, "id": directory.ID})
}

// knowledgeDirectoryItem 将知识目录模型转换为管理端和门户共用的 DTO。
func knowledgeDirectoryItem(directory model.KnowledgeDirectory) gin.H {
	return gin.H{"id": directory.ID, "parent_id": directory.ParentID, "name": directory.Name, "slug": directory.Slug, "description": directory.Description, "sort_order": directory.SortOrder, "is_public": directory.IsPublic, "updated_at": directory.UpdatedAt}
}

// resolveContentDirectory 校验内容类型对应的目录 ID，并返回可写入内容记录的规范值。
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

// respondContentDirectoryError 将目录不存在、类型不匹配等错误映射为 HTTP 响应，并返回是否已处理。
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

// AdminCreateContent 创建草稿内容，绑定目录并记录初始版本。
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
	content := model.Content{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, AuthorUserID: principal.UserID, Title: normalized.Title, Type: normalized.Type, Category: normalized.Category, KnowledgeDirectoryID: directoryID, Status: service.ContentStatusDraft, IsPublic: true, Excerpt: normalized.Excerpt, Body: normalized.Body}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&content).Error; err != nil {
			return err
		}
		if err := createContentRevision(tx, content, principal.UserID, "create"); err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "content.create", "content", content.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "content.create_failed", "内容草稿创建失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusCreated, h.contentAdminItem(content, principal))
}

// AdminUpdateContent 更新内容草稿；已发布或审核中的内容需遵循额外状态限制。
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
	if err := requireContentEdit(h.db, principal, content); err != nil {
		if errors.Is(err, errContentEditForbidden) {
			fail(c, http.StatusForbidden, "content.author_required", "普通编辑只能修改自己创建的内容。")
			return
		}
		fail(c, http.StatusInternalServerError, "content.permission_check_failed", "内容权限校验失败。")
		return
	}
	if content.Status == service.ContentStatusPublished {
		fail(c, http.StatusConflict, "content.published_immutable", "已发布内容不能直接编辑，请先下线。")
		return
	}
	if content.Status == service.ContentStatusReview {
		fail(c, http.StatusConflict, "content.review_immutable", "内容正在审核，退回后才能继续编辑。")
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
		if err := bindMarkdownAssets(tx, principal.OrganizationID, content.ID, content.Body); err != nil {
			return err
		}
		if err := createContentRevision(tx, content, principal.UserID, "update"); err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "content.update", "content", content.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "content.update_failed", "内容保存失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, h.contentAdminItem(content, principal))
}

type contentVisibilityRequest struct {
	IsPublic *bool `json:"is_public"`
}

// AdminUpdateContentVisibility 仅允许已发布内容切换门户公开或仅登录成员可见，不改动生命周期。
func (h *WorkspaceHandler) AdminUpdateContentVisibility(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var body contentVisibilityRequest
	if err := c.ShouldBindJSON(&body); err != nil || body.IsPublic == nil {
		fail(c, http.StatusBadRequest, "content.visibility_invalid", "请指定内容是否对未登录访客公开。")
		return
	}
	var content model.Content
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&content).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "content.not_found", "内容不存在或不属于当前组织。")
			return
		}
		fail(c, http.StatusInternalServerError, "content.visibility_failed", "内容公开状态暂时无法更新。")
		return
	}
	if err := requireContentEdit(h.db, principal, content); err != nil {
		if errors.Is(err, errContentEditForbidden) {
			fail(c, http.StatusForbidden, "content.author_required", "普通编辑只能设置自己创建的内容是否公开。")
			return
		}
		fail(c, http.StatusInternalServerError, "content.permission_check_failed", "内容权限校验失败。")
		return
	}
	if content.Status != service.ContentStatusPublished {
		fail(c, http.StatusConflict, "content.visibility_requires_published", "只有已发布内容可以设置门户公开或仅登录成员可见。")
		return
	}
	if content.IsPublic == *body.IsPublic {
		respond(c, http.StatusOK, h.contentAdminItem(content, principal))
		return
	}
	content.IsPublic = *body.IsPublic
	content.UpdatedAt = time.Now().UTC()
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&content).Updates(map[string]any{"is_public": content.IsPublic, "updated_at": content.UpdatedAt}).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "content.visibility", "content", content.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "content.visibility_failed", "内容公开状态更新失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, h.contentAdminItem(content, principal))
}

// AdminDeleteContent permanently removes a non-published content record along
// with its revisions and review requests. Uploaded media stays in the
// organization library but is detached from the removed content.
func (h *WorkspaceHandler) AdminDeleteContent(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var content model.Content
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&content).Error; err != nil {
			return err
		}
		if err := requireContentEdit(tx, principal, content); err != nil {
			return err
		}
		if content.Status == service.ContentStatusPublished {
			return errContentPublishedBlocked
		}
		if err := deleteContentReviewRecords(tx, principal.OrganizationID, content.ID); err != nil {
			return err
		}
		if err := tx.Where("content_id = ? AND organization_id = ?", content.ID, principal.OrganizationID).Delete(&model.ContentRevision{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.MediaAsset{}).
			Where("content_id = ? AND organization_id = ?", content.ID, principal.OrganizationID).
			Update("content_id", "").Error; err != nil {
			return err
		}
		result := tx.Where("id = ? AND organization_id = ?", content.ID, principal.OrganizationID).Delete(&model.Content{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "content.delete", "content", content.ID)
	})
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			fail(c, http.StatusNotFound, "content.not_found", "内容不存在或不属于当前组织。")
		case errors.Is(err, errContentEditForbidden):
			fail(c, http.StatusForbidden, "content.author_required", "普通编辑只能删除自己创建的内容。")
		case errors.Is(err, errContentPublishedBlocked):
			fail(c, http.StatusConflict, "content.published_delete_blocked", "已发布内容不能删除，请先下线后再删除。")
		default:
			fail(c, http.StatusInternalServerError, "content.delete_failed", "内容删除失败。")
		}
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, gin.H{"removed": true, "id": c.Param("id")})
}

// PublishContent 将内容提交为发布状态，具体迁移规则由 changeContentStatus 统一处理。
func (h *WorkspaceHandler) PublishContent(c *gin.Context) { h.changeContentStatus(c, "published") }

// ArchiveContent 将内容提交为归档状态，具体迁移规则由 changeContentStatus 统一处理。
func (h *WorkspaceHandler) ArchiveContent(c *gin.Context) { h.changeContentStatus(c, "archived") }

// changeContentStatus 执行内容状态迁移、权限检查、版本记录、资产绑定和审计写入。
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
		if err := tx.Save(&content).Error; err != nil {
			return err
		}
		if err := createContentRevision(tx, content, principal.UserID, status); err != nil {
			return err
		}
		revision, err := latestContentRevision(tx, principal.OrganizationID, content.ID)
		if err != nil {
			return err
		}
		if err := h.resolveContentReviewDecision(tx, content, revision, principal, status); err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "content."+status, "content", content.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "content.status_update_failed", "内容状态更新失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, h.contentAdminItem(content, principal))
}

// bindMarkdownAssets 从 Markdown 中提取资产链接，并把引用关系同步到内容资产关联表。
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

// createContentRevision 将当前内容快照写入递增版本号的修订记录。
func createContentRevision(tx *gorm.DB, content model.Content, actorUserID, reason string) error {
	var version int
	if err := tx.Model(&model.ContentRevision{}).Where("content_id = ?", content.ID).Select("COALESCE(MAX(version), 0)").Scan(&version).Error; err != nil {
		return err
	}
	directoryID := ""
	if content.KnowledgeDirectoryID != nil {
		directoryID = *content.KnowledgeDirectoryID
	}
	return tx.Create(&model.ContentRevision{
		ID: uuid.NewString(), OrganizationID: content.OrganizationID, ContentID: content.ID, Version: version + 1,
		CreatedBy: actorUserID, Reason: reason, Title: content.Title, Type: content.Type, Category: content.Category,
		KnowledgeDirectoryID: directoryID, Status: content.Status, Excerpt: content.Excerpt, Body: content.Body,
		PublishedAt: content.PublishedAt, CreatedAt: time.Now().UTC(),
	}).Error
}

// AdminContentRevisions 分页列出指定内容的历史版本。
func (h *WorkspaceHandler) AdminContentRevisions(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	if _, visible := h.loadVisibleAdminContent(c, principal, c.Param("id")); !visible {
		return
	}
	page, pageSize, ok := listMeta(c, 0)
	if !ok {
		return
	}
	query := h.db.Model(&model.ContentRevision{}).Where("organization_id = ? AND content_id = ?", principal.OrganizationID, c.Param("id"))
	var total int64
	if err := query.Count(&total).Error; err != nil {
		fail(c, http.StatusInternalServerError, "content.revision_list_failed", "内容修订历史暂时无法加载。")
		return
	}
	var revisions []model.ContentRevision
	if err := query.Order("version DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&revisions).Error; err != nil {
		fail(c, http.StatusInternalServerError, "content.revision_list_failed", "内容修订历史暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(revisions))
	for _, revision := range revisions {
		items = append(items, contentRevisionItem(revision, false, h.db))
	}
	respondWithMeta(c, http.StatusOK, items, gin.H{"page": page, "page_size": pageSize, "total": total})
}

// AdminContentRevisionDetail 返回指定版本的完整快照，包括正文和变更元数据。
func (h *WorkspaceHandler) AdminContentRevisionDetail(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	if _, visible := h.loadVisibleAdminContent(c, principal, c.Param("id")); !visible {
		return
	}
	var revision model.ContentRevision
	if err := h.db.Where("id = ? AND content_id = ? AND organization_id = ?", c.Param("revision_id"), c.Param("id"), principal.OrganizationID).First(&revision).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "content.revision_not_found", "内容修订版本不存在。")
			return
		}
		fail(c, http.StatusInternalServerError, "content.revision_detail_failed", "内容修订版本暂时无法加载。")
		return
	}
	respond(c, http.StatusOK, contentRevisionItem(revision, true, h.db))
}

// RestoreContentRevision 将历史版本复制回当前内容，并产生新的修订记录以保留可追溯性。
func (h *WorkspaceHandler) RestoreContentRevision(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var revision model.ContentRevision
	var content model.Content
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND content_id = ? AND organization_id = ?", c.Param("revision_id"), c.Param("id"), principal.OrganizationID).First(&revision).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&content).Error; err != nil {
			return err
		}
		if err := requireContentEdit(tx, principal, content); err != nil {
			return err
		}
		if content.Status == service.ContentStatusPublished || content.Status == service.ContentStatusReview {
			return errContentReviewStateInvalid
		}
		var directoryID *string
		if revision.KnowledgeDirectoryID != "" {
			directoryID = &revision.KnowledgeDirectoryID
		}
		content.Title, content.Type, content.Category, content.KnowledgeDirectoryID = revision.Title, revision.Type, revision.Category, directoryID
		content.Status, content.Excerpt, content.Body, content.PublishedAt = service.ContentStatusDraft, revision.Excerpt, revision.Body, nil
		if err := tx.Save(&content).Error; err != nil {
			return err
		}
		if err := bindMarkdownAssets(tx, principal.OrganizationID, content.ID, content.Body); err != nil {
			return err
		}
		if err := createContentRevision(tx, content, principal.UserID, "restore"); err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "content.revision_restore", "content", content.ID)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fail(c, http.StatusNotFound, "content.revision_not_found", "内容或修订版本不存在。")
			return
		}
		if errors.Is(err, errContentEditForbidden) {
			fail(c, http.StatusForbidden, "content.author_required", "普通编辑只能恢复自己创建的内容版本。")
			return
		}
		if errors.Is(err, errContentReviewStateInvalid) {
			fail(c, http.StatusConflict, "content.revision_restore_state_invalid", "已发布或审核中的内容不能恢复版本，请先完成审核或下线。")
			return
		}
		fail(c, http.StatusInternalServerError, "content.revision_restore_failed", "内容修订版本恢复失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, h.contentAdminItem(content, principal))
}

// contentRevisionItem 将内容修订模型投影为列表或详情响应，按 includeBody 控制是否返回正文。
func contentRevisionItem(revision model.ContentRevision, includeBody bool, db *gorm.DB) gin.H {
	var creator model.User
	_ = db.Select("display_name").First(&creator, "id = ?", revision.CreatedBy).Error
	changedFields := []string{}
	addedLines, removedLines := 0, 0
	var previous model.ContentRevision
	if revision.Version > 1 && db.Where("content_id = ? AND version = ?", revision.ContentID, revision.Version-1).First(&previous).Error == nil {
		changedFields = contentRevisionChangedFields(previous, revision)
		addedLines, removedLines = contentBodyDiffStats(previous.Body, revision.Body)
	}
	item := gin.H{"id": revision.ID, "content_id": revision.ContentID, "version": revision.Version, "created_by": revision.CreatedBy, "created_by_name": creator.DisplayName, "reason": revision.Reason, "title": revision.Title, "type": revision.Type, "category": revision.Category, "knowledge_directory_id": nil, "status": revision.Status, "excerpt": revision.Excerpt, "published_at": revision.PublishedAt, "created_at": revision.CreatedAt, "changed_fields": changedFields, "body_diff": gin.H{"added_lines": addedLines, "removed_lines": removedLines}}
	if revision.KnowledgeDirectoryID != "" {
		item["knowledge_directory_id"] = revision.KnowledgeDirectoryID
	}
	if includeBody {
		item["body"] = revision.Body
	}
	return item
}

// contentRevisionChangedFields 比较相邻快照并列出发生变化的业务字段。
func contentRevisionChangedFields(previous, current model.ContentRevision) []string {
	fields := make([]string, 0, 7)
	if previous.Title != current.Title {
		fields = append(fields, "title")
	}
	if previous.Type != current.Type {
		fields = append(fields, "type")
	}
	if previous.Category != current.Category {
		fields = append(fields, "category")
	}
	if previous.KnowledgeDirectoryID != current.KnowledgeDirectoryID {
		fields = append(fields, "knowledge_directory_id")
	}
	if previous.Excerpt != current.Excerpt {
		fields = append(fields, "excerpt")
	}
	if previous.Body != current.Body {
		fields = append(fields, "body")
	}
	if previous.Status != current.Status {
		fields = append(fields, "status")
	}
	return fields
}

// contentBodyDiffStats 以行文本为单位计算正文新增与删除行数，供版本详情展示。
func contentBodyDiffStats(previous, current string) (int, int) {
	previousLines := strings.Split(strings.ReplaceAll(previous, "\r\n", "\n"), "\n")
	currentLines := strings.Split(strings.ReplaceAll(current, "\r\n", "\n"), "\n")
	counts := make(map[string]int, len(previousLines))
	for _, line := range previousLines {
		counts[line]++
	}
	added := 0
	for _, line := range currentLines {
		if counts[line] > 0 {
			counts[line]--
		} else {
			added++
		}
	}
	removed := 0
	for _, count := range counts {
		removed += count
	}
	return added, removed
}

// canTransitionContentStatus 判断内容状态机是否允许从 current 迁移到 target。
func canTransitionContentStatus(current, target string) bool {
	return service.CanTransitionContentStatus(current, target)
}

// contentPublicItem 生成公开内容列表 DTO，隐藏草稿、审核和内部审计字段。
func contentPublicItem(content model.Content) gin.H {
	publishedAt := content.PublishedAt
	if publishedAt == nil {
		publishedAt = &content.UpdatedAt
	}
	category := content.Category
	if category == "" {
		category = content.Type
	}
	return gin.H{"id": content.ID, "title": content.Title, "excerpt": content.Excerpt, "category": category, "published_at": publishedAt, "reading_minutes": maxInt(1, len([]rune(content.Body))/900+1), "members_only": !content.IsPublic}
}

// contentPublicDetailItem 生成公开内容详情 DTO，并将正文资产路径改写为门户下载路径。
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
		"members_only":    !content.IsPublic,
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

// publicContentBody 替换管理端资产 URL，使公开正文不暴露内部管理路由。
func publicContentBody(slug, body string) string {
	publicPrefix := "/api/v1/portal/organizations/" + slug + "/assets/"
	return markdownAdminAssetPattern.ReplaceAllString(body, publicPrefix+"$1/download")
}

// resourcePublicItem 生成资源内容的公开 DTO，并附加可访问的资产下载地址。
func (h *WorkspaceHandler) resourcePublicItem(slug string, content model.Content) gin.H {
	kind := content.Category
	if kind != "document" && kind != "template" && kind != "package" && kind != "video" {
		kind = "document"
	}
	item := gin.H{"id": content.ID, "title": content.Title, "description": content.Excerpt, "kind": kind, "size_bytes": int64(0), "updated_at": content.UpdatedAt, "download_url": nil, "members_only": !content.IsPublic}
	var asset model.MediaAsset
	if h.db.Where("content_id = ? AND organization_id = ?", content.ID, content.OrganizationID).Order("created_at ASC").First(&asset).Error == nil {
		item["size_bytes"] = asset.SizeBytes
		item["download_url"] = "/api/v1/portal/organizations/" + slug + "/assets/" + asset.ID + "/download"
	}
	return item
}

// contentAdminItem 生成管理端内容 DTO，并根据当前主体计算可编辑等能力标记。
func (h *WorkspaceHandler) contentAdminItem(content model.Content, principal service.Principal) gin.H {
	var author model.User
	_ = h.db.First(&author, "id = ?", content.AuthorUserID).Error
	var revisionCount int64
	_ = h.db.Model(&model.ContentRevision{}).Where("content_id = ? AND organization_id = ?", content.ID, content.OrganizationID).Count(&revisionCount).Error
	var asset model.MediaAsset
	assetItem := interface{}(nil)
	if h.db.Where("content_id = ? AND organization_id = ?", content.ID, content.OrganizationID).Order("created_at ASC").First(&asset).Error == nil {
		assetItem = gin.H{"id": asset.ID, "original_name": asset.OriginalName, "mime_type": asset.MimeType, "size_bytes": asset.SizeBytes, "download_count": asset.DownloadCount, "last_downloaded_at": asset.LastDownloadedAt, "download_url": "/api/v1/admin/assets/" + asset.ID + "/download"}
	}
	var directoryID interface{} = nil
	if content.KnowledgeDirectoryID != nil && *content.KnowledgeDirectoryID != "" {
		directoryID = *content.KnowledgeDirectoryID
	}
	isAuthor := content.AuthorUserID == principal.UserID
	canModerate, _ := principalCanModerateContent(h.db, principal)
	canSubmitPermission, _ := principalHasPermission(h.db, principal, "content:submit")
	canPublishPermission, _ := principalHasPermission(h.db, principal, "content:publish")
	canArchivePermission, _ := principalHasPermission(h.db, principal, "content:archive")
	editableState := content.Status == service.ContentStatusDraft || content.Status == service.ContentStatusArchived
	canDelete := content.Status != service.ContentStatusPublished && (isAuthor || canModerate)
	var pendingReview any
	var review model.ContentReviewRequest
	if h.db.Where("organization_id = ? AND content_id = ? AND status = ?", content.OrganizationID, content.ID, contentReviewPending).Order("created_at DESC").First(&review).Error == nil {
		if isAuthor || canModerate {
			pendingReview = contentReviewItem(h.db, review)
		}
	}
	canReview := false
	if pendingReview != nil {
		canReview = (review.Type == contentReviewTypePublish && canPublishPermission) || (review.Type == contentReviewTypeArchive && canArchivePermission)
	}
	return gin.H{
		"id": content.ID, "title": content.Title, "type": content.Type, "category": content.Category,
		"knowledge_directory_id": directoryID, "status": content.Status, "is_public": content.IsPublic,
		"author_user_id": content.AuthorUserID, "author": author.DisplayName, "is_author": isAuthor,
		"excerpt": content.Excerpt, "body": content.Body, "published_at": content.PublishedAt,
		"updated_at": content.UpdatedAt, "revision_count": revisionCount, "asset": assetItem,
		"pending_review":      pendingReview,
		"can_edit":            editableState && (isAuthor || canModerate),
		"can_submit":          editableState && canSubmitPermission && (isAuthor || canModerate),
		"can_publish":         canPublishPermission && (content.Status == service.ContentStatusReview || content.Status == service.ContentStatusDraft || content.Status == service.ContentStatusArchived),
		"can_archive":         canArchivePermission && content.Status == service.ContentStatusPublished,
		"can_request_archive": isAuthor && content.Status == service.ContentStatusPublished && pendingReview == nil,
		"can_review":          canReview,
		"can_set_visibility":  content.Status == service.ContentStatusPublished && (isAuthor || canModerate),
		"can_delete":          canDelete,
	}
}

// maxInt 返回两个整数中的较大值，用于分页和统计结果的边界保护。
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// AdminUsers 分页列出组织成员，并附带角色、状态和用户资料。
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

// MembershipHistory 返回成员状态和角色变更的审计历史。
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

// LeaveMembership 允许当前用户主动离开组织，但不允许破坏最后的管理员保障。
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
		if err := tx.Create(&model.MembershipEvent{ID: uuid.NewString(), MembershipID: membership.ID, State: "left", Reason: "self_leave"}).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "membership.leave", "membership", membership.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "membership.leave_failed", "退出组织暂时无法完成。")
		return
	}
	respond(c, http.StatusOK, gin.H{"state": "left", "left_at": time.Now().UTC()})
}

// AdminUpdateUser 更新成员角色、状态和显示资料，并执行自我降权等保护规则。
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
		if err := tx.Create(&model.MembershipEvent{
			ID:           uuid.NewString(),
			MembershipID: membership.ID,
			State:        body.State,
			Reason:       membershipUpdateReason(currentState, targetRole, body.State, body.Role),
		}).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "membership.update", "membership", membership.ID)
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "membership.update_failed", "成员信息保存失败。")
		return
	}
	respond(c, http.StatusOK, gin.H{"id": user.ID, "name": user.DisplayName, "email": user.Email, "role": body.Role, "state": body.State, "joined_at": membership.CreatedAt})
}

func membershipDeleteError(actorIsSelf bool, currentRole string) string {
	if currentRole == "owner" {
		return "membership.owner_protected"
	}
	if actorIsSelf {
		return "membership.self_delete_forbidden"
	}
	return ""
}

func (h *WorkspaceHandler) AdminDeleteUser(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var membership model.Membership
	if err := h.db.Where("organization_id = ? AND user_id = ?", principal.OrganizationID, c.Param("id")).First(&membership).Error; err != nil {
		fail(c, http.StatusNotFound, "membership.not_found", "成员不存在。")
		return
	}
	currentRole := membershipRole(h.db, membership.ID)
	if code := membershipDeleteError(c.Param("id") == principal.UserID, currentRole); code != "" {
		status := http.StatusConflict
		message := "成员移除不符合保护规则。"
		if code == "membership.owner_protected" {
			status = http.StatusForbidden
			message = "所有者不能被移出组织。"
		} else if code == "membership.self_delete_forbidden" {
			message = "不能通过成员管理删除自己，请使用退出组织功能。"
		}
		fail(c, status, code, message)
		return
	}
	var ownedProjects int64
	if err := h.db.Model(&model.Project{}).
		Where("organization_id = ? AND owner_user_id = ?", principal.OrganizationID, c.Param("id")).
		Count(&ownedProjects).Error; err != nil {
		fail(c, http.StatusInternalServerError, "membership.delete_failed", "成员移除暂时无法完成。")
		return
	}
	if ownedProjects > 0 {
		fail(c, http.StatusConflict, "membership.owns_projects", "该成员仍是项目负责人，请先删除或转移名下项目。")
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("membership_id = ?", membership.ID).Delete(&model.MembershipRole{}).Error; err != nil {
			return err
		}
		if err := tx.Where("membership_id = ?", membership.ID).Delete(&model.MembershipEvent{}).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&model.RefreshToken{}).
			Where("organization_id = ? AND user_id = ? AND revoked_at IS NULL", principal.OrganizationID, c.Param("id")).
			Update("revoked_at", now).Error; err != nil {
			return err
		}
		var projectIDs []string
		if err := tx.Model(&model.Project{}).
			Where("organization_id = ?", principal.OrganizationID).
			Pluck("id", &projectIDs).Error; err != nil {
			return err
		}
		if len(projectIDs) > 0 {
			if err := tx.Where("project_id IN ? AND user_id = ?", projectIDs, c.Param("id")).
				Delete(&model.ProjectMember{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Delete(&membership).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "membership.delete", "membership", membership.ID)
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, "membership.delete_failed", "成员移除失败。")
		return
	}
	respond(c, http.StatusOK, gin.H{"removed": true, "id": membership.ID, "user_id": c.Param("id")})
}

// validMemberWriteState 判断成员写操作使用的状态是否属于系统枚举。
func validMemberWriteState(value string) bool {
	return value == "active" || value == "disabled"
}

// validRole 判断成员角色是否为受支持的角色枚举。
func validRole(value string) bool {
	return value == "member" || value == "editor" || value == "administrator" || value == "owner"
}

// membershipRole 查询成员记录对应的有效角色，查询失败时返回空字符串。
func membershipRole(db *gorm.DB, membershipID string) string {
	var role string
	db.Table("membership_roles AS mr").Select("COALESCE(MAX(r.`key`), 'member')").Joins("JOIN roles AS r ON r.id = mr.role_id").Where("mr.membership_id = ?", membershipID).Scan(&role)
	return role
}

// membershipRoleByUser 按组织和用户查询角色，避免跨组织读取成员权限。
func membershipRoleByUser(db *gorm.DB, organizationID, userID string) string {
	var role string
	db.Table("memberships AS m").Select("COALESCE(MAX(r.`key`), 'member')").Joins("JOIN membership_roles AS mr ON mr.membership_id = m.id").Joins("JOIN roles AS r ON r.id = mr.role_id").Where("m.organization_id = ? AND m.user_id = ? AND m.state = ?", organizationID, userID, "active").Scan(&role)
	return role
}

// membershipChangeError 返回成员变更违反权限或组织安全规则时的错误原因。
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

// membershipUpdateReason 生成写入成员审计事件的状态/角色变化描述。
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

// AdminProjects 分页列出管理端可见的组织项目。
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

// AdminCreateProject 创建组织项目并初始化其公开/管理字段。
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
		if err := tx.Create(&model.ProjectMember{ProjectID: project.ID, UserID: principal.UserID, Role: "owner"}).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "project.create", "project", project.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "project.create_failed", "项目创建失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusCreated, projectAdminItem(project, h.db))
}

// AdminUpdateProject 更新项目基础信息并记录变更审计。
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
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&project).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "project.update", "project", project.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "project.update_failed", "项目保存失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, projectAdminItem(project, h.db))
}

func (h *WorkspaceHandler) AdminDeleteProject(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var project model.Project
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&project).Error; err != nil {
		fail(c, http.StatusNotFound, "project.not_found", "项目不存在。")
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", project.ID).Delete(&model.ProjectMilestone{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", project.ID).Delete(&model.ProjectMember{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.ActivityPlan{}).
			Where("organization_id = ? AND project_id = ?", principal.OrganizationID, project.ID).
			Update("project_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Delete(&project).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "project.delete", "project", project.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "project.delete_failed", "项目删除失败。")
		return
	}
	h.invalidatePortalCache(principal.OrganizationID)
	respond(c, http.StatusOK, gin.H{"removed": true, "id": project.ID})
}

// projectForPrincipal 解析当前组织中的项目并提取主体；失败时直接写入对应响应。
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

// AdminProjectMembers 列出项目成员及其项目内角色。
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

// projectMemberRequest 描述添加或修改项目成员时的组织成员 ID 和项目角色。
type projectMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// validProjectMemberRole 判断项目角色是否属于 owner、manager 或 member。
func validProjectMemberRole(value string) bool {
	return value == "member" || value == "contributor" || value == "lead"
}

// AdminAddProjectMember 将组织成员加入项目并防止重复关联。
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
		if err := h.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(&member).Error; err != nil {
				return err
			}
			return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "project_member.update", "project_member", member.UserID)
		}); err != nil {
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
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&member).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "project_member.create", "project_member", member.UserID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "project_member.create_failed", "项目成员添加失败。")
		return
	}
	respond(c, http.StatusCreated, projectMemberItem(h.db, member))
}

// AdminUpdateProjectMember 修改项目成员角色。
func (h *WorkspaceHandler) AdminUpdateProjectMember(c *gin.Context) {
	project, principal, ok := h.projectForPrincipal(c)
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
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&member).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "project_member.update", "project_member", member.UserID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "project_member.update_failed", "项目成员角色保存失败。")
		return
	}
	respond(c, http.StatusOK, projectMemberItem(h.db, member))
}

// AdminRemoveProjectMember 从项目移除成员，并保留项目所有权约束。
func (h *WorkspaceHandler) AdminRemoveProjectMember(c *gin.Context) {
	project, principal, ok := h.projectForPrincipal(c)
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
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&member).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "project_member.delete", "project_member", member.UserID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "project_member.delete_failed", "项目成员移除失败。")
		return
	}
	respond(c, http.StatusOK, gin.H{"removed": true, "user_id": member.UserID, "project_id": project.ID})
}

// projectMemberItem 将项目成员模型与用户资料合并为管理端 DTO。
func projectMemberItem(db *gorm.DB, member model.ProjectMember) gin.H {
	var user model.User
	_ = db.First(&user, "id = ?", member.UserID).Error
	return gin.H{"user_id": member.UserID, "name": user.DisplayName, "email": user.Email, "state": user.State, "role": member.Role, "assigned_at": member.CreatedAt}
}

// AdminProjectMilestones 列出项目里程碑，并按项目权限限制访问范围。
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

// projectMilestoneRequest 描述里程碑名称、状态、截止时间和完成说明。
type projectMilestoneRequest struct {
	Title  string `json:"title"`
	Status string `json:"status"`
	DueAt  string `json:"due_at"`
}

// validProjectMilestoneStatus 判断里程碑状态是否为受支持的生命周期值。
func validProjectMilestoneStatus(value string) bool {
	return value == "planned" || value == "active" || value == "completed"
}

// parseOptionalTime 解析可为空的 RFC3339 时间；空值表示不设置时间，非法值返回 false。
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

// validProjectMilestoneRequest 验证里程碑标题、状态和描述长度。
func validProjectMilestoneRequest(body projectMilestoneRequest) bool {
	return strings.TrimSpace(body.Title) != "" && len([]rune(body.Title)) <= 160 && validProjectMilestoneStatus(body.Status)
}

// AdminCreateProjectMilestone 为项目创建里程碑并写入审计记录。
func (h *WorkspaceHandler) AdminCreateProjectMilestone(c *gin.Context) {
	project, principal, ok := h.projectForPrincipal(c)
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
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&milestone).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "project_milestone.create", "project_milestone", milestone.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "project_milestone.create_failed", "里程碑创建失败。")
		return
	}
	respond(c, http.StatusCreated, projectMilestoneItem(milestone))
}

// AdminUpdateProjectMilestone 更新项目里程碑并记录状态变化。
func (h *WorkspaceHandler) AdminUpdateProjectMilestone(c *gin.Context) {
	project, principal, ok := h.projectForPrincipal(c)
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
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&milestone).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "project_milestone.update", "project_milestone", milestone.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "project_milestone.update_failed", "里程碑保存失败。")
		return
	}
	respond(c, http.StatusOK, projectMilestoneItem(milestone))
}

// AdminDeleteProjectMilestone 删除指定项目里程碑。
func (h *WorkspaceHandler) AdminDeleteProjectMilestone(c *gin.Context) {
	project, principal, ok := h.projectForPrincipal(c)
	if !ok {
		return
	}
	var milestone model.ProjectMilestone
	if err := h.db.Where("id = ? AND project_id = ?", c.Param("milestone_id"), project.ID).First(&milestone).Error; err != nil {
		fail(c, http.StatusNotFound, "project_milestone.not_found", "里程碑不存在。")
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&milestone).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "project_milestone.delete", "project_milestone", milestone.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "project_milestone.delete_failed", "里程碑删除失败。")
		return
	}
	respond(c, http.StatusOK, gin.H{"removed": true, "id": milestone.ID, "project_id": project.ID})
}

// projectMilestoneItem 将里程碑模型转换为稳定的管理端/门户响应。
func projectMilestoneItem(milestone model.ProjectMilestone) gin.H {
	return gin.H{"id": milestone.ID, "project_id": milestone.ProjectID, "title": milestone.Title, "status": milestone.Status, "due_at": milestone.DueAt, "completed_at": milestone.CompletedAt, "updated_at": milestone.UpdatedAt}
}

// projectRequest 描述创建或更新项目时的基础资料和标签。
type projectRequest struct {
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	Status   string   `json:"status"`
	Tags     []string `json:"tags"`
	IsPublic bool     `json:"is_public"`
}

// validProjectRequest 校验项目名称、摘要、slug 和标签数量/长度。
func validProjectRequest(body projectRequest) bool {
	if strings.TrimSpace(body.Title) == "" || len([]rune(body.Title)) > 160 || len([]rune(body.Summary)) > 500 {
		return false
	}
	if body.Status != "active" && body.Status != "research" && body.Status != "completed" {
		return false
	}
	return len(body.Tags) <= 12
}

// projectPublicItem 生成公开项目 DTO，隐藏管理字段。
func projectPublicItem(project model.Project) gin.H {
	return gin.H{"id": project.ID, "title": project.Title, "summary": project.Summary, "status": project.Status, "tags": splitTags(project.Tags), "updated_at": project.UpdatedAt}
}

// projectAdminItem 生成管理端项目 DTO，并补充成员和里程碑统计。
func projectAdminItem(project model.Project, db *gorm.DB) gin.H {
	var owner model.User
	_ = db.First(&owner, "id = ?", project.OwnerUserID).Error
	var memberCount, milestoneCount int64
	db.Model(&model.ProjectMember{}).Where("project_id = ?", project.ID).Count(&memberCount)
	db.Model(&model.ProjectMilestone{}).Where("project_id = ?", project.ID).Count(&milestoneCount)
	return gin.H{"id": project.ID, "title": project.Title, "summary": project.Summary, "status": project.Status, "tags": splitTags(project.Tags), "is_public": project.IsPublic, "owner": owner.DisplayName, "member_count": memberCount, "milestone_count": milestoneCount, "updated_at": project.UpdatedAt}
}

// splitTags 将逗号分隔标签清洗为有序切片，并丢弃空标签。
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

// applicationRequest 描述加入组织或申请参与项目时提交的个人与动机信息。
type applicationRequest struct {
	Type      string `json:"type"`
	ClassName string `json:"class_name"`
	Name      string `json:"name"`
	GameID    string `json:"game_id"`
	QQNumber  string `json:"qq_number"`
	Email     string `json:"email"`
	Note      string `json:"note"`
}

// applicationDecisionRequest 指定管理员对申请采取的决定及可选备注。
type applicationDecisionRequest struct {
	Reason         string `json:"reason"`
	SkinInviteCode string `json:"skin_invite_code"`
}

var qqNumberPattern = regexp.MustCompile(`^[0-9]{5,15}$`)

// validApplicationRequest 按申请类型验证必填字段，并限制文本长度和审核模式枚举。
func validApplicationRequest(body applicationRequest) bool {
	if body.Type == "" {
		body.Type = "whitelist"
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	parsedEmail, err := mail.ParseAddress(email)
	if err != nil || parsedEmail.Address != email {
		return false
	}
	if body.Type != "whitelist" && body.Type != "membership" {
		return false
	}
	if strings.TrimSpace(body.Name) == "" || len([]rune(body.Name)) > 80 || len([]rune(body.Note)) > 500 {
		return false
	}
	if body.Type == "membership" {
		return len([]rune(body.ClassName)) <= 120 && len([]rune(body.GameID)) <= 80 && len([]rune(body.QQNumber)) <= 32
	}
	return strings.TrimSpace(body.ClassName) != "" && len([]rune(body.ClassName)) <= 120 &&
		strings.TrimSpace(body.GameID) != "" && len([]rune(body.GameID)) <= 80 &&
		qqNumberPattern.MatchString(strings.TrimSpace(body.QQNumber))
}

// SubmitApplication 创建公开申请，校验重复申请和组织/项目归属。
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
		fail(c, http.StatusBadRequest, "application.validation_failed", "申请人姓名、邮箱或当前申请类型所需字段不符合规范。")
		return
	}

	var organization model.Organization
	if err := h.db.Where("slug = ? AND is_public = ?", c.Param("slug"), true).First(&organization).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}

	var existing model.Application
	duplicateQuery := h.db.Where("organization_id = ? AND status = ? AND LOWER(email) = ?", organization.ID, "pending", body.Email)
	if body.Type == "whitelist" {
		duplicateQuery = h.db.Where("organization_id = ? AND status = ? AND (LOWER(email) = ? OR game_id = ?)", organization.ID, "pending", body.Email, body.GameID)
	}
	if err := duplicateQuery.First(&existing).Error; err == nil {
		fail(c, http.StatusConflict, "application.duplicate_pending", "相同邮箱或申请标识已有待处理申请。")
		return
	} else if err != gorm.ErrRecordNotFound {
		fail(c, http.StatusInternalServerError, "application.lookup_failed", "申请暂时无法提交。")
		return
	}

	note := body.Note
	if note == "" && body.Type == "whitelist" {
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
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&application).Error; err != nil {
			return err
		}
		if h.notifications == nil {
			return nil
		}
		recipients, err := permissionRecipientEmails(tx, organization.ID, "application:approve", "")
		if err != nil {
			return err
		}
		return h.notifications.EnqueueApplicationSubmitted(tx, application, recipients)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "application.create_failed", "申请暂时无法提交，请稍后重试。")
		return
	}
	respond(c, http.StatusCreated, gin.H{"id": application.ID, "status": application.Status, "submitted_at": application.CreatedAt})
}

// AdminApplications 分页列出组织申请，并支持按状态和类型过滤。
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

// AdminApplicationDecision 接受或拒绝申请，更新成员/项目关系并写入审计事件。
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
	application, err := h.applications.Decide(principal.OrganizationID, principal.UserID, decision, next, body.Reason, body.SkinInviteCode, ensureRequestID(c))
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
		case errors.Is(err, service.ErrApplicationSkinInviteCodeTooLong):
			fail(c, http.StatusBadRequest, "application.skin_invite_code_too_long", "皮肤站邀请码不能超过 500 个字符。")
		default:
			fail(c, http.StatusInternalServerError, "application.decision_failed", "申请状态暂时无法更新。")
		}
		return
	}
	respond(c, http.StatusOK, h.applicationAdminItem(application))
}

func (h *WorkspaceHandler) AdminDeleteApplication(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var application model.Application
	if err := h.db.Where("id = ? AND organization_id = ?", c.Param("id"), principal.OrganizationID).First(&application).Error; err != nil {
		fail(c, http.StatusNotFound, "application.not_found", "申请不存在。")
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("organization_id = ? AND target_type = ? AND target_id = ?", principal.OrganizationID, "application", application.ID).Delete(&model.NotificationOutbox{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&application).Error; err != nil {
			return err
		}
		return writeAudit(tx, c, principal.OrganizationID, principal.UserID, "application.delete", "application", application.ID)
	}); err != nil {
		fail(c, http.StatusInternalServerError, "application.delete_failed", "申请删除失败。")
		return
	}
	respond(c, http.StatusOK, gin.H{"removed": true, "id": application.ID})
}

// applicationAdminItem 将申请模型转换为管理端展示对象，隐藏不必要的内部字段。
func (h *WorkspaceHandler) applicationAdminItem(application model.Application) gin.H {
	return gin.H{"id": application.ID, "applicant": application.ApplicantName, "type": application.Type, "submitted_at": application.CreatedAt, "note": application.Note, "status": application.Status, "class_name": application.ClassName, "game_id": application.GameID, "qq_number": application.QQNumber, "email": application.Email, "decided_at": application.DecidedAt, "decided_by": application.DecidedBy, "decision_reason": application.DecisionReason, "skin_invite_code": application.SkinInviteCode}
}

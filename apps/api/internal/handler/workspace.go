package handler

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkspaceHandler owns the first content read/write model shared by the
// public portal and the protected CMS workspace.
type WorkspaceHandler struct {
	db                *gorm.DB
	mu                sync.RWMutex
	applicationStates map[string]string
}

func NewWorkspaceHandler(db *gorm.DB) *WorkspaceHandler {
	return &WorkspaceHandler{db: db, applicationStates: map[string]string{"application_001": "pending"}}
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
	var contents []model.Content
	if err := query.Find(&contents).Error; err != nil {
		fail(c, http.StatusInternalServerError, "portal.posts_failed", "公开动态暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(contents))
	for _, content := range contents {
		items = append(items, contentPublicItem(content))
	}
	pageOf(c, items)
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
	items := []gin.H{{"id": "project_cms", "title": "QUTCraft CMS", "summary": "面向校园社团与民间组织的公开门户与内容分发系统。", "status": "active", "tags": []string{"Vue 3", "Go", "API-first"}, "updated_at": "2026-07-17T03:00:00Z"}}
	if status != "" && items[0]["status"] != status {
		items = items[:0]
	}
	pageOf(c, items)
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
	var contents []model.Content
	if err := query.Find(&contents).Error; err != nil {
		fail(c, http.StatusInternalServerError, "portal.resources_failed", "公开资源暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(contents))
	for _, content := range contents {
		item := gin.H{"id": content.ID, "title": content.Title, "description": content.Excerpt, "kind": "document", "size_bytes": 0, "updated_at": content.UpdatedAt, "download_url": "#"}
		if kind == "" || kind == "document" {
			items = append(items, item)
		}
	}
	if kind != "" && kind != "document" {
		items = nil
	}
	pageOf(c, items)
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
		query = query.Where("title LIKE ? OR excerpt LIKE ?", "%"+category+"%", "%"+category+"%")
	}
	if q != "" {
		query = query.Where("title LIKE ? OR excerpt LIKE ? OR body LIKE ?", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	var contents []model.Content
	if err := query.Find(&contents).Error; err != nil {
		fail(c, http.StatusInternalServerError, "portal.knowledge_failed", "公开知识库暂时无法加载。")
		return
	}
	items := make([]gin.H, 0, len(contents))
	for _, content := range contents {
		items = append(items, gin.H{"id": content.ID, "title": content.Title, "summary": content.Excerpt, "category": "知识库", "updated_at": content.UpdatedAt, "reading_minutes": maxInt(1, len([]rune(content.Body))/900+1)})
	}
	pageOf(c, items)
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
	var published, total int64
	h.db.Model(&model.Content{}).Where("organization_id = ? AND status = ?", principal.OrganizationID, "published").Count(&published)
	h.db.Model(&model.Content{}).Where("organization_id = ?", principal.OrganizationID).Count(&total)
	var recent []model.Content
	h.db.Where("organization_id = ?", principal.OrganizationID).Order("updated_at DESC").Limit(12).Find(&recent)
	recentItems := make([]gin.H, 0, len(recent))
	for _, item := range recent {
		recentItems = append(recentItems, contentAdminItem(item, h.db))
	}
	respond(c, http.StatusOK, gin.H{"organization_name": organization.Name, "updated_at": time.Now().UTC(), "metrics": []gin.H{{"label": "活跃成员", "value": 1, "change": "开发环境", "tone": "primary"}, {"label": "已发布内容", "value": published, "change": "当前公开内容", "tone": "secondary"}, {"label": "内容总数", "value": total, "change": "含草稿", "tone": "neutral"}, {"label": "在线玩家", "value": 18, "change": "服务器状态正常", "tone": "neutral"}}, "pending_applications": applications(), "recent_content": recentItems, "server": serverStatus()})
}
func contentItems() []gin.H {
	return []gin.H{{"id": "content_001", "title": "QUTCraft CMS 项目正式启动", "type": "news", "status": "published", "author": "QUTCraft Admin", "updated_at": "2026-07-17T03:00:00Z"}}
}
func applications() []gin.H {
	return []gin.H{{"id": "application_001", "applicant": "Yukino", "type": "whitelist", "submitted_at": "2026-07-17T02:30:00Z", "note": "希望参与周末建筑测试。", "status": "pending"}}
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
func (h *WorkspaceHandler) AdminCreateContent(c *gin.Context) {
	var body struct {
		Title   string `json:"title"`
		Type    string `json:"type"`
		Excerpt string `json:"excerpt"`
		Body    string `json:"body"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Title == "" {
		fail(c, http.StatusBadRequest, "content.validation_failed", "内容标题不能为空。")
		return
	}
	if len([]rune(body.Title)) > 160 || len([]rune(body.Excerpt)) > 500 || (body.Type != "news" && body.Type != "resource" && body.Type != "knowledge") {
		fail(c, http.StatusBadRequest, "content.validation_failed", "title 最长 160 字符，type 必须为 news、resource 或 knowledge。")
		return
	}
	principal, _ := middleware.PrincipalFromContext(c)
	content := model.Content{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, AuthorUserID: principal.UserID, Title: body.Title, Type: body.Type, Status: "draft", Excerpt: body.Excerpt, Body: body.Body}
	if err := h.db.Create(&content).Error; err != nil {
		fail(c, http.StatusInternalServerError, "content.create_failed", "内容草稿创建失败。")
		return
	}
	respond(c, http.StatusCreated, contentAdminItem(content, h.db))
}

func (h *WorkspaceHandler) AdminUpdateContent(c *gin.Context) {
	var body struct {
		Title   string `json:"title"`
		Type    string `json:"type"`
		Excerpt string `json:"excerpt"`
		Body    string `json:"body"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, "content.validation_failed", "内容格式不正确。")
		return
	}
	if strings.TrimSpace(body.Title) == "" || len([]rune(body.Title)) > 160 || len([]rune(body.Excerpt)) > 500 || (body.Type != "news" && body.Type != "resource" && body.Type != "knowledge") {
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
	content.Title, content.Type, content.Excerpt, content.Body = strings.TrimSpace(body.Title), body.Type, body.Excerpt, body.Body
	if err := h.db.Save(&content).Error; err != nil {
		fail(c, http.StatusInternalServerError, "content.update_failed", "内容保存失败。")
		return
	}
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
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "content." + status, TargetType: "content", TargetID: content.ID, Result: "success", RequestID: ensureRequestID(c)}).Error
	respond(c, http.StatusOK, contentAdminItem(content, h.db))
}

func contentPublicItem(content model.Content) gin.H {
	publishedAt := content.PublishedAt
	if publishedAt == nil {
		publishedAt = &content.UpdatedAt
	}
	return gin.H{"id": content.ID, "title": content.Title, "excerpt": content.Excerpt, "category": content.Type, "published_at": publishedAt, "reading_minutes": maxInt(1, len([]rune(content.Body))/900+1)}
}

func contentAdminItem(content model.Content, db *gorm.DB) gin.H {
	var author model.User
	_ = db.First(&author, "id = ?", content.AuthorUserID).Error
	return gin.H{"id": content.ID, "title": content.Title, "type": content.Type, "status": content.Status, "author": author.DisplayName, "excerpt": content.Excerpt, "body": content.Body, "published_at": content.PublishedAt, "updated_at": content.UpdatedAt}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (h *WorkspaceHandler) AdminUsers(c *gin.Context) {
	pageOf(c, []gin.H{{"id": "bootstrap-admin", "name": "QUTCraft Admin", "email": "admin@qutcraft.local", "role": "owner", "state": "active", "joined_at": "2026-07-14T01:00:00Z"}})
}
func (h *WorkspaceHandler) AdminApplications(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	items := applications()
	for _, item := range items {
		item["status"] = h.applicationStates[item["id"].(string)]
	}
	pageOf(c, items)
}
func (h *WorkspaceHandler) AdminApplicationDecision(c *gin.Context) {
	decision := c.Param("id")
	next := "approved"
	if strings.HasSuffix(c.FullPath(), "/reject") {
		next = "rejected"
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	current, exists := h.applicationStates[decision]
	if !exists {
		fail(c, http.StatusNotFound, "application.not_found", "申请不存在。")
		return
	}
	if current != "pending" {
		fail(c, http.StatusConflict, "application.already_decided", "申请已经处理，不能重复审批。")
		return
	}
	h.applicationStates[decision] = next
	principal, _ := middleware.PrincipalFromContext(c)
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "application." + next, TargetType: "application", TargetID: decision, Result: "success", RequestID: ensureRequestID(c)}).Error
	respond(c, http.StatusOK, gin.H{"id": decision, "applicant": "Yukino", "type": "whitelist", "submitted_at": "2026-07-17T02:30:00Z", "note": "希望参与周末建筑测试。", "status": next})
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

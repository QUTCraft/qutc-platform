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

// WorkspaceHandler supplies the first usable read-model for the portal and
// management UI.  Content management persistence is introduced in the next
// milestone; these development fixtures deliberately keep the API contract
// available while the domain tables are being designed.
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
	items := []gin.H{{"id": "post_cms", "title": "QUTCraft CMS 项目正式启动", "excerpt": "从官网、资源分发到服务器适配，我们开始建设可持续的公共入口。", "category": "社团动态", "published_at": "2026-07-14T12:00:00Z", "reading_minutes": 4}}
	if category != "" {
		filtered := items[:0]
		for _, item := range items {
			if item["category"] == category {
				filtered = append(filtered, item)
			}
		}
		items = filtered
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
	items := []gin.H{{"id": "resource_overview", "title": "QUTCraft CMS 产品说明", "description": "项目目标、门户范围与 MVP 路线。", "kind": "document", "size_bytes": 2600000, "updated_at": "2026-07-17T01:00:00Z", "download_url": "#"}}
	if kind != "" && items[0]["kind"] != kind || q != "" && !strings.Contains(strings.ToLower(items[0]["title"].(string)+items[0]["description"].(string)), strings.ToLower(q)) {
		items = items[:0]
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
	items := []gin.H{{"id": "knowledge_handoff", "title": "如何让社团项目可交接", "summary": "建立不依赖个人记忆的项目协作方式。", "category": "项目协作", "updated_at": "2026-07-16T02:00:00Z", "reading_minutes": 8}}
	if category != "" && items[0]["category"] != category || q != "" && !strings.Contains(strings.ToLower(items[0]["title"].(string)+items[0]["summary"].(string)), strings.ToLower(q)) {
		items = items[:0]
	}
	pageOf(c, items)
}
func (h *WorkspaceHandler) PortalServer(c *gin.Context) {
	respond(c, http.StatusOK, gin.H{"enabled": true, "label": "QUTCraft Java 生存服", "state": "online", "version": "Java 1.21.x", "online_players": 18, "max_players": 60, "updated_at": time.Now().UTC(), "apply_url": "#join"})
}

func (h *WorkspaceHandler) AdminDashboard(c *gin.Context) {
	respond(c, http.StatusOK, gin.H{"organization_name": "QUTCraft Commons", "updated_at": time.Now().UTC(), "metrics": []gin.H{{"label": "活跃成员", "value": 1, "change": "开发环境", "tone": "primary"}, {"label": "已发布内容", "value": 1, "change": "开发数据", "tone": "secondary"}, {"label": "待处理申请", "value": 1, "change": "需要处理", "tone": "warning"}, {"label": "在线玩家", "value": 18, "change": "服务器状态正常", "tone": "neutral"}}, "pending_applications": applications(), "recent_content": contentItems(), "server": serverStatus()})
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
func (h *WorkspaceHandler) AdminContent(c *gin.Context) { pageOf(c, contentItems()) }
func (h *WorkspaceHandler) AdminCreateContent(c *gin.Context) {
	var body struct {
		Title string `json:"title"`
		Type  string `json:"type"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Title == "" {
		fail(c, http.StatusBadRequest, "content.validation_failed", "内容标题不能为空。")
		return
	}
	if len([]rune(body.Title)) > 160 || (body.Type != "news" && body.Type != "resource" && body.Type != "knowledge") {
		fail(c, http.StatusBadRequest, "content.validation_failed", "title 最长 160 字符，type 必须为 news、resource 或 knowledge。")
		return
	}
	respond(c, http.StatusCreated, gin.H{"id": uuid.NewString(), "title": body.Title, "type": body.Type, "status": "draft", "author": "QUTCraft Admin", "updated_at": time.Now().UTC()})
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
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "application." + next, TargetType: "application", TargetID: decision, Result: "success", RequestID: c.GetHeader("X-Request-ID")}).Error
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
	_ = h.db.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: "server.command", TargetType: "server", Result: "accepted", RequestID: c.GetHeader("X-Request-ID"), CreatedAt: now}).Error
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

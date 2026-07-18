package handler

import (
	"net/http"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkspaceHandler supplies the first usable read-model for the portal and
// management UI.  Content management persistence is introduced in the next
// milestone; these development fixtures deliberately keep the API contract
// available while the domain tables are being designed.
type WorkspaceHandler struct{ db *gorm.DB }

func NewWorkspaceHandler(db *gorm.DB) *WorkspaceHandler { return &WorkspaceHandler{db: db} }

func (h *WorkspaceHandler) Organization(c *gin.Context) {
	var org model.Organization
	if err := h.db.Where("slug = ?", c.Param("slug")).First(&org).Error; err != nil {
		fail(c, http.StatusNotFound, "portal.organization_not_found", "组织不存在或未公开。")
		return
	}
	respond(c, http.StatusOK, gin.H{"id": org.ID, "slug": org.Slug, "name": org.Name, "short_name": "QUTCraft", "tagline": "把社团正在发生的事，认真地呈现出来。", "introduction": "QUTCraft 是青岛理工大学的 Minecraft 社团，持续建设内容、项目与公共知识资产。", "contact_email": "contact@qutcraft.example", "social_links": []gin.H{{"label": "GitHub", "href": "https://github.com/QUTCraft/qutc-platform"}}})
}

func (h *WorkspaceHandler) PortalPosts(c *gin.Context) {
	respond(c, http.StatusOK, []gin.H{{"id": "post_cms", "title": "QUTCraft CMS 项目正式启动", "excerpt": "从官网、资源分发到服务器适配，我们开始建设可持续的公共入口。", "category": "社团动态", "published_at": "2026-07-14T12:00:00Z", "reading_minutes": 4}})
}
func (h *WorkspaceHandler) PortalProjects(c *gin.Context) {
	respond(c, http.StatusOK, []gin.H{{"id": "project_cms", "title": "QUTCraft CMS", "summary": "面向校园社团与民间组织的公开门户与内容分发系统。", "status": "active", "tags": []string{"Vue 3", "Go", "API-first"}, "updated_at": "2026-07-17T03:00:00Z"}})
}
func (h *WorkspaceHandler) PortalResources(c *gin.Context) {
	respond(c, http.StatusOK, []gin.H{{"id": "resource_overview", "title": "QUTCraft CMS 产品说明", "description": "项目目标、门户范围与 MVP 路线。", "kind": "document", "size_bytes": 2600000, "updated_at": "2026-07-17T01:00:00Z", "download_url": "#"}})
}
func (h *WorkspaceHandler) PortalKnowledge(c *gin.Context) {
	respond(c, http.StatusOK, []gin.H{{"id": "knowledge_handoff", "title": "如何让社团项目可交接", "summary": "建立不依赖个人记忆的项目协作方式。", "category": "项目协作", "updated_at": "2026-07-16T02:00:00Z", "reading_minutes": 8}})
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
func (h *WorkspaceHandler) AdminContent(c *gin.Context) { respond(c, http.StatusOK, contentItems()) }
func (h *WorkspaceHandler) AdminCreateContent(c *gin.Context) {
	var body struct {
		Title string `json:"title"`
		Type  string `json:"type"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Title == "" {
		fail(c, http.StatusBadRequest, "content.validation_failed", "内容标题不能为空。")
		return
	}
	respond(c, http.StatusCreated, gin.H{"id": uuid.NewString(), "title": body.Title, "type": body.Type, "status": "draft", "author": "QUTCraft Admin", "updated_at": time.Now().UTC()})
}
func (h *WorkspaceHandler) AdminUsers(c *gin.Context) {
	respond(c, http.StatusOK, []gin.H{{"id": "bootstrap-admin", "name": "QUTCraft Admin", "email": "admin@qutcraft.local", "role": "owner", "state": "active", "joined_at": "2026-07-14T01:00:00Z"}})
}
func (h *WorkspaceHandler) AdminApplications(c *gin.Context) {
	respond(c, http.StatusOK, applications())
}
func (h *WorkspaceHandler) AdminApplicationDecision(c *gin.Context) {
	respond(c, http.StatusOK, gin.H{"id": c.Param("id"), "applicant": "Yukino", "type": "whitelist", "submitted_at": "2026-07-17T02:30:00Z", "note": "希望参与周末建筑测试。", "status": "approved"})
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
	respond(c, http.StatusAccepted, gin.H{"accepted": true, "message": "开发环境已记录命令，未连接真实 RCON。", "executed_at": time.Now().UTC()})
}

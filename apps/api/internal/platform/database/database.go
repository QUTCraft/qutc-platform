package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/config"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Connect(cfg config.Config) (*gorm.DB, error) {
	deadline := time.Now().Add(45 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
		if err == nil {
			sqlDB, dbErr := db.DB()
			if dbErr == nil {
				if pingErr := sqlDB.Ping(); pingErr == nil {
					return db, nil
				} else {
					lastErr = pingErr
				}
			} else {
				lastErr = dbErr
			}
		} else {
			lastErr = err
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("connect mysql after retry: %w", lastErr)
}

func MigrateAndSeed(db *gorm.DB, cfg config.Config) error {
	if err := runMigrations(db); err != nil {
		return err
	}

	organization, err := findOrCreateOrganization(db, cfg)
	if err != nil {
		return err
	}
	if err := seedRBAC(db); err != nil {
		return err
	}
	if err := seedAgentDefinitions(db, organization); err != nil {
		return err
	}
	if err := seedBootstrapOwner(db, cfg, organization); err != nil {
		return err
	}
	if cfg.DemoSeedEnabled {
		if err := SeedDemoData(db, cfg, organization); err != nil {
			return err
		}
	}
	return nil
}

// SeedDemoData creates only stable, development-owned fixtures that are
// missing. It never updates or deletes existing records and is disabled unless
// DEMO_SEED_ENABLED is explicitly true.
func SeedDemoData(db *gorm.DB, cfg config.Config, organization model.Organization) error {
	if cfg.BootstrapAdminEmail == "" {
		return fmt.Errorf("demo seed requires BOOTSTRAP_ADMIN_EMAIL")
	}
	if err := seedKnowledgeDirectories(db, organization); err != nil {
		return err
	}
	if err := seedContent(db, cfg, organization); err != nil {
		return err
	}
	if err := seedProjects(db, cfg, organization); err != nil {
		return err
	}
	return seedApplications(db, cfg, organization)
}

func findOrCreateOrganization(db *gorm.DB, cfg config.Config) (model.Organization, error) {
	slug := cfg.DefaultOrganizationSlug
	var organization model.Organization
	err := db.Where("slug = ?", slug).First(&organization).Error
	if err == nil {
		if organization.ShortName == "" {
			organization.ShortName = organization.Name
			organization.SocialLinksJSON = "[]"
			organization.IsPublic = true
			if slug == "qutcraft" {
				organization.ShortName = "QUTCraft"
				organization.Tagline = "把社团正在发生的事，认真地呈现出来。"
				organization.Introduction = "QUTCraft 是青岛理工大学的 Minecraft 社团，持续建设内容、项目与公共知识资产。"
				organization.ContactEmail = "contact@qutcraft.example"
				organization.SocialLinksJSON = `[{"label":"GitHub","href":"https://github.com/QUTCraft/qutc-platform"}]`
			}
			if saveErr := db.Save(&organization).Error; saveErr != nil {
				return model.Organization{}, fmt.Errorf("backfill organization profile: %w", saveErr)
			}
		}
		return organization, nil
	}
	if err != gorm.ErrRecordNotFound {
		return model.Organization{}, fmt.Errorf("find organization: %w", err)
	}
	organization = model.Organization{ID: uuid.NewString(), Slug: slug, Name: slug, ShortName: slug, SocialLinksJSON: "[]", IsPublic: true}
	if cfg.DemoSeedProfile == "generic" {
		organization.Name = "Campus Commons"
		organization.ShortName = "Commons"
		organization.Tagline = "让组织信息、协作与公共内容持续流动。"
		organization.Introduction = "面向校园社团与民间组织的公共门户、内容分发和协作平台。"
	} else if slug == "qutcraft" {
		organization.Name = "QUTCraft Commons"
		organization.ShortName = "QUTCraft"
		organization.Tagline = "把社团正在发生的事，认真地呈现出来。"
		organization.Introduction = "QUTCraft 是青岛理工大学的 Minecraft 社团，持续建设内容、项目与公共知识资产。"
		organization.ContactEmail = "contact@qutcraft.example"
		organization.SocialLinksJSON = `[{"label":"GitHub","href":"https://github.com/QUTCraft/qutc-platform"}]`
	}
	if err := db.Create(&organization).Error; err != nil {
		return model.Organization{}, fmt.Errorf("create organization: %w", err)
	}
	return organization, nil
}

func seedRBAC(db *gorm.DB) error {
	permissions := map[string]string{
		"organization:read": "查看组织内部基础信息", "content:read": "查看内容工作区", "content:create": "创建内容草稿",
		"content:update": "编辑内容草稿", "content:submit": "提交内容审核", "content:publish": "发布或下线内容", "content:archive": "下线内容",
		"asset:read": "查看媒体资源", "asset:upload": "上传媒体资源", "asset:manage": "管理媒体资源", "membership:read": "查看成员目录", "membership:manage": "管理成员与角色",
		"project:read": "查看项目工作区", "project:manage": "管理项目、成员与里程碑",
		"knowledge:read": "查看知识库目录", "knowledge:manage": "管理知识库目录",
		"application:read": "查看申请", "application:approve": "处理申请", "server:read_status": "查看服务器后台状态",
		"server:command": "执行受限服务器命令", "organization:configure": "配置组织与门户", "audit:read": "查看审计记录",
		"ai:use": "使用组织运营智能体",
	}
	permissionIDs := map[string]string{}
	for key, displayName := range permissions {
		permission, err := findOrCreatePermission(db, key, displayName)
		if err != nil {
			return err
		}
		permissionIDs[key] = permission.ID
	}

	rolePermissions := map[string][]string{
		"member":        {"organization:read"},
		"editor":        {"organization:read", "content:read", "content:create", "content:update", "content:submit", "asset:read", "asset:upload", "project:read", "knowledge:read", "ai:use"},
		"administrator": {"organization:read", "content:read", "content:create", "content:update", "content:submit", "content:publish", "content:archive", "asset:read", "asset:upload", "asset:manage", "membership:read", "membership:manage", "project:read", "project:manage", "knowledge:read", "knowledge:manage", "application:read", "application:approve", "server:read_status", "audit:read", "ai:use"},
		"owner":         {"organization:read", "content:read", "content:create", "content:update", "content:submit", "content:publish", "content:archive", "asset:read", "asset:upload", "asset:manage", "membership:read", "membership:manage", "project:read", "project:manage", "knowledge:read", "knowledge:manage", "application:read", "application:approve", "server:read_status", "server:command", "organization:configure", "audit:read", "ai:use"},
	}
	for key, keys := range rolePermissions {
		role, err := findOrCreateRole(db, key, strings.ToUpper(key[:1])+key[1:])
		if err != nil {
			return err
		}
		for _, permissionKey := range keys {
			var link model.RolePermission
			err := db.Where("role_id = ? AND permission_id = ?", role.ID, permissionIDs[permissionKey]).First(&link).Error
			if err == gorm.ErrRecordNotFound {
				link = model.RolePermission{RoleID: role.ID, PermissionID: permissionIDs[permissionKey]}
				err = db.Create(&link).Error
			}
			if err != nil {
				return fmt.Errorf("seed role permission %s/%s: %w", key, permissionKey, err)
			}
		}
	}
	return nil
}

func seedAgentDefinitions(db *gorm.DB, organization model.Organization) error {
	definitions := []model.AgentDefinition{
		{
			ID: uuid.NewString(), OrganizationID: organization.ID, Key: "content-copilot",
			Name: "内容协作智能体", Purpose: "根据当前组织内已授权的知识资料生成带引用的 Markdown 内容提案；结果必须由人工确认。",
			SystemPolicyVersion: "content-copilot/v1", AllowedToolKeys: `["knowledge.search","knowledge.read"]`,
			ModelProfile: "content-generation", Enabled: true,
		},
		{
			ID: uuid.NewString(), OrganizationID: organization.ID, Key: "activity-planner",
			Name: "校园活动策划智能体", Purpose: "根据活动约束与组织知识生成带引用的可执行活动方案，并提出须由人工批准的项目、里程碑和公告草稿。",
			SystemPolicyVersion: "activity-planner/v2", AllowedToolKeys: `["knowledge.search","knowledge.read","project.create_proposal","milestone.create_proposal","content.create_draft_proposal"]`,
			ModelProfile: "activity-planning", Enabled: true,
		},
	}
	for _, definition := range definitions {
		var existing model.AgentDefinition
		err := db.Where("organization_id = ? AND `key` = ?", organization.ID, definition.Key).First(&existing).Error
		if err == nil {
			if err := db.Model(&model.AgentDefinition{}).Where("id = ?", existing.ID).Updates(map[string]any{
				"name": definition.Name, "purpose": definition.Purpose,
				"system_policy_version": definition.SystemPolicyVersion,
				"allowed_tool_keys":     definition.AllowedToolKeys, "model_profile": definition.ModelProfile,
			}).Error; err != nil {
				return fmt.Errorf("update agent definition %s: %w", definition.Key, err)
			}
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("find agent definition %s: %w", definition.Key, err)
		}
		if err := db.Create(&definition).Error; err != nil {
			return fmt.Errorf("seed agent definition %s: %w", definition.Key, err)
		}
	}
	return nil
}

func findOrCreatePermission(db *gorm.DB, key, displayName string) (model.Permission, error) {
	var permission model.Permission
	err := db.Where("`key` = ?", key).First(&permission).Error
	if err == nil {
		return permission, nil
	}
	if err != gorm.ErrRecordNotFound {
		return model.Permission{}, fmt.Errorf("find permission %s: %w", key, err)
	}
	permission = model.Permission{ID: uuid.NewString(), Key: key, DisplayName: displayName}
	if err := db.Create(&permission).Error; err != nil {
		return model.Permission{}, fmt.Errorf("create permission %s: %w", key, err)
	}
	return permission, nil
}

func findOrCreateRole(db *gorm.DB, key, displayName string) (model.Role, error) {
	var role model.Role
	err := db.Where("`key` = ?", key).First(&role).Error
	if err == nil {
		return role, nil
	}
	if err != gorm.ErrRecordNotFound {
		return model.Role{}, fmt.Errorf("find role %s: %w", key, err)
	}
	role = model.Role{ID: uuid.NewString(), Key: key, DisplayName: displayName}
	if err := db.Create(&role).Error; err != nil {
		return model.Role{}, fmt.Errorf("create role %s: %w", key, err)
	}
	return role, nil
}

func seedBootstrapOwner(db *gorm.DB, cfg config.Config, organization model.Organization) error {
	if cfg.BootstrapAdminEmail == "" || cfg.BootstrapAdminPassword == "" {
		return nil
	}
	var user model.User
	err := db.Where("email = ?", cfg.BootstrapAdminEmail).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(cfg.BootstrapAdminPassword), bcrypt.DefaultCost)
		if hashErr != nil {
			return fmt.Errorf("hash bootstrap password: %w", hashErr)
		}
		user = model.User{ID: uuid.NewString(), Email: cfg.BootstrapAdminEmail, DisplayName: cfg.BootstrapAdminName, PasswordHash: string(hash), State: "active"}
		if err := db.Create(&user).Error; err != nil {
			return fmt.Errorf("create bootstrap user: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find bootstrap user: %w", err)
	}

	var membership model.Membership
	err = db.Where("organization_id = ? AND user_id = ?", organization.ID, user.ID).First(&membership).Error
	if err == gorm.ErrRecordNotFound {
		membership = model.Membership{ID: uuid.NewString(), OrganizationID: organization.ID, UserID: user.ID, State: "active"}
		err = db.Create(&membership).Error
	}
	if err != nil {
		return fmt.Errorf("create bootstrap membership: %w", err)
	}
	var membershipEvent model.MembershipEvent
	eventErr := db.Where("membership_id = ? AND state = ? AND reason = ?", membership.ID, "active", "bootstrap").First(&membershipEvent).Error
	if eventErr == gorm.ErrRecordNotFound {
		if err := db.Create(&model.MembershipEvent{ID: uuid.NewString(), MembershipID: membership.ID, State: "active", Reason: "bootstrap"}).Error; err != nil {
			return fmt.Errorf("create bootstrap membership event: %w", err)
		}
	} else if eventErr != nil {
		return fmt.Errorf("find bootstrap membership event: %w", eventErr)
	}
	var owner model.Role
	if err := db.Where("`key` = ?", "owner").First(&owner).Error; err != nil {
		return fmt.Errorf("find owner role: %w", err)
	}
	var link model.MembershipRole
	err = db.Where("membership_id = ? AND role_id = ?", membership.ID, owner.ID).First(&link).Error
	if err == gorm.ErrRecordNotFound {
		link = model.MembershipRole{MembershipID: membership.ID, RoleID: owner.ID}
		err = db.Create(&link).Error
	}
	if err != nil {
		return fmt.Errorf("assign bootstrap owner role: %w", err)
	}
	return nil
}

func seedContent(db *gorm.DB, cfg config.Config, organization model.Organization) error {
	var user model.User
	if err := db.Where("email = ?", cfg.BootstrapAdminEmail).First(&user).Error; err != nil {
		return fmt.Errorf("find content seed author: %w", err)
	}
	publishedAt := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	items := []model.Content{
		{ID: "content_cms", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "QUTCraft CMS 项目正式启动", Type: "news", Category: "社团动态", Status: "published", Excerpt: "从官网、资源分发到服务器适配，我们开始建设可持续的公共入口。", Body: "# QUTCraft CMS 项目正式启动\n\n我们正在建设一个连接公开门户、内容管理与组织协作的公共平台。\n\n- 门户只展示已发布内容\n- 后台负责内容与成员协作\n- 外部系统通过受控 Adapter 接入", PublishedAt: &publishedAt},
		{ID: "content_build", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "暑期建筑活动报名", Type: "news", Category: "活动", Status: "draft", Excerpt: "面向成员开放的活动草稿。", Body: "该内容尚未发布，只能在后台看到。"},
		{ID: "content_resource", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "QUTCraft CMS 产品说明", Type: "resource", Category: "document", Status: "published", Excerpt: "项目目标、门户范围与 MVP 路线。", Body: "QUTCraft CMS 的公开产品说明与接入资料。", PublishedAt: &publishedAt},
		{ID: "content_knowledge", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "如何让社团项目可交接", Type: "knowledge", Category: "项目协作", KnowledgeDirectoryID: stringPointer("knowledge_directory_collaboration"), Status: "published", Excerpt: "建立不依赖个人记忆的项目协作方式。", Body: "从目标、角色、决策记录到发布节奏，建立可持续的知识库。", PublishedAt: &publishedAt},
	}
	if cfg.DemoSeedProfile == "generic" {
		items = []model.Content{
			{ID: "content_cms", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "组织数字化工作台正式启用", Type: "news", Category: "组织动态", Status: "published", Excerpt: "统一管理公共内容、项目进展、成员协作和知识资产。", Body: "# 组织数字化工作台正式启用\n\n我们开始使用统一平台维护公共门户、内容和组织协作。", PublishedAt: &publishedAt},
			{ID: "content_build", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "暑期开放活动报名", Type: "news", Category: "活动", Status: "draft", Excerpt: "面向成员开放的活动草稿。", Body: "该内容尚未发布，只能在后台看到。"},
			{ID: "content_resource", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "组织协作平台产品说明", Type: "resource", Category: "document", Status: "published", Excerpt: "公共门户、内容分发与协作管理的产品范围。", Body: "面向校园社团与民间组织的通用产品说明。", PublishedAt: &publishedAt},
			{ID: "content_knowledge", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "如何让组织项目可交接", Type: "knowledge", Category: "项目协作", KnowledgeDirectoryID: stringPointer("knowledge_directory_collaboration"), Status: "published", Excerpt: "建立不依赖个人记忆的项目协作方式。", Body: "从目标、角色、决策记录到发布节奏，建立可持续的知识库。", PublishedAt: &publishedAt},
		}
	}
	for _, item := range items {
		var existing model.Content
		err := db.Where("id = ?", item.ID).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("find seed content %s: %w", item.ID, err)
		}
		if err := db.Create(&item).Error; err != nil {
			return fmt.Errorf("seed content %s: %w", item.ID, err)
		}
	}
	return nil
}

func seedProjects(db *gorm.DB, cfg config.Config, organization model.Organization) error {
	var user model.User
	if err := db.Where("email = ?", cfg.BootstrapAdminEmail).First(&user).Error; err != nil {
		return fmt.Errorf("find project seed owner: %w", err)
	}
	items := []model.Project{
		{ID: "project_cms", OrganizationID: organization.ID, OwnerUserID: user.ID, Title: "QUTCraft CMS", Summary: "面向校园社团与民间组织的公开门户与内容分发系统，QUTCraft 是首个落地案例。", Status: "active", Tags: "Vue 3,Go,API-first", IsPublic: true},
		{ID: "project_spawn", OrganizationID: organization.ID, OwnerUserID: user.ID, Title: "主城公共区域计划", Summary: "把成员作品、活动路线与社区服务设施组织成一个可以长期生长的起点。", Status: "active", Tags: "建筑,社区共建", IsPublic: true},
		{ID: "project_wiki", OrganizationID: organization.ID, OwnerUserID: user.ID, Title: "社团知识库迁移", Summary: "将散落的经验、规则、活动资料和技术笔记逐步整理为可检索的公共知识库。", Status: "research", Tags: "知识库,信息架构", IsPublic: true},
	}
	if cfg.DemoSeedProfile == "generic" {
		items = []model.Project{
			{ID: "project_cms", OrganizationID: organization.ID, OwnerUserID: user.ID, Title: "组织数字化平台", Summary: "面向校园社团与民间组织的公共门户、内容分发与协作管理系统。", Status: "active", Tags: "Vue 3,Go,API-first", IsPublic: true},
			{ID: "project_spawn", OrganizationID: organization.ID, OwnerUserID: user.ID, Title: "校园开放活动计划", Summary: "组织报名、宣传、现场协作和活动成果归档。", Status: "active", Tags: "活动,公共协作", IsPublic: true},
			{ID: "project_wiki", OrganizationID: organization.ID, OwnerUserID: user.ID, Title: "组织知识库迁移", Summary: "将散落的制度、活动资料和经验整理为可检索、可交接的知识库。", Status: "research", Tags: "知识库,信息架构", IsPublic: true},
		}
	}
	for _, item := range items {
		var existing model.Project
		err := db.Where("id = ?", item.ID).First(&existing).Error
		if err == nil {
			item = existing
		} else if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("find seed project %s: %w", item.ID, err)
		} else if err := db.Create(&item).Error; err != nil {
			return fmt.Errorf("seed project %s: %w", item.ID, err)
		}
		var member model.ProjectMember
		memberErr := db.Where("project_id = ? AND user_id = ?", item.ID, user.ID).First(&member).Error
		if memberErr == gorm.ErrRecordNotFound {
			if err := db.Create(&model.ProjectMember{ProjectID: item.ID, UserID: user.ID, Role: "owner"}).Error; err != nil {
				return fmt.Errorf("seed project owner %s: %w", item.ID, err)
			}
		} else if memberErr != nil {
			return fmt.Errorf("find project owner %s: %w", item.ID, memberErr)
		}
	}
	return seedProjectMilestones(db)
}

func seedProjectMilestones(db *gorm.DB) error {
	dueAPI := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
	dueRelease := time.Date(2026, time.August, 31, 0, 0, 0, 0, time.UTC)
	items := []model.ProjectMilestone{
		{ID: "milestone_demo_api", ProjectID: "project_cms", Title: "完成 API 契约与核心闭环", Status: "active", DueAt: &dueAPI},
		{ID: "milestone_demo_release", ProjectID: "project_cms", Title: "发布比赛演示版本", Status: "planned", DueAt: &dueRelease},
	}
	for _, item := range items {
		var existing model.ProjectMilestone
		err := db.Where("id = ?", item.ID).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("find seed milestone %s: %w", item.ID, err)
		}
		if err := db.Create(&item).Error; err != nil {
			return fmt.Errorf("seed milestone %s: %w", item.ID, err)
		}
	}
	return nil
}

func seedKnowledgeDirectories(db *gorm.DB, organization model.Organization) error {
	items := []model.KnowledgeDirectory{
		{ID: "knowledge_directory_collaboration", OrganizationID: organization.ID, Name: "项目协作", Slug: "collaboration", Description: "项目目标、角色与交接记录。", SortOrder: 10, IsPublic: true},
		{ID: "knowledge_directory_technology", OrganizationID: organization.ID, Name: "技术规范", Slug: "technology", Description: "接口、前端和部署规范。", SortOrder: 20, IsPublic: true},
		{ID: "knowledge_directory_community", OrganizationID: organization.ID, Name: "社团实践", Slug: "community", Description: "适用于组织日常协作的经验。", SortOrder: 30, IsPublic: true},
	}
	for _, item := range items {
		var existing model.KnowledgeDirectory
		err := db.Where("id = ?", item.ID).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("find seed knowledge directory %s: %w", item.ID, err)
		}
		if err := db.Create(&item).Error; err != nil {
			return fmt.Errorf("seed knowledge directory %s: %w", item.ID, err)
		}
	}
	return nil
}

func seedApplications(db *gorm.DB, cfg config.Config, organization model.Organization) error {
	var owner model.User
	if err := db.Where("email = ?", cfg.BootstrapAdminEmail).First(&owner).Error; err != nil {
		return fmt.Errorf("find application seed reviewer: %w", err)
	}

	decidedAt := time.Date(2026, time.July, 28, 10, 0, 0, 0, time.UTC)
	applications := []model.Application{
		{ID: "application_demo", OrganizationID: organization.ID, Type: "whitelist", ClassName: "计算机231", ApplicantName: "Yukino", GameID: "YukinoCraft", QQNumber: "123456789", Email: "yukino@example.com", Note: "希望参与周末建筑测试。", Status: "pending"},
		{ID: "application_demo_approved", OrganizationID: organization.ID, Type: "whitelist", ClassName: "自动化231", ApplicantName: "Dawn", GameID: "DawnBuilder", QQNumber: "223456789", Email: "dawn@example.com", Note: "希望参与公共建筑项目。", Status: "approved", DecidedAt: &decidedAt, DecidedBy: owner.ID, DecisionReason: "资料完整，符合加入要求。"},
		{ID: "application_demo_rejected", OrganizationID: organization.ID, Type: "membership", ClassName: "设计231", ApplicantName: "Nova", GameID: "NovaDesign", QQNumber: "323456789", Email: "nova@example.com", Note: "希望加入内容组。", Status: "rejected", DecidedAt: &decidedAt, DecidedBy: owner.ID, DecisionReason: "申请资料需要补充作品说明。"},
	}
	for _, application := range applications {
		var existing model.Application
		err := db.Where("id = ?", application.ID).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("find seed application %s: %w", application.ID, err)
		}
		if err := db.Create(&application).Error; err != nil {
			return fmt.Errorf("seed application %s: %w", application.ID, err)
		}
	}

	completedAt := decidedAt.Add(2 * time.Second)
	syncRecord := model.ApplicationServerSync{
		ID:             "application_sync_demo_approved",
		OrganizationID: organization.ID,
		ApplicationID:  "application_demo_approved",
		Operation:      "whitelist.add",
		Adapter:        "minecraft-mock",
		Mode:           "mock",
		Status:         "succeeded",
		Attempts:       1,
		Message:        "演示 Mock 已记录白名单同步，未连接真实 RCON。",
		RequestedAt:    decidedAt,
		CompletedAt:    &completedAt,
	}
	var existingSync model.ApplicationServerSync
	err := db.Where("id = ?", syncRecord.ID).First(&existingSync).Error
	if err == gorm.ErrRecordNotFound {
		if err := db.Create(&syncRecord).Error; err != nil {
			return fmt.Errorf("seed application sync: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find seed application sync: %w", err)
	}
	return nil
}

func stringPointer(value string) *string {
	return &value
}

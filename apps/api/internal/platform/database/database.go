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
	if err := db.AutoMigrate(
		&model.Organization{}, &model.User{}, &model.Role{}, &model.Permission{},
		&model.RolePermission{}, &model.Membership{}, &model.MembershipEvent{}, &model.Invitation{}, &model.MembershipRole{},
		&model.RefreshToken{}, &model.AuditEvent{}, &model.Content{}, &model.KnowledgeDirectory{}, &model.MediaAsset{},
		&model.Project{}, &model.ProjectMember{}, &model.ProjectMilestone{},
		&model.Application{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	organization, err := findOrCreateOrganization(db, cfg.DefaultOrganizationSlug)
	if err != nil {
		return err
	}
	if err := seedRBAC(db); err != nil {
		return err
	}
	if err := seedBootstrapOwner(db, cfg, organization); err != nil {
		return err
	}
	if err := seedContent(db, cfg, organization); err != nil {
		return err
	}
	if err := seedKnowledgeDirectories(db, organization); err != nil {
		return err
	}
	if err := seedProjects(db, cfg, organization); err != nil {
		return err
	}
	return seedApplications(db, organization)
}

func findOrCreateOrganization(db *gorm.DB, slug string) (model.Organization, error) {
	var organization model.Organization
	err := db.Where("slug = ?", slug).First(&organization).Error
	if err == nil {
		return organization, nil
	}
	if err != gorm.ErrRecordNotFound {
		return model.Organization{}, fmt.Errorf("find organization: %w", err)
	}
	organization = model.Organization{ID: uuid.NewString(), Slug: slug, Name: "QUTCraft Commons"}
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
		"editor":        {"organization:read", "content:read", "content:create", "content:update", "content:submit", "asset:read", "asset:upload", "project:read", "knowledge:read"},
		"administrator": {"organization:read", "content:read", "content:create", "content:update", "content:submit", "content:publish", "content:archive", "asset:read", "asset:upload", "asset:manage", "membership:read", "membership:manage", "project:read", "project:manage", "knowledge:read", "knowledge:manage", "application:read", "application:approve", "server:read_status", "audit:read"},
		"owner":         {"organization:read", "content:read", "content:create", "content:update", "content:submit", "content:publish", "content:archive", "asset:read", "asset:upload", "asset:manage", "membership:read", "membership:manage", "project:read", "project:manage", "knowledge:read", "knowledge:manage", "application:read", "application:approve", "server:read_status", "server:command", "organization:configure", "audit:read"},
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
		{ID: "content_cms", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "QUTCraft CMS 项目正式启动", Type: "news", Category: "社团动态", Status: "published", Excerpt: "从官网、资源分发到服务器适配，我们开始建设可持续的公共入口。", Body: "QUTCraft CMS 内容闭环演示内容。", PublishedAt: &publishedAt},
		{ID: "content_build", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "暑期建筑活动报名", Type: "news", Category: "活动", Status: "draft", Excerpt: "面向成员开放的活动草稿。", Body: "该内容尚未发布，只能在后台看到。"},
		{ID: "content_resource", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "QUTCraft CMS 产品说明", Type: "resource", Category: "document", Status: "published", Excerpt: "项目目标、门户范围与 MVP 路线。", Body: "QUTCraft CMS 的公开产品说明与接入资料。", PublishedAt: &publishedAt},
		{ID: "content_knowledge", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "如何让社团项目可交接", Type: "knowledge", Category: "项目协作", Status: "published", Excerpt: "建立不依赖个人记忆的项目协作方式。", Body: "从目标、角色、决策记录到发布节奏，建立可持续的知识库。", PublishedAt: &publishedAt},
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

func seedApplications(db *gorm.DB, organization model.Organization) error {
	application := model.Application{
		ID:             "application_demo",
		OrganizationID: organization.ID,
		Type:           "whitelist",
		ClassName:      "计算机231",
		ApplicantName:  "Yukino",
		GameID:         "YukinoCraft",
		QQNumber:       "123456789",
		Email:          "yukino@example.com",
		Note:           "希望参与周末建筑测试。",
		Status:         "pending",
	}
	var existing model.Application
	err := db.Where("id = ?", application.ID).First(&existing).Error
	if err == nil {
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("find seed application: %w", err)
	}
	if err := db.Create(&application).Error; err != nil {
		return fmt.Errorf("seed application: %w", err)
	}
	return nil
}

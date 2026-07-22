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
		&model.RolePermission{}, &model.Membership{}, &model.MembershipRole{},
		&model.RefreshToken{}, &model.AuditEvent{}, &model.Content{}, &model.MediaAsset{},
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
	return seedContent(db, cfg, organization)
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
		"editor":        {"organization:read", "content:read", "content:create", "content:update", "content:submit", "asset:read", "asset:upload"},
		"administrator": {"organization:read", "content:read", "content:create", "content:update", "content:submit", "content:publish", "content:archive", "asset:read", "asset:upload", "asset:manage", "membership:read", "membership:manage", "application:read", "application:approve", "server:read_status", "audit:read"},
		"owner":         {"organization:read", "content:read", "content:create", "content:update", "content:submit", "content:publish", "content:archive", "asset:read", "asset:upload", "asset:manage", "membership:read", "membership:manage", "application:read", "application:approve", "server:read_status", "server:command", "organization:configure", "audit:read"},
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
		{ID: "content_cms", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "QUTCraft CMS 项目正式启动", Type: "news", Status: "published", Excerpt: "从官网、资源分发到服务器适配，我们开始建设可持续的公共入口。", Body: "QUTCraft CMS 内容闭环演示内容。", PublishedAt: &publishedAt},
		{ID: "content_build", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "暑期建筑活动报名", Type: "news", Status: "draft", Excerpt: "面向成员开放的活动草稿。", Body: "该内容尚未发布，只能在后台看到。"},
		{ID: "content_resource", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "QUTCraft CMS 产品说明", Type: "resource", Status: "published", Excerpt: "项目目标、门户范围与 MVP 路线。", Body: "QUTCraft CMS 的公开产品说明与接入资料。", PublishedAt: &publishedAt},
		{ID: "content_knowledge", OrganizationID: organization.ID, AuthorUserID: user.ID, Title: "如何让社团项目可交接", Type: "knowledge", Status: "published", Excerpt: "建立不依赖个人记忆的项目协作方式。", Body: "从目标、角色、决策记录到发布节奏，建立可持续的知识库。", PublishedAt: &publishedAt},
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

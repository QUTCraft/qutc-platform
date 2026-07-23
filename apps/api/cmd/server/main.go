package main

import (
	"log"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/config"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/handler"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/cache"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/database"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	if err := database.MigrateAndSeed(db, cfg); err != nil {
		log.Fatalf("database migration or seed failed: %v", err)
	}
	publicCache := cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.PublicCacheTTL)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "Idempotency-Key", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
	}))

	authService := service.NewAuthService(db, cfg)
	authHandler := handler.NewAuthHandler(authService)
	workspaceHandler := handler.NewWorkspaceHandler(db, publicCache, cfg.AppEnv)
	router.GET("/healthz", handler.Health)

	v1 := router.Group("/api/v1")
	auth := v1.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout)
	auth.GET("/me", middleware.RequireAuth(authService), authHandler.Me)
	auth.PATCH("/me", middleware.RequireAuth(authService), authHandler.UpdateMe)

	membership := v1.Group("/membership", middleware.RequireAuth(authService))
	membership.GET("/history", workspaceHandler.MembershipHistory)
	membership.POST("/leave", workspaceHandler.LeaveMembership)

	admin := v1.Group("/admin", middleware.RequireAuth(authService))
	admin.GET("/session", middleware.RequirePermission(authService, "organization:read"), authHandler.Me)
	admin.GET("/dashboard", middleware.RequirePermission(authService, "organization:read"), workspaceHandler.AdminDashboard)
	admin.GET("/content", middleware.RequirePermission(authService, "content:read"), workspaceHandler.AdminContent)
	admin.GET("/knowledge/directories", middleware.RequirePermission(authService, "knowledge:read"), workspaceHandler.AdminKnowledgeDirectories)
	admin.POST("/knowledge/directories", middleware.RequirePermission(authService, "knowledge:manage"), workspaceHandler.AdminCreateKnowledgeDirectory)
	admin.PATCH("/knowledge/directories/:id", middleware.RequirePermission(authService, "knowledge:manage"), workspaceHandler.AdminUpdateKnowledgeDirectory)
	admin.POST("/content", middleware.RequirePermission(authService, "content:create"), workspaceHandler.AdminCreateContent)
	admin.PATCH("/content/:id", middleware.RequirePermission(authService, "content:update"), workspaceHandler.AdminUpdateContent)
	admin.POST("/content/:id/publish", middleware.RequirePermission(authService, "content:publish"), workspaceHandler.PublishContent)
	admin.POST("/content/:id/archive", middleware.RequirePermission(authService, "content:archive"), workspaceHandler.ArchiveContent)
	admin.POST("/assets", middleware.RequirePermission(authService, "asset:upload"), workspaceHandler.UploadAsset)
	admin.GET("/assets/:id/download", middleware.RequirePermission(authService, "asset:read"), workspaceHandler.DownloadAsset)
	admin.GET("/users", middleware.RequirePermission(authService, "membership:read"), workspaceHandler.AdminUsers)
	admin.PATCH("/users/:id", middleware.RequirePermission(authService, "membership:manage"), workspaceHandler.AdminUpdateUser)
	admin.GET("/projects", middleware.RequirePermission(authService, "project:read"), workspaceHandler.AdminProjects)
	admin.POST("/projects", middleware.RequirePermission(authService, "project:manage"), workspaceHandler.AdminCreateProject)
	admin.PATCH("/projects/:id", middleware.RequirePermission(authService, "project:manage"), workspaceHandler.AdminUpdateProject)
	admin.GET("/projects/:id/members", middleware.RequirePermission(authService, "project:read"), workspaceHandler.AdminProjectMembers)
	admin.POST("/projects/:id/members", middleware.RequirePermission(authService, "project:manage"), workspaceHandler.AdminAddProjectMember)
	admin.PATCH("/projects/:id/members/:user_id", middleware.RequirePermission(authService, "project:manage"), workspaceHandler.AdminUpdateProjectMember)
	admin.DELETE("/projects/:id/members/:user_id", middleware.RequirePermission(authService, "project:manage"), workspaceHandler.AdminRemoveProjectMember)
	admin.GET("/projects/:id/milestones", middleware.RequirePermission(authService, "project:read"), workspaceHandler.AdminProjectMilestones)
	admin.POST("/projects/:id/milestones", middleware.RequirePermission(authService, "project:manage"), workspaceHandler.AdminCreateProjectMilestone)
	admin.PATCH("/projects/:id/milestones/:milestone_id", middleware.RequirePermission(authService, "project:manage"), workspaceHandler.AdminUpdateProjectMilestone)
	admin.DELETE("/projects/:id/milestones/:milestone_id", middleware.RequirePermission(authService, "project:manage"), workspaceHandler.AdminDeleteProjectMilestone)
	admin.GET("/applications", middleware.RequirePermission(authService, "application:read"), workspaceHandler.AdminApplications)
	admin.POST("/applications/:id/approve", middleware.RequirePermission(authService, "application:approve"), workspaceHandler.AdminApplicationDecision)
	admin.POST("/applications/:id/reject", middleware.RequirePermission(authService, "application:approve"), workspaceHandler.AdminApplicationDecision)
	admin.GET("/server/status", middleware.RequirePermission(authService, "server:read_status"), workspaceHandler.AdminServerStatus)
	admin.POST("/server/commands", middleware.RequirePermission(authService, "server:command"), workspaceHandler.AdminServerCommand)

	portal := v1.Group("/portal/organizations/:slug")
	portal.GET("", workspaceHandler.Organization)
	portal.GET("/posts", workspaceHandler.PortalPosts)
	portal.GET("/projects", workspaceHandler.PortalProjects)
	portal.GET("/resources", workspaceHandler.PortalResources)
	portal.GET("/knowledge/articles", workspaceHandler.PortalKnowledge)
	portal.GET("/knowledge/directories", workspaceHandler.PortalKnowledgeDirectories)
	portal.GET("/server-status", workspaceHandler.PortalServer)
	portal.GET("/assets/:id/download", workspaceHandler.DownloadAsset)

	log.Printf("qutcraft api listening on %s", cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("http server stopped: %v", err)
	}
}

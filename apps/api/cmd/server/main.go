package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/config"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/handler"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/cache"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/database"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/logging"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/mailadapter"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/modelprovider"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/serveradapter"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/storage"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := checkReadiness("http://127.0.0.1:8080/readyz"); err != nil {
			log.Printf("readiness check failed: %v", err)
			os.Exit(1)
		}
		return
	}
	cfg := config.Load()
	logging.Init(cfg.AppEnv)
	appLogger := slog.Default()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	if err := database.MigrateAndSeed(db, cfg); err != nil {
		log.Fatalf("database migration or seed failed: %v", err)
	}
	publicCache := cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.PublicCacheTTL)
	storageContext, cancelStorageInitialization := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelStorageInitialization()
	mediaStorage, err := storage.New(storageContext, storage.Config{
		Driver:    cfg.StorageDriver,
		LocalRoot: cfg.StorageLocalRoot,
		Endpoint:  cfg.S3Endpoint,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
		Bucket:    cfg.S3Bucket,
		Region:    cfg.S3Region,
		UseSSL:    cfg.S3UseSSL,
	})
	if err != nil {
		log.Fatalf("media storage initialization failed: %v", err)
	}
	emailSender, err := mailadapter.New(mailadapter.Config{
		Driver:      cfg.EmailDriver,
		Host:        cfg.SMTPHost,
		Port:        cfg.SMTPPort,
		Username:    cfg.SMTPUsername,
		Password:    cfg.SMTPPassword,
		FromAddress: cfg.SMTPFromAddress,
		FromName:    cfg.SMTPFromName,
		Security:    cfg.SMTPSecurity,
		Timeout:     cfg.SMTPTimeout,
	})
	if err != nil {
		log.Fatalf("email adapter initialization failed: %v", err)
	}
	modelProvider, err := modelprovider.New(modelprovider.Config{
		Driver: cfg.AIProvider, BaseURL: cfg.AIBaseURL, APIKey: cfg.AIAPIKey,
		Model: cfg.AIModel, Timeout: cfg.AIRequestTimeout,
	})
	if err != nil {
		log.Fatalf("model provider initialization failed: %v", err)
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatalf("configure trusted proxies: %v", err)
	}
	router.Use(middleware.RequestID(), middleware.StructuredLogger(appLogger), gin.Recovery(), cors.New(corsConfig(cfg)))

	authService := service.NewAuthService(db, cfg)
	authHandler := handler.NewAuthHandler(db, authService, cfg.JWTRefreshTTL, cfg.AppEnv == "production")
	notificationService := service.NewNotificationService(db, emailSender)
	notificationService.StartWorker(context.Background(), 2*time.Second)
	notificationHandler := handler.NewNotificationHandler(db, notificationService)
	invitationHandler := handler.NewInvitationHandler(db, authService, emailSender, cfg.PublicWebBaseURL)
	workspaceHandler := handler.NewWorkspaceHandlerWithDependenciesAndNotifications(db, publicCache, cfg.AppEnv, serveradapter.NewMock(), cfg.ServerAdapterTimeout, mediaStorage, notificationService)
	portalConfigHandler := handler.NewPortalConfigHandler(db)
	auditHandler := handler.NewAuditHandler(db)
	agentService := service.NewAgentServiceWithProviderConfig(
		db, modelProvider,
		modelprovider.Config{
			Driver: cfg.AIProvider, BaseURL: cfg.AIBaseURL, APIKey: cfg.AIAPIKey,
			Model: cfg.AIModel, Timeout: cfg.AIRequestTimeout,
		},
		cfg.JWTAccessSecret, cfg.AppEnv == "production", cfg.AIRunLimitPerHour, cfg.AIRequestTimeout,
	)
	if err := agentService.RecoverInterruptedRuns(); err != nil {
		log.Fatalf("recover interrupted agent runs: %v", err)
	}
	agentService.StartWorker(context.Background(), 200*time.Millisecond)
	aiHandler := handler.NewAIHandler(agentService)
	authRateLimiter := middleware.NewRateLimiter(cfg.AuthRateLimitPerMinute, time.Minute)
	publicWriteRateLimiter := middleware.NewRateLimiter(cfg.PublicWriteLimitPerHour, time.Hour)
	sensitiveRateLimiter := middleware.NewRateLimiter(cfg.SensitiveLimitPerMinute, time.Minute)
	healthHandler := handler.NewHealthHandler(db, publicCache)
	router.GET("/healthz", healthHandler.Liveness)
	router.GET("/readyz", healthHandler.Readiness)

	v1 := router.Group("/api/v1")
	auth := v1.Group("/auth")
	auth.POST("/register", authRateLimiter.Middleware(), authHandler.Register)
	auth.POST("/login", authRateLimiter.Middleware(), authHandler.Login)
	auth.POST("/refresh", authRateLimiter.Middleware(), authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout)
	auth.GET("/me", middleware.RequireAuth(authService), authHandler.Me)
	auth.PATCH("/me", middleware.RequireAuth(authService), authHandler.UpdateMe)
	auth.GET("/organizations", middleware.RequireAuth(authService), authHandler.Organizations)
	auth.POST("/switch-organization", authRateLimiter.Middleware(), middleware.RequireAuth(authService), authHandler.SwitchOrganization)

	invitations := v1.Group("/invitations")
	invitations.GET("/:token", authRateLimiter.Middleware(), invitationHandler.Preview)
	invitations.POST("/:token/accept", authRateLimiter.Middleware(), middleware.RequireAuth(authService), invitationHandler.Accept)

	membership := v1.Group("/membership", middleware.RequireAuth(authService))
	membership.GET("/history", workspaceHandler.MembershipHistory)
	membership.POST("/leave", workspaceHandler.LeaveMembership)

	admin := v1.Group("/admin", middleware.RequireAuth(authService))
	admin.GET("/session", middleware.RequirePermission(authService, "organization:read"), authHandler.Me)
	admin.GET("/dashboard", middleware.RequirePermission(authService, "organization:read"), workspaceHandler.AdminDashboard)
	admin.GET("/organization", middleware.RequirePermission(authService, "organization:configure"), workspaceHandler.AdminOrganization)
	admin.PATCH("/organization", middleware.RequirePermission(authService, "organization:configure"), workspaceHandler.AdminUpdateOrganization)
	admin.GET("/content", middleware.RequirePermission(authService, "content:read"), workspaceHandler.AdminContent)
	admin.GET("/content/:id", middleware.RequirePermission(authService, "content:read"), workspaceHandler.AdminContentDetail)
	admin.GET("/content/:id/revisions", middleware.RequirePermission(authService, "content:read"), workspaceHandler.AdminContentRevisions)
	admin.GET("/content/:id/revisions/:revision_id", middleware.RequirePermission(authService, "content:read"), workspaceHandler.AdminContentRevisionDetail)
	admin.POST("/content/:id/revisions/:revision_id/restore", middleware.RequirePermission(authService, "content:update"), workspaceHandler.RestoreContentRevision)
	admin.GET("/knowledge/directories", middleware.RequirePermission(authService, "knowledge:read"), workspaceHandler.AdminKnowledgeDirectories)
	admin.POST("/knowledge/directories", middleware.RequirePermission(authService, "knowledge:manage"), workspaceHandler.AdminCreateKnowledgeDirectory)
	admin.PATCH("/knowledge/directories/:id", middleware.RequirePermission(authService, "knowledge:manage"), workspaceHandler.AdminUpdateKnowledgeDirectory)
	admin.POST("/content", middleware.RequirePermission(authService, "content:create"), workspaceHandler.AdminCreateContent)
	admin.PATCH("/content/:id", middleware.RequirePermission(authService, "content:update"), workspaceHandler.AdminUpdateContent)
	admin.POST("/content/:id/publish", middleware.RequirePermission(authService, "content:publish"), workspaceHandler.PublishContent)
	admin.POST("/content/:id/archive", middleware.RequirePermission(authService, "content:archive"), workspaceHandler.ArchiveContent)
	admin.POST("/assets", sensitiveRateLimiter.Middleware(), middleware.RequirePermission(authService, "asset:upload"), workspaceHandler.UploadAsset)
	admin.GET("/assets/:id/download", middleware.RequirePermission(authService, "asset:read"), workspaceHandler.DownloadAsset)
	admin.GET("/assets/:id/stats", middleware.RequirePermission(authService, "asset:read"), workspaceHandler.AssetDownloadStats)
	admin.GET("/users", middleware.RequirePermission(authService, "membership:read"), workspaceHandler.AdminUsers)
	admin.PATCH("/users/:id", middleware.RequirePermission(authService, "membership:manage"), workspaceHandler.AdminUpdateUser)
	admin.GET("/invitations", middleware.RequirePermission(authService, "membership:manage"), invitationHandler.List)
	admin.POST("/invitations", middleware.RequirePermission(authService, "membership:manage"), invitationHandler.Create)
	admin.POST("/invitation-batches", sensitiveRateLimiter.Middleware(), middleware.RequirePermission(authService, "membership:manage"), invitationHandler.CreateBatch)
	admin.DELETE("/invitations/:id", middleware.RequirePermission(authService, "membership:manage"), invitationHandler.Revoke)
	admin.POST("/invitations/:id/email/retry", sensitiveRateLimiter.Middleware(), middleware.RequirePermission(authService, "membership:manage"), invitationHandler.RetryEmail)
	admin.GET("/notifications/email/status", middleware.RequirePermission(authService, "organization:configure"), invitationHandler.EmailStatus)
	admin.GET("/notifications/invitation-template", middleware.RequirePermission(authService, "organization:configure"), notificationHandler.GetInvitationTemplate)
	admin.PATCH("/notifications/invitation-template", middleware.RequirePermission(authService, "organization:configure"), notificationHandler.UpdateInvitationTemplate)
	admin.GET("/notifications/outbox", middleware.RequirePermission(authService, "organization:configure"), notificationHandler.ListOutbox)
	admin.POST("/notifications/outbox/:id/retry", middleware.RequirePermission(authService, "organization:configure"), notificationHandler.RetryOutbox)
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
	admin.POST("/applications/:id/server-sync/retry", middleware.RequirePermission(authService, "application:approve"), workspaceHandler.AdminRetryApplicationServerSync)
	admin.GET("/server/status", middleware.RequirePermission(authService, "server:read_status"), workspaceHandler.AdminServerStatus)
	admin.POST("/server/commands", sensitiveRateLimiter.Middleware(), middleware.RequirePermission(authService, "server:command"), workspaceHandler.AdminServerCommand)
	admin.GET("/portal/config", middleware.RequirePermission(authService, "organization:configure"), portalConfigHandler.Get)
	admin.PATCH("/portal/config", middleware.RequirePermission(authService, "organization:configure"), portalConfigHandler.SaveDraft)
	admin.POST("/portal/config/enable", middleware.RequirePermission(authService, "organization:configure"), portalConfigHandler.Enable)
	admin.POST("/portal/config/restore-default", middleware.RequirePermission(authService, "organization:configure"), portalConfigHandler.RestoreDefault)
	admin.GET("/audit", middleware.RequirePermission(authService, "audit:read"), auditHandler.List)
	admin.GET("/ai/config", middleware.RequirePermission(authService, "ai:use"), aiHandler.GetConfiguration)
	admin.PATCH("/ai/config", sensitiveRateLimiter.Middleware(), middleware.RequirePermission(authService, "organization:configure"), aiHandler.UpdateConfiguration)
	admin.GET("/ai/agents", middleware.RequirePermission(authService, "ai:use"), aiHandler.ListAgents)
	admin.POST("/ai/knowledge/search", middleware.RequirePermission(authService, "ai:use"), middleware.RequirePermission(authService, "knowledge:read"), aiHandler.SearchKnowledge)
	admin.POST("/ai/runs", sensitiveRateLimiter.Middleware(), middleware.RequirePermission(authService, "ai:use"), middleware.RequirePermission(authService, "knowledge:read"), aiHandler.CreateRun)
	admin.GET("/ai/runs/:run_id", middleware.RequirePermission(authService, "ai:use"), aiHandler.GetRun)
	admin.POST("/ai/runs/:run_id/cancel", middleware.RequirePermission(authService, "ai:use"), aiHandler.CancelRun)
	admin.POST("/ai/activity-plans", sensitiveRateLimiter.Middleware(), middleware.RequirePermission(authService, "ai:use"), middleware.RequirePermission(authService, "knowledge:read"), aiHandler.CreateActivityPlan)
	admin.GET("/ai/activity-plans", middleware.RequirePermission(authService, "ai:use"), aiHandler.ListActivityPlans)
	admin.GET("/ai/activity-plans/evaluation-summary", middleware.RequirePermission(authService, "ai:use"), aiHandler.GetActivityPlanEvaluationSummary)
	admin.GET("/ai/activity-plans/:plan_id", middleware.RequirePermission(authService, "ai:use"), aiHandler.GetActivityPlan)
	admin.GET("/ai/activity-plans/:plan_id/evaluation", middleware.RequirePermission(authService, "ai:use"), aiHandler.GetActivityPlanEvaluation)
	admin.PUT("/ai/activity-plans/:plan_id/evaluation", sensitiveRateLimiter.Middleware(), middleware.RequirePermission(authService, "ai:use"), aiHandler.SaveActivityPlanEvaluation)
	admin.POST("/ai/activity-plans/:plan_id/approve", sensitiveRateLimiter.Middleware(), middleware.RequirePermission(authService, "ai:use"), middleware.RequirePermission(authService, "project:manage"), middleware.RequirePermission(authService, "content:create"), aiHandler.ApproveActivityPlan)

	portal := v1.Group("/portal/organizations/:slug")
	portal.GET("", workspaceHandler.Organization)
	portal.GET("/configuration", portalConfigHandler.Public)
	portal.GET("/content/:id", workspaceHandler.PortalContentDetail)
	portal.GET("/posts", workspaceHandler.PortalPosts)
	portal.GET("/projects", workspaceHandler.PortalProjects)
	portal.GET("/resources", workspaceHandler.PortalResources)
	portal.GET("/knowledge/articles", workspaceHandler.PortalKnowledge)
	portal.GET("/knowledge/directories", workspaceHandler.PortalKnowledgeDirectories)
	portal.GET("/server-status", workspaceHandler.PortalServer)
	portal.POST("/apply", publicWriteRateLimiter.Middleware(), workspaceHandler.SubmitApplication)
	portal.GET("/assets/:id/download", workspaceHandler.DownloadAsset)

	appLogger.Info("qutcraft api listening", "address", cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("http server stopped: %v", err)
	}
}

func checkReadiness(target string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(target)
	if err != nil {
		return fmt.Errorf("request readiness endpoint: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("readiness endpoint returned HTTP %d", response.StatusCode)
	}
	return nil
}

func corsConfig(cfg config.Config) cors.Config {
	return cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "Idempotency-Key", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
	}
}

package main

import (
	"log"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/config"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/handler"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
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
	workspaceHandler := handler.NewWorkspaceHandler(db)
	router.GET("/healthz", handler.Health)

	v1 := router.Group("/api/v1")
	auth := v1.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout)
	auth.GET("/me", middleware.RequireAuth(authService), authHandler.Me)

	admin := v1.Group("/admin", middleware.RequireAuth(authService))
	admin.GET("/session", middleware.RequirePermission(authService, "organization:read"), authHandler.Me)
	admin.GET("/dashboard", workspaceHandler.AdminDashboard)
	admin.GET("/content", workspaceHandler.AdminContent)
	admin.POST("/content", workspaceHandler.AdminCreateContent)
	admin.GET("/users", workspaceHandler.AdminUsers)
	admin.GET("/applications", workspaceHandler.AdminApplications)
	admin.POST("/applications/:id/approve", workspaceHandler.AdminApplicationDecision)
	admin.POST("/applications/:id/reject", workspaceHandler.AdminApplicationDecision)
	admin.GET("/server/status", workspaceHandler.AdminServerStatus)
	admin.POST("/server/commands", workspaceHandler.AdminServerCommand)

	portal := v1.Group("/portal/organizations/:slug")
	portal.GET("", workspaceHandler.Organization)
	portal.GET("/posts", workspaceHandler.PortalPosts)
	portal.GET("/projects", workspaceHandler.PortalProjects)
	portal.GET("/resources", workspaceHandler.PortalResources)
	portal.GET("/knowledge/articles", workspaceHandler.PortalKnowledge)
	portal.GET("/server-status", workspaceHandler.PortalServer)

	log.Printf("qutcraft api listening on %s", cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("http server stopped: %v", err)
	}
}

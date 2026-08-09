package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

type IntegrationHandler struct {
	integrations *service.IntegrationService
}

type integrationSettingsRequest struct {
	PublicWebBaseURL string `json:"public_web_base_url"`
	Email            struct {
		Driver         string `json:"driver"`
		Host           string `json:"host"`
		Port           int    `json:"port"`
		Username       string `json:"username"`
		Password       string `json:"password"`
		ClearPassword  bool   `json:"clear_password"`
		FromAddress    string `json:"from_address"`
		FromName       string `json:"from_name"`
		Security       string `json:"security"`
		TimeoutSeconds int    `json:"timeout_seconds"`
	} `json:"email"`
	Storage struct {
		Driver         string `json:"driver"`
		Endpoint       string `json:"endpoint"`
		AccessKey      string `json:"access_key"`
		SecretKey      string `json:"secret_key"`
		ClearAccessKey bool   `json:"clear_access_key"`
		ClearSecretKey bool   `json:"clear_secret_key"`
		Bucket         string `json:"bucket"`
		Region         string `json:"region"`
		UseSSL         bool   `json:"use_ssl"`
	} `json:"storage"`
}

type integrationTestRequest struct {
	Section string `json:"section"`
}

func NewIntegrationHandler(integrations *service.IntegrationService) *IntegrationHandler {
	return &IntegrationHandler{integrations: integrations}
}

func (h *IntegrationHandler) Get(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	settings, err := h.integrations.Settings(c.Request.Context(), principal.OrganizationID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "integration.load_failed", "服务接入配置暂时无法加载。")
		return
	}
	respond(c, http.StatusOK, settings)
}

func (h *IntegrationHandler) Update(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request integrationSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "integration.validation_failed", "请填写完整、有效的服务接入配置。")
		return
	}
	settings, err := h.integrations.Update(c.Request.Context(), principal, service.IntegrationSettingsInput{
		PublicWebBaseURL: request.PublicWebBaseURL,
		Email: service.EmailIntegrationInput{
			Driver: request.Email.Driver, Host: request.Email.Host, Port: request.Email.Port,
			Username: request.Email.Username, Password: request.Email.Password, ClearPassword: request.Email.ClearPassword,
			FromAddress: request.Email.FromAddress, FromName: request.Email.FromName,
			Security: request.Email.Security, TimeoutSeconds: request.Email.TimeoutSeconds,
		},
		Storage: service.StorageIntegrationInput{
			Driver: request.Storage.Driver, Endpoint: request.Storage.Endpoint,
			AccessKey: request.Storage.AccessKey, SecretKey: request.Storage.SecretKey,
			ClearAccessKey: request.Storage.ClearAccessKey, ClearSecretKey: request.Storage.ClearSecretKey,
			Bucket: request.Storage.Bucket, Region: request.Storage.Region, UseSSL: request.Storage.UseSSL,
		},
	}, ensureRequestID(c))
	if err != nil {
		if errors.Is(err, service.ErrIntegrationValidation) {
			fail(c, http.StatusBadRequest, "integration.validation_failed", "配置格式不正确；请检查公网地址、端口、凭据和存储桶。")
			return
		}
		fail(c, http.StatusInternalServerError, "integration.update_failed", "服务接入配置暂时无法保存。")
		return
	}
	respond(c, http.StatusOK, settings)
}

func (h *IntegrationHandler) Test(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request integrationTestRequest
	if err := c.ShouldBindJSON(&request); err != nil || (strings.TrimSpace(request.Section) != "email" && strings.TrimSpace(request.Section) != "storage") {
		fail(c, http.StatusBadRequest, "integration.test_section_invalid", "section 仅支持 email 或 storage。")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	if err := h.integrations.Test(ctx, principal.OrganizationID, request.Section); err != nil {
		if errors.Is(err, service.ErrIntegrationSection) {
			fail(c, http.StatusBadRequest, "integration.test_section_invalid", "section 仅支持 email 或 storage。")
			return
		}
		fail(c, http.StatusServiceUnavailable, "integration.test_failed", "连接验证失败，请检查地址、网络、凭据和服务状态。")
		return
	}
	respond(c, http.StatusOK, gin.H{"section": request.Section, "reachable": true, "checked_at": time.Now().UTC()})
}

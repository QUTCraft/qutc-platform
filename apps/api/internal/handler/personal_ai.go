package handler

import (
	"errors"
	"net/http"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/modelprovider"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *AIHandler) GetPersonalConfiguration(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, 401, "auth.token_missing", "请先登录。")
		return
	}
	view, err := h.agents.PersonalAI(principal)
	if err != nil {
		h.failPersonalAI(c, err)
		return
	}
	respond(c, http.StatusOK, view)
}

func (h *AIHandler) SavePersonalConfiguration(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, 401, "auth.token_missing", "请先登录。")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	var input service.PersonalAIInput
	if c.ShouldBindJSON(&input) != nil {
		h.failPersonalAI(c, service.ErrAgentConfigValidation)
		return
	}
	view, err := h.agents.SavePersonalAI(principal, input, ensureRequestID(c))
	if err != nil {
		h.failPersonalAI(c, err)
		return
	}
	respond(c, http.StatusOK, view)
}

func (h *AIHandler) DeletePersonalConfiguration(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, 401, "auth.token_missing", "请先登录。")
		return
	}
	if err := h.agents.DeletePersonalAI(principal, ensureRequestID(c)); err != nil {
		h.failPersonalAI(c, err)
		return
	}
	respond(c, http.StatusOK, gin.H{"removed": true})
}

func (h *AIHandler) EditorChat(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, 401, "auth.token_missing", "请先登录。")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 300<<10)
	var input service.EditorChatInput
	if c.ShouldBindJSON(&input) != nil {
		h.failPersonalAI(c, service.ErrAgentValidation)
		return
	}
	result, err := h.agents.EditorChat(c.Request.Context(), principal, input, ensureRequestID(c))
	if err != nil {
		h.failPersonalAI(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}

func (h *AIHandler) failPersonalAI(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAgentConfigValidation):
		fail(c, 400, "ai.personal_config_invalid", "请填写公开 HTTPS 接口、模型与 API Key；更换调用地址时需要重新填写 Key。")
	case errors.Is(err, service.ErrAgentValidation):
		fail(c, 400, "ai.chat_invalid", "对话过长或格式无效，请清空对话后重试（最多 20 条消息、40000 字对话和 30000 字文章）。")
	case errors.Is(err, service.ErrChatPermission):
		fail(c, 403, "ai.permission_denied", "当前角色不能使用组织 AI，可配置自己的接口。")
	case errors.Is(err, service.ErrAgentRunQuotaExceeded):
		fail(c, 429, "ai.quota_exceeded", "本小时 AI 使用额度已用完，请稍后再试。")
	case errors.Is(err, service.ErrAgentProviderDisabled), errors.Is(err, service.ErrAgentFeatureDisabled):
		fail(c, 503, "ai.provider_unavailable", "所选 AI 尚未配置或已停用，请检查配置。")
	case errors.Is(err, modelprovider.ErrUnavailable), errors.Is(err, modelprovider.ErrInvalidData):
		fail(c, 502, "ai.upstream_failed", "AI 请求失败或超时，请检查调用地址、模型、额度与 Key 后重试。")
	default:
		fail(c, 500, "ai.chat_failed", "AI 服务暂时不可用，请稍后再试。")
	}
}

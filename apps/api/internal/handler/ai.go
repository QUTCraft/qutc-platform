package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	agents *service.AgentService
}

type knowledgeSearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type createAgentRunRequest struct {
	AgentKey    string                   `json:"agent_key"`
	Task        string                   `json:"task"`
	ContextRefs []service.AgentSourceRef `json:"context_refs"`
	OutputMode  string                   `json:"output_mode"`
}

type agentConfigurationRequest struct {
	Enabled               bool `json:"enabled"`
	RunLimitPerHour       int  `json:"run_limit_per_hour"`
	RequestTimeoutSeconds int  `json:"request_timeout_seconds"`
	MaxSources            int  `json:"max_sources"`
	MaxContextCharacters  int  `json:"max_context_characters"`
}

func NewAIHandler(agents *service.AgentService) *AIHandler {
	return &AIHandler{agents: agents}
}

func (h *AIHandler) GetConfiguration(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	configuration, err := h.agents.Configuration(principal.OrganizationID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "ai.config_load_failed", "智能体配置暂时无法加载。")
		return
	}
	respond(c, http.StatusOK, configuration)
}

func (h *AIHandler) UpdateConfiguration(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request agentConfigurationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "ai.config_validation_failed", "请提供完整、有效的智能体配置。")
		return
	}
	configuration, err := h.agents.UpdateConfiguration(principal, service.AgentConfigurationInput{
		Enabled: request.Enabled, RunLimitPerHour: request.RunLimitPerHour,
		RequestTimeoutSeconds: request.RequestTimeoutSeconds, MaxSources: request.MaxSources,
		MaxContextCharacters: request.MaxContextCharacters,
	}, ensureRequestID(c))
	if err != nil {
		if errors.Is(err, service.ErrAgentConfigValidation) {
			fail(c, http.StatusBadRequest, "ai.config_validation_failed", "智能体配置超出允许范围。")
			return
		}
		fail(c, http.StatusInternalServerError, "ai.config_update_failed", "智能体配置暂时无法保存。")
		return
	}
	respond(c, http.StatusOK, configuration)
}

func (h *AIHandler) ListAgents(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	agents, err := h.agents.ListAgents(principal.OrganizationID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "ai.agents_load_failed", "智能体列表暂时无法加载。")
		return
	}
	respond(c, http.StatusOK, gin.H{"agents": agents, "provider": h.agents.ProviderStatus()})
}

func (h *AIHandler) SearchKnowledge(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request knowledgeSearchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "ai.validation_failed", "请提供有效的知识检索条件。")
		return
	}
	results, err := h.agents.SearchKnowledge(principal.OrganizationID, request.Query, request.Limit)
	if err != nil {
		if errors.Is(err, service.ErrAgentValidation) {
			fail(c, http.StatusBadRequest, "ai.validation_failed", "query 必须为 1 到 80 个字符，limit 必须在 1 到 20 之间。")
			return
		}
		fail(c, http.StatusInternalServerError, "ai.knowledge_search_failed", "知识资料暂时无法检索。")
		return
	}
	respond(c, http.StatusOK, results)
}

func (h *AIHandler) CreateRun(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request createAgentRunRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "ai.validation_failed", "请提供有效的智能体运行参数。")
		return
	}
	run, err := h.agents.CreateRun(principal, service.AgentRunCreateInput{
		AgentKey: request.AgentKey, Task: request.Task, ContextRefs: request.ContextRefs,
		OutputMode: request.OutputMode,
	}, ensureRequestID(c))
	if err != nil {
		h.failRun(c, err)
		return
	}
	respond(c, http.StatusAccepted, run)
}

func (h *AIHandler) GetRun(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	run, err := h.agents.GetRun(principal.OrganizationID, strings.TrimSpace(c.Param("run_id")))
	if err != nil {
		h.failRun(c, err)
		return
	}
	respond(c, http.StatusOK, run)
}

func (h *AIHandler) CancelRun(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	run, err := h.agents.CancelRun(principal, strings.TrimSpace(c.Param("run_id")), ensureRequestID(c))
	if err != nil {
		h.failRun(c, err)
		return
	}
	respond(c, http.StatusOK, run)
}

func (h *AIHandler) failRun(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAgentValidation):
		fail(c, http.StatusBadRequest, "ai.validation_failed", "智能体运行参数不符合接口约束。")
	case errors.Is(err, service.ErrAgentNotFound):
		fail(c, http.StatusNotFound, "ai.agent_not_found", "当前组织没有可用的指定智能体。")
	case errors.Is(err, service.ErrAgentSourceNotFound):
		fail(c, http.StatusNotFound, "ai.source_not_found", "引用资料不存在或不在当前组织的可访问知识范围内。")
	case errors.Is(err, service.ErrAgentProviderDisabled):
		fail(c, http.StatusServiceUnavailable, "ai.provider_unavailable", "模型供应商尚未启用或配置不完整。")
	case errors.Is(err, service.ErrAgentFeatureDisabled):
		fail(c, http.StatusConflict, "ai.feature_disabled", "当前组织已停用智能体功能。")
	case errors.Is(err, service.ErrAgentRunQuotaExceeded):
		fail(c, http.StatusTooManyRequests, "ai.run_quota_exceeded", "当前用户本小时的智能体运行额度已用完。")
	case errors.Is(err, service.ErrAgentRunNotFound):
		fail(c, http.StatusNotFound, "ai.run_not_found", "智能体运行不存在。")
	case errors.Is(err, service.ErrAgentRunNotCancelable):
		fail(c, http.StatusConflict, "ai.run_not_cancelable", "智能体运行已经结束，不能取消。")
	default:
		fail(c, http.StatusInternalServerError, "ai.run_failed", "智能体运行暂时无法处理。")
	}
}

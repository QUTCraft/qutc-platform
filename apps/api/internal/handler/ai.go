// ai.go 提供组织级 AI 智能体配置、知识检索、运行任务和活动方案审核相关的 HTTP 处理器。
// Handler 只负责请求绑定、身份上下文提取、状态码映射和统一响应；实际业务由 AgentService 完成。
package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/middleware"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/service"
	"github.com/gin-gonic/gin"
)

// AIHandler 持有 AI 领域服务，并将其能力暴露为 Gin 路由处理函数。
type AIHandler struct {
	agents *service.AgentService
}

// knowledgeSearchRequest 描述知识库检索请求；Limit 用于限制返回条数。
type knowledgeSearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// createAgentRunRequest 描述一次智能体运行任务及其上下文来源。
type createAgentRunRequest struct {
	AgentKey    string                   `json:"agent_key"`
	Task        string                   `json:"task"`
	ContextRefs []service.AgentSourceRef `json:"context_refs"`
	OutputMode  string                   `json:"output_mode"`
}

// agentConfigurationRequest 是管理员更新组织 AI 配置时接收的 JSON 结构。
type agentConfigurationRequest struct {
	Enabled               bool   `json:"enabled"`
	RunLimitPerHour       int    `json:"run_limit_per_hour"`
	RequestTimeoutSeconds int    `json:"request_timeout_seconds"`
	MaxSources            int    `json:"max_sources"`
	MaxContextCharacters  int    `json:"max_context_characters"`
	Provider              string `json:"provider"`
	BaseURL               string `json:"base_url"`
	APIKey                string `json:"api_key"`
	Model                 string `json:"model"`
}

// createActivityPlanRequest 描述活动方案生成所需的目标、时间、参与人数和约束条件。
type createActivityPlanRequest struct {
	Title                string                   `json:"title"`
	Objective            string                   `json:"objective"`
	Audience             string                   `json:"audience"`
	Venue                string                   `json:"venue"`
	StartsAt             string                   `json:"starts_at"`
	EndsAt               string                   `json:"ends_at"`
	ExpectedParticipants int                      `json:"expected_participants"`
	Budget               string                   `json:"budget"`
	Constraints          string                   `json:"constraints"`
	ContextRefs          []service.AgentSourceRef `json:"context_refs"`
}

// approveActivityPlanRequest 包含管理员批准活动方案时选择的行动项。
type approveActivityPlanRequest struct {
	Actions []string `json:"actions"`
}

// activityPlanEvaluationRequest 是对活动方案进行多维评分和文字备注的请求体。
type activityPlanEvaluationRequest struct {
	Accuracy     int    `json:"accuracy"`
	Feasibility  int    `json:"feasibility"`
	CampusFit    int    `json:"campus_fit"`
	Clarity      int    `json:"clarity"`
	Adoptability int    `json:"adoptability"`
	Notes        string `json:"notes"`
}

// NewAIHandler 创建 AI 处理器，并注入组织级 AgentService 依赖。
func NewAIHandler(agents *service.AgentService) *AIHandler {
	return &AIHandler{agents: agents}
}

// GetConfiguration 返回当前用户所在组织的 AI 智能体配置。
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

// UpdateConfiguration 绑定并保存组织 AI 配置，同时将校验错误转换为 API 错误响应。
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
		MaxContextCharacters: request.MaxContextCharacters, Provider: request.Provider,
		BaseURL: request.BaseURL, APIKey: request.APIKey, Model: request.Model,
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

// ListAgents 返回当前组织可用的智能体目录。
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
	respond(c, http.StatusOK, gin.H{"agents": agents, "provider": h.agents.ProviderStatusForOrganization(principal.OrganizationID)})
}

// SearchKnowledge 根据查询词检索当前组织的知识内容，结果受请求 Limit 约束。
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
	results, err := h.agents.SearchKnowledge(principal, request.Query, request.Limit)
	if err != nil {
		if errors.Is(err, service.ErrAgentValidation) {
			fail(c, http.StatusBadRequest, "ai.validation_failed", "query 最多为 80 个字符，limit 必须在 1 到 20 之间。")
			return
		}
		fail(c, http.StatusInternalServerError, "ai.knowledge_search_failed", "知识资料暂时无法检索。")
		return
	}
	respond(c, http.StatusOK, results)
}

// CreateRun 创建一次异步或同步的智能体运行任务。
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

// GetRun 查询指定智能体运行任务的当前状态和输出。
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

// CancelRun 请求停止尚未完成的智能体运行任务。
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

// CreateActivityPlan 根据活动输入生成一份活动方案。
func (h *AIHandler) CreateActivityPlan(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request createActivityPlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "ai.activity_plan_validation_failed", "请提供完整、有效的活动需求。")
		return
	}
	startsAt, err := optionalRFC3339(request.StartsAt)
	if err != nil {
		fail(c, http.StatusBadRequest, "ai.activity_plan_validation_failed", "活动开始时间必须是 RFC3339 日期时间。")
		return
	}
	endsAt, err := optionalRFC3339(request.EndsAt)
	if err != nil {
		fail(c, http.StatusBadRequest, "ai.activity_plan_validation_failed", "活动结束时间必须是 RFC3339 日期时间。")
		return
	}
	plan, err := h.agents.CreateActivityPlan(principal, service.ActivityPlanCreateInput{
		Title: request.Title, Objective: request.Objective, Audience: request.Audience,
		Venue: request.Venue, StartsAt: startsAt, EndsAt: endsAt,
		ExpectedParticipants: request.ExpectedParticipants, Budget: request.Budget,
		Constraints: request.Constraints, ContextRefs: request.ContextRefs,
	}, ensureRequestID(c))
	if err != nil {
		h.failActivityPlan(c, err)
		return
	}
	respond(c, http.StatusAccepted, plan)
}

// GetActivityPlan 返回指定活动方案的详情。
func (h *AIHandler) GetActivityPlan(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	plan, err := h.agents.GetActivityPlan(principal.OrganizationID, strings.TrimSpace(c.Param("plan_id")))
	if err != nil {
		h.failActivityPlan(c, err)
		return
	}
	respond(c, http.StatusOK, plan)
}

// ListActivityPlans 分页列出当前组织的活动方案。
func (h *AIHandler) ListActivityPlans(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	page, pageSize, ok := listMeta(c, 0)
	if !ok {
		return
	}
	plans, total, err := h.agents.ListActivityPlans(principal, page, pageSize)
	if err != nil {
		fail(c, http.StatusInternalServerError, "ai.activity_plan_list_failed", "活动策划历史暂时无法加载。")
		return
	}
	respondWithMeta(c, http.StatusOK, plans, gin.H{"page": page, "page_size": pageSize, "total": total})
}

// GetActivityPlanEvaluation 返回指定方案的单条评价详情。
func (h *AIHandler) GetActivityPlanEvaluation(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	evaluation, err := h.agents.GetActivityPlanEvaluation(principal, strings.TrimSpace(c.Param("plan_id")))
	if err != nil {
		h.failActivityPlan(c, err)
		return
	}
	respond(c, http.StatusOK, evaluation)
}

// GetActivityPlanEvaluationSummary 汇总指定活动方案的评价统计。
func (h *AIHandler) GetActivityPlanEvaluationSummary(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	summary, err := h.agents.ActivityPlanEvaluationSummary(principal.OrganizationID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "ai.activity_plan_evaluation_summary_failed", "活动策划质量汇总暂时无法加载。")
		return
	}
	respond(c, http.StatusOK, summary)
}

// SaveActivityPlanEvaluation 创建或更新当前用户对活动方案的评价。
func (h *AIHandler) SaveActivityPlanEvaluation(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request activityPlanEvaluationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "ai.activity_plan_evaluation_validation_failed", "请提供完整、有效的五维评分。")
		return
	}
	evaluation, err := h.agents.SaveActivityPlanEvaluation(principal, strings.TrimSpace(c.Param("plan_id")), service.ActivityPlanEvaluationInput{
		Accuracy: request.Accuracy, Feasibility: request.Feasibility, CampusFit: request.CampusFit,
		Clarity: request.Clarity, Adoptability: request.Adoptability, Notes: request.Notes,
	}, ensureRequestID(c))
	if err != nil {
		h.failActivityPlan(c, err)
		return
	}
	respond(c, http.StatusOK, evaluation)
}

// ApproveActivityPlan 记录管理员对活动方案的批准及选定行动项。
func (h *AIHandler) ApproveActivityPlan(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	var request approveActivityPlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		fail(c, http.StatusBadRequest, "ai.activity_plan_validation_failed", "请选择需要人工批准的操作。")
		return
	}
	result, err := h.agents.ApproveActivityPlan(principal, strings.TrimSpace(c.Param("plan_id")), request.Actions, ensureRequestID(c))
	if err != nil {
		h.failActivityPlan(c, err)
		return
	}
	respond(c, http.StatusOK, result)
}

func (h *AIHandler) DeleteActivityPlan(c *gin.Context) {
	principal, ok := middleware.PrincipalFromContext(c)
	if !ok {
		fail(c, http.StatusUnauthorized, "auth.token_missing", "缺少访问令牌。")
		return
	}
	if err := h.agents.DeleteActivityPlan(principal, strings.TrimSpace(c.Param("plan_id")), ensureRequestID(c)); err != nil {
		h.failActivityPlan(c, err)
		return
	}
	respond(c, http.StatusOK, gin.H{"removed": true, "id": strings.TrimSpace(c.Param("plan_id"))})
}

// optionalRFC3339 将可选的 RFC3339 字符串转换为 UTC 时间；空字符串表示未设置。
func optionalRFC3339(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

// failActivityPlan 将活动方案服务层错误映射为稳定的 HTTP API 错误码。
func (h *AIHandler) failActivityPlan(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrActivityPlanValidation):
		fail(c, http.StatusBadRequest, "ai.activity_plan_validation_failed", "活动需求或批准操作不符合接口约束。")
	case errors.Is(err, service.ErrActivityPlanNotFound):
		fail(c, http.StatusNotFound, "ai.activity_plan_not_found", "活动策划不存在。")
	case errors.Is(err, service.ErrActivityPlanNotReady):
		fail(c, http.StatusConflict, "ai.activity_plan_not_ready", "活动方案尚未生成完成，不能执行建议操作。")
	case errors.Is(err, service.ErrActivityPlanAlreadyApplied):
		fail(c, http.StatusConflict, "ai.activity_plan_already_applied", "活动方案已经批准执行，不能重复创建业务对象。")
	default:
		h.failRun(c, err)
	}
}

// failRun 将智能体运行服务层错误映射为客户端可理解的状态码和错误码。
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

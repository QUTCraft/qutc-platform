package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/modelprovider"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrAgentNotFound         = errors.New("agent definition not found")
	ErrAgentRunNotFound      = errors.New("agent run not found")
	ErrAgentRunNotCancelable = errors.New("agent run is not cancelable")
	ErrAgentValidation       = errors.New("agent request validation failed")
	ErrAgentSourceNotFound   = errors.New("agent source not found")
	ErrAgentProviderDisabled = errors.New("agent model provider is unavailable")
	ErrAgentRunQuotaExceeded = errors.New("agent run quota exceeded")
	ErrAgentFeatureDisabled  = errors.New("agent feature is disabled")
	ErrAgentConfigValidation = errors.New("agent configuration validation failed")
)

const (
	AgentRunQueued    = "queued"
	AgentRunRunning   = "running"
	AgentRunSucceeded = "succeeded"
	AgentRunFailed    = "failed"
	AgentRunCanceled  = "canceled"
)

type AgentSourceRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type AgentRunCreateInput struct {
	AgentKey    string
	Task        string
	ContextRefs []AgentSourceRef
	OutputMode  string
}

type AgentDefinitionView struct {
	ID                  string   `json:"id"`
	Key                 string   `json:"key"`
	Name                string   `json:"name"`
	Purpose             string   `json:"purpose"`
	SystemPolicyVersion string   `json:"system_policy_version"`
	AllowedToolKeys     []string `json:"allowed_tool_keys"`
	ModelProfile        string   `json:"model_profile"`
	Enabled             bool     `json:"enabled"`
}

type AgentConfigurationInput struct {
	Enabled               bool
	RunLimitPerHour       int
	RequestTimeoutSeconds int
	MaxSources            int
	MaxContextCharacters  int
	Provider              string
	BaseURL               string
	APIKey                string
	Model                 string
}

type AgentProviderConfigurationView struct {
	Driver           string `json:"driver"`
	BaseURL          string `json:"base_url"`
	Model            string `json:"model"`
	APIKeyConfigured bool   `json:"api_key_configured"`
	APIKeyHint       string `json:"api_key_hint,omitempty"`
	Source           string `json:"source"`
}

type AgentConfigurationView struct {
	ID                    string                         `json:"id,omitempty"`
	Enabled               bool                           `json:"enabled"`
	RunLimitPerHour       int                            `json:"run_limit_per_hour"`
	RequestTimeoutSeconds int                            `json:"request_timeout_seconds"`
	MaxSources            int                            `json:"max_sources"`
	MaxContextCharacters  int                            `json:"max_context_characters"`
	Provider              modelprovider.Status           `json:"provider"`
	ProviderConfig        AgentProviderConfigurationView `json:"provider_config"`
	UpdatedBy             string                         `json:"updated_by,omitempty"`
	UpdatedAt             *time.Time                     `json:"updated_at,omitempty"`
}

type AgentKnowledgeResult struct {
	SourceType string    `json:"source_type"`
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Excerpt    string    `json:"excerpt"`
	Status     string    `json:"status"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type AgentCitationView struct {
	ID              string    `json:"id"`
	SourceType      string    `json:"source_type"`
	SourceID        string    `json:"source_id"`
	Title           string    `json:"title"`
	Excerpt         string    `json:"excerpt"`
	SourceUpdatedAt time.Time `json:"source_updated_at"`
}

type AgentRunView struct {
	ID             string              `json:"id"`
	AgentKey       string              `json:"agent_key"`
	AgentName      string              `json:"agent_name"`
	Status         string              `json:"status"`
	Task           string              `json:"task"`
	OutputTitle    string              `json:"output_title"`
	OutputExcerpt  string              `json:"output_excerpt"`
	OutputMarkdown string              `json:"output_markdown"`
	Provider       string              `json:"provider"`
	Mode           string              `json:"mode"`
	Model          string              `json:"model"`
	PromptVersion  string              `json:"prompt_version"`
	InputTokens    int                 `json:"input_tokens"`
	OutputTokens   int                 `json:"output_tokens"`
	FailureCode    string              `json:"failure_code"`
	FailureMessage string              `json:"failure_message"`
	RequestID      string              `json:"request_id"`
	Citations      []AgentCitationView `json:"citations"`
	StartedAt      *time.Time          `json:"started_at"`
	CompletedAt    *time.Time          `json:"completed_at"`
	ExpiresAt      time.Time           `json:"expires_at"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

type agentExecution struct {
	runID          string
	organizationID string
	actorUserID    string
	requestID      string
	agentKey       string
	promptVersion  string
	task           string
	sources        []modelprovider.Source
	timeout        time.Duration
	provider       modelprovider.Provider
}

type AgentService struct {
	db             *gorm.DB
	provider       modelprovider.Provider
	providerConfig modelprovider.Config
	credentialKey  []byte
	production     bool
	runLimit       int
	timeout        time.Duration
	cancelMu       sync.Mutex
	cancelRuns     map[string]context.CancelFunc
}

func NewAgentService(db *gorm.DB, provider modelprovider.Provider, runLimit int, timeout time.Duration) *AgentService {
	return NewAgentServiceWithProviderConfig(db, provider, modelprovider.Config{}, "development-only-agent-credential-key", false, runLimit, timeout)
}

func NewAgentServiceWithProviderConfig(db *gorm.DB, provider modelprovider.Provider, providerConfig modelprovider.Config, credentialSecret string, production bool, runLimit int, timeout time.Duration) *AgentService {
	if runLimit <= 0 {
		runLimit = 20
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &AgentService{
		db: db, provider: provider, providerConfig: providerConfig,
		credentialKey: deriveAgentCredentialKey(credentialSecret), production: production,
		runLimit: runLimit, timeout: timeout,
		cancelRuns: make(map[string]context.CancelFunc),
	}
}

func (s *AgentService) RecoverInterruptedRuns() error {
	now := time.Now().UTC()
	var runs []model.AgentRun
	if err := s.db.Where("status = ?", AgentRunRunning).Find(&runs).Error; err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, run := range runs {
			result := tx.Model(&model.AgentRun{}).
				Where("id = ? AND status = ?", run.ID, AgentRunRunning).
				Updates(map[string]any{
					"status":          AgentRunFailed,
					"failure_code":    "ai.run_interrupted",
					"failure_message": "服务重启中断了本次运行，请重新创建。",
					"completed_at":    now,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				continue
			}
			if err := tx.Create(&model.AuditEvent{
				ID: uuid.NewString(), OrganizationID: run.OrganizationID, ActorUserID: run.ActorUserID,
				Action: "ai.run_result", TargetType: "agent_run", TargetID: run.ID,
				Result: "failed", RequestID: run.RequestID, CreatedAt: now,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *AgentService) StartWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				execution, ok, err := s.claimQueuedRun()
				if err != nil || !ok {
					continue
				}
				go s.execute(execution)
			}
		}
	}()
}

func (s *AgentService) claimQueuedRun() (agentExecution, bool, error) {
	now := time.Now().UTC()
	var run model.AgentRun
	err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("status = ? AND expires_at > ?", AgentRunQueued, now).Order("created_at ASC, id ASC").Limit(1).Find(&run)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		claim := tx.Model(&model.AgentRun{}).Where("id = ? AND status = ?", run.ID, AgentRunQueued).Updates(map[string]any{"status": AgentRunRunning, "started_at": now, "updated_at": now})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected == 0 {
			run = model.AgentRun{}
			return nil
		}
		run.Status = AgentRunRunning
		run.StartedAt = &now
		return nil
	})
	if err != nil || run.ID == "" {
		return agentExecution{}, false, err
	}
	execution, err := s.executionForRun(run)
	if err != nil {
		_ = s.db.Model(&model.AgentRun{}).Where("id = ? AND status = ?", run.ID, AgentRunRunning).Updates(map[string]any{
			"status": AgentRunFailed, "failure_code": "ai.run_rehydrate_failed", "failure_message": "队列任务上下文无法恢复，请重新创建。", "completed_at": time.Now().UTC(),
		})
		s.audit(run.OrganizationID, run.ActorUserID, "ai.run_result", "agent_run", run.ID, "failed", run.RequestID)
		return agentExecution{}, false, nil
	}
	return execution, true, nil
}

func (s *AgentService) executionForRun(run model.AgentRun) (agentExecution, error) {
	var definition model.AgentDefinition
	if err := s.db.Where("id = ? AND organization_id = ?", run.AgentDefinitionID, run.OrganizationID).First(&definition).Error; err != nil {
		return agentExecution{}, err
	}
	var citations []model.AgentCitation
	if err := s.db.Where("run_id = ? AND organization_id = ?", run.ID, run.OrganizationID).Order("created_at ASC, id ASC").Find(&citations).Error; err != nil {
		return agentExecution{}, err
	}
	sources := make([]modelprovider.Source, 0, len(citations))
	for _, citation := range citations {
		body := citation.SourceBody
		if body == "" {
			var content model.Content
			if err := s.db.Where("id = ? AND organization_id = ?", citation.SourceID, run.OrganizationID).First(&content).Error; err != nil {
				return agentExecution{}, err
			}
			body = boundedText(content.Body, 12000)
		}
		sources = append(sources, modelprovider.Source{ID: citation.SourceID, Title: citation.Title, Excerpt: citation.Excerpt, Body: body})
	}
	timeout := s.timeout
	if configuration, err := s.Configuration(run.OrganizationID); err == nil && configuration.RequestTimeoutSeconds > 0 {
		timeout = time.Duration(configuration.RequestTimeoutSeconds) * time.Second
	}
	provider, status, _, err := s.providerForOrganization(run.OrganizationID)
	if err != nil || provider == nil || !status.Enabled || !status.Configured {
		return agentExecution{}, ErrAgentProviderDisabled
	}
	return agentExecution{runID: run.ID, organizationID: run.OrganizationID, actorUserID: run.ActorUserID, requestID: run.RequestID, agentKey: definition.Key, promptVersion: definition.SystemPolicyVersion, task: run.Task, sources: sources, timeout: timeout, provider: provider}, nil
}

func (s *AgentService) ProviderStatus() modelprovider.Status {
	return s.ProviderStatusForOrganization("")
}

func (s *AgentService) ProviderStatusForOrganization(organizationID string) modelprovider.Status {
	_, status, _, err := s.providerForOrganization(organizationID)
	if err == nil {
		return status
	}
	return s.defaultProviderStatus()
}

func (s *AgentService) Configuration(organizationID string) (AgentConfigurationView, error) {
	var configuration model.AgentConfiguration
	err := s.db.Where("organization_id = ?", organizationID).First(&configuration).Error
	provider, status, providerConfig, providerErr := s.providerForOrganization(organizationID)
	_ = provider
	if providerErr != nil {
		return AgentConfigurationView{}, providerErr
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return AgentConfigurationView{
			Enabled: true, RunLimitPerHour: s.runLimit,
			RequestTimeoutSeconds: int(s.timeout / time.Second),
			MaxSources:            10, MaxContextCharacters: 30000,
			Provider: status, ProviderConfig: providerConfig,
		}, nil
	}
	if err != nil {
		return AgentConfigurationView{}, err
	}
	if !validAgentConfiguration(AgentConfigurationInput{
		Enabled: configuration.Enabled, RunLimitPerHour: configuration.RunLimitPerHour,
		RequestTimeoutSeconds: configuration.RequestTimeoutSeconds, MaxSources: configuration.MaxSources,
		MaxContextCharacters: configuration.MaxContextCharacters,
	}) {
		return AgentConfigurationView{}, ErrAgentConfigValidation
	}
	updatedAt := configuration.UpdatedAt
	return AgentConfigurationView{
		ID: configuration.ID, Enabled: configuration.Enabled, RunLimitPerHour: configuration.RunLimitPerHour,
		RequestTimeoutSeconds: configuration.RequestTimeoutSeconds, MaxSources: configuration.MaxSources,
		MaxContextCharacters: configuration.MaxContextCharacters, Provider: status, ProviderConfig: providerConfig,
		UpdatedBy: configuration.UpdatedBy, UpdatedAt: &updatedAt,
	}, nil
}

func (s *AgentService) UpdateConfiguration(principal Principal, input AgentConfigurationInput, requestID string) (AgentConfigurationView, error) {
	if !validAgentConfiguration(input) {
		return AgentConfigurationView{}, ErrAgentConfigValidation
	}
	now := time.Now().UTC()
	var configuration model.AgentConfiguration
	err := s.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("organization_id = ?", principal.OrganizationID).First(&configuration).Error
		isNew := errors.Is(err, gorm.ErrRecordNotFound)
		if err != nil && !isNew {
			return err
		}

		driver, baseURL, modelName, apiKey, err := s.resolveConfigurationInput(configuration, isNew, input)
		if err != nil || !validAgentProviderConfiguration(driver, baseURL, apiKey, modelName, s.production) {
			return ErrAgentConfigValidation
		}
		encryptedAPIKey := configuration.ProviderAPIKey
		if strings.TrimSpace(input.APIKey) != "" {
			encryptedAPIKey, err = encryptAgentCredential(s.credentialKey, apiKey)
			if err != nil {
				return err
			}
		}
		if isNew {
			configuration = model.AgentConfiguration{
				ID: uuid.NewString(), OrganizationID: principal.OrganizationID,
				Enabled: input.Enabled, RunLimitPerHour: input.RunLimitPerHour,
				RequestTimeoutSeconds: input.RequestTimeoutSeconds, MaxSources: input.MaxSources,
				MaxContextCharacters: input.MaxContextCharacters, UpdatedBy: principal.UserID,
				Provider: driver, ProviderBaseURL: baseURL, ProviderAPIKey: encryptedAPIKey, ProviderModel: modelName,
				CreatedAt: now, UpdatedAt: now,
			}
			if err := tx.Create(&configuration).Error; err != nil {
				return err
			}
		} else {
			updates := map[string]any{
				"enabled": input.Enabled, "run_limit_per_hour": input.RunLimitPerHour,
				"request_timeout_seconds": input.RequestTimeoutSeconds, "max_sources": input.MaxSources,
				"max_context_characters": input.MaxContextCharacters, "updated_by": principal.UserID,
				"provider": driver, "provider_base_url": baseURL, "provider_model": modelName,
				"updated_at": now,
			}
			if strings.TrimSpace(input.APIKey) != "" {
				updates["provider_api_key_encrypted"] = encryptedAPIKey
			}
			if err := tx.Model(&model.AgentConfiguration{}).
				Where("id = ? AND organization_id = ?", configuration.ID, principal.OrganizationID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		return tx.Create(&model.AuditEvent{
			ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID,
			Action: "ai.config_update", TargetType: "agent_configuration", TargetID: configuration.ID,
			Result: "success", RequestID: requestID, CreatedAt: now,
		}).Error
	})
	if err != nil {
		return AgentConfigurationView{}, err
	}
	return s.Configuration(principal.OrganizationID)
}

func (s *AgentService) defaultProviderStatus() modelprovider.Status {
	if s.provider != nil {
		return s.provider.Status()
	}
	return modelprovider.Status{Provider: "disabled", Mode: "disabled", Enabled: false, Configured: false}
}

func (s *AgentService) providerForOrganization(organizationID string) (modelprovider.Provider, modelprovider.Status, AgentProviderConfigurationView, error) {
	defaultConfig := s.providerConfig
	if strings.TrimSpace(defaultConfig.Driver) == "" {
		status := s.defaultProviderStatus()
		return s.provider, status, AgentProviderConfigurationView{
			Driver: status.Provider, Model: status.Model, APIKeyConfigured: status.Configured, Source: "server",
		}, nil
	}

	effective := defaultConfig
	providerView := AgentProviderConfigurationView{Driver: effective.Driver, BaseURL: effective.BaseURL, Model: effective.Model, APIKeyConfigured: strings.TrimSpace(effective.APIKey) != "", Source: "server"}
	var configuration model.AgentConfiguration
	err := s.db.Where("organization_id = ?", organizationID).First(&configuration).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		providerView.APIKeyHint = agentCredentialHint(effective.APIKey)
	} else if err != nil {
		return nil, modelprovider.Status{}, AgentProviderConfigurationView{}, err
	} else {
		if strings.TrimSpace(configuration.Provider) != "" {
			effective.Driver = strings.ToLower(strings.TrimSpace(configuration.Provider))
			providerView.Source = "organization"
		}
		if configuration.ProviderBaseURL != "" {
			effective.BaseURL = strings.TrimRight(strings.TrimSpace(configuration.ProviderBaseURL), "/")
		}
		if configuration.ProviderModel != "" {
			effective.Model = strings.TrimSpace(configuration.ProviderModel)
		}
		if configuration.ProviderAPIKey != "" {
			decrypted, decryptErr := decryptAgentCredential(s.credentialKey, configuration.ProviderAPIKey)
			if decryptErr != nil {
				providerView.APIKeyConfigured = false
				providerView.APIKeyHint = "已保存但无法解密"
				return nil, unavailableProviderStatus(effective.Driver, effective.Model), providerView, nil
			}
			effective.APIKey = decrypted
		}
		providerView.APIKeyConfigured = strings.TrimSpace(effective.APIKey) != ""
		providerView.APIKeyHint = agentCredentialHint(effective.APIKey)
	}
	providerView.Driver = strings.ToLower(strings.TrimSpace(effective.Driver))
	providerView.BaseURL = strings.TrimRight(strings.TrimSpace(effective.BaseURL), "/")
	providerView.Model = strings.TrimSpace(effective.Model)
	if !validAgentProviderConfiguration(providerView.Driver, providerView.BaseURL, effective.APIKey, providerView.Model, s.production) {
		return nil, unavailableProviderStatus(providerView.Driver, providerView.Model), providerView, nil
	}
	provider, err := modelprovider.New(effective)
	if err != nil {
		return nil, unavailableProviderStatus(providerView.Driver, providerView.Model), providerView, nil
	}
	return provider, provider.Status(), providerView, nil
}

func unavailableProviderStatus(driver, modelName string) modelprovider.Status {
	driver = strings.ToLower(strings.TrimSpace(driver))
	mode := "real"
	if driver == "disabled" || driver == "" {
		driver = "disabled"
		mode = "disabled"
	} else if driver == "mock" {
		mode = "mock"
	}
	return modelprovider.Status{Provider: driver, Mode: mode, Model: strings.TrimSpace(modelName), Enabled: false, Configured: false}
}

func (s *AgentService) resolveConfigurationInput(configuration model.AgentConfiguration, isNew bool, input AgentConfigurationInput) (string, string, string, string, error) {
	defaultStatus := s.defaultProviderStatus()
	driver := strings.ToLower(strings.TrimSpace(input.Provider))
	baseURL := strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	modelName := strings.TrimSpace(input.Model)
	apiKey := strings.TrimSpace(input.APIKey)
	if driver == "" {
		driver = strings.ToLower(strings.TrimSpace(s.providerConfig.Driver))
		if driver == "" {
			driver = defaultStatus.Provider
		}
	}
	if baseURL == "" {
		baseURL = strings.TrimRight(strings.TrimSpace(s.providerConfig.BaseURL), "/")
	}
	if modelName == "" {
		modelName = strings.TrimSpace(s.providerConfig.Model)
		if modelName == "" {
			modelName = defaultStatus.Model
		}
	}
	if !isNew {
		if configuration.Provider != "" {
			driver = strings.ToLower(strings.TrimSpace(configuration.Provider))
		}
		if configuration.ProviderBaseURL != "" && strings.TrimSpace(input.BaseURL) == "" {
			baseURL = strings.TrimRight(strings.TrimSpace(configuration.ProviderBaseURL), "/")
		}
		if configuration.ProviderModel != "" && strings.TrimSpace(input.Model) == "" {
			modelName = strings.TrimSpace(configuration.ProviderModel)
		}
		if apiKey == "" && configuration.ProviderAPIKey != "" {
			var err error
			apiKey, err = decryptAgentCredential(s.credentialKey, configuration.ProviderAPIKey)
			if err != nil {
				return "", "", "", "", err
			}
		}
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(s.providerConfig.APIKey)
	}
	return driver, baseURL, modelName, apiKey, nil
}

func (s *AgentService) ListAgents(organizationID string) ([]AgentDefinitionView, error) {
	var definitions []model.AgentDefinition
	if err := s.db.Where("organization_id = ? AND enabled = ?", organizationID, true).
		Order("name ASC, id ASC").Find(&definitions).Error; err != nil {
		return nil, err
	}
	views := make([]AgentDefinitionView, 0, len(definitions))
	for _, definition := range definitions {
		var allowedTools []string
		_ = json.Unmarshal([]byte(definition.AllowedToolKeys), &allowedTools)
		views = append(views, AgentDefinitionView{
			ID: definition.ID, Key: definition.Key, Name: definition.Name, Purpose: definition.Purpose,
			SystemPolicyVersion: definition.SystemPolicyVersion, AllowedToolKeys: allowedTools,
			ModelProfile: definition.ModelProfile, Enabled: definition.Enabled,
		})
	}
	return views, nil
}

func (s *AgentService) SearchKnowledge(organizationID, query string, limit int) ([]AgentKnowledgeResult, error) {
	query = strings.TrimSpace(query)
	if query == "" || len([]rune(query)) > 80 {
		return nil, ErrAgentValidation
	}
	if limit == 0 {
		limit = 10
	}
	if limit < 1 || limit > 20 {
		return nil, ErrAgentValidation
	}
	pattern := "%" + query + "%"
	var contents []model.Content
	if err := s.db.Where(
		"organization_id = ? AND type = ? AND (title LIKE ? OR category LIKE ? OR excerpt LIKE ? OR body LIKE ?)",
		organizationID, "knowledge", pattern, pattern, pattern, pattern,
	).Order("updated_at DESC, id DESC").Limit(limit).Find(&contents).Error; err != nil {
		return nil, err
	}
	results := make([]AgentKnowledgeResult, 0, len(contents))
	for _, content := range contents {
		results = append(results, AgentKnowledgeResult{
			SourceType: "content", ID: content.ID, Title: content.Title,
			Excerpt: sourceExcerpt(content), Status: content.Status, UpdatedAt: content.UpdatedAt,
		})
	}
	return results, nil
}

func (s *AgentService) CreateRun(principal Principal, input AgentRunCreateInput, requestID string) (AgentRunView, error) {
	input.AgentKey = strings.TrimSpace(input.AgentKey)
	input.Task = strings.TrimSpace(input.Task)
	input.OutputMode = strings.TrimSpace(input.OutputMode)
	if input.OutputMode == "" {
		input.OutputMode = "proposal"
	}
	if input.AgentKey == "" || len([]rune(input.AgentKey)) > 64 ||
		input.Task == "" || len([]rune(input.Task)) > 1000 ||
		input.OutputMode != "proposal" || len(input.ContextRefs) < 1 || len(input.ContextRefs) > 10 {
		return AgentRunView{}, ErrAgentValidation
	}
	configuration, err := s.Configuration(principal.OrganizationID)
	if err != nil {
		return AgentRunView{}, err
	}
	_, status, _, err := s.providerForOrganization(principal.OrganizationID)
	if err != nil {
		return AgentRunView{}, err
	}
	if !configuration.Enabled {
		return AgentRunView{}, ErrAgentFeatureDisabled
	}
	if len(input.ContextRefs) > configuration.MaxSources {
		return AgentRunView{}, ErrAgentValidation
	}
	if !status.Enabled || !status.Configured {
		s.audit(principal.OrganizationID, principal.UserID, "ai.run_create", "agent_run", "", "failed", requestID)
		return AgentRunView{}, ErrAgentProviderDisabled
	}

	var definition model.AgentDefinition
	if err := s.db.Where(
		"organization_id = ? AND `key` = ? AND enabled = ?",
		principal.OrganizationID, input.AgentKey, true,
	).First(&definition).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AgentRunView{}, ErrAgentNotFound
		}
		return AgentRunView{}, err
	}

	var recentRuns int64
	if err := s.db.Model(&model.AgentRun{}).
		Where("organization_id = ? AND actor_user_id = ? AND created_at >= ?", principal.OrganizationID, principal.UserID, time.Now().UTC().Add(-time.Hour)).
		Count(&recentRuns).Error; err != nil {
		return AgentRunView{}, err
	}
	if recentRuns >= int64(configuration.RunLimitPerHour) {
		s.audit(principal.OrganizationID, principal.UserID, "ai.run_create", "agent_run", "", "quota_exceeded", requestID)
		return AgentRunView{}, ErrAgentRunQuotaExceeded
	}

	sources := make([]modelprovider.Source, 0, len(input.ContextRefs))
	citations := make([]model.AgentCitation, 0, len(input.ContextRefs))
	seen := make(map[string]struct{}, len(input.ContextRefs))
	remainingContextRunes := configuration.MaxContextCharacters
	runID := uuid.NewString()
	for _, reference := range input.ContextRefs {
		reference.Type = strings.TrimSpace(reference.Type)
		reference.ID = strings.TrimSpace(reference.ID)
		key := reference.Type + ":" + reference.ID
		if reference.Type != "content" || reference.ID == "" || len(reference.ID) > 64 {
			return AgentRunView{}, ErrAgentValidation
		}
		if _, exists := seen[key]; exists {
			return AgentRunView{}, ErrAgentValidation
		}
		seen[key] = struct{}{}
		var content model.Content
		if err := s.db.Where(
			"id = ? AND organization_id = ? AND type = ?",
			reference.ID, principal.OrganizationID, "knowledge",
		).First(&content).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return AgentRunView{}, ErrAgentSourceNotFound
			}
			return AgentRunView{}, err
		}
		excerpt := sourceExcerpt(content)
		bodyLimit := 12000
		if remainingContextRunes < bodyLimit {
			bodyLimit = remainingContextRunes
		}
		body := boundedText(content.Body, bodyLimit)
		remainingContextRunes -= len([]rune(body))
		sources = append(sources, modelprovider.Source{
			ID: content.ID, Title: content.Title, Excerpt: excerpt, Body: body,
		})
		citations = append(citations, model.AgentCitation{
			ID: uuid.NewString(), RunID: runID, OrganizationID: principal.OrganizationID,
			SourceType: "content", SourceID: content.ID, Title: content.Title, Excerpt: excerpt,
			SourceBody: body, SourceUpdatedAt: content.UpdatedAt,
		})
	}

	now := time.Now().UTC()
	run := model.AgentRun{
		ID: runID, OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID,
		AgentDefinitionID: definition.ID, Status: AgentRunQueued, Task: input.Task,
		Provider: status.Provider, Mode: status.Mode, Model: status.Model,
		PromptVersion: definition.SystemPolicyVersion, RequestID: requestID,
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&run).Error; err != nil {
			return err
		}
		if err := tx.Create(&citations).Error; err != nil {
			return err
		}
		return tx.Create(&model.AuditEvent{
			ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID,
			Action: "ai.run_create", TargetType: "agent_run", TargetID: run.ID,
			Result: "accepted", RequestID: requestID, CreatedAt: now,
		}).Error
	}); err != nil {
		return AgentRunView{}, err
	}

	return s.GetRun(principal.OrganizationID, run.ID)
}

func (s *AgentService) GetRun(organizationID, runID string) (AgentRunView, error) {
	var run model.AgentRun
	if err := s.db.Where("id = ? AND organization_id = ?", runID, organizationID).First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AgentRunView{}, ErrAgentRunNotFound
		}
		return AgentRunView{}, err
	}
	var definition model.AgentDefinition
	if err := s.db.Where("id = ? AND organization_id = ?", run.AgentDefinitionID, organizationID).First(&definition).Error; err != nil {
		return AgentRunView{}, err
	}
	var citations []model.AgentCitation
	if err := s.db.Where("run_id = ? AND organization_id = ?", run.ID, organizationID).
		Order("created_at ASC, id ASC").Find(&citations).Error; err != nil {
		return AgentRunView{}, err
	}
	citationViews := make([]AgentCitationView, 0, len(citations))
	for _, citation := range citations {
		citationViews = append(citationViews, AgentCitationView{
			ID: citation.ID, SourceType: citation.SourceType, SourceID: citation.SourceID,
			Title: citation.Title, Excerpt: citation.Excerpt, SourceUpdatedAt: citation.SourceUpdatedAt,
		})
	}
	return AgentRunView{
		ID: run.ID, AgentKey: definition.Key, AgentName: definition.Name, Status: run.Status, Task: run.Task,
		OutputTitle: run.OutputTitle, OutputExcerpt: run.OutputExcerpt, OutputMarkdown: run.OutputMarkdown,
		Provider: run.Provider, Mode: run.Mode, Model: run.Model, PromptVersion: run.PromptVersion,
		InputTokens: run.InputTokens, OutputTokens: run.OutputTokens,
		FailureCode: run.FailureCode, FailureMessage: run.FailureMessage, RequestID: run.RequestID,
		Citations: citationViews, StartedAt: run.StartedAt, CompletedAt: run.CompletedAt,
		ExpiresAt: run.ExpiresAt, CreatedAt: run.CreatedAt, UpdatedAt: run.UpdatedAt,
	}, nil
}

func (s *AgentService) CancelRun(principal Principal, runID, requestID string) (AgentRunView, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return AgentRunView{}, ErrAgentRunNotFound
	}
	now := time.Now().UTC()
	result := s.db.Model(&model.AgentRun{}).
		Where("id = ? AND organization_id = ? AND status IN ?", runID, principal.OrganizationID, []string{AgentRunQueued, AgentRunRunning}).
		Updates(map[string]any{"status": AgentRunCanceled, "completed_at": now, "failure_code": "", "failure_message": ""})
	if result.Error != nil {
		return AgentRunView{}, result.Error
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := s.db.Model(&model.AgentRun{}).Where("id = ? AND organization_id = ?", runID, principal.OrganizationID).Count(&count).Error; err != nil {
			return AgentRunView{}, err
		}
		if count == 0 {
			return AgentRunView{}, ErrAgentRunNotFound
		}
		return AgentRunView{}, ErrAgentRunNotCancelable
	}

	s.cancelMu.Lock()
	cancel := s.cancelRuns[runID]
	s.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.audit(principal.OrganizationID, principal.UserID, "ai.run_cancel", "agent_run", runID, "success", requestID)
	return s.GetRun(principal.OrganizationID, runID)
}

func (s *AgentService) execute(execution agentExecution) {
	ctx, cancel := context.WithTimeout(context.Background(), execution.timeout)
	s.cancelMu.Lock()
	s.cancelRuns[execution.runID] = cancel
	s.cancelMu.Unlock()
	defer func() {
		cancel()
		s.cancelMu.Lock()
		delete(s.cancelRuns, execution.runID)
		s.cancelMu.Unlock()
	}()

	provider := execution.provider
	if provider == nil {
		provider = s.provider
	}
	if provider == nil {
		completedAt := time.Now().UTC()
		_ = s.db.Model(&model.AgentRun{}).
			Where("id = ? AND organization_id = ? AND status = ?", execution.runID, execution.organizationID, AgentRunRunning).
			Updates(map[string]any{
				"status": AgentRunFailed, "failure_code": "ai.provider_disabled", "failure_message": "模型供应商未启用。",
				"completed_at": completedAt,
			})
		s.audit(execution.organizationID, execution.actorUserID, "ai.run_result", "agent_run", execution.runID, "failed", execution.requestID)
		return
	}
	generated, err := provider.Generate(ctx, modelprovider.GenerateRequest{
		AgentKey: execution.agentKey, PromptVersion: execution.promptVersion,
		Task: execution.task, Sources: execution.sources,
	})
	completedAt := time.Now().UTC()
	if err != nil {
		code, message := safeModelFailure(ctx, err)
		update := s.db.Model(&model.AgentRun{}).
			Where("id = ? AND organization_id = ? AND status = ?", execution.runID, execution.organizationID, AgentRunRunning).
			Updates(map[string]any{
				"status": AgentRunFailed, "failure_code": code, "failure_message": message,
				"completed_at": completedAt,
			})
		if update.Error == nil && update.RowsAffected > 0 {
			s.audit(execution.organizationID, execution.actorUserID, "ai.run_result", "agent_run", execution.runID, "failed", execution.requestID)
		}
		return
	}
	update := s.db.Model(&model.AgentRun{}).
		Where("id = ? AND organization_id = ? AND status = ?", execution.runID, execution.organizationID, AgentRunRunning).
		Updates(map[string]any{
			"status": AgentRunSucceeded, "output_title": boundedText(generated.Title, 160),
			"output_excerpt": boundedText(generated.Excerpt, 500), "output_markdown": generated.Markdown,
			"provider": generated.Provider, "mode": generated.Mode, "model": boundedText(generated.Model, 120),
			"prompt_version": boundedText(generated.PromptVersion, 64),
			"input_tokens":   generated.InputTokens, "output_tokens": generated.OutputTokens,
			"failure_code": "", "failure_message": "", "completed_at": completedAt,
		})
	if update.Error == nil && update.RowsAffected > 0 {
		s.audit(execution.organizationID, execution.actorUserID, "ai.run_result", "agent_run", execution.runID, "succeeded", execution.requestID)
	}
}

func (s *AgentService) audit(organizationID, actorUserID, action, targetType, targetID, result, requestID string) {
	_ = s.db.Create(&model.AuditEvent{
		ID: uuid.NewString(), OrganizationID: organizationID, ActorUserID: actorUserID,
		Action: action, TargetType: targetType, TargetID: targetID, Result: result,
		RequestID: requestID, CreatedAt: time.Now().UTC(),
	}).Error
}

func safeModelFailure(ctx context.Context, err error) (string, string) {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "ai.provider_timeout", "模型请求超时，请稍后重试。"
	}
	if errors.Is(err, modelprovider.ErrDisabled) {
		return "ai.provider_disabled", "模型供应商未启用。"
	}
	if errors.Is(err, modelprovider.ErrInvalidData) {
		return "ai.provider_invalid_response", "模型返回了无法使用的内容。"
	}
	return "ai.provider_unavailable", "模型供应商暂时不可用，请稍后重试。"
}

func sourceExcerpt(content model.Content) string {
	if value := strings.TrimSpace(content.Excerpt); value != "" {
		return boundedText(value, 500)
	}
	return boundedText(strings.ReplaceAll(content.Body, "\n", " "), 500)
}

func boundedText(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}

func validAgentConfiguration(input AgentConfigurationInput) bool {
	return input.RunLimitPerHour >= 1 && input.RunLimitPerHour <= 200 &&
		input.RequestTimeoutSeconds >= 5 && input.RequestTimeoutSeconds <= 120 &&
		input.MaxSources >= 1 && input.MaxSources <= 10 &&
		input.MaxContextCharacters >= 1000 && input.MaxContextCharacters <= 100000
}

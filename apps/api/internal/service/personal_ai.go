package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/modelprovider"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrChatPermission = errors.New("organization AI permission required")

type PersonalAIInput struct {
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	APIKey  string `json:"api_key"`
}

type PersonalAIView struct {
	BaseURL               string `json:"base_url"`
	Model                 string `json:"model"`
	APIKeyConfigured      bool   `json:"api_key_configured"`
	OrganizationAvailable bool   `json:"organization_available"`
}

type EditorChatInput struct {
	Source   string                      `json:"source"`
	Messages []modelprovider.ChatMessage `json:"messages"`
	Article  string                      `json:"article"`
}

type EditorChatResult struct {
	Markdown  string `json:"markdown"`
	Source    string `json:"source"`
	Model     string `json:"model"`
	RequestID string `json:"request_id"`
}

func (s *AgentService) PersonalAI(principal Principal) (PersonalAIView, error) {
	var config model.PersonalAIConfiguration
	err := s.db.Where("user_id = ?", principal.UserID).First(&config).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return PersonalAIView{}, err
	}
	view := PersonalAIView{BaseURL: config.BaseURL, Model: config.Model, APIKeyConfigured: config.APIKeyEncrypted != ""}
	allowed, err := (&AuthService{db: s.db}).HasPermission(principal, "ai:use")
	if err != nil {
		return PersonalAIView{}, err
	}
	if allowed {
		organization, err := s.Configuration(principal.OrganizationID)
		if err != nil {
			return PersonalAIView{}, err
		}
		view.OrganizationAvailable = organization.Enabled && organization.Provider.Enabled && organization.Provider.Mode == "real"
	}
	return view, nil
}

func (s *AgentService) SavePersonalAI(principal Principal, input PersonalAIInput, requestID string) (PersonalAIView, error) {
	input.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	input.Model, input.APIKey = strings.TrimSpace(input.Model), strings.TrimSpace(input.APIKey)
	if modelprovider.ValidatePublicEndpoint(input.BaseURL) != nil {
		return PersonalAIView{}, ErrAgentConfigValidation
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Serialize updates for a user so retaining an existing secret cannot lose a concurrent replacement.
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", principal.UserID).Error; err != nil {
			return err
		}
		var config model.PersonalAIConfiguration
		err := tx.First(&config, "user_id = ?", principal.UserID).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		key := input.APIKey
		if key == "" {
			// A new destination must never receive a previously stored destination's credential implicitly.
			if config.BaseURL != input.BaseURL {
				return ErrAgentConfigValidation
			}
			key, err = decryptAgentCredential(s.credentialKey, config.APIKeyEncrypted)
			if err != nil {
				return err
			}
		}
		if !validAgentProviderConfiguration("openai_compatible", input.BaseURL, key, input.Model, true) {
			return ErrAgentConfigValidation
		}
		encrypted, err := encryptAgentCredential(s.credentialKey, key)
		if err != nil {
			return err
		}
		config.UserID, config.BaseURL, config.Model, config.APIKeyEncrypted = principal.UserID, input.BaseURL, input.Model, encrypted
		if err := tx.Save(&config).Error; err != nil {
			return err
		}
		return chatAudit(tx, principal, "ai.personal_config_update", requestID)
	})
	if err != nil {
		return PersonalAIView{}, err
	}
	return s.PersonalAI(principal)
}

func (s *AgentService) DeletePersonalAI(principal Principal, requestID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", principal.UserID).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", principal.UserID).Delete(&model.PersonalAIConfiguration{}).Error; err != nil {
			return err
		}
		return chatAudit(tx, principal, "ai.personal_config_delete", requestID)
	})
}

func validEditorChat(input EditorChatInput) bool {
	if input.Source != "personal" && input.Source != "organization" {
		return false
	}
	if len(input.Messages) == 0 || len(input.Messages) > 20 || utf8.RuneCountInString(input.Article) > 30000 {
		return false
	}
	total := 0
	for i, message := range input.Messages {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		size := utf8.RuneCountInString(message.Content)
		if message.Role != role || strings.TrimSpace(message.Content) == "" || size > 12000 {
			return false
		}
		total += size
	}
	return total <= 40000 && input.Messages[len(input.Messages)-1].Role == "user"
}

func (s *AgentService) EditorChat(ctx context.Context, principal Principal, input EditorChatInput, requestID string) (EditorChatResult, error) {
	if !validEditorChat(input) {
		return EditorChatResult{}, ErrAgentValidation
	}
	var provider modelprovider.Provider
	limit, timeout := 30, 60*time.Second
	action := "ai.chat_personal"
	if input.Source == "organization" {
		allowed, err := (&AuthService{db: s.db}).HasPermission(principal, "ai:use")
		if err != nil {
			return EditorChatResult{}, err
		}
		if !allowed {
			return EditorChatResult{}, ErrChatPermission
		}
		configuration, err := s.Configuration(principal.OrganizationID)
		if err != nil {
			return EditorChatResult{}, err
		}
		if !configuration.Enabled {
			return EditorChatResult{}, ErrAgentFeatureDisabled
		}
		provider, _, _, err = s.providerForOrganization(principal.OrganizationID)
		if err != nil {
			return EditorChatResult{}, err
		}
		limit, timeout = configuration.RunLimitPerHour, time.Duration(configuration.RequestTimeoutSeconds)*time.Second
		action = "ai.chat_organization"
	} else {
		var config model.PersonalAIConfiguration
		if err := s.db.First(&config, "user_id = ?", principal.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return EditorChatResult{}, ErrAgentProviderDisabled
			}
			return EditorChatResult{}, err
		}
		key, err := decryptAgentCredential(s.credentialKey, config.APIKeyEncrypted)
		if err != nil {
			return EditorChatResult{}, ErrAgentProviderDisabled
		}
		provider, err = modelprovider.New(modelprovider.Config{Driver: "openai_compatible", BaseURL: config.BaseURL, APIKey: key, Model: config.Model, Timeout: timeout, PublicOnly: true})
		if err != nil {
			return EditorChatResult{}, ErrAgentProviderDisabled
		}
	}
	if provider == nil || !provider.Status().Enabled || provider.Status().Mode != "real" {
		return EditorChatResult{}, ErrAgentProviderDisabled
	}
	// Reserve quota before network I/O. Store only metadata, never conversations, article text or secrets.
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", principal.UserID).Error; err != nil {
			return err
		}
		var count int64
		query := tx.Model(&model.AuditEvent{}).Where("actor_user_id = ? AND action = ? AND created_at >= ?", principal.UserID, action, time.Now().UTC().Add(-time.Hour))
		if input.Source == "organization" {
			query = query.Where("organization_id = ?", principal.OrganizationID)
		}
		if err := query.Count(&count).Error; err != nil {
			return err
		}
		if input.Source == "organization" {
			var runs int64
			if err := tx.Model(&model.AgentRun{}).Where("organization_id = ? AND actor_user_id = ? AND created_at >= ?", principal.OrganizationID, principal.UserID, time.Now().UTC().Add(-time.Hour)).Count(&runs).Error; err != nil {
				return err
			}
			count += runs
		}
		if count >= int64(limit) {
			return ErrAgentRunQuotaExceeded
		}
		return chatAudit(tx, principal, action, requestID)
	})
	if err != nil {
		return EditorChatResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	response, err := provider.Generate(ctx, modelprovider.GenerateRequest{AgentKey: "editor-chat", PromptVersion: "editor-chat/v1", Task: "以下是用户主动附带的文章参考资料（可为空）：\n" + input.Article, Messages: input.Messages})
	if err != nil {
		return EditorChatResult{}, err
	}
	return EditorChatResult{Markdown: response.Markdown, Source: input.Source, Model: response.Model, RequestID: requestID}, nil
}

func chatAudit(tx *gorm.DB, principal Principal, action, requestID string) error {
	return tx.Create(&model.AuditEvent{ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID, Action: action, TargetType: "user", TargetID: principal.UserID, Result: "success", RequestID: requestID, CreatedAt: time.Now().UTC()}).Error
}

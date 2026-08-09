package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/mailadapter"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrIntegrationValidation = errors.New("integration configuration is invalid")
	ErrIntegrationSection    = errors.New("integration section is invalid")
)

type IntegrationDefaults struct {
	Environment      string
	PublicWebBaseURL string
	Email            mailadapter.Config
	Storage          storage.Config
}

type EmailIntegrationInput struct {
	Driver         string
	Host           string
	Port           int
	Username       string
	Password       string
	ClearPassword  bool
	FromAddress    string
	FromName       string
	Security       string
	TimeoutSeconds int
}

type StorageIntegrationInput struct {
	Driver         string
	Endpoint       string
	AccessKey      string
	SecretKey      string
	ClearAccessKey bool
	ClearSecretKey bool
	Bucket         string
	Region         string
	UseSSL         bool
}

type IntegrationSettingsInput struct {
	PublicWebBaseURL string
	Email            EmailIntegrationInput
	Storage          StorageIntegrationInput
}

type EmailIntegrationView struct {
	Driver             string `json:"driver"`
	Source             string `json:"source"`
	Enabled            bool   `json:"enabled"`
	Configured         bool   `json:"configured"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Username           string `json:"username"`
	PasswordConfigured bool   `json:"password_configured"`
	PasswordHint       string `json:"password_hint,omitempty"`
	FromAddress        string `json:"from_address"`
	FromName           string `json:"from_name"`
	Security           string `json:"security"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
}

type StorageIntegrationView struct {
	Driver              string `json:"driver"`
	Source              string `json:"source"`
	Configured          bool   `json:"configured"`
	Endpoint            string `json:"endpoint"`
	AccessKeyConfigured bool   `json:"access_key_configured"`
	AccessKeyHint       string `json:"access_key_hint,omitempty"`
	SecretKeyConfigured bool   `json:"secret_key_configured"`
	SecretKeyHint       string `json:"secret_key_hint,omitempty"`
	Bucket              string `json:"bucket"`
	Region              string `json:"region"`
	UseSSL              bool   `json:"use_ssl"`
}

type ManagedRuntimeItem struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	State       string `json:"state"`
	Description string `json:"description"`
}

type IntegrationSettingsView struct {
	PublicWebBaseURL string                 `json:"public_web_base_url"`
	Source           string                 `json:"source"`
	Email            EmailIntegrationView   `json:"email"`
	Storage          StorageIntegrationView `json:"storage"`
	ManagedRuntime   []ManagedRuntimeItem   `json:"managed_runtime"`
	UpdatedAt        *time.Time             `json:"updated_at,omitempty"`
}

// IntegrationService resolves organization-scoped adapters at request time.
// Deployment defaults remain a safe fallback until an administrator saves a
// web configuration for the organization.
type IntegrationService struct {
	db            *gorm.DB
	defaults      IntegrationDefaults
	defaultMail   mailadapter.Sender
	defaultStore  storage.Store
	credentialKey []byte
	production    bool
	storesMu      sync.Mutex
	stores        map[string]storage.Store
}

func NewIntegrationService(db *gorm.DB, defaults IntegrationDefaults, defaultMail mailadapter.Sender, defaultStore storage.Store, credentialSecret string) *IntegrationService {
	return &IntegrationService{
		db: db, defaults: defaults, defaultMail: defaultMail, defaultStore: defaultStore,
		credentialKey: deriveAgentCredentialKey(credentialSecret),
		production:    strings.EqualFold(strings.TrimSpace(defaults.Environment), "production"),
		stores:        map[string]storage.Store{},
	}
}

func (s *IntegrationService) Settings(ctx context.Context, organizationID string) (IntegrationSettingsView, error) {
	configuration, found, err := s.configuration(ctx, organizationID)
	if err != nil {
		return IntegrationSettingsView{}, err
	}
	return s.view(configuration, found)
}

func (s *IntegrationService) Update(ctx context.Context, principal Principal, input IntegrationSettingsInput, requestID string) (IntegrationSettingsView, error) {
	existing, found, err := s.configuration(ctx, principal.OrganizationID)
	if err != nil {
		return IntegrationSettingsView{}, err
	}
	publicURL, err := normalizePublicURL(input.PublicWebBaseURL)
	if err != nil {
		return IntegrationSettingsView{}, ErrIntegrationValidation
	}

	plainSMTPPassword := strings.TrimSpace(input.Email.Password)
	if plainSMTPPassword == "" && found && !input.Email.ClearPassword {
		plainSMTPPassword, err = decryptAgentCredential(s.credentialKey, existing.SMTPPassword)
		if err != nil {
			return IntegrationSettingsView{}, err
		}
	}
	if plainSMTPPassword == "" && !found && !input.Email.ClearPassword {
		plainSMTPPassword = s.defaults.Email.Password
	}
	if input.Email.ClearPassword {
		plainSMTPPassword = ""
	}
	emailConfig := normalizeEmailConfig(input.Email, plainSMTPPassword)
	if !validEmailConfiguration(emailConfig, s.production) {
		return IntegrationSettingsView{}, ErrIntegrationValidation
	}
	if _, err := mailadapter.New(emailConfig); err != nil {
		return IntegrationSettingsView{}, ErrIntegrationValidation
	}

	plainAccessKey := strings.TrimSpace(input.Storage.AccessKey)
	plainSecretKey := strings.TrimSpace(input.Storage.SecretKey)
	if found && plainAccessKey == "" && !input.Storage.ClearAccessKey {
		plainAccessKey, err = decryptAgentCredential(s.credentialKey, existing.S3AccessKey)
		if err != nil {
			return IntegrationSettingsView{}, err
		}
	}
	if found && plainSecretKey == "" && !input.Storage.ClearSecretKey {
		plainSecretKey, err = decryptAgentCredential(s.credentialKey, existing.S3SecretKey)
		if err != nil {
			return IntegrationSettingsView{}, err
		}
	}
	if !found && plainAccessKey == "" && !input.Storage.ClearAccessKey {
		plainAccessKey = s.defaults.Storage.AccessKey
	}
	if !found && plainSecretKey == "" && !input.Storage.ClearSecretKey {
		plainSecretKey = s.defaults.Storage.SecretKey
	}
	if input.Storage.ClearAccessKey {
		plainAccessKey = ""
	}
	if input.Storage.ClearSecretKey {
		plainSecretKey = ""
	}
	storageConfig, err := normalizeStorageConfig(input.Storage, plainAccessKey, plainSecretKey, s.defaults.Storage.LocalRoot)
	if err != nil {
		return IntegrationSettingsView{}, ErrIntegrationValidation
	}

	encryptedSMTPPassword, err := encryptAgentCredential(s.credentialKey, plainSMTPPassword)
	if err != nil {
		return IntegrationSettingsView{}, err
	}
	encryptedAccessKey, err := encryptAgentCredential(s.credentialKey, plainAccessKey)
	if err != nil {
		return IntegrationSettingsView{}, err
	}
	encryptedSecretKey, err := encryptAgentCredential(s.credentialKey, plainSecretKey)
	if err != nil {
		return IntegrationSettingsView{}, err
	}
	now := time.Now().UTC()
	configuration := model.IntegrationConfiguration{
		ID: existing.ID, OrganizationID: principal.OrganizationID, PublicWebBaseURL: publicURL,
		EmailDriver: emailConfig.Driver, SMTPHost: emailConfig.Host, SMTPPort: emailConfig.Port,
		SMTPUsername: emailConfig.Username, SMTPPassword: encryptedSMTPPassword,
		SMTPFromAddress: emailConfig.FromAddress, SMTPFromName: emailConfig.FromName,
		SMTPSecurity: emailConfig.Security, SMTPTimeoutSeconds: int(emailConfig.Timeout / time.Second),
		StorageDriver: storageConfig.Driver, S3Endpoint: storageConfig.Endpoint,
		S3AccessKey: encryptedAccessKey, S3SecretKey: encryptedSecretKey,
		S3Bucket: storageConfig.Bucket, S3Region: storageConfig.Region, S3UseSSL: storageConfig.UseSSL,
		UpdatedBy: principal.UserID, CreatedAt: existing.CreatedAt, UpdatedAt: now,
	}
	if configuration.ID == "" {
		configuration.ID = uuid.NewString()
		configuration.CreatedAt = now
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "organization_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"public_web_base_url", "email_driver", "smtp_host", "smtp_port", "smtp_username", "smtp_password_encrypted",
				"smtp_from_address", "smtp_from_name", "smtp_security", "smtp_timeout_seconds", "storage_driver",
				"s3_endpoint", "s3_access_key_encrypted", "s3_secret_key_encrypted", "s3_bucket", "s3_region", "s3_use_ssl", "updated_by", "updated_at",
			}),
		}).Create(&configuration).Error; err != nil {
			return err
		}
		return tx.Create(&model.AuditEvent{
			ID: uuid.NewString(), OrganizationID: principal.OrganizationID, ActorUserID: principal.UserID,
			Action: "organization.integrations_update", TargetType: "integration_configuration", TargetID: configuration.ID,
			Result: "success", RequestID: requestID, CreatedAt: now,
		}).Error
	})
	if err != nil {
		return IntegrationSettingsView{}, err
	}
	return s.view(configuration, true)
}

func (s *IntegrationService) Test(ctx context.Context, organizationID, section string) error {
	configuration, found, err := s.configuration(ctx, organizationID)
	if err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(section)) {
	case "email":
		cfg, err := s.emailConfig(configuration, found)
		if err != nil {
			return err
		}
		return mailadapter.Probe(ctx, cfg)
	case "storage":
		cfg, err := s.storageConfig(configuration, found, "")
		if err != nil {
			return err
		}
		return storage.Probe(ctx, cfg)
	default:
		return ErrIntegrationSection
	}
}

func (s *IntegrationService) MailSender(ctx context.Context, organizationID string) (mailadapter.Sender, error) {
	configuration, found, err := s.configuration(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if !found {
		return s.defaultMail, nil
	}
	cfg, err := s.emailConfig(configuration, true)
	if err != nil {
		return nil, err
	}
	return mailadapter.New(cfg)
}

func (s *IntegrationService) PublicWebBaseURL(ctx context.Context, organizationID string) string {
	configuration, found, err := s.configuration(ctx, organizationID)
	if err == nil && found && strings.TrimSpace(configuration.PublicWebBaseURL) != "" {
		return strings.TrimRight(configuration.PublicWebBaseURL, "/")
	}
	return strings.TrimRight(s.defaults.PublicWebBaseURL, "/")
}

func (s *IntegrationService) Storage(ctx context.Context, organizationID, requestedDriver string) (storage.Store, error) {
	configuration, found, err := s.configuration(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	cfg, err := s.storageConfig(configuration, found, requestedDriver)
	if err != nil {
		return nil, err
	}
	if !found && (requestedDriver == "" || strings.EqualFold(requestedDriver, s.defaultStore.Driver())) {
		return s.defaultStore, nil
	}
	key := storageCacheKey(organizationID, cfg)
	s.storesMu.Lock()
	defer s.storesMu.Unlock()
	if store := s.stores[key]; store != nil {
		return store, nil
	}
	initializeContext, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	store, err := storage.New(initializeContext, cfg)
	if err != nil {
		return nil, err
	}
	s.stores[key] = store
	return store, nil
}

func (s *IntegrationService) configuration(ctx context.Context, organizationID string) (model.IntegrationConfiguration, bool, error) {
	var configuration model.IntegrationConfiguration
	err := s.db.WithContext(ctx).Where("organization_id = ?", organizationID).First(&configuration).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.IntegrationConfiguration{}, false, nil
	}
	return configuration, err == nil, err
}

func (s *IntegrationService) view(configuration model.IntegrationConfiguration, found bool) (IntegrationSettingsView, error) {
	source := "deployment"
	publicURL := strings.TrimRight(s.defaults.PublicWebBaseURL, "/")
	updatedAt := (*time.Time)(nil)
	if found {
		source = "web"
		publicURL = strings.TrimRight(configuration.PublicWebBaseURL, "/")
		updatedAt = &configuration.UpdatedAt
	}
	emailConfig, err := s.emailConfig(configuration, found)
	if err != nil {
		return IntegrationSettingsView{}, err
	}
	storageConfig, err := s.storageConfig(configuration, found, "")
	if err != nil {
		return IntegrationSettingsView{}, err
	}
	mailStatus, statusErr := mailadapter.New(emailConfig)
	emailConfigured := statusErr == nil && mailStatus.Status().Configured
	var smtpPassword, accessKey, secretKey string
	if found {
		smtpPassword, err = decryptAgentCredential(s.credentialKey, configuration.SMTPPassword)
		if err != nil {
			return IntegrationSettingsView{}, err
		}
		accessKey, err = decryptAgentCredential(s.credentialKey, configuration.S3AccessKey)
		if err != nil {
			return IntegrationSettingsView{}, err
		}
		secretKey, err = decryptAgentCredential(s.credentialKey, configuration.S3SecretKey)
		if err != nil {
			return IntegrationSettingsView{}, err
		}
	} else {
		smtpPassword = s.defaults.Email.Password
		accessKey = s.defaults.Storage.AccessKey
		secretKey = s.defaults.Storage.SecretKey
	}
	storageConfigured := storageConfig.Driver == "local" || validS3Configuration(storageConfig)
	return IntegrationSettingsView{
		PublicWebBaseURL: publicURL, Source: source, UpdatedAt: updatedAt,
		Email: EmailIntegrationView{
			Driver: emailConfig.Driver, Source: source, Enabled: emailConfig.Driver == "smtp", Configured: emailConfigured,
			Host: emailConfig.Host, Port: emailConfig.Port, Username: emailConfig.Username,
			PasswordConfigured: smtpPassword != "", PasswordHint: agentCredentialHint(smtpPassword),
			FromAddress: emailConfig.FromAddress, FromName: emailConfig.FromName, Security: emailConfig.Security,
			TimeoutSeconds: int(emailConfig.Timeout / time.Second),
		},
		Storage: StorageIntegrationView{
			Driver: storageConfig.Driver, Source: source, Configured: storageConfigured, Endpoint: storageConfig.Endpoint,
			AccessKeyConfigured: accessKey != "", AccessKeyHint: agentCredentialHint(accessKey),
			SecretKeyConfigured: secretKey != "", SecretKeyHint: agentCredentialHint(secretKey),
			Bucket: storageConfig.Bucket, Region: storageConfig.Region, UseSSL: storageConfig.UseSSL,
		},
		ManagedRuntime: []ManagedRuntimeItem{
			{Key: "database", Label: "MySQL 数据库", State: "deployment", Description: "启动根基，由部署维护；网页仅使用当前连接。"},
			{Key: "cache", Label: "Redis 缓存", State: "deployment", Description: "启动根基，由部署维护；可通过健康检查确认状态。"},
			{Key: "security", Label: "JWT、CORS 与限流", State: "deployment", Description: "安全边界，由部署维护，修改后需要重启 API。"},
			{Key: "server", Label: "服务器命令适配器", State: "deferred", Description: "RCON 已按项目计划延期，当前保持安全 Mock。"},
		},
	}, nil
}

func (s *IntegrationService) emailConfig(configuration model.IntegrationConfiguration, found bool) (mailadapter.Config, error) {
	if !found {
		return s.defaults.Email, nil
	}
	password, err := decryptAgentCredential(s.credentialKey, configuration.SMTPPassword)
	if err != nil {
		return mailadapter.Config{}, err
	}
	return mailadapter.Config{
		Driver: configuration.EmailDriver, Host: configuration.SMTPHost, Port: configuration.SMTPPort,
		Username: configuration.SMTPUsername, Password: password, FromAddress: configuration.SMTPFromAddress,
		FromName: configuration.SMTPFromName, Security: configuration.SMTPSecurity,
		Timeout: time.Duration(configuration.SMTPTimeoutSeconds) * time.Second,
	}, nil
}

func (s *IntegrationService) storageConfig(configuration model.IntegrationConfiguration, found bool, requestedDriver string) (storage.Config, error) {
	requestedDriver = strings.ToLower(strings.TrimSpace(requestedDriver))
	if !found {
		cfg := s.defaults.Storage
		if requestedDriver == "local" && strings.ToLower(cfg.Driver) != "local" {
			cfg = storage.Config{Driver: "local", LocalRoot: s.defaults.Storage.LocalRoot}
		} else if requestedDriver != "" && requestedDriver != strings.ToLower(cfg.Driver) {
			return storage.Config{}, fmt.Errorf("storage driver %q is not configured", requestedDriver)
		}
		return cfg, nil
	}
	driver := strings.ToLower(strings.TrimSpace(configuration.StorageDriver))
	if requestedDriver != "" {
		driver = requestedDriver
	}
	if driver == "local" {
		return storage.Config{Driver: "local", LocalRoot: s.defaults.Storage.LocalRoot}, nil
	}
	if driver != "s3" {
		return storage.Config{}, fmt.Errorf("storage driver %q is not configured", driver)
	}
	accessKey, err := decryptAgentCredential(s.credentialKey, configuration.S3AccessKey)
	if err != nil {
		return storage.Config{}, err
	}
	secretKey, err := decryptAgentCredential(s.credentialKey, configuration.S3SecretKey)
	if err != nil {
		return storage.Config{}, err
	}
	return storage.Config{Driver: "s3", LocalRoot: s.defaults.Storage.LocalRoot, Endpoint: configuration.S3Endpoint, AccessKey: accessKey, SecretKey: secretKey, Bucket: configuration.S3Bucket, Region: configuration.S3Region, UseSSL: configuration.S3UseSSL}, nil
}

func normalizeEmailConfig(input EmailIntegrationInput, password string) mailadapter.Config {
	driver := strings.ToLower(strings.TrimSpace(input.Driver))
	if driver == "" {
		driver = "disabled"
	}
	timeout := input.TimeoutSeconds
	if timeout == 0 {
		timeout = 8
	}
	return mailadapter.Config{
		Driver: driver, Host: strings.TrimSpace(input.Host), Port: input.Port,
		Username: strings.TrimSpace(input.Username), Password: password,
		FromAddress: strings.TrimSpace(input.FromAddress), FromName: strings.TrimSpace(input.FromName),
		Security: strings.ToLower(strings.TrimSpace(input.Security)), Timeout: time.Duration(timeout) * time.Second,
	}
}

func validEmailConfiguration(cfg mailadapter.Config, production bool) bool {
	if cfg.Driver == "disabled" {
		return true
	}
	if cfg.Driver != "smtp" || len(cfg.Host) > 255 || len(cfg.Username) > 255 || len(cfg.Password) > 4096 || len(cfg.FromName) > 160 || cfg.Timeout < 2*time.Second || cfg.Timeout > 60*time.Second {
		return false
	}
	if production && cfg.Security == "none" {
		return false
	}
	return validEmailAddress(cfg.FromAddress)
}

func normalizeStorageConfig(input StorageIntegrationInput, accessKey, secretKey, localRoot string) (storage.Config, error) {
	driver := strings.ToLower(strings.TrimSpace(input.Driver))
	if driver == "" {
		driver = "local"
	}
	cfg := storage.Config{Driver: driver, LocalRoot: localRoot, Endpoint: strings.TrimSpace(input.Endpoint), AccessKey: accessKey, SecretKey: secretKey, Bucket: strings.TrimSpace(input.Bucket), Region: strings.TrimSpace(input.Region), UseSSL: input.UseSSL}
	if driver == "local" {
		return cfg, nil
	}
	if driver != "s3" {
		return storage.Config{}, ErrIntegrationValidation
	}
	endpoint, useSSL, err := normalizeS3Endpoint(cfg.Endpoint, cfg.UseSSL)
	if err != nil {
		return storage.Config{}, err
	}
	cfg.Endpoint, cfg.UseSSL = endpoint, useSSL
	if !validS3Configuration(cfg) {
		return storage.Config{}, ErrIntegrationValidation
	}
	return cfg, nil
}

func validS3Configuration(cfg storage.Config) bool {
	return strings.TrimSpace(cfg.Endpoint) != "" && strings.TrimSpace(cfg.AccessKey) != "" && strings.TrimSpace(cfg.SecretKey) != "" && strings.TrimSpace(cfg.Bucket) != "" && len(cfg.Endpoint) <= 500 && len(cfg.AccessKey) <= 4096 && len(cfg.SecretKey) <= 4096 && len(cfg.Bucket) <= 255 && len(cfg.Region) <= 120
}

func normalizeS3Endpoint(value string, useSSL bool) (string, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", useSSL, ErrIntegrationValidation
	}
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil || parsed.User != nil || parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return "", useSSL, ErrIntegrationValidation
		}
		return parsed.Host, parsed.Scheme == "https", nil
	}
	if strings.ContainsAny(value, "/?#@") {
		return "", useSSL, ErrIntegrationValidation
	}
	return value, useSSL, nil
}

func normalizePublicURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" || len(value) > 500 {
		return "", ErrIntegrationValidation
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", ErrIntegrationValidation
	}
	return value, nil
}

func storageCacheKey(organizationID string, cfg storage.Config) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{cfg.Driver, cfg.LocalRoot, cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, cfg.Bucket, cfg.Region, strconv.FormatBool(cfg.UseSSL)}, "\x00")))
	return organizationID + ":" + fmt.Sprintf("%x", digest[:])
}

func validEmailAddress(value string) bool {
	address, err := mail.ParseAddress(strings.TrimSpace(value))
	return err == nil && address.Address != ""
}

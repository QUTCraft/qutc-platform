package model

import "time"

type Organization struct {
	ID                        string `gorm:"primaryKey;type:char(36)"`
	Slug                      string `gorm:"uniqueIndex;size:64;not null"`
	Name                      string `gorm:"size:160;not null"`
	ShortName                 string `gorm:"size:40;not null;default:''"`
	Tagline                   string `gorm:"size:160;not null;default:''"`
	Introduction              string `gorm:"size:2000;not null;default:''"`
	ContactEmail              string `gorm:"size:254;not null;default:''"`
	FilingNumber              string `gorm:"size:80;not null;default:''"`
	LogoAssetID               string `gorm:"index;type:char(36);not null;default:''"`
	SocialLinksJSON           string `gorm:"type:text;not null;default:'[]'"`
	IsPublic                  bool   `gorm:"index;not null;default:true"`
	InvitationSubjectTemplate string `gorm:"size:255;not null;default:''"`
	InvitationBodyTemplate    string `gorm:"type:text;not null"`
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type User struct {
	ID                    string `gorm:"primaryKey;type:char(36)"`
	Email                 string `gorm:"uniqueIndex;size:254;not null"`
	DisplayName           string `gorm:"size:80;not null"`
	Bio                   string `gorm:"size:500;not null;default:''"`
	AvatarURL             string `gorm:"size:500;not null;default:''"`
	PasswordHash          string `gorm:"size:255;not null"`
	State                 string `gorm:"size:24;not null;default:active"`
	DefaultOrganizationID string `gorm:"index;type:char(36);not null;default:''"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type Role struct {
	ID          string `gorm:"primaryKey;type:char(36)"`
	Key         string `gorm:"uniqueIndex;size:64;not null"`
	DisplayName string `gorm:"size:80;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Permission struct {
	ID          string `gorm:"primaryKey;type:char(36)"`
	Key         string `gorm:"uniqueIndex;size:96;not null"`
	DisplayName string `gorm:"size:120;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RolePermission struct {
	RoleID       string `gorm:"primaryKey;type:char(36)"`
	PermissionID string `gorm:"primaryKey;type:char(36)"`
}

type Membership struct {
	ID             string `gorm:"primaryKey;type:char(36)"`
	OrganizationID string `gorm:"index;type:char(36);not null"`
	UserID         string `gorm:"index;type:char(36);not null"`
	State          string `gorm:"size:24;not null;default:active"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MembershipEvent struct {
	ID           string    `gorm:"primaryKey;type:char(36)"`
	MembershipID string    `gorm:"index;type:char(36);not null"`
	State        string    `gorm:"size:24;not null"`
	Reason       string    `gorm:"size:160;not null;default:''"`
	CreatedAt    time.Time `gorm:"index"`
}

type Invitation struct {
	ID             string     `gorm:"primaryKey;type:char(36)"`
	OrganizationID string     `gorm:"index;type:char(36);not null"`
	InvitedBy      string     `gorm:"index;type:char(36);not null"`
	Email          string     `gorm:"index;size:254;not null"`
	Role           string     `gorm:"size:24;not null;default:member"`
	TokenHash      string     `gorm:"uniqueIndex;type:char(64);not null"`
	ExpiresAt      time.Time  `gorm:"index;not null"`
	AcceptedAt     *time.Time `gorm:"index"`
	RevokedAt      *time.Time `gorm:"index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type InvitationDelivery struct {
	ID             string     `gorm:"primaryKey;type:char(36)"`
	InvitationID   string     `gorm:"uniqueIndex;type:char(36);not null"`
	OrganizationID string     `gorm:"index;type:char(36);not null"`
	Channel        string     `gorm:"size:24;not null;default:email"`
	Adapter        string     `gorm:"size:32;not null"`
	Status         string     `gorm:"index;size:24;not null"`
	Attempts       int        `gorm:"not null;default:0"`
	LastError      string     `gorm:"size:500;not null;default:''"`
	LastAttemptAt  *time.Time `gorm:"index"`
	SentAt         *time.Time `gorm:"index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type MembershipRole struct {
	MembershipID string `gorm:"primaryKey;type:char(36)"`
	RoleID       string `gorm:"primaryKey;type:char(36)"`
}

type RefreshToken struct {
	ID             string     `gorm:"primaryKey;type:char(36)"`
	UserID         string     `gorm:"index;type:char(36);not null"`
	OrganizationID string     `gorm:"index;type:char(36);not null;default:''"`
	TokenHash      string     `gorm:"uniqueIndex;type:char(64);not null"`
	ExpiresAt      time.Time  `gorm:"index;not null"`
	RevokedAt      *time.Time `gorm:"index"`
	CreatedAt      time.Time
}

type AuditEvent struct {
	ID             string    `gorm:"primaryKey;type:char(36)"`
	OrganizationID string    `gorm:"index;type:char(36);not null"`
	ActorUserID    string    `gorm:"index;type:char(36);not null"`
	Action         string    `gorm:"size:96;not null"`
	TargetType     string    `gorm:"size:64;not null"`
	TargetID       string    `gorm:"type:char(36);default:''"`
	Result         string    `gorm:"size:24;not null"`
	RequestID      string    `gorm:"index;size:64;not null"`
	CreatedAt      time.Time `gorm:"index"`
}

type Content struct {
	ID                   string     `gorm:"primaryKey;type:char(36)"`
	OrganizationID       string     `gorm:"index;type:char(36);not null"`
	AuthorUserID         string     `gorm:"index;type:char(36);not null"`
	Title                string     `gorm:"size:160;not null"`
	Type                 string     `gorm:"size:24;not null"`
	Category             string     `gorm:"size:64;not null;default:''"`
	KnowledgeDirectoryID *string    `gorm:"index;type:char(36)"`
	Status               string     `gorm:"index;size:24;not null;default:draft"`
	Excerpt              string     `gorm:"size:500"`
	Body                 string     `gorm:"type:longtext"`
	PublishedAt          *time.Time `gorm:"index"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type KnowledgeDirectory struct {
	ID             string `gorm:"primaryKey;type:char(36)"`
	OrganizationID string `gorm:"index;type:char(36);not null"`
	ParentID       string `gorm:"index;type:char(36);not null;default:''"`
	Name           string `gorm:"size:120;not null"`
	Slug           string `gorm:"size:120;not null"`
	Description    string `gorm:"size:500;not null;default:''"`
	SortOrder      int    `gorm:"not null;default:0"`
	IsPublic       bool   `gorm:"index;not null;default:false"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Project struct {
	ID             string `gorm:"primaryKey;type:char(36)"`
	OrganizationID string `gorm:"index;type:char(36);not null"`
	OwnerUserID    string `gorm:"index;type:char(36);not null"`
	Title          string `gorm:"size:160;not null"`
	Summary        string `gorm:"size:500;not null"`
	Status         string `gorm:"index;size:24;not null;default:research"`
	Tags           string `gorm:"type:text"`
	IsPublic       bool   `gorm:"index;not null;default:false"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ProjectMember struct {
	ProjectID string `gorm:"primaryKey;type:char(36)"`
	UserID    string `gorm:"primaryKey;type:char(36)"`
	Role      string `gorm:"size:64;not null;default:member"`
	CreatedAt time.Time
}

type ProjectMilestone struct {
	ID          string     `gorm:"primaryKey;type:char(36)"`
	ProjectID   string     `gorm:"index;type:char(36);not null"`
	Title       string     `gorm:"size:160;not null"`
	Status      string     `gorm:"size:24;not null;default:planned"`
	DueAt       *time.Time `gorm:"index"`
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type MediaAsset struct {
	ID               string     `gorm:"primaryKey;type:char(36)"`
	OrganizationID   string     `gorm:"index;type:char(36);not null"`
	ContentID        string     `gorm:"index;type:char(36)"`
	UploadedBy       string     `gorm:"index;type:char(36);not null"`
	OriginalName     string     `gorm:"size:255;not null"`
	StoredName       string     `gorm:"size:255;not null"`
	MimeType         string     `gorm:"size:120;not null"`
	SizeBytes        int64      `gorm:"not null"`
	StorageDriver    string     `gorm:"size:16;not null;default:local"`
	StoragePath      string     `gorm:"size:500;not null"`
	Provider         string     `gorm:"size:24;not null;default:local"`
	ExternalURL      string     `gorm:"size:500;not null;default:''"`
	DownloadCount    int64      `gorm:"not null;default:0"`
	LastDownloadedAt *time.Time `gorm:"index"`
	CreatedAt        time.Time
}

type ContentRevision struct {
	ID                   string     `gorm:"primaryKey;type:char(36)"`
	OrganizationID       string     `gorm:"index;type:char(36);not null"`
	ContentID            string     `gorm:"index;type:char(36);not null"`
	Version              int        `gorm:"uniqueIndex:idx_content_revision_version;not null"`
	CreatedBy            string     `gorm:"index;type:char(36);not null"`
	Reason               string     `gorm:"size:32;not null"`
	Title                string     `gorm:"size:160;not null"`
	Type                 string     `gorm:"size:24;not null"`
	Category             string     `gorm:"size:64;not null;default:''"`
	KnowledgeDirectoryID string     `gorm:"type:char(36);not null;default:''"`
	Status               string     `gorm:"size:24;not null"`
	Excerpt              string     `gorm:"size:500;not null;default:''"`
	Body                 string     `gorm:"type:longtext;not null"`
	PublishedAt          *time.Time `gorm:"index"`
	CreatedAt            time.Time  `gorm:"index"`
}

type ContentReviewRequest struct {
	ID              string     `gorm:"primaryKey;type:char(36)"`
	OrganizationID  string     `gorm:"index;type:char(36);not null"`
	ContentID       string     `gorm:"index;type:char(36);not null"`
	RevisionID      string     `gorm:"index;type:char(36);not null"`
	RequesterUserID string     `gorm:"index;type:char(36);not null"`
	Type            string     `gorm:"index;size:24;not null"`
	Status          string     `gorm:"index;size:24;not null;default:pending"`
	Note            string     `gorm:"size:1000;not null;default:''"`
	Feedback        string     `gorm:"size:1000;not null;default:''"`
	ReviewerUserID  string     `gorm:"index;type:char(36);not null;default:''"`
	ReviewedAt      *time.Time `gorm:"index"`
	CreatedAt       time.Time  `gorm:"index"`
	UpdatedAt       time.Time
}

type NotificationOutbox struct {
	ID             string     `gorm:"primaryKey;type:char(36)"`
	OrganizationID string     `gorm:"index;type:char(36);not null"`
	EventType      string     `gorm:"size:64;not null"`
	TargetType     string     `gorm:"size:64;not null"`
	TargetID       string     `gorm:"index;type:char(36);not null"`
	RecipientEmail string     `gorm:"size:254;not null"`
	Status         string     `gorm:"index;size:24;not null;default:pending"`
	Attempts       int        `gorm:"not null;default:0"`
	LastError      string     `gorm:"size:500;not null;default:''"`
	AvailableAt    time.Time  `gorm:"index"`
	LastAttemptAt  *time.Time `gorm:"index"`
	SentAt         *time.Time `gorm:"index"`
	CreatedAt      time.Time  `gorm:"index"`
	UpdatedAt      time.Time
}

type Application struct {
	ID             string     `gorm:"primaryKey;type:char(36)"`
	OrganizationID string     `gorm:"index;type:char(36);not null"`
	Type           string     `gorm:"index;size:24;not null;default:whitelist"`
	ClassName      string     `gorm:"size:120;not null"`
	ApplicantName  string     `gorm:"size:80;not null"`
	GameID         string     `gorm:"size:80;not null"`
	QQNumber       string     `gorm:"size:32;not null"`
	Email          string     `gorm:"size:254;not null"`
	Note           string     `gorm:"size:500;not null;default:''"`
	Status         string     `gorm:"index;size:24;not null;default:pending"`
	DecidedAt      *time.Time `gorm:"index"`
	DecidedBy      string     `gorm:"type:char(36);not null;default:''"`
	DecisionReason string     `gorm:"size:500;not null;default:''"`
	CreatedAt      time.Time  `gorm:"index"`
	UpdatedAt      time.Time
}

type PortalConfiguration struct {
	ID                 string     `gorm:"primaryKey;type:char(36)"`
	OrganizationID     string     `gorm:"uniqueIndex;type:char(36);not null"`
	DraftManifestJSON  string     `gorm:"type:longtext;not null"`
	ActiveManifestJSON string     `gorm:"type:longtext;not null"`
	UpdatedBy          string     `gorm:"index;type:char(36);not null"`
	ActivatedBy        *string    `gorm:"index;type:char(36)"`
	ActivatedAt        *time.Time `gorm:"index"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type AgentDefinition struct {
	ID                  string `gorm:"primaryKey;type:char(36)"`
	OrganizationID      string `gorm:"uniqueIndex:idx_agent_definition_org_key;type:char(36);not null"`
	Key                 string `gorm:"uniqueIndex:idx_agent_definition_org_key;size:64;not null"`
	Name                string `gorm:"size:120;not null"`
	Purpose             string `gorm:"size:500;not null"`
	SystemPolicyVersion string `gorm:"size:64;not null"`
	AllowedToolKeys     string `gorm:"type:text;not null"`
	ModelProfile        string `gorm:"size:64;not null"`
	Enabled             bool   `gorm:"index;not null;default:true"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type AgentConfiguration struct {
	ID                    string `gorm:"primaryKey;type:char(36)"`
	OrganizationID        string `gorm:"uniqueIndex;type:char(36);not null"`
	Provider              string `gorm:"size:32;not null;default:''"`
	ProviderBaseURL       string `gorm:"column:provider_base_url;size:500;not null;default:''"`
	ProviderAPIKey        string `gorm:"column:provider_api_key_encrypted;type:text;not null"`
	ProviderModel         string `gorm:"column:provider_model;size:120;not null;default:''"`
	Enabled               bool   `gorm:"index;not null"`
	RunLimitPerHour       int    `gorm:"not null;default:20"`
	RequestTimeoutSeconds int    `gorm:"not null;default:30"`
	MaxSources            int    `gorm:"not null;default:10"`
	MaxContextCharacters  int    `gorm:"not null;default:30000"`
	UpdatedBy             string `gorm:"index;type:char(36);not null"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// IntegrationConfiguration contains organization-scoped runtime adapters that
// administrators can manage without editing deployment files. Credentials are
// encrypted before persistence and are never serialized directly to clients.
type IntegrationConfiguration struct {
	ID                 string `gorm:"primaryKey;type:char(36)"`
	OrganizationID     string `gorm:"uniqueIndex;type:char(36);not null"`
	PublicWebBaseURL   string `gorm:"size:500;not null;default:''"`
	EmailDriver        string `gorm:"size:24;not null;default:disabled"`
	SMTPHost           string `gorm:"size:255;not null;default:''"`
	SMTPPort           int    `gorm:"not null;default:587"`
	SMTPUsername       string `gorm:"size:255;not null;default:''"`
	SMTPPassword       string `gorm:"column:smtp_password_encrypted;type:text;not null"`
	SMTPFromAddress    string `gorm:"size:254;not null;default:''"`
	SMTPFromName       string `gorm:"size:160;not null;default:''"`
	SMTPSecurity       string `gorm:"size:24;not null;default:starttls"`
	SMTPTimeoutSeconds int    `gorm:"not null;default:8"`
	StorageDriver      string `gorm:"size:24;not null;default:local"`
	S3Endpoint         string `gorm:"size:500;not null;default:''"`
	S3AccessKey        string `gorm:"column:s3_access_key_encrypted;type:text;not null"`
	S3SecretKey        string `gorm:"column:s3_secret_key_encrypted;type:text;not null"`
	S3Bucket           string `gorm:"size:255;not null;default:''"`
	S3Region           string `gorm:"size:120;not null;default:''"`
	S3UseSSL           bool   `gorm:"not null;default:true"`
	UpdatedBy          string `gorm:"index;type:char(36);not null"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type AgentRun struct {
	ID                string     `gorm:"primaryKey;type:char(36)"`
	OrganizationID    string     `gorm:"index;type:char(36);not null"`
	ActorUserID       string     `gorm:"index;type:char(36);not null"`
	AgentDefinitionID string     `gorm:"index;type:char(36);not null"`
	Status            string     `gorm:"index;size:24;not null"`
	Task              string     `gorm:"type:text;not null"`
	OutputTitle       string     `gorm:"size:160;not null;default:''"`
	OutputExcerpt     string     `gorm:"size:500;not null;default:''"`
	OutputMarkdown    string     `gorm:"type:longtext;not null"`
	Provider          string     `gorm:"size:32;not null"`
	Mode              string     `gorm:"size:24;not null"`
	Model             string     `gorm:"size:120;not null"`
	PromptVersion     string     `gorm:"size:64;not null"`
	InputTokens       int        `gorm:"not null;default:0"`
	OutputTokens      int        `gorm:"not null;default:0"`
	FailureCode       string     `gorm:"size:64;not null;default:''"`
	FailureMessage    string     `gorm:"size:500;not null;default:''"`
	RequestID         string     `gorm:"index;size:64;not null"`
	StartedAt         *time.Time `gorm:"index"`
	CompletedAt       *time.Time `gorm:"index"`
	ExpiresAt         time.Time  `gorm:"index;not null"`
	CreatedAt         time.Time  `gorm:"index"`
	UpdatedAt         time.Time
}

type AgentCitation struct {
	ID              string    `gorm:"primaryKey;type:char(36)"`
	RunID           string    `gorm:"uniqueIndex:idx_agent_citation_run_source;type:char(36);not null"`
	OrganizationID  string    `gorm:"index;type:char(36);not null"`
	SourceType      string    `gorm:"uniqueIndex:idx_agent_citation_run_source;size:32;not null"`
	SourceID        string    `gorm:"uniqueIndex:idx_agent_citation_run_source;size:64;not null"`
	Title           string    `gorm:"size:160;not null"`
	Excerpt         string    `gorm:"size:500;not null"`
	SourceBody      string    `gorm:"type:longtext;not null"`
	SourceUpdatedAt time.Time `gorm:"not null"`
	CreatedAt       time.Time
}

type ActivityPlan struct {
	ID                    string     `gorm:"primaryKey;type:char(36)"`
	OrganizationID        string     `gorm:"index;type:char(36);not null"`
	ActorUserID           string     `gorm:"index;type:char(36);not null"`
	AgentRunID            string     `gorm:"uniqueIndex;type:char(36);not null"`
	Title                 string     `gorm:"size:160;not null"`
	Objective             string     `gorm:"size:1000;not null"`
	Audience              string     `gorm:"size:500;not null"`
	Venue                 string     `gorm:"size:300;not null;default:''"`
	StartsAt              *time.Time `gorm:"index"`
	EndsAt                *time.Time `gorm:"index"`
	ExpectedParticipants  int        `gorm:"not null;default:0"`
	Budget                string     `gorm:"size:200;not null;default:''"`
	Constraints           string     `gorm:"size:2000;not null;default:''"`
	ContextRefsJSON       string     `gorm:"type:text;not null"`
	Status                string     `gorm:"index;size:24;not null"`
	ApprovedBy            *string    `gorm:"index;type:char(36)"`
	ApprovedAt            *time.Time `gorm:"index"`
	ProjectID             *string    `gorm:"index;type:char(36)"`
	AnnouncementContentID *string    `gorm:"index;type:char(36)"`
	ApprovedActionsJSON   string     `gorm:"type:text;not null"`
	CreatedAt             time.Time  `gorm:"index"`
	UpdatedAt             time.Time
}

type ActivityPlanEvaluation struct {
	ID             string    `gorm:"primaryKey;type:char(36)"`
	OrganizationID string    `gorm:"index;type:char(36);not null"`
	PlanID         string    `gorm:"uniqueIndex:idx_activity_plan_evaluation_reviewer;index;type:char(36);not null"`
	ReviewerUserID string    `gorm:"uniqueIndex:idx_activity_plan_evaluation_reviewer;index;type:char(36);not null"`
	Accuracy       int       `gorm:"not null"`
	Feasibility    int       `gorm:"not null"`
	CampusFit      int       `gorm:"not null"`
	Clarity        int       `gorm:"not null"`
	Adoptability   int       `gorm:"not null"`
	Notes          string    `gorm:"size:1000;not null;default:''"`
	CreatedAt      time.Time `gorm:"index"`
	UpdatedAt      time.Time
}

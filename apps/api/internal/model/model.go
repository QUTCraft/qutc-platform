package model

import "time"

type Organization struct {
	ID        string `gorm:"primaryKey;type:char(36)"`
	Slug      string `gorm:"uniqueIndex;size:64;not null"`
	Name      string `gorm:"size:160;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	ID           string `gorm:"primaryKey;type:char(36)"`
	Email        string `gorm:"uniqueIndex;size:254;not null"`
	DisplayName  string `gorm:"size:80;not null"`
	Bio          string `gorm:"size:500;not null;default:''"`
	AvatarURL    string `gorm:"size:500;not null;default:''"`
	PasswordHash string `gorm:"size:255;not null"`
	State        string `gorm:"size:24;not null;default:active"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
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

type MembershipRole struct {
	MembershipID string `gorm:"primaryKey;type:char(36)"`
	RoleID       string `gorm:"primaryKey;type:char(36)"`
}

type RefreshToken struct {
	ID        string     `gorm:"primaryKey;type:char(36)"`
	UserID    string     `gorm:"index;type:char(36);not null"`
	TokenHash string     `gorm:"uniqueIndex;type:char(64);not null"`
	ExpiresAt time.Time  `gorm:"index;not null"`
	RevokedAt *time.Time `gorm:"index"`
	CreatedAt time.Time
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
	ID             string `gorm:"primaryKey;type:char(36)"`
	OrganizationID string `gorm:"index;type:char(36);not null"`
	ContentID      string `gorm:"index;type:char(36)"`
	UploadedBy     string `gorm:"index;type:char(36);not null"`
	OriginalName   string `gorm:"size:255;not null"`
	StoredName     string `gorm:"size:255;not null"`
	MimeType       string `gorm:"size:120;not null"`
	SizeBytes      int64  `gorm:"not null"`
	StoragePath    string `gorm:"size:500;not null"`
	CreatedAt      time.Time
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

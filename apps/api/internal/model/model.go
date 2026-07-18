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

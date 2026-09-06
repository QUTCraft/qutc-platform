package model

import "time"

// PersonalAIConfiguration belongs only to its user; APIKeyEncrypted is never serialized.
type PersonalAIConfiguration struct {
	UserID          string `gorm:"type:char(36);primaryKey" json:"-"`
	BaseURL         string `gorm:"size:500;not null"`
	Model           string `gorm:"size:120;not null"`
	APIKeyEncrypted string `gorm:"type:text;not null" json:"-"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TableName is explicit because the acronym in the Go type must not change the
// migration's stable snake_case table name across GORM naming strategies.
func (PersonalAIConfiguration) TableName() string { return "personal_ai_configurations" }

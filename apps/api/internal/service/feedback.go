package service

import (
	"errors"
	"net/mail"
	"strings"
)

const (
	FeedbackKindBug        = "bug"
	FeedbackKindFeature    = "feature"
	FeedbackStatusOpen     = "open"
	FeedbackStatusResolved = "resolved"
)

var (
	ErrFeedbackKindInvalid         = errors.New("feedback kind is invalid")
	ErrFeedbackTitleRequired       = errors.New("feedback title is required")
	ErrFeedbackTitleTooLong        = errors.New("feedback title is too long")
	ErrFeedbackDescriptionRequired = errors.New("feedback description is required")
	ErrFeedbackDescriptionTooLong  = errors.New("feedback description is too long")
	ErrFeedbackNameRequired        = errors.New("feedback contact name is required")
	ErrFeedbackNameTooLong         = errors.New("feedback contact name is too long")
	ErrFeedbackEmailInvalid        = errors.New("feedback contact email is invalid")
	ErrFeedbackPageURLTooLong      = errors.New("feedback page url is too long")
	ErrFeedbackPageURLInvalid      = errors.New("feedback page url is invalid")
)

type FeedbackInput struct {
	Kind         string `json:"kind"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ContactName  string `json:"contact_name"`
	ContactEmail string `json:"contact_email"`
	PageURL      string `json:"page_url"`
}

func NormalizeFeedbackInput(input FeedbackInput) (FeedbackInput, error) {
	input.Kind = strings.TrimSpace(input.Kind)
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.ContactName = strings.TrimSpace(input.ContactName)
	input.ContactEmail = strings.ToLower(strings.TrimSpace(input.ContactEmail))
	input.PageURL = strings.TrimSpace(input.PageURL)

	switch {
	case input.Kind != FeedbackKindBug && input.Kind != FeedbackKindFeature:
		return FeedbackInput{}, ErrFeedbackKindInvalid
	case input.Title == "":
		return FeedbackInput{}, ErrFeedbackTitleRequired
	case len([]rune(input.Title)) > 120:
		return FeedbackInput{}, ErrFeedbackTitleTooLong
	case input.Description == "":
		return FeedbackInput{}, ErrFeedbackDescriptionRequired
	case len([]rune(input.Description)) > 2000:
		return FeedbackInput{}, ErrFeedbackDescriptionTooLong
	case input.ContactName == "":
		return FeedbackInput{}, ErrFeedbackNameRequired
	case len([]rune(input.ContactName)) > 80:
		return FeedbackInput{}, ErrFeedbackNameTooLong
	case !validFeedbackEmail(input.ContactEmail):
		return FeedbackInput{}, ErrFeedbackEmailInvalid
	case len([]rune(input.PageURL)) > 500:
		return FeedbackInput{}, ErrFeedbackPageURLTooLong
	case input.PageURL != "" && !validFeedbackPageURL(input.PageURL):
		return FeedbackInput{}, ErrFeedbackPageURLInvalid
	default:
		return input, nil
	}
}

func FeedbackKindLabel(kind string) string {
	if kind == FeedbackKindFeature {
		return "功能建议"
	}
	return "网页缺陷"
}

func IsFeedbackStatus(value string) bool {
	return value == FeedbackStatusOpen || value == FeedbackStatusResolved
}

func validFeedbackEmail(value string) bool {
	parsed, err := mail.ParseAddress(value)
	return err == nil && parsed.Address != "" && len(value) <= 254
}

func validFeedbackPageURL(value string) bool {
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return false
	}
	for _, char := range value {
		if char < 32 || char == 127 {
			return false
		}
	}
	return true
}

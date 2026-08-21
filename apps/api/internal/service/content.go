package service

import (
	"errors"
	"strings"
)

const (
	ContentTypeNews      = "news"
	ContentTypeResource  = "resource"
	ContentTypeKnowledge = "knowledge"

	ContentStatusDraft     = "draft"
	ContentStatusReview    = "review"
	ContentStatusPublished = "published"
	ContentStatusArchived  = "archived"
)

var (
	ErrContentTitleRequired = errors.New("content title is required")
	ErrContentTitleTooLong  = errors.New("content title is too long")
	ErrContentTypeInvalid   = errors.New("content type is invalid")
	ErrContentCategoryLong  = errors.New("content category is too long")
	ErrContentExcerptLong   = errors.New("content excerpt is too long")
	ErrContentDirectoryLong = errors.New("content knowledge directory id is too long")
)

// ContentInput is the shared write DTO for creating and editing CMS content.
// It intentionally contains no author, organization, status, or publication
// fields; those values are controlled by the authenticated server workflow.
type ContentInput struct {
	Title                string `json:"title"`
	Type                 string `json:"type"`
	Category             string `json:"category"`
	KnowledgeDirectoryID string `json:"knowledge_directory_id"`
	Excerpt              string `json:"excerpt"`
	Body                 string `json:"body"`
}

// NormalizeContentInput applies the same whitespace and length rules to both
// create and update requests before the handler writes anything to the DB.
func NormalizeContentInput(input ContentInput) (ContentInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Type = strings.TrimSpace(input.Type)
	input.Category = strings.TrimSpace(input.Category)
	input.KnowledgeDirectoryID = strings.TrimSpace(input.KnowledgeDirectoryID)

	switch {
	case input.Title == "":
		return ContentInput{}, ErrContentTitleRequired
	case len([]rune(input.Title)) > 160:
		return ContentInput{}, ErrContentTitleTooLong
	case !IsContentType(input.Type):
		return ContentInput{}, ErrContentTypeInvalid
	case len([]rune(input.Category)) > 64:
		return ContentInput{}, ErrContentCategoryLong
	case len([]rune(input.Excerpt)) > 500:
		return ContentInput{}, ErrContentExcerptLong
	case len([]rune(input.KnowledgeDirectoryID)) > 64:
		return ContentInput{}, ErrContentDirectoryLong
	default:
		return input, nil
	}
}

func IsContentType(value string) bool {
	switch value {
	case ContentTypeNews, ContentTypeResource, ContentTypeKnowledge:
		return true
	default:
		return false
	}
}

func IsContentStatus(value string) bool {
	switch value {
	case ContentStatusDraft, ContentStatusReview, ContentStatusPublished, ContentStatusArchived:
		return true
	default:
		return false
	}
}

// CanTransitionContentStatus is the only allowed publication state machine.
// Authors submit draft or archived content to review; reviewers can publish or
// return it to draft, and published content must be archived before editing.
func CanTransitionContentStatus(current, target string) bool {
	if !IsContentStatus(current) || !IsContentStatus(target) || current == target {
		return false
	}
	switch target {
	case ContentStatusDraft:
		return current == ContentStatusReview
	case ContentStatusReview:
		return current == ContentStatusDraft || current == ContentStatusArchived
	case ContentStatusPublished:
		return current == ContentStatusDraft || current == ContentStatusReview || current == ContentStatusArchived
	case ContentStatusArchived:
		return current == ContentStatusPublished
	default:
		return false
	}
}

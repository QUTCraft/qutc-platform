package service

import (
	"errors"
	"strings"
	"testing"
)

func validContentInput() ContentInput {
	return ContentInput{
		Title:    "  QUTCraft CMS 项目进展  ",
		Type:     ContentTypeNews,
		Category: "  社团动态  ",
		Excerpt:  "项目阶段性进展。",
		Body:     "正文内容。",
	}
}

func TestNormalizeContentInput(t *testing.T) {
	input, err := NormalizeContentInput(validContentInput())
	if err != nil {
		t.Fatalf("NormalizeContentInput() error = %v", err)
	}
	if input.Title != "QUTCraft CMS 项目进展" || input.Category != "社团动态" {
		t.Fatalf("expected title and category to be normalized, got %#v", input)
	}
}

func TestNormalizeContentInputTrimsKnowledgeDirectoryID(t *testing.T) {
	input, err := NormalizeContentInput(ContentInput{Title: " article ", Type: ContentTypeKnowledge, KnowledgeDirectoryID: " directory-1 "})
	if err != nil {
		t.Fatalf("NormalizeContentInput() error = %v", err)
	}
	if input.KnowledgeDirectoryID != "directory-1" {
		t.Fatalf("knowledge directory id = %q, want directory-1", input.KnowledgeDirectoryID)
	}
}

func TestNormalizeContentInputRejectsContractViolations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ContentInput)
		want   error
	}{
		{name: "missing title", mutate: func(input *ContentInput) { input.Title = "  " }, want: ErrContentTitleRequired},
		{name: "long title", mutate: func(input *ContentInput) { input.Title = strings.Repeat("字", 161) }, want: ErrContentTitleTooLong},
		{name: "invalid type", mutate: func(input *ContentInput) { input.Type = "admin" }, want: ErrContentTypeInvalid},
		{name: "long category", mutate: func(input *ContentInput) { input.Category = strings.Repeat("字", 65) }, want: ErrContentCategoryLong},
		{name: "long excerpt", mutate: func(input *ContentInput) { input.Excerpt = strings.Repeat("字", 501) }, want: ErrContentExcerptLong},
		{name: "long knowledge directory id", mutate: func(input *ContentInput) { input.KnowledgeDirectoryID = strings.Repeat("d", 65) }, want: ErrContentDirectoryLong},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validContentInput()
			test.mutate(&input)
			_, err := NormalizeContentInput(input)
			if !errors.Is(err, test.want) {
				t.Fatalf("NormalizeContentInput() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestCanTransitionContentStatus(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
		want    bool
	}{
		{name: "draft publishes", current: ContentStatusDraft, target: ContentStatusPublished, want: true},
		{name: "review publishes", current: ContentStatusReview, target: ContentStatusPublished, want: true},
		{name: "published archives", current: ContentStatusPublished, target: ContentStatusArchived, want: true},
		{name: "archived republishes", current: ContentStatusArchived, target: ContentStatusPublished, want: true},
		{name: "draft cannot archive", current: ContentStatusDraft, target: ContentStatusArchived, want: false},
		{name: "published cannot publish twice", current: ContentStatusPublished, target: ContentStatusPublished, want: false},
		{name: "unknown current rejected", current: "deleted", target: ContentStatusPublished, want: false},
		{name: "review is not directly writable", current: ContentStatusDraft, target: ContentStatusReview, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CanTransitionContentStatus(test.current, test.target); got != test.want {
				t.Fatalf("CanTransitionContentStatus(%q, %q) = %v, want %v", test.current, test.target, got, test.want)
			}
		})
	}
}

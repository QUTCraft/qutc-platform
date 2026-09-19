package service

import "testing"

func TestNormalizeFeedbackInput(t *testing.T) {
	input, err := NormalizeFeedbackInput(FeedbackInput{
		Kind: " bug ", Title: " 按钮错位 ", Description: "门户顶栏在窄屏重叠。",
		ContactName: " 林沐 ", ContactEmail: " LinMu@Example.com ", PageURL: " /posts ",
	})
	if err != nil {
		t.Fatalf("NormalizeFeedbackInput() error = %v", err)
	}
	if input.Kind != FeedbackKindBug || input.Title != "按钮错位" || input.ContactEmail != "linmu@example.com" || input.PageURL != "/posts" {
		t.Fatalf("normalized = %+v", input)
	}
}

func TestIsFeedbackStatus(t *testing.T) {
	if !IsFeedbackStatus(FeedbackStatusOpen) || !IsFeedbackStatus(FeedbackStatusResolved) {
		t.Fatal("open and resolved should be accepted")
	}
	if IsFeedbackStatus("pending") || IsFeedbackStatus("") {
		t.Fatal("unknown statuses should be rejected")
	}
}

func TestNormalizeFeedbackInputRejectsUnsafePageURL(t *testing.T) {
	_, err := NormalizeFeedbackInput(FeedbackInput{
		Kind: FeedbackKindFeature, Title: "夜间模式", Description: "希望增加夜间模式。",
		ContactName: "Nova", ContactEmail: "nova@example.com", PageURL: "javascript:alert(1)",
	})
	if err != ErrFeedbackPageURLInvalid {
		t.Fatalf("error = %v, want page url invalid", err)
	}
}

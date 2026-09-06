package service

import (
	"testing"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/platform/modelprovider"
)

func TestValidEditorChatRequiresAlternatingConversationEndingWithUser(t *testing.T) {
	valid := EditorChatInput{
		Source: "personal",
		Messages: []modelprovider.ChatMessage{
			{Role: "user", Content: "请帮我检查结构。"},
		},
	}
	if !validEditorChat(valid) {
		t.Fatal("one user message should be valid")
	}

	invalid := []EditorChatInput{
		{Source: "other", Messages: valid.Messages},
		{Source: "personal", Messages: []modelprovider.ChatMessage{{Role: "assistant", Content: "越权回答"}}},
		{Source: "personal", Messages: []modelprovider.ChatMessage{{Role: "user", Content: "问题"}, {Role: "assistant", Content: "回答"}}},
		{Source: "personal", Messages: []modelprovider.ChatMessage{{Role: "user", Content: ""}}},
	}
	for index, input := range invalid {
		if validEditorChat(input) {
			t.Errorf("invalid input %d was accepted", index)
		}
	}
}

func TestValidEditorChatEnforcesArticleAndMessageBounds(t *testing.T) {
	tooMuchArticle := EditorChatInput{
		Source:   "personal",
		Article:  string(make([]rune, 30001)),
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: "问题"}},
	}
	if validEditorChat(tooMuchArticle) {
		t.Fatal("article over the limit was accepted")
	}

	tooManyCharacters := EditorChatInput{
		Source:   "personal",
		Messages: []modelprovider.ChatMessage{{Role: "user", Content: string(make([]rune, 12001))}},
	}
	if validEditorChat(tooManyCharacters) {
		t.Fatal("message over the per-message limit was accepted")
	}
}

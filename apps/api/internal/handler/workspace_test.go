package handler

import (
	"bytes"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
)

func TestDetectAssetTypeUsesFileSignature(t *testing.T) {
	png, err := detectAssetType(bytes.NewReader([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}))
	if err != nil || png != "image/png" {
		t.Fatalf("detectAssetType(png) = %q, %v", png, err)
	}

	spoofed, err := detectAssetType(bytes.NewReader([]byte("not an image")))
	if err != nil || allowedAssetType(spoofed) {
		t.Fatalf("expected spoofed content to be rejected, got %q, %v", spoofed, err)
	}
}

func TestAssetResponseUsesNullForUnlinkedAsset(t *testing.T) {
	response := assetResponse(model.MediaAsset{ID: "asset-1", OriginalName: "guide.pdf", MimeType: "application/pdf", SizeBytes: 12})
	if response["content_id"] != nil {
		t.Fatalf("unlinked asset content_id = %#v, want nil", response["content_id"])
	}
}

func validApplicationFixture() applicationRequest {
	return applicationRequest{
		Type:      "whitelist",
		ClassName: "计算机231",
		Name:      "Yukino",
		GameID:    "YukinoCraft",
		QQNumber:  "123456789",
		Email:     "yukino@example.com",
		Note:      "希望参与周末建筑测试。",
	}
}

func TestValidApplicationRequest(t *testing.T) {
	if !validApplicationRequest(validApplicationFixture()) {
		t.Fatal("expected fixture to be valid")
	}

	tests := []struct {
		name   string
		mutate func(*applicationRequest)
	}{
		{name: "invalid type", mutate: func(value *applicationRequest) { value.Type = "admin" }},
		{name: "invalid email", mutate: func(value *applicationRequest) { value.Email = "not-an-email" }},
		{name: "invalid qq number", mutate: func(value *applicationRequest) { value.QQNumber = "1234" }},
		{name: "missing name", mutate: func(value *applicationRequest) { value.Name = "  " }},
		{name: "long note", mutate: func(value *applicationRequest) { value.Note = string(make([]rune, 501)) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validApplicationFixture()
			test.mutate(&value)
			if validApplicationRequest(value) {
				t.Fatalf("expected %s to be rejected", test.name)
			}
		})
	}
}

func TestValidApplicationRequestDefaultsToWhitelist(t *testing.T) {
	value := validApplicationFixture()
	value.Type = ""
	if !validApplicationRequest(value) {
		t.Fatal("expected an omitted application type to default to whitelist")
	}
}

func TestContentStatusTransitions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		target  string
		valid   bool
	}{
		{name: "draft publishes", current: "draft", target: "published", valid: true},
		{name: "review publishes", current: "review", target: "published", valid: true},
		{name: "published archives", current: "published", target: "archived", valid: true},
		{name: "archived republishes", current: "archived", target: "published", valid: true},
		{name: "draft cannot archive", current: "draft", target: "archived", valid: false},
		{name: "unknown target rejected", current: "draft", target: "review", valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := canTransitionContentStatus(test.current, test.target); got != test.valid {
				t.Fatalf("canTransitionContentStatus(%q, %q) = %v, want %v", test.current, test.target, got, test.valid)
			}
		})
	}
}

func TestKnowledgeDirectoryRequestRequiresSafeSlug(t *testing.T) {
	valid := knowledgeDirectoryRequest{Name: "技术规范", Slug: "technology", Description: "接口规范", ParentID: "", SortOrder: 10, IsPublic: true}
	if !validKnowledgeDirectoryRequest(valid) {
		t.Fatal("expected a lowercase hyphenated slug to be valid")
	}
	for _, slug := range []string{"Technology", "tech_spec", "技术规范", "-technology", "technology-"} {
		value := valid
		value.Slug = slug
		if validKnowledgeDirectoryRequest(value) {
			t.Fatalf("slug %q should be rejected", slug)
		}
	}
}

func TestContentPublicItemsExcludeInternalFields(t *testing.T) {
	publishedAt := time.Date(2026, time.July, 27, 0, 0, 0, 0, time.UTC)
	content := model.Content{
		ID:             "content-public-1",
		OrganizationID: "org-private",
		AuthorUserID:   "author-private",
		Title:          "公开动态",
		Type:           "news",
		Category:       "公告",
		Status:         "published",
		Excerpt:        "公开摘要",
		Body:           "公开正文",
		PublishedAt:    &publishedAt,
	}

	listItem := contentPublicItem(content)
	detailItem := (&WorkspaceHandler{}).contentPublicDetailItem("qutcraft", content)
	for name, item := range map[string]map[string]interface{}{"list": listItem, "detail": detailItem} {
		for _, privateField := range []string{"organization_id", "author_user_id", "status"} {
			if _, exists := item[privateField]; exists {
				t.Fatalf("%s response leaked private field %q", name, privateField)
			}
		}
	}
	if _, exists := listItem["body"]; exists {
		t.Fatal("list response must not include full body")
	}
	if detailItem["body"] != "公开正文" {
		t.Fatal("detail response should include the published body")
	}
}

func TestContentPublicDetailRewritesAdminAssetURLs(t *testing.T) {
	content := model.Content{ID: "content-public-markdown", Type: "news", Body: "![封面](/api/v1/admin/assets/asset-1/download)"}
	item := (&WorkspaceHandler{}).contentPublicDetailItem("qutcraft", content)
	want := "![封面](/api/v1/portal/organizations/qutcraft/assets/asset-1/download)"
	if item["body"] != want {
		t.Fatalf("public body = %v, want %q", item["body"], want)
	}
}

func TestMembershipRoleProtection(t *testing.T) {
	tests := []struct {
		name        string
		actorRole   string
		actorIsSelf bool
		currentRole string
		nextRole    string
		nextState   string
		wantCode    string
	}{
		{name: "owner cannot be disabled", actorRole: "owner", currentRole: "owner", nextRole: "owner", nextState: "disabled", wantCode: "membership.owner_protected"},
		{name: "owner cannot be demoted", actorRole: "owner", currentRole: "owner", nextRole: "administrator", nextState: "active", wantCode: "membership.owner_protected"},
		{name: "administrator cannot grant owner", actorRole: "administrator", currentRole: "member", nextRole: "owner", nextState: "active", wantCode: "membership.owner_only"},
		{name: "member cannot change self role", actorRole: "member", actorIsSelf: true, currentRole: "member", nextRole: "editor", nextState: "active", wantCode: "membership.self_change_forbidden"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := membershipChangeError(test.actorRole, test.actorIsSelf, test.currentRole, test.nextRole, test.nextState); got != test.wantCode {
				t.Fatalf("membershipChangeError() = %q, want %q", got, test.wantCode)
			}
		})
	}
}

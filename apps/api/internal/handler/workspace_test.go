package handler

import "testing"

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

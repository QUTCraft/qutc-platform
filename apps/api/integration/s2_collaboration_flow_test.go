//go:build integration

package integration_test

import (
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type invitationDTO struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	InviteURL      string `json:"invite_url"`
	Delivery       struct {
		Status   string `json:"status"`
		Adapter  string `json:"adapter"`
		Attempts int    `json:"attempts"`
	} `json:"delivery"`
}

type tokenPairDTO struct {
	AccessToken string `json:"access_token"`
	User        struct {
		ID    string   `json:"id"`
		Email string   `json:"email"`
		Roles []string `json:"roles"`
	} `json:"user"`
}

type projectMemberDTO struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type projectMilestoneDTO struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	DueAt       *time.Time `json:"due_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

func TestS2InvitationAndProjectCollaboration(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	client := &http.Client{Timeout: 10 * time.Second, Jar: jar}
	db := openIntegrationDB(t, cfg.mysqlDSN)
	ownerToken := loginAsOwner(t, client, cfg)

	var organization model.Organization
	if err := db.Where("slug = ?", cfg.organizationSlug).First(&organization).Error; err != nil {
		t.Fatalf("load organization: %v", err)
	}
	var project model.Project
	if err := db.Where("organization_id = ?", organization.ID).Order("created_at ASC").First(&project).Error; err != nil {
		t.Fatalf("load collaboration project: %v", err)
	}

	email := "s2-" + uuid.NewString() + "@integration.invalid"
	password := "S2-integration-password!"
	var invitationID, userID, membershipID, milestoneID string
	t.Cleanup(func() {
		cleanupS2Fixture(t, db, invitationID, userID, membershipID, project.ID, milestoneID)
	})

	createBody := request(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/invitations", ownerToken, map[string]any{
		"email":            email,
		"role":             "editor",
		"expires_in_hours": 24,
	}, http.StatusCreated)
	if strings.Contains(string(createBody), "token_hash") {
		t.Fatal("invitation response exposed token_hash")
	}
	var createdEnvelope apiEnvelope[invitationDTO]
	decodeJSON(t, createBody, &createdEnvelope)
	invitation := createdEnvelope.Data
	invitationID = invitation.ID
	if invitation.Status != "pending" || invitation.Role != "editor" || invitation.InviteURL == "" {
		t.Fatalf("created invitation = %+v, want pending editor invitation with URL", invitation)
	}
	if invitation.Delivery.Status != "disabled" || invitation.Delivery.Adapter != "disabled" || invitation.Delivery.Attempts != 0 {
		t.Fatalf("disabled email delivery = %+v, want explicit disabled status without attempts", invitation.Delivery)
	}
	rawToken := strings.TrimPrefix(invitation.InviteURL, "/invite/")
	if rawToken == "" || rawToken == invitation.InviteURL {
		t.Fatalf("invite_url = %q, want /invite/<token>", invitation.InviteURL)
	}

	var storedInvitation model.Invitation
	if err := db.First(&storedInvitation, "id = ?", invitationID).Error; err != nil {
		t.Fatalf("load stored invitation: %v", err)
	}
	if storedInvitation.TokenHash == "" || storedInvitation.TokenHash == rawToken || strings.Contains(string(createBody), storedInvitation.TokenHash) {
		t.Fatal("invitation token was not safely separated from its stored hash")
	}
	var storedDelivery model.InvitationDelivery
	if err := db.Where("invitation_id = ?", invitationID).First(&storedDelivery).Error; err != nil {
		t.Fatalf("load invitation delivery: %v", err)
	}
	if storedDelivery.Status != "disabled" || storedDelivery.Attempts != 0 {
		t.Fatalf("stored invitation delivery = %+v, want disabled without attempts", storedDelivery)
	}

	previewBody := request(t, client, http.MethodGet, cfg.apiURL+"/api/v1/invitations/"+rawToken, "", nil, http.StatusOK)
	if strings.Contains(string(previewBody), "token_hash") || strings.Contains(string(previewBody), rawToken) {
		t.Fatal("public invitation preview exposed token material")
	}
	var previewEnvelope apiEnvelope[invitationDTO]
	decodeJSON(t, previewBody, &previewEnvelope)
	if previewEnvelope.Data.Email != email || previewEnvelope.Data.OrganizationID != organization.ID {
		t.Fatalf("invitation preview = %+v, want invited account and organization", previewEnvelope.Data)
	}

	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/invitations", ownerToken, map[string]any{
		"email": email,
		"role":  "editor",
	}, http.StatusConflict)
	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/auth/register", "", map[string]any{
		"email":            "wrong-" + email,
		"display_name":     "S2 Wrong Email",
		"password":         password,
		"invitation_token": rawToken,
	}, http.StatusBadRequest)

	registerBody := request(t, client, http.MethodPost, cfg.apiURL+"/api/v1/auth/register", "", map[string]any{
		"email":            email,
		"display_name":     "S2 Integration Member",
		"password":         password,
		"invitation_token": rawToken,
	}, http.StatusCreated)
	var registerEnvelope apiEnvelope[tokenPairDTO]
	decodeJSON(t, registerBody, &registerEnvelope)
	editor := registerEnvelope.Data
	userID = editor.User.ID
	if userID == "" || editor.AccessToken == "" || editor.User.Email != email || !containsString(editor.User.Roles, "editor") {
		t.Fatalf("registered account = %+v, want authenticated editor", editor.User)
	}
	if err := db.Where("organization_id = ? AND user_id = ?", organization.ID, userID).First(&model.Membership{}).Error; err != nil {
		t.Fatalf("load accepted membership: %v", err)
	}
	var membership model.Membership
	if err := db.Where("organization_id = ? AND user_id = ?", organization.ID, userID).First(&membership).Error; err != nil {
		t.Fatalf("load accepted membership id: %v", err)
	}
	membershipID = membership.ID

	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/invitations/"+rawToken, "", nil, http.StatusGone)
	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/auth/register", "", map[string]any{
		"email":            "reuse-" + email,
		"display_name":     "S2 Reused Token",
		"password":         password,
		"invitation_token": rawToken,
	}, http.StatusConflict)
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/content", editor.AccessToken, nil, http.StatusOK)
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/users", editor.AccessToken, nil, http.StatusForbidden)
	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/admin/projects/"+project.ID+"/members", editor.AccessToken, map[string]any{
		"user_id": userID,
		"role":    "contributor",
	}, http.StatusForbidden)

	var owner model.User
	if err := db.Where("email = ?", cfg.adminEmail).First(&owner).Error; err != nil {
		t.Fatalf("load owner user: %v", err)
	}
	requireStatus(t, client, http.MethodPatch, cfg.apiURL+"/api/v1/admin/users/"+owner.ID, ownerToken, map[string]any{
		"role":  "owner",
		"state": "disabled",
	}, http.StatusConflict)

	userAdminURL := cfg.apiURL + "/api/v1/admin/users/" + userID
	requireStatus(t, client, http.MethodPatch, userAdminURL, ownerToken, map[string]any{
		"role":  "member",
		"state": "active",
	}, http.StatusOK)
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/content", editor.AccessToken, nil, http.StatusForbidden)
	requireStatus(t, client, http.MethodPatch, userAdminURL, ownerToken, map[string]any{
		"role":  "editor",
		"state": "disabled",
	}, http.StatusOK)
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/membership/history", editor.AccessToken, nil, http.StatusUnauthorized)
	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/auth/refresh", "", map[string]any{}, http.StatusUnauthorized)
	requireStatus(t, client, http.MethodPost, cfg.apiURL+"/api/v1/auth/login", "", map[string]any{
		"email": email, "password": password,
	}, http.StatusUnauthorized)

	var disabledUser model.User
	if err := db.First(&disabledUser, "id = ?", userID).Error; err != nil {
		t.Fatalf("load disabled member account: %v", err)
	}
	if disabledUser.State != "active" {
		t.Fatalf("organization disable changed global user state to %q", disabledUser.State)
	}
	if err := db.First(&membership, "id = ?", membershipID).Error; err != nil {
		t.Fatalf("reload disabled membership: %v", err)
	}
	if membership.State != "disabled" {
		t.Fatalf("membership state = %q, want disabled", membership.State)
	}
	var activeRefreshCount int64
	if err := db.Model(&model.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", userID).Count(&activeRefreshCount).Error; err != nil {
		t.Fatalf("count active refresh tokens: %v", err)
	}
	if activeRefreshCount != 0 {
		t.Fatalf("active refresh tokens after disable = %d, want 0", activeRefreshCount)
	}

	requireStatus(t, client, http.MethodPatch, userAdminURL, ownerToken, map[string]any{
		"role":  "editor",
		"state": "active",
	}, http.StatusOK)
	reloginBody := request(t, client, http.MethodPost, cfg.apiURL+"/api/v1/auth/login", "", map[string]any{
		"email": email, "password": password,
	}, http.StatusOK)
	decodeJSON(t, reloginBody, &registerEnvelope)
	editor = registerEnvelope.Data
	if editor.AccessToken == "" || !containsString(editor.User.Roles, "editor") {
		t.Fatalf("reactivated login = %+v, want editor access", editor.User)
	}
	requireStatus(t, client, http.MethodGet, cfg.apiURL+"/api/v1/admin/content", editor.AccessToken, nil, http.StatusOK)
	for _, reason := range []string{"admin_role_changed", "admin_disabled", "admin_reactivated"} {
		var count int64
		if err := db.Model(&model.MembershipEvent{}).Where("membership_id = ? AND reason = ?", membershipID, reason).Count(&count).Error; err != nil {
			t.Fatalf("count membership event %s: %v", reason, err)
		}
		if count != 1 {
			t.Fatalf("membership event %s count = %d, want 1", reason, count)
		}
	}

	memberURL := cfg.apiURL + "/api/v1/admin/projects/" + project.ID + "/members"
	memberBody := request(t, client, http.MethodPost, memberURL, ownerToken, map[string]any{
		"user_id": userID,
		"role":    "contributor",
	}, http.StatusCreated)
	var memberEnvelope apiEnvelope[projectMemberDTO]
	decodeJSON(t, memberBody, &memberEnvelope)
	if memberEnvelope.Data.UserID != userID || memberEnvelope.Data.Role != "contributor" {
		t.Fatalf("created project member = %+v", memberEnvelope.Data)
	}

	memberBody = request(t, client, http.MethodPost, memberURL, ownerToken, map[string]any{
		"user_id": userID,
		"role":    "member",
	}, http.StatusOK)
	decodeJSON(t, memberBody, &memberEnvelope)
	if memberEnvelope.Data.Role != "member" {
		t.Fatalf("idempotent member assignment role = %q, want member", memberEnvelope.Data.Role)
	}

	memberItemURL := memberURL + "/" + userID
	memberBody = request(t, client, http.MethodPatch, memberItemURL, ownerToken, map[string]any{"role": "lead"}, http.StatusOK)
	decodeJSON(t, memberBody, &memberEnvelope)
	if memberEnvelope.Data.Role != "lead" {
		t.Fatalf("updated project member role = %q, want lead", memberEnvelope.Data.Role)
	}
	requireStatus(t, client, http.MethodPatch, memberURL+"/"+project.OwnerUserID, ownerToken, map[string]any{"role": "member"}, http.StatusConflict)
	requireStatus(t, client, http.MethodDelete, memberURL+"/"+project.OwnerUserID, ownerToken, nil, http.StatusConflict)

	milestonesURL := cfg.apiURL + "/api/v1/admin/projects/" + project.ID + "/milestones"
	requireStatus(t, client, http.MethodPost, milestonesURL, ownerToken, map[string]any{
		"title":  "S2 invalid milestone",
		"status": "planned",
		"due_at": "not-rfc3339",
	}, http.StatusBadRequest)

	dueAt := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Second)
	milestoneBody := request(t, client, http.MethodPost, milestonesURL, ownerToken, map[string]any{
		"title":  "S2 collaboration gate " + uuid.NewString(),
		"status": "planned",
		"due_at": dueAt.Format(time.RFC3339),
	}, http.StatusCreated)
	var milestoneEnvelope apiEnvelope[projectMilestoneDTO]
	decodeJSON(t, milestoneBody, &milestoneEnvelope)
	milestone := milestoneEnvelope.Data
	milestoneID = milestone.ID
	if milestone.ID == "" || milestone.Status != "planned" || milestone.CompletedAt != nil {
		t.Fatalf("created milestone = %+v", milestone)
	}

	milestoneURL := milestonesURL + "/" + milestone.ID
	milestoneBody = request(t, client, http.MethodPatch, milestoneURL, ownerToken, map[string]any{
		"title":  milestone.Title,
		"status": "completed",
		"due_at": dueAt.Format(time.RFC3339),
	}, http.StatusOK)
	decodeJSON(t, milestoneBody, &milestoneEnvelope)
	if milestoneEnvelope.Data.Status != "completed" || milestoneEnvelope.Data.CompletedAt == nil {
		t.Fatalf("completed milestone = %+v, want completed_at", milestoneEnvelope.Data)
	}

	var listEnvelope apiEnvelope[[]projectMilestoneDTO]
	decodeJSON(t, request(t, client, http.MethodGet, milestonesURL, ownerToken, nil, http.StatusOK), &listEnvelope)
	if !containsMilestone(listEnvelope.Data, milestone.ID, "completed") {
		t.Fatalf("completed milestone %s missing from project list", milestone.ID)
	}

	requireStatus(t, client, http.MethodDelete, milestoneURL, ownerToken, nil, http.StatusOK)
	milestoneID = ""
	requireStatus(t, client, http.MethodDelete, memberItemURL, ownerToken, nil, http.StatusOK)
	var membersEnvelope apiEnvelope[[]projectMemberDTO]
	decodeJSON(t, request(t, client, http.MethodGet, memberURL, ownerToken, nil, http.StatusOK), &membersEnvelope)
	if containsProjectMember(membersEnvelope.Data, userID) {
		t.Fatalf("removed project member %s remained in project list", userID)
	}

	cleanupS2Fixture(t, db, invitationID, userID, membershipID, project.ID, milestoneID)
	invitationID, userID, membershipID, milestoneID = "", "", "", ""
	requireS2FixtureAbsent(t, db, email)
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func containsMilestone(values []projectMilestoneDTO, id, status string) bool {
	for _, value := range values {
		if value.ID == id && value.Status == status {
			return true
		}
	}
	return false
}

func containsProjectMember(values []projectMemberDTO, userID string) bool {
	for _, value := range values {
		if value.UserID == userID {
			return true
		}
	}
	return false
}

func cleanupS2Fixture(t *testing.T, db *gorm.DB, invitationID, userID, membershipID, projectID, milestoneID string) {
	t.Helper()
	cleanup := func(description string, result *gorm.DB) {
		if result.Error != nil {
			t.Errorf("cleanup %s: %v", description, result.Error)
		}
	}
	if milestoneID != "" {
		cleanup("project milestone", db.Where("id = ?", milestoneID).Delete(&model.ProjectMilestone{}))
	}
	if userID != "" {
		cleanup("project member", db.Where("project_id = ? AND user_id = ?", projectID, userID).Delete(&model.ProjectMember{}))
		cleanup("refresh tokens", db.Where("user_id = ?", userID).Delete(&model.RefreshToken{}))
		cleanup("user audit events", db.Where("actor_user_id = ?", userID).Delete(&model.AuditEvent{}))
	}
	if membershipID != "" {
		cleanup("membership audit events", db.Where("target_type = ? AND target_id = ?", "membership", membershipID).Delete(&model.AuditEvent{}))
		cleanup("membership roles", db.Where("membership_id = ?", membershipID).Delete(&model.MembershipRole{}))
		cleanup("membership events", db.Where("membership_id = ?", membershipID).Delete(&model.MembershipEvent{}))
		cleanup("membership", db.Where("id = ?", membershipID).Delete(&model.Membership{}))
	}
	if invitationID != "" {
		cleanup("invitation audit events", db.Where("target_type = ? AND target_id = ?", "invitation", invitationID).Delete(&model.AuditEvent{}))
		cleanup("invitation delivery", db.Where("invitation_id = ?", invitationID).Delete(&model.InvitationDelivery{}))
		cleanup("invitation", db.Where("id = ?", invitationID).Delete(&model.Invitation{}))
	}
	if userID != "" {
		cleanup("user", db.Where("id = ?", userID).Delete(&model.User{}))
	}
}

func requireS2FixtureAbsent(t *testing.T, db *gorm.DB, email string) {
	t.Helper()
	for description, query := range map[string]*gorm.DB{
		"user":       db.Model(&model.User{}).Where("email = ?", email),
		"invitation": db.Model(&model.Invitation{}).Where("email = ?", email),
		"delivery":   db.Model(&model.InvitationDelivery{}).Where("invitation_id IN (?)", db.Model(&model.Invitation{}).Select("id").Where("email = ?", email)),
		"milestone":  db.Model(&model.ProjectMilestone{}).Where("title LIKE ?", "S2 collaboration gate %"),
	} {
		var count int64
		if err := query.Count(&count).Error; err != nil {
			t.Fatalf("count cleanup %s fixtures: %v", description, err)
		}
		if count != 0 {
			t.Fatalf("%s cleanup fixtures remaining = %d, want 0", description, count)
		}
	}
}

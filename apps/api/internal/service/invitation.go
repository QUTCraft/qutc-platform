package service

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	DefaultInvitationExpiry = 7 * 24 * time.Hour
	MaxInvitationExpiry     = 30 * 24 * time.Hour
)

var (
	ErrInvitationNotFound      = errors.New("invitation not found")
	ErrInvitationExpired       = errors.New("invitation expired")
	ErrInvitationRevoked       = errors.New("invitation revoked")
	ErrInvitationAccepted      = errors.New("invitation already accepted")
	ErrInvitationPending       = errors.New("invitation already pending")
	ErrInvitationAlreadyMember = errors.New("user is already an active member")
	ErrInvitationMemberExists  = errors.New("user already has a managed membership")
	ErrInvitationEmailMismatch = errors.New("invitation email does not match account")
	ErrInvitationInvalidEmail  = errors.New("invitation email is invalid")
	ErrInvitationInvalidRole   = errors.New("invitation role is invalid")
	ErrInvitationInvalidExpiry = errors.New("invitation expiry is invalid")
	ErrInvitationInvalidStatus = errors.New("invitation status is invalid")
)

type InvitationView struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Organization   string    `json:"organization_name"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	Status         string    `json:"status"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type InvitationCreateResult struct {
	InvitationView
	Token string `json:"-"`
}

type InvitationAcceptance struct {
	InvitationView
	MembershipID string `json:"membership_id"`
}

func (s *AuthService) CreateInvitation(organizationID, invitedBy, email, role string, expiresIn time.Duration) (InvitationCreateResult, error) {
	email = normalizeInvitationEmail(email)
	if _, err := mail.ParseAddress(email); err != nil || email == "" {
		return InvitationCreateResult{}, ErrInvitationInvalidEmail
	}
	if !validInvitationRole(role) {
		return InvitationCreateResult{}, ErrInvitationInvalidRole
	}
	if expiresIn <= 0 {
		expiresIn = DefaultInvitationExpiry
	}
	if expiresIn > MaxInvitationExpiry {
		return InvitationCreateResult{}, ErrInvitationInvalidExpiry
	}

	token, err := randomToken()
	if err != nil {
		return InvitationCreateResult{}, err
	}

	now := time.Now().UTC()
	var result InvitationCreateResult
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var pending model.Invitation
		if err := tx.Where("organization_id = ? AND email = ? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > ?", organizationID, email, now).First(&pending).Error; err == nil {
			return ErrInvitationPending
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		invitationRole, err := roleByKey(tx, role)
		if err != nil {
			return err
		}

		var user model.User
		userErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("email = ?", email).First(&user).Error
		if userErr != nil && userErr != gorm.ErrRecordNotFound {
			return userErr
		}
		if userErr == gorm.ErrRecordNotFound {
			user = model.User{
				ID:                    uuid.NewString(),
				Email:                 email,
				DisplayName:           invitedDisplayName(email),
				PasswordHash:          "",
				State:                 "invited",
				DefaultOrganizationID: organizationID,
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		}

		var membership model.Membership
		membershipErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("organization_id = ? AND user_id = ?", organizationID, user.ID).
			First(&membership).Error
		switch {
		case membershipErr == nil && membership.State == "active":
			return ErrInvitationAlreadyMember
		case membershipErr == nil && membership.State != "invited":
			return ErrInvitationMemberExists
		case membershipErr != nil && membershipErr != gorm.ErrRecordNotFound:
			return membershipErr
		case membershipErr == gorm.ErrRecordNotFound:
			membership = model.Membership{ID: uuid.NewString(), OrganizationID: organizationID, UserID: user.ID, State: "invited"}
			if err := tx.Create(&membership).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.MembershipEvent{ID: uuid.NewString(), MembershipID: membership.ID, State: "invited", Reason: "invitation_created"}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("membership_id = ?", membership.ID).Delete(&model.MembershipRole{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.MembershipRole{MembershipID: membership.ID, RoleID: invitationRole.ID}).Error; err != nil {
			return err
		}

		invit := model.Invitation{
			ID:             uuid.NewString(),
			OrganizationID: organizationID,
			InvitedBy:      invitedBy,
			Email:          email,
			Role:           role,
			TokenHash:      tokenHash(token),
			ExpiresAt:      now.Add(expiresIn),
		}
		if err := tx.Create(&invit).Error; err != nil {
			return err
		}
		view, err := s.invitationViewWithDB(tx, invit)
		if err != nil {
			return err
		}
		result = InvitationCreateResult{InvitationView: view, Token: token}
		return nil
	})
	return result, err
}

func (s *AuthService) LookupInvitation(rawToken string) (InvitationView, error) {
	invit, err := s.findInvitation(s.db, rawToken, false)
	if err != nil {
		return InvitationView{}, err
	}
	return s.invitationView(invit)
}

func (s *AuthService) ListInvitations(organizationID, status string, page, pageSize int) ([]InvitationView, int64, error) {
	status = strings.TrimSpace(status)
	if status != "" && status != "pending" && status != "accepted" && status != "expired" && status != "revoked" {
		return nil, 0, ErrInvitationInvalidStatus
	}

	now := time.Now().UTC()
	query := s.db.Model(&model.Invitation{}).Where("organization_id = ?", organizationID)
	switch status {
	case "pending":
		query = query.Where("accepted_at IS NULL AND revoked_at IS NULL AND expires_at > ?", now)
	case "accepted":
		query = query.Where("accepted_at IS NOT NULL")
	case "expired":
		query = query.Where("accepted_at IS NULL AND revoked_at IS NULL AND expires_at <= ?", now)
	case "revoked":
		query = query.Where("accepted_at IS NULL AND revoked_at IS NOT NULL")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var invitations []model.Invitation
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&invitations).Error; err != nil {
		return nil, 0, err
	}
	var organization model.Organization
	if err := s.db.First(&organization, "id = ?", organizationID).Error; err != nil {
		return nil, 0, err
	}
	items := make([]InvitationView, 0, len(invitations))
	for _, invitation := range invitations {
		items = append(items, invitationViewForOrganization(invitation, organization.Name, now))
	}
	return items, total, nil
}

func (s *AuthService) RevokeInvitation(organizationID, invitationID string) (InvitationView, error) {
	var result InvitationView
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var invitation model.Invitation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND organization_id = ?", strings.TrimSpace(invitationID), organizationID).
			First(&invitation).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrInvitationNotFound
			}
			return err
		}
		now := time.Now().UTC()
		switch invitationStatus(invitation, now) {
		case "expired":
			return ErrInvitationExpired
		case "revoked":
			return ErrInvitationRevoked
		case "accepted":
			return ErrInvitationAccepted
		}
		if err := tx.Model(&invitation).Updates(map[string]interface{}{
			"revoked_at": now,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		invitation.RevokedAt = &now
		invitation.UpdatedAt = now
		if err := s.removePrecreatedMembership(tx, invitation); err != nil {
			return err
		}
		view, err := s.invitationViewWithDB(tx, invitation)
		if err != nil {
			return err
		}
		view.Status = "revoked"
		result = view
		return nil
	})
	return result, err
}

// RotateInvitationToken invalidates the previously issued link before an
// administrator retries email delivery. The raw token is never persisted.
func (s *AuthService) RotateInvitationToken(organizationID, invitationID string) (InvitationCreateResult, error) {
	var result InvitationCreateResult
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var invitation model.Invitation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND organization_id = ?", strings.TrimSpace(invitationID), organizationID).
			First(&invitation).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrInvitationNotFound
			}
			return err
		}
		switch invitationStatus(invitation, time.Now().UTC()) {
		case "expired":
			return ErrInvitationExpired
		case "revoked":
			return ErrInvitationRevoked
		case "accepted":
			return ErrInvitationAccepted
		}
		token, err := randomToken()
		if err != nil {
			return err
		}
		if err := tx.Model(&invitation).Updates(map[string]interface{}{
			"token_hash": tokenHash(token),
			"updated_at": time.Now().UTC(),
		}).Error; err != nil {
			return err
		}
		view, err := s.invitationViewWithDB(tx, invitation)
		if err != nil {
			return err
		}
		result = InvitationCreateResult{InvitationView: view, Token: token}
		return nil
	})
	return result, err
}

func (s *AuthService) AcceptInvitation(principal Principal, rawToken string) (InvitationAcceptance, error) {
	var result InvitationAcceptance
	err := s.db.Transaction(func(tx *gorm.DB) error {
		invit, err := s.findInvitation(tx, rawToken, true)
		if err != nil {
			return err
		}
		if !strings.EqualFold(strings.TrimSpace(principal.Email), invit.Email) {
			return ErrInvitationEmailMismatch
		}
		var membership model.Membership
		membershipErr := tx.Where("organization_id = ? AND user_id = ?", invit.OrganizationID, principal.UserID).First(&membership).Error
		if membershipErr == nil && membership.State == "active" {
			return ErrInvitationAlreadyMember
		}
		if membershipErr != nil && membershipErr != gorm.ErrRecordNotFound {
			return membershipErr
		}
		role, err := roleByKey(tx, invit.Role)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if membershipErr == gorm.ErrRecordNotFound {
			membership = model.Membership{ID: uuid.NewString(), OrganizationID: invit.OrganizationID, UserID: principal.UserID, State: "active"}
			if err := tx.Create(&membership).Error; err != nil {
				return err
			}
		} else if err := tx.Model(&membership).Updates(map[string]interface{}{"state": "active", "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Where("membership_id = ?", membership.ID).Delete(&model.MembershipRole{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.MembershipRole{MembershipID: membership.ID, RoleID: role.ID}).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.MembershipEvent{ID: uuid.NewString(), MembershipID: membership.ID, State: "active", Reason: "invitation_accepted"}).Error; err != nil {
			return err
		}
		if err := tx.Model(&invit).Updates(map[string]interface{}{"accepted_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		view, err := s.invitationViewWithDB(tx, invit)
		if err != nil {
			return err
		}
		view.Status = "accepted"
		result = InvitationAcceptance{InvitationView: view, MembershipID: membership.ID}
		return nil
	})
	return result, err
}

func (s *AuthService) invitationView(invit model.Invitation) (InvitationView, error) {
	return s.invitationViewWithDB(s.db, invit)
}

func (s *AuthService) invitationViewWithDB(db *gorm.DB, invit model.Invitation) (InvitationView, error) {
	var organization model.Organization
	if err := db.First(&organization, "id = ?", invit.OrganizationID).Error; err != nil {
		return InvitationView{}, err
	}
	return invitationViewForOrganization(invit, organization.Name, time.Now().UTC()), nil
}

func invitationViewForOrganization(invit model.Invitation, organizationName string, now time.Time) InvitationView {
	return InvitationView{ID: invit.ID, OrganizationID: invit.OrganizationID, Organization: organizationName, Email: invit.Email, Role: invit.Role, Status: invitationStatus(invit, now), ExpiresAt: invit.ExpiresAt, CreatedAt: invit.CreatedAt}
}

func (s *AuthService) findInvitation(db *gorm.DB, rawToken string, lock bool) (model.Invitation, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return model.Invitation{}, ErrInvitationNotFound
	}
	query := db.Where("token_hash = ?", tokenHash(rawToken))
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var invit model.Invitation
	if err := query.First(&invit).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.Invitation{}, ErrInvitationNotFound
		}
		return model.Invitation{}, err
	}
	switch invitationStatus(invit, time.Now().UTC()) {
	case "expired":
		return model.Invitation{}, ErrInvitationExpired
	case "revoked":
		return model.Invitation{}, ErrInvitationRevoked
	case "accepted":
		return model.Invitation{}, ErrInvitationAccepted
	}
	return invit, nil
}

func invitationStatus(invit model.Invitation, now time.Time) string {
	switch {
	case invit.AcceptedAt != nil:
		return "accepted"
	case invit.RevokedAt != nil:
		return "revoked"
	case !invit.ExpiresAt.After(now):
		return "expired"
	default:
		return "pending"
	}
}

func roleByKey(db *gorm.DB, key string) (model.Role, error) {
	var role model.Role
	if err := db.Where("`key` = ?", key).First(&role).Error; err != nil {
		return model.Role{}, err
	}
	return role, nil
}

func validInvitationRole(role string) bool {
	return role == "member" || role == "editor" || role == "administrator"
}

func normalizeInvitationEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func invitedDisplayName(email string) string {
	local := strings.TrimSpace(strings.SplitN(email, "@", 2)[0])
	if local == "" {
		return "待激活成员"
	}
	value := []rune(local)
	if len(value) > 80 {
		value = value[:80]
	}
	return string(value)
}

func (s *AuthService) removePrecreatedMembership(tx *gorm.DB, invitation model.Invitation) error {
	var user model.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("email = ?", invitation.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	var membership model.Membership
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ? AND user_id = ? AND state = ?", invitation.OrganizationID, user.ID, "invited").
		First(&membership).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	if err := tx.Where("membership_id = ?", membership.ID).Delete(&model.MembershipRole{}).Error; err != nil {
		return err
	}
	if err := tx.Where("membership_id = ?", membership.ID).Delete(&model.MembershipEvent{}).Error; err != nil {
		return err
	}
	if err := tx.Delete(&membership).Error; err != nil {
		return err
	}
	if user.State != "invited" {
		return nil
	}
	var remaining int64
	if err := tx.Model(&model.Membership{}).Where("user_id = ?", user.ID).Count(&remaining).Error; err != nil {
		return err
	}
	if remaining == 0 {
		return tx.Delete(&user).Error
	}
	return nil
}

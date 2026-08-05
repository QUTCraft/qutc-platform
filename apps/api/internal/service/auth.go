package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QUTCraft/qutc-platform/apps/api/internal/config"
	"github.com/QUTCraft/qutc-platform/apps/api/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrInvalidRefresh          = errors.New("invalid refresh token")
	ErrInvalidPassword         = errors.New("password must be at least 12 characters")
	ErrEmailInUse              = errors.New("email already registered")
	ErrSessionInactive         = errors.New("user or organization membership is inactive")
	ErrOrganizationUnavailable = errors.New("organization membership is unavailable")
)

type Principal struct {
	UserID         string
	OrganizationID string
	Email          string
}

type Profile struct {
	ID             string   `json:"id"`
	Email          string   `json:"email"`
	DisplayName    string   `json:"display_name"`
	Bio            string   `json:"bio"`
	AvatarURL      string   `json:"avatar_url"`
	OrganizationID string   `json:"organization_id"`
	Roles          []string `json:"roles"`
}

type OrganizationMembershipView struct {
	ID        string   `json:"id"`
	Slug      string   `json:"slug"`
	Name      string   `json:"name"`
	ShortName string   `json:"short_name"`
	Roles     []string `json:"roles"`
	Current   bool     `json:"current"`
}

type TokenPair struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
	TokenType    string  `json:"token_type"`
	ExpiresIn    int64   `json:"expires_in"`
	User         Profile `json:"user"`
}

type accessClaims struct {
	OrganizationID string `json:"org"`
	Email          string `json:"email"`
	jwt.RegisteredClaims
}

type AuthService struct {
	db  *gorm.DB
	cfg config.Config
}

func NewAuthService(db *gorm.DB, cfg config.Config) *AuthService {
	return &AuthService{db: db, cfg: cfg}
}

func (s *AuthService) Register(email, displayName, password string) (TokenPair, error) {
	return s.register(email, displayName, password, "")
}

func (s *AuthService) RegisterWithInvitation(email, displayName, password, invitationToken string) (TokenPair, error) {
	return s.register(email, displayName, password, invitationToken)
}

func (s *AuthService) register(email, displayName, password, invitationToken string) (TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)
	if len(password) < 12 {
		return TokenPair{}, ErrInvalidPassword
	}
	if email == "" || displayName == "" {
		return TokenPair{}, fmt.Errorf("email and display name are required")
	}

	var defaultOrganization model.Organization
	if strings.TrimSpace(invitationToken) == "" {
		if err := s.db.Where("slug = ?", s.cfg.DefaultOrganizationSlug).First(&defaultOrganization).Error; err != nil {
			return TokenPair{}, fmt.Errorf("find default organization: %w", err)
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return TokenPair{}, fmt.Errorf("hash password: %w", err)
	}

	var pair TokenPair
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var existing model.User
		if err := tx.Where("email = ?", email).First(&existing).Error; err == nil {
			return ErrEmailInUse
		} else if err != gorm.ErrRecordNotFound {
			return err
		}
		organizationID := defaultOrganization.ID
		roleKey := "member"
		reason := "registered"
		var invitation model.Invitation
		if strings.TrimSpace(invitationToken) != "" {
			var invitationErr error
			invitation, invitationErr = s.findInvitation(tx, invitationToken, true)
			if invitationErr != nil {
				return invitationErr
			}
			if !strings.EqualFold(email, invitation.Email) {
				return ErrInvitationEmailMismatch
			}
			organizationID = invitation.OrganizationID
			roleKey = invitation.Role
			reason = "invitation_accepted"
		}
		user := model.User{ID: uuid.NewString(), Email: email, DisplayName: displayName, PasswordHash: string(hash), State: "active"}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		membership := model.Membership{ID: uuid.NewString(), OrganizationID: organizationID, UserID: user.ID, State: "active"}
		if err := tx.Create(&membership).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.MembershipEvent{ID: uuid.NewString(), MembershipID: membership.ID, State: "active", Reason: reason}).Error; err != nil {
			return err
		}
		memberRole, err := roleByKey(tx, roleKey)
		if err != nil {
			return err
		}
		if err := tx.Create(&model.MembershipRole{MembershipID: membership.ID, RoleID: memberRole.ID}).Error; err != nil {
			return err
		}
		if strings.TrimSpace(invitationToken) != "" {
			now := time.Now().UTC()
			if err := tx.Model(&model.Invitation{}).Where("token_hash = ?", tokenHash(strings.TrimSpace(invitationToken))).Updates(map[string]interface{}{"accepted_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
		}
		var pairErr error
		pair, pairErr = s.issueTokenPair(tx, user, organizationID)
		return pairErr
	})
	return pair, err
}

func (s *AuthService) Login(email, password string) (TokenPair, error) {
	var user model.User
	if err := s.db.Where("email = ? AND state = ?", strings.ToLower(strings.TrimSpace(email)), "active").First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return TokenPair{}, ErrInvalidCredentials
		}
		return TokenPair{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}
	organizationID, err := s.activeOrganizationID(user.ID)
	if err != nil {
		return TokenPair{}, err
	}
	return s.issueTokenPair(s.db, user, organizationID)
}

func (s *AuthService) Refresh(refreshToken string) (TokenPair, error) {
	hash := tokenHash(refreshToken)
	var stored model.RefreshToken
	if err := s.db.Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, time.Now().UTC()).First(&stored).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return TokenPair{}, ErrInvalidRefresh
		}
		return TokenPair{}, err
	}
	var user model.User
	if err := s.db.Where("id = ? AND state = ?", stored.UserID, "active").First(&user).Error; err != nil {
		return TokenPair{}, ErrInvalidRefresh
	}
	organizationID := strings.TrimSpace(stored.OrganizationID)
	if organizationID == "" {
		var err error
		organizationID, err = s.activeOrganizationID(user.ID)
		if err != nil {
			return TokenPair{}, err
		}
	} else {
		active, err := s.hasActiveMembership(s.db, user.ID, organizationID)
		if err != nil {
			return TokenPair{}, err
		}
		if !active {
			return TokenPair{}, ErrInvalidRefresh
		}
	}
	var pair TokenPair
	err := s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		result := tx.Model(&model.RefreshToken{}).Where("id = ? AND revoked_at IS NULL", stored.ID).Update("revoked_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInvalidRefresh
		}
		var issueErr error
		pair, issueErr = s.issueTokenPair(tx, user, organizationID)
		return issueErr
	})
	return pair, err
}

func (s *AuthService) ListOrganizations(principal Principal) ([]OrganizationMembershipView, error) {
	type organizationMembershipRow struct {
		ID        string
		Slug      string
		Name      string
		ShortName string
	}
	var rows []organizationMembershipRow
	if err := s.db.Table("organizations AS organizations").
		Select("organizations.id, organizations.slug, organizations.name, organizations.short_name").
		Joins("JOIN memberships ON memberships.organization_id = organizations.id").
		Where("memberships.user_id = ? AND memberships.state = ?", principal.UserID, "active").
		Order("memberships.created_at ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]OrganizationMembershipView, 0, len(rows))
	for _, row := range rows {
		roles, err := s.rolesFor(s.db, principal.UserID, row.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, OrganizationMembershipView{
			ID: row.ID, Slug: row.Slug, Name: row.Name, ShortName: row.ShortName,
			Roles: roles, Current: row.ID == principal.OrganizationID,
		})
	}
	return items, nil
}

func (s *AuthService) SwitchOrganization(principal Principal, organizationID, refreshToken string) (TokenPair, error) {
	organizationID = strings.TrimSpace(organizationID)
	refreshToken = strings.TrimSpace(refreshToken)
	if organizationID == "" {
		return TokenPair{}, ErrOrganizationUnavailable
	}
	if refreshToken == "" {
		return TokenPair{}, ErrInvalidRefresh
	}

	var pair TokenPair
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var stored model.RefreshToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash(refreshToken), principal.UserID, time.Now().UTC()).
			First(&stored).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrInvalidRefresh
			}
			return err
		}
		var user model.User
		if err := tx.Where("id = ? AND state = ?", principal.UserID, "active").First(&user).Error; err != nil {
			return ErrInvalidRefresh
		}
		active, err := s.hasActiveMembership(tx, principal.UserID, organizationID)
		if err != nil {
			return err
		}
		if !active {
			return ErrOrganizationUnavailable
		}
		now := time.Now().UTC()
		result := tx.Model(&model.RefreshToken{}).Where("id = ? AND revoked_at IS NULL", stored.ID).Update("revoked_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInvalidRefresh
		}
		var issueErr error
		pair, issueErr = s.issueTokenPair(tx, user, organizationID)
		return issueErr
	})
	return pair, err
}

func (s *AuthService) Logout(refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	now := time.Now().UTC()
	return s.db.Model(&model.RefreshToken{}).Where("token_hash = ? AND revoked_at IS NULL", tokenHash(refreshToken)).Update("revoked_at", now).Error
}

func (s *AuthService) ParseAccessToken(raw string) (Principal, error) {
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTAccessSecret), nil
	}, jwt.WithIssuer(s.cfg.JWTIssuer), jwt.WithLeeway(30*time.Second))
	if err != nil || !token.Valid || claims.Subject == "" || claims.OrganizationID == "" {
		return Principal{}, ErrInvalidCredentials
	}
	return Principal{UserID: claims.Subject, OrganizationID: claims.OrganizationID, Email: claims.Email}, nil
}

func (s *AuthService) AuthenticateAccessToken(raw string) (Principal, error) {
	principal, err := s.ParseAccessToken(raw)
	if err != nil {
		return Principal{}, err
	}
	var count int64
	err = s.db.Table("memberships AS m").
		Joins("JOIN users AS u ON u.id = m.user_id").
		Where(
			"m.user_id = ? AND m.organization_id = ? AND m.state = ? AND u.state = ?",
			principal.UserID,
			principal.OrganizationID,
			"active",
			"active",
		).
		Count(&count).Error
	if err != nil {
		return Principal{}, err
	}
	if count != 1 {
		return Principal{}, ErrSessionInactive
	}
	return principal, nil
}

func (s *AuthService) ProfileFor(principal Principal) (Profile, error) {
	var user model.User
	if err := s.db.Where("id = ? AND state = ?", principal.UserID, "active").First(&user).Error; err != nil {
		return Profile{}, ErrInvalidCredentials
	}
	roles, err := s.rolesFor(s.db, user.ID, principal.OrganizationID)
	if err != nil {
		return Profile{}, err
	}
	return Profile{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Bio: user.Bio, AvatarURL: user.AvatarURL, OrganizationID: principal.OrganizationID, Roles: roles}, nil
}

func (s *AuthService) UpdateProfile(principal Principal, displayName, bio, avatarURL string) (Profile, error) {
	displayName = strings.TrimSpace(displayName)
	bio = strings.TrimSpace(bio)
	avatarURL = strings.TrimSpace(avatarURL)
	if displayName == "" || len([]rune(displayName)) > 80 || len([]rune(bio)) > 500 || len([]rune(avatarURL)) > 500 {
		return Profile{}, fmt.Errorf("profile fields are invalid")
	}
	var user model.User
	if err := s.db.Where("id = ? AND state = ?", principal.UserID, "active").First(&user).Error; err != nil {
		return Profile{}, ErrInvalidCredentials
	}
	user.DisplayName, user.Bio, user.AvatarURL = displayName, bio, avatarURL
	if err := s.db.Save(&user).Error; err != nil {
		return Profile{}, err
	}
	return s.ProfileFor(principal)
}

func (s *AuthService) HasPermission(principal Principal, permission string) (bool, error) {
	var count int64
	err := s.db.Table("permissions").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN membership_roles ON membership_roles.role_id = role_permissions.role_id").
		Joins("JOIN memberships ON memberships.id = membership_roles.membership_id").
		Where("memberships.user_id = ? AND memberships.organization_id = ? AND memberships.state = ? AND permissions.key = ?", principal.UserID, principal.OrganizationID, "active", permission).
		Count(&count).Error
	return count > 0, err
}

func (s *AuthService) issueTokenPair(db *gorm.DB, user model.User, organizationID string) (TokenPair, error) {
	now := time.Now().UTC()
	accessClaims := accessClaims{OrganizationID: organizationID, Email: user.Email, RegisteredClaims: jwt.RegisteredClaims{
		Issuer: s.cfg.JWTIssuer, Subject: user.ID, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTAccessTTL)),
	}}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTAccessSecret))
	if err != nil {
		return TokenPair{}, fmt.Errorf("sign access token: %w", err)
	}
	refreshToken, err := randomToken()
	if err != nil {
		return TokenPair{}, err
	}
	if err := db.Create(&model.RefreshToken{ID: uuid.NewString(), UserID: user.ID, OrganizationID: organizationID, TokenHash: tokenHash(refreshToken), ExpiresAt: now.Add(s.cfg.JWTRefreshTTL)}).Error; err != nil {
		return TokenPair{}, fmt.Errorf("store refresh token: %w", err)
	}
	roles, err := s.rolesFor(db, user.ID, organizationID)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "Bearer", ExpiresIn: int64(s.cfg.JWTAccessTTL.Seconds()), User: Profile{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Bio: user.Bio, AvatarURL: user.AvatarURL, OrganizationID: organizationID, Roles: roles}}, nil
}

func (s *AuthService) hasActiveMembership(db *gorm.DB, userID, organizationID string) (bool, error) {
	var count int64
	if err := db.Model(&model.Membership{}).
		Where("user_id = ? AND organization_id = ? AND state = ?", userID, organizationID, "active").
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == 1, nil
}

func (s *AuthService) activeOrganizationID(userID string) (string, error) {
	var membership model.Membership
	if err := s.db.Where("user_id = ? AND state = ?", userID, "active").Order("created_at ASC").First(&membership).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", ErrInvalidCredentials
		}
		return "", err
	}
	return membership.OrganizationID, nil
}

func (s *AuthService) rolesFor(db *gorm.DB, userID, organizationID string) ([]string, error) {
	var roles []string
	err := db.Table("roles").
		Select("roles.key").
		Joins("JOIN membership_roles ON membership_roles.role_id = roles.id").
		Joins("JOIN memberships ON memberships.id = membership_roles.membership_id").
		Where("memberships.user_id = ? AND memberships.organization_id = ? AND memberships.state = ?", userID, organizationID, "active").
		Order("roles.key ASC").Scan(&roles).Error
	return roles, err
}

func randomToken() (string, error) {
	bytes := make([]byte, 48)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum[:])
}

package model

import (
	"time"

	"github.com/rs/xid"
)

type UserID string

func NewUserID() UserID {
	return UserID(xid.New().String())
}

type User interface {
	WithID[UserID]

	Email() string

	Subject() string
	Provider() string

	DisplayName() string

	Roles() []string

	Active() bool

	Preferences() UserPreferences
}

type BaseUser struct {
	id          UserID
	displayName string
	email       string
	subject     string
	provider    string
	roles       []string
	active      bool
	preferences UserPreferences
}

// Preferences implements [User].
func (u *BaseUser) Preferences() UserPreferences {
	return u.preferences
}

// Active implements [User].
func (u *BaseUser) Active() bool {
	return u.active
}

// Email implements [User].
func (u *BaseUser) Email() string {
	return u.email
}

// ID implements [User].
func (u *BaseUser) ID() UserID {
	return u.id
}

// DisplayName implements User.
func (u *BaseUser) DisplayName() string {
	return u.displayName
}

// Provider implements User.
func (u *BaseUser) Provider() string {
	return u.provider
}

// Roles implements User.
func (u *BaseUser) Roles() []string {
	return u.roles
}

// Subject implements User.
func (u *BaseUser) Subject() string {
	return u.subject
}

var _ User = &BaseUser{}

func CopyUser(user User) *BaseUser {
	return &BaseUser{
		id:          user.ID(),
		displayName: user.DisplayName(),
		email:       user.Email(),
		subject:     user.Subject(),
		provider:    user.Provider(),
		active:      user.Active(),
		preferences: user.Preferences(),
		roles:       append([]string{}, user.Roles()...),
	}
}

func NewUser(provider, subject, email string, displayName string, active bool, roles ...string) *BaseUser {
	return &BaseUser{
		id:          NewUserID(),
		displayName: displayName,
		email:       email,
		subject:     subject,
		provider:    provider,
		roles:       roles,
		active:      active,
	}
}

func (u *BaseUser) SetDisplayName(displayName string) {
	u.displayName = displayName
}

func (u *BaseUser) SetActive(active bool) {
	u.active = active
}

func (u *BaseUser) SetEmail(email string) {
	u.email = email
}

func (u *BaseUser) SetRoles(roles ...string) {
	u.roles = roles
}

func (u *BaseUser) SetPreferences(preferences UserPreferences) {
	u.preferences = preferences
}

type AuthTokenID string

func NewAuthTokenID() AuthTokenID {
	return AuthTokenID(xid.New().String())
}

type AuthToken interface {
	WithID[AuthTokenID]
	WithOwner

	Label() string
	Value() string
	OrgID() OrgID
	ExpiresAt() *time.Time
}

type BaseAuthToken struct {
	id        AuthTokenID
	owner     User
	label     string
	value     string
	orgID     OrgID
	expiresAt *time.Time
}

// ID implements AuthToken.
func (t *BaseAuthToken) ID() AuthTokenID { return t.id }

// Owner implements AuthToken.
func (t *BaseAuthToken) Owner() User { return t.owner }

// Label implements AuthToken.
func (t *BaseAuthToken) Label() string { return t.label }

// Value implements AuthToken.
func (t *BaseAuthToken) Value() string { return t.value }

// OrgID implements AuthToken.
func (t *BaseAuthToken) OrgID() OrgID { return t.orgID }

// ExpiresAt implements AuthToken.
func (t *BaseAuthToken) ExpiresAt() *time.Time { return t.expiresAt }

var _ AuthToken = &BaseAuthToken{}

func NewAuthToken(owner User, orgID OrgID, label, value string, expiresAt *time.Time) *BaseAuthToken {
	return &BaseAuthToken{
		id:        NewAuthTokenID(),
		owner:     owner,
		orgID:     orgID,
		label:     label,
		value:     value,
		expiresAt: expiresAt,
	}
}

type UserPreferences interface {
	DarkMode() (bool, bool)
}

type BaseUserPreferences struct {
	darkMode *bool
}

// DarkMode implements [UserPreferences].
func (p *BaseUserPreferences) DarkMode() (bool, bool) {
	if p.darkMode == nil {
		return false, false
	}

	return *p.darkMode, true
}

func SetUserPrefencesDarkMode(darkMode *bool) BaseUserPreferencesSetter {
	return func(p *BaseUserPreferences) {
		p.darkMode = darkMode
	}
}

func (p *BaseUserPreferences) Set(setters ...BaseUserPreferencesSetter) {
	for _, s := range setters {
		s(p)
	}
}

type BaseUserPreferencesSetter func(p *BaseUserPreferences)

func NewUserPreferences(setters ...BaseUserPreferencesSetter) *BaseUserPreferences {
	preferences := &BaseUserPreferences{
		darkMode: nil,
	}
	preferences.Set(setters...)
	return preferences
}

var _ UserPreferences = &BaseUserPreferences{}

package gorm

import (
	"slices"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type User struct {
	ID string `gorm:"primaryKey;autoIncrement:false"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Subject  string `gorm:"index"`
	Provider string `gorm:"index"`

	DisplayName string
	Email       string `gorm:"unique"`

	AuthTokens []*AuthToken `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE;"`

	Roles []*UserRole `gorm:"constraint:OnDelete:CASCADE;"`

	Active      bool
	Preferences *UserPreferences `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// wrappedUserPreference implements model.UserPreferences
type wrappedUserPreference struct {
	p *UserPreferences
}

// DarkMode implements model.UserPreferences.
func (p *wrappedUserPreference) DarkMode() (bool, bool) {
	if p.p.DarkMode == nil {
		return false, false
	}
	return *p.p.DarkMode, true
}

var _ model.UserPreferences = &wrappedUserPreference{}

// UserPreference is a separate table for storing user preferences
type UserPreferences struct {
	ID       uint   `gorm:"primaryKey"`
	UserID   string `gorm:"unique;index"`
	DarkMode *bool
}

// fromUser converts a model.User to a GORM User
func fromUser(u model.User) *User {
	user := &User{
		ID:          string(u.ID()),
		Subject:     u.Subject(),
		Provider:    u.Provider(),
		DisplayName: u.DisplayName(),
		Email:       u.Email(),
		Active:      u.Active(),
	}

	prefs := u.Preferences()

	darkMode, darkModeExists := prefs.DarkMode()

	user.Preferences = &UserPreferences{
		UserID:   string(u.ID()),
		DarkMode: nil,
	}

	if darkModeExists {
		user.Preferences.DarkMode = &darkMode
	}

	for _, r := range u.Roles() {
		user.Roles = append(user.Roles, &UserRole{
			User:   user,
			UserID: user.ID,
			Role:   r,
		})
	}

	return user
}

type AuthToken struct {
	ID string `gorm:"primaryKey;autoIncrement:false"`

	CreatedAt time.Time
	UpdatedAt time.Time

	Owner   *User
	OwnerID string

	Label     string
	Value     string     `gorm:"unique"`
	OrgID     string     `gorm:"index"`
	ExpiresAt *time.Time `gorm:"index"`
}

type UserRole struct {
	ID uint `gorm:"primaryKey"`

	CreatedAt time.Time

	User   *User
	UserID string `gorm:"index:user_role_index,unique"`

	Role string `gorm:"index:user_role_index,unique"`
}

// wrappedUser implements the model.User interface
type wrappedUser struct {
	u *User
}

// Active implements [model.User].
func (w *wrappedUser) Active() bool {
	return w.u.Active
}

// ID implements model.User.
func (w *wrappedUser) ID() model.UserID {
	return model.UserID(w.u.ID)
}

// Email implements model.User.
func (w *wrappedUser) Email() string {
	return w.u.Email
}

// Subject implements model.User.
func (w *wrappedUser) Subject() string {
	return w.u.Subject
}

// Provider implements model.User.
func (w *wrappedUser) Provider() string {
	return w.u.Provider
}

// DisplayName implements model.User.
func (w *wrappedUser) DisplayName() string {
	return w.u.DisplayName
}

// Roles implements model.User.
func (w *wrappedUser) Roles() []string {
	return slices.Collect(func(yield func(string) bool) {
		for _, r := range w.u.Roles {
			if !yield(r.Role) {
				return
			}
		}
	})
}

// Preferences implements model.User.
func (w *wrappedUser) Preferences() model.UserPreferences {
	if w.u.Preferences == nil {
		return model.NewUserPreferences()
	}
	return &wrappedUserPreference{w.u.Preferences}
}

var _ model.User = &wrappedUser{}

// wrappedAuthToken implements the model.AuthToken interface
type wrappedAuthToken struct {
	t *AuthToken
}

func (w *wrappedAuthToken) ID() model.AuthTokenID { return model.AuthTokenID(w.t.ID) }
func (w *wrappedAuthToken) Owner() model.User     { return &wrappedUser{w.t.Owner} }
func (w *wrappedAuthToken) Label() string         { return w.t.Label }
func (w *wrappedAuthToken) Value() string         { return w.t.Value }
func (w *wrappedAuthToken) OrgID() model.OrgID    { return model.OrgID(w.t.OrgID) }
func (w *wrappedAuthToken) ExpiresAt() *time.Time { return w.t.ExpiresAt }

var _ model.AuthToken = &wrappedAuthToken{}

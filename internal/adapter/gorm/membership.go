package gorm

import (
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type Membership struct {
	ID        string `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt time.Time
	UserID    string `gorm:"index;not null"`
	OrgID     string `gorm:"index;not null"`
	Role      string `gorm:"not null"`

	User *User         `gorm:"foreignKey:UserID"`
	Org  *Organization `gorm:"foreignKey:OrgID"`
}

type wrappedMembership struct {
	m *Membership
}

func (w *wrappedMembership) ID() model.MembershipID   { return model.MembershipID(w.m.ID) }
func (w *wrappedMembership) UserID() model.UserID      { return model.UserID(w.m.UserID) }
func (w *wrappedMembership) OrgID() model.OrgID        { return model.OrgID(w.m.OrgID) }
func (w *wrappedMembership) Role() string              { return w.m.Role }
func (w *wrappedMembership) CreatedAt() time.Time      { return w.m.CreatedAt }
func (w *wrappedMembership) User() model.User {
	if w.m.User == nil {
		return nil
	}
	return &wrappedUser{w.m.User}
}
func (w *wrappedMembership) Org() model.Organization {
	if w.m.Org == nil {
		return nil
	}
	return &wrappedOrganization{w.m.Org}
}

var _ model.Membership = &wrappedMembership{}

func fromMembership(m model.Membership) *Membership {
	return &Membership{
		ID:     string(m.ID()),
		UserID: string(m.UserID()),
		OrgID:  string(m.OrgID()),
		Role:   m.Role(),
	}
}

package model

import (
	"time"

	"github.com/rs/xid"
)

type MembershipID string

func NewMembershipID() MembershipID {
	return MembershipID(xid.New().String())
}

const (
	RoleOrgOwner = "org:owner"
	RoleOrgAdmin = "org:admin"
	RoleMember   = "member"
)

type Membership interface {
	WithID[MembershipID]

	UserID() UserID
	OrgID() OrgID
	Role() string
	CreatedAt() time.Time

	// Populated via preload
	User() User
	Org() Organization
}

type BaseMembership struct {
	id        MembershipID
	userID    UserID
	orgID     OrgID
	role      string
	createdAt time.Time
	user      User
	org       Organization
}

func (m *BaseMembership) ID() MembershipID     { return m.id }
func (m *BaseMembership) UserID() UserID        { return m.userID }
func (m *BaseMembership) OrgID() OrgID          { return m.orgID }
func (m *BaseMembership) Role() string           { return m.role }
func (m *BaseMembership) CreatedAt() time.Time   { return m.createdAt }
func (m *BaseMembership) User() User             { return m.user }
func (m *BaseMembership) Org() Organization      { return m.org }

var _ Membership = &BaseMembership{}

func NewMembership(userID UserID, orgID OrgID, role string) *BaseMembership {
	return &BaseMembership{
		id:        NewMembershipID(),
		userID:    userID,
		orgID:     orgID,
		role:      role,
		createdAt: time.Now(),
	}
}

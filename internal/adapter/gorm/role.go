package gorm

import (
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type Role struct {
	ID          string `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	OrgID       string `gorm:"index;uniqueIndex:role_org_name_index;not null"`
	Name        string `gorm:"uniqueIndex:role_org_name_index;not null"`
	Description string
	Builtin     bool   `gorm:"not null;default:false"`
	BuiltinKind string `gorm:"index"`

	Org         *Organization     `gorm:"foreignKey:OrgID"`
	Permissions []*RolePermission `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE"`
	ModelGrants []*RoleModel      `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE"`
}

type RolePermission struct {
	ID     uint   `gorm:"primaryKey"`
	RoleID string `gorm:"index:role_perm_index,unique;not null"`
	Code   string `gorm:"index:role_perm_index,unique;not null"`
}

type RoleModel struct {
	ID        uint   `gorm:"primaryKey"`
	RoleID    string `gorm:"index:role_model_index,unique;not null"`
	ModelID   string `gorm:"index:role_model_index,unique;not null"`
	ModelKind string `gorm:"index:role_model_index,unique;not null"`
}

// MembershipRole is the join table between memberships and roles.
type MembershipRole struct {
	MembershipID string `gorm:"primaryKey"`
	RoleID       string `gorm:"primaryKey"`
	CreatedAt    time.Time
}

type wrappedRole struct {
	r *Role
}

func (w *wrappedRole) ID() model.RoleID      { return model.RoleID(w.r.ID) }
func (w *wrappedRole) OrgID() model.OrgID    { return model.OrgID(w.r.OrgID) }
func (w *wrappedRole) Name() string          { return w.r.Name }
func (w *wrappedRole) Description() string    { return w.r.Description }
func (w *wrappedRole) Builtin() bool         { return w.r.Builtin }
func (w *wrappedRole) BuiltinKind() string    { return w.r.BuiltinKind }
func (w *wrappedRole) CreatedAt() time.Time   { return w.r.CreatedAt }
func (w *wrappedRole) UpdatedAt() time.Time   { return w.r.UpdatedAt }

func (w *wrappedRole) Permissions() []string {
	codes := make([]string, 0, len(w.r.Permissions))
	for _, p := range w.r.Permissions {
		codes = append(codes, p.Code)
	}
	return codes
}

func (w *wrappedRole) ModelGrants() []model.ModelGrant {
	grants := make([]model.ModelGrant, 0, len(w.r.ModelGrants))
	for _, g := range w.r.ModelGrants {
		grants = append(grants, model.ModelGrant{ModelID: g.ModelID, Kind: g.ModelKind})
	}
	return grants
}

var _ model.Role = &wrappedRole{}

func fromRole(r model.Role) *Role {
	role := &Role{
		ID:          string(r.ID()),
		OrgID:       string(r.OrgID()),
		Name:        r.Name(),
		Description: r.Description(),
		Builtin:     r.Builtin(),
		BuiltinKind: r.BuiltinKind(),
		CreatedAt:   r.CreatedAt(),
		UpdatedAt:   r.UpdatedAt(),
	}
	for _, code := range r.Permissions() {
		role.Permissions = append(role.Permissions, &RolePermission{
			RoleID: role.ID,
			Code:   code,
		})
	}
	for _, grant := range r.ModelGrants() {
		role.ModelGrants = append(role.ModelGrants, &RoleModel{
			RoleID:    role.ID,
			ModelID:   grant.ModelID,
			ModelKind: grant.Kind,
		})
	}
	return role
}

package events

import (
	"context"
	"strconv"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

// membershipResolver resolves the org and user a membership belongs to. It lets
// the role decorator attribute membership-role changes (a cross-aggregate
// operation) to the right organization and user.
type membershipResolver interface {
	GetMembership(ctx context.Context, id model.MembershipID) (model.Membership, error)
}

// RoleStore decorates a port.RoleStore, emitting events on custom role
// mutations. Builtin roles (created via EnsureBuiltinRoles or otherwise flagged)
// never produce events.
type RoleStore struct {
	port.RoleStore
	emitter     port.EventEmitter
	memberships membershipResolver
}

func NewRoleStore(backend port.RoleStore, emitter port.EventEmitter, memberships membershipResolver) *RoleStore {
	return &RoleStore{RoleStore: backend, emitter: emitter, memberships: memberships}
}

func (s *RoleStore) CreateRole(ctx context.Context, role model.Role) error {
	if err := s.RoleStore.CreateRole(ctx, role); err != nil {
		return err
	}
	if role.Builtin() {
		return nil
	}
	emit(ctx, s.emitter, role.OrgID(), model.SeverityInfo, model.EventTypeRoleCreated,
		"Rôle créé : "+role.Name(),
		map[string]string{"role_id": string(role.ID()), "role_name": role.Name()})
	return nil
}

func (s *RoleStore) SaveRole(ctx context.Context, role model.Role) error {
	if err := s.RoleStore.SaveRole(ctx, role); err != nil {
		return err
	}
	if role.Builtin() {
		return nil
	}
	emit(ctx, s.emitter, role.OrgID(), model.SeverityInfo, model.EventTypeRoleUpdated,
		"Rôle modifié : "+role.Name(),
		map[string]string{"role_id": string(role.ID()), "role_name": role.Name()})
	return nil
}

func (s *RoleStore) DeleteRole(ctx context.Context, id model.RoleID) error {
	existing, _ := s.RoleStore.GetRoleByID(ctx, id)
	if err := s.RoleStore.DeleteRole(ctx, id); err != nil {
		return err
	}
	if existing != nil && existing.Builtin() {
		return nil
	}
	orgID := model.OrgID("")
	attrs := map[string]string{"role_id": string(id)}
	msg := "Rôle supprimé"
	if existing != nil {
		orgID = existing.OrgID()
		attrs["role_name"] = existing.Name()
		msg = "Rôle supprimé : " + existing.Name()
	}
	emit(ctx, s.emitter, orgID, model.SeverityWarning, model.EventTypeRoleDeleted, msg, attrs)
	return nil
}

func (s *RoleStore) SetMembershipRoles(ctx context.Context, membershipID model.MembershipID, roleIDs []model.RoleID) error {
	if err := s.RoleStore.SetMembershipRoles(ctx, membershipID, roleIDs); err != nil {
		return err
	}
	orgID := model.OrgID("")
	userID := model.UserID("")
	if s.memberships != nil {
		if m, err := s.memberships.GetMembership(ctx, membershipID); err == nil && m != nil {
			orgID = m.OrgID()
			userID = m.UserID()
		}
	}
	emit(ctx, s.emitter, orgID, model.SeverityInfo, model.EventTypeMemberUpdated,
		"Rôles de membre modifiés",
		map[string]string{
			"membership_id":  string(membershipID),
			"member_user_id": string(userID),
			"role_count":     strconv.Itoa(len(roleIDs)),
		})
	return nil
}

var _ port.RoleStore = &RoleStore{}

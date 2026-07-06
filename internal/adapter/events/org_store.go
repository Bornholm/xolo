package events

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

// OrgStore decorates a port.OrgStore, emitting events on membership changes.
type OrgStore struct {
	port.OrgStore
	emitter port.EventEmitter
}

func NewOrgStore(backend port.OrgStore, emitter port.EventEmitter) *OrgStore {
	return &OrgStore{OrgStore: backend, emitter: emitter}
}

func (s *OrgStore) AddMember(ctx context.Context, membership model.Membership) error {
	if err := s.OrgStore.AddMember(ctx, membership); err != nil {
		return err
	}
	emit(ctx, s.emitter, membership.OrgID(), model.SeverityInfo, model.EventTypeMemberAdded,
		"Membre ajouté à l'organisation",
		map[string]string{
			"membership_id":  string(membership.ID()),
			"member_user_id": string(membership.UserID()),
		})
	return nil
}

func (s *OrgStore) RemoveMember(ctx context.Context, id model.MembershipID) error {
	existing, _ := s.OrgStore.GetMembership(ctx, id)
	if err := s.OrgStore.RemoveMember(ctx, id); err != nil {
		return err
	}
	orgID := model.OrgID("")
	attrs := map[string]string{"membership_id": string(id)}
	if existing != nil {
		orgID = existing.OrgID()
		attrs["member_user_id"] = string(existing.UserID())
	}
	emit(ctx, s.emitter, orgID, model.SeverityWarning, model.EventTypeMemberRemoved,
		"Membre retiré de l'organisation", attrs)
	return nil
}

var _ port.OrgStore = &OrgStore{}

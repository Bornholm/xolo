package events

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

// InviteStore decorates a port.InviteStore, emitting events on invite mutations.
type InviteStore struct {
	port.InviteStore
	emitter port.EventEmitter
}

func NewInviteStore(backend port.InviteStore, emitter port.EventEmitter) *InviteStore {
	return &InviteStore{InviteStore: backend, emitter: emitter}
}

func inviteAttrs(invite model.InviteToken) map[string]string {
	attrs := map[string]string{
		"invite_id": string(invite.ID()),
		"role":      invite.Role(),
	}
	if email := invite.InviteeEmail(); email != nil {
		attrs["email"] = *email
	}
	return attrs
}

func (s *InviteStore) CreateInvite(ctx context.Context, invite model.InviteToken) error {
	if err := s.InviteStore.CreateInvite(ctx, invite); err != nil {
		return err
	}
	emit(ctx, s.emitter, invite.OrgID(), model.SeverityInfo, model.EventTypeInviteCreated,
		"Invitation créée", inviteAttrs(invite))
	return nil
}

func (s *InviteStore) DeleteInvite(ctx context.Context, id model.InviteTokenID) error {
	existing, _ := s.InviteStore.GetInviteByID(ctx, id)
	if err := s.InviteStore.DeleteInvite(ctx, id); err != nil {
		return err
	}
	orgID := model.OrgID("")
	attrs := map[string]string{"invite_id": string(id)}
	if existing != nil {
		orgID = existing.OrgID()
		attrs = inviteAttrs(existing)
	}
	emit(ctx, s.emitter, orgID, model.SeverityWarning, model.EventTypeInviteDeleted,
		"Invitation supprimée", attrs)
	return nil
}

var _ port.InviteStore = &InviteStore{}

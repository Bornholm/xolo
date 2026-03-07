package port

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

type InviteStore interface {
	CreateInvite(ctx context.Context, invite model.InviteToken) error
	GetInviteByID(ctx context.Context, id model.InviteTokenID) (model.InviteToken, error)
	ListInvites(ctx context.Context, orgID model.OrgID) ([]model.InviteToken, error)
	RevokeInvite(ctx context.Context, id model.InviteTokenID) error
	DeleteInvite(ctx context.Context, id model.InviteTokenID) error
	IncrementInviteUses(ctx context.Context, id model.InviteTokenID) error
	// ListPendingInvitesForEmail returns non-expired, non-revoked targeted invites for an email.
	ListPendingInvitesForEmail(ctx context.Context, email string) ([]model.InviteToken, error)
}

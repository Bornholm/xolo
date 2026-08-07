package context

import (
	"context"

	"github.com/xolo-gateway/xolo/internal/core/model"
)

const keyMemberships contextKey = "memberships"

func Memberships(ctx context.Context) []model.Membership {
	memberships, ok := ctx.Value(keyMemberships).([]model.Membership)
	if !ok {
		return nil
	}
	return memberships
}

func SetMemberships(ctx context.Context, memberships []model.Membership) context.Context {
	return context.WithValue(ctx, keyMemberships, memberships)
}

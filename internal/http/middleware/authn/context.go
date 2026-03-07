package authn

import (
	"context"

	"github.com/pkg/errors"
)

type contextKey string

const keyUser contextKey = "user"

func ContextUser(ctx context.Context) *User {
	user, ok := ctx.Value(keyUser).(*User)
	if !ok {
		panic(errors.New("no user in context"))
	}

	return user
}

func setContextUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, keyUser, user)
}

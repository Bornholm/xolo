package context

import (
	"context"
	"net/url"

	"github.com/pkg/errors"
)

const keyCurrentURL contextKey = "currentURL"

func CurrentURL(ctx context.Context) *url.URL {
	currentURL, ok := ctx.Value(keyCurrentURL).(*url.URL)
	if !ok {
		panic(errors.New("no current url in context"))
	}

	return currentURL
}

func SetCurrentURL(ctx context.Context, u *url.URL) context.Context {
	return context.WithValue(ctx, keyCurrentURL, u)
}

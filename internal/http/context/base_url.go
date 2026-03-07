package context

import (
	"context"
	"net/url"

	"github.com/pkg/errors"
)

const keyBaseURL contextKey = "baseURL"

func BaseURL(ctx context.Context) *url.URL {
	rawBaseURL, ok := ctx.Value(keyBaseURL).(string)
	if !ok {
		panic(errors.New("no base url in context"))
	}

	baseURL, err := url.Parse(rawBaseURL)
	if err != nil {
		panic(errors.WithStack(err))
	}

	return baseURL
}

func SetBaseURL(ctx context.Context, baseURL string) context.Context {
	return context.WithValue(ctx, keyBaseURL, baseURL)
}

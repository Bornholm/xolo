package context

import "context"

func ColorScheme(ctx context.Context) string {
	scheme, ok := ctx.Value(keyColorScheme).(string)
	if !ok {
		return "light"
	}

	return scheme
}

func SetColorScheme(ctx context.Context, scheme string) context.Context {
	return context.WithValue(ctx, keyColorScheme, scheme)
}

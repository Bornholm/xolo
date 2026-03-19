package colorscheme

import (
	"net/http"

	httpCtx "github.com/bornholm/xolo/internal/http/context"
)

const colorSchemeHeader string = "Sec-CH-Prefers-Color-Scheme"

func Middleware() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		var fn http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			scheme := r.Header.Get(colorSchemeHeader)
			if scheme == "" {
				scheme = "light"
			}

			ctx = httpCtx.SetColorScheme(ctx, scheme)
			r = r.WithContext(ctx)

			w.Header().Add("Accept-CH", colorSchemeHeader)

			h.ServeHTTP(w, r)
		}

		return fn
	}
}

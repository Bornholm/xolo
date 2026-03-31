package authn

import (
	"log/slog"
	"net/http"

	"github.com/bornholm/xolo/internal/http/handler/webui/common"
	"github.com/bornholm/xolo/internal/metrics"
	"github.com/pkg/errors"
)

var (
	ErrSkipRequest = errors.New("skip request")
)

type Authenticator interface {
	Authenticate(w http.ResponseWriter, r *http.Request) (*User, error)
}

func Middleware(onUnauthorized func(w http.ResponseWriter, r *http.Request), authenticators ...Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		var fn http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			for _, authenticator := range authenticators {
				user, err := authenticator.Authenticate(w, r)
				if err != nil {
					if errors.Is(err, ErrSkipRequest) {
						return
					}

					slog.ErrorContext(r.Context(), "could not authenticate user", slog.Any("error", errors.WithStack(err)))
					common.HandleError(w, r, err)
					return
				}

				if user == nil {
					continue
				}

				ctx := r.Context()
				ctx = setContextUser(ctx, user)

				r = r.WithContext(ctx)

				next.ServeHTTP(w, r)
				return
			}

			metrics.AuthFailures.Inc()
			onUnauthorized(w, r)
		}

		return fn
	}
}

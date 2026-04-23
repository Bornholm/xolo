package memberships

import (
	"log/slog"
	"net/http"

	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/go-x/slogx"
)

// Middleware fetches the current user's org memberships and stores them in the
// request context so that templates can render role-aware navigation items
// without each handler needing to fetch them individually.
func Middleware(orgStore port.OrgStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user := httpCtx.User(ctx)

			slog.Debug("memberships middleware: start", "hasUser", user != nil)

			if user != nil {
				memberships, err := orgStore.GetUserMemberships(ctx, user.ID())
				slog.Debug("memberships middleware: fetched", "userID", user.ID(), "count", len(memberships), "error", err)
				if err != nil {
					slog.ErrorContext(ctx, "memberships middleware: could not fetch memberships", slogx.Error(err))
				} else {
					ctx = httpCtx.SetMemberships(ctx, memberships)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

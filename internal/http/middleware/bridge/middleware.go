package bridge

import (
	"errors"
	"net/http"
	"slices"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/http/handler/webui/common"
	"github.com/bornholm/xolo/internal/http/middleware/authn"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
)

func Middleware(userStore port.UserStore, activeByDefault bool, defaultAdmins ...string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		var fn http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			authnUser := authn.ContextUser(ctx)
			if authnUser == nil {
				common.HandleError(w, r, common.NewHTTPError(http.StatusUnauthorized))
				return
			}

			user, err := userStore.FindOrCreateUser(ctx, authnUser.Provider, authnUser.Subject)
			if err != nil {
				if errors.Is(err, port.ErrNotFound) {
					user = model.NewUser(authnUser.Provider, authnUser.Subject, authnUser.Email, authnUser.DisplayName, activeByDefault, authz.RoleUser)

					if err := userStore.SaveUser(ctx, user); err != nil {
						common.HandleError(w, r, err)
						return
					}
				} else {
					common.HandleError(w, r, err)
					return
				}
			}

			missingRole := len(user.Roles()) == 0
			shouldBeAdmin := slices.Contains(defaultAdmins, authnUser.Email) && !slices.Contains(user.Roles(), authz.RoleAdmin)

			changed := (authnUser.DisplayName != "" && user.DisplayName() != authnUser.DisplayName) ||
				user.Email() != authnUser.Email

			if changed || shouldBeAdmin || missingRole {
				updatable := model.CopyUser(user)
				if authnUser.DisplayName != "" {
					updatable.SetDisplayName(authnUser.DisplayName)
				}
				updatable.SetEmail(authnUser.Email)

				if missingRole {
					updatable.SetRoles(authz.RoleUser)
				}

				if shouldBeAdmin {
					newRoles := append(user.Roles(), authz.RoleAdmin)
					updatable.SetRoles(newRoles...)
					updatable.SetActive(true)
				}

				if err := userStore.SaveUser(ctx, updatable); err != nil {
					common.HandleError(w, r, err)
					return
				}

				user = updatable
			}

			ctx = httpCtx.SetUser(ctx, user)
			r = r.WithContext(ctx)

			h.ServeHTTP(w, r)
		}

		return fn
	}
}

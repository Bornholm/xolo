package bridge

import (
	"context"
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

func Middleware(userStore port.UserStore, emitter port.EventEmitter, activeByDefault bool, defaultAdmins ...string) func(http.Handler) http.Handler {
	emitLoginFailed := func(ctx context.Context, authnUser *authn.User, reason string) {
		if emitter == nil || authnUser == nil {
			return
		}
		emitter.Emit(ctx, model.NewEvent(model.EventSourcePlatform, model.EventTypeAuthLoginFailed,
			model.WithEventSeverity(model.SeverityWarning),
			model.WithEventMessage("Échec de connexion: "+reason),
			model.WithEventAttribute("email", authnUser.Email),
			model.WithEventAttribute("provider", authnUser.Provider),
			model.WithEventAttribute("reason", reason),
		))
	}

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
						if errors.Is(err, port.ErrAlreadyExists) {
							emitLoginFailed(ctx, authnUser, "un compte existe déjà avec cette adresse email")
							common.HandleError(w, r, common.NewError(
								err.Error(),
								"Un compte existe déjà avec cette adresse email. Contactez un administrateur pour faire fusionner vos comptes.",
								http.StatusConflict,
							))
							return
						}

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

			// Never overwrite a stored value with an empty incoming one: some
			// authenticators (e.g. OAuth2 introspection) resolve an identity
			// without an email or display name.
			changed := (authnUser.DisplayName != "" && user.DisplayName() != authnUser.DisplayName) ||
				(authnUser.Email != "" && user.Email() != authnUser.Email)

			if changed || shouldBeAdmin || missingRole {
				updatable := model.CopyUser(user)
				if authnUser.DisplayName != "" {
					updatable.SetDisplayName(authnUser.DisplayName)
				}
				if authnUser.Email != "" {
					updatable.SetEmail(authnUser.Email)
				}

				if missingRole {
					updatable.SetRoles(authz.RoleUser)
				}

				if shouldBeAdmin {
					newRoles := append(user.Roles(), authz.RoleAdmin)
					updatable.SetRoles(newRoles...)
					updatable.SetActive(true)
				}

				if err := userStore.SaveUser(ctx, updatable); err != nil {
					if errors.Is(err, port.ErrAlreadyExists) {
						common.HandleError(w, r, common.NewError(
							err.Error(),
							"Un compte existe déjà avec cette adresse email. Contactez un administrateur pour faire fusionner vos comptes.",
							http.StatusConflict,
						))
						return
					}

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

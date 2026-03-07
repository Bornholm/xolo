package token

import (
	"net/http"
	"strings"

	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/http/middleware/authn"
	"github.com/pkg/errors"
)

// Authenticate implements [authn.Authenticator].
func (h *Handler) Authenticate(w http.ResponseWriter, r *http.Request) (*authn.User, error) {
	authorization := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authorization, "Bearer ")

	if token != "" {
		user, err := h.getUserFromToken(r.Context(), token)
		if err != nil && !errors.Is(err, port.ErrNotFound) {
			return nil, errors.WithStack(err)
		}

		return user, nil
	}

	user, err := h.retrieveSessionUser(r)
	if err != nil {
		if errors.Is(err, errSessionNotFound) {
			return nil, nil
		}

		return nil, errors.WithStack(err)
	}

	return user, nil
}

var _ authn.Authenticator = &Handler{}

package oidc

import (
	"net/http"

	"github.com/bornholm/xolo/internal/http/middleware/authn"
	"github.com/pkg/errors"
)

// Authenticate implements [authn.Authenticator].
func (h *Handler) Authenticate(w http.ResponseWriter, r *http.Request) (*authn.User, error) {
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

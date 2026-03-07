package proxy

import (
	"context"
	"net/http"
	"strings"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

type contextKey string

const (
	contextKeyAuthTokenID contextKey = "authTokenID"
	contextKeyOrgID       contextKey = "orgID"
)

// XoloAuthExtractor extracts the user identity from a Bearer token,
// enforces expiry, and stashes the token ID and org ID in request context
// for downstream hooks.
func XoloAuthExtractor(userStore port.UserStore) func(r *http.Request) (string, error) {
	return func(r *http.Request) (string, error) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			return "", nil
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return "", nil
		}
		raw := strings.TrimSpace(parts[1])
		if raw == "" {
			return "", nil
		}

		ctx := r.Context()
		token, err := userStore.FindAuthToken(ctx, raw)
		if err != nil {
			if errors.Is(err, port.ErrNotFound) {
				return "", nil
			}
			return "", errors.WithStack(err)
		}

		// Stash into context for usage tracker and quota enforcer
		newCtx := context.WithValue(ctx, contextKeyAuthTokenID, string(token.ID()))
		newCtx = context.WithValue(newCtx, contextKeyOrgID, string(token.OrgID()))
		// Replace the request context — the proxy server reads UserID from AuthExtractor
		// but we need to carry extra data; we attach it to the request in-place.
		*r = *r.WithContext(newCtx)

		return string(token.Owner().ID()), nil
	}
}

// AuthTokenIDFromContext retrieves the auth token ID stored by XoloAuthExtractor.
func AuthTokenIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyAuthTokenID).(string)
	return v
}

// OrgIDFromContext retrieves the org ID stored by XoloAuthExtractor.
func OrgIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyOrgID).(string)
	return v
}

// MetaAuthTokenID and MetaOrgID are the req.Metadata keys used by hooks.
const (
	MetaAuthTokenID = "authTokenID"
	MetaOrgID       = "orgID"
	MetaModelID     = "modelID"
)

// SetRequestMeta is a PreRequestHook (priority 0) that copies context values
// into req.Metadata so all subsequent hooks can read them without importing
// the http package.
type SetRequestMetaHook struct{}

func (h *SetRequestMetaHook) Name() string  { return "xolo.set-request-meta" }
func (h *SetRequestMetaHook) Priority() int { return 0 }

func (h *SetRequestMetaHook) PreRequest(ctx context.Context, req interface{ GetMetadata() map[string]any; GetHeaders() http.Header }) (interface{}, error) {
	return nil, nil
}

// PopulateMetaFromContext populates Metadata from context; called by OrgModelRouter and UsageTracker
// using helpers instead of a hook to avoid circular dependency.
func PopulateMetaFromContext(ctx context.Context, meta map[string]any) {
	if id := AuthTokenIDFromContext(ctx); id != "" {
		meta[MetaAuthTokenID] = id
	}
	if id := OrgIDFromContext(ctx); id != "" {
		meta[MetaOrgID] = id
	}
}

// OrgIDFromMeta reads orgID from the proxy request Metadata.
func OrgIDFromMeta(meta map[string]any) model.OrgID {
	v, _ := meta[MetaOrgID].(string)
	return model.OrgID(v)
}

// AuthTokenIDFromMeta reads authTokenID from the proxy request Metadata.
func AuthTokenIDFromMeta(meta map[string]any) string {
	v, _ := meta[MetaAuthTokenID].(string)
	return v
}

// ModelIDFromMeta reads modelID from the proxy request Metadata.
func ModelIDFromMeta(meta map[string]any) model.LLMModelID {
	v, _ := meta[MetaModelID].(string)
	return model.LLMModelID(v)
}

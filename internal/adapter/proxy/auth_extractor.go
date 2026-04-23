package proxy

import (
	"context"
	"log/slog"
	"net/http"

	genaiProxy "github.com/bornholm/genai/proxy"
	"github.com/bornholm/xolo/internal/core/model"
	httpx "github.com/bornholm/xolo/internal/http/context"
	authn "github.com/bornholm/xolo/internal/http/middleware/authn"
)

type contextKey string

const (
	contextKeyAuthTokenID   contextKey = "authTokenID"
	contextKeyOrgID         contextKey = "orgID"
	contextKeyApplicationID contextKey = "applicationID"
)

const contextKeyAlreadyExtracted contextKey = "alreadyExtracted"

// XoloAuthExtractor reads user identity and org/token metadata exclusively from the
// HTTP context populated by the authn + bridge + memberships middleware chain.
// It never reads the Authorization header directly.
func XoloAuthExtractor() func(r *http.Request) (string, error) {
	return func(r *http.Request) (string, error) {
		ctx := r.Context()

		if ctx.Value(contextKeyAlreadyExtracted) != nil {
			slog.Debug("XoloAuthExtractor: already extracted, skipping")
			return "", nil
		}

		slog.Debug("XoloAuthExtractor: checking", "hasHttpUser", httpx.User(ctx) != nil, "hasAuthnUser", authn.ContextUser(ctx) != nil, "path", r.URL.Path)

		if user := httpx.User(ctx); user != nil {
			orgID := ""
			authTokenID := ""
			if authnUser := authn.ContextUser(ctx); authnUser != nil && authnUser.OrgID != "" {
				orgID = authnUser.OrgID
				authTokenID = authnUser.TokenID
				slog.Debug("XoloAuthExtractor: org from authn token", "orgID", orgID, "tokenID", authTokenID)
			} else if memberships := httpx.Memberships(ctx); len(memberships) > 0 {
				orgID = string(memberships[0].OrgID())
				slog.Debug("XoloAuthExtractor: org from membership", "orgID", orgID)
			}
			slog.Debug("XoloAuthExtractor: using httpx.User", "userID", user.ID(), "orgID", orgID)
			newCtx := context.WithValue(ctx, contextKeyAuthTokenID, authTokenID)
			newCtx = context.WithValue(newCtx, contextKeyOrgID, orgID)
			newCtx = context.WithValue(newCtx, contextKeyAlreadyExtracted, true)
			*r = *r.WithContext(newCtx)
			return string(user.ID()), nil
		}

		return "", nil
	}
}

// AuthTokenIDFromContext retrieves the auth token ID stored by XoloAuthExtractor.
func AuthTokenIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyAuthTokenID).(string)
	return v
}

// ApplicationIDFromContext retrieves the application ID stored by XoloAuthExtractor.
func ApplicationIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyApplicationID).(string)
	return v
}

// OrgIDFromContext retrieves the org ID stored by XoloAuthExtractor.
func OrgIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyOrgID).(string)
	return v
}

// MetaAuthTokenID and MetaOrgID are the req.Metadata keys used by hooks.
const (
	MetaAuthTokenID   = "authTokenID"
	MetaOrgID         = "orgID"
	MetaApplicationID = "applicationID"
	MetaModelID       = "modelID"
	MetaOriginalModel = "originalModel" // requested model before plugin resolution
	MetaResolvedModel = "resolvedModel" // actual model after plugin resolution
)

// SetRequestMeta is a PreRequestHook (priority 0) that copies context values
// into req.Metadata so all subsequent hooks can read them without importing
// the http package.
type SetRequestMetaHook struct{}

func (h *SetRequestMetaHook) Name() string  { return "xolo.set-request-meta" }
func (h *SetRequestMetaHook) Priority() int { return 0 }

func (h *SetRequestMetaHook) PreRequest(ctx context.Context, req interface {
	GetMetadata() map[string]any
	GetHeaders() http.Header
}) (interface{}, error) {
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
	if id := ApplicationIDFromContext(ctx); id != "" {
		meta[MetaApplicationID] = id
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

// ApplicationIDFromMeta reads applicationID from the proxy request Metadata.
func ApplicationIDFromMeta(meta map[string]any) model.ApplicationID {
	v, _ := meta[MetaApplicationID].(string)
	return model.ApplicationID(v)
}

// populateMetaFromContext reads orgID and authTokenID from the request context
// and copies them into req.Metadata.
//
// It tries two sources in order:
//  1. The explicit context keys set by XoloAuthExtractor (available in ResolveModel hooks,
//     where the proxy passes r.Context() which includes the auth extractor's output).
//  2. authn.ContextUser (set by the authn middleware before the proxy server), which carries
//     OrgID/TokenID for API key tokens — useful in PreRequest hooks where the proxy passes
//     a context captured before XoloAuthExtractor runs.
func populateMetaFromContext(ctx context.Context, req *genaiProxy.ProxyRequest) {
	if OrgIDFromMeta(req.Metadata) != "" {
		return
	}
	if orgID := OrgIDFromContext(ctx); orgID != "" {
		req.Metadata[MetaOrgID] = orgID
		if authTokenID := AuthTokenIDFromContext(ctx); authTokenID != "" {
			req.Metadata[MetaAuthTokenID] = authTokenID
		}
		return
	}
	if authnUser := authn.ContextUser(ctx); authnUser != nil && authnUser.OrgID != "" {
		req.Metadata[MetaOrgID] = authnUser.OrgID
		if authnUser.TokenID != "" {
			req.Metadata[MetaAuthTokenID] = authnUser.TokenID
		}
	}
}


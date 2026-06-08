package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	genaiProxy "github.com/bornholm/genai/proxy"
	"github.com/bornholm/genai/llm"
)

// SessionIDHook is a PreRequestHook (priority 1) that injects a session ID
// into ChatOptions for sticky routing at the upstream provider (e.g. OpenRouter).
//
// Priority: if the client already sent an x-session-id header, it is forwarded
// as-is. Otherwise a stable opaque value is derived from userID + orgID via
// SHA-256 so that all requests from the same user/org land on the same cache node.
type SessionIDHook struct{}

func (h *SessionIDHook) Name() string  { return "xolo.session-id" }
func (h *SessionIDHook) Priority() int { return 1 }

func (h *SessionIDHook) PreRequest(ctx context.Context, req *genaiProxy.ProxyRequest) (*genaiProxy.HookResult, error) {
	if clientSession := req.Headers.Get("x-session-id"); clientSession != "" {
		req.ChatOptions = append(req.ChatOptions, llm.WithSessionID(clientSession))
		return nil, nil
	}

	populateMetaFromContext(ctx, req)
	userID := req.UserID
	orgID := string(OrgIDFromMeta(req.Metadata))
	if userID == "" || orgID == "" {
		return nil, nil
	}

	raw := sha256.Sum256([]byte(userID + ":" + orgID))
	sessionID := hex.EncodeToString(raw[:])[:24]
	req.ChatOptions = append(req.ChatOptions, llm.WithSessionID(sessionID))
	return nil, nil
}

var _ genaiProxy.PreRequestHook = &SessionIDHook{}

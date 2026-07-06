package proxy

import (
	"context"
	"strconv"

	genaiProxy "github.com/bornholm/genai/proxy"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

// XoloEventEmitterHook is a PostResponseHook that emits a "proxy.request" event
// for every successful proxy call, so the activity shows up in the event system.
type XoloEventEmitterHook struct {
	emitter port.EventEmitter
}

func NewXoloEventEmitterHook(emitter port.EventEmitter) *XoloEventEmitterHook {
	return &XoloEventEmitterHook{emitter: emitter}
}

func (h *XoloEventEmitterHook) Name() string  { return "xolo.event-emitter" }
func (h *XoloEventEmitterHook) Priority() int { return 110 }

// PostResponse implements proxy.PostResponseHook.
func (h *XoloEventEmitterHook) PostResponse(ctx context.Context, req *genaiProxy.ProxyRequest, res *genaiProxy.ProxyResponse) (*genaiProxy.HookResult, error) {
	if req.UserID == "" {
		return nil, nil
	}

	PopulateMetaFromContext(ctx, req.Metadata)
	orgID := OrgIDFromMeta(req.Metadata)

	attrs := map[string]string{
		"model": req.Model,
	}
	if id := AuthTokenIDFromMeta(req.Metadata); id != "" {
		attrs["auth_token_id"] = id
	}
	if res.TokensUsed != nil {
		attrs["prompt_tokens"] = strconv.Itoa(res.TokensUsed.PromptTokens)
		attrs["completion_tokens"] = strconv.Itoa(res.TokensUsed.CompletionTokens)
		attrs["total_tokens"] = strconv.Itoa(res.TokensUsed.PromptTokens + res.TokensUsed.CompletionTokens)
	}

	event := model.NewEvent(model.EventSourcePlatform, model.EventTypeProxyRequest,
		model.WithEventOrg(orgID),
		model.WithEventUser(model.UserID(req.UserID)),
		model.WithEventSeverity(model.SeverityInfo),
		model.WithEventMessage("Requête proxy: "+req.Model),
		model.WithEventAttributes(attrs),
	)
	h.emitter.Emit(ctx, event)

	return nil, nil
}

var _ genaiProxy.PostResponseHook = &XoloEventEmitterHook{}

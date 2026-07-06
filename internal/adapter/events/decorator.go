// Package events provides store decorators that emit platform events on
// create/update/delete operations. Wrapping the stores (rather than sprinkling
// emit calls across HTTP handlers) guarantees every caller — web UI, API,
// future callers — produces the same lifecycle events, and keeps the emission
// concern out of the transport layer.
package events

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
)

// emit records a lifecycle event for a user-initiated store mutation. It is a
// no-op when there is no acting user in context (system seeding, migrations,
// tests…), which keeps automated operations out of the event stream and lets us
// attribute the event to the acting user.
func emit(ctx context.Context, emitter port.EventEmitter, orgID model.OrgID, severity model.EventSeverity, typ, message string, attrs map[string]string) {
	if emitter == nil {
		return
	}
	user := httpCtx.User(ctx)
	if user == nil {
		return
	}
	if attrs == nil {
		attrs = map[string]string{}
	}
	attrs["actor"] = user.DisplayName()
	attrs["actor_id"] = string(user.ID())

	emitter.Emit(ctx, model.NewEvent(model.EventSourcePlatform, typ,
		model.WithEventOrg(orgID),
		model.WithEventSeverity(severity),
		model.WithEventMessage(message),
		model.WithEventAttributes(attrs),
	))
}

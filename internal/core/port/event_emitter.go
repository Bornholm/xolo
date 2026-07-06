package port

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

// EventEmitter accepts events for asynchronous recording. Implementations must
// never block the caller (the request/proxy path); on overflow they drop and
// log rather than wait.
type EventEmitter interface {
	Emit(ctx context.Context, event model.Event)
}

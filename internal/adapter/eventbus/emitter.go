// Package eventbus provides an asynchronous, non-blocking implementation of
// port.EventEmitter backed by a bounded channel and a worker goroutine that
// persists events to a port.EventStore.
package eventbus

import (
	"context"
	"log/slog"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/pkg/errors"
)

const defaultBufferSize = 1024

// AsyncEmitter buffers events in a channel and records them from a background
// worker. Emit never blocks: when the buffer is full the event is dropped and a
// warning is logged, so the request/proxy path is never slowed down.
type AsyncEmitter struct {
	store port.EventStore
	ch    chan model.Event
}

// NewAsyncEmitter creates an emitter writing to the given store. A bufferSize
// <= 0 falls back to the default.
func NewAsyncEmitter(store port.EventStore, bufferSize int) *AsyncEmitter {
	if bufferSize <= 0 {
		bufferSize = defaultBufferSize
	}
	return &AsyncEmitter{
		store: store,
		ch:    make(chan model.Event, bufferSize),
	}
}

// Emit implements port.EventEmitter. It is non-blocking.
func (e *AsyncEmitter) Emit(ctx context.Context, event model.Event) {
	select {
	case e.ch <- event:
	default:
		slog.WarnContext(ctx, "event emitter buffer full, dropping event",
			slog.String("type", event.Type()),
			slog.String("source", event.Source()))
	}
}

// Run consumes buffered events and persists them until ctx is cancelled. It is
// meant to be started in a dedicated goroutine.
func (e *AsyncEmitter) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-e.ch:
			recordCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := e.store.RecordEvent(recordCtx, event); err != nil {
				slog.ErrorContext(ctx, "could not record event",
					slog.Any("error", errors.WithStack(err)),
					slog.String("type", event.Type()))
			}
			cancel()
		}
	}
}

var _ port.EventEmitter = &AsyncEmitter{}

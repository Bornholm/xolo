package events

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

// MiddlewareStore decorates a port.MiddlewareStore, emitting events on
// middleware mutations.
type MiddlewareStore struct {
	port.MiddlewareStore
	emitter port.EventEmitter
}

func NewMiddlewareStore(backend port.MiddlewareStore, emitter port.EventEmitter) *MiddlewareStore {
	return &MiddlewareStore{MiddlewareStore: backend, emitter: emitter}
}

func (s *MiddlewareStore) CreateMiddleware(ctx context.Context, m model.Middleware) error {
	if err := s.MiddlewareStore.CreateMiddleware(ctx, m); err != nil {
		return err
	}
	emit(ctx, s.emitter, m.OrgID(), model.SeverityInfo, model.EventTypeMiddlewareCreated,
		"Middleware créé : "+m.Name(),
		map[string]string{"middleware_id": string(m.ID()), "middleware_name": m.Name()})
	return nil
}

func (s *MiddlewareStore) SaveMiddleware(ctx context.Context, m model.Middleware) error {
	if err := s.MiddlewareStore.SaveMiddleware(ctx, m); err != nil {
		return err
	}
	emit(ctx, s.emitter, m.OrgID(), model.SeverityInfo, model.EventTypeMiddlewareUpdated,
		"Middleware modifié : "+m.Name(),
		map[string]string{"middleware_id": string(m.ID()), "middleware_name": m.Name()})
	return nil
}

func (s *MiddlewareStore) DeleteMiddleware(ctx context.Context, id model.MiddlewareID) error {
	existing, _ := s.MiddlewareStore.GetMiddlewareByID(ctx, id)
	if err := s.MiddlewareStore.DeleteMiddleware(ctx, id); err != nil {
		return err
	}
	orgID := model.OrgID("")
	attrs := map[string]string{"middleware_id": string(id)}
	msg := "Middleware supprimé"
	if existing != nil {
		orgID = existing.OrgID()
		attrs["middleware_name"] = existing.Name()
		msg = "Middleware supprimé : " + existing.Name()
	}
	emit(ctx, s.emitter, orgID, model.SeverityWarning, model.EventTypeMiddlewareDeleted, msg, attrs)
	return nil
}

var _ port.MiddlewareStore = &MiddlewareStore{}

package port

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

type MiddlewareStore interface {
	CreateMiddleware(ctx context.Context, m model.Middleware) error
	GetMiddlewareByID(ctx context.Context, id model.MiddlewareID) (model.Middleware, error)
	ListMiddlewares(ctx context.Context, orgID model.OrgID) ([]model.Middleware, error)
	// ListEnabledMiddlewares returns the org's enabled middlewares, ordered by
	// ascending Priority (outermost first).
	ListEnabledMiddlewares(ctx context.Context, orgID model.OrgID) ([]model.Middleware, error)
	SaveMiddleware(ctx context.Context, m model.Middleware) error
	DeleteMiddleware(ctx context.Context, id model.MiddlewareID) error
}

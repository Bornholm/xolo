package port

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

type ApplicationStore interface {
	CreateApplication(ctx context.Context, app model.Application) error
	GetApplication(ctx context.Context, appID model.ApplicationID) (model.Application, error)
	QueryApplications(ctx context.Context, orgID model.OrgID) ([]model.Application, error)
	UpdateApplication(ctx context.Context, app model.Application) error
	DeleteApplication(ctx context.Context, appID model.ApplicationID) error

	FindApplicationAuthToken(ctx context.Context, token string) (model.AuthToken, error)
	GetApplicationAuthTokens(ctx context.Context, appID model.ApplicationID) ([]model.AuthToken, error)
	CreateApplicationAuthToken(ctx context.Context, token model.AuthToken) error
	DeleteApplicationAuthToken(ctx context.Context, tokenID model.AuthTokenID) error
}

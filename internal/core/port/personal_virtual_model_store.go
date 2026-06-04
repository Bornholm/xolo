package port

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

type PersonalVirtualModelStore interface {
	CreatePersonalVirtualModel(ctx context.Context, vm model.PersonalVirtualModel) error
	GetPersonalVirtualModelByID(ctx context.Context, id model.PersonalVirtualModelID) (model.PersonalVirtualModel, error)
	GetPersonalVirtualModelByName(ctx context.Context, userID model.UserID, name string) (model.PersonalVirtualModel, error)
	ListPersonalVirtualModels(ctx context.Context, userID model.UserID) ([]model.PersonalVirtualModel, error)
	SavePersonalVirtualModel(ctx context.Context, vm model.PersonalVirtualModel) error
	DeletePersonalVirtualModel(ctx context.Context, id model.PersonalVirtualModelID) error
}

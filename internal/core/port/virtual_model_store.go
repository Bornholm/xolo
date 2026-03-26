package port

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

type VirtualModelStore interface {
	CreateVirtualModel(ctx context.Context, vm model.VirtualModel) error
	GetVirtualModelByID(ctx context.Context, id model.VirtualModelID) (model.VirtualModel, error)
	GetVirtualModelByName(ctx context.Context, orgID model.OrgID, name string) (model.VirtualModel, error)
	ListVirtualModels(ctx context.Context, orgID model.OrgID) ([]model.VirtualModel, error)
	SaveVirtualModel(ctx context.Context, vm model.VirtualModel) error
	DeleteVirtualModel(ctx context.Context, id model.VirtualModelID) error
}

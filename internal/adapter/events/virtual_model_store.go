package events

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

// VirtualModelStore decorates a port.VirtualModelStore, emitting events on
// virtual model mutations.
type VirtualModelStore struct {
	port.VirtualModelStore
	emitter port.EventEmitter
}

func NewVirtualModelStore(backend port.VirtualModelStore, emitter port.EventEmitter) *VirtualModelStore {
	return &VirtualModelStore{VirtualModelStore: backend, emitter: emitter}
}

func (s *VirtualModelStore) CreateVirtualModel(ctx context.Context, vm model.VirtualModel) error {
	if err := s.VirtualModelStore.CreateVirtualModel(ctx, vm); err != nil {
		return err
	}
	emit(ctx, s.emitter, vm.OrgID(), model.SeverityInfo, model.EventTypeVirtualModelCreated,
		"Modèle virtuel créé : "+vm.Name(),
		map[string]string{"virtual_model_id": string(vm.ID()), "virtual_model_name": vm.Name()})
	return nil
}

func (s *VirtualModelStore) SaveVirtualModel(ctx context.Context, vm model.VirtualModel) error {
	if err := s.VirtualModelStore.SaveVirtualModel(ctx, vm); err != nil {
		return err
	}
	emit(ctx, s.emitter, vm.OrgID(), model.SeverityInfo, model.EventTypeVirtualModelUpdated,
		"Modèle virtuel modifié : "+vm.Name(),
		map[string]string{"virtual_model_id": string(vm.ID()), "virtual_model_name": vm.Name()})
	return nil
}

func (s *VirtualModelStore) DeleteVirtualModel(ctx context.Context, id model.VirtualModelID) error {
	existing, _ := s.VirtualModelStore.GetVirtualModelByID(ctx, id)
	if err := s.VirtualModelStore.DeleteVirtualModel(ctx, id); err != nil {
		return err
	}
	orgID := model.OrgID("")
	attrs := map[string]string{"virtual_model_id": string(id)}
	msg := "Modèle virtuel supprimé"
	if existing != nil {
		orgID = existing.OrgID()
		attrs["virtual_model_name"] = existing.Name()
		msg = "Modèle virtuel supprimé : " + existing.Name()
	}
	emit(ctx, s.emitter, orgID, model.SeverityWarning, model.EventTypeVirtualModelDeleted, msg, attrs)
	return nil
}

var _ port.VirtualModelStore = &VirtualModelStore{}

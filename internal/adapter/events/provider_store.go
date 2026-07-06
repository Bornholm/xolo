package events

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

// ProviderStore decorates a port.ProviderStore, emitting events on provider and
// LLM model mutations. Read methods and any other interface methods are promoted
// from the embedded backend.
type ProviderStore struct {
	port.ProviderStore
	emitter port.EventEmitter
}

func NewProviderStore(backend port.ProviderStore, emitter port.EventEmitter) *ProviderStore {
	return &ProviderStore{ProviderStore: backend, emitter: emitter}
}

func (s *ProviderStore) CreateProvider(ctx context.Context, p model.Provider) error {
	if err := s.ProviderStore.CreateProvider(ctx, p); err != nil {
		return err
	}
	emit(ctx, s.emitter, p.OrgID(), model.SeverityInfo, model.EventTypeProviderCreated,
		"Fournisseur créé : "+p.Name(),
		map[string]string{"provider_id": string(p.ID()), "provider_name": p.Name()})
	return nil
}

func (s *ProviderStore) SaveProvider(ctx context.Context, p model.Provider) error {
	if err := s.ProviderStore.SaveProvider(ctx, p); err != nil {
		return err
	}
	emit(ctx, s.emitter, p.OrgID(), model.SeverityInfo, model.EventTypeProviderUpdated,
		"Fournisseur modifié : "+p.Name(),
		map[string]string{"provider_id": string(p.ID()), "provider_name": p.Name()})
	return nil
}

func (s *ProviderStore) DeleteProvider(ctx context.Context, id model.ProviderID) error {
	existing, _ := s.ProviderStore.GetProviderByID(ctx, id)
	if err := s.ProviderStore.DeleteProvider(ctx, id); err != nil {
		return err
	}
	orgID := model.OrgID("")
	attrs := map[string]string{"provider_id": string(id)}
	msg := "Fournisseur supprimé"
	if existing != nil {
		orgID = existing.OrgID()
		attrs["provider_name"] = existing.Name()
		msg = "Fournisseur supprimé : " + existing.Name()
	}
	emit(ctx, s.emitter, orgID, model.SeverityWarning, model.EventTypeProviderDeleted, msg, attrs)
	return nil
}

func (s *ProviderStore) CreateLLMModel(ctx context.Context, m model.LLMModel) error {
	if err := s.ProviderStore.CreateLLMModel(ctx, m); err != nil {
		return err
	}
	emit(ctx, s.emitter, m.OrgID(), model.SeverityInfo, model.EventTypeModelCreated,
		"Modèle créé : "+m.ProxyName(),
		map[string]string{"model_id": string(m.ID()), "model_name": m.ProxyName()})
	return nil
}

func (s *ProviderStore) SaveLLMModel(ctx context.Context, m model.LLMModel) error {
	if err := s.ProviderStore.SaveLLMModel(ctx, m); err != nil {
		return err
	}
	emit(ctx, s.emitter, m.OrgID(), model.SeverityInfo, model.EventTypeModelUpdated,
		"Modèle modifié : "+m.ProxyName(),
		map[string]string{"model_id": string(m.ID()), "model_name": m.ProxyName()})
	return nil
}

func (s *ProviderStore) DeleteLLMModel(ctx context.Context, id model.LLMModelID) error {
	existing, _ := s.ProviderStore.GetLLMModelByID(ctx, id)
	if err := s.ProviderStore.DeleteLLMModel(ctx, id); err != nil {
		return err
	}
	orgID := model.OrgID("")
	attrs := map[string]string{"model_id": string(id)}
	msg := "Modèle supprimé"
	if existing != nil {
		orgID = existing.OrgID()
		attrs["model_name"] = existing.ProxyName()
		msg = "Modèle supprimé : " + existing.ProxyName()
	}
	emit(ctx, s.emitter, orgID, model.SeverityWarning, model.EventTypeModelDeleted, msg, attrs)
	return nil
}

var _ port.ProviderStore = &ProviderStore{}

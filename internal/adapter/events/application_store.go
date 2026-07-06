package events

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

// ApplicationStore decorates a port.ApplicationStore, emitting events on
// application mutations.
type ApplicationStore struct {
	port.ApplicationStore
	emitter port.EventEmitter
}

func NewApplicationStore(backend port.ApplicationStore, emitter port.EventEmitter) *ApplicationStore {
	return &ApplicationStore{ApplicationStore: backend, emitter: emitter}
}

func (s *ApplicationStore) CreateApplication(ctx context.Context, app model.Application) error {
	if err := s.ApplicationStore.CreateApplication(ctx, app); err != nil {
		return err
	}
	emit(ctx, s.emitter, app.OrgID(), model.SeverityInfo, model.EventTypeApplicationCreated,
		"Application créée : "+app.Name(),
		map[string]string{"application_id": string(app.ID()), "application_name": app.Name()})
	return nil
}

func (s *ApplicationStore) UpdateApplication(ctx context.Context, app model.Application) error {
	if err := s.ApplicationStore.UpdateApplication(ctx, app); err != nil {
		return err
	}
	emit(ctx, s.emitter, app.OrgID(), model.SeverityInfo, model.EventTypeApplicationUpdated,
		"Application modifiée : "+app.Name(),
		map[string]string{"application_id": string(app.ID()), "application_name": app.Name()})
	return nil
}

func (s *ApplicationStore) DeleteApplication(ctx context.Context, appID model.ApplicationID) error {
	existing, _ := s.ApplicationStore.GetApplication(ctx, appID)
	if err := s.ApplicationStore.DeleteApplication(ctx, appID); err != nil {
		return err
	}
	orgID := model.OrgID("")
	attrs := map[string]string{"application_id": string(appID)}
	msg := "Application supprimée"
	if existing != nil {
		orgID = existing.OrgID()
		attrs["application_name"] = existing.Name()
		msg = "Application supprimée : " + existing.Name()
	}
	emit(ctx, s.emitter, orgID, model.SeverityWarning, model.EventTypeApplicationDeleted, msg, attrs)
	return nil
}

func tokenAttrs(token model.AuthToken) map[string]string {
	attrs := map[string]string{
		"token_id": string(token.ID()),
		"label":    token.Label(),
	}
	if app := token.Application(); app != nil {
		attrs["application_id"] = string(app.ID())
		attrs["application_name"] = app.Name()
	}
	return attrs
}

func (s *ApplicationStore) CreateApplicationAuthToken(ctx context.Context, token model.AuthToken) error {
	if err := s.ApplicationStore.CreateApplicationAuthToken(ctx, token); err != nil {
		return err
	}
	emit(ctx, s.emitter, token.OrgID(), model.SeverityInfo, model.EventTypeApplicationTokenCreated,
		"Jeton d'application créé", tokenAttrs(token))
	return nil
}

func (s *ApplicationStore) DeleteApplicationAuthToken(ctx context.Context, tokenID model.AuthTokenID) error {
	existing, _ := s.ApplicationStore.GetApplicationAuthToken(ctx, tokenID)
	if err := s.ApplicationStore.DeleteApplicationAuthToken(ctx, tokenID); err != nil {
		return err
	}
	orgID := model.OrgID("")
	attrs := map[string]string{"token_id": string(tokenID)}
	if existing != nil {
		orgID = existing.OrgID()
		attrs = tokenAttrs(existing)
	}
	emit(ctx, s.emitter, orgID, model.SeverityWarning, model.EventTypeApplicationTokenDeleted,
		"Jeton d'application supprimé", attrs)
	return nil
}

var _ port.ApplicationStore = &ApplicationStore{}

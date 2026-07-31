package main

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	gormadapter "github.com/bornholm/xolo/internal/adapter/gorm"
	"github.com/bornholm/xolo/internal/core/model"
)

func (s *seeder) seedEvents(ctx context.Context) error {
	if err := s.seedEventSettings(ctx); err != nil {
		return errors.WithStack(err)
	}
	if err := s.seedEventRecords(ctx); err != nil {
		return errors.WithStack(err)
	}
	return s.seedAlerts(ctx)
}

func (s *seeder) seedEventSettings(ctx context.Context) error {
	maxEvents := 5000

	return s.create(&gormadapter.EventSettings{
		OrgID:     orgAcme,
		MaxEvents: &maxEvents,
		UpdatedAt: now.AddDate(0, 0, -3),
	})
}

func (s *seeder) seedEventRecords(ctx context.Context) error {
	incidentID := "inc-acme-auth"

	// Fixed, meaningful events: they are the ones E2E tests assert on.
	fixed := []*gormadapter.Event{
		{
			ID: "evt-acme-provider-created", CreatedAt: now.AddDate(0, 0, -10),
			OrgID: orgAcme, UserID: userAlice, Source: model.EventSourcePlatform,
			Type: model.EventTypeProviderCreated, Severity: string(model.SeverityInfo),
			Message: "Fournisseur « OpenAI » créé",
			Attributes: gormadapter.JSONColumn[map[string]string]{Val: &map[string]string{
				"provider_id": providerAcmeOpenAI,
			}},
		},
		{
			ID: "evt-acme-model-created", CreatedAt: now.AddDate(0, 0, -10).Add(5 * time.Minute),
			OrgID: orgAcme, UserID: userAlice, Source: model.EventSourcePlatform,
			Type: model.EventTypeModelCreated, Severity: string(model.SeverityInfo),
			Message: "Modèle « gpt-4o » ajouté",
			Attributes: gormadapter.JSONColumn[map[string]string]{Val: &map[string]string{
				"model_id": modelAcmeGPT4o,
			}},
		},
		{
			ID: "evt-acme-invite", CreatedAt: now.AddDate(0, 0, -3),
			OrgID: orgAcme, UserID: userAlice, Source: model.EventSourcePlatform,
			Type: model.EventTypeInviteCreated, Severity: string(model.SeverityInfo),
			Message: "Invitation créée pour grace@acme.test",
			Attributes: gormadapter.JSONColumn[map[string]string]{Val: &map[string]string{
				"invite_id": "inv-acme-grace",
			}},
		},
		{
			// Pinned + attached to an incident: exercises both flags at once.
			ID: "evt-acme-quota-exceeded", CreatedAt: now.Add(-2 * time.Hour),
			OrgID: orgAcme, UserID: userDave, Source: model.EventSourcePlatform,
			Type: model.EventTypeProxyRequest, Severity: string(model.SeverityError),
			Message: "Quota journalier dépassé pour dave@acme.test",
			Attributes: gormadapter.JSONColumn[map[string]string]{Val: &map[string]string{
				"quota_id": "qta-user-dave",
				"scope":    string(model.QuotaScopeUser),
			}},
			Pinned:     true,
			IncidentID: incidentID,
		},
	}

	for _, event := range fixed {
		if err := s.create(event); err != nil {
			return errors.WithStack(err)
		}
	}

	// A burst of failed logins over the last hours, feeding the "auth failures"
	// alert below.
	var batch []*gormadapter.Event
	for i := 0; i < 25; i++ {
		batch = append(batch, &gormadapter.Event{
			ID:        fmt.Sprintf("evt-acme-login-%03d", i),
			CreatedAt: now.Add(-time.Duration(s.rand.Intn(6*3600)) * time.Second),
			OrgID:     orgAcme, UserID: userBob, Source: model.EventSourcePlatform,
			Type: model.EventTypeAuthLoginFailed, Severity: string(model.SeverityWarning),
			Message: "Échec d'authentification : identifiants invalides",
			Attributes: gormadapter.JSONColumn[map[string]string]{Val: &map[string]string{
				"provider": "local",
			}},
			IncidentID: incidentID,
		})
	}

	// Routine proxy traffic on both orgs, so the event stream is not empty.
	for i := 0; i < 60; i++ {
		orgID, userID := orgAcme, userCarol
		if i%3 == 0 {
			orgID, userID = orgGlobex, userErin
		}
		batch = append(batch, &gormadapter.Event{
			ID:        fmt.Sprintf("evt-proxy-%03d", i),
			CreatedAt: now.Add(-time.Duration(s.rand.Intn(72*3600)) * time.Second),
			OrgID:     orgID, UserID: userID, Source: model.EventSourcePlatform,
			Type: model.EventTypeProxyRequest, Severity: string(model.SeverityInfo),
			Message: "Requête de complétion traitée",
		})
	}

	if err := s.db.Create(&batch).Error; err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s *seeder) seedAlerts(ctx context.Context) error {
	pendingSince := now.Add(-30 * time.Minute)
	lastEval := now.Add(-time.Minute)

	alerts := []*gormadapter.Alert{
		{
			// Currently firing: an open incident is attached below.
			ID: "alr-acme-auth-failures", OrgID: orgAcme, OwnerID: userAlice,
			Scope: string(model.AlertScopeOrg),
			Name:  "Échecs d'authentification", Description: "Trop d'échecs de connexion sur une heure.",
			Query: `{type="auth.login.failed"}`, Aggregation: string(model.AggregationCount),
			WindowSeconds: 3600, Comparator: string(model.ComparatorGT), Threshold: 10, ForSeconds: 300,
			Enabled: true, State: string(model.AlertStateFiring), PendingSince: &pendingSince,
			LastEvaluatedAt: &lastEval,
			CreatedAt:       now.AddDate(0, 0, -20), UpdatedAt: lastEval,
		},
		{
			ID: "alr-acme-errors", OrgID: orgAcme, OwnerID: userAlice,
			Scope: string(model.AlertScopeOrg),
			Name:  "Erreurs proxy", Description: "Remontée des erreurs de la passerelle.",
			Query: `{severity="error"}`, Aggregation: string(model.AggregationCount),
			WindowSeconds: 900, Comparator: string(model.ComparatorGTE), Threshold: 5, ForSeconds: 0,
			Enabled: true, State: string(model.AlertStateOK), LastEvaluatedAt: &lastEval,
			CreatedAt: now.AddDate(0, 0, -18), UpdatedAt: lastEval,
		},
		{
			// Personal alert owned by a simple member.
			ID: "alr-carol-personal", OrgID: orgAcme, OwnerID: userCarol,
			Scope: string(model.AlertScopePersonal),
			Name:  "Mes requêtes en erreur", Description: "Alerte personnelle sur mes propres événements.",
			Query: `{type="proxy.request", severity="error"}`, Aggregation: string(model.AggregationCount),
			WindowSeconds: 1800, Comparator: string(model.ComparatorGT), Threshold: 3, ForSeconds: 0,
			Enabled: false, State: string(model.AlertStateOK),
			CreatedAt: now.AddDate(0, 0, -6), UpdatedAt: now.AddDate(0, 0, -6),
		},
	}

	for _, alert := range alerts {
		if err := s.create(alert); err != nil {
			return errors.WithStack(err)
		}
	}

	resolved := now.AddDate(0, 0, -8)
	incidents := []*gormadapter.AlertIncident{
		{
			ID: "inc-acme-auth", AlertID: "alr-acme-auth-failures", OrgID: orgAcme,
			Status: model.IncidentStatusFiring, StartedAt: now.Add(-25 * time.Minute), PeakValue: 25,
		},
		{
			ID: "inc-acme-auth-past", AlertID: "alr-acme-auth-failures", OrgID: orgAcme,
			Status: model.IncidentStatusResolved, StartedAt: now.AddDate(0, 0, -9),
			ResolvedAt: &resolved, PeakValue: 14,
		},
	}

	for _, incident := range incidents {
		if err := s.create(incident); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

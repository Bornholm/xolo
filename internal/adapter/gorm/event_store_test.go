package gorm_test

import (
	"context"
	"testing"

	"github.com/bornholm/xolo/internal/core/eventql"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

func ptr[T any](v T) *T { return &v }

func TestEventStore_VisibilityAndQuery(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	org := model.OrgID("org1")

	events := []*model.BaseEvent{
		model.NewEvent(model.EventSourcePlatform, model.EventTypeAuthLoginFailed,
			model.WithEventOrg(org), model.WithEventUser("u1"),
			model.WithEventSeverity(model.SeverityWarning),
			model.WithEventMessage("login failed for u1"),
			model.WithEventAttribute("provider", "oidc")),
		model.NewEvent(model.EventSourcePlatform, model.EventTypeProxyRequest,
			model.WithEventOrg(org), model.WithEventUser("u2"),
			model.WithEventAttribute("provider", "openai")),
		model.NewEvent(model.EventSourcePlatform, model.EventTypeProviderCreated,
			model.WithEventOrg(org)), // global (no user)
	}
	for _, e := range events {
		if err := store.RecordEvent(ctx, e); err != nil {
			t.Fatalf("RecordEvent: %v", err)
		}
	}

	// Lambda u1: only own events.
	own, err := store.QueryEvents(ctx, port.EventFilter{OrgID: &org, UserID: ptr(model.UserID("u1"))})
	if err != nil {
		t.Fatalf("QueryEvents own: %v", err)
	}
	if len(own) != 1 || own[0].UserID() != "u1" {
		t.Fatalf("expected 1 own event for u1, got %d", len(own))
	}

	// u1 + global.
	ownGlobal, err := store.QueryEvents(ctx, port.EventFilter{OrgID: &org, UserID: ptr(model.UserID("u1")), IncludeGlobal: true})
	if err != nil {
		t.Fatalf("QueryEvents own+global: %v", err)
	}
	if len(ownGlobal) != 2 {
		t.Fatalf("expected 2 events (own+global), got %d", len(ownGlobal))
	}

	// All users.
	all, err := store.QueryEvents(ctx, port.EventFilter{OrgID: &org, AllUsers: true})
	if err != nil {
		t.Fatalf("QueryEvents all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 events (all), got %d", len(all))
	}

	// Pushdown label filter.
	q, _ := eventql.Compile(`{type="auth.login.failed"}`)
	filtered, err := store.QueryEvents(ctx, port.EventFilter{OrgID: &org, AllUsers: true, Query: q})
	if err != nil {
		t.Fatalf("QueryEvents label: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 auth.login.failed, got %d", len(filtered))
	}

	// In-memory attribute filter.
	qa, _ := eventql.Compile(`{} | provider="openai"`)
	attrFiltered, err := store.QueryEvents(ctx, port.EventFilter{OrgID: &org, AllUsers: true, Query: qa})
	if err != nil {
		t.Fatalf("QueryEvents attr: %v", err)
	}
	if len(attrFiltered) != 1 || attrFiltered[0].Type() != model.EventTypeProxyRequest {
		t.Fatalf("expected 1 openai proxy event, got %d", len(attrFiltered))
	}
}

func TestEventStore_EvictOverflow(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	org := model.OrgID("org1")

	var pinnedID model.EventID
	for i := range 10 {
		e := model.NewEvent(model.EventSourcePlatform, model.EventTypeProxyRequest, model.WithEventOrg(org))
		if err := store.RecordEvent(ctx, e); err != nil {
			t.Fatalf("RecordEvent: %v", err)
		}
		if i == 0 {
			pinnedID = e.ID()
		}
	}

	// Pin the oldest event to an incident so it survives eviction.
	if err := store.PinEvents(ctx, []model.EventID{pinnedID}, "incident1"); err != nil {
		t.Fatalf("PinEvents: %v", err)
	}

	count, err := store.CountEvents(ctx, org)
	if err != nil {
		t.Fatalf("CountEvents: %v", err)
	}
	if count != 9 { // pinned events are not counted
		t.Fatalf("expected 9 non-pinned events, got %d", count)
	}

	deleted, err := store.EvictOverflow(ctx, org, 3)
	if err != nil {
		t.Fatalf("EvictOverflow: %v", err)
	}
	if deleted != 6 { // 9 non-pinned, keep 3, delete 6
		t.Fatalf("expected 6 deleted, got %d", deleted)
	}

	// Pinned event must still be there.
	all, err := store.QueryEvents(ctx, port.EventFilter{OrgID: &org, AllUsers: true})
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if len(all) != 4 { // 3 kept + 1 pinned
		t.Fatalf("expected 4 remaining events, got %d", len(all))
	}
	found := false
	for _, e := range all {
		if e.ID() == pinnedID {
			found = true
			if !e.Pinned() || e.IncidentID() != "incident1" {
				t.Fatalf("pinned event not properly marked")
			}
		}
	}
	if !found {
		t.Fatal("pinned event was evicted")
	}
}

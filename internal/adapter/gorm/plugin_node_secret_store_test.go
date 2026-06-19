package gorm_test

import (
	"context"
	"testing"

	xologorm "github.com/bornholm/xolo/internal/adapter/gorm"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	gormpkg "gorm.io/gorm"
)

func newTestStore(t *testing.T) *xologorm.Store {
	t.Helper()
	db, err := gormpkg.Open(gormlite.Open(":memory:"), &gormpkg.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return xologorm.NewStore(db)
}

func TestPluginNodeSecretStore_SetGetDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, found, err := store.GetSecret(ctx, "org-1", "mcp-bridge", "node-1", "authValue")
	if err != nil {
		t.Fatalf("GetSecret (missing): %v", err)
	}
	if found {
		t.Fatal("expected secret not to be found before SetSecret")
	}

	if err := store.SetSecret(ctx, "org-1", "mcp-bridge", "node-1", "authValue", "encrypted-payload-1"); err != nil {
		t.Fatalf("SetSecret: %v", err)
	}

	value, found, err := store.GetSecret(ctx, "org-1", "mcp-bridge", "node-1", "authValue")
	if err != nil {
		t.Fatalf("GetSecret: %v", err)
	}
	if !found {
		t.Fatal("expected secret to be found after SetSecret")
	}
	if value != "encrypted-payload-1" {
		t.Errorf("expected %q, got %q", "encrypted-payload-1", value)
	}

	// Overwriting must update in place, not create a duplicate row.
	if err := store.SetSecret(ctx, "org-1", "mcp-bridge", "node-1", "authValue", "encrypted-payload-2"); err != nil {
		t.Fatalf("SetSecret (overwrite): %v", err)
	}
	value, found, err = store.GetSecret(ctx, "org-1", "mcp-bridge", "node-1", "authValue")
	if err != nil {
		t.Fatalf("GetSecret (after overwrite): %v", err)
	}
	if !found || value != "encrypted-payload-2" {
		t.Errorf("expected overwritten value %q, got %q (found=%v)", "encrypted-payload-2", value, found)
	}

	// A different node instance has its own isolated namespace.
	_, found, err = store.GetSecret(ctx, "org-1", "mcp-bridge", "node-2", "authValue")
	if err != nil {
		t.Fatalf("GetSecret (other node): %v", err)
	}
	if found {
		t.Error("expected secret to be isolated per node instance")
	}

	if err := store.DeleteSecret(ctx, "org-1", "mcp-bridge", "node-1", "authValue"); err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}
	_, found, err = store.GetSecret(ctx, "org-1", "mcp-bridge", "node-1", "authValue")
	if err != nil {
		t.Fatalf("GetSecret (after delete): %v", err)
	}
	if found {
		t.Error("expected secret to be gone after DeleteSecret")
	}
}

func TestPluginNodeSecretStore_DeleteAllForNode(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.SetSecret(ctx, "org-1", "mcp-bridge", "node-1", "authValue", "secret-a"); err != nil {
		t.Fatalf("SetSecret: %v", err)
	}
	if err := store.SetSecret(ctx, "org-1", "mcp-bridge", "node-1", "otherKey", "secret-b"); err != nil {
		t.Fatalf("SetSecret: %v", err)
	}
	if err := store.SetSecret(ctx, "org-1", "mcp-bridge", "node-2", "authValue", "secret-c"); err != nil {
		t.Fatalf("SetSecret: %v", err)
	}

	if err := store.DeleteAllForNode(ctx, "node-1"); err != nil {
		t.Fatalf("DeleteAllForNode: %v", err)
	}

	if _, found, _ := store.GetSecret(ctx, "org-1", "mcp-bridge", "node-1", "authValue"); found {
		t.Error("expected node-1's authValue secret to be gone")
	}
	if _, found, _ := store.GetSecret(ctx, "org-1", "mcp-bridge", "node-1", "otherKey"); found {
		t.Error("expected node-1's otherKey secret to be gone")
	}
	if _, found, _ := store.GetSecret(ctx, "org-1", "mcp-bridge", "node-2", "authValue"); !found {
		t.Error("expected node-2's secret to be untouched")
	}
}

package secretcleanup_test

import (
	"context"
	"testing"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/secretcleanup"
)

type fakeSecretStore struct {
	deletedNodes []string
}

func (s *fakeSecretStore) GetSecret(ctx context.Context, orgID, pluginName, nodeID, key string) (string, bool, error) {
	return "", false, nil
}

func (s *fakeSecretStore) SetSecret(ctx context.Context, orgID, pluginName, nodeID, key, value string) error {
	return nil
}

func (s *fakeSecretStore) DeleteSecret(ctx context.Context, orgID, pluginName, nodeID, key string) error {
	return nil
}

func (s *fakeSecretStore) DeleteAllForNode(ctx context.Context, nodeID string) error {
	s.deletedNodes = append(s.deletedNodes, nodeID)
	return nil
}

func graphWithNodes(ids ...string) *model.PipelineGraph {
	g := &model.PipelineGraph{}
	for _, id := range ids {
		g.Nodes = append(g.Nodes, model.PipelineNode{ID: id})
	}
	return g
}

func TestPruneRemovedNodes_DeletesOnlyRemovedNodes(t *testing.T) {
	store := &fakeSecretStore{}
	oldGraph := graphWithNodes("node-1", "node-2", "node-3")
	newGraph := graphWithNodes("node-1", "node-3")

	if err := secretcleanup.PruneRemovedNodes(context.Background(), store, oldGraph, newGraph); err != nil {
		t.Fatalf("PruneRemovedNodes: %v", err)
	}

	if len(store.deletedNodes) != 1 || store.deletedNodes[0] != "node-2" {
		t.Errorf("expected only node-2 to be pruned, got %v", store.deletedNodes)
	}
}

func TestPruneRemovedNodes_NilNewGraph_PrunesAllOldNodes(t *testing.T) {
	store := &fakeSecretStore{}
	oldGraph := graphWithNodes("node-1", "node-2")

	if err := secretcleanup.PruneRemovedNodes(context.Background(), store, oldGraph, nil); err != nil {
		t.Fatalf("PruneRemovedNodes: %v", err)
	}

	if len(store.deletedNodes) != 2 {
		t.Errorf("expected both nodes to be pruned when the whole virtual model is deleted, got %v", store.deletedNodes)
	}
}

func TestPruneRemovedNodes_NoChange_PrunesNothing(t *testing.T) {
	store := &fakeSecretStore{}
	graph := graphWithNodes("node-1", "node-2")

	if err := secretcleanup.PruneRemovedNodes(context.Background(), store, graph, graph); err != nil {
		t.Fatalf("PruneRemovedNodes: %v", err)
	}

	if len(store.deletedNodes) != 0 {
		t.Errorf("expected no pruning when the graph is unchanged, got %v", store.deletedNodes)
	}
}

func TestPruneRemovedNodes_NilOldGraph_NoOp(t *testing.T) {
	store := &fakeSecretStore{}

	if err := secretcleanup.PruneRemovedNodes(context.Background(), store, nil, graphWithNodes("node-1")); err != nil {
		t.Fatalf("PruneRemovedNodes: %v", err)
	}

	if len(store.deletedNodes) != 0 {
		t.Errorf("expected no pruning when there is no old graph, got %v", store.deletedNodes)
	}
}

// Package secretcleanup removes plugin node secrets (internal/core/port.SecretStore)
// left behind when a pipeline node is removed from a graph, or when the
// virtual model owning the graph is deleted entirely.
package secretcleanup

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
)

// PruneRemovedNodes deletes secrets for every node present in oldGraph but
// absent from newGraph. Pass a nil newGraph to prune all of oldGraph's nodes
// (e.g. when the virtual model itself is being deleted).
func PruneRemovedNodes(ctx context.Context, store port.SecretStore, oldGraph, newGraph *model.PipelineGraph) error {
	if store == nil || oldGraph == nil {
		return nil
	}

	keep := make(map[string]struct{}, len(newGraph.NodeIDs()))
	for _, id := range newGraph.NodeIDs() {
		keep[id] = struct{}{}
	}

	for _, id := range oldGraph.NodeIDs() {
		if _, ok := keep[id]; ok {
			continue
		}
		if err := store.DeleteAllForNode(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

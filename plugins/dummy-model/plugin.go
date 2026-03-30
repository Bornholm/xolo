package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

const PluginName = "dummy-model"
const PluginVersion = "0.1.0"

const configSchemaJSON = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "trigger_models": {
      "type": "array",
      "items": { "type": "string" },
      "title": "Modèles déclencheurs",
      "description": "Noms des modèles virtuels interceptés par ce plugin (format local, sans préfixe org)."
    }
  }
}`

// Plugin implements the dummy-model gRPC plugin.
type Plugin struct {
	proto.UnimplementedXoloPluginServer
}

func (p *Plugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:         PluginName,
		Version:      PluginVersion,
		Description:  "Retourne une réponse forgée à la place du LLM, à des fins de test. Supporte les modes streaming et non-streaming.",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_RESOLVE_MODEL},
		ConfigSchema: configSchemaJSON,
	}, nil
}

// ResolveModel intercepts requests destined to configured trigger models and
// returns a forged response content. The proxy handles both streaming and
// non-streaming transparently via a DummyLLMClient.
func (p *Plugin) ResolveModel(ctx context.Context, in *proto.ResolveModelInput) (*proto.ResolveModelOutput, error) {
	cfg, err := ParseConfig(in.GetCtx().GetConfigJson())
	if err != nil {
		slog.WarnContext(ctx, "dummy-model: failed to parse config, skipping", slog.Any("error", err))
		return &proto.ResolveModelOutput{}, nil
	}

	// Only handle known virtual models.
	if !isVirtualModel(in.GetRequestedModel(), in.GetVirtualModels()) {
		return &proto.ResolveModelOutput{}, nil
	}

	// Strip org prefix (e.g. "orgslug/model-name" → "model-name") for trigger check.
	localModel := localModelName(in.GetRequestedModel())
	if !cfg.isTriggerModel(localModel) {
		return &proto.ResolveModelOutput{}, nil
	}

	// Identify the requesting user.
	userLabel := in.GetCtx().GetDisplayName()
	if userLabel == "" {
		userLabel = in.GetCtx().GetUserId()
	}
	if userLabel == "" {
		userLabel = "(inconnu)"
	}

	// Extract the last user message from messages_json.
	lastUserMessage := extractLastUserMessage(in.GetMessagesJson())

	content := fmt.Sprintf(
		"**[dummy-model — réponse de test]**\n\n"+
			"- **Utilisateur** : %s\n"+
			"- **Modèle invoqué** : %s\n"+
			"- **Message reçu** : %s\n\n"+
			"_Cette réponse a été produite directement par le plugin dummy-model à des fins de test, sans appel à un LLM réel._",
		userLabel,
		in.GetRequestedModel(),
		lastUserMessage,
	)

	return &proto.ResolveModelOutput{ResponseContent: content}, nil
}

// isVirtualModel returns true when the requested model matches a virtual model entry.
// Comparison handles both qualified ("org/model") and local ("model") names.
func isVirtualModel(requestedModel string, virtualModels []*proto.VirtualModelInfo) bool {
	local := localModelName(requestedModel)
	for _, vm := range virtualModels {
		if vm.Name == requestedModel || vm.Name == local {
			return true
		}
	}
	return false
}

// localModelName strips the org prefix from a qualified model name.
func localModelName(name string) string {
	if idx := strings.IndexByte(name, '/'); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// extractLastUserMessage parses a JSON messages array and returns the content
// of the last message with role "user". Returns a placeholder on failure.
func extractLastUserMessage(messagesJSON string) string {
	if messagesJSON == "" {
		return "(aucun message)"
	}

	var messages []struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		return "(impossible de lire le message)"
	}

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != "user" {
			continue
		}
		switch v := messages[i].Content.(type) {
		case string:
			return v
		default:
			if b, err := json.Marshal(v); err == nil {
				return string(b)
			}
		}
	}

	return "(aucun message utilisateur)"
}

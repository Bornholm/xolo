package component

import (
	"strings"

	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// pluginCapabilityLabel turns a capability of the plugin protocol into the
// wording of the interface: the card says what a plugin can hook into, which is
// what an administrator reads it for.
func pluginCapabilityLabel(capability proto.PluginDescriptor_Capability) string {
	switch capability {
	case proto.PluginDescriptor_PRE_REQUEST:
		return "Pré-requête"
	case proto.PluginDescriptor_POST_RESPONSE:
		return "Post-réponse"
	case proto.PluginDescriptor_RESOLVE_MODEL:
		return "Résolution de modèle"
	case proto.PluginDescriptor_LIST_MODELS:
		return "Liste de modèles"
	case proto.PluginDescriptor_TOOL_PROVIDER:
		return "Fournisseur d'outils"
	default:
		return strings.ToLower(capability.String())
	}
}

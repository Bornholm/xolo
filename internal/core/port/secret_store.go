package port

import "context"

// SecretStore persists opaque key/value secrets scoped to a single plugin
// node instance within a pipeline graph (identified by nodeID). Values are
// stored verbatim; callers (e.g. XoloHostService) are responsible for
// encrypting before SetSecret and decrypting after GetSecret, the same way
// Provider.APIKey is encrypted/decrypted by its callers rather than by the
// store. This backs the GetSecret/SetSecret/DeleteSecret RPCs exposed to
// plugins, so sensitive node configuration (e.g. an MCP server auth token)
// never has to be stored in the pipeline graph's visible JSON.
type SecretStore interface {
	GetSecret(ctx context.Context, orgID, pluginName, nodeID, key string) (value string, found bool, err error)
	SetSecret(ctx context.Context, orgID, pluginName, nodeID, key, value string) error
	DeleteSecret(ctx context.Context, orgID, pluginName, nodeID, key string) error
	// DeleteAllForNode removes every secret stored for nodeID, regardless of
	// key. Node IDs are unique per pipeline node instance across the whole
	// system, so this is the only scoping needed when a node (or the virtual
	// model it belongs to) is deleted.
	DeleteAllForNode(ctx context.Context, nodeID string) error
}

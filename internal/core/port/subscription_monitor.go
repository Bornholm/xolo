package port

import "github.com/bornholm/xolo/internal/core/model"

// SubscriptionMonitor gives read-only access to the in-memory subscription
// enforcement state (concurrency counters + exhaustion cooldowns).
// Implemented by proxy.SubscriptionStateDetailed without code change.
type SubscriptionMonitor interface {
	CurrentInFlight(orgID model.OrgID, providerID model.ProviderID) int
	CurrentUserInFlight(orgID model.OrgID, providerID model.ProviderID, userID model.UserID) int
	IsExhausted(orgID model.OrgID, providerID model.ProviderID, label string) bool
}

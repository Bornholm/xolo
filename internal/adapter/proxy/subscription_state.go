package proxy

import (
	"sync"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
)

type scopeKey struct {
	orgID      model.OrgID
	providerID model.ProviderID
}

// SubscriptionState holds in-memory per-scope state for subscription enforcement.
// State is not shared across replicas; the upstream resync on 429 acts as a safety net.
type SubscriptionState struct {
	mu          sync.Mutex
	inFlight    map[scopeKey]int
	exhaustedAt map[scopeKey]time.Time // per-constraint label
}

type constraintKey struct {
	scope scopeKey
	label string
}

type SubscriptionStateDetailed struct {
	mu              sync.Mutex
	inFlight        map[scopeKey]int
	exhaustedUntil  map[constraintKey]time.Time
}

func NewSubscriptionState() *SubscriptionStateDetailed {
	return &SubscriptionStateDetailed{
		inFlight:       make(map[scopeKey]int),
		exhaustedUntil: make(map[constraintKey]time.Time),
	}
}

// IncrementInFlight atomically increments the in-flight counter and returns the new value.
func (s *SubscriptionStateDetailed) IncrementInFlight(orgID model.OrgID, providerID model.ProviderID) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := scopeKey{orgID, providerID}
	s.inFlight[k]++
	return s.inFlight[k]
}

// DecrementInFlight atomically decrements the in-flight counter (floors at 0).
func (s *SubscriptionStateDetailed) DecrementInFlight(orgID model.OrgID, providerID model.ProviderID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := scopeKey{orgID, providerID}
	if s.inFlight[k] > 0 {
		s.inFlight[k]--
	}
}

// CurrentInFlight returns the current number of in-flight requests for the scope.
func (s *SubscriptionStateDetailed) CurrentInFlight(orgID model.OrgID, providerID model.ProviderID) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inFlight[scopeKey{orgID, providerID}]
}

// MarkExhausted sets a cooldown for a specific constraint label until the given time.
func (s *SubscriptionStateDetailed) MarkExhausted(orgID model.OrgID, providerID model.ProviderID, label string, until time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exhaustedUntil[constraintKey{scopeKey{orgID, providerID}, label}] = until
}

// IsExhausted reports whether a constraint is currently in cooldown.
func (s *SubscriptionStateDetailed) IsExhausted(orgID model.OrgID, providerID model.ProviderID, label string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, ok := s.exhaustedUntil[constraintKey{scopeKey{orgID, providerID}, label}]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		delete(s.exhaustedUntil, constraintKey{scopeKey{orgID, providerID}, label})
		return false
	}
	return true
}

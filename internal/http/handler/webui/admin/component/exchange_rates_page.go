package component

import (
	"fmt"

	"github.com/xolo-gateway/xolo/internal/core/model"
)

// lastSyncMessage states when the cache was last filled, from the freshest rate
// it holds — the service does not record a synchronisation timestamp of its own.
func lastSyncMessage(rates []model.ExchangeRate) string {
	if len(rates) == 0 {
		return "Aucune synchronisation n'a encore abouti."
	}

	latest := rates[0].FetchedAt
	for _, rate := range rates[1:] {
		if rate.FetchedAt.After(latest) {
			latest = rate.FetchedAt
		}
	}

	return fmt.Sprintf(
		"Dernière synchronisation le %s. Le rafraîchissement est automatique et périodique.",
		latest.Format("02/01/2006 à 15:04"),
	)
}

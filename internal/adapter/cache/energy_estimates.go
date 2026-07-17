package cache

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type EnergyTotals struct {
	TotalEnergyWh    float64
	TotalCO2GramsMid float64
}

// EnergyEstimateCache memoizes the per-period energy/CO₂ totals. Computing them is
// the only usage-page figure that still requires scanning every record of the period
// (the estimate is non-linear per request and cannot be aggregated in SQL), so this
// cache is what keeps that scan off the common request path. A few minutes of
// staleness on an approximate figure is an acceptable trade for that; the live cost
// charts are unaffected as they are aggregated in SQL on every request.
var EnergyEstimateCache = expirable.NewLRU[string, EnergyTotals](1000, nil, 5*time.Minute)

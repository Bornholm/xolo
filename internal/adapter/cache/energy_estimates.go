package cache

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type EnergyTotals struct {
	TotalEnergyWh    float64
	TotalCO2GramsMid float64
}

var EnergyEstimateCache = expirable.NewLRU[string, EnergyTotals](1000, nil, time.Minute)

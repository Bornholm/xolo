package component

import (
	"fmt"

	"github.com/bornholm/xolo/internal/estimator/energy"
)

// ChartDataPoint represents a single labeled value for bar/pie charts.
type ChartDataPoint struct {
	Label string
	Value float64 // currency units (NOT microcents)
}

// ChartLabels extracts the labels from a slice of ChartDataPoint.
func ChartLabels(pts []ChartDataPoint) []string {
	labels := make([]string, len(pts))
	for i, p := range pts {
		labels[i] = p.Label
	}
	return labels
}

// ChartValues extracts the values from a slice of ChartDataPoint.
func ChartValues(pts []ChartDataPoint) []float64 {
	vals := make([]float64, len(pts))
	for i, p := range pts {
		vals[i] = p.Value
	}
	return vals
}

// ChartShare represents a value's share of a total, as a percentage, with
// an associated chart color (cycling through the design system's palette).
type ChartShare struct {
	Label string
	Pct   int
	Color string
}

// ChartShares converts a list of data points into percentage shares of
// their total, cycling through the design system's chart color palette.
func ChartShares(pts []ChartDataPoint) []ChartShare {
	var total float64
	for _, p := range pts {
		total += p.Value
	}
	colors := []string{"var(--chart-1)", "var(--chart-2)", "var(--chart-3)", "var(--chart-4)"}
	shares := make([]ChartShare, 0, len(pts))
	for i, p := range pts {
		pct := 0
		if total > 0 {
			pct = int(p.Value / total * 100)
		}
		shares = append(shares, ChartShare{Label: p.Label, Pct: pct, Color: colors[i%len(colors)]})
	}
	return shares
}

// CurrencySymbol returns the display symbol for an ISO 4217 currency code.
func CurrencySymbol(currency string) string {
	switch currency {
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "JPY":
		return "¥"
	case "CHF":
		return "CHF "
	case "CAD":
		return "CA$"
	case "AUD":
		return "AU$"
	default:
		return "$" // USD
	}
}

// FormatCost formats an absolute cost stored in microcents (1 microcent = $0.000001).
func FormatCost(v int64, currency string) string {
	return fmt.Sprintf("%.6f%s", float64(v)/1_000_000, CurrencySymbol(currency))
}

// UsagePercent returns the percentage of budget consumed by used, capped at 100.
func UsagePercent(used int64, budget *int64) int {
	if budget == nil || *budget == 0 {
		return 0
	}
	pct := int(used * 100 / *budget)
	if pct > 100 {
		return 100
	}
	return pct
}

// FormatEnergyWh formats an energy value with auto-scaling (kWh, Wh, mWh, µWh).
func FormatEnergyWh(wh float64) string {
	if wh <= 0 {
		return "—"
	}
	if wh >= 1000 {
		return fmt.Sprintf("%.3f kWh", wh/1000)
	}
	if wh >= 1 {
		return fmt.Sprintf("%.3f Wh", wh)
	}
	if wh >= 0.001 {
		return fmt.Sprintf("%.3f mWh", wh*1000)
	}
	return fmt.Sprintf("%.3f µWh", wh*1_000_000)
}

// FormatCO2Grams formats a CO₂ quantity in grams with auto-scaling (g, mg, µg).
func FormatCO2Grams(g float64) string {
	if g <= 0 {
		return "—"
	}
	if g >= 1 {
		return fmt.Sprintf("%.3f gCO₂", g)
	}
	if g >= 0.001 {
		return fmt.Sprintf("%.3f mgCO₂", g*1000)
	}
	return fmt.Sprintf("%.3f µgCO₂", g*1_000_000)
}

// FormatCO2ToCarKilometers converts CO2 grams to equivalent car kilometers
// based on ADEME data (~109g CO2/km for petrol vehicle).
func FormatCO2ToCarKilometers(grams float64) string {
	if grams <= 0 {
		return ""
	}
	km := grams / 109
	if km < 1 {
		return fmt.Sprintf("Soit environ %d m parcouru seul dans un véhicule thermique à essence, selon les chiffres de l'ADEME.", int(km*1000))
	}
	if km < 1000 {
		return fmt.Sprintf("Soit environ %d km parcouru seul dans un véhicule thermique à essence, selon les chiffres de l'ADEME.", int(km))
	}
	return fmt.Sprintf("Soit environ %s km parcouru seul dans un véhicule thermique à essence, selon les chiffres de l'ADEME.", FormatNumber(km))
}

// FormatEnergyToHuman converts energy in Wh to human-equivalent appliance usage.
func FormatEnergyToHuman(wh float64) string {
	if wh <= 0 {
		return ""
	}
	return energy.HumanEquivalent(wh)
}

// FormatNumber formats a float with one decimal place.
func FormatNumber(n float64) string {
	return fmt.Sprintf("%.1f", n)
}

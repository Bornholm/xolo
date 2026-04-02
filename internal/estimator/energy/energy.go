package energy

import "fmt"

type Appliance struct {
	Name          string
	AnnualKWh     float64
	AlwaysOn      bool
	CyclesPerYear int
	UnitLabel     string
}

var appliances = []Appliance{
	{Name: "four à micro-ondes", AnnualKWh: 39},
	{Name: "four électrique", AnnualKWh: 146, CyclesPerYear: 187, UnitLabel: "cuisson"},
	{Name: "plaque vitrocéramique", AnnualKWh: 159, CyclesPerYear: 409, UnitLabel: "cuisson"},
	{Name: "réfrigérateur", AnnualKWh: 174, AlwaysOn: true},
	{Name: "lave-vaisselle", AnnualKWh: 192, CyclesPerYear: 166, UnitLabel: "cycle"},
	{Name: "cave à vin", AnnualKWh: 193, AlwaysOn: true},
	{Name: "congélateur", AnnualKWh: 308, AlwaysOn: true},
	{Name: "réfrigérateur combiné", AnnualKWh: 346, AlwaysOn: true},
	{Name: "box Internet", AnnualKWh: 97, AlwaysOn: true},
	{Name: "décodeur TV", AnnualKWh: 87, AlwaysOn: true},
	{Name: "téléviseur LCD", AnnualKWh: 187},
	{Name: "téléviseur OLED", AnnualKWh: 153},
	{Name: "console de jeux vidéo", AnnualKWh: 103},
	{Name: "enceinte connectée", AnnualKWh: 18, AlwaysOn: true},
	{Name: "chaîne Hi-Fi", AnnualKWh: 25},
	{Name: "ordinateur portable", AnnualKWh: 22},
	{Name: "ordinateur fixe avec écran", AnnualKWh: 123},
	{Name: "imprimante", AnnualKWh: 23},
	{Name: "écran d'ordinateur", AnnualKWh: 60},
	{Name: "lave-linge", AnnualKWh: 101, CyclesPerYear: 198, UnitLabel: "lessive"},
	{Name: "sèche-linge", AnnualKWh: 301, CyclesPerYear: 198, UnitLabel: "cycle de séchage"},
	{Name: "fer à repasser", AnnualKWh: 26},
	{Name: "sèche-cheveux", AnnualKWh: 29},
	{Name: "chauffe-eau électrique (200L)", AnnualKWh: 1676, AlwaysOn: true},
	{Name: "ampoule LED (allumée 6h/jour)", AnnualKWh: 7.3},
	{Name: "ampoule halogène (allumée 6h/jour)", AnnualKWh: 45},
	{Name: "radiateur électrique (pièce de 20m²)", AnnualKWh: 3200},
	{Name: "climatiseur mobile", AnnualKWh: 500},
	{Name: "ventilateur", AnnualKWh: 15},
	{Name: "chaudière (circulateur + brûleur)", AnnualKWh: 250},
	{Name: "tondeuse électrique", AnnualKWh: 20},
	{Name: "piscine (pompe de filtration)", AnnualKWh: 1750, AlwaysOn: true},
	{Name: "recharge de smartphone", AnnualKWh: 3.5},
	{Name: "recharge de tablette", AnnualKWh: 9},
	{Name: "recharge de voiture électrique (15 000 km/an)", AnnualKWh: 2500},
	{Name: "ensemble des veilles d'un foyer", AnnualKWh: 300, AlwaysOn: true},
}

func HumanEquivalent(wh float64) string {
	kWh := wh / 1000.0

	type candidate struct {
		text  string
		score float64
	}

	var best candidate

	for _, a := range appliances {
		ratio := kWh / a.AnnualKWh

		text, score := formatDuration(ratio, a)
		if best.text == "" || score > best.score {
			best = candidate{text: text, score: score}
		}
	}

	return best.text
}

func formatDuration(years float64, a Appliance) (string, float64) {
	if a.CyclesPerYear > 0 {
		cycles := years * float64(a.CyclesPerYear)
		if cycles >= 0.5 {
			return fmt.Sprintf("Soit environ %s de %s",
				pluralize(cycles, a.UnitLabel), a.Name), scoreness(cycles)
		}
	}

	hours := years * 365.25 * 24
	days := years * 365.25
	weeks := days / 7
	months := years * 12

	switch {
	case hours < 1:
		minutes := hours * 60
		return fmt.Sprintf("Soit environ %s de fonctionnement d'un %s",
			pluralize(minutes, "minute"), a.Name), scoreness(minutes)
	case hours < 48:
		return fmt.Sprintf("Soit environ %s de fonctionnement d'un %s",
			pluralize(hours, "heure"), a.Name), scoreness(hours)
	case days < 14:
		return fmt.Sprintf("Soit environ %s de fonctionnement d'un %s",
			pluralize(days, "jour"), a.Name), scoreness(days)
	case weeks < 8:
		return fmt.Sprintf("Soit environ %s de fonctionnement d'un %s",
			pluralize(weeks, "semaine"), a.Name), scoreness(weeks)
	case months < 24:
		return fmt.Sprintf("Soit environ %s de fonctionnement d'un %s",
			pluralize(months, "mois"), a.Name), scoreness(months)
	default:
		return fmt.Sprintf("Soit environ %s de fonctionnement d'un %s",
			pluralize(years, "an"), a.Name), scoreness(years)
	}
}

func scoreness(value float64) float64 {
	if value < 0.5 {
		return value
	}
	if value > 1000 {
		return 1.0 / value
	}
	if value >= 2 && value <= 30 {
		return 100
	}
	if value >= 1 && value < 2 {
		return 80
	}
	if value > 30 && value <= 100 {
		return 50
	}
	return 10
}

func pluralize(value float64, unit string) string {
	var display string
	switch {
	case value < 10:
		display = fmt.Sprintf("%.1f", value)
	default:
		display = fmt.Sprintf("%.0f", value)
	}

	plural := unit + "s"
	switch unit {
	case "mois":
		plural = "mois"
	}

	if value >= 2 {
		return display + " " + plural
	}
	return display + " " + unit
}

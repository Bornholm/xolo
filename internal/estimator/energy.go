package estimator

// CarbonIntensity represents the carbon intensity of an electricity grid.
type CarbonIntensity struct {
	Name  string
	Value float64 // kgCO₂/kWh (= gCO₂/Wh)
}

// Carbon intensity constants (kgCO₂/kWh).
const (
	CarbonFrance    = 0.027
	CarbonSweden    = 0.045
	CarbonEUAverage = 0.276
	CarbonUSAverage = 0.390
	CarbonChina     = 0.5366
	CarbonGasPlant  = 0.490
	CarbonCoalPlant = 0.960
	CarbonWorldAvg  = 0.475
)

// Pre-defined CarbonIntensity values for common grids.
var (
	CarbonIntensityFrance    = CarbonIntensity{Name: "France (nucléaire)", Value: CarbonFrance}
	CarbonIntensitySweden    = CarbonIntensity{Name: "Suède", Value: CarbonSweden}
	CarbonIntensityEUAverage = CarbonIntensity{Name: "Moyenne UE", Value: CarbonEUAverage}
	CarbonIntensityUSAverage = CarbonIntensity{Name: "Moyenne US", Value: CarbonUSAverage}
	CarbonIntensityChina     = CarbonIntensity{Name: "Chine", Value: CarbonChina}
	CarbonIntensityGasPlant  = CarbonIntensity{Name: "Gaz naturel", Value: CarbonGasPlant}
	CarbonIntensityCoalPlant = CarbonIntensity{Name: "Charbon", Value: CarbonCoalPlant}
	CarbonIntensityWorldAvg  = CarbonIntensity{Name: "Monde (moyenne)", Value: CarbonWorldAvg}
)

// Equivalences provides human-readable analogies for an energy quantity.
type Equivalences struct {
	LEDBulbSeconds    float64 // seconds powering a 10W LED bulb
	GoogleSearches    float64 // equivalent Google searches (~0.3 Wh each)
	SmartphoneCharges float64 // smartphone full charges (~14 Wh each)
	CO2Grams          float64 // grams CO₂ at chosen carbon intensity
	CO2GramsMin       float64 // grams CO₂ at France nuclear intensity (best case)
	CO2GramsMax       float64 // grams CO₂ at coal plant intensity (worst case)
}

// computeEquivalences converts a Wh value into human-readable equivalences.
// ci is used for CO2Grams; CO2GramsMin/Max use France and coal constants.
func computeEquivalences(wh float64, ci CarbonIntensity) Equivalences {
	return Equivalences{
		LEDBulbSeconds:    (wh / 10.0) * 3600.0, // 10 W LED
		GoogleSearches:    wh / 0.3,              // ~0.3 Wh per search
		SmartphoneCharges: wh / 14.0,             // ~14 Wh per full charge
		CO2Grams:          wh * ci.Value,         // 1 kgCO₂/kWh = 1 gCO₂/Wh
		CO2GramsMin:       wh * CarbonFrance,
		CO2GramsMax:       wh * CarbonCoalPlant,
	}
}

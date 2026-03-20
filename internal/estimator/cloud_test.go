package estimator

import (
	"math"
	"testing"
)

func TestCloudEstimator_ScalingWithParams(t *testing.T) {
	ce := NewCloudEstimator(TierHyperscaler)
	req := InferenceRequest{InputTokens: 500, OutputTokens: 200}

	r1 := ce.EstimateFromParams(7e9, req, 0, 0)
	r2 := ce.EstimateFromParams(70e9, req, 0, 0)

	// Avec le modèle hybride TDP+FLOP, le TDP scale en params^1.4 (larger models are slower).
	// On vérifie juste que 70B est significativement plus énergivore que 7B.
	ratio := r2.Mid.TotalJoules / r1.Mid.TotalJoules
	if ratio < 5.0 || ratio > 100.0 {
		t.Errorf("ratio attendu 5-100x, got %.1fx", ratio)
	}
}

func TestCloudEstimator_ScalingWithTokens(t *testing.T) {
	ce := NewCloudEstimator(TierMajorCloud)

	r1 := ce.EstimateFromParams(70e9, InferenceRequest{InputTokens: 500, OutputTokens: 100}, 0, 0)
	r2 := ce.EstimateFromParams(70e9, InferenceRequest{InputTokens: 500, OutputTokens: 1000}, 0, 0)

	// 10x plus de tokens output → significativement plus d'énergie
	if r2.Mid.TotalJoules <= r1.Mid.TotalJoules*2 {
		t.Error("10x plus de tokens output devrait coûter beaucoup plus cher")
	}
}

func TestCloudEstimator_LowLessThanHigh(t *testing.T) {
	for _, tier := range []CloudTier{TierHyperscaler, TierMajorCloud, TierSmallProvider} {
		ce := NewCloudEstimator(tier)
		r := ce.EstimateFromParams(70e9, InferenceRequest{InputTokens: 1000, OutputTokens: 500}, 0, 0)

		if r.Low.TotalJoules >= r.High.TotalJoules {
			t.Errorf("tier %v: Low doit être < High", tier)
		}
		if r.Mid.TotalJoules <= r.Low.TotalJoules || r.Mid.TotalJoules >= r.High.TotalJoules {
			t.Errorf("tier %v: Mid doit être entre Low et High", tier)
		}
	}
}

func TestCloudEstimator_PositiveEnergy(t *testing.T) {
	ce := NewCloudEstimator(TierHyperscaler)
	r := ce.EstimateFromParams(7e9, InferenceRequest{InputTokens: 100, OutputTokens: 50}, 0, 0)

	if r.Mid.TotalJoules <= 0 {
		t.Error("l'énergie médiane doit être > 0")
	}
	if r.Mid.TotalWh <= 0 {
		t.Error("TotalWh médian doit être > 0")
	}
}

func TestCloudEstimator_ZeroTokens(t *testing.T) {
	ce := NewCloudEstimator(TierHyperscaler)
	r := ce.EstimateFromParams(70e9, InferenceRequest{InputTokens: 0, OutputTokens: 0}, 0, 0)

	// Avec 0 tokens, l'énergie doit être 0
	if r.Mid.TotalJoules != 0 {
		t.Errorf("avec 0 tokens, énergie attendue = 0, got %f", r.Mid.TotalJoules)
	}
	// Pas de division par zéro
	if r.Mid.JoulesPerToken != 0 {
		t.Errorf("JoulesPerToken doit être 0 quand totalTokens=0")
	}
}

func TestCloudEstimator_ReasonableRange(t *testing.T) {
	// Validation : une requête GPT-like (280B params, 2000+500 tokens, hyperscaler)
	// doit être dans une plage raisonnable : 0.001 à 10 Wh (IEA : 1-10 mWh par requête)
	ce := NewCloudEstimator(TierHyperscaler)
	r := ce.EstimateFromParams(280e9, InferenceRequest{InputTokens: 2000, OutputTokens: 500}, 0, 0)

	if r.Mid.TotalWh < 0.001 || r.Mid.TotalWh > 10 {
		t.Errorf("énergie hors plage raisonnable: %.6f Wh", r.Mid.TotalWh)
	}

	t.Logf("280B params, 2000+500 tokens (hyperscaler): %.6f - %.6f - %.6f Wh (×%.1f incertitude)",
		r.Low.TotalWh, r.Mid.TotalWh, r.High.TotalWh,
		r.High.TotalJoules/r.Low.TotalJoules)
}

func TestCarbonIntensityRange(t *testing.T) {
	ce := NewCloudEstimator(TierHyperscaler)
	r := ce.EstimateFromParams(70e9, InferenceRequest{InputTokens: 1000, OutputTokens: 500}, 0, 0)

	eq := r.Mid.Equivalences
	if eq.CO2GramsMin >= eq.CO2Grams {
		t.Errorf("CO2GramsMin (%f) doit être < CO2Grams (%f)", eq.CO2GramsMin, eq.CO2Grams)
	}
	if eq.CO2Grams >= eq.CO2GramsMax {
		t.Errorf("CO2Grams (%f) doit être < CO2GramsMax (%f)", eq.CO2Grams, eq.CO2GramsMax)
	}
}

func TestHybridVsFlopOnly(t *testing.T) {
	// Vérifie que le résultat hybride (TDP+FLOP) diffère de l'ancien FLOP-seul.
	// On utilise un modèle 70B pour lequel les deux méthodes donnent des résultats différents.
	ce := NewCloudEstimator(TierHyperscaler)
	req := InferenceRequest{InputTokens: 1000, OutputTokens: 500}

	hybrid := ce.EstimateFromParams(70e9, req, 0, 0)

	// Calcul FLOP-seul (référence ancienne implémentation, avec facteur 0.3 prefill)
	flopPerToken := 2.0 * 70e9
	prefillFLOP := flopPerToken * 1000 * 0.3 // ancien facteur 0.3
	decodeFLOP := flopPerToken * 500
	totalFLOP := prefillFLOP + decodeFLOP
	// Efficacité ancienne : 800-2000 GFLOP/J, overhead 1.10
	oldHighJoules := totalFLOP / (2000 * 1e9) * 1.08 * 1.10
	oldLowJoules := totalFLOP / (800 * 1e9) * 1.15 * 1.10
	oldMid := math.Sqrt(oldHighJoules * oldLowJoules)

	// Le résultat hybride doit être différent de l'ancien FLOP-seul
	if math.Abs(hybrid.Mid.TotalJoules-oldMid)/oldMid < 0.05 {
		t.Errorf("le résultat hybride (%.2f J) devrait différer significativement du FLOP-seul (%.2f J)",
			hybrid.Mid.TotalJoules, oldMid)
	}
}

func TestGPUPowerFloor(t *testing.T) {
	// Vérifie que le plancher GPU est effectif pour un très petit modèle.
	// 0.5B params → WattsPerBParamsLow × 0.5 = 0.05 W << MinGPUWattsLow (2 W)
	// On vérifie via TDPMidWh (qui n'est pas affecté par l'enveloppe min/max FLOP).
	ce := NewCloudEstimator(TierHyperscaler)
	req := InferenceRequest{InputTokens: 100, OutputTokens: 50}

	r := ce.EstimateFromParams(0.5e9, req, 0, 0)

	// Durée optimiste ≈ decode + prefill avec tps optimiste
	tps := estimateTokensPerSec(0.5e9, true)
	dur := decodeDuration(50, tps) + prefillDuration(100, tps)

	// tdpLow (optimiste) doit utiliser le plancher MinGPUWattsLow, pas 0.10×0.5=0.05 W
	tdpLowExpected := PresetHyperscaler.MinGPUWattsLow * dur * PresetHyperscaler.PUELow * 1.20
	tdpLowWithoutFloor := PresetHyperscaler.WattsPerBParamsLow * 0.5 * dur * PresetHyperscaler.PUELow * 1.20

	// Le plancher doit être significativement plus grand que sans plancher
	if tdpLowExpected <= tdpLowWithoutFloor*1.5 {
		t.Fatalf("configuration de test incorrecte : plancher (%.4f J) pas assez > sans plancher (%.4f J)", tdpLowExpected, tdpLowWithoutFloor)
	}

	// TDPMidWh×3600 >= tdpLow (géométrique mean >= min)
	if r.TDPMidWh*3600 < tdpLowExpected*0.99 {
		t.Errorf("plancher GPU non respecté dans TDP : TDPMid=%.4f J < tdpLow attendu=%.4f J",
			r.TDPMidWh*3600, tdpLowExpected)
	}
	t.Logf("0.5B params, 100+50 tokens: TDPMid=%.4f J (plancher=%.4f J, sans plancher=%.4f J)",
		r.TDPMidWh*3600, tdpLowExpected, tdpLowWithoutFloor)
}

func TestDynamicPrefill(t *testing.T) {
	// Vérifie que le facteur prefill est dynamique et décroissant.
	if prefillSpeedup(100) < 15 {
		t.Errorf("prefillSpeedup(100) = %.2f, attendu >= 15", prefillSpeedup(100))
	}
	if prefillSpeedup(50000) < 1.5 {
		t.Errorf("prefillSpeedup(50000) = %.2f, attendu >= 1.5", prefillSpeedup(50000))
	}
	if prefillSpeedup(100) <= prefillSpeedup(20000) {
		t.Errorf("prefillSpeedup doit être décroissant : speedup(100)=%.2f <= speedup(20000)=%.2f",
			prefillSpeedup(100), prefillSpeedup(20000))
	}
	t.Logf("prefillSpeedup: 100tok=%.2f×, 1000tok=%.2f×, 5000tok=%.2f×, 20000tok=%.2f×, 50000tok=%.2f×",
		prefillSpeedup(100), prefillSpeedup(1000), prefillSpeedup(5000), prefillSpeedup(20000), prefillSpeedup(50000))
}

func TestAttentionFactor(t *testing.T) {
	// Un long contexte doit coûter plus en FLOP qu'une simple proportion de tokens,
	// grâce à l'attention factor O(n²) en prefill.
	ce := NewCloudEstimator(TierHyperscaler)
	shortReq := InferenceRequest{InputTokens: 100, OutputTokens: 100}
	longReq := InferenceRequest{InputTokens: 50000, OutputTokens: 100}

	rShort := ce.EstimateFromParams(70e9, shortReq, 100, 50)
	rLong := ce.EstimateFromParams(70e9, longReq, 100, 50)

	tokenRatio := float64(longReq.InputTokens+longReq.OutputTokens) / float64(shortReq.InputTokens+shortReq.OutputTokens)
	flopRatio := rLong.FLOPMidWh / rShort.FLOPMidWh

	// L'attention factor doit amplifier le coût au-delà du simple ratio de tokens
	if flopRatio <= tokenRatio {
		t.Errorf("ratio FLOP long/court (%.2fx) devrait dépasser le ratio tokens (%.2fx) grâce à l'attention factor",
			flopRatio, tokenRatio)
	}
	t.Logf("short (%d+%d tokens): FLOPMid=%.4f mWh", shortReq.InputTokens, shortReq.OutputTokens, rShort.FLOPMidWh*1000)
	t.Logf("long (%d+%d tokens):  FLOPMid=%.4f mWh", longReq.InputTokens, longReq.OutputTokens, rLong.FLOPMidWh*1000)
	t.Logf("ratio tokens: %.1fx, ratio FLOP: %.1fx (surcoût attention: +%.0f%%)",
		tokenRatio, flopRatio, (flopRatio/tokenRatio-1)*100)
}

func TestTDPFloorFallback(t *testing.T) {
	// 7B hyperscaler : plancher actif (0.10×7=0.7W < 2W)
	// → TDPFloorActive=true, spread compact ≈ ×1.5 (au lieu de ×22)
	ce := NewCloudEstimator(TierHyperscaler)
	r := ce.EstimateFromParams(7e9, InferenceRequest{InputTokens: 1000, OutputTokens: 200}, 0, 0)

	if !r.TDPFloorActive {
		t.Error("7B hyperscaler : plancher TDP devrait être actif")
	}
	spread := r.High.TotalWh / r.Low.TotalWh
	if spread > 5.0 {
		t.Errorf("spread attendu < 5× pour FLOP-only, got %.1f×", spread)
	}
	t.Logf("7B hyperscaler: Low=%.4f mWh, High=%.4f mWh, spread=%.1f×", r.Low.TotalWh*1000, r.High.TotalWh*1000, spread)
}

func TestTDPFloorInactive(t *testing.T) {
	// 70B hyperscaler : pas de plancher (0.10×70=7W > 2W)
	// → TDPFloorActive=false, enveloppe hybride normale
	ce := NewCloudEstimator(TierHyperscaler)
	r := ce.EstimateFromParams(70e9, InferenceRequest{InputTokens: 1000, OutputTokens: 500}, 0, 0)

	if r.TDPFloorActive {
		t.Error("70B hyperscaler : plancher TDP ne devrait pas être actif")
	}
	t.Logf("70B hyperscaler: TDPFloorActive=%v, spread=%.1f×", r.TDPFloorActive, r.High.TotalWh/r.Low.TotalWh)
}

func TestMethodDivergence(t *testing.T) {
	// Vérifie que les midpoints TDP et FLOP sont exposés et positifs.
	ce := NewCloudEstimator(TierHyperscaler)
	r := ce.EstimateFromParams(70e9, InferenceRequest{InputTokens: 1000, OutputTokens: 500}, 0, 0)

	if r.TDPMidWh <= 0 {
		t.Errorf("TDPMidWh doit être > 0, got %f", r.TDPMidWh)
	}
	if r.FLOPMidWh <= 0 {
		t.Errorf("FLOPMidWh doit être > 0, got %f", r.FLOPMidWh)
	}
	ratio := r.TDPMidWh / r.FLOPMidWh
	t.Logf("70B, 1000+500 tokens: TDP_mid=%.4f mWh, FLOP_mid=%.4f mWh, ratio=%.2f",
		r.TDPMidWh*1000, r.FLOPMidWh*1000, ratio)
}

package energy

import (
	"fmt"
	"testing"
)

func TestHumanEquivalent(t *testing.T) {
	tests := []struct {
		wh float64
	}{
		{wh: 1},
		{wh: 10},
		{wh: 50},
		{wh: 500},
		{wh: 3_500},
		{wh: 5_000},
		{wh: 22_000},
		{wh: 50_000},
		{wh: 174_000},
		{wh: 500_000},
		{wh: 1_000_000},
		{wh: 2_500_000},
		{wh: 10_000_000},
		{wh: 100_000_000},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%.0fWh", tt.wh), func(t *testing.T) {
			got := HumanEquivalent(tt.wh)
			if got == "" {
				t.Error("résultat vide")
			}
			t.Logf("%12.0f Wh → %s", tt.wh, got)
		})
	}
}

package component

import "testing"

// The expected values below are written with explicit escapes: the two
// separators are invisible, so a literal " " in a test would happily accept a
// regression back to an ASCII space.
const (
	nnbsp = " " // narrow no-break space — thousands
	nbsp  = " " // no-break space — before a unit
)

func TestFormatCost(t *testing.T) {
	for _, tc := range []struct {
		name     string
		microc   int64
		currency string
		want     string
	}{
		{"amount above one unit keeps two decimals", 1_284_400_000, "EUR", "1" + nnbsp + "284,40" + nbsp + "€"},
		{"zero is not a fraction", 0, "EUR", "0,00" + nbsp + "€"},
		{"sub-cent call cost keeps four decimals", 41_200, "EUR", "0,0412" + nbsp + "€"},
		{"thousands are grouped", 9_504_560_000, "EUR", "9" + nnbsp + "504,56" + nbsp + "€"},
		{"negative amounts keep their sign", -1_500_000, "USD", "-1,50" + nbsp + "$"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatCost(tc.microc, tc.currency); got != tc.want {
				t.Errorf("FormatCost(%d, %q) = %q, want %q", tc.microc, tc.currency, got, tc.want)
			}
		})
	}
}

func TestFormatCount(t *testing.T) {
	for _, tc := range []struct {
		name string
		n    int64
		want string
	}{
		{"units are printed as-is", 0, "0"},
		{"hundreds are not grouped", 940, "940"},
		{"thousands are grouped in full", 31_940, "31" + nnbsp + "940"},
		{"millions switch to a suffix", 48_200_000, "48,2" + nbsp + "M"},
		{"billions switch to a suffix", 2_400_000_000, "2,4" + nbsp + "B"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatCount(tc.n); got != tc.want {
				t.Errorf("FormatCount(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}

func TestFormatDecimalGroupsFromTheRight(t *testing.T) {
	// Grouping is inserted by walking the digits left to right, so the boundary
	// cases are the ones where the first group is shorter than three digits.
	for _, tc := range []struct {
		v    float64
		want string
	}{
		{1, "1"},
		{100, "100"},
		{1000, "1" + nnbsp + "000"},
		{10000, "10" + nnbsp + "000"},
		{100000, "100" + nnbsp + "000"},
		{1000000, "1" + nnbsp + "000" + nnbsp + "000"},
	} {
		if got := FormatDecimal(tc.v, 0); got != tc.want {
			t.Errorf("FormatDecimal(%v, 0) = %q, want %q", tc.v, got, tc.want)
		}
	}
}

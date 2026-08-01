package component

// Tone is the semantic colour of a value: the design system reserves colour for
// meaning, so a tone is what a screen declares — never a class.
type Tone string

const (
	ToneNeutral  Tone = ""
	TonePositive Tone = "positive"
	ToneWarning  Tone = "warning"
	ToneNegative Tone = "negative"
	ToneInfo     Tone = "info"
)

// TextClass returns the foreground utility of a tone.
func (t Tone) TextClass() string {
	switch t {
	case TonePositive:
		return "text-success"
	case ToneWarning:
		return "text-warning"
	case ToneNegative:
		return "text-destructive"
	case ToneInfo:
		return "text-primary"
	default:
		return "text-muted-foreground"
	}
}

// BadgeClass returns the tint/ink pair of a badge, as listed in §06 « Badges &
// états » of the design system: « actif » on #eaf6ef/#16794a, « brouillon » on
// #fdf0dc/#8a5a08, « dégradé » on #fbe4e3/#a8231c, « visite admin » on
// #f0eafa/#6d4aa8, « inactif » on the neutral grey. Every pair is a tint token
// and its semantic colour — the design system never outlines a badge.
func (t Tone) BadgeClass() string {
	switch t {
	case TonePositive:
		return "bg-success-tint text-success"
	case ToneWarning:
		return "bg-warning-tint text-warning"
	case ToneNegative:
		return "bg-destructive-tint text-destructive"
	case ToneInfo:
		return "bg-platform-tint text-platform"
	default:
		return "bg-muted text-muted-foreground"
	}
}

// BudgetTone returns the tone of a budget consumption ratio, on the thresholds
// of the design system: green below 85 %, amber from 85 to 100 %, red at or
// above 100 %.
func BudgetTone(pct int) Tone {
	switch {
	case pct >= 100:
		return ToneNegative
	case pct >= 85:
		return ToneWarning
	default:
		return TonePositive
	}
}

// BudgetBarClass returns the fill utility of a budget gauge, on the same
// thresholds as BudgetTone.
//
// It is the design-system counterpart of QuotaBarClass, which predates the
// rework and uses chart colours instead of the semantic ones.
func BudgetBarClass(pct int) string {
	switch {
	case pct >= 100:
		return "bg-destructive-indicator"
	case pct >= 85:
		return "bg-warning-indicator"
	default:
		return "bg-success-indicator"
	}
}

package complexity

import (
	"fmt"
	"testing"
)

func TestAnalyzeDefault(t *testing.T) {
	cases := []struct {
		name    string
		prompt  string
		wantMin float64
		wantMax float64
		wantLvl string
	}{
		{
			name:    "trivial greeting",
			prompt:  "Bonjour",
			wantMin: 0.0,
			wantMax: 0.30,
			wantLvl: "trivial",
		},
		{
			name:    "simple question",
			prompt:  "Quelle est la capitale de la France ?",
			wantMin: 0.05,
			wantMax: 0.45,
		},
		{
			name:    "moderate multi-step",
			prompt:  "Explique-moi les différences entre TCP et UDP. Donne des exemples concrets d'utilisation pour chacun. Quel protocole choisir pour un jeu en ligne temps réel ?",
			wantMin: 0.25,
			wantMax: 0.70,
		},
		{
			name: "complex constrained",
			prompt: `Tu es un expert en architecture logicielle. Analyse le pattern CQRS et compare-le avec une architecture classique en couches.
Donne-moi :
1. Les avantages et inconvénients de chaque approche
2. Un diagramme en ASCII art montrant les flux de données
3. Un exemple de code en Go illustrant CQRS
4. Des métriques de performance comparatives

Réponds en français, en moins de 2000 mots, au format markdown avec des headers et du code formaté.
Si le contexte est un système e-commerce avec 10 000 requêtes/seconde, quel serait ton choix et pourquoi ?`,
			wantMin: 0.50,
			wantMax: 1.0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := AnalyzeDefault(tc.prompt)
			if s.Composite < tc.wantMin || s.Composite > tc.wantMax {
				t.Errorf("composite=%.3f, want [%.2f, %.2f]", s.Composite, tc.wantMin, tc.wantMax)
			}
			if tc.wantLvl != "" && s.Level != tc.wantLvl {
				t.Errorf("level=%q, want %q", s.Level, tc.wantLvl)
			}
			t.Logf("%-25s composite=%.3f level=%-12s tokens=%d constraints=%d entropy=%.2f",
				tc.name, s.Composite, s.Level, s.Stats.TokenCount, s.Stats.ConstraintCount, s.Stats.ShannonEntropy)
		})
	}
}

func TestShannonEntropy(t *testing.T) {
	// Repeated char should have very low entropy
	low := shannonEntropy("aaaaaaaaaaaaaaa")
	high := shannonEntropy("The quick brown fox jumps over the lazy dog")
	if low >= high {
		t.Errorf("expected repeated text entropy (%.3f) < varied text (%.3f)", low, high)
	}
}

func TestCountConstraints(t *testing.T) {
	text := "Réponds en format JSON, en moins de 500 mots. Tu dois inclure un tableau. Si le résultat est vide, alors retourne null."
	c := countConstraints(text)
	if c < 3 {
		t.Errorf("expected at least 3 constraints, got %d", c)
	}
}

func TestNestingDepth(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"hello", 0},
		{"(a)", 1},
		{"(a (b (c)))", 3},
		{"[{(x)}]", 3},
	}
	for _, tc := range cases {
		got := nestingDepth(tc.text)
		if got != tc.want {
			t.Errorf("nestingDepth(%q)=%d, want %d", tc.text, got, tc.want)
		}
	}
}

func TestCountSyllables(t *testing.T) {
	cases := []struct {
		word string
		want int
	}{
		{"the", 1},
		{"hello", 2},
		{"beautiful", 3},
		{"architecture", 4},
	}
	for _, tc := range cases {
		got := countSyllables(tc.word)
		if got != tc.want {
			t.Errorf("countSyllables(%q)=%d, want %d", tc.word, got, tc.want)
		}
	}
}

// Benchmark to demonstrate CPU efficiency
func BenchmarkAnalyze(b *testing.B) {
	prompt := `Tu es un expert en architecture logicielle. Analyse le pattern CQRS et compare-le.
Donne-moi les avantages, un diagramme ASCII, du code Go, et des métriques de performance.
Réponds en français, en moins de 2000 mots, au format markdown.`

	w := DefaultWeights()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Analyze(prompt, w)
	}
	s := Analyze(prompt, w)
	fmt.Printf("Benchmark result: composite=%.3f level=%s\n", s.Composite, s.Level)
}

package main

import (
	"context"
	"log/slog"
	"slices"
	"strings"

	goanon "github.com/bornholm/go-anon"
	"github.com/bornholm/go-anon/pkg/modelstore"
)

const (
	// LanguageAuto active la détection automatique de la langue, requête par requête.
	LanguageAuto = "auto"
	// defaultLanguage est la langue utilisée par défaut et en dernier recours.
	defaultLanguage = "fr"
	// maxDetectionSample borne la taille du texte soumis au détecteur de langue.
	maxDetectionSample = 4000
)

// supportedLanguages retourne les langues gérées par le pipeline go-anon,
// restreintes à celles pour lesquelles le modelstore publie effectivement un
// modèle. En cas d'échec (hors-ligne, manifest injoignable), la liste complète
// du pipeline est retournée.
func supportedLanguages(ctx context.Context, store *modelstore.Store) []string {
	pipeline := goanon.SupportedLanguages()
	if store == nil {
		return pipeline
	}

	available, err := store.Available(ctx)
	if err != nil {
		slog.WarnContext(ctx, "pseudonymizer: could not list available models, using full language list",
			slog.Any("error", err),
		)
		return pipeline
	}

	langs := make([]string, 0, len(pipeline))
	for _, lang := range pipeline {
		if slices.Contains(available, lang) {
			langs = append(langs, lang)
		}
	}
	if len(langs) == 0 {
		return pipeline
	}
	return langs
}

// detectLanguage détecte la langue de sample parmi candidates. Retourne
// fallback (et false) lorsque la détection est jugée non fiable ou que la
// langue détectée n'est pas exploitable.
func detectLanguage(ctx context.Context, detector goanon.LanguageDetector, sample string, candidates []string, fallback string) (string, bool) {
	if detector == nil || strings.TrimSpace(sample) == "" {
		return fallback, false
	}

	res, err := detector.Detect(sample)
	if err != nil {
		slog.WarnContext(ctx, "pseudonymizer: language detection failed", slog.Any("error", err))
		return fallback, false
	}

	if res.Lang == "" || !res.Reliable || !slices.Contains(candidates, res.Lang) {
		slog.DebugContext(ctx, "pseudonymizer: language detection inconclusive, using fallback",
			slog.String("detected", res.Lang),
			slog.Float64("confidence", res.Confidence),
			slog.Bool("reliable", res.Reliable),
			slog.String("fallback", fallback),
		)
		return fallback, false
	}

	slog.DebugContext(ctx, "pseudonymizer: language detected",
		slog.String("language", res.Lang),
		slog.Float64("confidence", res.Confidence),
	)
	return res.Lang, true
}

// detectionSample construit un échantillon de texte représentatif de la langue
// de la conversation. Les messages utilisateur les plus récents sont prioritaires :
// ce sont eux qui portent la langue réelle de l'échange, les prompts système
// étant souvent rédigés dans une autre langue.
func detectionSample(messages []map[string]any, maxLen int) string {
	var b strings.Builder

	appendText := func(text string) bool {
		text = strings.TrimSpace(text)
		if text == "" {
			return true
		}
		remaining := maxLen - b.Len()
		if remaining <= 0 {
			return false
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
			remaining--
		}
		if len(text) > remaining {
			text = strings.ToValidUTF8(text[:remaining], "")
		}
		b.WriteString(text)
		return b.Len() < maxLen
	}

	// Première passe : messages utilisateur, du plus récent au plus ancien.
	// Seconde passe : les autres messages, si l'échantillon est encore trop court.
	for _, userOnly := range []bool{true, false} {
		for i := len(messages) - 1; i >= 0; i-- {
			role, _ := messages[i]["role"].(string)
			if (role == "user") != userOnly {
				continue
			}
			for _, text := range messageTexts(messages[i]) {
				if !appendText(text) {
					return b.String()
				}
			}
		}
	}

	return b.String()
}

// messageTexts extrait les contenus textuels d'un message, qu'il porte une
// chaîne simple ou un tableau de parties.
func messageTexts(msg map[string]any) []string {
	switch content := msg["content"].(type) {
	case string:
		return []string{content}
	case []any:
		texts := make([]string, 0, len(content))
		for _, part := range content {
			partMap, ok := part.(map[string]any)
			if !ok {
				continue
			}
			if partType, _ := partMap["type"].(string); partType != "text" {
				continue
			}
			if text, ok := partMap["text"].(string); ok {
				texts = append(texts, text)
			}
		}
		return texts
	default:
		return nil
	}
}

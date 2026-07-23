package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Padiwa/go-safe/pkg/dataset"
)

func formatFloat(f float64) string {
	if f == 0 {
		return ""
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func joinLabels(labels []string) string {
	return strings.Join(labels, ",")
}

func formatInt(n int) string {
	return fmt.Sprintf("%d", n)
}

func datasetDocument(text string) dataset.Document {
	return dataset.Document{Text: text}
}

func jsonUnmarshal(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// extractLastUserText lit messages_json et renvoie le texte du dernier
// message user. Retourne "" si introuvable.
func extractLastUserText(messagesJSON string) string {
	if messagesJSON == "" {
		return ""
	}
	var msgs []struct {
		Role    string `json:"role"`
		Content any    `json:"content"`
	}
	if err := json.Unmarshal([]byte(messagesJSON), &msgs); err != nil {
		return ""
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != "user" {
			continue
		}
		switch v := msgs[i].Content.(type) {
		case string:
			return v
		default:
			b, err := json.Marshal(v)
			if err == nil {
				return string(b)
			}
		}
	}
	return ""
}

// extractFromInputs lit inputs_json (objet {portName: value}) et renvoie la
// valeur textuelle du port "request" si elle est une string. Sinon renvoie "".
func extractFromInputs(inputsJSON string) string {
	if inputsJSON == "" {
		return ""
	}
	var inputs map[string]any
	if err := json.Unmarshal([]byte(inputsJSON), &inputs); err != nil {
		return ""
	}
	v, ok := inputs["request"]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// extractText combine les deux sources : d'abord le port "request" du
// pipeline editor (prioritaire si non vide), sinon le dernier message
// user du payload LLM. Cela permet au bloc d'être branché soit sur un
// autre nœud producteur de `request`, soit en direct sur la requête
// OpenAI standard.
func extractText(messagesJSON, inputsJSON string) string {
	if s := extractFromInputs(inputsJSON); s != "" {
		return s
	}
	return extractLastUserText(messagesJSON)
}

// renderRejectionMessage construit le message final renvoyé au client lors
// d'un blocage. Si template == "", retourne un défaut localisé selon la
// langue (fr/en). Placeholders substitués : {label} (label brut),
// {score} et {threshold} en float 2 décimales.
func renderRejectionMessage(template string, language string, label string, score float32, threshold float64) string {
	if template == "" {
		switch language {
		case "en":
			return fmt.Sprintf("Request blocked: classified as %q (confidence: %.0f%%).", label, score*100)
		default:
			return fmt.Sprintf("Requête bloquée : classifiée comme %q (confiance : %.0f%%).", label, score*100)
		}
	}
	s := template
	s = strings.ReplaceAll(s, "{label}", label)
	s = strings.ReplaceAll(s, "{score}", fmt.Sprintf("%.2f", score))
	s = strings.ReplaceAll(s, "{threshold}", fmt.Sprintf("%.2f", threshold))
	return s
}
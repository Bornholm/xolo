package main

import (
	"strings"
)

// GenerateDSL generates a go-fuzzy DSL string from simple signal→tag mappings.
// Each signal gets a DEFINE block and an IF rule. "budget_preference" gets a
// special DEFINE with low/high terms oriented for cost preference.
// Returns an empty string if mappings is empty.
func GenerateDSL(mappings []SimpleMapping) string {
	if len(mappings) == 0 {
		return ""
	}

	var sb strings.Builder

	// Track defined signals to avoid duplicate DEFINE blocks.
	defined := make(map[string]bool)

	for _, m := range mappings {
		if !defined[m.Signal] {
			defined[m.Signal] = true
			if m.Signal == "budget_preference" {
				sb.WriteString("DEFINE budget_preference (\n")
				sb.WriteString("  TERM low LINEAR(0, 3),\n")
				sb.WriteString("  TERM high LINEAR(7, 10)\n")
				sb.WriteString(");\n")
			} else {
				sb.WriteString("DEFINE " + m.Signal + " (\n")
				sb.WriteString("  TERM low LINEAR(0, 0.3),\n")
				sb.WriteString("  TERM high LINEAR(0.5, 1)\n")
				sb.WriteString(");\n")
			}
		}
	}

	sb.WriteString("\n")

	for _, m := range mappings {
		if m.Signal == "budget_preference" {
			sb.WriteString("IF budget_preference IS low THEN tag_" + m.Tag + " IS preferred;\n")
		} else {
			sb.WriteString("IF " + m.Signal + " IS high THEN tag_" + m.Tag + " IS preferred;\n")
		}
	}

	return sb.String()
}

package main

import (
	"fmt"
	"strconv"
)

// formatFloat formats a float64 as a string for HTML input fields.
// Returns empty string for zero to let the placeholder show.
func formatFloat(f float64) string {
	if f == 0 {
		return ""
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// formatInt formats an int as a string for HTML input fields.
// Returns empty string for zero to let the placeholder show.
func formatInt(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d", n)
}

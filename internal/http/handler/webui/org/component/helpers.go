package component

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
)

func displayOwner(users map[model.UserID]model.User, apps map[model.ApplicationID]model.Application, rec model.UsageRecord) string {
	// Check if this is an Application record (has non-empty ApplicationID)
	appID := rec.ApplicationID()
	if appID != "" {
		if app, ok := apps[appID]; ok {
			return app.Name()
		}
		return string(appID)
	}
	// Otherwise it's a User record
	uid := rec.UserID()
	if u, ok := users[uid]; ok {
		return u.DisplayName()
	}
	return string(uid)
}

// formatCostRate formats a pricing rate stored as microcents per 1K tokens,
// displaying it as dollars per million tokens (industry standard).
func formatCostRate(v int64, currency string) string {
	return fmt.Sprintf("%.4f%s/1M", float64(v)/1_000, common.CurrencySymbol(currency))
}

func fmtInt(v int64) string {
	if v == 0 {
		return ""
	}
	return fmt.Sprintf("%d", v)
}

func formatCostField(v int64) string {
	// v is microcents/1K tokens; return dollars per 1M tokens for the form field
	return fmt.Sprintf("%.6f", float64(v)/1_000)
}

// formatActiveParamsBillions converts raw param count to billions for display in the form.
func formatActiveParamsBillions(v int64) string {
	if v <= 0 {
		return ""
	}
	return strconv.FormatFloat(float64(v)/1e9, 'f', -1, 64)
}

func formatBudgetField(v int64) string {
	// v is microcents; convert to the display unit (e.g. EUR/USD) with full precision.
	// Use 'f' format with prec=-1 so strconv uses the minimum number of digits needed
	// to represent the value exactly, avoiding both truncation ("0.00" for 100 µ¢)
	// and unnecessary trailing zeros ("100.000000" for 100_000_000 µ¢).
	return strconv.FormatFloat(float64(v)/1_000_000, 'f', -1, 64)
}

// durationValue returns the numeric component of d in its natural unit (minutes,
// seconds, or milliseconds). Returns "" for a zero duration.
func durationValue(d time.Duration) string {
	if d == 0 {
		return ""
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%d", int64(d/time.Minute))
	}
	if d%time.Second == 0 {
		return fmt.Sprintf("%d", int64(d/time.Second))
	}
	return fmt.Sprintf("%d", int64(d/time.Millisecond))
}

// durationUnit returns "min", "s", or "ms" depending on the most coarse
// whole unit that can represent d. Returns "s" for a zero duration.
func durationUnit(d time.Duration) string {
	if d != 0 && d%time.Minute == 0 {
		return "min"
	}
	if d != 0 && d%time.Second == 0 {
		return "s"
	}
	if d != 0 {
		return "ms"
	}
	return "s"
}

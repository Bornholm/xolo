package component

import (
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/http/handler/webui/templui/component/badge"
)

// explorerHref builds an events-explorer URL, preserving and properly
// URL-encoding the scope, query, page and browsing context (view).
func explorerHref(base, scope, query string, page int, view string) string {
	v := url.Values{}
	if scope != "" {
		v.Set("scope", scope)
	}
	if query != "" {
		v.Set("query", query)
	}
	if page > 1 {
		v.Set("page", strconv.Itoa(page))
	}
	if view != "" {
		v.Set("view", view)
	}
	if len(v) == 0 {
		return base
	}
	return base + "?" + v.Encode()
}

// viewedPath appends the browsing context marker to a simple path.
func viewedPath(path, view string) string {
	if view == "" {
		return path
	}
	return path + "?view=" + view
}

// severityBadgeVariant maps an event severity to a badge variant.
func severityBadgeVariant(sev model.EventSeverity) badge.Variant {
	switch sev {
	case model.SeverityError:
		return badge.VariantDestructive
	case model.SeverityWarning:
		return badge.VariantSecondary
	default:
		return badge.VariantOutline
	}
}

// alertStateBadgeVariant maps an alert state to a badge variant.
func alertStateBadgeVariant(state model.AlertState) badge.Variant {
	switch state {
	case model.AlertStateFiring:
		return badge.VariantDestructive
	case model.AlertStatePending:
		return badge.VariantSecondary
	default:
		return badge.VariantOutline
	}
}

type kv struct {
	Key   string
	Value string
}

// sortedAttrs returns the attributes of an event as a key-sorted slice.
func sortedAttrs(attrs map[string]string) []kv {
	result := make([]kv, 0, len(attrs))
	for k, v := range attrs {
		result = append(result, kv{Key: k, Value: v})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

// formatEventTime renders a timestamp for the event tables.
func formatEventTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05")
}

// formatIncidentTime renders an optional resolved-at timestamp.
func formatOptionalTime(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

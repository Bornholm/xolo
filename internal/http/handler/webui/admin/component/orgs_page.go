package component

import (
	"context"

	common "github.com/xolo-gateway/xolo/internal/http/handler/webui/common/component"
)

// OrgStatusActive and OrgStatusInactive are the values of the `status` query
// parameter backing the segmented filter.
const (
	OrgStatusActive   = "active"
	OrgStatusInactive = "inactive"
)

// orgsSegments builds the filter segments of the organisation list.
//
// "En dépassement" is present but inert: selecting it would mean comparing every
// organisation's spending to its budget, and the console has no cross-org quota
// aggregate (cf. lot 7 du plan). Hiding it would quietly change the information
// architecture of the mockup, so it stays, disabled.
func orgsSegments(ctx context.Context, vmodel OrgsPageVModel) []common.Segment {
	href := func(status string) string {
		values := []string{}
		if status != "" {
			values = append(values, "status", status)
		}
		if vmodel.Search != "" {
			values = append(values, "q", vmodel.Search)
		}
		if len(values) == 0 {
			return common.BaseURLString(ctx, common.WithPath("/admin/orgs"))
		}
		return common.BaseURLString(ctx, common.WithPath("/admin/orgs"), common.WithValues(values...))
	}

	return []common.Segment{
		{Label: "Toutes", Href: href(""), Active: vmodel.Status == ""},
		{Label: "Actives", Href: href(OrgStatusActive), Active: vmodel.Status == OrgStatusActive},
		{Label: "En dépassement", Disabled: true},
		{Label: "Inactives", Href: href(OrgStatusInactive), Active: vmodel.Status == OrgStatusInactive},
	}
}

// orgsSuccessMessage translates the redirect marker of a mutation into the
// sentence the banner shows.
func orgsSuccessMessage(success string) string {
	switch success {
	case "created":
		return "L'organisation a été créée."
	case "saved":
		return "Les modifications ont été enregistrées."
	default:
		return "Opération réussie."
	}
}

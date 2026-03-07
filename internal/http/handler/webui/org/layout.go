package org

import (
	"github.com/a-h/templ"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
)

// orgAdminNav returns the NavigationItems and FooterItems closures for the org
// admin section, using org-specific nav items and the org-admin footer.
func orgAdminNav(orgSlug string) (func(common.AppLayoutVModel) templ.Component, func(common.AppLayoutVModel) templ.Component) {
	nav := func(vmodel common.AppLayoutVModel) templ.Component {
		return component.OrgNavItems(orgSlug, vmodel.SelectedItem)
	}
	footer := func(vmodel common.AppLayoutVModel) templ.Component {
		return common.OrgAdminFooterItems(vmodel)
	}
	return nav, footer
}

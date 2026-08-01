package context

import "context"

// SidebarCookie is the cookie templui's sidebar.min.js writes when the user
// folds or unfolds the rail. Reading it server-side is what makes the choice
// survive a reload: the shell is rendered once per full page load, and without
// it every refresh would come back unfolded.
const SidebarCookie = "sidebar_state"

// SidebarExpanded reports whether the sidebar should be rendered unfolded.
// Unfolded is the default — a first visit carries no cookie.
func SidebarExpanded(ctx context.Context) bool {
	expanded, ok := ctx.Value(keySidebarExpanded).(bool)
	if !ok {
		return true
	}

	return expanded
}

func SetSidebarExpanded(ctx context.Context, expanded bool) context.Context {
	return context.WithValue(ctx, keySidebarExpanded, expanded)
}

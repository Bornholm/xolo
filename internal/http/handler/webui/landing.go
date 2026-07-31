package webui

import (
	"net/url"
	"slices"

	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
)

// landingWithoutOrg names where a user with no organisation belongs.
//
// A plain account has nothing to see and lands on /no-org, which explains the
// emptiness and lists the invitations waiting for it. A platform administrator
// does have somewhere to go — the console, from which organisations are created
// — and sending them to the same dead end leaves the very account that could fix
// the situation with a logout button for only exit. That is the state of a fresh
// install, where the first administrator is precisely the one who has no
// organisation yet.
func landingWithoutOrg(user model.User, baseURL *url.URL) string {
	if user != nil && slices.Contains(user.Roles(), authz.RoleAdmin) {
		return baseURL.JoinPath("/admin/").String()
	}

	return baseURL.JoinPath("/no-org").String()
}

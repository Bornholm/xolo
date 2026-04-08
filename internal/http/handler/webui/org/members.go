package org

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
	"github.com/pkg/errors"
)

func (h *Handler) getMembersPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	const membersPageSize = 20

	page := 0
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 1 {
			page = n - 1
		}
	}

	opts := port.ListOrgMembersOptions{
		Page:  &page,
		Limit: func() *int { l := membersPageSize; return &l }(),
	}

	members, total, err := h.orgStore.ListOrgMembers(ctx, org.ID(), opts)
	if err != nil {
		slog.ErrorContext(ctx, "could not list members", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	vmodel := component.MembersPageVModel{
		Org:          org,
		Members:      members,
		Success:      r.URL.Query().Get("success"),
		CurrentPage:  page + 1,
		PageSize:     membersPageSize,
		TotalMembers: int(total),
		AppLayoutVModel: common.AppLayoutVModel{
			User:          user,
			SelectedItem:  "org-" + orgSlug + "-members",
			HomeLink:      "/orgs/" + orgSlug,
			AdminSubtitle: "Admin. " + org.Name(),
			Breadcrumbs: []common.BreadcrumbItem{
				{Label: org.Name(), Href: "/orgs/" + orgSlug + "/usage"},
				{Label: "Membres", Href: ""},
			},
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}

	templ.Handler(component.MembersPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) deleteMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	membershipID := r.PathValue("membershipID")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	membership, err := h.orgStore.GetMembership(ctx, model.MembershipID(membershipID))
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Membership not found", http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if membership.OrgID() != org.ID() {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.orgStore.RemoveMember(ctx, model.MembershipID(membershipID)); err != nil {
		slog.ErrorContext(ctx, "could not remove member", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orgs/"+orgSlug+"/admin/members?success=removed", http.StatusSeeOther)
}

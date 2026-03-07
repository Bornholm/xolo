package profile

import (
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/a-h/templ"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/crypto"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/profile/component"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
	"github.com/bornholm/go-x/slogx"
)

type Handler struct {
	mux         *http.ServeMux
	userStore   port.UserStore
	orgStore    port.OrgStore
	usageStore  port.UsageStore
	inviteStore port.InviteStore
	quotaStore  port.QuotaStore
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(userStore port.UserStore, orgStore port.OrgStore, usageStore port.UsageStore, inviteStore port.InviteStore, quotaStore port.QuotaStore) *Handler {
	h := &Handler{
		mux:         http.NewServeMux(),
		userStore:   userStore,
		orgStore:    orgStore,
		usageStore:  usageStore,
		inviteStore: inviteStore,
		quotaStore:  quotaStore,
	}

	// Require authentication for all profile routes
	assertUser := authz.Middleware(http.HandlerFunc(h.getForbiddenPage), authz.OneOf(authz.Has(authz.RoleUser), authz.Has(authz.RoleAdmin)))

	h.mux.Handle("GET /", assertUser(http.HandlerFunc(h.getProfilePage)))
	h.mux.Handle("GET /tokens", assertUser(http.HandlerFunc(h.getTokensPage)))
	h.mux.Handle("POST /tokens", assertUser(http.HandlerFunc(h.createToken)))
	h.mux.Handle("DELETE /tokens/{tokenID}", assertUser(http.HandlerFunc(h.deleteToken)))
	h.mux.Handle("POST /preferences", assertUser(http.HandlerFunc(h.updatePreferences)))
	h.mux.Handle("GET /invitations", assertUser(http.HandlerFunc(h.getInvitationsPage)))
	h.mux.Handle("GET /usage", assertUser(http.HandlerFunc(h.getUsagePage)))
	h.mux.Handle("GET /quota", assertUser(http.HandlerFunc(h.getQuotaPage)))

	return h
}

func (h *Handler) getProfilePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	vmodel := component.ProfilePageVModel{
		User:        user,
		Preferences: user.Preferences(),
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "profile",
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}

	profilePage := component.ProfilePage(vmodel)
	templ.Handler(profilePage).ServeHTTP(w, r)
}

func (h *Handler) getTokensPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	// Fetch user's auth tokens
	tokens, err := h.userStore.GetUserAuthTokens(ctx, user.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not fetch user auth tokens", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Fetch user's org memberships
	memberships, err := h.orgStore.GetUserMemberships(ctx, user.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not fetch user memberships", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Check for success messages
	createdToken := r.URL.Query().Get("token_created")
	deletedToken := r.URL.Query().Get("token_deleted")

	vmodel := component.TokensPageVModel{
		AuthTokens:     tokens,
		OrgMemberships: memberships,
		CreatedToken:   createdToken,
		DeletedToken:   deletedToken != "",
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "tokens",
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.TokensPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) updatePreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	// Parse form values
	darkMode := r.FormValue("dark_mode") == "on"

	// Fetch the full user from the database (with preferences loaded)
	existingUser, err := h.userStore.GetUserByID(ctx, user.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not fetch user", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Copy user to get a mutable BaseUser and update preferences
	updatedUser := model.CopyUser(existingUser)
	updatedUser.SetPreferences(model.NewUserPreferences(
		model.SetUserPrefencesDarkMode(darkMode),
	))

	// Save user with updated preferences
	if err := h.userStore.SaveUser(ctx, updatedUser); err != nil {
		slog.ErrorContext(ctx, "could not save user preferences", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("HX-Refresh", "true")

	// Redirect back to profile page
	http.Redirect(w, r, "/profile/", http.StatusSeeOther)
}

func (h *Handler) createToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	label := r.FormValue("label")
	if label == "" {
		http.Error(w, "Le nom du jeton est requis", http.StatusBadRequest)
		return
	}

	orgID := model.OrgID(r.FormValue("org_id"))

	var expiresAt *time.Time
	if expiresStr := r.FormValue("expires_at"); expiresStr != "" {
		t, err := time.Parse("2006-01-02", expiresStr)
		if err == nil {
			expiresAt = &t
		}
	}

	// Generate secure token
	tokenValue, err := crypto.GenerateSecureToken()
	if err != nil {
		slog.ErrorContext(ctx, "could not generate secure token", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	authToken := model.NewAuthToken(user, orgID, label, tokenValue, expiresAt)
	if err := h.userStore.CreateAuthToken(ctx, authToken); err != nil {
		slog.ErrorContext(ctx, "could not create auth token", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Redirect back to tokens page with success message
	http.Redirect(w, r, "/profile/tokens?token_created="+tokenValue, http.StatusSeeOther)
}

func (h *Handler) deleteToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tokenID := r.PathValue("tokenID")
	if tokenID == "" {
		http.Error(w, "Token ID is required", http.StatusBadRequest)
		return
	}

	// Verify the token belongs to the current user
	tokens, err := h.userStore.GetUserAuthTokens(ctx, user.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not fetch user auth tokens", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Check if token belongs to user
	found := false
	for _, token := range tokens {
		if string(token.ID()) == tokenID {
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Token not found", http.StatusNotFound)
		return
	}

	// Delete the token
	if err := h.userStore.DeleteAuthToken(ctx, model.AuthTokenID(tokenID)); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Token not found", http.StatusNotFound)
			return
		}
		slog.ErrorContext(ctx, "could not delete auth token", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Redirect back to tokens page
	http.Redirect(w, r, "/profile/tokens?token_deleted=1", http.StatusSeeOther)
}

func (h *Handler) getInvitationsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	invites, err := h.inviteStore.ListPendingInvitesForEmail(ctx, user.Email())
	if err != nil {
		slog.ErrorContext(ctx, "could not fetch invitations", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	vmodel := component.InvitationsPageVModel{
		Invites: invites,
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "profile",
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.InvitationsPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) getUsagePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	rangeParam := r.URL.Query().Get("range")
	since := profileRangeToSince(rangeParam)

	memberships, err := h.orgStore.GetUserMemberships(ctx, user.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not fetch memberships", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	userID := user.ID()
	portAgg, err := h.usageStore.AggregateUsage(ctx, port.UsageFilter{UserID: &userID, Since: &since})
	if err != nil {
		slog.ErrorContext(ctx, "could not aggregate usage", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var agg *component.UsageAggregate
	if portAgg != nil {
		agg = &component.UsageAggregate{
			TotalRequests:    portAgg.TotalRequests,
			TotalCost:        portAgg.TotalCost,
			Currency:         portAgg.Currency,
			PromptTokens:     portAgg.PromptTokens,
			CompletionTokens: portAgg.CompletionTokens,
			TotalTokens:      portAgg.TotalTokens,
		}
	}

	// Fetch records for chart data
	limit := 500
	chartRecords, _ := h.usageStore.QueryUsage(ctx, port.UsageFilter{
		UserID: &userID,
		Since:  &since,
		Limit:  &limit,
	})

	// Build chart aggregates
	perModel := make(map[string]int64)
	perDay := make(map[string]int64)
	perOrg := make(map[string]int64)
	for _, rec := range chartRecords {
		perModel[rec.ProxyModelName()] += rec.Cost()
		perDay[rec.CreatedAt().Format("2006-01-02")] += rec.Cost()
		orgID := string(rec.OrgID())
		orgName := orgID
		for _, m := range memberships {
			if string(m.OrgID()) == orgID && m.Org() != nil {
				orgName = m.Org().Name()
				break
			}
		}
		perOrg[orgName] += rec.Cost()
	}

	vmodel := component.UsagePageVModel{
		Aggregate:      agg,
		OrgMemberships: memberships,
		Range:          rangeParam,
		ChartPerDay:    profileChartByDate(perDay),
		ChartPerModel:  profileChartByValue(perModel),
		ChartPerOrg:    profileChartByValue(perOrg),
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "profile",
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.UsagePage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) getQuotaPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)

	memberships, err := h.orgStore.GetUserMemberships(ctx, user.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not fetch memberships", slogx.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Resolve quota for each org membership
	orgQuotas := make([]component.OrgEffectiveQuota, 0, len(memberships))
	for _, m := range memberships {
		eq, err := h.quotaStore.ResolveEffectiveQuota(ctx, user.ID(), m.OrgID())
		if err != nil {
			slog.ErrorContext(ctx, "could not resolve quota", slogx.Error(err))
			continue
		}
		orgQuotas = append(orgQuotas, component.OrgEffectiveQuota{
			Membership: m,
			Quota:      eq,
		})
	}

	vmodel := component.QuotaPageVModel{
		OrgQuotas: orgQuotas,
		AppLayoutVModel: common.AppLayoutVModel{
			User:         user,
			SelectedItem: "profile",
			NavigationItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppNavigationItems(vmodel)
			},
			FooterItems: func(vmodel common.AppLayoutVModel) templ.Component {
				return common.AppFooterItems(vmodel)
			},
		},
	}

	templ.Handler(component.QuotaPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) getForbiddenPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vmodel := common.ErrorPageVModel{
		Message: "L'accès à cette page ne vous est pas autorisé. Veuillez contacter l'administrateur.",
		Links: []common.LinkItem{
			{
				URL:   string(common.BaseURL(ctx, common.WithPath("/auth/oidc/logout"))),
				Label: "Se déconnecter",
			},
		},
	}

	forbiddenPage := common.ErrorPage(vmodel)

	w.WriteHeader(http.StatusForbidden)
	templ.Handler(forbiddenPage).ServeHTTP(w, r)
}

func profileChartByValue(m map[string]int64) []component.ProfileChartDataPoint {
	pts := make([]component.ProfileChartDataPoint, 0, len(m))
	for label, cost := range m {
		pts = append(pts, component.ProfileChartDataPoint{Label: label, Value: float64(cost) / 1_000_000})
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].Value > pts[j].Value })
	return pts
}

func profileChartByDate(m map[string]int64) []component.ProfileChartDataPoint {
	dates := make([]string, 0, len(m))
	for k := range m {
		dates = append(dates, k)
	}
	sort.Strings(dates)
	pts := make([]component.ProfileChartDataPoint, 0, len(dates))
	for _, d := range dates {
		pts = append(pts, component.ProfileChartDataPoint{Label: d, Value: float64(m[d]) / 1_000_000})
	}
	return pts
}

func profileRangeToSince(r string) time.Time {
	now := time.Now()
	switch r {
	case "1d":
		return now.AddDate(0, 0, -1)
	case "30d":
		return now.AddDate(0, -1, 0)
	case "90d":
		return now.AddDate(0, -3, 0)
	case "180d":
		return now.AddDate(0, -6, 0)
	case "365d":
		return now.AddDate(-1, 0, 0)
	default: // 7d and anything else
		return now.AddDate(0, 0, -7)
	}
}

var _ http.Handler = &Handler{}

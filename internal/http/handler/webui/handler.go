package webui

import (
	"net/http"
	"strings"

	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/core/service"
	"github.com/bornholm/xolo/internal/http/handler/webui/admin"
	"github.com/bornholm/xolo/internal/http/handler/webui/join"
	"github.com/bornholm/xolo/internal/http/handler/webui/org"
	"github.com/bornholm/xolo/internal/http/handler/webui/profile"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
)

type Handler struct {
	mux                 *http.ServeMux
	inviteStore         port.InviteStore
	quotaStore          port.QuotaStore
	usageStore          port.UsageStore
	userStore           port.UserStore
	orgStore            port.OrgStore
	providerStore       port.ProviderStore
	exchangeRateService *service.ExchangeRateService
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(
	taskRunner port.TaskRunner,
	userStore port.UserStore,
	orgStore port.OrgStore,
	providerStore port.ProviderStore,
	usageStore port.UsageStore,
	inviteStore port.InviteStore,
	quotaStore port.QuotaStore,
	exchangeRateService *service.ExchangeRateService,
	secretKey string,
) *Handler {
	h := &Handler{
		mux:                 http.NewServeMux(),
		inviteStore:         inviteStore,
		quotaStore:          quotaStore,
		usageStore:          usageStore,
		userStore:           userStore,
		orgStore:            orgStore,
		providerStore:       providerStore,
		exchangeRateService: exchangeRateService,
	}

	isActive := authz.Middleware(http.HandlerFunc(h.getInactiveUserPage), authz.Active())

	mount(h.mux, "/", isActive(http.HandlerFunc(h.getHomePage)))
	mount(h.mux, "/no-org", isActive(http.HandlerFunc(h.getNoOrgPage)))
	h.mux.Handle("POST /no-org/invitations/{tokenID}/decline", isActive(http.HandlerFunc(h.declineInvitation)))
	mount(h.mux, "/usage", isActive(http.HandlerFunc(h.getDashboardPage)))
	mount(h.mux, "/models", isActive(http.HandlerFunc(h.getModelsPage)))
	mount(h.mux, "/profile/", isActive(profile.NewHandler(userStore, orgStore, usageStore, inviteStore, quotaStore)))
	mount(h.mux, "/admin/", isActive(admin.NewHandler(userStore, orgStore, taskRunner, exchangeRateService)))
	mount(h.mux, "/orgs/", isActive(org.NewHandler(orgStore, providerStore, usageStore, inviteStore, userStore, exchangeRateService, secretKey)))

	// Public join flow — no isActive wrapper (unauthenticated users get a sign-in prompt)
	h.mux.Handle("/join/", http.StripPrefix("/join", join.NewHandler(orgStore, inviteStore)))

	return h
}

func mount(mux *http.ServeMux, prefix string, handler http.Handler) {
	trimmed := strings.TrimSuffix(prefix, "/")

	if len(trimmed) > 0 {
		mux.Handle(prefix, http.StripPrefix(trimmed, handler))
	} else {
		mux.Handle(prefix, handler)
	}
}

var _ http.Handler = &Handler{}

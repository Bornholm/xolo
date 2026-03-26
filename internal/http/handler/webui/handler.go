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
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

type pluginManagerIface interface {
	List() []*proto.PluginDescriptor
	HTTPPort(pluginName string) uint32
}

type Handler struct {
	mux                   *http.ServeMux
	inviteStore           port.InviteStore
	quotaService          *service.QuotaService
	usageStore            port.UsageStore
	userStore             port.UserStore
	orgStore              port.OrgStore
	providerStore         port.ProviderStore
	virtualModelStore     port.VirtualModelStore
	exchangeRateService   *service.ExchangeRateService
	pluginManager         pluginManagerIface
	pluginActivationStore port.PluginActivationStore
	pluginConfigStore     port.PluginConfigStore
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
	virtualModelStore port.VirtualModelStore,
	usageStore port.UsageStore,
	inviteStore port.InviteStore,
	quotaStore port.QuotaStore,
	quotaService *service.QuotaService,
	exchangeRateService *service.ExchangeRateService,
	secretKey string,
	pluginManager pluginManagerIface,
	pluginActivationStore port.PluginActivationStore,
	pluginConfigStore port.PluginConfigStore,
) *Handler {
	h := &Handler{
		mux:                   http.NewServeMux(),
		inviteStore:           inviteStore,
		quotaService:          quotaService,
		usageStore:            usageStore,
		userStore:             userStore,
		orgStore:              orgStore,
		providerStore:         providerStore,
		virtualModelStore:     virtualModelStore,
		exchangeRateService:   exchangeRateService,
		pluginManager:         pluginManager,
		pluginActivationStore: pluginActivationStore,
		pluginConfigStore:     pluginConfigStore,
	}

	isActive := authz.Middleware(http.HandlerFunc(h.getInactiveUserPage), authz.Active())

	mount(h.mux, "/", isActive(http.HandlerFunc(h.getHomePage)))
	mount(h.mux, "/no-org", isActive(http.HandlerFunc(h.getNoOrgPage)))
	h.mux.Handle("POST /no-org/invitations/{tokenID}/decline", isActive(http.HandlerFunc(h.declineInvitation)))
	mount(h.mux, "/usage", isActive(http.HandlerFunc(h.getDashboardPage)))
	mount(h.mux, "/models", isActive(http.HandlerFunc(h.getModelsPage)))
	mount(h.mux, "/profile/", isActive(profile.NewHandler(userStore, orgStore, inviteStore)))
	mount(h.mux, "/admin/", isActive(admin.NewHandler(userStore, orgStore, taskRunner, exchangeRateService, pluginManager)))
	mount(h.mux, "/orgs/", isActive(org.NewHandler(orgStore, providerStore, virtualModelStore, usageStore, inviteStore, userStore, exchangeRateService, quotaStore, secretKey, pluginManager, pluginActivationStore, pluginConfigStore)))

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

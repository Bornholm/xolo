package org

import (
	"net/http"

	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/core/service"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

type pluginManagerIface interface {
	List() []*proto.PluginDescriptor
	HTTPPort(pluginName string) uint32
}

// Handler serves the org-admin section: /orgs/{orgSlug}/admin/
type Handler struct {
	mux                   *http.ServeMux
	orgStore              port.OrgStore
	providerStore         port.ProviderStore
	virtualModelStore     port.VirtualModelStore
	usageStore            port.UsageStore
	inviteStore           port.InviteStore
	userStore             port.UserStore
	quotaStore            port.QuotaStore
	secretKey             string
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
	orgStore port.OrgStore,
	providerStore port.ProviderStore,
	virtualModelStore port.VirtualModelStore,
	usageStore port.UsageStore,
	inviteStore port.InviteStore,
	userStore port.UserStore,
	exchangeRateService *service.ExchangeRateService,
	quotaStore port.QuotaStore,
	secretKey string,
	pluginManager pluginManagerIface,
	pluginActivationStore port.PluginActivationStore,
	pluginConfigStore port.PluginConfigStore,
) *Handler {
	h := &Handler{
		mux:                   http.NewServeMux(),
		orgStore:              orgStore,
		providerStore:         providerStore,
		virtualModelStore:     virtualModelStore,
		usageStore:            usageStore,
		inviteStore:           inviteStore,
		userStore:             userStore,
		quotaStore:            quotaStore,
		secretKey:             secretKey,
		exchangeRateService:   exchangeRateService,
		pluginManager:         pluginManager,
		pluginActivationStore: pluginActivationStore,
		pluginConfigStore:     pluginConfigStore,
	}

	assertOrgAdmin := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgSlug := r.PathValue("orgSlug")
			authz.Middleware(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Forbidden", http.StatusForbidden)
				}),
				h.hasOrgAdminRole(orgSlug),
			)(next).ServeHTTP(w, r)
		})
	}

	assertOrgMember := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgSlug := r.PathValue("orgSlug")
			authz.Middleware(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Forbidden", http.StatusForbidden)
				}),
				h.hasOrgMembership(orgSlug),
			)(next).ServeHTTP(w, r)
		})
	}

	// Org admin routes — redirect /{orgSlug}/admin/ to /{orgSlug}/usage
	h.mux.Handle("GET /{orgSlug}/admin/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgSlug := r.PathValue("orgSlug")
		http.Redirect(w, r, "/orgs/"+orgSlug+"/usage", http.StatusMovedPermanently)
	}))
	// Redirect /{orgSlug} to /{orgSlug}/usage
	h.mux.Handle("GET /{orgSlug}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgSlug := r.PathValue("orgSlug")
		http.Redirect(w, r, "/orgs/"+orgSlug+"/usage", http.StatusMovedPermanently)
	}))
	h.mux.Handle("GET /{orgSlug}/admin/members", assertOrgAdmin(http.HandlerFunc(h.getMembersPage)))
	h.mux.Handle("DELETE /{orgSlug}/admin/members/{membershipID}", assertOrgAdmin(http.HandlerFunc(h.deleteMember)))

	h.mux.Handle("GET /{orgSlug}/admin/providers", assertOrgAdmin(http.HandlerFunc(h.getProvidersPage)))
	h.mux.Handle("GET /{orgSlug}/admin/providers/new", assertOrgAdmin(http.HandlerFunc(h.getNewProviderPage)))
	h.mux.Handle("POST /{orgSlug}/admin/providers", assertOrgAdmin(http.HandlerFunc(h.createProvider)))
	h.mux.Handle("GET /{orgSlug}/admin/providers/{providerID}/edit", assertOrgAdmin(http.HandlerFunc(h.getEditProviderPage)))
	h.mux.Handle("POST /{orgSlug}/admin/providers/{providerID}/edit", assertOrgAdmin(http.HandlerFunc(h.updateProvider)))
	h.mux.Handle("DELETE /{orgSlug}/admin/providers/{providerID}", assertOrgAdmin(http.HandlerFunc(h.deleteProvider)))
	h.mux.Handle("POST /{orgSlug}/admin/providers/{providerID}/test", assertOrgAdmin(http.HandlerFunc(h.testProvider)))

	h.mux.Handle("GET /{orgSlug}/admin/providers/{providerID}/models", assertOrgAdmin(http.HandlerFunc(h.getModelsPage)))
	h.mux.Handle("GET /{orgSlug}/admin/providers/{providerID}/models/new", assertOrgAdmin(http.HandlerFunc(h.getNewModelPage)))
	h.mux.Handle("POST /{orgSlug}/admin/providers/{providerID}/models", assertOrgAdmin(http.HandlerFunc(h.createModel)))
	h.mux.Handle("GET /{orgSlug}/admin/providers/{providerID}/models/{modelID}/edit", assertOrgAdmin(http.HandlerFunc(h.getEditModelPage)))
	h.mux.Handle("POST /{orgSlug}/admin/providers/{providerID}/models/{modelID}/edit", assertOrgAdmin(http.HandlerFunc(h.updateModel)))
	h.mux.Handle("DELETE /{orgSlug}/admin/providers/{providerID}/models/{modelID}", assertOrgAdmin(http.HandlerFunc(h.deleteModel)))

	h.mux.Handle("GET /{orgSlug}/admin/quota", assertOrgAdmin(http.HandlerFunc(h.getOrgQuotaPage)))
	h.mux.Handle("POST /{orgSlug}/admin/quota", assertOrgAdmin(http.HandlerFunc(h.saveOrgQuota)))
	h.mux.Handle("GET /{orgSlug}/admin/members/{membershipID}/quota", assertOrgAdmin(http.HandlerFunc(h.getMemberQuotaPage)))
	h.mux.Handle("POST /{orgSlug}/admin/members/{membershipID}/quota", assertOrgAdmin(http.HandlerFunc(h.saveMemberQuota)))

	h.mux.Handle("GET /{orgSlug}/admin/invites", assertOrgAdmin(http.HandlerFunc(h.getInvitesPage)))
	h.mux.Handle("GET /{orgSlug}/admin/invites/new", assertOrgAdmin(http.HandlerFunc(h.getNewInvitePage)))
	h.mux.Handle("POST /{orgSlug}/admin/invites", assertOrgAdmin(http.HandlerFunc(h.createInvite)))
	h.mux.Handle("DELETE /{orgSlug}/admin/invites/{inviteID}", assertOrgAdmin(http.HandlerFunc(h.revokeInvite)))

	h.mux.Handle("GET /{orgSlug}/usage", assertOrgAdmin(http.HandlerFunc(h.getUsagePage)))

	h.mux.Handle("GET /{orgSlug}/admin/settings", assertOrgAdmin(http.HandlerFunc(h.getSettingsPage)))
	h.mux.Handle("POST /{orgSlug}/admin/settings", assertOrgAdmin(http.HandlerFunc(h.saveSettings)))

	h.mux.Handle("GET /{orgSlug}/admin/virtual-models", assertOrgAdmin(http.HandlerFunc(h.getVirtualModelsPage)))
	h.mux.Handle("GET /{orgSlug}/admin/virtual-models/new", assertOrgAdmin(http.HandlerFunc(h.getNewVirtualModelPage)))
	h.mux.Handle("POST /{orgSlug}/admin/virtual-models", assertOrgAdmin(http.HandlerFunc(h.createVirtualModel)))
	h.mux.Handle("GET /{orgSlug}/admin/virtual-models/{modelID}/edit", assertOrgAdmin(http.HandlerFunc(h.getEditVirtualModelPage)))
	h.mux.Handle("POST /{orgSlug}/admin/virtual-models/{modelID}/edit", assertOrgAdmin(http.HandlerFunc(h.updateVirtualModel)))
	h.mux.Handle("DELETE /{orgSlug}/admin/virtual-models/{modelID}", assertOrgAdmin(http.HandlerFunc(h.deleteVirtualModel)))

	h.mux.Handle("GET /{orgSlug}/admin/plugins", assertOrgAdmin(http.HandlerFunc(h.getPluginsPage)))
	h.mux.Handle("POST /{orgSlug}/admin/plugins/{pluginName}/activate", assertOrgAdmin(http.HandlerFunc(h.postActivatePlugin)))
	h.mux.Handle("POST /{orgSlug}/admin/plugins/{pluginName}/deactivate", assertOrgAdmin(http.HandlerFunc(h.postDeactivatePlugin)))
	h.mux.Handle("GET /{orgSlug}/admin/plugins/{pluginName}/config", assertOrgAdmin(http.HandlerFunc(h.getPluginConfigPage)))
	h.mux.Handle("POST /{orgSlug}/admin/plugins/{pluginName}/config", assertOrgAdmin(http.HandlerFunc(h.postPluginConfig)))
	h.mux.Handle("/{orgSlug}/plugins/{pluginName}/ui/{uiPath...}", assertOrgAdmin(http.HandlerFunc(h.servePluginUI)))

	// Org member routes
	_ = assertOrgMember

	return h
}

var _ http.Handler = &Handler{}

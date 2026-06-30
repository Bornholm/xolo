package org

import (
	"net/http"

	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/core/rbac"
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
	mux                  *http.ServeMux
	orgStore             port.OrgStore
	roleStore            port.RoleStore
	providerStore        port.ProviderStore
	virtualModelStore    port.VirtualModelStore
	usageStore           port.UsageStore
	inviteStore          port.InviteStore
	userStore            port.UserStore
	applicationStore     port.ApplicationStore
	quotaStore           port.QuotaStore
	secretStore          port.SecretStore
	secretKey            string
	exchangeRateService  *service.ExchangeRateService
	pluginManager        pluginManagerIface
	subscriptionMonitor  port.SubscriptionMonitor
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(
	orgStore port.OrgStore,
	roleStore port.RoleStore,
	providerStore port.ProviderStore,
	virtualModelStore port.VirtualModelStore,
	usageStore port.UsageStore,
	inviteStore port.InviteStore,
	userStore port.UserStore,
	applicationStore port.ApplicationStore,
	exchangeRateService *service.ExchangeRateService,
	quotaStore port.QuotaStore,
	secretStore port.SecretStore,
	secretKey string,
	pluginManager pluginManagerIface,
	subscriptionMonitor port.SubscriptionMonitor,
) *Handler {
	h := &Handler{
		mux:                  http.NewServeMux(),
		orgStore:             orgStore,
		roleStore:            roleStore,
		providerStore:        providerStore,
		virtualModelStore:    virtualModelStore,
		usageStore:           usageStore,
		inviteStore:          inviteStore,
		userStore:            userStore,
		applicationStore:     applicationStore,
		quotaStore:           quotaStore,
		secretStore:          secretStore,
		secretKey:            secretKey,
		exchangeRateService:  exchangeRateService,
		pluginManager:        pluginManager,
		subscriptionMonitor:  subscriptionMonitor,
	}

	// assertPerm gates a route on a single RBAC permission resolved for the
	// org identified by the {orgSlug} path value.
	assertPerm := func(perm rbac.Permission) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				orgSlug := r.PathValue("orgSlug")
				authz.Middleware(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						http.Error(w, "Forbidden", http.StatusForbidden)
					}),
					h.hasPermission(orgSlug, perm),
				)(next).ServeHTTP(w, r)
			})
		}
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

	h.mux.Handle("GET /{orgSlug}/admin/members", assertPerm(rbac.PermMembersRead)(http.HandlerFunc(h.getMembersPage)))
	h.mux.Handle("DELETE /{orgSlug}/admin/members/{membershipID}", assertPerm(rbac.PermMembersWrite)(http.HandlerFunc(h.deleteMember)))
	h.mux.Handle("GET /{orgSlug}/admin/members/{membershipID}/edit", assertPerm(rbac.PermMembersRead)(http.HandlerFunc(h.getEditMemberPage)))
	h.mux.Handle("POST /{orgSlug}/admin/members/{membershipID}/edit", assertPerm(rbac.PermMembersWrite)(http.HandlerFunc(h.postEditMember)))

	h.mux.Handle("GET /{orgSlug}/admin/roles", assertPerm(rbac.PermRolesRead)(http.HandlerFunc(h.getRolesPage)))
	h.mux.Handle("GET /{orgSlug}/admin/roles/new", assertPerm(rbac.PermRolesWrite)(http.HandlerFunc(h.getNewRolePage)))
	h.mux.Handle("POST /{orgSlug}/admin/roles", assertPerm(rbac.PermRolesWrite)(http.HandlerFunc(h.createRole)))
	h.mux.Handle("GET /{orgSlug}/admin/roles/{roleID}/edit", assertPerm(rbac.PermRolesRead)(http.HandlerFunc(h.getEditRolePage)))
	h.mux.Handle("POST /{orgSlug}/admin/roles/{roleID}/edit", assertPerm(rbac.PermRolesWrite)(http.HandlerFunc(h.updateRole)))
	h.mux.Handle("DELETE /{orgSlug}/admin/roles/{roleID}", assertPerm(rbac.PermRolesWrite)(http.HandlerFunc(h.deleteRole)))

	h.mux.Handle("GET /{orgSlug}/admin/providers", assertPerm(rbac.PermProvidersRead)(http.HandlerFunc(h.getProvidersPage)))
	h.mux.Handle("GET /{orgSlug}/admin/providers/new", assertPerm(rbac.PermProvidersWrite)(http.HandlerFunc(h.getNewProviderPage)))
	h.mux.Handle("POST /{orgSlug}/admin/providers", assertPerm(rbac.PermProvidersWrite)(http.HandlerFunc(h.createProvider)))
	h.mux.Handle("GET /{orgSlug}/admin/providers/{providerID}/edit", assertPerm(rbac.PermProvidersRead)(http.HandlerFunc(h.getEditProviderPage)))
	h.mux.Handle("POST /{orgSlug}/admin/providers/{providerID}/edit", assertPerm(rbac.PermProvidersWrite)(http.HandlerFunc(h.updateProvider)))
	h.mux.Handle("DELETE /{orgSlug}/admin/providers/{providerID}", assertPerm(rbac.PermProvidersWrite)(http.HandlerFunc(h.deleteProvider)))
	h.mux.Handle("POST /{orgSlug}/admin/providers/{providerID}/test", assertPerm(rbac.PermProvidersWrite)(http.HandlerFunc(h.testProvider)))

	h.mux.Handle("GET /{orgSlug}/admin/providers/{providerID}/models", assertPerm(rbac.PermProvidersRead)(http.HandlerFunc(h.getModelsPage)))
	h.mux.Handle("GET /{orgSlug}/admin/providers/{providerID}/models/new", assertPerm(rbac.PermProvidersWrite)(http.HandlerFunc(h.getNewModelPage)))
	h.mux.Handle("POST /{orgSlug}/admin/providers/{providerID}/models", assertPerm(rbac.PermProvidersWrite)(http.HandlerFunc(h.createModel)))
	h.mux.Handle("GET /{orgSlug}/admin/providers/{providerID}/models/{modelID}/edit", assertPerm(rbac.PermProvidersRead)(http.HandlerFunc(h.getEditModelPage)))
	h.mux.Handle("POST /{orgSlug}/admin/providers/{providerID}/models/{modelID}/edit", assertPerm(rbac.PermProvidersWrite)(http.HandlerFunc(h.updateModel)))
	h.mux.Handle("DELETE /{orgSlug}/admin/providers/{providerID}/models/{modelID}", assertPerm(rbac.PermProvidersWrite)(http.HandlerFunc(h.deleteModel)))

	h.mux.Handle("GET /{orgSlug}/admin/quota", assertPerm(rbac.PermQuotaRead)(http.HandlerFunc(h.getOrgQuotaPage)))
	h.mux.Handle("POST /{orgSlug}/admin/quota", assertPerm(rbac.PermQuotaWrite)(http.HandlerFunc(h.saveOrgQuota)))
	h.mux.Handle("GET /{orgSlug}/admin/members/{membershipID}/quota", assertPerm(rbac.PermQuotaRead)(http.HandlerFunc(h.getMemberQuotaPage)))
	h.mux.Handle("POST /{orgSlug}/admin/members/{membershipID}/quota", assertPerm(rbac.PermQuotaWrite)(http.HandlerFunc(h.saveMemberQuota)))

	h.mux.Handle("GET /{orgSlug}/admin/invites", assertPerm(rbac.PermInvitesRead)(http.HandlerFunc(h.getInvitesPage)))
	h.mux.Handle("GET /{orgSlug}/admin/invites/new", assertPerm(rbac.PermInvitesWrite)(http.HandlerFunc(h.getNewInvitePage)))
	h.mux.Handle("POST /{orgSlug}/admin/invites", assertPerm(rbac.PermInvitesWrite)(http.HandlerFunc(h.createInvite)))
	h.mux.Handle("DELETE /{orgSlug}/admin/invites/{inviteID}", assertPerm(rbac.PermInvitesWrite)(http.HandlerFunc(h.revokeInvite)))
	h.mux.Handle("DELETE /{orgSlug}/admin/invites/{inviteID}/delete", assertPerm(rbac.PermInvitesWrite)(http.HandlerFunc(h.deleteInvite)))

	h.mux.Handle("GET /{orgSlug}/usage", assertPerm(rbac.PermUsageRead)(http.HandlerFunc(h.getUsagePage)))

	h.mux.Handle("GET /{orgSlug}/admin/settings", assertPerm(rbac.PermSettingsRead)(http.HandlerFunc(h.getSettingsPage)))
	h.mux.Handle("POST /{orgSlug}/admin/settings", assertPerm(rbac.PermSettingsWrite)(http.HandlerFunc(h.saveSettings)))

	h.mux.Handle("GET /{orgSlug}/admin/virtual-models", assertPerm(rbac.PermVirtualModelsRead)(http.HandlerFunc(h.getVirtualModelsPage)))
	h.mux.Handle("GET /{orgSlug}/admin/virtual-models/new", assertPerm(rbac.PermVirtualModelsWrite)(http.HandlerFunc(h.getNewVirtualModelPage)))
	h.mux.Handle("POST /{orgSlug}/admin/virtual-models", assertPerm(rbac.PermVirtualModelsWrite)(http.HandlerFunc(h.createVirtualModel)))
	h.mux.Handle("GET /{orgSlug}/admin/virtual-models/{modelID}/edit", assertPerm(rbac.PermVirtualModelsRead)(http.HandlerFunc(h.getEditVirtualModelPage)))
	h.mux.Handle("POST /{orgSlug}/admin/virtual-models/{modelID}/edit", assertPerm(rbac.PermVirtualModelsWrite)(http.HandlerFunc(h.updateVirtualModel)))
	h.mux.Handle("DELETE /{orgSlug}/admin/virtual-models/{modelID}", assertPerm(rbac.PermVirtualModelsWrite)(http.HandlerFunc(h.deleteVirtualModel)))
	h.mux.Handle("GET /{orgSlug}/admin/virtual-models/{modelID}/pipeline", assertPerm(rbac.PermVirtualModelsRead)(http.HandlerFunc(h.getPipelineEditorPage)))

	h.mux.Handle("GET /{orgSlug}/admin/applications", assertPerm(rbac.PermApplicationsRead)(http.HandlerFunc(h.getApplicationsPage)))
	h.mux.Handle("GET /{orgSlug}/admin/applications/new", assertPerm(rbac.PermApplicationsWrite)(http.HandlerFunc(h.getNewApplicationPage)))
	h.mux.Handle("POST /{orgSlug}/admin/applications", assertPerm(rbac.PermApplicationsWrite)(http.HandlerFunc(h.createApplication)))
	h.mux.Handle("GET /{orgSlug}/admin/applications/{appID}/edit", assertPerm(rbac.PermApplicationsRead)(http.HandlerFunc(h.getEditApplicationPage)))
	h.mux.Handle("POST /{orgSlug}/admin/applications/{appID}/edit", assertPerm(rbac.PermApplicationsWrite)(http.HandlerFunc(h.updateApplication)))
	h.mux.Handle("POST /{orgSlug}/admin/applications/{appID}/delete", assertPerm(rbac.PermApplicationsWrite)(http.HandlerFunc(h.deleteApplication)))
	h.mux.Handle("POST /{orgSlug}/admin/applications/{appID}/tokens", assertPerm(rbac.PermApplicationsWrite)(http.HandlerFunc(h.createApplicationToken)))
	h.mux.Handle("POST /{orgSlug}/admin/applications/{appID}/tokens/{tokenID}/delete", assertPerm(rbac.PermApplicationsWrite)(http.HandlerFunc(h.deleteApplicationToken)))

	// Plugin HTTP UI proxy — tied to virtual model / pipeline administration.
	h.mux.Handle("/{orgSlug}/plugins/{pluginName}/ui/{uiPath...}", assertPerm(rbac.PermVirtualModelsWrite)(http.HandlerFunc(h.servePluginUI)))

	// Org member routes
	_ = assertOrgMember

	return h
}

var _ http.Handler = &Handler{}

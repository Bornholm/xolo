package admin

import (
	"net/http"

	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/core/service"
	"github.com/bornholm/xolo/internal/http/middleware/authz"
)

type Handler struct {
	mux                 *http.ServeMux
	userStore           port.UserStore
	orgStore            port.OrgStore
	taskRunner          port.TaskRunner
	exchangeRateService *service.ExchangeRateService
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(userStore port.UserStore, orgStore port.OrgStore, taskRunner port.TaskRunner, exchangeRateService *service.ExchangeRateService) *Handler {
	h := &Handler{
		mux:                 http.NewServeMux(),
		userStore:           userStore,
		orgStore:            orgStore,
		taskRunner:          taskRunner,
		exchangeRateService: exchangeRateService,
	}

	// Admin middleware - only allow admin users
	assertAdmin := authz.Middleware(http.HandlerFunc(h.getForbiddenPage), authz.Has(authz.RoleAdmin))

	h.mux.Handle("GET /", assertAdmin(http.HandlerFunc(h.getIndexPage)))

	// User routes
	h.mux.Handle("GET /users", assertAdmin(http.HandlerFunc(h.getUsersPage)))
	h.mux.Handle("GET /users/{id}/edit", assertAdmin(http.HandlerFunc(h.getEditUserPage)))
	h.mux.Handle("POST /users/{id}/edit", assertAdmin(http.HandlerFunc(h.postEditUser)))

	// Org routes
	h.mux.Handle("GET /orgs", assertAdmin(http.HandlerFunc(h.getOrgsPage)))
	h.mux.Handle("GET /orgs/new", assertAdmin(http.HandlerFunc(h.getNewOrgPage)))
	h.mux.Handle("POST /orgs", assertAdmin(http.HandlerFunc(h.createOrg)))
	h.mux.Handle("GET /orgs/{orgID}/edit", assertAdmin(http.HandlerFunc(h.getEditOrgPage)))
	h.mux.Handle("POST /orgs/{orgID}/edit", assertAdmin(http.HandlerFunc(h.updateOrg)))

	// Exchange rate routes
	h.mux.Handle("GET /exchange-rates", assertAdmin(http.HandlerFunc(h.getExchangeRatesPage)))

	return h
}

func (h *Handler) getForbiddenPage(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Forbidden", http.StatusForbidden)
}

var _ http.Handler = &Handler{}

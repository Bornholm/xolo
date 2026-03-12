package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	proxyAdapter "github.com/bornholm/xolo/internal/adapter/proxy"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/core/service"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/go-x/slogx"
)

type Handler struct {
	providerStore       port.ProviderStore
	orgStore            port.OrgStore
	exchangeRateService *service.ExchangeRateService
	mux                 *http.ServeMux
}

func NewHandler(providerStore port.ProviderStore, orgStore port.OrgStore, exchangeRateService *service.ExchangeRateService) *Handler {
	h := &Handler{
		providerStore:       providerStore,
		orgStore:            orgStore,
		exchangeRateService: exchangeRateService,
		mux:                 http.NewServeMux(),
	}
	h.mux.HandleFunc("GET /api/v1/models", h.handleModels)
	h.mux.HandleFunc("GET /api/models-dev/lookup", h.handleModelsDevLookup)
	h.mux.HandleFunc("GET /api/exchange-rate", h.handleExchangeRate)
	return h
}

type exchangeRateResponse struct {
	From string  `json:"from"`
	To   string  `json:"to"`
	Rate float64 `json:"rate"`
}

func (h *Handler) handleExchangeRate(w http.ResponseWriter, r *http.Request) {
	from := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("from")))
	to := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("to")))
	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "missing from or to parameter")
		return
	}
	if from == to {
		writeJSON(w, http.StatusOK, exchangeRateResponse{From: from, To: to, Rate: 1})
		return
	}
	// Convert 1_000_000 microcents to get the rate as a float64.
	converted, err := h.exchangeRateService.Convert(r.Context(), 1_000_000, from, to)
	if err != nil {
		slog.WarnContext(r.Context(), "exchange rate unavailable", slog.String("from", from), slog.String("to", to), slog.Any("error", err))
		writeError(w, http.StatusServiceUnavailable, "exchange rate unavailable")
		return
	}
	writeJSON(w, http.StatusOK, exchangeRateResponse{From: from, To: to, Rate: float64(converted) / 1_000_000})
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type modelObj struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type modelsResponse struct {
	Object string     `json:"object"`
	Data   []modelObj `json:"data"`
}

func (h *Handler) handleModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var orgIDs []model.OrgID

	if orgID := model.OrgID(proxyAdapter.OrgIDFromContext(ctx)); orgID != "" {
		// Token is org-scoped: list only models for that org.
		orgIDs = []model.OrgID{orgID}
	} else {
		// Token is not org-scoped: list models for all orgs the user belongs to.
		user := httpCtx.User(ctx)
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		memberships, err := h.orgStore.GetUserMemberships(ctx, user.ID())
		if err != nil {
			slog.ErrorContext(ctx, "could not fetch user memberships", slogx.Error(err))
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		for _, m := range memberships {
			orgIDs = append(orgIDs, m.OrgID())
		}
	}

	data := make([]modelObj, 0)
	for _, orgID := range orgIDs {
		org, err := h.orgStore.GetOrgByID(ctx, orgID)
		if err != nil {
			slog.WarnContext(ctx, "could not get org for models listing", slogx.Error(err), slog.String("orgID", string(orgID)))
			continue
		}
		models, err := h.providerStore.ListEnabledLLMModels(ctx, orgID)
		if err != nil {
			slog.WarnContext(ctx, "could not list enabled models", slogx.Error(err), slog.String("orgID", string(orgID)))
			continue
		}
		for _, m := range models {
			data = append(data, modelObj{
				ID:      org.Slug() + "/" + m.ProxyName(),
				Object:  "model",
				Created: 0,
				OwnedBy: "xolo",
			})
		}
	}

	writeJSON(w, http.StatusOK, modelsResponse{
		Object: "list",
		Data:   data,
	})
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("could not write JSON response", slog.Any("error", err))
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	var resp apiError
	resp.Error.Message = msg
	resp.Error.Type = "invalid_request_error"
	writeJSON(w, status, resp)
}

package setup

import (
	"context"

	"github.com/bornholm/genai/proxy"
	proxyAdapter "github.com/bornholm/xolo/internal/adapter/proxy"
	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/http"
	"github.com/bornholm/xolo/internal/http/handler/api"
	"github.com/bornholm/xolo/internal/http/handler/metrics"
	"github.com/bornholm/xolo/internal/http/handler/webui"
	"github.com/bornholm/xolo/internal/http/handler/webui/common"
	"github.com/bornholm/xolo/internal/http/middleware/authn"
	membershipsMiddleware "github.com/bornholm/xolo/internal/http/middleware/memberships"
	"github.com/bornholm/xolo/internal/http/middleware/ratelimit"
	"github.com/pkg/errors"

	gohttp "net/http"
)


func NewHTTPServerFromConfig(ctx context.Context, conf *config.Config) (*http.Server, error) {
	oidcAuthn, err := getOIDCAuthnHandlerFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not configure authn oidc handler from config")
	}

	tokenAuthn, err := getTokenAuthnHandlerFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not configure authn token handler from config")
	}

	authnMiddleware := authn.Middleware(
		func(w gohttp.ResponseWriter, r *gohttp.Request) {
			// By default, redirect to OIDC login page if no user has been found
			gohttp.Redirect(w, r, "/auth/oidc/login", gohttp.StatusSeeOther)
		},
		tokenAuthn,
		oidcAuthn,
	)

	bridgeMiddleware, err := getBridgeMiddlewareFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not configure bridge middleware from config")
	}

	authChain := func(h gohttp.Handler) gohttp.Handler {
		return authnMiddleware(bridgeMiddleware(h))
	}

	assets := common.NewHandler()

	rateLimiter := ratelimit.Middleware(
		conf.HTTP.RateLimit.TrustHeaders,
		conf.HTTP.RateLimit.RequestInterval,
		conf.HTTP.RateLimit.RequestMaxBurst,
		conf.HTTP.RateLimit.CacheSize,
		conf.HTTP.RateLimit.CacheTTL,
	)

	taskRunner, err := getTaskRunner(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not create task runner from config")
	}

	userStore, err := getUserStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not create user store from config")
	}

	orgStore, err := getOrgStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not create org store from config")
	}

	providerStore, err := getProviderStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not create provider store from config")
	}

	quotaStore, err := getQuotaStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not create quota store from config")
	}

	quotaService, err := getQuotaServiceFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not create quota service from config")
	}

	usageStore, err := getUsageStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not create usage store from config")
	}

	inviteStore, err := getInviteStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not create invite store from config")
	}

	exchangeRateService, err := getExchangeRateServiceFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not create exchange rate service from config")
	}
	exchangeRateService.StartRefresher(ctx, model.SupportedCurrencies, conf.ExchangeRate.RefreshInterval)

	withMemberships := membershipsMiddleware.Middleware(orgStore)

	webuiHandler := webui.NewHandler(taskRunner, userStore, orgStore, providerStore, usageStore, inviteStore, quotaStore, quotaService, exchangeRateService, conf.SecretKey)

	apiHandler := api.NewHandler(providerStore, orgStore, exchangeRateService)

	proxyServer := proxy.NewServer(
		proxy.WithAuthExtractor(proxyAdapter.XoloAuthExtractor(userStore)),
		proxy.WithHook(proxyAdapter.NewOrgModelRouter(providerStore, orgStore, conf.SecretKey)),
		proxy.WithHook(proxyAdapter.NewXoloQuotaEnforcer(quotaService, quotaStore, usageStore, userStore)),
		proxy.WithHook(proxyAdapter.NewXoloUsageTracker(usageStore, providerStore, orgStore, exchangeRateService)),
	)

	options := []http.OptionFunc{
		http.WithAddress(conf.HTTP.Address),
		http.WithBaseURL(conf.HTTP.BaseURL),
		http.WithMount("/assets/", assets),
		http.WithMount("/auth/oidc/", rateLimiter(oidcAuthn)),
		http.WithMount("/auth/token/", rateLimiter(tokenAuthn)),
		http.WithMount("/metrics/", rateLimiter(authChain(metrics.NewHandler()))),
		http.WithMount("/api/v1/", rateLimiter(authChain(proxyServer))),
		http.WithRoute("GET /api/v1/models", rateLimiter(authChain(apiHandler))),
		http.WithRoute("GET /api/models-dev/lookup", rateLimiter(authChain(apiHandler))),
		http.WithRoute("GET /api/exchange-rate", rateLimiter(authChain(apiHandler))),
		http.WithMount("/", authChain(withMemberships(webuiHandler))),
	}

	server := http.NewServer(options...)

	return server, nil
}

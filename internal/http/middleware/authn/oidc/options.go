package oidc

import "github.com/bornholm/xolo/internal/http/middleware/authn/oidc/component"

type Provider = component.Provider

type Options struct {
	Providers        []component.Provider
	ProvidersWithJWKS []ProviderWithJWKS
	SessionName     string
}

type OptionFunc func(opts *Options)

func NewOptions(funcs ...OptionFunc) *Options {
	opts := &Options{
		Providers:   make([]Provider, 0),
		SessionName: "xolo_auth_oidc",
	}

	for _, fn := range funcs {
		fn(opts)
	}

	return opts
}

func WithProviders(providers ...Provider) OptionFunc {
	return func(opts *Options) {
		opts.Providers = providers
	}
}

func WithSessionName(sessionName string) OptionFunc {
	return func(opts *Options) {
		opts.SessionName = sessionName
	}
}

func WithProvidersWithJWKS(providers []ProviderWithJWKS) OptionFunc {
	return func(opts *Options) {
	(opts).ProvidersWithJWKS = providers
	}
}

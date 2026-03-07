package http

import (
	"net/http"

	"github.com/rs/cors"
)

type Options struct {
	Address string
	BaseURL string
	Mounts  map[string]http.Handler
	Routes  map[string]http.Handler
	CORS    cors.Options
}

type OptionFunc func(opts *Options)

func NewOptions(funcs ...OptionFunc) *Options {
	opts := &Options{
		Address: ":3002",
		BaseURL: "",
		Mounts:  map[string]http.Handler{},
		Routes:  map[string]http.Handler{},
		CORS: cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
			Debug:            false,
		},
	}
	for _, fn := range funcs {
		fn(opts)
	}
	return opts
}

func WithMount(prefix string, handler http.Handler) OptionFunc {
	return func(opts *Options) {
		opts.Mounts[prefix] = handler
	}
}

// WithRoute registers a handler for an exact pattern (method + path), e.g.
// "GET /api/v1/models". Unlike WithMount, no path stripping is applied, and
// the pattern takes precedence over any prefix mount that would also match.
func WithRoute(pattern string, handler http.Handler) OptionFunc {
	return func(opts *Options) {
		opts.Routes[pattern] = handler
	}
}

func WithBaseURL(baseURL string) OptionFunc {
	return func(opts *Options) {
		opts.BaseURL = baseURL
	}
}

func WithAddress(addr string) OptionFunc {
	return func(opts *Options) {
		opts.Address = addr
	}
}

func WithCORS(options cors.Options) OptionFunc {
	return func(opts *Options) {
		opts.CORS = options
	}
}

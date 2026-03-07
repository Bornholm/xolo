package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/cors"
	sloghttp "github.com/samber/slog-http"

	httpCtx "github.com/bornholm/xolo/internal/http/context"
)

type Server struct {
	opts *Options
}

func (s *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := &http.ServeMux{}
	for mountpoint, handler := range s.opts.Mounts {
		mount(mux, mountpoint, handler)
	}

	handler := sloghttp.Recovery(mux)
	handler = sloghttp.New(slog.Default())(handler)

	cors := cors.New(s.opts.CORS)

	handler = cors.Handler(handler)

	handler = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ctx = httpCtx.SetBaseURL(ctx, s.opts.BaseURL)
			ctx = httpCtx.SetCurrentURL(ctx, r.URL)

			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}(handler)

	server := http.Server{
		Addr:    s.opts.Address,
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		if err := server.Close(); err != nil {
			slog.ErrorContext(ctx, "could not close server", slog.Any("error", errors.WithStack(err)))
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.WithStack(err)
	}

	return nil
}

func mount(mux *http.ServeMux, prefix string, handler http.Handler) {
	trimmed := strings.TrimSuffix(prefix, "/")

	if len(trimmed) > 0 {
		mux.Handle(prefix, http.StripPrefix(trimmed, handler))
	} else {
		mux.Handle(prefix, handler)
	}
}

func NewServer(funcs ...OptionFunc) *Server {
	opts := NewOptions(funcs...)
	return &Server{
		opts: opts,
	}
}

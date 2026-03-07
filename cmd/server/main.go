package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/bornholm/go-x/slogx"
	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/setup"
	"github.com/pkg/errors"

	// Adapters
	_ "github.com/bornholm/xolo/internal/adapter/memory"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf, err := config.Parse()
	if err != nil {
		slog.ErrorContext(ctx, "could not parse config", slog.Any("error", errors.WithStack(err)))
		os.Exit(1)
	}

	logger := slog.New(slogx.ContextHandler{
		Handler: slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.Level(conf.Logger.Level),
			AddSource: true,
		}),
	})

	slog.SetDefault(logger)

	slog.DebugContext(ctx, "using configuration", slog.Any("config", conf))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	go func() {
		slog.InfoContext(ctx, "use ctrl+c to interrupt")
		<-sig
		cancel()
	}()

	server, err := setup.NewHTTPServerFromConfig(ctx, conf)
	if err != nil {
		slog.ErrorContext(ctx, "could not setup http server", slog.Any("error", errors.WithStack(err)))
		os.Exit(1)
	}

	slog.InfoContext(ctx, "starting server", slog.Any("address", conf.HTTP.Address))

	if err := server.Run(ctx); err != nil {
		slog.Error("could not run server", slog.Any("error", errors.WithStack(err)))
		os.Exit(1)
	}
}

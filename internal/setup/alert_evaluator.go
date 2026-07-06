package setup

import (
	"context"
	"log/slog"

	"github.com/bornholm/xolo/internal/adapter/eventeval"
	"github.com/bornholm/xolo/internal/config"
	"github.com/pkg/errors"
)

// getAlertEvaluatorFromConfig creates the periodic alert evaluator and starts its
// background loop. Created at most once per config.
var getAlertEvaluatorFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (*eventeval.Evaluator, error) {
	alertStore, err := getAlertStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	incidentStore, err := getAlertIncidentStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	eventStore, err := getEventStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	evaluator := eventeval.NewEvaluator(alertStore, incidentStore, eventStore, conf.Events.EvaluationInterval)

	go func() {
		evalCtx := context.Background()
		if err := evaluator.Run(evalCtx); err != nil {
			slog.ErrorContext(evalCtx, "alert evaluator stopped", slog.Any("error", errors.WithStack(err)))
		}
	}()

	return evaluator, nil
})

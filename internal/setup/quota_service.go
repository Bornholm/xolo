package setup

import (
	"context"

	"github.com/bornholm/xolo/internal/config"
	"github.com/bornholm/xolo/internal/core/service"
	"github.com/pkg/errors"
)

var getQuotaServiceFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (*service.QuotaService, error) {
	quotaStore, err := getQuotaStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	orgStore, err := getOrgStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return service.NewQuotaService(quotaStore, orgStore), nil
})

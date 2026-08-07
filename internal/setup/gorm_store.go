package setup

import (
	"context"

	gormAdapter "github.com/xolo-gateway/xolo/internal/adapter/gorm"
	"github.com/xolo-gateway/xolo/internal/config"
	"github.com/pkg/errors"
)

var getGormStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (*gormAdapter.Store, error) {
	db, err := getGormDatabaseFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return gormAdapter.NewStore(db), nil
})

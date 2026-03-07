package memory

import (
	"net/url"
	"strconv"
	"time"

	"github.com/bornholm/xolo/internal/core/port"
	"github.com/bornholm/xolo/internal/setup"
	"github.com/pkg/errors"
)

func init() {
	setup.TaskRunner.Register("memory", func(u *url.URL) (port.TaskRunner, error) {
		parallelism := 100
		if rawValue := u.Query().Get("parallelism"); rawValue != "" {
			v, err := strconv.ParseInt(rawValue, 10, 32)
			if err != nil {
				return nil, errors.Wrapf(err, "could not parse 'parallelism' parameter")
			}
			parallelism = int(v)
		}

		cleanupDelay := time.Minute * 60
		if rawValue := u.Query().Get("cleanupDelay"); rawValue != "" {
			v, err := time.ParseDuration(rawValue)
			if err != nil {
				return nil, errors.Wrapf(err, "could not parse 'cleanupDelay' parameter")
			}
			cleanupDelay = v
		}

		cleanupInterval := time.Minute * 10
		if rawValue := u.Query().Get("cleanupInterval"); rawValue != "" {
			v, err := time.ParseDuration(rawValue)
			if err != nil {
				return nil, errors.Wrapf(err, "could not parse 'cleanupInterval' parameter")
			}
			cleanupInterval = v
		}

		return NewTaskRunner(parallelism, cleanupDelay, cleanupInterval), nil
	})
}

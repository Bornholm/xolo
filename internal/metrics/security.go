package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	NameAuthFailures          = "auth_failures_total"
	NameRateLimitRejections   = "rate_limit_rejections_total"
)

var AuthFailures = promauto.NewCounter(
	prometheus.CounterOpts{
		Name:      NameAuthFailures,
		Help:      "Total number of authentication failures",
		Namespace: Namespace,
	},
)

var RateLimitRejections = promauto.NewCounter(
	prometheus.CounterOpts{
		Name:      NameRateLimitRejections,
		Help:      "Total number of requests rejected by the rate limiter",
		Namespace: Namespace,
	},
)

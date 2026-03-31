package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	NameProxyRequestDuration = "proxy_request_duration_seconds"
	NameProxyErrors          = "proxy_errors_total"
	LabelModel               = "model"
)

var ProxyRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:      NameProxyRequestDuration,
		Help:      "Duration of LLM upstream requests in seconds",
		Namespace: Namespace,
		Buckets:   []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120},
	},
	[]string{LabelOrg, LabelModel},
)

var ProxyErrors = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:      NameProxyErrors,
		Help:      "Total number of failed LLM upstream requests",
		Namespace: Namespace,
	},
	[]string{LabelOrg},
)

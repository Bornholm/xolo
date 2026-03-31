package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	NameHTTPRequestsTotal        = "http_requests_total"
	NameHTTPRequestDuration      = "http_request_duration_seconds"
	NameHTTPRequestsInFlight     = "http_requests_in_flight"
	LabelMethod                  = "method"
	LabelRoute                   = "route"
	LabelStatusCode              = "status_code"
)

var HTTPRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name:      NameHTTPRequestsTotal,
		Help:      "Total number of HTTP requests",
		Namespace: Namespace,
	},
	[]string{LabelMethod, LabelRoute, LabelStatusCode},
)

var HTTPRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:      NameHTTPRequestDuration,
		Help:      "HTTP request duration in seconds",
		Namespace: Namespace,
		Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	},
	[]string{LabelMethod, LabelRoute},
)

var HTTPRequestsInFlight = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name:      NameHTTPRequestsInFlight,
		Help:      "Number of HTTP requests currently being processed",
		Namespace: Namespace,
	},
)

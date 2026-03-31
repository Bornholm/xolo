package httpmetrics

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bornholm/xolo/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	reUUID    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	reNumeric = regexp.MustCompile(`^\d+$`)
)

// normalizePath remplace les segments UUID et numériques par {id} et tronque à 4 niveaux.
func normalizePath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 4 {
		parts = parts[:4]
	}
	for i, p := range parts {
		if reUUID.MatchString(p) || reNumeric.MatchString(p) {
			parts[i] = "{id}"
		}
	}
	return "/" + strings.Join(parts, "/")
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Middleware enregistre les métriques HTTP (total, durée, in-flight) pour chaque requête.
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := normalizePath(r.URL.Path)
			method := r.Method

			metrics.HTTPRequestsInFlight.Inc()
			defer metrics.HTTPRequestsInFlight.Dec()

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(rec, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(rec.status)

			metrics.HTTPRequestsTotal.With(prometheus.Labels{
				metrics.LabelMethod:     method,
				metrics.LabelRoute:      route,
				metrics.LabelStatusCode: status,
			}).Inc()

			metrics.HTTPRequestDuration.With(prometheus.Labels{
				metrics.LabelMethod: method,
				metrics.LabelRoute:  route,
			}).Observe(duration)
		})
	}
}

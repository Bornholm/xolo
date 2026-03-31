package proxy

import (
	"context"
	"time"

	genaiProxy "github.com/bornholm/genai/proxy"
	"github.com/bornholm/xolo/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

const metaMetricsStart = "_metrics_start"

// XoloMetricsHook mesure la latence des appels LLM upstream et compte les erreurs proxy.
type XoloMetricsHook struct{}

func NewXoloMetricsHook() *XoloMetricsHook {
	return &XoloMetricsHook{}
}

func (h *XoloMetricsHook) Name() string  { return "xolo.metrics" }
func (h *XoloMetricsHook) Priority() int { return 0 }

// PreRequest implémente proxy.PreRequestHook : stocke le timestamp de début.
func (h *XoloMetricsHook) PreRequest(ctx context.Context, req *genaiProxy.ProxyRequest) (*genaiProxy.HookResult, error) {
	req.Metadata[metaMetricsStart] = time.Now()
	return nil, nil
}

// PostResponse implémente proxy.PostResponseHook : enregistre durée ou erreur.
func (h *XoloMetricsHook) PostResponse(ctx context.Context, req *genaiProxy.ProxyRequest, res *genaiProxy.ProxyResponse) (*genaiProxy.HookResult, error) {
	PopulateMetaFromContext(ctx, req.Metadata)

	orgID := string(OrgIDFromMeta(req.Metadata))
	if orgID == "" {
		orgID = "unknown"
	}

	if res.TokensUsed == nil {
		metrics.ProxyErrors.With(prometheus.Labels{
			metrics.LabelOrg: orgID,
		}).Inc()
		return nil, nil
	}

	startVal, ok := req.Metadata[metaMetricsStart]
	if !ok {
		return nil, nil
	}
	start, ok := startVal.(time.Time)
	if !ok {
		return nil, nil
	}

	modelID := string(ModelIDFromMeta(req.Metadata))
	if modelID == "" {
		modelID = req.Model
	}

	metrics.ProxyRequestDuration.With(prometheus.Labels{
		metrics.LabelOrg:   orgID,
		metrics.LabelModel: modelID,
	}).Observe(time.Since(start).Seconds())

	return nil, nil
}

var _ genaiProxy.PreRequestHook = &XoloMetricsHook{}
var _ genaiProxy.PostResponseHook = &XoloMetricsHook{}

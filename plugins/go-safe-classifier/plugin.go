package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	gosafe "github.com/Padiwa/go-safe"
	"github.com/Padiwa/go-safe/pkg/classifier"
	"github.com/Padiwa/go-safe/pkg/modelstore"
	"github.com/Padiwa/go-safe/pkg/vectorizer"
	"github.com/bornholm/xolo/pkg/pluginsdk"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// Plugin implémente proto.XoloPluginServer et bloque les requêtes LLM dont
// le contenu est classé comme sensible par un classifieur de la famille
// `go-safe`.
type Plugin struct {
	proto.UnimplementedXoloPluginServer

	mu         sync.Mutex
	store      *modelstore.Store
	classifier *safetyClassifier
	lastCfgKey string

	hostMu     sync.Mutex
	hostClient pluginsdk.HostClient
}

// SetHostClient implémente pluginsdk.HostClientSetter : le runtime l'appelle
// une fois la connexion au XoloHostService établie (dans Initialize).
func (p *Plugin) SetHostClient(c pluginsdk.HostClient) {
	p.hostMu.Lock()
	defer p.hostMu.Unlock()
	p.hostClient = c
}

func (p *Plugin) getHostClient() pluginsdk.HostClient {
	p.hostMu.Lock()
	defer p.hostMu.Unlock()
	return p.hostClient
}

type safetyClassifier struct {
	vec     *vectorizer.NgramVectorizer
	clf     *classifier.LogisticRegression
	labels  []string
	labelID map[string]int
}

func newPlugin() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:        "go-safe-classifier",
		Version:     "0.1.0",
		Description: "Bloque les requêtes LLM sensibles via un classifieur téléchargé automatiquement (modèle safety-fr par défaut).",
		Capabilities: []proto.PluginDescriptor_Capability{
			proto.PluginDescriptor_PRE_REQUEST,
		},
		InputPorts: []*proto.PortDescriptor{
			{Name: "request", PortType: "request", Required: true},
		},
		OutputPorts: []*proto.PortDescriptor{
			{Name: "request", PortType: "request"},
		},
		ConfigSchema:    configSchemaJSON,
		DefaultRequired: false,
	}, nil
}

func (p *Plugin) PreRequest(ctx context.Context, in *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
	cfg, err := parseConfig(in.GetCtx().GetConfigJson())
	if err != nil {
		slog.WarnContext(ctx, "go-safe-classifier: config error, passing through", slog.Any("error", err))
		return &proto.PreRequestOutput{Allowed: true}, nil
	}

	sc, err := p.getClassifier(ctx, cfg)
	if err != nil {
		slog.WarnContext(ctx, "go-safe-classifier: failed to initialize classifier", slog.Any("error", err))
		if cfg.FailureMode == "block" {
			return &proto.PreRequestOutput{Allowed: false, RejectionReason: "classifier unavailable"}, nil
		}
		return &proto.PreRequestOutput{Allowed: true}, nil
	}

	text := extractText(in.GetMessagesJson(), in.GetInputsJson())
	if text == "" {
		return &proto.PreRequestOutput{Allowed: true}, nil
	}

	doc := datasetDocument(text)
	x := sc.vec.Transform(doc)
	idx, score := sc.clf.Predict(x)
	if idx < 0 || idx >= len(sc.labels) {
		return &proto.PreRequestOutput{Allowed: true}, nil
	}

	label := sc.labels[idx]
	slog.DebugContext(ctx, "go-safe-classifier: classification done",
		slog.String("label", label),
		slog.Float64("score", float64(score)),
		slog.String("threshold", fmt.Sprintf("%.2f", cfg.Threshold)),
	)

	if !containsString(cfg.LabelsToBlock, label) {
		return &proto.PreRequestOutput{Allowed: true}, nil
	}
	if float64(score) < cfg.Threshold {
		return &proto.PreRequestOutput{Allowed: true}, nil
	}

	p.emitSafetyDetectedEvent(in, label, score, cfg)

	msg := renderRejectionMessage(cfg.RejectionMessage, cfg.Language, label, score, cfg.Threshold)
	return &proto.PreRequestOutput{
		Allowed:         false,
		RejectionReason: msg,
	}, nil
}

// getClassifier retourne (en créant si besoin) le classifieur correspondant à
// la configuration. Recrée le store et le classifieur quand la config change.
func (p *Plugin) getClassifier(ctx context.Context, cfg Config) (*safetyClassifier, error) {
	cfgKey := buildCfgKey(cfg)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.lastCfgKey == cfgKey && p.classifier != nil {
		return p.classifier, nil
	}

	opts := []modelstore.Option{}
	if cfg.CacheDir != "" {
		opts = append(opts, modelstore.WithCacheDir(cfg.CacheDir))
	}
	if cfg.ManifestURL != "" {
		opts = append(opts, modelstore.WithManifestURL(cfg.ManifestURL))
	}
	if cfg.Offline {
		opts = append(opts, modelstore.WithOfflineMode(true))
	}

	store, err := modelstore.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create model store: %w", err)
	}

	modelName := cfg.Model
	if modelName == "" {
		modelName = "safety-fr"
	}

	var path string
	if modelName == "auto" {
		path, err = store.Resolve(ctx, "auto", "")
	} else {
		path, err = store.Get(ctx, modelName)
	}
	if err != nil {
		return nil, fmt.Errorf("load model %q: %w", modelName, err)
	}

	m, err := gosafe.LoadModel(path)
	if err != nil {
		return nil, fmt.Errorf("decode model file: %w", err)
	}

	ft := gosafe.FromEmbeddings(m.Embeddings)
	vec := gosafe.NewNgramVectorizer(ft, len(m.NgramVocab))
	vec.Vocab = m.NgramVocab

	clf := &classifier.LogisticRegression{
		Weights:    m.Weights,
		Bias:       m.Bias,
		Labels:     m.Labels,
		NumClasses: len(m.Labels),
		Dim:        m.EmbeddingDim + len(m.NgramVocab),
	}

	labelID := make(map[string]int, len(m.Labels))
	for i, l := range m.Labels {
		labelID[l] = i
	}

	p.store = store
	p.classifier = &safetyClassifier{
		vec:     vec,
		clf:     clf,
		labels:  m.Labels,
		labelID: labelID,
	}
	p.lastCfgKey = cfgKey
	return p.classifier, nil
}

func buildCfgKey(cfg Config) string {
	b, _ := json.Marshal(cfg)
	return string(b)
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
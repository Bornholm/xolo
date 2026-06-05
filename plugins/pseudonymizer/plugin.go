package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	goanon "github.com/bornholm/go-anon"
	"github.com/bornholm/go-anon/pkg/anonymizer"
	"github.com/bornholm/go-anon/pkg/modelstore"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// Plugin implémente proto.XoloPluginServer.
type Plugin struct {
	proto.UnimplementedXoloPluginServer

	mu         sync.Mutex
	store      *modelstore.Store
	anons      map[string]*anonymizer.Anonymizer
	lastCfgKey string
}

func newPlugin() *Plugin {
	return &Plugin{
		anons: make(map[string]*anonymizer.Anonymizer),
	}
}

func (p *Plugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:        "pseudonymizer",
		Version:     "0.1.0",
		Description: "Anonymise les messages avant le LLM et rétablit les données originales dans la réponse.",
		Capabilities: []proto.PluginDescriptor_Capability{
			proto.PluginDescriptor_PRE_REQUEST,
			proto.PluginDescriptor_POST_RESPONSE,
		},
		InputPorts: []*proto.PortDescriptor{
			{Name: "request", PortType: "request", Required: true},
		},
		OutputPorts: []*proto.PortDescriptor{
			{Name: "request", PortType: "request"},
		},
		ConfigSchema:    configSchemaJSON,
		DefaultRequired: true,
	}, nil
}

func (p *Plugin) PreRequest(ctx context.Context, in *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
	cfg, err := parseConfig(in.GetCtx().GetConfigJson())
	if err != nil {
		slog.WarnContext(ctx, "pseudonymizer: config error, passing through", slog.Any("error", err))
		return &proto.PreRequestOutput{Allowed: true}, nil
	}

	anon, err := p.getAnonymizer(ctx, cfg)
	if err != nil {
		slog.WarnContext(ctx, "pseudonymizer: failed to initialize anonymizer, passing through", slog.Any("error", err))
		return &proto.PreRequestOutput{Allowed: true}, nil
	}

	// Resolve request body: prefer from input port, fall back to Model (full request body JSON).
	requestJSON := ""
	if in.InputsJson != "" {
		var inputs map[string]any
		if err := json.Unmarshal([]byte(in.InputsJson), &inputs); err == nil {
			if v, ok := inputs["request"].(string); ok {
				requestJSON = v
			}
		}
	}
	if requestJSON == "" {
		requestJSON = in.Model
	}

	// Parse full request body to modify the messages field and rebuild it.
	var requestBody map[string]any
	if requestJSON != "" {
		if err := json.Unmarshal([]byte(requestJSON), &requestBody); err != nil {
			slog.WarnContext(ctx, "pseudonymizer: failed to parse request body, passing through", slog.Any("error", err))
			return &proto.PreRequestOutput{Allowed: true}, nil
		}
	}

	// Resolve messages JSON.
	messagesJSON := in.GetMessagesJson()
	if messagesJSON == "" && requestBody != nil {
		if msgs, ok := requestBody["messages"]; ok {
			if b, err := json.Marshal(msgs); err == nil {
				messagesJSON = string(b)
			}
		}
	}
	if messagesJSON == "" {
		return &proto.PreRequestOutput{Allowed: true}, nil
	}

	var messages []map[string]any
	if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
		slog.WarnContext(ctx, "pseudonymizer: failed to parse messages, passing through", slog.Any("error", err))
		return &proto.PreRequestOutput{Allowed: true}, nil
	}

	slog.DebugContext(ctx, "pseudonymizer: anonymizing messages",
		slog.Int("count", len(messages)),
		slog.String("language", cfg.Language),
		slog.String("strategy", cfg.Strategy),
	)

	// Anonymize all text content using a shared session for consistent numbering.
	// Non-anonymizable attachments (documents, files…) are removed and tracked.
	session := anonymizer.NewSession()
	var removedParts []removedPart

	filtered := make([]map[string]any, 0, len(messages))
	for i, msg := range messages {
		role, _ := msg["role"].(string)
		content, ok := msg["content"]
		if !ok {
			filtered = append(filtered, messages[i])
			continue
		}
		switch c := content.(type) {
		case string:
			result, err := anon.Anonymize(c, anonymizer.WithSession(session))
			if err != nil {
				slog.WarnContext(ctx, "pseudonymizer: failed to anonymize string content", slog.Any("error", err))
			} else {
				messages[i]["content"] = result.Text
			}
			filtered = append(filtered, messages[i])
		case []any:
			var kept []any
			for _, part := range c {
				partMap, ok := part.(map[string]any)
				if !ok {
					kept = append(kept, part)
					continue
				}
				partType, _ := partMap["type"].(string)
				switch {
				case partType == "text":
					text, _ := partMap["text"].(string)
					result, err := anon.Anonymize(text, anonymizer.WithSession(session))
					if err != nil {
						slog.WarnContext(ctx, "pseudonymizer: failed to anonymize text part", slog.Any("error", err))
						kept = append(kept, part)
					} else {
						updated := make(map[string]any, len(partMap))
						for k, v := range partMap {
							updated[k] = v
						}
						updated["text"] = result.Text
						kept = append(kept, updated)
					}
				default:
					// Document, file, or unknown attachment: remove and track.
					name := partName(partMap)
					removedParts = append(removedParts, removedPart{
						Role: role,
						Type: partType,
						Name: name,
					})
					slog.DebugContext(ctx, "pseudonymizer: removed non-anonymizable attachment",
						slog.String("role", role),
						slog.String("type", partType),
						slog.String("name", name),
					)
				}
			}
			// Drop the message entirely if all its parts were removed.
			if len(kept) > 0 {
				messages[i]["content"] = kept
				filtered = append(filtered, messages[i])
			}
		default:
			filtered = append(filtered, messages[i])
		}
	}

	// Serialize anonymized+filtered messages.
	modifiedMessagesJSON, err := json.Marshal(filtered)
	if err != nil {
		slog.WarnContext(ctx, "pseudonymizer: failed to marshal anonymized messages, passing through", slog.Any("error", err))
		return &proto.PreRequestOutput{Allowed: true}, nil
	}

	// Rebuild the full request body with filtered messages for the output port.
	var outputsJSON string
	if requestBody != nil {
		requestBody["messages"] = filtered
		if b, err := json.Marshal(requestBody); err == nil {
			outputs := map[string]any{"request": string(b)}
			if ob, err := json.Marshal(outputs); err == nil {
				outputsJSON = string(ob)
			}
		}
	}

	// Serialize full state (mapping + removed parts) for PostResponse.
	state := pluginState{
		Mapping:      session.Mapping,
		RemovedParts: removedParts,
	}
	stateJSON, err := json.Marshal(state)
	if err != nil {
		slog.WarnContext(ctx, "pseudonymizer: failed to marshal state", slog.Any("error", err))
		stateJSON = []byte("{}")
	}

	slog.DebugContext(ctx, "pseudonymizer: anonymization done",
		slog.Int("entities", len(session.Mapping)),
		slog.Int("removed_attachments", len(removedParts)),
	)

	return &proto.PreRequestOutput{
		Allowed:              true,
		ModifiedMessagesJson: string(modifiedMessagesJSON),
		OutputsJson:          outputsJSON,
		NodeState:            stateJSON,
	}, nil
}

func (p *Plugin) PostResponse(_ context.Context, in *proto.PostResponseInput) (*proto.PostResponseOutput, error) {
	if len(in.NodeState) == 0 {
		return &proto.PostResponseOutput{}, nil
	}

	var state pluginState
	if err := json.Unmarshal(in.NodeState, &state); err != nil {
		return &proto.PostResponseOutput{}, nil
	}

	restored := deanonymize(in.ResponseContent, state.Mapping)

	slog.Debug("pseudonymizer: deanonymization done",
		slog.Int("entities", len(state.Mapping)),
		slog.Bool("modified", restored != in.ResponseContent),
		slog.Int("removed_attachments", len(state.RemovedParts)),
	)

	if len(state.RemovedParts) > 0 {
		restored = removedPartsWarning(state.RemovedParts) + restored
	}

	return &proto.PostResponseOutput{ModifiedResponseContent: restored}, nil
}

// pluginState is the opaque blob stored in node_state between PreRequest and PostResponse.
type pluginState struct {
	Mapping      map[string]string `json:"mapping"`
	RemovedParts []removedPart     `json:"removed_parts,omitempty"`
}

// removedPart describes a content part that was stripped from the request
// because the plugin cannot anonymize it.
type removedPart struct {
	Role string `json:"role"`
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// partName extracts a human-readable name from a content part map, checking
// common fields used by various providers (Anthropic, OpenAI…).
func partName(partMap map[string]any) string {
	for _, key := range []string{"title", "name", "filename", "file_id"} {
		if v, ok := partMap[key].(string); ok && v != "" {
			return v
		}
	}
	// Nested source object (Anthropic document format).
	if src, ok := partMap["source"].(map[string]any); ok {
		for _, key := range []string{"name", "filename"} {
			if v, ok := src[key].(string); ok && v != "" {
				return v
			}
		}
	}
	// Nested file object (OpenAI file format).
	if file, ok := partMap["file"].(map[string]any); ok {
		for _, key := range []string{"name", "filename", "file_id"} {
			if v, ok := file[key].(string); ok && v != "" {
				return v
			}
		}
	}
	return ""
}

// removedPartsWarning builds a markdown warning block to prepend to the
// LLM response when attachments were stripped from the request.
func removedPartsWarning(parts []removedPart) string {
	var b strings.Builder
	b.WriteString("> ⚠️ **Avertissement pseudonymiseur** : les pièces jointes suivantes n'ont pas pu être traitées par le filtre d'anonymisation et ont été automatiquement retirées de la requête :\n")
	for _, p := range parts {
		if p.Name != "" {
			fmt.Fprintf(&b, "> - **%s** (type : `%s`", p.Name, p.Type)
		} else {
			fmt.Fprintf(&b, "> - *pièce jointe sans nom* (type : `%s`", p.Type)
		}
		if p.Role != "" {
			fmt.Fprintf(&b, ", rôle : %s", p.Role)
		}
		b.WriteString(")\n")
	}
	b.WriteString("\n")
	return b.String()
}

// deanonymize replaces all placeholders in text with their original values from mapping.
func deanonymize(text string, mapping map[string]string) string {
	result := text
	for placeholder, original := range mapping {
		result = strings.ReplaceAll(result, placeholder, original)
	}
	return result
}

// getAnonymizer returns (creating if needed) the Anonymizer for cfg.Language.
// Thread-safe; recreates the store and anonymizers when config changes.
func (p *Plugin) getAnonymizer(ctx context.Context, cfg Config) (*anonymizer.Anonymizer, error) {
	cfgKey := buildCfgKey(cfg)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.lastCfgKey != cfgKey {
		opts := storeOptionsFromConfig(cfg)
		store, err := modelstore.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("create model store: %w", err)
		}
		p.store = store
		p.anons = make(map[string]*anonymizer.Anonymizer)
		p.lastCfgKey = cfgKey
	}

	if anon, ok := p.anons[cfg.Language]; ok {
		return anon, nil
	}

	anon, err := p.buildAnonymizer(ctx, cfg)
	if err != nil {
		return nil, err
	}
	p.anons[cfg.Language] = anon
	return anon, nil
}

// buildAnonymizer downloads the model and creates a new Anonymizer for cfg.Language.
// Must be called with p.mu held.
func (p *Plugin) buildAnonymizer(ctx context.Context, cfg Config) (*anonymizer.Anonymizer, error) {
	modelPath, err := p.store.Get(ctx, cfg.Language)
	if err != nil {
		return nil, fmt.Errorf("get model for language %q: %w", cfg.Language, err)
	}

	f, err := os.Open(modelPath)
	if err != nil {
		return nil, fmt.Errorf("open model file: %w", err)
	}
	defer f.Close()

	model, err := goanon.LoadModel(f)
	if err != nil {
		return nil, fmt.Errorf("load model: %w", err)
	}

	// Load gazetteers (may be empty if not available).
	gazs, _ := p.store.GetGazetteers(ctx, cfg.Language)
	loadedGazs, firstnamesGaz := loadGazetteers(ctx, gazs)

	// Build recognizer options in the correct filter application order.
	var recOpts []goanon.RecognizerOption
	recOpts = append(recOpts, goanon.WithLanguage(cfg.Language))

	if cfg.BuiltinRegexPatterns {
		recOpts = append(recOpts, goanon.WithBuiltinRegexPatterns())
	}
	if cfg.BuiltinSecretPatterns {
		recOpts = append(recOpts, goanon.WithBuiltinSecretPatterns())
	}
	if len(loadedGazs) > 0 {
		recOpts = append(recOpts, goanon.WithGazetteers(loadedGazs))
	}

	// Post-filters: pruning first, then structural passes.
	var postFilters []goanon.EntityFilter
	if cfg.MinConfidence > 0 {
		postFilters = append(postFilters, goanon.MinConfidenceFilter(cfg.MinConfidence))
	}
	if cfg.MaxTokens > 0 {
		postFilters = append(postFilters, goanon.MaxTokensFilter(cfg.MaxTokens))
	}
	for typeStr, words := range cfg.Blocklist {
		if len(words) > 0 {
			postFilters = append(postFilters, goanon.BlocklistFilter(goanon.EntityType(typeStr), words...))
		}
	}
	if len(postFilters) > 0 {
		recOpts = append(recOpts, goanon.WithPostFilters(postFilters...))
	}
	if cfg.FirstNameReclassify && firstnamesGaz != nil {
		recOpts = append(recOpts, goanon.WithFirstNameReclassify(firstnamesGaz))
	}
	if cfg.Merge {
		recOpts = append(recOpts, goanon.WithMergePass())
	}
	if cfg.NameCompletion && firstnamesGaz != nil {
		recOpts = append(recOpts, goanon.WithNameCompletionPass(firstnamesGaz))
	}
	// Mirror server behaviour: WithFirstNameDetectionPass is applied alongside
	// WithFirstNameReclassify to detect first names not covered by the NER model.
	if cfg.FirstNameReclassify && firstnamesGaz != nil {
		recOpts = append(recOpts, goanon.WithFirstNameDetectionPass(firstnamesGaz))
	}

	rec, err := goanon.NewRecognizer(model, recOpts...)
	if err != nil {
		return nil, fmt.Errorf("create recognizer: %w", err)
	}

	// ConsistentMap is always true, matching the go-anon server behaviour.
	anonCfg := goanon.Config{
		Strategy:      strategyFromString(cfg.Strategy),
		ConsistentMap: true,
	}

	if len(cfg.SkipTypes) > 0 {
		skipSet := make(map[goanon.EntityType]bool, len(cfg.SkipTypes))
		for _, t := range cfg.SkipTypes {
			skipSet[goanon.EntityType(t)] = true
		}
		for _, t := range allEntityTypes {
			if !skipSet[t] {
				anonCfg.EntityTypes = append(anonCfg.EntityTypes, t)
			}
		}
	}

	return goanon.NewAnonymizer(rec, anonCfg), nil
}

// loadGazetteers opens and parses gazetteer files from the given path map.
func loadGazetteers(ctx context.Context, paths map[string]string) (map[string]*goanon.Gazetteer, *goanon.Gazetteer) {
	gazs := make(map[string]*goanon.Gazetteer, len(paths))
	var firstnamesGaz *goanon.Gazetteer
	for name, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			slog.WarnContext(ctx, "pseudonymizer: failed to open gazetteer", slog.String("name", name), slog.Any("error", err))
			continue
		}
		g, err := goanon.LoadGazetteer(name, f)
		f.Close()
		if err != nil {
			slog.WarnContext(ctx, "pseudonymizer: failed to load gazetteer", slog.String("name", name), slog.Any("error", err))
			continue
		}
		gazs[name] = g
		if name == "firstnames" {
			firstnamesGaz = g
		}
	}
	return gazs, firstnamesGaz
}

// storeOptionsFromConfig builds modelstore.Option list from Config.
func storeOptionsFromConfig(cfg Config) []modelstore.Option {
	var opts []modelstore.Option
	if cfg.CacheDir != "" {
		opts = append(opts, modelstore.WithCacheDir(cfg.CacheDir))
	}
	if cfg.ManifestURL != "" {
		opts = append(opts, modelstore.WithManifestURL(cfg.ManifestURL))
	}
	if cfg.Offline {
		opts = append(opts, modelstore.WithOfflineMode(true))
	}
	return opts
}

// buildCfgKey returns a string that changes whenever the config changes in a way
// that requires recreating the store or the anonymizers.
func buildCfgKey(cfg Config) string {
	b, _ := json.Marshal(cfg)
	return string(b)
}

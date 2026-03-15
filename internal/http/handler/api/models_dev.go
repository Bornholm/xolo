package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	modelsDevURL      = "https://models.dev/api.json"
	modelsDevCacheTTL = 24 * time.Hour
)

// ModelsDevInfo is the pre-filled model info returned to the form.
type ModelsDevInfo struct {
	Name           string `json:"name"`
	ContextWindow  int64  `json:"context_window"`
	OutputWindow   int64  `json:"output_window"`
	PromptCost     int64  `json:"prompt_cost"`     // microcents / 1K tokens
	CompletionCost int64  `json:"completion_cost"` // microcents / 1K tokens
	CapTools       bool   `json:"cap_tools"`
	CapVision      bool   `json:"cap_vision"`
	CapReasoning   bool   `json:"cap_reasoning"`
	CapAudio       bool   `json:"cap_audio"`
}

// modelsDevRaw mirrors the relevant fields of a models.dev model entry.
type modelsDevRaw struct {
	Name      string `json:"name"`
	ToolCall  bool   `json:"tool_call"`
	Reasoning bool   `json:"reasoning"`
	// "attachment" in models.dev means the model accepts attachments (images/docs)
	Attachment bool `json:"attachment"`
	Limit      struct {
		Context int64 `json:"context"`
		Output  int64 `json:"output"`
	} `json:"limit"`
	Cost struct {
		Input  float64 `json:"input"`
		Output float64 `json:"output"`
	} `json:"cost"`
	Modalities struct {
		Input []string `json:"input"`
	} `json:"modalities"`
}

type modelsDevProvider struct {
	Models map[string]modelsDevRaw `json:"models"`
}

// modelsDevCache holds a parsed, indexed catalog with a TTL.
type modelsDevCache struct {
	mu        sync.RWMutex
	index     map[string]ModelsDevInfo // key = "provider/model" or "model"
	fetchedAt time.Time
}

var catalog = &modelsDevCache{}

func (c *modelsDevCache) lookup(id, providerHint string) (ModelsDevInfo, bool) {
	c.mu.RLock()
	if time.Since(c.fetchedAt) > modelsDevCacheTTL {
		c.mu.RUnlock()
		if err := c.refresh(); err != nil {
			slog.Warn("models.dev cache refresh failed", slog.Any("error", err))
		}
		c.mu.RLock()
	}
	defer c.mu.RUnlock()

	// Build candidate list: try the id as-is first, then without "-latest" suffix.
	// models.dev has BOTH "mistral-large-latest" (exact) and "mistral-small-2506"
	// (no -latest variant), so we must try the exact name before stripping.
	candidates := []string{id}
	if base, ok := strings.CutSuffix(id, "-latest"); ok {
		candidates = append(candidates, base)
	}

	for _, cand := range candidates {
		// 1. With provider hint: exact "providerHint/cand".
		if providerHint != "" {
			if info, ok := c.index[providerHint+"/"+cand]; ok {
				return info, true
			}
		}
		// 2. Exact short key.
		if info, ok := c.index[cand]; ok {
			return info, true
		}
		// 3. Strip an embedded provider prefix: "mistral/mistral-small" → "mistral-small".
		if _, after, ok := strings.Cut(cand, "/"); ok {
			if info, ok := c.index[after]; ok {
				return info, true
			}
		}
		// 4. Suffix scan: any "xxx/cand".
		for key, info := range c.index {
			if strings.HasSuffix(key, "/"+cand) {
				return info, true
			}
		}
		// 5. Prefix scan within provider hint: "providerHint/cand-XXXX".
		//    Handles e.g. "mistral-small" → "mistral/mistral-small-2506".
		if providerHint != "" {
			prefix := providerHint + "/" + cand
			for key, info := range c.index {
				if strings.HasPrefix(key, prefix) {
					return info, true
				}
			}
		}
	}
	return ModelsDevInfo{}, false
}

func (c *modelsDevCache) refresh() error {
	resp, err := http.Get(modelsDevURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var raw map[string]modelsDevProvider
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}

	index := make(map[string]ModelsDevInfo)
	for providerID, provider := range raw {
		for modelKey, m := range provider.Models {
			info := ModelsDevInfo{
				Name:          m.Name,
				ContextWindow: m.Limit.Context,
				OutputWindow:  m.Limit.Output,
				// models.dev cost = USD / 1M tokens → microcents / 1K tokens (* 1000)
				PromptCost:     int64(m.Cost.Input * 1_000),
				CompletionCost: int64(m.Cost.Output * 1_000),
				CapTools:       m.ToolCall,
				CapReasoning:   m.Reasoning,
			}
			// Vision: explicit attachment flag or "image" in input modalities.
			info.CapVision = m.Attachment
			for _, mod := range m.Modalities.Input {
				if mod == "image" {
					info.CapVision = true
				}
				if mod == "audio" {
					info.CapAudio = true
				}
			}

			// Index by "provider/model" and by "model" alone.
			fullKey := providerID + "/" + modelKey
			index[fullKey] = info
			// Only set the short key if unambiguous (first win).
			if _, exists := index[modelKey]; !exists {
				index[modelKey] = info
			}
		}
	}

	c.mu.Lock()
	c.index = index
	c.fetchedAt = time.Now()
	c.mu.Unlock()
	return nil
}

func (h *Handler) handleModelsDevLookup(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing id parameter")
		return
	}

	providerHint := strings.TrimSpace(r.URL.Query().Get("provider"))

	info, ok := catalog.lookup(id, providerHint)
	if !ok {
		writeError(w, http.StatusNotFound, "model not found in models.dev catalog")
		return
	}

	writeJSON(w, http.StatusOK, info)
}

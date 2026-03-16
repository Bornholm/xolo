package org

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	httpCtx "github.com/bornholm/xolo/internal/http/context"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	common "github.com/bornholm/xolo/internal/http/handler/webui/common/component"
	"github.com/bornholm/xolo/internal/http/handler/webui/org/component"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"github.com/pkg/errors"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func (h *Handler) getPluginsPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
		} else {
			slog.ErrorContext(ctx, "could not get organization", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	activations, err := h.pluginActivationStore.ListActivations(ctx, org.ID())
	if err != nil {
		slog.ErrorContext(ctx, "could not list plugin activations", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	activeMap := map[string]bool{}
	for _, a := range activations {
		activeMap[a.PluginName] = true
	}

	vmodel := component.PluginsPageVModel{
		Org:         org,
		Descriptors: h.pluginManager.List(),
		Active:      activeMap,
		AppLayoutVModel: common.AppLayoutVModel{
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-plugins",
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}
	templ.Handler(component.PluginsPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) postActivatePlugin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	pluginName := r.PathValue("pluginName")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
		} else {
			slog.ErrorContext(ctx, "could not get organization", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	act := &model.PluginActivation{
		OrgID:      org.ID(),
		PluginName: pluginName,
		Enabled:    true,
		Required:   false,
		Order:      0,
	}
	if err := h.pluginActivationStore.SaveActivation(ctx, act); err != nil {
		slog.ErrorContext(ctx, "could not save plugin activation", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, common.BaseURLString(ctx, common.WithPath("/orgs/", orgSlug, "/admin/plugins")), http.StatusSeeOther)
}

func (h *Handler) postDeactivatePlugin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgSlug := r.PathValue("orgSlug")
	pluginName := r.PathValue("pluginName")

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
		} else {
			slog.ErrorContext(ctx, "could not get organization", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if err := h.pluginActivationStore.DeleteActivation(ctx, org.ID(), pluginName); err != nil && !errors.Is(err, port.ErrNotFound) {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, common.BaseURLString(ctx, common.WithPath("/orgs/", orgSlug, "/admin/plugins")), http.StatusSeeOther)
}

func (h *Handler) getPluginConfigPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	pluginName := r.PathValue("pluginName")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
		} else {
			slog.ErrorContext(ctx, "could not get organization", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	descriptor := h.pluginDescriptor(pluginName)
	if descriptor == nil {
		http.NotFound(w, r)
		return
	}

	existingCfg, _ := h.pluginConfigStore.GetConfig(ctx, org.ID(), pluginName,
		model.PluginConfigScopeOrg, string(org.ID()))

	configJSON := "{}"
	if existingCfg != nil {
		configJSON = existingCfg.ConfigJSON
	}

	vmodel := component.PluginConfigPageVModel{
		Org:        org,
		Descriptor: descriptor,
		Properties: renderableProperties(parseSchemaProperties(descriptor.ConfigSchema)),
		Values:     jsonToStringMap(configJSON),
		HasHTTPUI:  h.pluginManager.HTTPPort(pluginName) > 0,
		AppLayoutVModel: common.AppLayoutVModel{
			User:            user,
			SelectedItem:    "org-" + orgSlug + "-plugins",
			NavigationItems: nav,
			FooterItems:     footer,
		},
	}
	templ.Handler(component.PluginConfigPage(vmodel)).ServeHTTP(w, r)
}

func (h *Handler) postPluginConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	orgSlug := r.PathValue("orgSlug")
	pluginName := r.PathValue("pluginName")
	nav, footer := orgAdminNav(orgSlug)

	org, err := h.orgFromSlug(ctx, orgSlug)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			http.Error(w, "Organization not found", http.StatusNotFound)
		} else {
			slog.ErrorContext(ctx, "could not get organization", slog.Any("error", err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	descriptor := h.pluginDescriptor(pluginName)
	if descriptor == nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	allProperties := parseSchemaProperties(descriptor.ConfigSchema)
	// Only handle fields that can be rendered as simple HTML form inputs.
	// Array/object fields are preserved from existing config below.
	renderable := renderableProperties(allProperties)
	values := formToStringMap(r, renderable)

	// Load existing config to preserve non-renderable fields (e.g. array slots).
	existingCfg, _ := h.pluginConfigStore.GetConfig(ctx, org.ID(), pluginName,
		model.PluginConfigScopeOrg, string(org.ID()))
	existingJSON := "{}"
	if existingCfg != nil {
		existingJSON = existingCfg.ConfigJSON
	}

	configJSON := mergeFormIntoConfig(existingJSON, values, renderable)

	fieldErrors := validateAgainstSchema(schemaForRenderable(descriptor.ConfigSchema, renderable), configJSON)
	if len(fieldErrors) > 0 {
		vmodel := component.PluginConfigPageVModel{
			Org: org, Descriptor: descriptor,
			Properties: renderable, Values: values, FieldErrors: fieldErrors,
			AppLayoutVModel: common.AppLayoutVModel{
				User: user, SelectedItem: "org-" + orgSlug + "-plugins",
				NavigationItems: nav, FooterItems: footer,
			},
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		templ.Handler(component.PluginConfigPage(vmodel)).ServeHTTP(w, r)
		return
	}

	cfg := &model.PluginConfig{
		OrgID:      org.ID(),
		PluginName: pluginName,
		Scope:      model.PluginConfigScopeOrg,
		ScopeID:    string(org.ID()),
		ConfigJSON: configJSON,
	}
	if err := h.pluginConfigStore.SaveConfig(ctx, cfg); err != nil {
		slog.ErrorContext(ctx, "could not save plugin config", slog.Any("error", err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, common.BaseURLString(ctx, common.WithPath("/orgs/", orgSlug, "/admin/plugins/", pluginName, "/config")), http.StatusSeeOther)
}

func (h *Handler) pluginDescriptor(name string) *proto.PluginDescriptor {
	for _, d := range h.pluginManager.List() {
		if d.Name == name {
			return d
		}
	}
	return nil
}

func parseSchemaProperties(schemaJSON string) map[string]map[string]any {
	if schemaJSON == "" {
		return nil
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return nil
	}
	props, _ := schema["properties"].(map[string]any)
	result := map[string]map[string]any{}
	for k, v := range props {
		if m, ok := v.(map[string]any); ok {
			result[k] = m
		}
	}
	return result
}

func jsonToStringMap(j string) map[string]string {
	var m map[string]any
	if err := json.Unmarshal([]byte(j), &m); err != nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for k, v := range m {
		switch val := v.(type) {
		case []any, map[string]any:
			b, _ := json.Marshal(val)
			out[k] = string(b)
		default:
			out[k] = fmt.Sprintf("%v", val)
		}
	}
	return out
}

// renderableProperties returns all properties that can be rendered in the form:
// simple types (string, number, integer, boolean) and arrays-of-objects
// (rendered as a dynamic list editor). Standalone "object" fields are excluded.
func renderableProperties(properties map[string]map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for name, schema := range properties {
		t, _ := schema["type"].(string)
		if t == "object" {
			continue
		}
		if t == "array" {
			items, _ := schema["items"].(map[string]any)
			if items == nil {
				continue
			}
			itemType, _ := items["type"].(string)
			if itemType != "object" {
				continue // plain arrays (non-object items) not supported
			}
		}
		out[name] = schema
	}
	return out
}

// isArrayOfObjects reports whether a schema describes an array of objects.
func isArrayOfObjects(schema map[string]any) bool {
	if t, _ := schema["type"].(string); t != "array" {
		return false
	}
	items, _ := schema["items"].(map[string]any)
	if items == nil {
		return false
	}
	itemType, _ := items["type"].(string)
	return itemType == "object"
}

func formToStringMap(r *http.Request, properties map[string]map[string]any) map[string]string {
	out := map[string]string{}
	for name, schema := range properties {
		if isArrayOfObjects(schema) {
			// Array-of-objects fields are submitted as a hidden JSON field named "{name}__json"
			out[name+"__json"] = r.FormValue(name + "__json")
		} else {
			out[name] = r.FormValue(name)
		}
	}
	return out
}

// mergeFormIntoConfig merges form values into existingJSON.
// Keys ending in "__json" are treated as pre-serialised JSON arrays (from dynamic list editors).
func mergeFormIntoConfig(existingJSON string, formValues map[string]string, properties map[string]map[string]any) string {
	base := map[string]any{}
	if existingJSON != "" {
		_ = json.Unmarshal([]byte(existingJSON), &base)
	}
	for k, v := range formValues {
		if actualName, ok := strings.CutSuffix(k, "__json"); ok {
			// Array-of-objects field submitted as JSON
			var arr []any
			if json.Unmarshal([]byte(v), &arr) == nil {
				base[actualName] = arr
			}
			continue
		}
		prop := properties[k]
		fieldType, _ := prop["type"].(string)
		switch fieldType {
		case "integer":
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				base[k] = i
			} else {
				base[k] = v
			}
		case "number":
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				base[k] = f
			} else {
				base[k] = v
			}
		case "boolean":
			base[k] = v == "true" || v == "on"
		default:
			base[k] = v
		}
	}
	b, _ := json.Marshal(base)
	return string(b)
}


// schemaForRenderable returns a modified schema where non-renderable fields
// (array, object) are removed from the "required" array so that validation
// only enforces constraints on fields that can actually be set via the form.
func schemaForRenderable(schemaJSON string, renderable map[string]map[string]any) string {
	var schema map[string]any
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return schemaJSON
	}
	required, _ := schema["required"].([]any)
	if len(required) == 0 {
		return schemaJSON
	}
	filtered := make([]any, 0, len(required))
	for _, r := range required {
		if name, ok := r.(string); ok {
			if _, isRenderable := renderable[name]; isRenderable {
				filtered = append(filtered, name)
			}
		}
	}
	schema["required"] = filtered
	b, _ := json.Marshal(schema)
	return string(b)
}

func validateAgainstSchema(schemaJSON, configJSON string) map[string]string {
	if schemaJSON == "" {
		return nil
	}
	var schemaDoc any
	if err := json.Unmarshal([]byte(schemaJSON), &schemaDoc); err != nil {
		return nil
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", schemaDoc); err != nil {
		return nil
	}
	sch, err := c.Compile("schema.json")
	if err != nil {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(configJSON), &v); err != nil {
		return map[string]string{"_": "invalid JSON"}
	}
	if err := sch.Validate(v); err != nil {
		return map[string]string{"_": err.Error()}
	}
	return nil
}

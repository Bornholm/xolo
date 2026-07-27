package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bornholm/xolo/pkg/pluginsdk"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
)

// fakeUIHost implémente pluginsdk.HostClient avec un store en mémoire.
type fakeUIHost struct {
	secrets map[string]string
	emit    pluginsdk.Event
}

func newFakeUIHost() *fakeUIHost {
	return &fakeUIHost{secrets: map[string]string{}}
}

func (h *fakeUIHost) GetConfig(_ context.Context, _, _ string) (string, error) {
	return "{}", nil
}
func (h *fakeUIHost) SaveConfig(_ context.Context, _, _, _ string) error { return nil }
func (h *fakeUIHost) ListModels(_ context.Context, _ string) ([]*proto.ModelInfo, error) {
	return nil, nil
}
func (h *fakeUIHost) GetSecret(_ context.Context, _, _, nodeID, key string) (string, bool, error) {
	v, ok := h.secrets[nodeID+":"+key]
	return v, ok, nil
}
func (h *fakeUIHost) SetSecret(_ context.Context, _, _, nodeID, key, value string) error {
	h.secrets[nodeID+":"+key] = value
	return nil
}
func (h *fakeUIHost) DeleteSecret(_ context.Context, _, _, nodeID, key string) error {
	delete(h.secrets, nodeID+":"+key)
	return nil
}
func (h *fakeUIHost) EmitEvent(_ context.Context, e pluginsdk.Event) error {
	h.emit = e
	return nil
}

var _ pluginsdk.HostClient = (*fakeUIHost)(nil)

const uiHexKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// uiRequest construit une requête HTTP pré-injectée avec le host client et
// le nom du plugin dans le contexte (comme le fait ServeWithUI en production).
func uiRequest(method, target, body, orgID, nodeID string, host pluginsdk.HostClient) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	if orgID != "" {
		r.Header.Set("X-Xolo-Org-Id", orgID)
	}
	if nodeID != "" {
		r.Header.Set("X-Xolo-Node-Id", nodeID)
	}
	ctx := pluginsdk.ContextWithHostClientForTest(r.Context(), host)
	ctx = pluginsdk.ContextWithPluginNameForTest(ctx, "pseudonymizer")
	return r.WithContext(ctx)
}

func TestHandleSaveHashKey_Success(t *testing.T) {
	host := newFakeUIHost()
	handler := newUIHandler(&Plugin{})

	req := uiRequest(http.MethodPost, "/api/secrets/hash_key",
		"hash_key="+uiHexKey, "org-1", "node-1", host)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d (%s)", rec.Code, rec.Body.String())
	}
	if got := host.secrets["node-1:hash_key"]; got != uiHexKey {
		t.Errorf("expected stored secret %q, got %q", uiHexKey, got)
	}
}

func TestHandleSaveHashKey_Invalid(t *testing.T) {
	host := newFakeUIHost()
	handler := newUIHandler(&Plugin{})

	req := uiRequest(http.MethodPost, "/api/secrets/hash_key",
		"hash_key=too-short", "org-1", "node-1", host)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", rec.Code, rec.Body.String())
	}
	if _, ok := host.secrets["node-1:hash_key"]; ok {
		t.Errorf("invalid key should not be stored")
	}
}

func TestHandleDeleteHashKey(t *testing.T) {
	host := newFakeUIHost()
	host.secrets["node-1:hash_key"] = uiHexKey
	handler := newUIHandler(&Plugin{})

	req := uiRequest(http.MethodPost, "/api/secrets/hash_key/delete",
		"", "org-1", "node-1", host)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d (%s)", rec.Code, rec.Body.String())
	}
	if _, ok := host.secrets["node-1:hash_key"]; ok {
		t.Errorf("expected secret to be deleted")
	}
}
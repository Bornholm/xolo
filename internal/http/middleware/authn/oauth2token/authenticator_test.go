package oauth2token

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// newIntrospectionServer returns a fake RFC 7662 endpoint that echoes the
// provided response for the token "good" and reports every other token as
// inactive. It counts the number of introspection calls.
func newIntrospectionServer(t *testing.T, resp map[string]any) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.PostFormValue("token") != "good" {
			json.NewEncoder(w).Encode(map[string]any{"active": false})
			return
		}
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func requestWithToken(token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

func TestAuthenticate_ActiveToken(t *testing.T) {
	srv, _ := newIntrospectionServer(t, map[string]any{
		"active":             true,
		"sub":                "user-123",
		"email":              "jane@example.com",
		"preferred_username": "jane",
	})

	h := NewHandler([]Provider{{ID: "gitea", IntrospectionURL: srv.URL, ClientID: "xolo"}})

	user, err := h.Authenticate(nil, requestWithToken("good"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected a user, got nil")
	}
	if user.Subject != "user-123" || user.Provider != "gitea" || user.Email != "jane@example.com" || user.DisplayName != "jane" {
		t.Errorf("unexpected user: %+v", user)
	}
}

func TestAuthenticate_InactiveToken(t *testing.T) {
	srv, _ := newIntrospectionServer(t, nil)

	h := NewHandler([]Provider{{ID: "gitea", IntrospectionURL: srv.URL, ClientID: "xolo"}})

	user, err := h.Authenticate(nil, requestWithToken("revoked"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user for inactive token, got %+v", user)
	}
}

func TestAuthenticate_NoBearer(t *testing.T) {
	srv, calls := newIntrospectionServer(t, nil)

	h := NewHandler([]Provider{{ID: "gitea", IntrospectionURL: srv.URL, ClientID: "xolo"}})

	user, err := h.Authenticate(nil, requestWithToken(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user without bearer, got %+v", user)
	}
	if *calls != 0 {
		t.Errorf("introspection should not be called without a token, got %d calls", *calls)
	}
}

func TestAuthenticate_RequiredScope(t *testing.T) {
	srv, _ := newIntrospectionServer(t, map[string]any{
		"active": true,
		"sub":    "user-123",
		"scope":  "openid profile",
	})

	h := NewHandler([]Provider{{ID: "gitea", IntrospectionURL: srv.URL, ClientID: "xolo", RequiredScope: "xolo-api"}})
	if user, _ := h.Authenticate(nil, requestWithToken("good")); user != nil {
		t.Errorf("expected rejection when required scope is absent, got %+v", user)
	}

	h2 := NewHandler([]Provider{{ID: "gitea", IntrospectionURL: srv.URL, ClientID: "xolo", RequiredScope: "profile"}})
	if user, _ := h2.Authenticate(nil, requestWithToken("good")); user == nil {
		t.Error("expected success when required scope is present")
	}
}

func TestAuthenticate_RequiredAudience(t *testing.T) {
	srv, _ := newIntrospectionServer(t, map[string]any{
		"active": true,
		"sub":    "user-123",
		"aud":    []string{"other-rs", "xolo"},
	})

	h := NewHandler([]Provider{{ID: "gitea", IntrospectionURL: srv.URL, ClientID: "xolo", RequiredAudience: "xolo"}})
	if user, _ := h.Authenticate(nil, requestWithToken("good")); user == nil {
		t.Error("expected success when audience matches")
	}

	h2 := NewHandler([]Provider{{ID: "gitea", IntrospectionURL: srv.URL, ClientID: "xolo", RequiredAudience: "not-xolo"}})
	if user, _ := h2.Authenticate(nil, requestWithToken("good")); user != nil {
		t.Errorf("expected rejection when audience mismatches, got %+v", user)
	}
}

func TestAuthenticate_EnrichesFromUserInfo(t *testing.T) {
	// Introspection returns sub + username but no email (Gitea behaviour).
	introspect, _ := newIntrospectionServer(t, map[string]any{
		"active":   true,
		"sub":      "3",
		"username": "wpetit",
	})

	var userInfoCalls int32
	userInfo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&userInfoCalls, 1)
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"sub":                "3",
			"email":              "wpetit@cadoles.com",
			"preferred_username": "wpetit",
		})
	}))
	t.Cleanup(userInfo.Close)

	h := NewHandler([]Provider{{
		ID:               "gitea",
		IntrospectionURL: introspect.URL,
		UserInfoURL:      userInfo.URL,
		ClientID:         "xolo",
	}})

	user, err := h.Authenticate(nil, requestWithToken("good"))
	if err != nil || user == nil {
		t.Fatalf("user=%v err=%v", user, err)
	}
	if user.Email != "wpetit@cadoles.com" {
		t.Errorf("expected email enriched from userinfo, got %q", user.Email)
	}
	if user.DisplayName != "wpetit" {
		t.Errorf("expected display name %q, got %q", "wpetit", user.DisplayName)
	}
	if user.Subject != "3" {
		t.Errorf("expected subject preserved from introspection, got %q", user.Subject)
	}
	if userInfoCalls != 1 {
		t.Errorf("expected exactly one userinfo call, got %d", userInfoCalls)
	}
}

func TestAuthenticate_UserInfoSubjectMismatchIgnored(t *testing.T) {
	introspect, _ := newIntrospectionServer(t, map[string]any{
		"active":   true,
		"sub":      "3",
		"username": "wpetit",
	})
	userInfo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"sub": "999", "email": "attacker@example.com"})
	}))
	t.Cleanup(userInfo.Close)

	h := NewHandler([]Provider{{ID: "gitea", IntrospectionURL: introspect.URL, UserInfoURL: userInfo.URL, ClientID: "xolo"}})

	user, _ := h.Authenticate(nil, requestWithToken("good"))
	if user == nil {
		t.Fatal("expected a user")
	}
	if user.Email != "" {
		t.Errorf("expected mismatched userinfo to be ignored, got email %q", user.Email)
	}
	if user.Subject != "3" {
		t.Errorf("expected subject %q, got %q", "3", user.Subject)
	}
}

// newUserInfoServer returns a fake OIDC UserInfo endpoint valid only for the
// bearer token "good", counting calls.
func newUserInfoServer(t *testing.T, resp map[string]any) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

// A provider without an introspection endpoint (e.g. Auth0) validates opaque
// tokens through UserInfo.
func TestAuthenticate_UserInfoOnlyValidation(t *testing.T) {
	srv, calls := newUserInfoServer(t, map[string]any{
		"sub":                "auth0|abc",
		"email":              "jane@example.com",
		"preferred_username": "jane",
	})

	h := NewHandler([]Provider{{ID: "auth0", UserInfoURL: srv.URL}})

	user, err := h.Authenticate(nil, requestWithToken("good"))
	if err != nil || user == nil {
		t.Fatalf("user=%v err=%v", user, err)
	}
	if user.Subject != "auth0|abc" || user.Provider != "auth0" || user.Email != "jane@example.com" || user.DisplayName != "jane" {
		t.Errorf("unexpected user: %+v", user)
	}

	// Invalid token → userinfo 401 → no user.
	if u, _ := h.Authenticate(nil, requestWithToken("revoked")); u != nil {
		t.Errorf("expected nil user for invalid token, got %+v", u)
	}
	if *calls != 2 {
		t.Errorf("expected 2 userinfo calls, got %d", *calls)
	}
}

func TestAuthenticate_CachesResult(t *testing.T) {
	srv, calls := newIntrospectionServer(t, map[string]any{
		"active": true,
		"sub":    "user-123",
	})

	h := NewHandler([]Provider{{ID: "gitea", IntrospectionURL: srv.URL, ClientID: "xolo"}})

	for i := range 3 {
		if user, err := h.Authenticate(nil, requestWithToken("good")); err != nil || user == nil {
			t.Fatalf("call %d: user=%v err=%v", i, user, err)
		}
	}
	if *calls != 1 {
		t.Errorf("expected a single introspection call thanks to caching, got %d", *calls)
	}
}

package mcpclient_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/bornholm/xolo/internal/adapter/mcpclient"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestConnect_RetriesOnTooManyRequests(t *testing.T) {
	var attempts atomic.Int32

	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "greet", Description: "greets"}, greet)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return mcpServer }, nil)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) <= 2 {
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	defer srv.Close()

	session, err := mcpclient.Connect(t.Context(), mcpclient.Config{
		Endpoint:       srv.URL + "/mcp",
		MaxRetries:     5,
		RetryBaseDelay: 1,
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer session.Close()

	if attempts.Load() < 3 {
		t.Errorf("expected at least 3 attempts (2 failures + 1 success), got %d", attempts.Load())
	}
}

func TestConnect_GivesUpAfterMaxRetries(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.Error(w, "slow down", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := mcpclient.Connect(t.Context(), mcpclient.Config{
		Endpoint:       srv.URL + "/mcp",
		MaxRetries:     2,
		RetryBaseDelay: 1,
	})
	if err == nil {
		t.Fatal("expected Connect to eventually fail")
	}
	// The SDK's handshake issues more than one HTTP call (e.g. "initialize"
	// then "notifications/initialized"); each independently retries up to
	// 1+MaxRetries times before giving up, so attempts is a multiple of 3.
	if got := attempts.Load(); got < 3 || got%3 != 0 {
		t.Errorf("expected attempts to be a positive multiple of 3 (1 + 2 retries per call), got %d", got)
	}
}

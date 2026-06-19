package mcpclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bornholm/xolo/internal/adapter/mcpclient"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type greetInput struct {
	Name string `json:"name" jsonschema:"the name to greet"`
}

func greet(_ context.Context, _ *mcp.CallToolRequest, in greetInput) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "hello " + in.Name}},
	}, nil, nil
}

func newTestServer(t *testing.T, requireAuthHeader string) *httptest.Server {
	t.Helper()

	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "v1"}, nil)
	mcp.AddTool(mcpServer, &mcp.Tool{Name: "greet", Description: "greets someone"}, greet)

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return mcpServer
	}, nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if requireAuthHeader != "" && r.Header.Get("Authorization") != requireAuthHeader {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})

	return httptest.NewServer(mux)
}

func TestBuildTools_ListsAndCallsTool(t *testing.T) {
	srv := newTestServer(t, "")
	defer srv.Close()

	session, err := mcpclient.Connect(context.Background(), mcpclient.Config{Endpoint: srv.URL + "/mcp"})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer session.Close()

	tools, err := mcpclient.BuildTools(context.Background(), session, nil)
	if err != nil {
		t.Fatalf("BuildTools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name() != "greet" {
		t.Fatalf("expected exactly one tool named 'greet', got %v", tools)
	}

	result, err := tools[0].Execute(context.Background(), map[string]any{"name": "xolo"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Text() != "hello xolo" {
		t.Errorf("expected %q, got %q", "hello xolo", result.Text())
	}
}

func TestBuildTools_FiltersByName(t *testing.T) {
	srv := newTestServer(t, "")
	defer srv.Close()

	session, err := mcpclient.Connect(context.Background(), mcpclient.Config{Endpoint: srv.URL + "/mcp"})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer session.Close()

	tools, err := mcpclient.BuildTools(context.Background(), session, []string{"not-greet"})
	if err != nil {
		t.Fatalf("BuildTools: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected no tools to match the filter, got %d", len(tools))
	}
}

func TestConnect_InjectsAuthHeader(t *testing.T) {
	srv := newTestServer(t, "Bearer secret-token")
	defer srv.Close()

	_, err := mcpclient.Connect(context.Background(), mcpclient.Config{Endpoint: srv.URL + "/mcp"})
	if err == nil {
		t.Fatal("expected connection without auth header to fail")
	}

	session, err := mcpclient.Connect(context.Background(), mcpclient.Config{
		Endpoint:       srv.URL + "/mcp",
		AuthHeaderName: "Authorization",
		AuthValue:      "Bearer secret-token",
	})
	if err != nil {
		t.Fatalf("Connect with correct auth header: %v", err)
	}
	defer session.Close()
}

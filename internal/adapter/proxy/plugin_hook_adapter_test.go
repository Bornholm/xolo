package proxy_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	genaiProxy "github.com/bornholm/genai/proxy"
	proxyAdapter "github.com/bornholm/xolo/internal/adapter/proxy"
	"github.com/bornholm/xolo/internal/core/model"
	"github.com/bornholm/xolo/internal/core/port"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// ── stubs ──────────────────────────────────────────────────────────────────

type stubAuthToken struct {
	id          model.AuthTokenID
	owner       model.User
	application model.Application
	orgID       model.OrgID
}

func (t *stubAuthToken) ID() model.AuthTokenID          { return t.id }
func (t *stubAuthToken) Owner() model.User              { return t.owner }
func (t *stubAuthToken) Application() model.Application { return t.application }
func (t *stubAuthToken) Label() string                  { return "test-token" }
func (t *stubAuthToken) Value() string                  { return "raw-token" }
func (t *stubAuthToken) OrgID() model.OrgID             { return t.orgID }
func (t *stubAuthToken) ExpiresAt() *time.Time          { return nil }

var _ model.AuthToken = &stubAuthToken{}

type stubUserStore struct {
	token model.AuthToken
}

func (s *stubUserStore) FindAuthToken(_ context.Context, _ string) (model.AuthToken, error) {
	return s.token, nil
}

type stubActivationStore struct {
	activations []*model.PluginActivation
}

func (s *stubActivationStore) GetActivation(_ context.Context, _ model.OrgID, _ string) (*model.PluginActivation, error) {
	return nil, nil
}
func (s *stubActivationStore) ListActivations(_ context.Context, _ model.OrgID) ([]*model.PluginActivation, error) {
	return s.activations, nil
}
func (s *stubActivationStore) SaveActivation(_ context.Context, _ *model.PluginActivation) error {
	return nil
}
func (s *stubActivationStore) DeleteActivation(_ context.Context, _ model.OrgID, _ string) error {
	return nil
}

type stubConfigStore struct{}

func (s *stubConfigStore) GetConfig(_ context.Context, _ model.OrgID, _ string, _ model.PluginConfigScope, _ string) (*model.PluginConfig, error) {
	return nil, nil
}
func (s *stubConfigStore) ListConfigsByPlugin(_ context.Context, _ string) ([]model.PluginConfig, error) {
	return nil, nil
}
func (s *stubConfigStore) SaveConfig(_ context.Context, _ *model.PluginConfig) error { return nil }
func (s *stubConfigStore) DeleteConfig(_ context.Context, _ model.OrgID, _ string, _ model.PluginConfigScope, _ string) error {
	return nil
}

type stubProviderStore struct{}

func (s *stubProviderStore) CreateProvider(_ context.Context, _ model.Provider) error { return nil }
func (s *stubProviderStore) GetProviderByID(_ context.Context, _ model.ProviderID) (model.Provider, error) {
	return nil, nil
}
func (s *stubProviderStore) ListProviders(_ context.Context, _ model.OrgID) ([]model.Provider, error) {
	return nil, nil
}
func (s *stubProviderStore) SaveProvider(_ context.Context, _ model.Provider) error     { return nil }
func (s *stubProviderStore) DeleteProvider(_ context.Context, _ model.ProviderID) error { return nil }
func (s *stubProviderStore) CreateLLMModel(_ context.Context, _ model.LLMModel) error   { return nil }
func (s *stubProviderStore) GetLLMModelByID(_ context.Context, _ model.LLMModelID) (model.LLMModel, error) {
	return nil, nil
}
func (s *stubProviderStore) GetLLMModelByProxyName(_ context.Context, _ model.OrgID, _ string) (model.LLMModel, error) {
	return nil, nil
}
func (s *stubProviderStore) ListLLMModels(_ context.Context, _ model.OrgID) ([]model.LLMModel, error) {
	return nil, nil
}
func (s *stubProviderStore) ListEnabledLLMModels(_ context.Context, _ model.OrgID) ([]model.LLMModel, error) {
	return nil, nil
}
func (s *stubProviderStore) SaveLLMModel(_ context.Context, _ model.LLMModel) error     { return nil }
func (s *stubProviderStore) DeleteLLMModel(_ context.Context, _ model.LLMModelID) error { return nil }

type stubVirtualModelStore struct{}

func (s *stubVirtualModelStore) GetVirtualModelByID(_ context.Context, _ model.VirtualModelID) (model.VirtualModel, error) {
	var vm model.VirtualModel
	return vm, port.ErrNotFound
}
func (s *stubVirtualModelStore) GetVirtualModelByName(_ context.Context, _ model.OrgID, _ string) (model.VirtualModel, error) {
	var vm model.VirtualModel
	return vm, port.ErrNotFound
}
func (s *stubVirtualModelStore) ListVirtualModels(_ context.Context, _ model.OrgID) ([]model.VirtualModel, error) {
	return nil, nil
}
func (s *stubVirtualModelStore) CreateVirtualModel(_ context.Context, _ model.VirtualModel) error {
	return nil
}
func (s *stubVirtualModelStore) SaveVirtualModel(_ context.Context, _ model.VirtualModel) error {
	return nil
}
func (s *stubVirtualModelStore) DeleteVirtualModel(_ context.Context, _ model.VirtualModelID) error {
	return nil
}

// ── in-process gRPC plugin server ─────────────────────────────────────────

type allowPlugin struct {
	proto.UnimplementedXoloPluginServer
}

func (p *allowPlugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:         "test-allow",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
	}, nil
}

func (p *allowPlugin) PreRequest(_ context.Context, in *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
	return &proto.PreRequestOutput{Allowed: true}, nil
}

type blockPlugin struct {
	proto.UnimplementedXoloPluginServer
}

func (p *blockPlugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:         "test-block",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
	}, nil
}

func (p *blockPlugin) PreRequest(_ context.Context, in *proto.PreRequestInput) (*proto.PreRequestOutput, error) {
	return &proto.PreRequestOutput{Allowed: false, RejectionReason: "blocked by policy"}, nil
}

// startBufconnServer starts an in-process gRPC server using bufconn and returns a client.
func startBufconnServer(t *testing.T, srv proto.XoloPluginServer) proto.XoloPluginClient {
	t.Helper()
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)

	grpcSrv := grpc.NewServer()
	proto.RegisterXoloPluginServer(grpcSrv, srv)

	go func() {
		if err := grpcSrv.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			t.Logf("gRPC server error: %v", err)
		}
	}()
	t.Cleanup(func() { grpcSrv.Stop(); lis.Close() })

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to dial bufconn: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	return proto.NewXoloPluginClient(conn)
}

// ── helpers ────────────────────────────────────────────────────────────────

func makeRequest(t *testing.T) *genaiProxy.ProxyRequest {
	t.Helper()
	headers := http.Header{}
	headers.Set("Authorization", "Bearer raw-token")
	return &genaiProxy.ProxyRequest{
		Model:    "acme/gpt-4",
		UserID:   "user-1",
		Headers:  headers,
		Metadata: map[string]any{},
	}
}

// ── tests ──────────────────────────────────────────────────────────────────

func TestPluginHookAdapter_PreRequest_Allowed(t *testing.T) {
	ctx := context.Background()

	client := startBufconnServer(t, &allowPlugin{})

	orgID := model.OrgID("org-1")
	token := &stubAuthToken{
		id:    model.AuthTokenID("tok-1"),
		orgID: orgID,
	}
	userStore := &stubUserStore{token: token}
	activationStore := &stubActivationStore{
		activations: []*model.PluginActivation{
			{OrgID: orgID, PluginName: "test-allow", Enabled: true, Required: false},
		},
	}
	configStore := &stubConfigStore{}
	clients := map[string]proto.XoloPluginClient{
		"test-allow": client,
	}
	descriptors := map[string]*proto.PluginDescriptor{
		"test-allow": {
			Name:         "test-allow",
			Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
		},
	}

	adapter := proxyAdapter.NewPluginHookAdapter(clients, descriptors, activationStore, configStore, userStore, &stubProviderStore{}, &stubVirtualModelStore{}, nil, nil)

	req := makeRequest(t)
	result, err := adapter.PreRequest(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil HookResult (allowed), got %+v", result)
	}
}

func TestPluginHookAdapter_PreRequest_Blocked(t *testing.T) {
	ctx := context.Background()

	client := startBufconnServer(t, &blockPlugin{})

	orgID := model.OrgID("org-1")
	token := &stubAuthToken{
		id:    model.AuthTokenID("tok-1"),
		orgID: orgID,
	}
	userStore := &stubUserStore{token: token}
	activationStore := &stubActivationStore{
		activations: []*model.PluginActivation{
			{OrgID: orgID, PluginName: "test-block", Enabled: true, Required: false},
		},
	}
	configStore := &stubConfigStore{}
	clients := map[string]proto.XoloPluginClient{
		"test-block": client,
	}
	descriptors := map[string]*proto.PluginDescriptor{
		"test-block": {
			Name:         "test-block",
			Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_PRE_REQUEST},
		},
	}

	adapter := proxyAdapter.NewPluginHookAdapter(clients, descriptors, activationStore, configStore, userStore, &stubProviderStore{}, &stubVirtualModelStore{}, nil, nil)

	req := makeRequest(t)
	result, err := adapter.PreRequest(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil HookResult (blocked)")
	}
	if result.Response == nil {
		t.Fatal("expected non-nil Response in HookResult")
	}
	if result.Response.StatusCode != 403 {
		t.Fatalf("expected status 403, got %d", result.Response.StatusCode)
	}
}

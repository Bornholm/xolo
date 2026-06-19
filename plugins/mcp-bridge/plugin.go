package main

import (
	"context"
	"sync"
	"time"

	"github.com/bornholm/xolo/internal/adapter/mcpclient"
	"github.com/bornholm/xolo/pkg/pluginsdk"
	proto "github.com/bornholm/xolo/pkg/pluginsdk/proto"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

const secretKeyAuthValue = "authValue"

// Plugin implémente proto.XoloPluginServer.
type Plugin struct {
	proto.UnimplementedXoloPluginServer

	mu         sync.Mutex
	hostClient pluginsdk.HostClient
}

// SetHostClient implémente pluginsdk.HostClientSetter.
func (p *Plugin) SetHostClient(c pluginsdk.HostClient) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hostClient = c
}

func (p *Plugin) getHostClient() pluginsdk.HostClient {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.hostClient
}

func (p *Plugin) Describe(_ context.Context, _ *proto.DescribeRequest) (*proto.PluginDescriptor, error) {
	return &proto.PluginDescriptor{
		Name:         "mcp-bridge",
		Version:      "0.0.1",
		Description:  "Connecte un serveur MCP (Streamable HTTP) externe : expose ses tools au LLM et résout les appels d'outils côté serveur.",
		Capabilities: []proto.PluginDescriptor_Capability{proto.PluginDescriptor_TOOL_PROVIDER},
		ConfigSchema: configSchemaJSON,
		InputPorts: []*proto.PortDescriptor{
			{Name: "request", PortType: "request", Required: true},
		},
		OutputPorts: []*proto.PortDescriptor{
			{Name: "request", PortType: "request", Required: true},
		},
	}, nil
}

// connect parses the node's config and opens an MCP session, resolving the
// auth value from the per-node secret store (never from ConfigJson).
func (p *Plugin) connect(ctx context.Context, reqCtx *proto.RequestContext) (*mcp.ClientSession, Config, error) {
	cfg, err := parseConfig(reqCtx.GetConfigJson())
	if err != nil {
		return nil, Config{}, errors.Wrap(err, "parse config")
	}
	if cfg.Endpoint == "" {
		return nil, Config{}, errors.New("mcp-bridge: endpoint not configured")
	}

	var authValue string
	if hc := p.getHostClient(); hc != nil {
		v, found, err := hc.GetSecret(ctx, reqCtx.GetOrgId(), "mcp-bridge", reqCtx.GetNodeId(), secretKeyAuthValue)
		if err != nil {
			return nil, Config{}, errors.Wrap(err, "get auth secret")
		}
		if found {
			authValue = v
		}
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	session, err := mcpclient.Connect(ctx, mcpclient.Config{
		Endpoint:       cfg.Endpoint,
		AuthHeaderName: cfg.AuthHeaderName,
		AuthValue:      authValue,
		Timeout:        timeout,
	})
	if err != nil {
		return nil, Config{}, errors.Wrap(err, "connect to MCP server")
	}
	return session, cfg, nil
}

func (p *Plugin) ListTools(ctx context.Context, in *proto.ListToolsInput) (*proto.ListToolsOutput, error) {
	session, cfg, err := p.connect(ctx, in.GetCtx())
	if err != nil {
		return nil, err
	}
	defer session.Close()

	infos, err := mcpclient.ListToolInfos(ctx, session, cfg.ToolFilter)
	if err != nil {
		return nil, err
	}

	tools := make([]*proto.ToolDescriptor, 0, len(infos))
	for _, info := range infos {
		tools = append(tools, &proto.ToolDescriptor{
			Name:            info.Name,
			Description:     info.Description,
			InputSchemaJson: info.InputSchemaJSON,
		})
	}
	return &proto.ListToolsOutput{
		Tools:                   tools,
		MaxConsecutiveToolCalls: int32(cfg.MaxConsecutiveToolCalls),
	}, nil
}

func (p *Plugin) CallTool(ctx context.Context, in *proto.CallToolInput) (*proto.CallToolOutput, error) {
	session, _, err := p.connect(ctx, in.GetCtx())
	if err != nil {
		return nil, err
	}
	defer session.Close()

	text, isError, err := mcpclient.CallToolText(ctx, session, in.Name, in.ArgumentsJson)
	if err != nil {
		return nil, err
	}
	return &proto.CallToolOutput{ResultText: text, IsError: isError}, nil
}

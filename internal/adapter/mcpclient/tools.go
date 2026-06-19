package mcpclient

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/bornholm/genai/llm"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
)

// ToolInfo is a lightweight, transport-agnostic description of an MCP tool,
// suitable for crossing a gRPC boundary (e.g. towards the mcp-bridge plugin
// protocol) without depending on llm.Tool.
type ToolInfo struct {
	Name            string
	Description     string
	InputSchemaJSON string
}

// ListToolInfos lists the tools exposed by session as ToolInfo values,
// filtered by name if filter is non-empty.
func ListToolInfos(ctx context.Context, session *mcp.ClientSession, filter []string) ([]ToolInfo, error) {
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, errors.Wrap(err, "list MCP tools")
	}

	infos := make([]ToolInfo, 0, len(result.Tools))
	for _, t := range result.Tools {
		if len(filter) > 0 && !slices.Contains(filter, t.Name) {
			continue
		}
		schemaJSON, _ := json.Marshal(t.InputSchema)
		infos = append(infos, ToolInfo{Name: t.Name, Description: t.Description, InputSchemaJSON: string(schemaJSON)})
	}
	return infos, nil
}

// CallToolText calls the named tool with JSON-encoded arguments and returns
// its result as plain text, plus whether the call ended in an error.
func CallToolText(ctx context.Context, session *mcp.ClientSession, name, argumentsJSON string) (string, bool, error) {
	var args map[string]any
	if argumentsJSON != "" {
		if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
			return "", false, errors.Wrap(err, "unmarshal tool arguments")
		}
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return "", false, errors.Wrapf(err, "call MCP tool %q", name)
	}

	return contentToText(result.Content), result.IsError, nil
}

// BuildTools lists the tools exposed by session and wraps each as an
// llm.Tool whose Execute calls back into the MCP server. If filter is
// non-empty, only tools whose name appears in it are kept.
func BuildTools(ctx context.Context, session *mcp.ClientSession, filter []string) ([]llm.Tool, error) {
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, errors.Wrap(err, "list MCP tools")
	}

	tools := make([]llm.Tool, 0, len(result.Tools))
	for _, t := range result.Tools {
		if len(filter) > 0 && !slices.Contains(filter, t.Name) {
			continue
		}
		tools = append(tools, newMCPTool(session, t))
	}
	return tools, nil
}

func newMCPTool(session *mcp.ClientSession, t *mcp.Tool) llm.Tool {
	schema, _ := t.InputSchema.(map[string]any)
	return llm.NewFuncTool(t.Name, t.Description, schema, func(ctx context.Context, params map[string]any) (llm.ToolResult, error) {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      t.Name,
			Arguments: params,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "call MCP tool %q", t.Name)
		}

		text := contentToText(result.Content)
		if result.IsError {
			return nil, errors.New(text)
		}
		return llm.NewToolResult(text), nil
	})
}

func contentToText(content []mcp.Content) string {
	var b strings.Builder
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

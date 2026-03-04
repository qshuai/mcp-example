package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCP 工具定义
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// MCPClient MCP 客户端配置
type MCPClient struct {
	endpoint   string
	httpClient *http.Client
	client     *mcp.Client
	session    *mcp.ClientSession
}

// NewMCPClient 创建 MCP 客户端
func NewMCPClient(endpoint string) *MCPClient {
	if endpoint == "" {
		endpoint = "http://localhost:8081"
	}

	return &MCPClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		client: mcp.NewClient(
			&mcp.Implementation{Name: "agent-client", Version: "v0.1.0"},
			nil,
		),
	}
}

// ListTools 获取 MCP 工具列表
func (c *MCPClient) ListTools(ctx context.Context) ([]MCPTool, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}

	resp, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	tools := make([]MCPTool, 0, len(resp.Tools))
	for _, tool := range resp.Tools {
		mcpTool := MCPTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: toSchemaMap(tool.InputSchema),
		}
		tools = append(tools, mcpTool)
	}

	return tools, nil
}

// CallTool 调用 MCP 工具
func (c *MCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	if err := c.ensureSession(ctx); err != nil {
		return "", err
	}

	resp, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("call tool: %w", err)
	}

	if resp.IsError {
		return "", fmt.Errorf("tool returned error: %s", firstText(resp.Content))
	}

	resultJSON, err := marshalToolResult(resp)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// Close 关闭 MCP 会话
func (c *MCPClient) Close() error {
	if c.session == nil {
		return nil
	}
	err := c.session.Close()
	c.session = nil
	return err
}

func (c *MCPClient) ensureSession(ctx context.Context) error {
	if c.session != nil {
		return nil
	}

	transport := &mcp.SSEClientTransport{
		Endpoint:   c.endpoint,
		HTTPClient: c.httpClient,
	}

	session, err := c.client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("connect MCP server: %w", err)
	}
	c.session = session
	return nil
}

func toSchemaMap(v any) map[string]interface{} {
	if v == nil {
		return nil
	}

	if schema, ok := v.(map[string]interface{}); ok {
		return schema
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil
	}
	return schema
}

func marshalToolResult(result *mcp.CallToolResult) ([]byte, error) {
	if result.StructuredContent != nil {
		return json.Marshal(result.StructuredContent)
	}
	if len(result.Content) == 1 {
		if text, ok := result.Content[0].(*mcp.TextContent); ok {
			return json.Marshal(map[string]string{"text": text.Text})
		}
	}
	return json.Marshal(result.Content)
}

func firstText(content []mcp.Content) string {
	for _, c := range content {
		if text, ok := c.(*mcp.TextContent); ok {
			return text.Text
		}
	}
	return "unknown tool error"
}

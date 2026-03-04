package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// MCP 工具定义
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// MCPClient MCP 客户端配置
type MCPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewMCPClient 创建 MCP 客户端
func NewMCPClient(baseURL string) *MCPClient {
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}
	return &MCPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListTools 获取 MCP 工具列表
func (c *MCPClient) ListTools(ctx context.Context) ([]MCPTool, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// 解析 tools/list 响应
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid result format")
	}

	toolsRaw, ok := result["tools"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid tools format")
	}

	tools := make([]MCPTool, 0, len(toolsRaw))
	for _, tool := range toolsRaw {
		toolMap, ok := tool.(map[string]interface{})
		if !ok {
			continue
		}

		mcpTool := MCPTool{
			Name:        getString(toolMap, "name"),
			Description: getString(toolMap, "description"),
			InputSchema: getMap(toolMap, "input_schema"),
		}
		tools = append(tools, mcpTool)
	}

	return tools, nil
}

// CallTool 调用 MCP 工具
func (c *MCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: mustJSON(map[string]interface{}{
			"name":      name,
			"arguments": args,
		}),
	}

	log.Printf("req: %+v", req)
	resp, err := c.sendRequest(ctx, req)
	if err != nil {
		return "", err
	}

	// 将结果转换为 JSON 字符串返回
	resultJSON, err := json.Marshal(resp.Result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// sendRequest 发送 JSON-RPC 请求
func (c *MCPClient) sendRequest(ctx context.Context, req JSONRPCRequest) (*JSONRPCResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/rpc", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var jsonResp JSONRPCResponse
	if err := json.Unmarshal(data, &jsonResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if jsonResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %v", jsonResp.Error)
	}

	return &jsonResp, nil
}

// 辅助函数

func mustJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

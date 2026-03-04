package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServer MCP 服务端
type MCPServer struct {
	addr   string
	server *mcp.Server
}

// NewMCPServer 创建 MCP 服务端
func NewMCPServer(addr string) *MCPServer {
	if addr == "" {
		addr = ":8081"
	}

	s := mcp.NewServer(
		&mcp.Implementation{Name: "pod-count-server", Version: "v0.1.0"},
		nil,
	)

	return &MCPServer{
		addr:   addr,
		server: s,
	}
}

// Start 启动服务端
func (s *MCPServer) Start() error {
	handler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server {
		return s.server
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	log.Printf("MCP Server running on %s", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

// 示例：注册 get_pod_count 工具
func registerPodCountTool(server *MCPServer) {
	type PodCountInput struct {
		Namespace string `json:"namespace" jsonschema:"namespace to query"`
	}
	type PodCountOutput struct {
		Count int `json:"count" jsonschema:"pod count"`
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest, input PodCountInput) (*mcp.CallToolResult, PodCountOutput, error) {
		if input.Namespace == "" {
			return nil, PodCountOutput{}, fmt.Errorf("namespace is required")
		}

		return nil, PodCountOutput{
			Count: mockGetPodCount(input.Namespace),
		}, nil
	}

	mcp.AddTool(server.server, &mcp.Tool{
		Name:        "get_pod_count",
		Description: "Get pod count in a namespace",
	}, handler)
}

// mockGetPodCount 模拟获取 Pod 数量
func mockGetPodCount(namespace string) int {
	if namespace == "default" {
		return 12
	}
	return 3
}

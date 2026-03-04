package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// MCPServer MCP 服务端
type MCPServer struct {
	addr  string
	tools map[string]*ToolHandler
}

// ToolHandler 工具处理器
type ToolHandler struct {
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	Handler     func(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// NewMCPServer 创建 MCP 服务端
func NewMCPServer(addr string) *MCPServer {
	return &MCPServer{
		addr:  addr,
		tools: make(map[string]*ToolHandler),
	}
}

// RegisterTool 注册工具
func (s *MCPServer) RegisterTool(name string, handler *ToolHandler) {
	s.tools[name] = handler
}

// Start 启动服务端
func (s *MCPServer) Start() error {
	http.HandleFunc("/rpc", s.handleRPC)
	log.Printf("MCP Server running on %s", s.addr)
	return http.ListenAndServe(s.addr, nil)
}

// handleRPC 处理 JSON-RPC 请求
func (s *MCPServer) handleRPC(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, 0, &JSONRPCError{
			Code:    -32700,
			Message: "Parse error",
		})
		return
	}

	switch req.Method {
	case "tools/list":
		s.handleToolsList(w, req.ID)
	case "tools/call":
		s.handleToolsCall(w, req)
	default:
		s.writeError(w, req.ID, &JSONRPCError{
			Code:    -32601,
			Message: "Method not found",
		})
	}
}

// handleToolsList 处理 tools/list 请求
func (s *MCPServer) handleToolsList(w http.ResponseWriter, id int) {
	tools := make([]map[string]interface{}, 0, len(s.tools))
	for name, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":         name,
			"description":  tool.Description,
			"input_schema": tool.InputSchema,
		})
	}

	s.writeResult(w, id, map[string]interface{}{
		"tools": tools,
	})
}

// handleToolsCall 处理 tools/call 请求
func (s *MCPServer) handleToolsCall(w http.ResponseWriter, req JSONRPCRequest) {
	var params struct {
		Name string          `json:"name"`
		Args json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal([]byte(req.Params), &params); err != nil {
		s.writeError(w, req.ID, &JSONRPCError{
			Code:    -32602,
			Message: "Invalid params",
		})
		return
	}

	tool, ok := s.tools[params.Name]
	if !ok {
		s.writeError(w, req.ID, &JSONRPCError{
			Code:    -32602,
			Message: fmt.Sprintf("Unknown tool: %s", params.Name),
		})
		return
	}

	// 解析参数
	var args map[string]interface{}
	if err := json.Unmarshal(params.Args, &args); err != nil {
		s.writeError(w, req.ID, &JSONRPCError{
			Code:    -32602,
			Message: "Invalid arguments",
		})
		return
	}

	// 调用工具
	ctx := context.Background()
	result, err := tool.Handler(ctx, args)
	if err != nil {
		s.writeError(w, req.ID, &JSONRPCError{
			Code:    -32000,
			Message: err.Error(),
		})
		return
	}

	s.writeResult(w, req.ID, result)
}

// writeResult 写入成功响应
func (s *MCPServer) writeResult(w http.ResponseWriter, id int, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	json.NewEncoder(w).Encode(resp)
}

// writeError 写入错误响应
func (s *MCPServer) writeError(w http.ResponseWriter, id int, err *JSONRPCError) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   err,
	}
	json.NewEncoder(w).Encode(resp)
}

// JSONRPCError JSON-RPC 错误
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 示例：注册 get_pod_count 工具
func registerPodCountTool(server *MCPServer) {
	server.RegisterTool("get_pod_count", &ToolHandler{
		Description: "Get pod count in a namespace",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"namespace": map[string]string{
					"type": "string",
				},
			},
			"required": []string{"namespace"},
		},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			ns, ok := args["namespace"].(string)
			if !ok {
				return nil, fmt.Errorf("namespace must be a string")
			}
			return map[string]interface{}{
				"count": mockGetPodCount(ns),
			}, nil
		},
	})
}

// mockGetPodCount 模拟获取 Pod 数量
func mockGetPodCount(namespace string) int {
	if namespace == "default" {
		return 12
	}
	return 3
}

package main

import (
	"context"
	"fmt"
	"log"
)

// Agent LLM Agent，协调 LLM 与 MCP 工具的交互
type Agent struct {
	llmClient *LLMClient
	mcpClient *MCPClient
}

// NewAgent 创建 Agent
func NewAgent(llmClient *LLMClient, mcpClient *MCPClient) *Agent {
	return &Agent{
		llmClient: llmClient,
		mcpClient: mcpClient,
	}
}

// Run 运行 Agent，处理用户问题
func (a *Agent) Run(ctx context.Context, question string) (string, error) {
	messages := []Message{
		{Role: "user", Content: question},
	}

	// 获取 MCP 工具列表
	tools, err := a.mcpClient.ListTools(ctx)
	if err != nil {
		log.Printf("Warning: failed to list tools: %v", err)
	}

	// 第一次调用 LLM
	result, err := a.llmClient.Chat(ctx, messages, toToolDefinitions(tools))
	if err != nil {
		return "", fmt.Errorf("first chat call: %w", err)
	}

	// 检查是否需要调用工具
	if result.ToolCall == nil {
		return result.Content, nil
	}

	log.Printf("Tool call requested: %s", result.ToolCall.Function.Name)

	// 解析工具调用参数
	args, err := result.ToolCall.GetArgs()
	if err != nil {
		return "", fmt.Errorf("parse tool args: %w", err)
	}

	// 调用 MCP 工具
	toolResult, err := a.mcpClient.CallTool(ctx, result.ToolCall.Function.Name, args)
	if err != nil {
		return "", fmt.Errorf("call tool: %w", err)
	}
	log.Printf("Tool result: %s", toolResult)

	// 构建带工具调用的消息历史
	messages = append(messages, Message{
		Role:      "assistant",
		Content:   "",
		ToolCalls: []ToolCall{*result.ToolCall},
	})

	messages = append(messages, Message{
		Role:    "tool",
		Name:    result.ToolCall.Function.Name,
		Content: toolResult,
	})

	// 第二次调用 LLM，获取最终回答
	finalResult, err := a.llmClient.Chat(ctx, messages, nil)
	if err != nil {
		return "", fmt.Errorf("second chat call: %w", err)
	}

	return finalResult.Content, nil
}

// toToolDefinitions 将 MCP 工具转换为 LLM API 兼容的格式
func toToolDefinitions(tools []MCPTool) []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		defs = append(defs, ToolDefinition{
			Type: "function",
			Function: FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}
	return defs
}

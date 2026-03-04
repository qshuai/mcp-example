package main

import (
	"context"
	"fmt"
	"log"
	"os"
)

func main() {
	// 检查是否有 server 参数，决定运行服务端还是客户端
	if len(os.Args) > 1 && os.Args[1] == "server" {
		runMCPServer()
		return
	}
	runAgent()
}

// runMCPServer 运行 MCP 服务端
func runMCPServer() {
	// 创建 MCP 服务器
	server := NewMCPServer(":8081")

	// 注册工具
	registerPodCountTool(server)

	// 启动服务
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// runAgent 运行 Agent 客户端
func runAgent() {
	// 检查 API Key 是否配置
	if os.Getenv("LLM_API_KEY") == "" {
		log.Println("Warning: LLM_API_KEY environment variable not set, use export LLM_API_KEY=your_key")
	}

	// 初始化客户端
	llmConfig := DefaultLLMConfig()
	llmClient := NewLLMClient(llmConfig)
	mcpClient := NewMCPClient("http://localhost:8081")

	// 创建 Agent
	agent := NewAgent(llmClient, mcpClient)

	// 用户问题
	question := "查询 xxx namespace 的 pod 数量并给出简单分析"

	// 运行 Agent
	ctx := context.Background()
	response, err := agent.Run(ctx, question)
	if err != nil {
		log.Fatalf("Agent run failed: %v", err)
	}

	fmt.Println("最终回答：")
	fmt.Println(response)
}

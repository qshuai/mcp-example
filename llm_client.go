package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// LLM 客户端配置
type LLMConfig struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// DefaultLLMConfig 从环境变量读取配置，返回默认 LLM 配置
func DefaultLLMConfig() *LLMConfig {
	return &LLMConfig{
		BaseURL: getEnv("LLM_API_URL", "https://coding.dashscope.aliyuncs.com/v1/chat/completions"),
		APIKey:  getEnv("LLM_API_KEY", ""),
		Model:   getEnv("LLM_MODEL", "qwen3.5-plus"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Message LLM 消息结构
type Message struct {
	Role      string      `json:"role"`
	Content   interface{} `json:"content,omitempty"`
	Name      string      `json:"name,omitempty"`
	ToolCalls []ToolCall  `json:"tool_calls,omitempty"`
}

// ToolDefinition 工具定义结构
type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCallFunction 工具调用的函数信息
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCall 工具调用结构（匹配 LLM API 响应格式）
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Index    int              `json:"index"`
	Function ToolCallFunction `json:"function"`
}

// GetArgs 解析工具调用参数
func (t *ToolCall) GetArgs() (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(t.Function.Arguments), &args); err != nil {
		return nil, err
	}
	return args, nil
}

// LLMResponse LLM API 响应结构
type LLMResponse struct {
	Choices []struct {
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
}

// LLMClient LLM 客户端
type LLMClient struct {
	config *LLMConfig
}

// NewLLMClient 创建 LLM 客户端
func NewLLMClient(config *LLMConfig) *LLMClient {
	return &LLMClient{config: config}
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
}

// ChatResult 聊天结果
type ChatResult struct {
	RawResponse string
	Content     string
	ToolCall    *ToolCall
}

// Chat 发送聊天请求
func (c *LLMClient) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*ChatResult, error) {
	reqBody := ChatRequest{
		Model:    "qwen3.5-plus",
		Messages: messages,
	}
	if len(tools) > 0 {
		reqBody.Tools = tools
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(data))
	}

	log.Printf("LLM response (status %d): %s", resp.StatusCode, string(data))

	return parseChatResponse(string(data))
}

// parseChatResponse 解析 LLM 响应
func parseChatResponse(respData string) (*ChatResult, error) {
	var llmResp LLMResponse
	if err := json.Unmarshal([]byte(respData), &llmResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(llmResp.Choices) == 0 {
		return nil, fmt.Errorf("empty choices in response")
	}

	msg := llmResp.Choices[0].Message
	result := &ChatResult{
		RawResponse: respData,
		Content:     msg.Content,
	}

	if len(msg.ToolCalls) > 0 {
		result.ToolCall = &msg.ToolCalls[0]
	}

	return result, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

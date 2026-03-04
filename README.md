# MCP Agent 示例（Go）

一个最小可运行的示例：
- MCP Server 暴露工具 `get_pod_count`
- Agent 先调用 LLM 判断是否需要工具，再通过 MCP Client 调工具并生成最终回答

## 功能说明

- MCP 服务端（SSE）运行在 `:8081`
- 内置工具：`get_pod_count(namespace)`
- Agent 自动完成两轮对话：
  1. 让 LLM 决策是否调用工具
  2. 调用工具后再让 LLM生成最终回答

## 项目结构

- `main.go`：程序入口（`server` 模式/`agent` 模式）
- `mcp_server.go`：MCP 服务端和工具注册
- `mcp_client.go`：MCP 客户端（列工具、调工具）
- `llm_client.go`：LLM HTTP 客户端
- `agent.go`：Agent 编排逻辑

## 环境要求

- Go `1.24.1`（见 `go.mod`）
- 可用的 LLM API Key

## 环境变量

- `LLM_API_KEY`：必填，LLM 平台密钥
- `LLM_API_URL`：可选，默认 `https://coding.dashscope.aliyuncs.com/v1/chat/completions`
- `LLM_MODEL`：可选，默认 `qwen3.5-plus`

> 注意：当前代码在发送请求时模型名写死为 `qwen3.5-plus`，即使设置了 `LLM_MODEL` 也不会生效。

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 配置环境变量

```bash
export LLM_API_KEY="your_api_key"
# 可选
# export LLM_API_URL="https://coding.dashscope.aliyuncs.com/v1/chat/completions"
# export LLM_MODEL="qwen3.5-plus"
```

### 3. 启动 MCP Server（终端 1）

```bash
go run . server
```

看到日志类似：

```text
MCP Server running on :8081
```

### 4. 启动 Agent（终端 2）

```bash
go run .
```

程序会使用内置问题：

```text
查询 xxx namespace 的 pod 数量并给出简单分析
```

最后输出类似：

```text
最终回答：
...（LLM 结合工具结果的分析）
```

## 工具行为说明

`get_pod_count` 是演示用 mock：
- 当 `namespace == default` 时返回 `12`
- 其他 namespace 返回 `3`

## 常见问题

- 报错 `LLM_API_KEY environment variable not set`
  - 请先 `export LLM_API_KEY=...` 后再运行。
- 报错 MCP 连接失败
  - 请确认 `go run . server` 已启动，并监听 `http://localhost:8081`。

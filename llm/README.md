# llm

Go 语言 LLM 客户端库，提供统一、类型安全的接口来与各大厂商的大模型交互。

## API 兼容性说明

本库基于 **OpenAI Chat Completions API** 格式实现，这是目前业界通用的标准协议，被各大厂商广泛兼容。

- ✅ 支持所有兼容 OpenAI 格式的服务商
- ✅ 统一的请求/响应结构
- ⚠️ 不支持 OpenAI 2025 新推出的 Responses API（这是可选的高级格式，主流厂商尚未跟进）

## 支持的 Provider

| Provider | Chat | Embeddings | 文档 |
|----------|------|------------|------|
| OpenAI | ✅ | ✅ | [provider/openai](./provider/openai) |
| DeepSeek | ✅ | ✅ | [provider/deepseek](./provider/deepseek/README.md) |
| Kimi (Moonshot) | ✅ | ✅ | [provider/kimi](./provider/kimi/README.md) |
| Qwen (通义千问) | ✅ | ✅ | [provider/qwen](./provider/qwen/README.md) |
| Ollama | ✅ | ✅ | [provider/ollama](./provider/ollama/README.md) |

## 快速开始

### 安装

```bash
go get github.com/lgc202/go-kit/llm
```

### 基础对话

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lgc202/go-kit/llm"
    "github.com/lgc202/go-kit/llm/provider/openai/chat"
    "github.com/lgc202/go-kit/llm/schema"
)

	func main() {
	    client, err := chat.New(chat.Config{
	        BaseConfig: chat.BaseConfig{
	            APIKey: os.Getenv("OPENAI_API_KEY"),
	        },
	        DefaultOptions: []llm.ChatOption{
	            llm.WithModel("gpt-4o-mini"),
	        },
	    })
    if err != nil {
        log.Fatal(err)
    }

    resp, err := client.Chat(context.Background(), []schema.Message{
        schema.UserMessage("什么是 Go 语言？"),
    },
        llm.WithTemperature(0.7),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Choices[0].Message.Text())
}
```

### 流式输出

```go
stream, err := client.ChatStream(context.Background(), []schema.Message{
    schema.UserMessage("用一句话介绍人工智能"),
})
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    event, err := stream.Recv()
    if err != nil {
        break // 流结束
    }
    if event.Type == schema.StreamEventDelta {
        fmt.Print(event.Delta)
    }
}
```

### 多模态输入

```go
messages := []schema.Message{
    schema.UserMessage("这张图片里有什么？"),
}

// 添加图片
messages[0].Content = append(messages[0].Content,
    schema.ImageURLPart("https://example.com/image.jpg"),
)

resp, err := client.Chat(ctx, messages)
```

### 工具调用

```go
weatherTool, _ := schema.NewFunctionTool(
    "get_weather",
    "获取指定地点的当前天气",
    map[string]any{
        "type": "object",
        "properties": map[string]any{
            "location": map[string]any{
                "type":        "string",
                "description": "城市名称",
            },
        },
        "required": []string{"location"},
    },
)

messages := []schema.Message{
    schema.UserMessage("北京今天天气怎么样？"),
}

// 循环处理：模型可能需要多次调用工具
const maxSteps = 8
for step := 0; step < maxSteps; step++ {
    resp, err := client.Chat(ctx, messages, llm.WithTools(weatherTool))
    if err != nil {
        break
    }

    msg := resp.Choices[0].Message
    messages = append(messages, msg)  // 重要：添加 assistant 消息到历史

    if len(msg.ToolCalls) == 0 {
        // 模型不再调用工具，返回最终回复
        fmt.Println(msg.Text())
        break
    }

    // 执行工具调用
    for _, tc := range msg.ToolCalls {
        fmt.Printf("调用: %s, 参数: %s\n", tc.Function.Name, tc.Function.Arguments)

        // 执行工具获取结果
        result := getWeather(tc.Function.Arguments)

        // 将工具结果添加到历史
        messages = append(messages, schema.ToolResultMessage(tc.ID, result))
    }
}
```

## 目录结构

```
llm/
├── llm.go              # 核心接口定义（ChatModel、Embedder、Stream）
├── options.go          # 请求选项配置
├── api_error.go        # 错误类型和辅助函数
├── schema/             # 数据结构定义
│   ├── message.go      # 消息和多模态内容
│   ├── tools.go        # 工具/函数调用
│   ├── chat.go         # 聊天响应
│   ├── stream.go       # 流式事件
│   └── builders.go     # 便捷构造函数
├── provider/           # 各厂商实现
│   ├── openai/         # OpenAI
│   ├── deepseek/       # DeepSeek
│   ├── kimi/           # Moonshot Kimi
│   ├── qwen/           # 阿里通义千问
│   └── ollama/         # Ollama 本地模型
├── internal/           # 内部实现
│   └── openai_compat/  # OpenAI 兼容协议复用
└── examples/           # 使用示例
```

## 核心概念

### Provider（厂商）

Provider 代表具体的 LLM 服务提供商。每个 provider 实现相同的接口，可以无缝切换：

```go
var client llm.ChatModel

// 使用 OpenAI
client, _ = openai.New(...)

// 或使用 DeepSeek
client, _ = deepseek.New(...)

// 使用相同的接口调用
client.Chat(ctx, messages, ...)
```

### Message（消息）

消息是对话的基本单位，支持多模态内容：

```go
// 简单文本消息
msg := schema.UserMessage("你好")

// 多模态消息
msg := schema.Message{
    Role: schema.RoleUser,
    Content: []schema.ContentPart{
        schema.TextPart("描述这张图片"),
        schema.ImageURLPart("https://example.com/image.jpg"),
        schema.BinaryPart("image/jpeg", imageData),
    },
}
```

### Option 模式

所有请求参数通过 Option 传递，链式调用，灵活组合：

```go
resp, err := client.Chat(ctx, messages,
    llm.WithModel("gpt-4o"),
    llm.WithTemperature(0.7),
    llm.WithMaxTokens(1000),
    llm.WithTools(tool),
)
```

### Stream（流式响应）

流式响应通过简单的 `Recv()` 接口逐块读取：

```go
stream, _ := client.ChatStream(ctx, messages, ...)
defer stream.Close()

for {
    event, err := stream.Recv()
    if err != nil {
        break // io.EOF 表示流正常结束
    }
    // 处理 event
}
```

## 错误处理

```go
resp, err := client.Chat(ctx, messages, ...)
if err != nil {
    // 检查错误类型
    if llm.IsRateLimit(err) {
        // 限流错误，等待后重试
    } else if llm.IsAuth(err) {
        // 认证错误，检查 API Key
    } else if llm.IsTemporary(err) {
        // 临时错误，可以重试
    }

    // 获取详细错误信息
    if apiErr, ok := llm.AsAPIError(err); ok {
        fmt.Printf("Provider: %s\n", apiErr.Provider)
        fmt.Printf("StatusCode: %d\n", apiErr.StatusCode)
        fmt.Printf("RequestID: %s\n", apiErr.RequestID)
        fmt.Printf("Message: %s\n", apiErr.Message)
    }
}
```

## 扩展机制

### ExtraFields - 传递厂商特有字段

```go
// DeepSeek 特有的 thinking 参数
resp, err := client.Chat(ctx, messages,
    llm.WithExtraField("thinking", map[string]string{"type": "enabled"}),
)

// 多个 extra fields
llm.WithExtraFields(map[string]any{
    "field1": value1,
    "field2": value2,
})
```

### Hook - 自定义响应处理

```go
// ResponseHook - 处理完整响应
llm.WithResponseHook(func(dst *schema.ChatResponse, raw json.RawMessage) error {
    // 从原始响应中提取额外字段
    var extra struct {
        CustomField string `json:"custom_field"`
    }
    if err := json.Unmarshal(raw, &extra); err != nil {
        return err
    }
    if dst.ExtraFields == nil {
        dst.ExtraFields = make(map[string]any)
    }
    dst.ExtraFields["custom_field"] = extra.CustomField
    return nil
})

// StreamEventHook - 处理流式事件
llm.WithStreamEventHook(func(dst *schema.StreamEvent, raw json.RawMessage) error {
    // 自定义流事件处理
    return nil
})
```

### ExtraHeaders - 自定义请求头

```go
llm.WithHeader("X-Custom-Header", "value")

// 批量设置
llm.WithExtraHeaders(map[string]string{
    "X-Header-1": "value1",
    "X-Header-2": "value2",
})
```

## 常见用法

### 客户端级默认配置

```go
	client, err := chat.New(chat.Config{
	    BaseConfig: chat.BaseConfig{
	        APIKey: os.Getenv("OPENAI_API_KEY"),
	    },
	    DefaultOptions: []llm.ChatOption{
	        llm.WithModel("gpt-4o-mini"),
	        llm.WithTemperature(0.7),
	    },
	})

// 后续请求会自动使用默认配置
resp, _ := client.Chat(ctx, messages)
```

### 自定义 Base URL

```go
	client, err := chat.New(chat.Config{
	    BaseConfig: chat.BaseConfig{
	        BaseURL: "https://your-proxy.com/v1",
	        APIKey:  os.Getenv("API_KEY"),
	    },
	})
	```

### 获取原始响应

```go
resp, err := client.Chat(ctx, messages,
    llm.WithKeepRaw(true), // 保留原始 JSON 响应
)

// 访问原始响应
fmt.Println(string(resp.Raw))
```

### Token 使用统计

```go
resp, err := client.Chat(ctx, messages, ...)

usage := resp.Usage
fmt.Printf("输入: %d, 输出: %d, 总计: %d\n",
    usage.PromptTokens,
    usage.CompletionTokens,
    usage.TotalTokens,
)

// 缓存命中（支持的 provider）
fmt.Printf("缓存命中: %d\n", usage.PromptCacheHitTokens)
```

## 更多示例

```bash
# DeepSeek 示例
API_KEY=xxx MODEL=deepseek-chat go run examples/deepseek/basic/main.go       # 基础对话
API_KEY=xxx MODEL=deepseek-reasoner go run examples/deepseek/reasoning/main.go   # 推理模型
API_KEY=xxx MODEL=deepseek-chat go run examples/deepseek/tools/main.go       # 工具调用
API_KEY=xxx MODEL=deepseek-chat go run examples/deepseek/stream/main.go      # 流式输出

# Ollama 示例
MODEL=qwen2.5 go run examples/ollama/basic/main.go         # 基础对话
MODEL=deepseek-r1:1.5b go run examples/ollama/reasoning/main.go     # 推理模型
MODEL=llama3.2 go run examples/ollama/stream/main.go      # 流式输出
MODEL=llama3.2 go run examples/ollama/tools/main.go       # 工具调用
```

## 相关文档

- [OpenAI Provider 文档](./provider/openai/README.md)
- [DeepSeek Provider 文档](./provider/deepseek/README.md)
- [Ollama Provider 文档](./provider/ollama/README.md)
- [Qwen Provider 文档](./provider/qwen/README.md)
- [Kimi Provider 文档](./provider/kimi/README.md)

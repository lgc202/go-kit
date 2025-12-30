# DeepSeek Provider

go-kit 的企业级 DeepSeek API 客户端。

## 功能特性

- ✅ **基础对话与流式输出** - 完整支持对话补全和 SSE 流式传输
- ✅ **推理模型** - 原生支持 `deepseek-reasoner` 及思维控制
- ✅ **工具/函数调用** - 完整的函数调用及工具选择模式
- ✅ **JSON 模式** - 支持 `json_object` 和 `json_schema` 结构化输出
- ✅ **Token 使用详情** - 包含缓存命中/未命中和推理 token
- ✅ **流式选项** - 流式响应中包含使用统计
- ✅ **类型安全选项** - provider 特定的便捷函数

## 安装

```go
import "github.com/lgc202/go-kit/llm/provider/deepseek"
```

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lgc202/go-kit/llm"
    "github.com/lgc202/go-kit/llm/provider/deepseek"
    "github.com/lgc202/go-kit/llm/schema"
)

func main() {
    client, err := deepseek.New(deepseek.Config{
        APIKey: os.Getenv("DEEPSEEK_API_KEY"),
        DefaultRequest: llm.RequestConfig{
            Model: "deepseek-chat",
        },
    })
    if err != nil {
        panic(err)
    }

    resp, err := client.Chat(context.Background(), []schema.Message{
        schema.UserMessage("2+2 等于几？"),
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Message.Text())
    // 输出: 等于 4。
}
```

## 模型

| 模型 | 描述 | 最大 Token |
|-------|-------------|------------|
| `deepseek-chat` | 通用对话模型 | 128K |
| `deepseek-reasoner` | 高级推理模型 | 64K |

## 选项配置

### 基础选项（来自 `llm` 包）

```go
llm.WithModel("deepseek-chat")
llm.WithTemperature(0.7)
llm.WithMaxTokens(1000)
llm.WithTopP(0.9)
llm.WithFrequencyPenalty(0.5)
llm.WithPresencePenalty(0.5)
llm.WithStop([]string{"\n", "END"})
```

### DeepSeek 特有选项

有两种方式设置 DeepSeek 特有选项：

#### 方式 1：类型安全的便捷函数（推荐）

```go
// 思维模式（用于 deepseek-reasoner）
deepseek.WithThinking(true)   // 启用推理（deepseek-reasoner 默认值）
deepseek.WithThinking(false)  // 禁用推理

// 流式选项
deepseek.WithStreamIncludeUsage()  // 获取流式响应中的使用统计
deepseek.WithStreamOptions(deepseek.StreamOptions{IncludeUsage: true})
```

#### 方式 2：通用 ExtraField（灵活）

```go
// 使用导出的常量
llm.WithExtraField(deepseek.ExtThinking, deepseek.Thinking{Type: deepseek.ThinkingTypeEnabled})
llm.WithExtraField(deepseek.ExtStreamOptions, deepseek.StreamOptions{IncludeUsage: true})

// 或直接使用 key（适用于未文档化/新字段）
llm.WithExtraField("thinking", map[string]any{"type": "enabled"})
llm.WithExtraField("new_beta_feature", value)
```

### 标准选项（来自 `llm` 包）

```go
// 模型参数
llm.WithTemperature(0.7)
llm.WithMaxTokens(1000)
llm.WithTopP(0.9)
llm.WithFrequencyPenalty(0.5)
llm.WithPresencePenalty(0.5)
llm.WithStop([]string{"\n", "END"})

// 工具调用
llm.WithTools(tool1, tool2)
llm.WithToolChoice(schema.ToolChoice{Mode: "required"})  // 强制调用工具
llm.WithToolChoice(schema.ToolChoice{Mode: "auto"})       // 由模型决定
llm.WithToolChoice(schema.ToolChoice{Mode: "none"})       // 禁用工具调用

// 响应格式
llm.WithResponseFormat(schema.ResponseFormat{Type: "json_object"})
llm.WithResponseFormat(schema.ResponseFormat{
    Type:       "json_schema",
    JSONSchema: schemaJSON,
})
```

## 推理模型

`deepseek-reasoner` 模型提供高级推理能力，响应中包含 `reasoning_content` 字段：

```go
client, _ := deepseek.New(deepseek.Config{
    APIKey: os.Getenv("DEEPSEEK_API_KEY"),
    DefaultRequest: llm.RequestConfig{
        Model: "deepseek-reasoner",
    },
})

resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserMessage("如果我有 5 个苹果，吃了 2 个，又买了 3 个，现在有几个？"),
},
    deepseek.WithThinking(true), // 启用推理（deepseek-reasoner 默认值）
)

msg := resp.Choices[0].Message
fmt.Println("推理过程:", msg.ReasoningContent)
// 输出: 让我们一步步来思考...

fmt.Println("答案:", msg.Text())
// 输出: 你现在有 6 个苹果。
```

### 禁用推理以提升速度

对于简单问题，禁用推理可以获得更快、更便宜的响应：

```go
resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserMessage("法国的首都是哪里？"),
},
    deepseek.WithThinking(false), // 禁用推理
)
```

### 流式输出推理内容

在流式响应中分离推理内容和实际内容：

```go
resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserMessage("计算：15 * 23 - 47"),
},
    llm.WithStreamingReasoningFunc(
        func(ctx context.Context, reasoningChunk, contentChunk []byte) error {
            if len(reasoningChunk) > 0 {
                fmt.Printf("[思考] %s", reasoningChunk)
            }
            if len(contentChunk) > 0 {
                fmt.Printf("[内容] %s", contentChunk)
            }
            return nil
        },
    ),
    deepseek.WithThinking(true),
)
```

## 工具/函数调用

```go
weatherTool := schema.Tool{
    Type: schema.ToolTypeFunction,
    Function: schema.FunctionDefinition{
        Name:        "get_weather",
        Description: "获取指定地点的当前天气",
        Parameters: json.RawMessage(`{
            "type": "object",
            "properties": {
                "location": {"type": "string"}
            },
            "required": ["location"]
        }`),
    },
}

resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserMessage("东京天气怎么样？"),
},
    llm.WithTools(weatherTool),
)

msg := resp.Choices[0].Message
if len(msg.ToolCalls) > 0 {
    // 模型想要调用工具
    for _, tc := range msg.ToolCalls {
        fmt.Printf("调用: %s\n", tc.Function.Name)
        fmt.Printf("参数: %s\n", tc.Function.Arguments)

        // 执行工具并将结果返回
        result := `{"temp": "22°C", "condition": "晴朗"}`
        resp2, _ := client.Chat(context.Background(), []schema.Message{
            schema.UserMessage("东京天气怎么样？"),
            msg,
            schema.ToolResultMessage(tc.ID, result),
        })
        fmt.Println(resp2.Choices[0].Message.Text())
    }
}
```

## JSON 模式

### JSON 对象响应

```go
resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserMessage("列出 3 种编程语言"),
},
    llm.WithResponseFormat(schema.ResponseFormat{Type: "json_object"}),
)
fmt.Println(resp.Choices[0].Message.Text())
```

### JSON Schema 响应

```go
schemaJSON := json.RawMessage(`{
    "type": "object",
    "properties": {
        "name": {"type": "string"},
        "year": {"type": "integer"}
    },
    "required": ["name", "year"]
}`)

resp, err := client.Chat(context.Background(), []schema.Message{
    schema.UserMessage("介绍一下 Go 语言"),
},
    llm.WithResponseFormat(schema.ResponseFormat{
        Type:       "json_schema",
        JSONSchema: schemaJSON,
    }),
)
fmt.Println(resp.Choices[0].Message.Text())
```

## 流式输出

### 基础流式

```go
stream, err := client.ChatStream(context.Background(), []schema.Message{
    schema.UserMessage("从 1 数到 10"),
})
defer stream.Close()

for {
    event, err := stream.Recv()
    if err != nil {
        break
    }
    if event.Type == schema.StreamEventDelta {
        fmt.Print(event.Delta)
    }
}
```

### 流式输出包含使用统计

```go
stream, err := client.ChatStream(context.Background(), []schema.Message{
    schema.UserMessage("你好"),
},
    deepseek.WithStreamIncludeUsage(), // 在最终事件中获取使用统计
)
```

## Token 使用统计

DeepSeek 提供详细的 token 使用统计：

```go
resp, err := client.Chat(...)
usage := resp.Usage

fmt.Println("输入 token:", usage.PromptTokens)
fmt.Println("输出 token:", usage.CompletionTokens)
fmt.Println("总 token:", usage.TotalTokens)

// 缓存使用（前缀缓存）
fmt.Println("缓存命中:", usage.PromptCacheHitTokens)
fmt.Println("缓存未命中:", usage.PromptCacheMissTokens)

// 推理 token（deepseek-reasoner）
if usage.CompletionTokensDetails != nil {
    fmt.Println("推理 token:", usage.CompletionTokensDetails.ReasoningTokens)
}
```

## 配置

### 客户端级默认配置

```go
client, err := deepseek.New(deepseek.Config{
    APIKey: os.Getenv("DEEPSEEK_API_KEY"),
    DefaultRequest: llm.RequestConfig{
        Model:       "deepseek-chat",
        Temperature: llm.Of(0.7),
        MaxTokens:   llm.Of(1000),
    },
})
```

### 自定义 Base URL

```go
client, err := deepseek.New(deepseek.Config{
    BaseURL: "https://your-proxy.example.com",
    APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
})
```

### 自定义 HTTP 客户端

```go
httpClient := &http.Client{
    Timeout: 30 * time.Second,
}

client, err := deepseek.New(deepseek.Config{
    APIKey:     os.Getenv("DEEPSEEK_API_KEY"),
    HTTPClient: httpClient,
})
```

### 默认请求头

```go
headers := make(http.Header)
headers.Set("X-Custom-Header", "value")

client, err := deepseek.New(deepseek.Config{
    APIKey:         os.Getenv("DEEPSEEK_API_KEY"),
    DefaultHeaders: headers,
})
```

## Provider 检测

```go
import "github.com/lgc202/go-kit/llm"

var client llm.ChatModel = deepseek.New(...)

// 检测 provider
provider := llm.ProviderOf(client)
if provider == llm.ProviderDeepSeek {
    // 使用 DeepSeek 特有选项
    deepseek.WithThinking(true)
}
```

## 示例

更多示例请参阅 [`examples/`](./examples) 包：

- [`examples.Basic()`](./examples/basic.go) - 基础对话和流式输出
- [`examples.Reasoning()`](./examples/reasoning.go) - 推理模型使用
- [`examples.ToolCalling()`](./examples/tools.go) - 函数调用和 JSON 模式

## API 参考

- [DeepSeek API 文档](https://api-docs.deepseek.com/)
- [llm 包](../../README.md)

## 许可证

MIT

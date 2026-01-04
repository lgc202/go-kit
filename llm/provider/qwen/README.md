# Qwen (通义千问) Provider

go-kit 的企业级通义千问 API 客户端。

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lgc202/go-kit/llm"
    qwen "github.com/lgc202/go-kit/llm/provider/qwen/chat"
    "github.com/lgc202/go-kit/llm/schema"
)

func main() {
    client, err := qwen.New(qwen.Config{
        APIKey: os.Getenv("DASHSCOPE_API_KEY"),
        DefaultOptions: []llm.ChatOption{
            llm.WithModel("qwen-plus"),
        },
    })
    if err != nil {
        panic(err)
    }

    resp, err := client.Chat(context.Background(), []schema.Message{
        schema.UserMessage("什么是 Go 语言？"),
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Message.Text())
}
```

## 选项配置

### 基础选项

```go
llm.WithTemperature(0.7)
llm.WithMaxCompletionTokens(4096)
llm.WithTopP(0.9)
```

### Qwen 特有选项

```go
// 启用深度思考模式
qwen.WithThinking(true)

// 或使用通用 ExtraField
llm.WithExtraField("enable_thinking", true)
```

## 深度思考模式

Qwen 支持深度思考模型，可以获取模型的推理过程：

```go
resp, err := client.Chat(ctx, []schema.Message{
    schema.UserMessage("解决这个数学问题: 如果我有 5 个苹果，吃了 2 个，又买了 3 个，现在有几个？"),
},
    qwen.WithThinking(true),
)

msg := resp.Choices[0].Message
if msg.ReasoningContent != "" {
    fmt.Println("推理过程:", msg.ReasoningContent)
}
fmt.Println("答案:", msg.Text())
```

### 禁用推理以提升速度

对于简单问题，禁用推理可以获得更快、更便宜的响应：

```go
resp, err := client.Chat(ctx, []schema.Message{
    schema.UserMessage("法国的首都是哪里？"),
},
    qwen.WithThinking(false),
)
```

### 流式输出推理内容

在流式响应中分离推理内容和实际内容：

```go
stream, err := client.ChatStream(ctx, []schema.Message{
    schema.UserMessage("计算：15 * 23 - 47"),
},
    qwen.WithThinking(true),
)

for {
    ev, err := stream.Recv()
    if err != nil {
        break
    }
    if len(ev.Reasoning) > 0 {
        fmt.Printf("[思考] %s", ev.Reasoning)
    }
    if len(ev.Delta) > 0 {
        fmt.Printf("[内容] %s", ev.Delta)
    }
}
```

## 工具/函数调用

```go
weatherTool := schema.Tool{
    Type: schema.ToolTypeFunction,
    Function: schema.FunctionDefinition{
        Name:        "get_weather",
        Description: "获取指定地点的当前天气",
        Parameters:  schema.MustJSON(map[string]any{
            "type": "object",
            "properties": map[string]any{
                "location": map[string]any{
                    "type":        "string",
                    "description": "城市名称",
                },
            },
            "required": []string{"location"},
        }),
    },
}

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
    messages = append(messages, msg)  // 重要：添加 assistant 消息（包含 tool_calls）到历史

    if len(msg.ToolCalls) == 0 {
        // 模型不再调用工具，返回最终回复
        fmt.Println(msg.Text())
        break
    }

    // 执行工具调用
    for _, tc := range msg.ToolCalls {
        fmt.Printf("调用: %s\n", tc.Function.Name)
        fmt.Printf("参数: %s\n", tc.Function.Arguments)

        // 执行工具获取结果并添加到历史
        result := getWeather(tc.Function.Arguments)
        messages = append(messages, schema.ToolResultMessage(tc.ID, result))
    }
}
```

## 流式输出

### 基础流式

```go
stream, err := client.ChatStream(ctx, []schema.Message{
    schema.UserMessage("讲一个故事"),
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
stream, err := client.ChatStream(ctx, []schema.Message{
    schema.UserMessage("你好"),
},
    llm.WithStreamIncludeUsage(),
)
```

## 配置

### 客户端级默认配置

```go
client, err := qwen.New(qwen.Config{
    APIKey: os.Getenv("DASHSCOPE_API_KEY"),
    DefaultOptions: []llm.ChatOption{
        llm.WithModel("qwen-plus"),
        llm.WithTemperature(0.7),
    },
})
```

### 自定义 Base URL

```go
client, err := qwen.New(qwen.Config{
    BaseURL: "https://your-proxy.example.com",
    APIKey:  os.Getenv("DASHSCOPE_API_KEY"),
})
```

### 自定义 HTTP 客户端

```go
httpClient := &http.Client{
    Timeout: 30 * time.Second,
}

client, err := qwen.New(qwen.Config{
    APIKey:     os.Getenv("DASHSCOPE_API_KEY"),
    HTTPClient: httpClient,
})
```

### 默认请求头

```go
headers := make(http.Header)
headers.Set("X-Custom-Header", "value")

client, err := qwen.New(qwen.Config{
    APIKey:         os.Getenv("DASHSCOPE_API_KEY"),
    DefaultHeaders: headers,
})
```

## API 文档

- [通义千问 API 文档](https://help.aliyun.com/zh/model-studio/developer-reference/use-qwen-by-calling-api)
- [深度思考模式](https://www.alibabacloud.com/help/en/model-studio/deep-thinking)
- [llm 包](../../README.md)
